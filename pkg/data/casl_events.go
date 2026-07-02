package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

func isCASLActionSource(sourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "user_action", "mob_user_action", "ppk_action", "ppk_service", "system_action", "system_event", "m3_in",
		"mgr_action", "grd_object_action", "norm_msg_action", "db_change", "login_action", "device_action", "read_journal_action", "rtsp_action",
		"post-proc-alarm-report":
		return true
	default:
		return false
	}
}

func effectiveCASLSourceType(row CASLObjectEvent) string {
	sourceType := strings.TrimSpace(row.Type)
	switch strings.ToLower(sourceType) {
	case "user_action", "mob_user_action":
		if value := strings.TrimSpace(row.UserActionType); value != "" {
			return value
		}
		if value := strings.TrimSpace(row.MgrActionType); value != "" {
			return value
		}
		if value := strings.TrimSpace(row.PPKActionType); value != "" {
			return value
		}
	case "m3_in":
		if value := strings.TrimSpace(row.UserActionType); value != "" {
			return value
		}
		return "mgr_action"
	}
	return sourceType
}

func isCASLUnknownText(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "unknown", "undefined", "unset", "not set", "не встановлено", "невідомо", "none", "null":
		return true
	default:
		return false
	}
}

func (p *CASLCloudProvider) resolveCASLAlarmFlag(ctx context.Context, deviceType string, code string, contactID string, subtype string) (bool, bool) {
	if !p.isCASLCustomDeviceType(ctx, deviceType) {
		flags := p.loadAlarmEventsCatalogMap(ctx)
		return resolveCASLAlarmFlagFromAlarmCatalog(flags, code, contactID, deviceType)
	}

	flags := p.loadTranslatorAlarmFlags(ctx, deviceType)
	return resolveCASLAlarmFlagFromMap(flags, code, contactID, subtype)
}

func resolveCASLAlarmFlagFromMap(flags map[string]bool, code string, contactID string, subtype string) (bool, bool) {
	if len(flags) == 0 {
		return false, false
	}

	for _, candidate := range caslTranslatorAlarmLookupCandidates(code, contactID, subtype) {
		if isAlarm, ok := flags[candidate]; ok {
			return isAlarm, true
		}
	}

	return false, false
}

func resolveCASLAlarmFlagFromAlarmCatalog(flags map[string]bool, code string, contactID string, deviceType string) (bool, bool) {
	if len(flags) == 0 {
		return false, false
	}

	for _, candidate := range caslAlarmCatalogLookupCandidates(code, contactID, deviceType) {
		if isAlarm, ok := flags[candidate]; ok {
			return isAlarm, true
		}
	}

	return false, false
}

func caslTranslatorAlarmLookupCandidates(code string, contactID string, subtype string) []string {
	subtype = strings.ToUpper(strings.TrimSpace(subtype))
	rawCandidates := make([]string, 0, 4)
	if prefixed := withCASLTranslatorEventPrefix(contactID, subtype); prefixed != "" {
		rawCandidates = append(rawCandidates, prefixed)
	}
	if prefixed := withCASLTranslatorEventPrefix(code, subtype); prefixed != "" {
		rawCandidates = append(rawCandidates, prefixed)
	}
	rawCandidates = append(rawCandidates,
		strings.TrimSpace(contactID),
		strings.TrimSpace(code),
	)

	out := make([]string, 0, len(rawCandidates)*2)
	seen := make(map[string]struct{}, len(rawCandidates)*2)
	for _, candidate := range rawCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		for _, normalized := range []string{candidate, normalizeCASLTranslatorNumericTail(candidate)} {
			normalized = strings.TrimSpace(normalized)
			if normalized == "" {
				continue
			}
			if _, exists := seen[normalized]; exists {
				continue
			}
			seen[normalized] = struct{}{}
			out = append(out, normalized)
		}
	}

	return out
}

func caslAlarmCatalogLookupCandidates(code string, contactID string, deviceType string) []string {
	rawCandidates := []string{
		strings.ToUpper(strings.TrimSpace(code)),
		strings.ToUpper(strings.TrimSpace(contactID)),
	}
	if decoded, ok := decodeCASLProtocolCode(code, deviceType); ok {
		rawCandidates = append(rawCandidates, strings.ToUpper(strings.TrimSpace(decoded.MessageKey)))
	}

	out := make([]string, 0, len(rawCandidates))
	seen := make(map[string]struct{}, len(rawCandidates))
	for _, candidate := range rawCandidates {
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	return out
}

func withCASLTranslatorEventPrefix(value string, subtype string) string {
	value = strings.TrimSpace(value)
	subtype = strings.ToUpper(strings.TrimSpace(subtype))
	if value == "" || subtype == "" {
		return ""
	}
	if subtype != "E" && subtype != "R" && subtype != "P" {
		return ""
	}
	if !isCASLTranslatorNumericCode(value) {
		return ""
	}
	return subtype + value
}

func normalizeCASLTranslatorNumericTail(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) < 2 {
		return ""
	}
	if value[0] != 'E' && value[0] != 'R' && value[0] != 'P' {
		return ""
	}
	if !isCASLTranslatorNumericCode(value[1:]) {
		return ""
	}
	return value[1:]
}

func (p *CASLCloudProvider) isCASLCustomDeviceType(ctx context.Context, deviceType string) bool {
	deviceType = strings.TrimSpace(deviceType)
	if deviceType == "" {
		return false
	}

	customTypes := p.loadCASLCustomDeviceTypeSet(ctx)
	if len(customTypes) == 0 {
		return false
	}

	_, ok := customTypes[strings.ToUpper(deviceType)]
	return ok
}

func (p *CASLCloudProvider) loadCASLCustomDeviceTypeSet(ctx context.Context) map[string]struct{} {
	dict, ok := p.cachedDictionarySnapshot(ctx)
	if !ok || len(dict) == 0 {
		return nil
	}

	result := make(map[string]struct{})
	for _, candidate := range extractCASLUserDeviceTypes(dict) {
		candidate = strings.ToUpper(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		result[candidate] = struct{}{}
	}
	return result
}

func (p *CASLCloudProvider) loadAlarmEventsCatalogMap(ctx context.Context) map[string]bool {
	p.mu.RLock()
	if len(p.cachedAlarmEvents) > 0 && time.Since(p.cachedAlarmEventsAt) <= caslDictionaryTTL {
		cached := cloneBoolMap(p.cachedAlarmEvents)
		p.mu.RUnlock()
		return cached
	}
	p.mu.RUnlock()

	rows, err := p.ReadAlarmEventsCatalog(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося прочитати read_alarm_events для класифікації тривог")
		return nil
	}

	result := make(map[string]bool, len(rows))
	for _, row := range rows {
		code := strings.ToUpper(strings.TrimSpace(row.Code))
		if code == "" {
			continue
		}
		result[code] = row.IsAlarm > 0
	}

	p.mu.Lock()
	p.cachedAlarmEvents = cloneBoolMap(result)
	p.cachedAlarmEventsAt = time.Now()
	p.mu.Unlock()

	return cloneBoolMap(result)
}

func (p *CASLCloudProvider) resolveCASLPPKByDeviceIDWithCache(
	ctx context.Context,
	deviceID string,
	resolved map[string]int64,
	unresolved map[string]struct{},
) int64 {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return 0
	}
	if resolved != nil {
		if ppkNum, ok := resolved[deviceID]; ok {
			return ppkNum
		}
	}
	if unresolved != nil {
		if _, failed := unresolved[deviceID]; failed {
			return 0
		}
	}

	ppkNum, ok := p.resolvePPKByDeviceID(ctx, deviceID)
	if !ok || ppkNum <= 0 {
		if unresolved != nil {
			unresolved[deviceID] = struct{}{}
		}
		return 0
	}
	if resolved != nil {
		resolved[deviceID] = ppkNum
	}
	return ppkNum
}

func fallbackCASLActionDetails(row CASLObjectEvent, sourceType string) string {
	action := strings.TrimSpace(row.Action)
	if action == "" {
		action = strings.TrimSpace(row.Code)
	}
	actionUpper := strings.ToUpper(action)
	actionType := strings.ToLower(strings.TrimSpace(row.UserActionType))
	if actionType == "" {
		actionType = strings.ToLower(strings.TrimSpace(sourceType))
	}

	switch {
	case strings.HasPrefix(actionUpper, "GRD_OBJ_"):
		return "Дія оператора"
	case actionUpper == "DEVICE_BLOCK":
		return "Пристрій заблоковано"
	case actionUpper == "DEVICE_UNBLOCK":
		return "Пристрій розблоковано"
	case actionType == "mgr_action":
		return "Дія МГР"
	case actionType == "grd_object_action":
		return "Дія з тривогою"
	case actionType == "norm_msg_action":
		return "Дія із заявочним повідомленням"
	case actionType == "db_change":
		return "Зміна даних"
	case actionType == "login_action":
		return "Вхід користувача"
	case actionType == "read_journal_action":
		return "Перегляд журналу"
	case actionType == "device_action":
		return "Дія з пристроєм"
	case actionType == "rtsp_action":
		return "RTSP дія"
	case actionType == "ppk_action":
		return "Дія з ППК"
	case isCASLActionSource(sourceType):
		switch strings.ToLower(strings.TrimSpace(sourceType)) {
		case "user_action", "mob_user_action":
			return "Дія оператора"
		case "ppk_action", "ppk_service":
			return "Сервісна подія ППК"
		default:
			return "Системна подія CASL"
		}
	default:
		return ""
	}
}

func caslActionTypeLabel(sourceType string) string {
	switch strings.ToLower(strings.TrimSpace(sourceType)) {
	case "read_journal_action":
		return "ПЕРЕГЛЯД ЖУРНАЛУ"
	default:
		return ""
	}
}

func (p *CASLCloudProvider) buildSharedObjectContext(ctx context.Context) (
	byPPK map[int64]caslEventContext,
	byObject map[string]caslEventContext,
	err error,
) {
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, nil, err
	}
	_, _ = p.loadDevices(ctx) // прогріваємо кеш пристроїв

	ppkFilter := make(map[int64]struct{}, len(records))
	objFilter := make(map[string]struct{}, len(records))
	for _, record := range records {
		if objID := strings.TrimSpace(record.ObjID); objID != "" {
			objFilter[objID] = struct{}{}
		}
		if ppkNum := record.DeviceNumber.Int64(); ppkNum > 0 {
			ppkFilter[ppkNum] = struct{}{}
		}
	}

	byPPK, err = p.loadEventContextsByPPK(ctx, ppkFilter)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося побудувати PPK контексти")
		byPPK = make(map[int64]caslEventContext)
		err = nil // не фатально — продовжуємо з порожнім byPPK
	}

	byObject = p.loadEventContextsByObjectNum(ctx, objFilter, byPPK)
	return byPPK, byObject, nil
}

// caslGetEventsHTTPTimeout — короткий таймаут для HTTP-опитування в GetEvents.
// Менший за timeout агрегатора, щоб не блокувати загальне оновлення журналу.
const caslGetEventsHTTPTimeout = 1500 * time.Millisecond

func (p *CASLCloudProvider) GetEvents() []models.Event {
	// Realtime bootstrap не має блокувати отримання журналу подій.
	// Якщо websocket ще не готовий, піднімаємо його у фоні, а сам GetEvents
	// працює через коротке HTTP-опитування або віддає кеш.
	p.ensureRealtimeStreamAsync()

	p.mu.RLock()
	token := p.token
	p.mu.RUnlock()
	if token == "" {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return append([]models.Event(nil), p.cachedEvents...)
	}

	// Якщо WebSocket-підписка активна — повертаємо кеш одразу.
	// Realtime loop підтримує cachedEvents свіжими через mergeCachedEventsLocked.
	p.realtimeMu.Lock()
	subscribed := p.realtimeRunning && p.realtimeSubscribed
	p.realtimeMu.Unlock()

	if subscribed {
		p.mu.RLock()
		defer p.mu.RUnlock()
		return append([]models.Event(nil), p.cachedEvents...)
	}

	// Стрім ще не підключений — HTTP-опитування для початкового завантаження.
	// Використовуємо короткий таймаут, щоб не перевищувати ліміт combined provider.
	ctx, cancel := context.WithTimeout(context.Background(), caslGetEventsHTTPTimeout)
	defer cancel()
	if events, err := p.readEventsJournalAsEvents(ctx); err == nil {
		return events
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *CASLCloudProvider) FrontendSourceHealth() contracts.FrontendSourceHealthInfo {
	if p == nil {
		return contracts.FrontendSourceHealthInfo{
			HealthStatus:   contracts.FrontendSourceHealthStatusUnknown,
			APIStatus:      contracts.FrontendConnectionStatusUnknown,
			RealtimeStatus: contracts.FrontendConnectionStatusUnknown,
		}
	}

	p.ensureRealtimeStreamAsync()

	now := time.Now()
	p.mu.RLock()
	apiSuccessAt := p.lastAPISuccessAt
	apiFailureAt := p.lastAPIFailureAt
	apiErrorText := strings.TrimSpace(p.lastAPIErrorText)
	lastRealtimeMsgAt := p.lastRealtimeMsgAt
	lastRealtimePingAt := p.lastRealtimePingAt
	wsURL := strings.TrimSpace(p.wsURL)
	p.mu.RUnlock()

	p.realtimeMu.Lock()
	realtimeRunning := p.realtimeRunning
	realtimeSubscribed := p.realtimeSubscribed
	p.realtimeMu.Unlock()

	apiStatus := contracts.FrontendConnectionStatusUnknown
	switch {
	case !apiSuccessAt.IsZero() &&
		(apiFailureAt.IsZero() || !apiFailureAt.After(apiSuccessAt) || now.Sub(apiSuccessAt) <= caslAPIHealthGrace):
		apiStatus = contracts.FrontendConnectionStatusOnline
	case !apiFailureAt.IsZero() && (apiSuccessAt.IsZero() || apiFailureAt.After(apiSuccessAt)):
		apiStatus = contracts.FrontendConnectionStatusOffline
	}

	realtimeStatus := contracts.FrontendConnectionStatusUnknown
	switch {
	case realtimeSubscribed && !lastRealtimePingAt.IsZero() && now.Sub(lastRealtimePingAt) <= caslRealtimePingGracePeriod:
		realtimeStatus = contracts.FrontendConnectionStatusOnline
	case realtimeSubscribed && !lastRealtimeMsgAt.IsZero() && now.Sub(lastRealtimeMsgAt) <= caslRealtimePingGracePeriod:
		realtimeStatus = contracts.FrontendConnectionStatusOnline
	case realtimeRunning || realtimeSubscribed || wsURL != "" || p.canRelogin():
		realtimeStatus = contracts.FrontendConnectionStatusOffline
	}

	healthStatus := contracts.FrontendSourceHealthStatusUnknown
	healthText := "CASL: стан з'єднання ще не визначено"

	switch {
	case apiStatus == contracts.FrontendConnectionStatusOnline && realtimeStatus == contracts.FrontendConnectionStatusOnline:
		healthStatus = contracts.FrontendSourceHealthStatusOnline
		healthText = "CASL: API і WS online"
	case apiStatus == contracts.FrontendConnectionStatusOffline && realtimeStatus == contracts.FrontendConnectionStatusOffline:
		healthStatus = contracts.FrontendSourceHealthStatusOffline
		healthText = "CASL: API і WS недоступні"
	case apiStatus == contracts.FrontendConnectionStatusOffline:
		healthStatus = contracts.FrontendSourceHealthStatusDegraded
		if apiErrorText != "" {
			healthText = "CASL: API недоступний, WS нестабільний. " + apiErrorText
		} else {
			healthText = "CASL: API недоступний, WS нестабільний"
		}
	case realtimeStatus == contracts.FrontendConnectionStatusOffline:
		healthStatus = contracts.FrontendSourceHealthStatusDegraded
		if !lastRealtimePingAt.IsZero() {
			healthText = "CASL: API online, але WS не отримує ping понад 12 с"
		} else {
			healthText = "CASL: API online, але WS не підписаний або не отримує ping"
		}
	}

	return contracts.FrontendSourceHealthInfo{
		HealthStatus:     healthStatus,
		HealthText:       healthText,
		APIStatus:        apiStatus,
		RealtimeStatus:   realtimeStatus,
		LastRealtimePing: lastRealtimePingAt,
	}
}

func (p *CASLCloudProvider) readEventsJournalAsEvents(ctx context.Context) ([]models.Event, error) {
	now := time.Now().UnixMilli()
	p.mu.RLock()
	start := p.eventsCursorMs
	startGate := p.eventsStartAtMs
	if start <= 0 {
		start = p.eventsStartAtMs
	}
	if start <= 0 {
		start = now
	}
	p.mu.RUnlock()

	rows, err := p.ReadEventsJournalNoRelogin(ctx, CASLReadEventsRequest{
		TimeStart:   start,
		TimeEnd:     now,
		TimeRequest: now,
	})
	if err != nil {
		return nil, err
	}
	logCASLReadEventsRows(start, now, rows)

	events, maxEventTime := p.mapCASLRowsToEvents(ctx, rows, startGate)
	// p.updateRealtimeAlarmsFromRows(ctx, rows) // ВИДАЛЕНО: події з журналу не повинні потрапляти в тривоги
	p.mu.Lock()
	if maxEventTime > p.eventsCursorMs {
		p.eventsCursorMs = maxEventTime
	}
	if now > p.eventsCursorMs {
		p.eventsCursorMs = now
	}
	added := p.mergeCachedEventsLocked(events)
	if added > 0 {
		p.eventsRevision++
	}
	cached := append([]models.Event(nil), p.cachedEvents...)
	p.mu.Unlock()

	return cached, nil
}

func logCASLReadEventsRows(start, end int64, rows []CASLObjectEvent) {
	log.Debug().
		Int64("time_start", start).
		Int64("time_end", end).
		Int("rows", len(rows)).
		Msg("CASL read_events: отримано події")

	const maxRowsToLog = 200
	for idx, row := range rows {
		if idx >= maxRowsToLog {
			log.Debug().
				Int("logged_rows", maxRowsToLog).
				Int("total_rows", len(rows)).
				Msg("CASL read_events: лог скорочено")
			return
		}

		log.Debug().
			Int("idx", idx).
			Int64("ppk_num", row.PPKNum).
			Str("device_id", strings.TrimSpace(row.DeviceID)).
			Str("obj_id", strings.TrimSpace(row.ObjID)).
			Str("code", strings.TrimSpace(row.Code)).
			Str("contact_id", strings.TrimSpace(row.ContactID)).
			Int64("number", row.Number).
			Str("type", strings.TrimSpace(row.Type)).
			Int64("time", row.Time).
			Msg("CASL read_events row")
	}
}

func (p *CASLCloudProvider) mapCASLRowsToEvents(ctx context.Context, rows []CASLObjectEvent, startGate int64) ([]models.Event, int64) {
	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	filteredRows := make([]CASLObjectEvent, 0, len(rows))
	resolvedByDeviceID := make(map[string]int64)
	unresolvedByDeviceID := make(map[string]struct{})
	maxEventTime := int64(0)
	for _, sourceRow := range rows {
		row := sourceRow
		if row.PPKNum <= 0 {
			deviceID := strings.TrimSpace(row.DeviceID)
			if deviceID != "" {
				if resolved, ok := resolvedByDeviceID[deviceID]; ok {
					row.PPKNum = resolved
				} else if _, failed := unresolvedByDeviceID[deviceID]; !failed {
					if resolved, ok := p.resolvePPKByDeviceID(ctx, deviceID); ok {
						resolvedByDeviceID[deviceID] = resolved
						row.PPKNum = resolved
					} else {
						unresolvedByDeviceID[deviceID] = struct{}{}
					}
				}
			}
		}
		rawObjID := strings.TrimSpace(row.ObjID)
		sourceType := effectiveCASLSourceType(row)
		if row.PPKNum <= 0 && rawObjID == "" && !isCASLActionSource(sourceType) {
			continue
		}
		if row.Time <= 0 {
			continue
		}
		if startGate > 0 && row.Time < startGate {
			continue
		}
		if row.Time > maxEventTime {
			maxEventTime = row.Time
		}
		filteredRows = append(filteredRows, row)
		if row.PPKNum > 0 {
			ppkFilter[row.PPKNum] = struct{}{}
		}
		if rawObjID != "" {
			objFilter[rawObjID] = struct{}{}
		}
	}
	if len(filteredRows) == 0 {
		return nil, maxEventTime
	}

	contextByPPK, err := p.loadEventContextsByPPK(ctx, ppkFilter)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося побудувати контексти ППК для журналу подій")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)
	users := map[string]caslUser(nil)
	if shouldLoadCASLEventUsers(filteredRows) {
		if loadedUsers, err := p.loadUsers(ctx); err == nil {
			users = loadedUsers
		} else {
			log.Debug().Err(err).Msg("CASL: не вдалося завантажити користувачів для підписів подій")
		}
	}

	events := make([]models.Event, 0, len(filteredRows))
	for _, row := range filteredRows {
		ppkNum := row.PPKNum
		number := int(row.Number)
		code := strings.TrimSpace(row.Code)
		contactID := strings.TrimSpace(row.ContactID)
		sourceType := effectiveCASLSourceType(row)
		rawObjID := strings.TrimSpace(row.ObjID)

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := 0
		if ppkNum > 0 {
			objectID = mapCASLObjectID(strconv.FormatInt(ppkNum, 10))
		} else if rawObjID != "" {
			objectID = mapCASLObjectID(rawObjID)
		}
		objectName := strings.TrimSpace(row.ObjName)
		if objectName == "" {
			if ppkNum > 0 {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			} else if rawObjID != "" {
				objectName = "Об'єкт #" + rawObjID
			} else if isCASLActionSource(sourceType) {
				objectName = "CASL система"
			} else {
				objectName = "Об'єкт"
			}
		}
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)
		if hasCtx {
			objectID = ctxItem.ObjectID
			objectNum = ctxItem.ObjectNum
			objectName = ctxItem.ObjectName
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		translator := map[string]string(nil)
		lineInfos := map[int]caslEventLineInfo(nil)
		translatorAlarms := map[string]bool(nil)
		deviceType := ""
		if hasCtx {
			translator = ctxItem.Translator
			lineInfos = ctxItem.LineInfos
			translatorAlarms = ctxItem.TranslatorAlarms
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := buildCASLUserActionDetails(row, dictMap, users)
		if details == "" && isCASLPPKMessageSource(sourceType) {
			details = buildCASLPPKEventDetails(row, translator, dictMap, deviceType, lineInfos, users)
		}
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, number, deviceType)
		}
		if details == "" && isCASLActionSource(sourceType) {
			details = fallbackCASLActionDetails(row, sourceType)
		}
		if details == "" {
			switch {
			case contactID != "" && code != "":
				details = fmt.Sprintf("%s (%s)", contactID, code)
			case contactID != "":
				details = contactID
			case code != "":
				details = code
			default:
				details = "CASL подія"
			}
		}
		classifierCode := resolveCASLEventClassificationKey(translator, code, contactID, deviceType, number, lineInfos)
		if classifierCode == "" {
			classifierCode = strings.TrimSpace(row.Action)
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, contactID, sourceType, details)
		if isAlarm, ok := resolveCASLAlarmFlagFromMap(translatorAlarms, classifierCode, contactID, strings.TrimSpace(row.Subtype)); ok {
			eventType = classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
		}
		eventTime := time.UnixMilli(row.Time).Local()
		eventTS := row.Time

		seed := stableCASLAlarmSeed(code, contactID, number)
		// objectNum := objectNums[rawObjID]
		if objectNum == "" {
			objectNum = rawObjID
		}

		events = append(events, models.Event{
			ID:           stableCASLEventID(firstCASLValue(strconv.FormatInt(ppkNum, 10), rawObjID, sourceType), eventTS, seed, 0),
			Time:         eventTime,
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Type:         eventType,
			TypeLabel:    caslActionTypeLabel(sourceType),
			ZoneNumber:   number,
			Details:      details,
			SC1:          mapCASLEventSC1(eventType),
			Source:       models.EventSourceCASL,
		})
	}

	sortEvents(events)
	return events, maxEventTime
}

func (p *CASLCloudProvider) loadEventContextsByObjectNum(ctx context.Context, objFilter map[string]struct{}, byPPK map[int64]caslEventContext) map[string]caslEventContext {
	if len(objFilter) == 0 {
		return nil
	}

	contexts := make(map[string]caslEventContext, len(objFilter))
	for _, ctxItem := range byPPK {
		if objNum := strings.TrimSpace(ctxItem.ObjectNum); objNum != "" {
			if _, need := objFilter[objNum]; need {
				contexts[objNum] = ctxItem
			}
		}
	}
	if len(contexts) == len(objFilter) {
		return contexts
	}

	records, err := p.loadObjects(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити об'єкти для object context")
		return contexts
	}
	_, _ = p.loadDevices(ctx)

	for _, record := range records {
		objNum := strings.TrimSpace(record.ObjID)
		if objNum == "" {
			continue
		}
		if _, need := objFilter[objNum]; !need {
			continue
		}
		if _, exists := contexts[objNum]; exists {
			continue
		}

		ppkNum := record.DeviceNumber.Int64()
		if ppkNum > 0 {
			if ctxItem, ok := byPPK[ppkNum]; ok {
				contexts[objNum] = ctxItem
				continue
			}
		}

		ctxItem := caslEventContext{
			ObjectID:  mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(ppkNum, 10)),
			ObjectNum: preferredCASLObjectNumber(record.ObjID, record.Name, ppkNum),
		}
		ctxItem.ObjectName = strings.TrimSpace(record.Name)
		if ctxItem.ObjectName == "" {
			ctxItem.ObjectName = "Об'єкт #" + ctxItem.ObjectNum
		}
		ctxItem.ObjectName = formatCASLJournalObjectName(ctxItem.ObjectNum, ctxItem.ObjectName)

		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice {
			ctxItem.DeviceType = strings.TrimSpace(device.Type.String())
			ctxItem.LineNames = buildCASLLineNameIndex(device.Lines)
			ctxItem.LineInfos = p.buildCASLLineInfoIndex(ctx, device.Lines)
			ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
			ctxItem.TranslatorAlarms = p.loadTranslatorAlarmFlags(ctx, ctxItem.DeviceType)
		}
		contexts[objNum] = ctxItem
	}

	return contexts
}

func (p *CASLCloudProvider) loadEventContextsByPPK(ctx context.Context, ppkFilter map[int64]struct{}) (map[int64]caslEventContext, error) {
	if len(ppkFilter) == 0 {
		return nil, nil
	}

	contexts := make(map[int64]caslEventContext, len(ppkFilter))

	// Завантажуємо об'єкти щоб отримати інформацію про них
	records, err := p.loadObjects(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити об'єкти для PPK context")
		records = nil
	}

	// Завантажуємо пристрої для інформації про тип та лінії
	devices, devErr := p.loadDevices(ctx)
	if devErr != nil {
		log.Debug().Err(devErr).Msg("CASL: не вдалося завантажити пристрої для PPK context")
		return contexts, devErr
	}

	// Будуємо індекс об'єктів за їх DeviceNumber для швидкого пошуку
	objectByPPK := make(map[int64]caslGrdObject)
	for _, record := range records {
		ppkNum := record.DeviceNumber.Int64()
		if ppkNum > 0 {
			if _, exists := objectByPPK[ppkNum]; !exists {
				objectByPPK[ppkNum] = record
			}
		}
	}

	// Для кожного потрібного ППК номера будуємо контекст
	for _, device := range devices {
		ppkNum := device.Number.Int64()
		if ppkNum <= 0 {
			continue
		}

		if _, need := ppkFilter[ppkNum]; !need {
			continue
		}

		if _, exists := contexts[ppkNum]; exists {
			continue
		}

		ctxItem := caslEventContext{
			ObjectID:   0, // За замовчуванням
			ObjectNum:  strconv.FormatInt(ppkNum, 10),
			DeviceType: strings.TrimSpace(device.Type.String()),
		}

		// Якщо є об'єкт з цим DeviceNumber, заповнюємо ObjectID та ObjectNum з нього
		if objRecord, hasObj := objectByPPK[ppkNum]; hasObj {
			ctxItem.ObjectID = mapCASLObjectID(objRecord.ObjID, objRecord.Name, strconv.FormatInt(ppkNum, 10))
			ctxItem.ObjectNum = preferredCASLObjectNumber(objRecord.ObjID, objRecord.Name, ppkNum)
			ctxItem.ObjectName = strings.TrimSpace(objRecord.Name)
			if ctxItem.ObjectName == "" {
				ctxItem.ObjectName = "Об'єкт #" + ctxItem.ObjectNum
			}
		} else {
			// Якщо немає об'єкту, використовуємо ім'я пристрою
			ctxItem.ObjectName = strings.TrimSpace(device.Name.String())
			if ctxItem.ObjectName == "" {
				ctxItem.ObjectName = "Пристрій ППК #" + ctxItem.ObjectNum
			}
		}
		ctxItem.ObjectName = formatCASLJournalObjectName(ctxItem.ObjectNum, ctxItem.ObjectName)

		ctxItem.LineNames = buildCASLLineNameIndex(device.Lines)
		ctxItem.LineInfos = p.buildCASLLineInfoIndex(ctx, device.Lines)
		ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
		ctxItem.TranslatorAlarms = p.loadTranslatorAlarmFlags(ctx, ctxItem.DeviceType)

		contexts[ppkNum] = ctxItem
	}

	return contexts, nil
}

func (p *CASLCloudProvider) mergeCachedEventsLocked(events []models.Event) int {
	if len(events) == 0 {
		return 0
	}

	seen := make(map[int]struct{}, len(p.cachedEvents)+len(events))
	for _, item := range p.cachedEvents {
		seen[item.ID] = struct{}{}
	}
	added := 0
	for _, item := range events {
		if _, exists := seen[item.ID]; exists {
			continue
		}
		seen[item.ID] = struct{}{}
		p.cachedEvents = append(p.cachedEvents, item)
		added++
	}
	if added > 0 {
		sortEvents(p.cachedEvents)
	}
	if len(p.cachedEvents) > caslMaxCachedEvents {
		p.cachedEvents = p.cachedEvents[:caslMaxCachedEvents]
	}
	return added
}

func normalizeCASLGeneralTapeRow(row map[string]any) CASLObjectEvent {
	action := firstCASLValue(
		asString(row["action"]),
		asString(row["dict_name"]),
		asString(row["last_act"]),
	)
	code := firstCASLValue(
		asString(row["code"]),
		asString(row["event_code"]),
		action,
	)
	rowType := firstCASLValue(
		asString(row["event_type"]),
		asString(row["type"]),
	)
	if rowType == "" && strings.HasPrefix(strings.ToUpper(strings.TrimSpace(action)), "GRD_OBJ_") {
		rowType = "user_action"
	}

	return CASLObjectEvent{
		PPKNum:    int64(parseCASLAnyInt(row["ppk_num"])),
		DeviceID:  firstCASLValue(asString(row["device_id"]), asString(row["deviceId"])),
		ObjID:     firstCASLValue(asString(row["obj_id"]), asString(row["object_id"])),
		ObjName:   firstCASLValue(asString(row["obj_name"]), asString(row["name"])),
		ObjAddr:   firstCASLValue(asString(row["obj_address"]), asString(row["address"])),
		Action:    action,
		AlarmType: strings.TrimSpace(asString(row["alarm_type"])),
		MgrID:     strings.TrimSpace(asString(row["mgr_id"])),
		UserID:    strings.TrimSpace(asString(row["user_id"])),
		UserFIO:   firstCASLValue(asString(row["user_fio"]), asString(row["userFio"])),
		Time:      int64(parseCASLAnyInt(row["time"])),
		Code:      code,
		Type:      rowType,
		Subtype:   firstCASLValue(asString(row["type_event"]), asString(row["typeEvent"])),
		Number:    int64(firstCASLNonZeroInt(parseCASLAnyInt(row["zone"]), parseCASLAnyInt(row["number"]), parseCASLAnyInt(row["num"]))),
		ContactID: firstCASLValue(asString(row["contact_id"]), asString(row["contactId"])),
		HozUserID: strings.TrimSpace(asString(row["hoz_user_id"])),
	}
}

func firstCASLNonZeroInt(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (p *CASLCloudProvider) classifyCASLAlarmEvent(
	ctx context.Context,
	code string,
	contactID string,
	sourceType string,
	details string,
	deviceType string,
	subtype string,
	translatorAlarms map[string]bool,
) models.EventType {
	eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)
	if isAlarm, ok := resolveCASLAlarmFlagFromMap(translatorAlarms, code, contactID, subtype); ok {
		return classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
	}
	if isAlarm, ok := p.resolveCASLAlarmFlag(ctx, deviceType, code, contactID, subtype); ok {
		return classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
	}
	return eventType
}

func (p *CASLCloudProvider) decodeCASLReasonAlarmDetails(
	ctx context.Context,
	raw string,
	translator map[string]string,
	translatorAlarms map[string]bool,
	dictMap map[string]string,
	deviceType string,
	zoneNumber int,
) (string, models.EventType, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", models.EventFault, false
	}

	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		var payload any
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			if details, eventType, ok := p.decodeCASLReasonAlarmValue(ctx, payload, translator, translatorAlarms, dictMap, deviceType, zoneNumber); ok {
				return details, eventType, true
			}
		}
	}

	details := decodeCASLEventDescription(translator, dictMap, raw, "", zoneNumber, deviceType)
	if details == "" {
		details = decodeCASLEventDescription(nil, dictMap, raw, "", zoneNumber, deviceType)
	}
	if details == "" {
		return "", models.EventFault, false
	}

	eventType := p.classifyCASLAlarmEvent(ctx, raw, "", "alarm", details, deviceType, "", translatorAlarms)
	return details, eventType, true
}

func (p *CASLCloudProvider) decodeCASLReasonAlarmValue(
	ctx context.Context,
	payload any,
	translator map[string]string,
	translatorAlarms map[string]bool,
	dictMap map[string]string,
	deviceType string,
	zoneNumber int,
) (string, models.EventType, bool) {
	switch value := payload.(type) {
	case string:
		return p.decodeCASLReasonAlarmDetails(ctx, value, translator, translatorAlarms, dictMap, deviceType, zoneNumber)
	case []any:
		for _, item := range value {
			if details, eventType, ok := p.decodeCASLReasonAlarmValue(ctx, item, translator, translatorAlarms, dictMap, deviceType, zoneNumber); ok {
				return details, eventType, true
			}
		}
		return "", models.EventFault, false
	case map[string]any:
		for _, key := range []string{"text", "message", "description", "title"} {
			if text := strings.TrimSpace(asString(value[key])); text != "" {
				eventType := classifyCASLEventTypeWithContext("", "", "alarm", text)
				return text, eventType, true
			}
		}

		reasonZone := zoneNumber
		for _, key := range []string{"num", "number", "zone"} {
			if parsed := parseCASLAnyInt(value[key]); parsed > 0 {
				reasonZone = parsed
				break
			}
		}

		code := firstCASLValue(
			asString(value["code"]),
			asString(value["dict_name"]),
			asString(value["msg"]),
			asString(value["name"]),
		)
		contactID := firstCASLValue(
			asString(value["contact_id"]),
			asString(value["contactId"]),
		)
		if code != "" || contactID != "" {
			details := decodeCASLEventDescription(translator, dictMap, code, contactID, reasonZone, deviceType)
			if details == "" {
				details = decodeCASLEventDescription(nil, dictMap, code, contactID, reasonZone, deviceType)
			}
			if details == "" && code != "" {
				details = strings.TrimSpace(code)
			}
			if details != "" {
				eventType := p.classifyCASLAlarmEvent(ctx, code, contactID, "alarm", details, deviceType, "", translatorAlarms)
				return details, eventType, true
			}
		}
	}

	return "", models.EventFault, false
}

func stringifyCASLTapeItemMsg(raw any) string {
	switch typed := raw.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		payload, err := json.Marshal(raw)
		if err != nil {
			return strings.TrimSpace(fmt.Sprintf("%v", raw))
		}
		return strings.TrimSpace(string(payload))
	}
}

func resolveCASLTapeMessageKey(code string, dictName string, deviceType string) string {
	if key := strings.ToUpper(strings.TrimSpace(dictName)); key != "" {
		return key
	}
	if decoded, ok := decodeCASLProtocolCode(code, deviceType); ok {
		if key := strings.ToUpper(strings.TrimSpace(decoded.MessageKey)); key != "" {
			return key
		}
	}
	return strings.ToUpper(strings.TrimSpace(code))
}

func mapCASLTapeMessagesToAlarmMsgs(messages []caslTapeMessage) []models.AlarmMsg {
	if len(messages) == 0 {
		return nil
	}
	result := make([]models.AlarmMsg, 0, len(messages))
	for _, msg := range messages {
		result = append(result, models.AlarmMsg{
			Time:      time.UnixMilli(msg.Time).Local(),
			Code:      strings.TrimSpace(msg.Code),
			ContactID: strings.TrimSpace(msg.ContactID),
			Number:    msg.Number,
			Details:   strings.TrimSpace(msg.Details),
			SC1:       mapCASLEventSC1(msg.Type),
			IsAlarm:   msg.IsAlarm,
		})
	}
	return result
}

func (p *CASLCloudProvider) mapCASLHistoryRowToTapeMessage(
	ctx context.Context,
	rawRow map[string]any,
	translator map[string]string,
	translatorAlarms map[string]bool,
	dictMap map[string]string,
	deviceType string,
) (caslTapeMessage, bool) {
	row := normalizeCASLGeneralTapeRow(rawRow)
	if row.Time <= 0 {
		return caslTapeMessage{}, false
	}

	action := strings.ToUpper(strings.TrimSpace(row.Action))
	// We no longer filter out GRD_OBJ_* or internal actions for tape messages,
	// because they are often part of the alarm story (e.g. who picked it up).
	if action == "GRD_OBJ_FINISH" {
		return caslTapeMessage{}, false
	}

	details := strings.TrimSpace(asString(rawRow["description"]))
	var (
		eventType models.EventType
		hasType   bool
	)
	if details == "" {
		if reason, reasonType, ok := p.decodeCASLReasonAlarmDetails(ctx, asString(rawRow["reasonAlarm"]), translator, translatorAlarms, dictMap, deviceType, int(row.Number)); ok {
			details = reason
			eventType = reasonType
			hasType = true
		}
	}
	if details == "" {
		details = decodeCASLEventDescription(translator, dictMap, row.Code, row.ContactID, int(row.Number), deviceType)
	}
	if details == "" {
		details = strings.TrimSpace(row.Action)
	}
	if details == "" {
		details = strings.TrimSpace(row.Code)
	}
	if details == "" {
		return caslTapeMessage{}, false
	}

	classifierCode := strings.TrimSpace(row.Code)
	if classifierCode == "" {
		classifierCode = strings.TrimSpace(row.Action)
	}
	if !hasType {
		eventType = classifyCASLEventTypeWithContext(classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Type), details)
	}

	isAlarm, hasAlarmFlag := resolveCASLAlarmFlagFromMap(translatorAlarms, classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Subtype))
	if !hasAlarmFlag {
		if resolved, ok := p.resolveCASLAlarmFlag(ctx, deviceType, classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Subtype)); ok {
			isAlarm = resolved
			hasAlarmFlag = true
		}
	}
	if hasAlarmFlag {
		eventType = classifyCASLActiveAlarmEventType(eventType, isAlarm, true)
	}

	messageKey := resolveCASLTapeMessageKey(row.Code, asString(rawRow["dict_name"]), deviceType)
	isAlarmCandidate := false
	if !isCASLNotAlarmMessageKey(messageKey) {
		if isCASLEventAlarmCandidate(eventType) {
			if _, include := mapEventTypeToAlarmType(eventType); include {
				isAlarmCandidate = true
			}
		}
	}

	return caslTapeMessage{
		Time:         row.Time,
		Code:         strings.TrimSpace(row.Code),
		DictName:     strings.TrimSpace(asString(rawRow["dict_name"])),
		ContactID:    strings.TrimSpace(row.ContactID),
		Number:       int(row.Number),
		EventType:    strings.TrimSpace(row.Type),
		Subtype:      strings.TrimSpace(row.Subtype),
		Details:      details,
		MessageKey:   messageKey,
		Type:         eventType,
		IsAlarm:      isAlarmCandidate,
		HasAlarmFlag: hasAlarmFlag,
	}, true
}

func (p *CASLCloudProvider) mapCASLTapeMessagesFromAny(
	ctx context.Context,
	raw any,
	translator map[string]string,
	translatorAlarms map[string]bool,
	dictMap map[string]string,
	deviceType string,
) []caslTapeMessage {
	if raw == nil {
		return nil
	}

	var rows []map[string]any
	switch typed := raw.(type) {
	case []map[string]any:
		rows = append(rows, typed...)
	case []any:
		for _, item := range typed {
			if row, ok := item.(map[string]any); ok {
				rows = append(rows, row)
			}
		}
	}
	if len(rows) == 0 {
		return nil
	}

	messages := make([]caslTapeMessage, 0, len(rows))
	for _, row := range rows {
		if msg, ok := p.mapCASLHistoryRowToTapeMessage(ctx, row, translator, translatorAlarms, dictMap, deviceType); ok {
			messages = append(messages, msg)
		}
	}
	if len(messages) > 1 {
		sort.SliceStable(messages, func(i, j int) bool {
			return messages[i].Time > messages[j].Time
		})
	}
	return messages
}

func selectCASLTapeCauseMessage(messages []caslTapeMessage) (caslTapeMessage, bool) {
	if len(messages) == 0 {
		return caslTapeMessage{}, false
	}

	finishIndex := -1
	for i := range messages {
		key := strings.ToUpper(strings.TrimSpace(firstCASLValue(messages[i].DictName, messages[i].MessageKey, messages[i].Code)))
		if key == "GRD_OBJ_FINISH" {
			finishIndex = i
			break
		}
	}

	start := len(messages) - 1
	if finishIndex >= 0 {
		start = finishIndex
	}

	for i := start; i >= 0; i-- {
		msg := messages[i]
		key := strings.ToUpper(strings.TrimSpace(firstCASLValue(msg.MessageKey, msg.DictName, msg.Code)))
		if key == "" {
			continue
		}
		if isCASLNotAlarmMessageKey(key) {
			continue
		}
		if !msg.IsAlarm {
			continue
		}
		return msg, true
	}
	return caslTapeMessage{}, false
}

func (p *CASLCloudProvider) mergeCASLTapeMessages(primary []caslTapeMessage, secondary []caslTapeMessage) []caslTapeMessage {
	if len(secondary) == 0 {
		return primary
	}
	if len(primary) == 0 {
		return secondary
	}

	out := append([]caslTapeMessage(nil), primary...)
	seen := make(map[string]struct{}, len(out))
	for _, m := range out {
		seen[fmt.Sprintf("%d|%s", m.Time, m.Code)] = struct{}{}
	}

	for _, m := range secondary {
		key := fmt.Sprintf("%d|%s", m.Time, m.Code)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, m)
	}

	if len(out) > 1 {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].Time > out[j].Time
		})
	}
	return out
}

func (p *CASLCloudProvider) findCASLGeneralTapeAlarmCauseInHistory(
	ctx context.Context,
	historyRows []map[string]any,
	translator map[string]string,
	translatorAlarms map[string]bool,
	dictMap map[string]string,
	deviceType string,
) (string, models.EventType, []caslTapeMessage, bool) {
	messages := make([]caslTapeMessage, 0, len(historyRows))
	for _, rawRow := range historyRows {
		if msg, ok := p.mapCASLHistoryRowToTapeMessage(ctx, rawRow, translator, translatorAlarms, dictMap, deviceType); ok {
			messages = append(messages, msg)
		}
	}
	if len(messages) > 1 {
		sort.SliceStable(messages, func(i, j int) bool {
			return messages[i].Time > messages[j].Time
		})
	}
	cause, ok := selectCASLTapeCauseMessage(messages)
	if !ok {
		return "", models.EventFault, messages, false
	}
	return strings.TrimSpace(cause.Details), cause.Type, messages, true
}

func (p *CASLCloudProvider) readGeneralTapeAsAlarms(ctx context.Context, byObject map[string]caslEventContext) ([]models.Alarm, error) {
	rows, err := p.ReadGeneralTapeObjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	dictMap := p.loadDictionaryMap(ctx)
	resolvedByDeviceID := make(map[string]int64)
	unresolvedByDeviceID := make(map[string]struct{})

	historyByObject := map[string][]map[string]any(nil)
	objIDs := selectCASLGeneralTapeInitialItemObjectIDs(rows, caslGeneralTapeInitialItemLimit)
	if len(objIDs) > 0 {
		historyByObject, err = p.ReadGeneralTapeItem(ctx, objIDs)
		if err != nil {
			log.Debug().Err(err).Msg("CASL: не вдалося enrich-нути active tape через get_general_tape_item")
			historyByObject = nil
		}
	}

	currentUserID := p.currentCASLUserID()
	users := map[string]caslUser(nil)
	if shouldLoadCASLGeneralTapeUsers(rows) {
		if loadedUsers, err := p.loadUsers(ctx); err == nil {
			users = loadedUsers
		} else {
			log.Debug().Err(err).Msg("CASL: не вдалося завантажити користувачів для стану обробки тривог")
		}
	}

	items := make([]caslTapeItem, 0, len(rows))
	for idx, rawRow := range rows {
		row := normalizeCASLGeneralTapeRow(rawRow)
		if row.Time <= 0 {
			continue
		}

		ppkNum := row.PPKNum
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(rawRow["device_number"]))
		}
		if ppkNum <= 0 {
			ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, row.DeviceID, resolvedByDeviceID, unresolvedByDeviceID)
		}

		objectID := mapCASLObjectID(row.ObjID, strconv.FormatInt(ppkNum, 10), row.DeviceID)
		objectName := strings.TrimSpace(row.ObjName)
		if objectName == "" {
			objectName = "Об'єкт #" + strings.TrimSpace(row.ObjID)
		}
		objectNum := preferredCASLObjectNumber(row.ObjID, objectName, ppkNum)
		translator := map[string]string(nil)
		translatorAlarms := map[string]bool(nil)
		deviceType := strings.TrimSpace(asString(rawRow["device_type"]))
		if ctxItem, hasCtx := byObject[row.ObjID]; hasCtx {
			objectID = ctxItem.ObjectID
			objectNum = ctxItem.ObjectNum
			objectName = ctxItem.ObjectName
			translator = ctxItem.Translator
			translatorAlarms = ctxItem.TranslatorAlarms
			if ctxItem.DeviceType != "" {
				deviceType = ctxItem.DeviceType
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		seed := stableCASLAlarmSeed(firstCASLValue(row.Code, row.Action), row.ContactID, int(row.Number))
		objectKey := canonicalCASLRealtimeObjectKey(row.ObjID, objectNum, objectID)
		item := caslTapeItem{
			ID:              stableCASLAlarmID(objectKey, row.Time, seed+"|"+strconv.Itoa(idx)),
			Time:            row.Time,
			ObjectID:        objectID,
			ObjectNum:       objectNum,
			ObjectName:      objectName,
			ObjID:           row.ObjID,
			DeviceID:        row.DeviceID,
			DeviceType:      deviceType,
			ObjAddr:         strings.TrimSpace(row.ObjAddr),
			ZoneNumber:      int(row.Number),
			Code:            strings.TrimSpace(row.Code),
			ContactID:       strings.TrimSpace(row.ContactID),
			EventType:       strings.TrimSpace(row.Type),
			Subtype:         strings.TrimSpace(row.Subtype),
			AlarmType:       strings.TrimSpace(row.AlarmType),
			PultID:          strings.TrimSpace(asString(rawRow["pult_id"])),
			UserID:          firstCASLValue(strings.TrimSpace(row.UserID), strings.TrimSpace(asString(rawRow["user_id"]))),
			UserFIO:         firstCASLValue(strings.TrimSpace(row.UserFIO), strings.TrimSpace(asString(rawRow["user_fio"]))),
			MgrID:           firstCASLValue(strings.TrimSpace(row.MgrID), strings.TrimSpace(asString(rawRow["mgr_id"]))),
			LastAct:         firstCASLValue(strings.TrimSpace(asString(rawRow["last_act"])), strings.TrimSpace(row.Action)),
			Msg:             stringifyCASLTapeItemMsg(rawRow["msg"]),
			ReasonAlarm:     strings.TrimSpace(asString(rawRow["reasonAlarm"])),
			Translator:      translator,
			TranslatorFlags: translatorAlarms,
		}
		item.PPKMsgs = p.mapCASLTapeMessagesFromAny(ctx, rawRow["ppk_msgs"], translator, translatorAlarms, dictMap, deviceType)
		items = append(items, item)
	}

	alarms := make([]models.Alarm, 0, len(items))
	for idx := range items {
		item := items[idx]

		details, causeType, historyMsgs, hasCause := p.findCASLGeneralTapeAlarmCauseInHistory(ctx, historyByObject[item.ObjID], item.Translator, item.TranslatorFlags, dictMap, item.DeviceType)
		if len(historyMsgs) > 0 {
			item.PPKMsgs = p.mergeCASLTapeMessages(item.PPKMsgs, historyMsgs)
		}
		if hasCause && strings.TrimSpace(item.Msg) == "" {
			item.Msg = details
		}
		if !hasCause {
			details = strings.TrimSpace(item.Msg)
		}
		if !hasCause && details == "" {
			details = decodeCASLEventDescription(item.Translator, dictMap, item.Code, item.ContactID, item.ZoneNumber, item.DeviceType)
		}
		if !hasCause && details == "" {
			if reason, reasonType, ok := p.decodeCASLReasonAlarmDetails(ctx, item.ReasonAlarm, item.Translator, item.TranslatorFlags, dictMap, item.DeviceType, item.ZoneNumber); ok {
				details = reason
				causeType = reasonType
				hasCause = true
			}
		}
		if details == "" {
			details = decodeCASLEventDescription(nil, dictMap, item.Code, item.ContactID, item.ZoneNumber, item.DeviceType)
		}
		if details == "" {
			details = "CASL тривога"
		}

		eventType := causeType
		if !hasCause {
			eventType = p.classifyCASLAlarmEvent(ctx, item.Code, item.ContactID, item.EventType, details, item.DeviceType, item.Subtype, item.TranslatorFlags)
		}
		if eventType == models.EventRestore || eventType == models.EventPowerOK || eventType == models.EventOnline {
			continue
		}

		alarmType, include := mapEventTypeToAlarmType(eventType)
		if mapped, ok := mapCASLAlarmType(item.AlarmType); ok {
			alarmType = mapped
			include = true
		} else if strings.EqualFold(item.EventType, "alarm") && !include {
			alarmType = models.AlarmNotification
			include = true
		}
		if !include {
			continue
		}

		inProgressBy, isOwnedByMe := p.resolveCASLAlarmAssignee(item, users, currentUserID)
		isInProgress := isCASLAlarmInProgressAction(item.LastAct)
		groupDispatched := strings.EqualFold(strings.TrimSpace(item.LastAct), "GRD_OBJ_ASS_MGR") || strings.EqualFold(strings.TrimSpace(item.LastAct), "GRD_OBJ_MGR_ARRIVE")
		groupArrived := strings.EqualFold(strings.TrimSpace(item.LastAct), "GRD_OBJ_MGR_ARRIVE")

		alarms = append(alarms, models.Alarm{
			ID:                        item.ID,
			ObjectID:                  item.ObjectID,
			ObjectNumber:              item.ObjectNum,
			ObjectName:                item.ObjectName,
			Address:                   item.ObjAddr,
			Time:                      time.UnixMilli(item.Time).Local(),
			Details:                   details,
			Type:                      alarmType,
			ZoneNumber:                item.ZoneNumber,
			SC1:                       mapCASLEventSC1(eventType),
			IsInProgress:              isInProgress,
			InProgressBy:              inProgressBy,
			InProgressUser:            strings.TrimSpace(item.UserID),
			IsOwnedByMe:               isInProgress && isOwnedByMe,
			ResponseGroupID:           strings.TrimSpace(item.MgrID),
			IsResponseGroupDispatched: groupDispatched,
			IsResponseGroupArrived:    groupArrived,
			SourceMsgs:                mapCASLTapeMessagesToAlarmMsgs(item.PPKMsgs),
		})
	}

	sortCASLAlarms(alarms)
	return alarms, nil
}

func (p *CASLCloudProvider) replaceRealtimeAlarmsSnapshot(alarms []models.Alarm) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key := range p.realtimeAlarmByObjID {
		delete(p.realtimeAlarmByObjID, key)
	}
	if len(alarms) == 0 {
		return
	}
	if p.realtimeAlarmByObjID == nil {
		p.realtimeAlarmByObjID = make(map[string]models.Alarm, len(alarms))
	}

	for _, alarm := range alarms {
		objectKey := canonicalCASLRealtimeObjectKey("", alarm.ObjectNumber, alarm.ObjectID)
		key := canonicalCASLRealtimeAlarmKey(objectKey, alarm.ZoneNumber)
		if key == "" {
			continue
		}
		p.realtimeAlarmByObjID[key] = alarm
	}
}

func (p *CASLCloudProvider) GetObjectEvents(objectID string) []models.Event {
	now := time.Now()
	return p.getObjectEventsRange(objectID, now.Add(-caslObjectEventsSpan), now, true)
}

func (p *CASLCloudProvider) GetObjectEventsRange(objectID string, from time.Time, to time.Time) []models.Event {
	now := time.Now()
	if to.IsZero() || to.After(now) {
		to = now
	}
	if from.IsZero() || !from.Before(to) {
		from = to.Add(-caslObjectEventsSpan)
	}
	return p.getObjectEventsRange(objectID, from, to, false)
}

func (p *CASLCloudProvider) getObjectEventsRange(objectID string, from time.Time, to time.Time, useDefaultCache bool) []models.Event {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	now := time.Now()

	if useDefaultCache {
		p.mu.RLock()
		ts, cached := p.cachedObjectEventsAt[internalID]
		if cached && now.Sub(ts) <= caslObjectEventsTTL {
			events := append([]models.Event(nil), p.cachedObjectEvents[internalID]...)
			p.mu.RUnlock()
			return events
		}
		p.mu.RUnlock()
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	rawEvents, err := p.readEventsByIDRange(ctx, record, from, to)
	if err != nil {
		log.Debug().Err(err).Int("objectID", internalID).Msg("CASL: не вдалося отримати події об'єкта")
	}

	events := p.mapCASLObjectEvents(ctx, record, rawEvents)
	if historyRows, historyErr := p.readGeneralTapeItemRowsForObjectIDs(ctx, []string{strings.TrimSpace(record.ObjID)}); historyErr != nil {
		log.Debug().Err(historyErr).Int("objectID", internalID).Msg("CASL: не вдалося отримати історію кейсу через get_general_tape_item")
	} else if len(historyRows) > 0 {
		historyEvents, _ := p.mapCASLRowsToEvents(ctx, historyRows, 0)
		events = mergeCASLObjectEvents(events, historyEvents)
	}
	events = filterEventsByTimeRange(events, from, to)
	sortEvents(events)

	if useDefaultCache {
		p.mu.Lock()
		p.cachedObjectEvents[internalID] = append([]models.Event(nil), events...)
		p.cachedObjectEventsAt[internalID] = now
		p.mu.Unlock()
	}

	return events
}

func (p *CASLCloudProvider) GetAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	if len(alarm.SourceMsgs) == 0 {
		return nil
	}
	return append([]models.AlarmMsg(nil), alarm.SourceMsgs...)
}

func mergeCASLObjectEvents(primary []models.Event, secondary []models.Event) []models.Event {
	if len(primary) == 0 {
		return append([]models.Event(nil), secondary...)
	}
	if len(secondary) == 0 {
		return append([]models.Event(nil), primary...)
	}

	out := append([]models.Event(nil), primary...)
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[caslObjectEventMergeKey(item)] = struct{}{}
	}

	for _, item := range secondary {
		key := caslObjectEventMergeKey(item)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}

	return out
}

func caslObjectEventMergeKey(event models.Event) string {
	return strings.Join([]string{
		strconv.Itoa(event.ObjectID),
		strconv.FormatInt(event.Time.UnixMilli(), 10),
		string(event.Type),
		strconv.Itoa(event.ZoneNumber),
		strings.TrimSpace(event.Details),
	}, "|")
}

func isCASLAlarmInProgressAction(action string) bool {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "GRD_OBJ_PICK", "GRD_OBJ_HIJACK":
		return true
	default:
		return false
	}
}

func isCASLAlarmFinishBlockedAction(action string) bool {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "GRD_OBJ_ASS_MGR", "GRD_OBJ_MGR_ARRIVE":
		return true
	default:
		return false
	}
}

func (p *CASLCloudProvider) refreshCASLAlarmSnapshot(ctx context.Context, forceTapeSnapshot bool) {
	// Кожен підзапит отримує власний незалежний таймаут — як оригінальний CASL
	// web-client, що шле кожен /command без спільного бюджету. Батьківський ctx
	// використовується лише як сигнал дострокового виходу між етапами.
	buildCtx, buildCancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer buildCancel()
	_, byObject, ctxErr := p.buildSharedObjectContext(buildCtx)
	if ctxErr != nil {
		log.Debug().Err(ctxErr).Msg("CASL bg: не вдалося побудувати спільний контекст об'єктів")
	}
	if ctx.Err() != nil {
		return
	}

	evCtx, evCancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer evCancel()
	if _, err := p.readEventsJournalAsEvents(evCtx); err != nil {
		log.Debug().Err(err).Msg("CASL bg: read_events недоступний")
	}
	if ctx.Err() != nil {
		return
	}

	if !forceTapeSnapshot && len(p.snapshotRealtimeAlarms()) > 0 {
		return
	}

	tapeCtx, tapeCancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer tapeCancel()
	tapeAlarms, err := p.readGeneralTapeAsAlarms(tapeCtx, byObject)
	if err != nil {
		log.Debug().Err(err).Msg("CASL bg: get_general_tape_objects недоступний")
		return
	}
	if len(tapeAlarms) > 0 || forceTapeSnapshot {
		p.replaceRealtimeAlarmsSnapshot(tapeAlarms)
	}
}

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	p.ensureRealtimeStreamAsync()

	initialLoad := len(p.snapshotRealtimeAlarms()) == 0
	p.mu.RLock()
	hasToken := strings.TrimSpace(p.token) != ""
	p.mu.RUnlock()

	if p.alarmsRefreshing.CompareAndSwap(false, true) {
		forceTape := initialLoad && hasToken
		go func() {
			defer p.alarmsRefreshing.Store(false)
			ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
			defer cancel()
			p.refreshCASLAlarmSnapshot(ctx, forceTape)
		}()
	}

	alarms := p.snapshotRealtimeAlarms()
	if len(alarms) == 0 {
		return nil
	}
	sortCASLAlarms(alarms)
	return alarms
}

func selectCASLGeneralTapeInitialItemObjectIDs(rows []map[string]any, limit int) []string {
	if limit <= 0 || len(rows) == 0 {
		return nil
	}

	capacity := limit
	if len(rows) < capacity {
		capacity = len(rows)
	}
	objIDs := make([]string, 0, capacity)
	seenObjIDs := make(map[string]struct{}, capacity)
	for _, row := range rows {
		if strings.TrimSpace(asString(row["alarm_type"])) == "" {
			continue
		}
		objID := strings.TrimSpace(asString(row["obj_id"]))
		if objID == "" {
			objID = strings.TrimSpace(asString(row["object_id"]))
		}
		if objID == "" {
			continue
		}
		if _, exists := seenObjIDs[objID]; exists {
			continue
		}
		seenObjIDs[objID] = struct{}{}
		objIDs = append(objIDs, objID)
		if len(objIDs) >= limit {
			break
		}
	}
	return objIDs
}

func (p *CASLCloudProvider) readGeneralTapeItemRowsForObjectIDs(ctx context.Context, objIDs []string) ([]CASLObjectEvent, error) {
	if len(objIDs) == 0 {
		return nil, nil
	}

	payload, err := p.ReadGeneralTapeItem(ctx, objIDs)
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}

	rows := make([]CASLObjectEvent, 0, len(payload)*2)
	for objID, items := range payload {
		for _, rawRow := range items {
			rowMap := copyStringAnyMap(rawRow)
			if _, exists := rowMap["obj_id"]; !exists || strings.TrimSpace(asString(rowMap["obj_id"])) == "" {
				rowMap["obj_id"] = objID
			}
			if strings.TrimSpace(asString(rowMap["action"])) == "" {
				if dictName := strings.TrimSpace(asString(rowMap["dict_name"])); dictName != "" {
					rowMap["action"] = dictName
				}
			}
			if strings.TrimSpace(asString(rowMap["code"])) == "" {
				if dictName := strings.TrimSpace(asString(rowMap["dict_name"])); dictName != "" {
					rowMap["code"] = dictName
				}
			}

			rowType := strings.TrimSpace(asString(rowMap["type"]))
			if rowType == "" {
				rowType = strings.TrimSpace(asString(rowMap["event_type"]))
			}
			if rowType == "" {
				rowType = "ppk_event"
			}

			row, ok := mapCASLRealtimeRow(rowMap, rowType)
			if !ok {
				continue
			}
			if strings.TrimSpace(row.ObjID) == "" {
				row.ObjID = strings.TrimSpace(objID)
			}
			rows = append(rows, row)
		}
	}

	return rows, nil
}

func (p *CASLCloudProvider) ProcessAlarm(id string, user string, note string) error {
	alarmID, err := strconv.Atoi(id)
	if err != nil || alarmID <= 0 {
		return fmt.Errorf("casl: invalid alarm ID: %s", id)
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	err = p.ProcessAlarmWithRequest(ctx, models.Alarm{
		ID: alarmID,
	}, user, contracts.AlarmProcessingRequest{
		CauseCode: "CAUSES_FALSE_ALARM",
		Note:      note,
	})
	if err != nil {
		log.Debug().Err(err).Int("alarmID", alarmID).Msg("CASL: ProcessAlarm failed")
		return fmt.Errorf("casl: failed to process alarm %d: %w", alarmID, err)
	}
	return nil
}

func (p *CASLCloudProvider) GetAlarmProcessingOptions(ctx context.Context, _ models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	dict, ok := p.cachedDictionarySnapshot(ctx)
	if !ok || len(dict) == 0 {
		loaded, err := p.ReadDictionary(ctx)
		if err != nil {
			return nil, err
		}
		p.mu.Lock()
		p.cachedDictionary = copyStringAnyMap(loaded)
		p.cachedDictionaryAt = time.Now()
		p.mu.Unlock()
		dict = loaded
	}

	codes := extractCASLAlarmCauseCodes(dict)
	dictMap := p.loadDictionaryMap(ctx)
	options := make([]contracts.AlarmProcessingOption, 0, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		label := strings.TrimSpace(dictMap[code])
		if label == "" {
			label = code
		}
		options = append(options, contracts.AlarmProcessingOption{
			Code:  code,
			Label: label,
		})
	}
	if len(options) == 0 {
		options = append(options, contracts.AlarmProcessingOption{
			Code:  "CAUSES_FALSE_ALARM",
			Label: resolveCASLTextFromMap(dictMap, "CAUSES_FALSE_ALARM", "Хибна тривога"),
		})
	}
	return options, nil
}

func (p *CASLCloudProvider) PickAlarm(ctx context.Context, alarm models.Alarm, _ string) error {
	if p == nil {
		return errors.New("casl provider is nil")
	}

	target, err := p.resolveCASLAlarmTarget(ctx, alarm)
	if err != nil {
		return err
	}
	currentUserID := p.currentCASLUserID()
	workflow, workflowErr := p.cachedCASLAlarmWorkflowState(ctx, target)
	target.Alarm = p.applyCASLWorkflowStateToAlarm(target.Alarm, workflow, currentUserID)

	if workflowErr != nil {
		log.Debug().Err(workflowErr).Str("obj_id", target.CASLObjectID).Msg("CASL: не вдалося прочитати workflow state перед pick")
	}
	if isCASLAlarmFinishBlockedAction(workflow.LastAct) {
		return fmt.Errorf("casl alarm pick: alarm %d already has response group workflow in state %s", target.Alarm.ID, workflow.LastAct)
	}
	if target.Alarm.IsInProgress {
		if target.Alarm.IsOwnedByMe {
			return nil
		}
		if err := p.HijackGuardObject(ctx, target.CASLObjectID); err != nil {
			return fmt.Errorf("casl alarm hijack: %w", err)
		}
		p.markCASLAlarmPicked(target.CacheKey, target.Alarm, target.Record, time.Now())
		return nil
	}

	if err := p.PickGuardObject(ctx, target.CASLObjectID, ""); err != nil {
		return fmt.Errorf("casl alarm pick: %w", err)
	}

	p.markCASLAlarmPicked(target.CacheKey, target.Alarm, target.Record, time.Now())
	return nil
}

func (p *CASLCloudProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, _ string, request contracts.AlarmProcessingRequest) error {
	if p == nil {
		return errors.New("casl provider is nil")
	}

	alarmID := alarm.ID
	if alarmID <= 0 {
		return errors.New("alarm ID is required")
	}

	causeCode := strings.TrimSpace(request.CauseCode)
	if causeCode == "" {
		options, err := p.GetAlarmProcessingOptions(ctx, alarm)
		if err != nil {
			return err
		}
		if len(options) > 0 {
			causeCode = strings.TrimSpace(options[0].Code)
		}
	}
	if causeCode == "" {
		causeCode = "CAUSES_FALSE_ALARM"
	}

	target, err := p.resolveCASLAlarmTarget(ctx, alarm)
	if err != nil {
		return err
	}
	currentUserID := p.currentCASLUserID()
	workflow, workflowErr := p.cachedCASLAlarmWorkflowState(ctx, target)
	target.Alarm = p.applyCASLWorkflowStateToAlarm(target.Alarm, workflow, currentUserID)
	if workflowErr != nil {
		log.Debug().Err(workflowErr).Str("obj_id", target.CASLObjectID).Msg("CASL: не вдалося прочитати workflow state перед finish")
	}
	if isCASLAlarmFinishBlockedAction(workflow.LastAct) {
		return fmt.Errorf("casl alarm processing: alarm %d cannot be finished while response group workflow is active (%s)", alarmID, workflow.LastAct)
	}
	if target.Alarm.IsInProgress && !target.Alarm.IsOwnedByMe {
		assignee := strings.TrimSpace(target.Alarm.InProgressBy)
		if assignee == "" {
			assignee = "інший оператор"
		}
		return fmt.Errorf("casl alarm processing: alarm %d is already handled by %s; pick it first", alarmID, assignee)
	}

	if !target.Alarm.IsInProgress || !target.Alarm.IsOwnedByMe {
		if err := p.PickGuardObject(ctx, target.CASLObjectID, ""); err != nil {
			return fmt.Errorf("casl alarm processing pick: %w", err)
		}
	}
	if err := p.FinishGuardObject(ctx, target.CASLObjectID, "", causeCode, strings.TrimSpace(request.Note)); err != nil {
		return fmt.Errorf("casl alarm processing finish: %w", err)
	}

	if target.CacheKey != "" {
		p.mu.Lock()
		delete(p.realtimeAlarmByObjID, target.CacheKey)
		p.mu.Unlock()
	}

	return nil
}

func (p *CASLCloudProvider) cachedCASLAlarmWorkflowState(ctx context.Context, target caslAlarmTarget) (caslAlarmWorkflowState, error) {
	if target.CacheKey == "" {
		return p.readCASLAlarmWorkflowState(ctx, target.CASLObjectID)
	}

	state := caslAlarmWorkflowState{
		UserID:      strings.TrimSpace(target.Alarm.InProgressUser),
		DisplayName: strings.TrimSpace(target.Alarm.InProgressBy),
		MgrID:       strings.TrimSpace(target.Alarm.ResponseGroupID),
	}
	switch {
	case target.Alarm.IsResponseGroupArrived:
		state.LastAct = "GRD_OBJ_MGR_ARRIVE"
	case target.Alarm.IsResponseGroupDispatched:
		state.LastAct = "GRD_OBJ_ASS_MGR"
	case target.Alarm.IsInProgress && !target.Alarm.IsOwnedByMe:
		state.LastAct = "GRD_OBJ_PICK"
	case target.Alarm.IsInProgress:
		state.LastAct = "GRD_OBJ_PICK"
	}
	if state.UserID == "" && state.DisplayName != "" {
		state.UserFIO = state.DisplayName
	}
	if state.LastAct == "" && state.UserID == "" && state.DisplayName == "" && state.MgrID == "" {
		return p.readCASLAlarmWorkflowState(ctx, target.CASLObjectID)
	}
	return state, nil
}

type caslAlarmWorkflowState struct {
	LastAct     string
	UserID      string
	UserFIO     string
	DisplayName string
	MgrID       string
}

func (p *CASLCloudProvider) readCASLAlarmWorkflowState(ctx context.Context, objID string) (caslAlarmWorkflowState, error) {
	objID = strings.TrimSpace(objID)
	if objID == "" {
		return caslAlarmWorkflowState{}, nil
	}

	rows, err := p.ReadGeneralTapeObjects(ctx)
	if err != nil {
		return caslAlarmWorkflowState{}, err
	}

	for _, rawRow := range rows {
		row := normalizeCASLGeneralTapeRow(rawRow)
		if strings.TrimSpace(row.ObjID) != objID {
			continue
		}

		state := caslAlarmWorkflowState{
			LastAct: strings.TrimSpace(firstCASLValue(asString(rawRow["last_act"]), row.Action)),
			UserID:  strings.TrimSpace(firstCASLValue(asString(rawRow["user_id"]), row.UserID)),
			UserFIO: strings.TrimSpace(firstCASLValue(asString(rawRow["userFio"]), asString(rawRow["user_fio"]), row.UserFIO)),
			MgrID:   strings.TrimSpace(firstCASLValue(asString(rawRow["mgr_id"]), row.MgrID)),
		}
		if state.UserFIO != "" {
			state.DisplayName = state.UserFIO
			return state, nil
		}
		if state.UserID != "" {
			if displayName := p.resolveCASLUserDisplayName(ctx, state.UserID); displayName != "" {
				state.DisplayName = displayName
			} else {
				state.DisplayName = "Оператор #" + state.UserID
			}
		}
		return state, nil
	}

	return caslAlarmWorkflowState{}, nil
}

func (p *CASLCloudProvider) applyCASLWorkflowStateToAlarm(alarm models.Alarm, state caslAlarmWorkflowState, currentUserID string) models.Alarm {
	alarm.ResponseGroupID = strings.TrimSpace(state.MgrID)
	alarm.IsResponseGroupDispatched = strings.EqualFold(strings.TrimSpace(state.LastAct), "GRD_OBJ_ASS_MGR") || strings.EqualFold(strings.TrimSpace(state.LastAct), "GRD_OBJ_MGR_ARRIVE")
	alarm.IsResponseGroupArrived = strings.EqualFold(strings.TrimSpace(state.LastAct), "GRD_OBJ_MGR_ARRIVE")
	if !isCASLAlarmInProgressAction(state.LastAct) {
		return alarm
	}

	alarm.IsInProgress = true
	alarm.InProgressUser = strings.TrimSpace(state.UserID)
	alarm.InProgressBy = strings.TrimSpace(state.DisplayName)
	alarm.IsOwnedByMe = alarm.InProgressUser != "" && alarm.InProgressUser == currentUserID
	return alarm
}

type caslAlarmTarget struct {
	Alarm        models.Alarm
	CacheKey     string
	CASLObjectID string
	Record       caslGrdObject
}

func (p *CASLCloudProvider) resolveCASLAlarmTarget(ctx context.Context, alarm models.Alarm) (caslAlarmTarget, error) {
	foundAlarm := alarm
	foundObjectID := alarm.ObjectID
	var foundCacheKey string

	p.mu.RLock()
	for key, cachedAlarm := range p.realtimeAlarmByObjID {
		if cachedAlarm.ID == alarm.ID {
			foundAlarm = cachedAlarm
			foundObjectID = cachedAlarm.ObjectID
			foundCacheKey = key
			break
		}
	}
	record, hasRecord := p.objectByInternalID[foundObjectID]
	p.mu.RUnlock()

	if !hasRecord && foundObjectID > 0 {
		resolved, found, err := p.resolveObjectRecord(ctx, foundObjectID)
		if err != nil {
			return caslAlarmTarget{}, err
		}
		if found {
			record = resolved
			hasRecord = true
		}
	}

	caslObjID := strings.TrimSpace(record.ObjID)
	if !hasRecord || caslObjID == "" {
		return caslAlarmTarget{}, fmt.Errorf("casl alarm processing: object record not found for alarm %d", alarm.ID)
	}

	return caslAlarmTarget{
		Alarm:        foundAlarm,
		CacheKey:     foundCacheKey,
		CASLObjectID: caslObjID,
		Record:       record,
	}, nil
}

func (p *CASLCloudProvider) markCASLAlarmPicked(cacheKey string, alarm models.Alarm, record caslGrdObject, pickedAt time.Time) {
	currentUserID := p.currentCASLUserID()
	inProgressBy := p.resolveCASLUserDisplayName(context.Background(), currentUserID)
	if inProgressBy == "" {
		inProgressBy = "Поточний оператор"
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	update := alarm
	update.IsInProgress = true
	update.InProgressUser = currentUserID
	update.InProgressBy = inProgressBy
	update.IsOwnedByMe = true
	if !pickedAt.IsZero() {
		update.Time = pickedAt
	}

	if cacheKey != "" {
		p.realtimeAlarmByObjID[cacheKey] = update
		return
	}

	objectKey := canonicalCASLRealtimeObjectKey(strings.TrimSpace(record.ObjID), update.ObjectNumber, update.ObjectID)
	seed := stableCASLAlarmSeed(string(update.Type), "", update.ZoneNumber)
	p.realtimeAlarmByObjID[objectKey+"|"+seed] = update
}

func (p *CASLCloudProvider) currentCASLUserID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return strings.TrimSpace(p.userID)
}

func shouldLoadCASLGeneralTapeUsers(rows []map[string]any) bool {
	for _, rawRow := range rows {
		row := normalizeCASLGeneralTapeRow(rawRow)
		if strings.TrimSpace(row.UserID) != "" || strings.TrimSpace(row.UserFIO) != "" {
			return true
		}
	}
	return false
}

func (p *CASLCloudProvider) resolveCASLAlarmAssignee(item caslTapeItem, users map[string]caslUser, currentUserID string) (string, bool) {
	if !isCASLAlarmInProgressAction(item.LastAct) {
		return "", false
	}

	userID := strings.TrimSpace(item.UserID)
	displayName := strings.TrimSpace(item.UserFIO)
	if displayName == "" && userID != "" {
		if user, ok := users[userID]; ok {
			displayName = strings.TrimSpace(user.FullName())
		}
	}
	if displayName == "" && userID != "" {
		displayName = "Оператор #" + userID
	}
	return displayName, userID != "" && userID == strings.TrimSpace(currentUserID)
}

func (p *CASLCloudProvider) resolveCASLUserDisplayName(ctx context.Context, userID string) string {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return ""
	}
	users, err := p.loadUsers(ctx)
	if err != nil {
		return ""
	}
	user, ok := users[userID]
	if !ok {
		return ""
	}
	return strings.TrimSpace(user.FullName())
}

func extractCASLAlarmCauseCodes(dict map[string]any) []string {
	if len(dict) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	codes := make([]string, 0, 16)
	appendCodes := func(raw any) {
		items, ok := raw.([]any)
		if !ok {
			return
		}
		for _, item := range items {
			code := strings.TrimSpace(asString(item))
			if code == "" {
				continue
			}
			if _, exists := seen[code]; exists {
				continue
			}
			seen[code] = struct{}{}
			codes = append(codes, code)
		}
	}

	appendCodes(dict["alarm_causes"])
	if nestedRaw, ok := dict["dictionary"]; ok {
		if nested, ok := nestedRaw.(map[string]any); ok {
			appendCodes(nested["alarm_causes"])
		}
	}

	return codes
}

func resolveCASLTextFromMap(dict map[string]string, key string, fallback string) string {
	value := strings.TrimSpace(dict[key])
	if value != "" {
		return value
	}
	return fallback
}

func (p *CASLCloudProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return "н/д", "н/д", time.Time{}, time.Time{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return "н/д", "н/д", time.Time{}, time.Time{}
	}

	state, stateErr := p.readDeviceState(ctx, record)
	stats, statsErr := p.readStatsAlarms(ctx, record)

	signalParts := []string{"н/д"}
	testParts := make([]string, 0, 4)

	if stateErr == nil {
		if state.LastPingDate.Int64() > 0 {
			lastMsg = time.UnixMilli(state.LastPingDate.Int64()).Local()
			lastTest = lastMsg
		}
	}

	if statsErr == nil {
		testParts = append(testParts,
			fmt.Sprintf("freq=%d", stats.ResponseFrequencies.Int64()),
			fmt.Sprintf("quality=%d", stats.CommunicQuality.Int64()),
			fmt.Sprintf("alarms=%d", stats.CustomWins.Int64()),
			fmt.Sprintf("power=%d", stats.PowerFailure.Int64()),
		)
	}

	if stateErr != nil && statsErr != nil {
		log.Debug().Err(stateErr).Msg("CASL: не вдалося отримати read_device_state")
		log.Debug().Err(statsErr).Msg("CASL: не вдалося отримати stats_alarms")
		testParts = append(testParts, "н/д")
	}

	return strings.Join(signalParts, "; "), strings.Join(testParts, "; "), lastTest, lastMsg
}

func (p *CASLCloudProvider) GetTestMessages(objectID string) []models.TestMessage {
	events := p.GetObjectEvents(objectID)
	if len(events) == 0 {
		return nil
	}

	messages := make([]models.TestMessage, 0, 32)
	for _, event := range events {
		if event.Type != models.EventTest && !strings.Contains(strings.ToUpper(event.Details), "TEST") {
			continue
		}
		messages = append(messages, models.TestMessage{Time: event.Time, Info: event.GetTypeDisplay(), Details: event.Details})
		if len(messages) >= 200 {
			break
		}
	}
	return messages
}

// GetLatestEventID повертає компактний курсор змін для scheduler.
func (p *CASLCloudProvider) GetLatestEventID() (int64, error) {
	p.ensureRealtimeStream()

	p.mu.RLock()
	revision := p.eventsRevision
	p.mu.RUnlock()
	return revision, nil
}

func (p *CASLCloudProvider) readEventsByID(ctx context.Context, record caslGrdObject) ([]caslObjectEvent, error) {
	end := time.Now()
	return p.readEventsByIDRange(ctx, record, end.Add(-caslObjectEventsSpan), end)
}

func (p *CASLCloudProvider) readEventsByIDRange(
	ctx context.Context,
	record caslGrdObject,
	from time.Time,
	to time.Time,
) ([]caslObjectEvent, error) {
	if strings.TrimSpace(record.ObjID) == "" || record.DeviceID.Int64() <= 0 {
		return nil, nil
	}

	end := to.UnixMilli()
	start := from.UnixMilli()

	payload := map[string]any{
		"type":             "read_events_by_id",
		"isFullEventsInfo": false,
		"time_start":       start,
		"time_end":         end,
		"time_request":     end,
		"objIds":           []string{strings.TrimSpace(record.ObjID)},
		"deviceIds":        []string{strconv.FormatInt(record.DeviceID.Int64(), 10)},
		"deviceNumbers":    []int64{record.DeviceNumber.Int64()},
	}

	var resp caslReadEventsByIDResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	rows := resp.Data
	if len(rows) == 0 {
		rows = resp.Events
	}
	if err := validateCASLObjectEvents(rows, "casl read_events_by_id"); err != nil {
		return nil, err
	}

	return append([]caslObjectEvent(nil), rows...), nil
}

func (p *CASLCloudProvider) readDeviceState(ctx context.Context, record caslGrdObject) (caslDeviceState, error) {
	deviceID := record.DeviceID.Int64()
	if deviceID <= 0 {
		return caslDeviceState{}, errors.New("casl: empty device_id")
	}

	p.mu.RLock()
	if p.deviceStateCache != nil {
		if state, ok := p.deviceStateCache[deviceID]; ok && time.Since(p.deviceStateCacheAt[deviceID]) < 3*time.Second {
			p.mu.RUnlock()
			return state, nil
		}
	}
	p.mu.RUnlock()

	deviceIDStr := strconv.FormatInt(deviceID, 10)
	val, err, _ := p.deviceStateSF.Do(deviceIDStr, func() (any, error) {
		payload := map[string]any{"type": "read_device_state", "device_id": deviceIDStr}

		var resp caslReadDeviceStateResponse
		if err := p.postCommand(ctx, payload, &resp, true); err != nil {
			return caslDeviceState{}, err
		}
		if err := validateCASLDeviceState(resp.State, "casl read_device_state"); err != nil {
			return caslDeviceState{}, err
		}

		p.mu.Lock()
		if p.deviceStateCache == nil {
			p.deviceStateCache = make(map[int64]caslDeviceState)
			p.deviceStateCacheAt = make(map[int64]time.Time)
		}
		p.deviceStateCache[deviceID] = resp.State
		p.deviceStateCacheAt[deviceID] = time.Now()
		p.mu.Unlock()

		return resp.State, nil
	})

	if err != nil {
		return caslDeviceState{}, err
	}
	return val.(caslDeviceState), nil
}

func (p *CASLCloudProvider) readStatsAlarms(ctx context.Context, record caslGrdObject) (caslStatsAlarmsData, error) {
	deviceID := record.DeviceID.Int64()
	if deviceID <= 0 || strings.TrimSpace(record.ObjID) == "" {
		return caslStatsAlarmsData{}, errors.New("casl: empty object/device identifiers")
	}

	end := time.Now().UnixMilli()
	start := end - caslStatsSpan.Milliseconds()

	payload := map[string]any{
		"type":      "get_statistic",
		"name":      "stats_alarms",
		"deviceId":  strconv.FormatInt(deviceID, 10),
		"objectId":  strings.TrimSpace(record.ObjID),
		"startTime": start,
		"endTime":   end,
	}

	var resp caslGetStatisticResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return caslStatsAlarmsData{}, err
	}
	if err := validateCASLStatsAlarmsData(resp.Data, "casl get_statistic"); err != nil {
		return caslStatsAlarmsData{}, err
	}
	return resp.Data, nil
}

func (p *CASLCloudProvider) mapCASLObjectEvents(ctx context.Context, record caslGrdObject, raw []caslObjectEvent) []models.Event {
	if len(raw) == 0 {
		return nil
	}

	result := make([]models.Event, 0, len(raw))
	objectID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
	objectNum := preferredCASLObjectNumber(record.ObjID, record.Name, record.DeviceNumber.Int64())
	objectName := strings.TrimSpace(record.Name)
	if objectName == "" {
		objectName = "Об'єкт #" + objectNum
	}
	objectName = formatCASLJournalObjectName(objectNum, objectName)

	var (
		translator map[string]string
		dictMap    map[string]string
		lineInfos  map[int]caslEventLineInfo
		deviceType string
	)

	device, hasDevice := p.resolveDeviceForObject(record)
	if hasDevice {
		deviceType = strings.TrimSpace(device.Type.String())
		translator = p.loadTranslatorMap(ctx, deviceType)
		lineInfos = p.buildCASLLineInfoIndex(ctx, device.Lines)
	}
	dictMap = p.loadDictionaryMap(ctx)
	users := map[string]caslUser(nil)
	normalizedRows := make([]CASLObjectEvent, 0, len(raw))
	for _, item := range raw {
		normalizedRows = append(normalizedRows, normalizeCASLObjectEvent(item))
	}
	if shouldLoadCASLEventUsers(normalizedRows) {
		if loadedUsers, err := p.loadUsers(ctx); err == nil {
			users = loadedUsers
		} else {
			log.Debug().Err(err).Msg("CASL: не вдалося завантажити користувачів для подій об'єкта")
		}
	}

	for idx, row := range normalizedRows {
		ts := row.Time
		if ts <= 0 {
			continue
		}
		eventTime := time.UnixMilli(ts).Local()

		code := row.Code
		if code == "" {
			code = "UNKNOWN"
		}

		zoneNumber := int(row.Number)
		contactID := strings.TrimSpace(row.ContactID)
		sourceType := effectiveCASLSourceType(row)

		details := buildCASLUserActionDetails(row, dictMap, users)
		if details == "" && isCASLPPKMessageSource(sourceType) {
			details = buildCASLPPKEventDetails(row, translator, dictMap, deviceType, lineInfos, users)
		}
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, zoneNumber, deviceType)
		}
		if isCASLUnknownText(details) || (isCASLActionSource(sourceType) && isCASLUnknownText(code) && isCASLUnknownText(contactID)) {
			details = fallbackCASLActionDetails(row, sourceType)
		}
		if details == "" {
			switch {
			case contactID != "" && code != "":
				details = fmt.Sprintf("%s (%s)", contactID, code)
			case contactID != "":
				details = contactID
			default:
				details = code
			}
		}
		if isCASLUnknownText(details) {
			if fallback := fallbackCASLActionDetails(row, sourceType); fallback != "" {
				details = fallback
			}
		}
		classifierCode := resolveCASLEventClassificationKey(translator, code, contactID, deviceType, zoneNumber, lineInfos)
		if classifierCode == "" {
			classifierCode = code
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, contactID, sourceType, details)
		if sourceType != "" &&
			!strings.EqualFold(sourceType, "ppk_event") &&
			!isCASLActionSource(sourceType) {
			details += " | src=" + sourceType
		}

		result = append(result, models.Event{
			ID:           stableCASLEventID(record.ObjID, ts, code, idx),
			Time:         eventTime,
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Type:         eventType,
			TypeLabel:    caslActionTypeLabel(sourceType),
			ZoneNumber:   zoneNumber,
			Details:      details,
			SC1:          mapCASLEventSC1(eventType),
		})
	}

	return result
}

// ListResponseGroups implements contracts.ResponseGroupProvider.
func (p *CASLCloudProvider) ListResponseGroups(ctx context.Context) ([]contracts.ResponseGroup, error) {
	rows, err := p.ReadManagers(ctx, 0, 500)
	if err != nil {
		return nil, fmt.Errorf("casl: read_mgr: %w", err)
	}
	activeByGroup := make(map[string]models.Alarm)
	for _, alarm := range p.snapshotRealtimeAlarms() {
		groupID := strings.TrimSpace(alarm.ResponseGroupID)
		if groupID == "" || !alarm.IsResponseGroupDispatched {
			continue
		}
		activeByGroup[groupID] = alarm
	}
	result := make([]contracts.ResponseGroup, 0, len(rows))
	for _, row := range rows {
		id := strings.TrimSpace(asString(row["mgr_id"]))
		name := strings.TrimSpace(asString(row["name"]))
		if id == "" {
			continue
		}
		status := caslResponseGroupStatus(asString(row["last_act"]))
		statusText := responseGroupStatusText(status, "")
		objectNumber := ""
		objectName := ""
		if alarm, ok := activeByGroup[id]; ok {
			objectNumber = alarm.GetObjectNumberDisplay()
			objectName = strings.TrimSpace(alarm.ObjectName)
			if alarm.IsResponseGroupArrived {
				status = contracts.ResponseGroupStatusArrived
			} else {
				status = contracts.ResponseGroupStatusDispatched
			}
			statusText = responseGroupStatusText(status, "")
		}
		result = append(result, contracts.ResponseGroup{
			ID:              id,
			Name:            name,
			Phone:           strings.TrimSpace(asString(row["phone_number"])),
			Source:          contracts.FrontendSourceCASL,
			Status:          status,
			StatusText:      statusText,
			ObjectNumber:    objectNumber,
			ObjectName:      objectName,
			StatusChangedAt: parseCASLAnyTime(row["last_act_time"]),
		})
	}
	return result, nil
}

func caslResponseGroupStatus(lastAction string) contracts.ResponseGroupStatus {
	switch strings.ToUpper(strings.TrimSpace(lastAction)) {
	case "GRD_OBJ_ASS_MGR":
		return contracts.ResponseGroupStatusDispatched
	case "GRD_OBJ_MGR_ARRIVE":
		return contracts.ResponseGroupStatusArrived
	case "GRD_OBJ_MGR_CANCEL", "":
		return contracts.ResponseGroupStatusFree
	default:
		return contracts.ResponseGroupStatusUnknown
	}
}

// AssignResponseGroup implements contracts.ResponseGroupProvider.
func (p *CASLCloudProvider) AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error {
	target, err := p.resolveCASLAlarmTarget(ctx, alarm)
	if err != nil {
		return err
	}
	return p.AssignMGRToAlarm(ctx, target.CASLObjectID, groupID)
}

// NotifyGroupArrived implements contracts.ResponseGroupProvider.
func (p *CASLCloudProvider) NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error {
	target, err := p.resolveCASLAlarmTarget(ctx, alarm)
	if err != nil {
		return err
	}
	return p.NotifyMGRArrived(ctx, target.CASLObjectID)
}

// CancelResponseGroup implements contracts.ResponseGroupProvider.
func (p *CASLCloudProvider) CancelResponseGroup(ctx context.Context, alarm models.Alarm) error {
	target, err := p.resolveCASLAlarmTarget(ctx, alarm)
	if err != nil {
		return err
	}
	return p.CancelMGRFromAlarm(ctx, target.CASLObjectID)
}
