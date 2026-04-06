package data

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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

func isCASLUnknownText(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "unknown", "undefined", "unset", "not set", "не встановлено", "невідомо", "none", "null":
		return true
	default:
		return false
	}
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

func (p *CASLCloudProvider) GetEvents() []models.Event {
	p.ensureRealtimeStream()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()
	if events, err := p.readEventsJournalAsEvents(ctx); err == nil {
		return events
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *CASLCloudProvider) getEventsFromBasketFallback(ctx context.Context) []models.Event {
	basketCount, err := p.readBasketCount(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося отримати count з кошика тривог")
		p.mu.RLock()
		events := append([]models.Event(nil), p.cachedEvents...)
		p.mu.RUnlock()
		return events
	}

	objectID, objectName := p.primaryObjectContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.hasBasketCount {
		p.lastBasketCount = basketCount
		p.hasBasketCount = true
		return append([]models.Event(nil), p.cachedEvents...)
	}

	if basketCount != p.lastBasketCount {
		eventType := models.EventFault
		eventSC1 := 2
		if basketCount == 0 {
			eventType = models.EventRestore
			eventSC1 = 5
		}

		event := models.Event{
			ID:           nextCASLEventID(),
			Time:         time.Now(),
			ObjectID:     objectID,
			ObjectNumber: strconv.Itoa(objectID),
			ObjectName:   objectName,
			Type:         eventType,
			Details:      fmt.Sprintf("CASL Cloud: активних тривог у кошику %d (було %d)", basketCount, p.lastBasketCount),
			SC1:          eventSC1,
		}

		p.cachedEvents = append([]models.Event{event}, p.cachedEvents...)
		if len(p.cachedEvents) > caslMaxCachedEvents {
			p.cachedEvents = p.cachedEvents[:caslMaxCachedEvents]
		}
		p.lastBasketCount = basketCount
	}

	return append([]models.Event(nil), p.cachedEvents...)
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

	rows, err := p.ReadEventsJournal(ctx, CASLReadEventsRequest{
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
		if row.PPKNum <= 0 && rawObjID == "" {
			continue
		}
		if startGate > 0 && row.Time > 0 && row.Time < startGate {
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

	events := make([]models.Event, 0, len(filteredRows))
	for _, row := range filteredRows {
		ppkNum := row.PPKNum
		number := int(row.Number)
		code := strings.TrimSpace(row.Code)
		contactID := strings.TrimSpace(row.ContactID)
		sourceType := strings.TrimSpace(row.Type)
		rawObjID := strings.TrimSpace(row.ObjID)

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := mapCASLObjectID(strconv.FormatInt(ppkNum, 10))
		objectName := strings.TrimSpace(row.ObjName)
		if objectName == "" {
			if ppkNum > 0 {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			} else if rawObjID != "" {
				objectName = "Об'єкт #" + rawObjID
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
		lineNames := map[int]string(nil)
		deviceType := ""
		if hasCtx {
			translator = ctxItem.Translator
			lineNames = ctxItem.LineNames
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := buildCASLUserActionDetails(row)
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, number, deviceType)
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
		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[number]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}

		classifierCode := code
		if classifierCode == "" {
			classifierCode = strings.TrimSpace(row.Action)
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, contactID, sourceType, details)
		eventTime := time.Now()
		if row.Time > 0 {
			eventTime = time.UnixMilli(row.Time).Local()
		}
		eventTS := row.Time
		if eventTS <= 0 {
			eventTS = eventTime.UnixMilli()
		}

		seed := stableCASLAlarmSeed(code, contactID, number)
		// objectNum := objectNums[rawObjID]
		if objectNum == "" {
			objectNum = rawObjID
		}

		events = append(events, models.Event{
			ID:           stableCASLEventID(strconv.FormatInt(ppkNum, 10), eventTS, seed, 0),
			Time:         eventTime,
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Type:         eventType,
			ZoneNumber:   number,
			Details:      details,
			SC1:          mapCASLEventSC1(eventType),
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
			ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
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
		ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)

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

func (p *CASLCloudProvider) readGeneralTapeAsEvents(ctx context.Context, byObject map[string]caslEventContext) ([]models.Event, error) {
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

	events := make([]models.Event, 0, len(rows))
	for idx, row := range rows {
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		deviceID := strings.TrimSpace(asString(row["device_id"]))
		if deviceID == "" {
			deviceID = strings.TrimSpace(asString(row["deviceId"]))
		}
		if ppkNum <= 0 {
			ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, deviceID, resolvedByDeviceID, unresolvedByDeviceID)
		}

		// Використання дефолтних значень
		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10), asString(row["number"]), asString(row["device_number"]), deviceID)
		objectName := strings.TrimSpace(asString(row["obj_name"]))
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name"]))
		}
		if objectName == "" {
			objectName = "Об'єкт #" + strings.TrimSpace(rawObjID)
		}
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)
		translator := map[string]string(nil)
		deviceType := strings.TrimSpace(asString(row["device_type"]))

		// Збагачення даними з попередньо побудованого контексту
		if ctxItem, hasCtx := byObject[rawObjID]; hasCtx {
			objectID = ctxItem.ObjectID
			objectNum = ctxItem.ObjectNum
			objectName = ctxItem.ObjectName
			translator = ctxItem.Translator
			if ctxItem.DeviceType != "" {
				deviceType = ctxItem.DeviceType
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		eventTime := parseCASLAnyTime(row["time"])
		if eventTime.IsZero() {
			eventTime = time.Now()
		}

		sourceType := strings.TrimSpace(asString(row["event_type"]))
		code := strings.TrimSpace(asString(row["code"]))
		contactID := strings.TrimSpace(asString(row["contact_id"]))
		details := strings.TrimSpace(asString(row["description"]))
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, parseCASLAnyInt(row["zone"]), deviceType)
		}
		if details == "" {
			reason := strings.TrimSpace(asString(row["reasonAlarm"]))
			if reason != "" {
				if translated := decodeCASLEventDescription(nil, dictMap, reason, "", parseCASLAnyInt(row["zone"]), deviceType); translated != "" {
					details = translated
				} else {
					details = "Причина: " + reason
				}
			}
		}
		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)

		eventID := parseCASLAnyInt(row["event_id"])
		if eventID <= 0 {
			eventID = stableCASLEventID(rawObjID, eventTime.UnixMilli(), sourceType+"|"+code+"|"+contactID, idx)
		}

		events = append(events, models.Event{
			ID:           eventID,
			Time:         eventTime,
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Type:         eventType,
			ZoneNumber:   parseCASLAnyInt(row["zone"]),
			Details:      details,
			UserName:     strings.TrimSpace(asString(row["user_id"])),
			SC1:          mapCASLEventSC1(eventType),
		})
	}

	sortEvents(events)
	if len(events) > caslMaxCachedEvents {
		events = events[:caslMaxCachedEvents]
	}
	return events, nil
}

func (p *CASLCloudProvider) readFromBasketAsAlarms(ctx context.Context, byPPK map[int64]caslEventContext, byObject map[string]caslEventContext) ([]models.Alarm, error) {
	rows, err := p.ReadFromBasket(ctx, 0, 10_000)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	dictMap := p.loadDictionaryMap(ctx)
	resolvedByDeviceID := make(map[string]int64)
	unresolvedByDeviceID := make(map[string]struct{})

	alarms := make([]models.Alarm, 0, len(rows))
	for _, row := range rows {
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		deviceID := strings.TrimSpace(asString(row["device_id"]))
		if deviceID == "" {
			deviceID = strings.TrimSpace(asString(row["deviceId"]))
		}
		if ppkNum <= 0 {
			ppkNum = p.resolveCASLPPKByDeviceIDWithCache(ctx, deviceID, resolvedByDeviceID, unresolvedByDeviceID)
		}
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		if ppkNum <= 0 && rawObjID == "" && deviceID == "" {
			continue
		}
		number := parseCASLAnyInt(row["number"])
		if number <= 0 {
			number = parseCASLAnyInt(row["zone"])
		}
		code := strings.TrimSpace(asString(row["code"]))
		contactID := strings.TrimSpace(asString(row["contact_id"]))
		sourceType := strings.TrimSpace(asString(row["event_type"]))
		if sourceType == "" {
			sourceType = strings.TrimSpace(asString(row["type"]))
		}
		description := strings.TrimSpace(asString(row["description"]))

		ctxItem, hasCtx := byPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := byObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10), deviceID)
		objectName := strings.TrimSpace(asString(row["obj_name"]))
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name_obj"]))
		}
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name"]))
		}
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)
		if hasCtx {
			objectID = ctxItem.ObjectID
			if strings.TrimSpace(ctxItem.ObjectNum) != "" {
				objectNum = strings.TrimSpace(ctxItem.ObjectNum)
			}
			if objectName == "" {
				objectName = ctxItem.ObjectName
			}
		}
		if objectName == "" {
			if objectNum != "" {
				objectName = "Об'єкт #" + objectNum
			} else {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		translator := map[string]string(nil)
		lineNames := map[int]string(nil)
		deviceType := ""
		if hasCtx {
			translator = ctxItem.Translator
			lineNames = ctxItem.LineNames
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := description
		if details == "" {
			details = decodeCASLEventDescription(translator, dictMap, code, contactID, number, deviceType)
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
				details = "CASL тривога"
			}
		}
		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[number]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}

		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)
		alarmType, include := mapEventTypeToAlarmType(eventType)
		if !include {
			continue
		}

		tsValue := parseCASLAnyTime(row["time"])
		if tsValue.IsZero() {
			tsValue = parseCASLAnyTime(row["create_date"])
		}
		if tsValue.IsZero() {
			tsValue = time.Now()
		}

		seed := stableCASLAlarmSeed(code, contactID, number)
		objectKey := canonicalCASLRealtimeObjectKey(rawObjID, objectNum, objectID)
		if objectKey == "" {
			objectKey = "casl"
		}

		alarms = append(alarms, models.Alarm{
			ID:           stableCASLAlarmID(objectKey, tsValue.UnixMilli(), seed),
			ObjectID:     objectID,
			ObjectNumber: objectNum,
			ObjectName:   objectName,
			Address:      strings.TrimSpace(asString(row["address"])),
			Time:         tsValue.Local(),
			Details:      details,
			Type:         alarmType,
			ZoneNumber:   number,
			SC1:          mapCASLEventSC1(eventType),
		})
	}

	sortCASLAlarms(alarms)

	return alarms, nil
}

func (p *CASLCloudProvider) GetObjectEvents(objectID string) []models.Event {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	now := time.Now()

	p.mu.RLock()
	if ts, ok := p.cachedObjectEventsAt[internalID]; ok && now.Sub(ts) <= caslObjectEventsTTL {
		events := append([]models.Event(nil), p.cachedObjectEvents[internalID]...)
		p.mu.RUnlock()
		return events
	}
	p.mu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	rawEvents, err := p.readEventsByID(ctx, record)
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
	sortEvents(events)

	p.mu.Lock()
	p.cachedObjectEvents[internalID] = append([]models.Event(nil), events...)
	p.cachedObjectEventsAt[internalID] = now
	p.mu.Unlock()

	return events
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

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	p.ensureRealtimeStream()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	// 1. БУДУЄМО СПІЛЬНИЙ КОНТЕКСТ ОДИН РАЗ
	byPPK, byObject, ctxErr := p.buildSharedObjectContext(ctx)
	if ctxErr != nil {
		log.Debug().Err(ctxErr).Msg("CASL: не вдалося побудувати спільний контекст об'єктів")
	}

	if _, err := p.readEventsJournalAsEvents(ctx); err != nil {
		log.Debug().Err(err).Msg("CASL: read_events недоступний під час формування активних тривог")
	}

	// 2. ПЕРЕДАЄМО КОНТЕКСТ В БАСКЕТ
	basketAlarms, err := p.readFromBasketAsAlarms(ctx, byPPK, byObject)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: read_from_basket недоступний під час формування активних тривог")
	} else {
		p.syncRealtimeAlarmsFromBasket(basketAlarms)
	}

	alarms := p.snapshotRealtimeAlarms()
	if len(alarms) == 0 {
		// 3. ПЕРЕДАЄМО КОНТЕКСТ В ТЕЙП
		tapeEvents, err := p.readGeneralTapeAsEvents(ctx, byObject)
		if err != nil {
			log.Debug().Err(err).Msg("CASL: get_general_tape_objects недоступний під час формування активних тривог")
		} else if len(tapeEvents) > 0 {
			p.updateRealtimeAlarmsFromEvents(ctx, tapeEvents)
			alarms = p.snapshotRealtimeAlarms()
		}
	}
	if len(alarms) == 0 {
		// get_general_tape_item повертає історичний ланцюжок подій по об'єкту
		// ("від тривоги далі"), а не перелік поточних активних тривог.
		// Тому не використовуємо його як fallback для стрічки активних тривог,
		// інакше в UI зависають уже завершені CASL-кейси.
	}
	if len(alarms) == 0 {
		return nil
	}
	sortCASLAlarms(alarms)
	return alarms
}
func (p *CASLCloudProvider) syncRealtimeAlarmsFromBasket(basket []models.Alarm) {
	p.mu.Lock()
	defer p.mu.Unlock()

	activeIDs := make(map[int]struct{}, len(basket))
	for _, alarm := range basket {
		activeIDs[alarm.ID] = struct{}{}

		objectKey := canonicalCASLRealtimeObjectKey("", alarm.ObjectNumber, alarm.ObjectID)
		cacheKey := canonicalCASLRealtimeAlarmKey(objectKey, alarm.ZoneNumber)
		if existing, exists := p.realtimeAlarmByObjID[cacheKey]; !exists || alarm.Time.After(existing.Time) {
			p.realtimeAlarmByObjID[cacheKey] = alarm
		}
	}

	for key, cached := range p.realtimeAlarmByObjID {
		if _, active := activeIDs[cached.ID]; !active {
			delete(p.realtimeAlarmByObjID, key)
		}
	}
}

func (p *CASLCloudProvider) readGeneralTapeItemRows(ctx context.Context) ([]CASLObjectEvent, error) {
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, err
	}

	objIDs := make([]string, 0, len(records))
	for _, record := range records {
		objID := strings.TrimSpace(record.ObjID)
		if objID == "" {
			continue
		}
		objIDs = append(objIDs, objID)
	}
	return p.readGeneralTapeItemRowsForObjectIDs(ctx, objIDs)
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

func logCASLGeneralTapeItemRows(rows []CASLObjectEvent) {
	log.Debug().
		Int("rows", len(rows)).
		Msg("CASL get_general_tape_item: отримано події")

	const maxRowsToLog = 200
	for idx, row := range rows {
		if idx >= maxRowsToLog {
			log.Debug().
				Int("logged_rows", maxRowsToLog).
				Int("total_rows", len(rows)).
				Msg("CASL get_general_tape_item: лог скорочено")
			return
		}

		log.Debug().
			Int("idx", idx).
			Str("obj_id", strings.TrimSpace(row.ObjID)).
			Int64("ppk_num", row.PPKNum).
			Str("code", strings.TrimSpace(row.Code)).
			Str("contact_id", strings.TrimSpace(row.ContactID)).
			Int64("number", row.Number).
			Str("type", strings.TrimSpace(row.Type)).
			Int64("time", row.Time).
			Msg("CASL get_general_tape_item row")
	}
}

func (p *CASLCloudProvider) ProcessAlarm(id string, user string, note string) {
	alarmID, _ := strconv.Atoi(id)
	if alarmID <= 0 {
		return
	}

	var foundObjectID int
	var foundCacheKey string

	p.mu.Lock()
	for key, alarm := range p.realtimeAlarmByObjID {
		if alarm.ID == alarmID {
			foundObjectID = alarm.ObjectID
			foundCacheKey = key
			break
		}
	}

	if foundCacheKey != "" {
		delete(p.realtimeAlarmByObjID, foundCacheKey)
	}

	record, hasRecord := p.objectByInternalID[foundObjectID]
	p.mu.Unlock()

	if hasRecord && strings.TrimSpace(record.ObjID) != "" {
		ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
		defer cancel()

		caslObjID := strings.TrimSpace(record.ObjID)
		// Для grd_obj_pick/finish у CASL API достатньо obj_id, якщо ми не маємо event_id.
		if err := p.PickGuardObject(ctx, caslObjID, ""); err != nil {
			log.Debug().Err(err).Str("objID", caslObjID).Msg("CASL: PickGuardObject failed")
		}
		if err := p.FinishGuardObject(ctx, caslObjID, "", "CAUSES_FALSE_ALARM", note); err != nil {
			log.Debug().Err(err).Str("objID", caslObjID).Msg("CASL: FinishGuardObject failed")
		}
	} else {
		log.Debug().Int("alarmID", alarmID).Int("objectID", foundObjectID).Msg("CASL: record not found for ProcessAlarm or not a CASL alarm")
	}
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
	if strings.TrimSpace(record.ObjID) == "" || record.DeviceID.Int64() <= 0 {
		return nil, nil
	}

	end := time.Now().UnixMilli()
	start := end - caslObjectEventsSpan.Milliseconds()

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

	if len(resp.Data) > 0 {
		return append([]caslObjectEvent(nil), resp.Data...), nil
	}
	return append([]caslObjectEvent(nil), resp.Events...), nil
}

func (p *CASLCloudProvider) readDeviceState(ctx context.Context, record caslGrdObject) (caslDeviceState, error) {
	deviceID := record.DeviceID.Int64()
	if deviceID <= 0 {
		return caslDeviceState{}, errors.New("casl: empty device_id")
	}

	payload := map[string]any{"type": "read_device_state", "device_id": strconv.FormatInt(deviceID, 10)}

	var resp caslReadDeviceStateResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return caslDeviceState{}, err
	}
	return resp.State, nil
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
		lineNames  map[int]string
		deviceType string
	)

	device, hasDevice := p.resolveDeviceForObject(record)
	if hasDevice {
		deviceType = strings.TrimSpace(device.Type.String())
		translator = p.loadTranslatorMap(ctx, deviceType)
		lineNames = buildCASLLineNameIndex(device.Lines)
	}
	dictMap = p.loadDictionaryMap(ctx)

	for idx, item := range raw {
		row := normalizeCASLObjectEvent(item)
		ts := item.Time.Int64()
		eventTime := time.Now()
		if ts > 0 {
			eventTime = time.UnixMilli(ts).Local()
		}

		code := row.Code
		if code == "" {
			code = "UNKNOWN"
		}

		zoneNumber := int(row.Number)
		contactID := strings.TrimSpace(row.ContactID)
		sourceType := strings.TrimSpace(row.Type)
		if strings.EqualFold(sourceType, "user_action") && strings.TrimSpace(row.UserActionType) != "" {
			sourceType = strings.TrimSpace(row.UserActionType)
		}

		details := buildCASLUserActionDetails(row)
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
		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)

		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[zoneNumber]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}
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
			ZoneNumber:   zoneNumber,
			Details:      details,
			SC1:          mapCASLEventSC1(eventType),
		})
	}

	return result
}
