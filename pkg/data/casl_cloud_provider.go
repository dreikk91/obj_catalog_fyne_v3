package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"
)

const (
	caslCommandPath    = "/command"
	caslLoginPath      = "/login"
	caslDefaultBaseURL = "http://127.0.0.1:50003"

	caslHTTPTimeout       = 12 * time.Second
	caslObjectsCacheTTL   = 20 * time.Second
	caslUsersCacheTTL     = 5 * time.Minute
	caslObjectEventsTTL   = 10 * time.Second
	caslObjectEventsSpan  = 7 * 24 * time.Hour
	caslJournalEventsSpan = 72 * time.Hour
	caslStatsSpan         = 30 * 24 * time.Hour
	caslDictionaryTTL     = 15 * time.Minute
	caslTranslatorTTL     = 15 * time.Minute
	caslProbeEventsSpan   = 2 * time.Minute
	caslRealtimeBackoff   = 10 * time.Second

	caslMaxCachedEvents = 2000
	caslReadLimit       = 100000
	caslDebugBodyLimit  = 8192

	caslObjectStatusText = "НОРМА"

	caslObjectIDNamespaceStart = 1_500_000_000
	caslObjectIDNamespaceEnd   = 1_999_999_999
	caslObjectIDNamespaceSize  = caslObjectIDNamespaceEnd - caslObjectIDNamespaceStart + 1
)

// CASLCloudProvider реалізує DataProvider для CASL Cloud API.
// Підтримує:
//   - login + автоматичне оновлення token;
//   - список об'єктів (read_grd_object);
//   - базові деталі об'єкта (кімнати, відповідальні, події, стан обладнання, статистика).
type CASLCloudProvider struct {
	baseURL string
	pultID  int64
	email   string
	pass    string

	httpClient *http.Client

	authMu sync.Mutex
	mu     sync.RWMutex

	token  string
	wsURL  string
	userID string

	cachedObjects      []caslGrdObject
	cachedObjectsAt    time.Time
	objectByInternalID map[int]caslGrdObject
	deviceByDeviceID   map[string]caslDevice
	deviceByObjectID   map[string]caslDevice
	deviceByNumber     map[int64]caslDevice
	cachedDevicesAt    time.Time

	cachedUsers   map[string]caslUser
	cachedUsersAt time.Time

	cachedObjectEvents   map[int][]models.Event
	cachedObjectEventsAt map[int]time.Time

	cachedEvents    []models.Event
	lastBasketCount int
	hasBasketCount  bool
	eventsStartAtMs int64
	eventsCursorMs  int64
	eventsRevision  int64

	cachedDictionary        map[string]any
	cachedDictionaryAt      time.Time
	cachedTranslators       map[string]map[string]string
	cachedTransAt           map[string]time.Time
	translatorDisabledUntil time.Time

	realtimeMu         sync.Mutex
	realtimeCancel     context.CancelFunc
	realtimeRunning    bool
	realtimeSubscribed bool

	realtimeAlarmByObjID map[string]models.Alarm
}

func NewCASLCloudProvider(baseURL string, token string, pultID int64, credentials ...string) *CASLCloudProvider {
	nowMS := time.Now().UnixMilli()
	email := ""
	pass := ""
	if len(credentials) > 0 {
		email = strings.TrimSpace(credentials[0])
	}
	if len(credentials) > 1 {
		pass = strings.TrimSpace(credentials[1])
	}

	return &CASLCloudProvider{
		baseURL: normalizeCASLBaseURL(baseURL),
		pultID:  pultID,
		email:   email,
		pass:    pass,
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: caslHTTPTimeout,
		},
		objectByInternalID:   make(map[int]caslGrdObject),
		deviceByDeviceID:     make(map[string]caslDevice),
		deviceByObjectID:     make(map[string]caslDevice),
		deviceByNumber:       make(map[int64]caslDevice),
		cachedUsers:          make(map[string]caslUser),
		cachedObjectEvents:   make(map[int][]models.Event),
		cachedObjectEventsAt: make(map[int]time.Time),
		cachedTranslators:    make(map[string]map[string]string),
		cachedTransAt:        make(map[string]time.Time),
		eventsStartAtMs:      nowMS,
		eventsCursorMs:       nowMS,
		realtimeAlarmByObjID: make(map[string]models.Alarm),
	}
}

func (p *CASLCloudProvider) GetObjects() []models.Object {
	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	records, err := p.loadObjects(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("CASL: read_grd_object недоступний, fallback на read_pult")
		pults, pErr := p.readPultsPublic(ctx)
		if pErr != nil {
			log.Error().Err(pErr).Msg("CASL: не вдалося завантажити об'єкти")
			return nil
		}
		objects := make([]models.Object, 0, len(pults))
		for _, item := range pults {
			objects = append(objects, mapCASLPultToObject(item))
		}
		return objects
	}

	if _, devicesErr := p.loadDevices(ctx); devicesErr != nil {
		log.Debug().Err(devicesErr).Msg("CASL: не вдалося завантажити read_device (продовжую без enrich)")
	}

	objects := make([]models.Object, 0, len(records))
	for _, record := range records {
		device, hasDevice := p.resolveDeviceForObject(record)
		obj := mapCASLGrdObjectToObject(record, selectCASLDevice(hasDevice, device))
		p.enrichCASLObjectWithDeviceMeta(ctx, &obj, hasDevice, device)
		objects = append(objects, obj)
	}
	return objects
}

func (p *CASLCloudProvider) GetObjectByID(idStr string) *models.Object {
	objectID, ok := parseObjectID(idStr)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, objectID)
	if err != nil || !found {
		return nil
	}

	if _, devicesErr := p.loadDevices(ctx); devicesErr != nil {
		log.Debug().Err(devicesErr).Msg("CASL: не вдалося завантажити read_device (GetObjectByID)")
	}

	device, hasDevice := p.resolveDeviceForObject(record)
	obj := mapCASLGrdObjectToObject(record, selectCASLDevice(hasDevice, device))
	p.enrichCASLObjectWithDeviceMeta(ctx, &obj, hasDevice, device)
	if state, stateErr := p.readDeviceState(ctx, record); stateErr == nil {
		obj.PowerFault = normalizeCASLAlarmState(state.Power.Int64())
		obj.AkbState = normalizeCASLAlarmState(state.Accum.Int64())
		obj.PowerSource = models.PowerMains
		if obj.PowerFault > 0 {
			obj.PowerSource = models.PowerBattery
		}
		if state.Online.Int64() > 0 {
			obj.IsConnState = 1
			obj.IsConnOK = true
		}
		if state.LastPingDate.Int64() > 0 {
			msgTime := time.UnixMilli(state.LastPingDate.Int64()).Local()
			obj.LastMessageTime = msgTime
			obj.LastTestTime = msgTime
		}
		obj.Groups = mapCASLDeviceGroupsToObjectGroups(state.Groups, record.Rooms)
	}

	return &obj
}

func (p *CASLCloudProvider) GetZones(objectID string) []models.Zone {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	if _, devicesErr := p.loadDevices(ctx); devicesErr == nil {
		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice && len(device.Lines) > 0 {
			zones := make([]models.Zone, 0, len(device.Lines))
			for idx, line := range device.Lines {
				number := int(line.Number.Int64())
				if number <= 0 {
					number = int(line.ID.Int64())
				}
				if number <= 0 {
					number = idx + 1
				}

				name := strings.TrimSpace(line.Name.String())
				if name == "" {
					name = fmt.Sprintf("Зона %d", number)
				}

				sensorType := strings.TrimSpace(line.Type.String())
				if sensorType == "" {
					sensorType = "Шлейф"
				}

				zones = append(zones, models.Zone{
					Number:     number,
					Name:       name,
					SensorType: sensorType,
					Status:     models.ZoneNormal,
				})
			}
			sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
			return zones
		}
	}

	if len(record.Rooms) == 0 {
		return []models.Zone{{Number: 1, Name: "Об'єкт", SensorType: "Приміщення", Status: models.ZoneNormal}}
	}

	zones := make([]models.Zone, 0, len(record.Rooms))
	for idx, room := range record.Rooms {
		number := idx + 1
		if parsed := parseCASLID(room.RoomID); parsed > 0 {
			number = parsed
		}

		name := strings.TrimSpace(room.Name)
		if name == "" {
			name = fmt.Sprintf("Приміщення %d", idx+1)
		}

		sensorType := "Приміщення"
		if strings.TrimSpace(room.Description) != "" {
			sensorType = strings.TrimSpace(room.Description)
		}

		zones = append(zones, models.Zone{Number: number, Name: name, SensorType: sensorType, Status: models.ZoneNormal})
	}

	sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
	return zones
}

func (p *CASLCloudProvider) GetEmployees(objectID string) []models.Contact {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	orderedIDs := normalizeContactIDs(record.InCharge, record.ManagerID)
	if len(orderedIDs) == 0 {
		return nil
	}

	users, usersErr := p.loadUsers(ctx)
	if usersErr != nil {
		log.Debug().Err(usersErr).Msg("CASL: не вдалося завантажити read_user, повертаю fallback контакти")
	}

	contacts := make([]models.Contact, 0, len(orderedIDs))
	for idx, userID := range orderedIDs {
		user, hasUser := users[userID]
		if !hasUser {
			contacts = append(contacts, models.Contact{Name: "Користувач #" + userID, Position: "IN_CHARGE", Priority: idx + 1})
			continue
		}

		contacts = append(contacts, models.Contact{
			Name:     user.FullName(),
			Position: strings.TrimSpace(user.Role),
			Phone:    user.PrimaryPhone(),
			Priority: idx + 1,
			CodeWord: strings.TrimSpace(user.Tag.String()),
		})
	}

	return contacts
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
			ID:         nextCASLEventID(),
			Time:       time.Now(),
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			Details:    fmt.Sprintf("CASL Cloud: активних тривог у кошику %d (було %d)", basketCount, p.lastBasketCount),
			SC1:        eventSC1,
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
	p.updateRealtimeAlarmsFromRows(ctx, rows)
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

		seed := code + "|" + contactID + "|" + strconv.Itoa(number) + "|" + strings.TrimSpace(row.HozUserID)
		events = append(events, models.Event{
			ID:         stableCASLEventID(strconv.FormatInt(ppkNum, 10), eventTS, seed, 0),
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: number,
			Details:    details,
			SC1:        mapCASLEventSC1(eventType),
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

func extractCASLRealtimeConnID(raw []byte) string {
	payload, ok := decodeCASLRealtimePayload(raw)
	if ok {
		if id := extractCASLConnIDFromAny(payload); id != "" {
			return id
		}
	}

	text := strings.TrimSpace(string(raw))
	if text == "" {
		return ""
	}
	if isLikelyCASLConnID(text) {
		return text
	}

	lower := strings.ToLower(text)
	candidates := []string{"conn_id", "connid", "sid", "connection_id"}
	for _, key := range candidates {
		idx := strings.Index(lower, key)
		if idx < 0 {
			continue
		}
		tail := text[idx+len(key):]
		tail = strings.TrimLeft(tail, " \t\r\n:=\">'`")
		if tail == "" {
			continue
		}

		end := len(tail)
		for i, r := range tail {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				continue
			}
			end = i
			break
		}
		id := strings.TrimSpace(tail[:end])
		if isLikelyCASLConnID(id) {
			return id
		}
	}

	return ""
}

func extractCASLRealtimeRows(raw []byte) []CASLObjectEvent {
	payload, ok := decodeCASLRealtimePayload(raw)
	if !ok {
		return nil
	}
	rows := make([]CASLObjectEvent, 0, 4)
	collectCASLRealtimeRows(payload, "", &rows)
	return rows
}

func decodeCASLRealtimePayload(raw []byte) (any, bool) {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 {
		return nil, false
	}
	if body[0] != '{' && body[0] != '[' {
		idx := bytes.IndexAny(body, "{[")
		if idx < 0 {
			return nil, false
		}
		body = body[idx:]
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	return payload, true
}

func collectCASLRealtimeRows(value any, fallbackType string, rows *[]CASLObjectEvent) {
	switch typed := value.(type) {
	case map[string]any:
		nextType := fallbackType
		if tag := strings.TrimSpace(asString(typed["tag"])); tag != "" {
			nextType = tag
		}
		if eventName := strings.TrimSpace(asString(typed["event"])); eventName != "" {
			nextType = eventName
		}
		if row, ok := mapCASLRealtimeRow(typed, nextType); ok {
			*rows = append(*rows, row)
		}
		for _, nested := range typed {
			collectCASLRealtimeRows(nested, nextType, rows)
		}
	case []any:
		for _, nested := range typed {
			collectCASLRealtimeRows(nested, fallbackType, rows)
		}
	}
}

func mapCASLRealtimeRow(source map[string]any, fallbackType string) (CASLObjectEvent, bool) {
	deviceID := strings.TrimSpace(asString(source["device_id"]))
	if deviceID == "" {
		deviceID = strings.TrimSpace(asString(source["deviceId"]))
	}
	rawObjID := strings.TrimSpace(asString(source["obj_id"]))
	if rawObjID == "" {
		rawObjID = strings.TrimSpace(asString(source["object_id"]))
	}

	ppkNum := int64(parseCASLAnyInt(source["ppk_num"]))
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["ppkNum"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["ppk"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["device_number"]))
	}
	if ppkNum <= 0 {
		ppkNum = int64(parseCASLAnyInt(source["device_num"]))
	}
	if ppkNum <= 0 && deviceID == "" && rawObjID == "" {
		return CASLObjectEvent{}, false
	}

	number := int64(parseCASLAnyInt(source["number"]))
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["zone"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["line"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["line_number"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["group"]))
	}
	if number <= 0 {
		number = int64(parseCASLAnyInt(source["group_number"]))
	}

	code := strings.TrimSpace(asString(source["code"]))
	if code == "" {
		code = strings.TrimSpace(asString(source["event_code"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["ppk_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["user_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["mgr_action_type"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["action"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["subtype"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["msg"]))
	}
	if code == "" {
		code = strings.TrimSpace(asString(source["type_event"]))
	}
	contactID := strings.TrimSpace(asString(source["contact_id"]))
	if contactID == "" {
		contactID = strings.TrimSpace(asString(source["contactId"]))
	}
	rowType := strings.TrimSpace(asString(source["type"]))
	if rowType == "" {
		rowType = fallbackType
	}
	if rowType == "" {
		rowType = strings.TrimSpace(asString(source["user_action_type"]))
	}
	if rowType == "" {
		rowType = strings.TrimSpace(asString(source["module"]))
	}
	if code == "" && contactID == "" {
		code = strings.TrimSpace(asString(source["status"]))
	}
	if code == "" && contactID == "" {
		code = rowType
	}

	ts := parseCASLAnyTime(source["time"])
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["timestamp"])
	}
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["ts"])
	}
	if ts.IsZero() {
		ts = parseCASLAnyTime(source["create_date"])
	}

	row := CASLObjectEvent{
		PPKNum:    ppkNum,
		DeviceID:  deviceID,
		ObjID:     rawObjID,
		ObjName:   strings.TrimSpace(asString(source["obj_name"])),
		ObjAddr:   strings.TrimSpace(asString(source["obj_address"])),
		Action:    strings.TrimSpace(asString(source["action"])),
		AlarmType: strings.TrimSpace(asString(source["alarm_type"])),
		MgrID:     strings.TrimSpace(asString(source["mgr_id"])),
		UserID:    strings.TrimSpace(asString(source["user_id"])),
		UserFIO:   strings.TrimSpace(asString(source["user_fio"])),
		Time:      0,
		Code:      code,
		Type:      rowType,
		Number:    number,
		ContactID: contactID,
		HozUserID: strings.TrimSpace(asString(source["hoz_user_id"])),
	}
	if row.HozUserID == "" {
		row.HozUserID = strings.TrimSpace(asString(source["user_id"]))
	}
	if !ts.IsZero() {
		row.Time = ts.UnixMilli()
	}
	if row.Code == "" && row.ContactID == "" && row.Number == 0 && row.Action == "" {
		return CASLObjectEvent{}, false
	}
	return row, true
}

func extractCASLConnIDFromAny(value any) string {
	switch typed := value.(type) {
	case map[string]any:
		rowType := strings.ToLower(strings.TrimSpace(asString(typed["type"])))
		if rowType == "conn_id" || rowType == "get_id" {
			if id := strings.TrimSpace(asString(typed["id"])); isLikelyCASLConnID(id) {
				return id
			}
		}
		candidates := []string{"conn_id", "connId", "sid", "connection_id"}
		for _, key := range candidates {
			if id := strings.TrimSpace(asString(typed[key])); id != "" {
				return id
			}
		}
		for _, nested := range typed {
			if id := extractCASLConnIDFromAny(nested); id != "" {
				return id
			}
		}
	case []any:
		for _, nested := range typed {
			if id := extractCASLConnIDFromAny(nested); id != "" {
				return id
			}
		}
	}
	return ""
}

func randomCASLConnID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(buf)
}

func extractCASLConnIDFromWSURL(raw string) string {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	query := parsed.Query()
	for _, key := range []string{"conn_id", "connId", "sid", "connection_id", "id"} {
		value := strings.TrimSpace(query.Get(key))
		if isLikelyCASLConnID(value) {
			return value
		}
	}
	return ""
}

func sendCASLRealtimeGetID(conn *websocket.Conn, userID string) error {
	payload := map[string]any{
		"type": "get_id",
	}
	if strings.TrimSpace(userID) != "" {
		payload["user_id"] = strings.TrimSpace(userID)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return websocket.Message.Send(conn, body)
}

func isLikelyCASLConnID(value string) bool {
	id := strings.TrimSpace(value)
	if len(id) < 8 || len(id) > 128 {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func (p *CASLCloudProvider) readGeneralTapeAsEvents(ctx context.Context) ([]models.Event, error) {
	rows, err := p.ReadGeneralTapeObjects(ctx)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	objectNames := map[string]string{}
	if records, loadErr := p.loadObjects(ctx); loadErr == nil {
		for _, record := range records {
			objID := strings.TrimSpace(record.ObjID)
			if objID == "" {
				continue
			}
			name := strings.TrimSpace(record.Name)
			if name == "" {
				name = "Об'єкт #" + objID
			}
			objectNames[objID] = name
		}
	}
	dictMap := p.loadDictionaryMap(ctx)

	events := make([]models.Event, 0, len(rows))
	for idx, row := range rows {
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		objectID := mapCASLObjectID(rawObjID, asString(row["number"]), asString(row["device_number"]))

		objectName := strings.TrimSpace(objectNames[rawObjID])
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["obj_name"]))
		}
		if objectName == "" {
			objectName = strings.TrimSpace(asString(row["name"]))
		}
		if objectName == "" {
			objectName = "Об'єкт #" + strings.TrimSpace(rawObjID)
		}
		objectName = formatCASLJournalObjectName(rawObjID, objectName)

		eventTime := parseCASLAnyTime(row["time"])
		if eventTime.IsZero() {
			eventTime = time.Now()
		}

		sourceType := strings.TrimSpace(asString(row["event_type"]))
		code := strings.TrimSpace(asString(row["code"]))
		contactID := strings.TrimSpace(asString(row["contact_id"]))
		details := strings.TrimSpace(asString(row["description"]))
		deviceType := strings.TrimSpace(asString(row["device_type"]))
		if details == "" {
			details = decodeCASLEventDescription(nil, dictMap, code, contactID, parseCASLAnyInt(row["zone"]), deviceType)
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
			ID:         eventID,
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: parseCASLAnyInt(row["zone"]),
			Details:    details,
			UserName:   strings.TrimSpace(asString(row["user_id"])),
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	sortEvents(events)
	if len(events) > caslMaxCachedEvents {
		events = events[:caslMaxCachedEvents]
	}
	return events, nil
}

func (p *CASLCloudProvider) readFromBasketAsAlarms(ctx context.Context) ([]models.Alarm, error) {
	rows, err := p.ReadFromBasket(ctx, 0, 10_000)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}

	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		if ppkNum > 0 {
			ppkFilter[ppkNum] = struct{}{}
		}
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		if rawObjID != "" {
			objFilter[rawObjID] = struct{}{}
		}
	}

	contextByPPK, ctxErr := p.loadEventContextsByPPK(ctx, ppkFilter)
	if ctxErr != nil {
		log.Debug().Err(ctxErr).Msg("CASL: не вдалося побудувати контексти ППК для кошика тривог")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)

	alarms := make([]models.Alarm, 0, len(rows))
	for _, row := range rows {
		ppkNum := int64(parseCASLAnyInt(row["ppk_num"]))
		if ppkNum <= 0 {
			ppkNum = int64(parseCASLAnyInt(row["device_number"]))
		}
		rawObjID := strings.TrimSpace(asString(row["obj_id"]))
		if rawObjID == "" {
			rawObjID = strings.TrimSpace(asString(row["object_id"]))
		}
		if ppkNum <= 0 && rawObjID == "" {
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

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}
		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10))
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

		seed := code + "|" + contactID + "|" + strconv.Itoa(number)
		seedObjectID := strconv.FormatInt(ppkNum, 10)
		if strings.TrimSpace(seedObjectID) == "" || seedObjectID == "0" {
			seedObjectID = objectNum
		}
		if seedObjectID == "" {
			seedObjectID = "casl"
		}
		alarms = append(alarms, models.Alarm{
			ID:         stableCASLEventID(seedObjectID, tsValue.UnixMilli(), seed, 0),
			ObjectID:   objectID,
			ObjectName: objectName,
			Address:    strings.TrimSpace(asString(row["address"])),
			Time:       tsValue.Local(),
			Details:    details,
			Type:       alarmType,
			ZoneNumber: number,
			SC1:        mapCASLEventSC1(eventType),
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
		return nil
	}

	events := p.mapCASLObjectEvents(ctx, record, rawEvents)
	sortEvents(events)

	p.mu.Lock()
	p.cachedObjectEvents[internalID] = append([]models.Event(nil), events...)
	p.cachedObjectEventsAt[internalID] = now
	p.mu.Unlock()

	return events
}

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	p.ensureRealtimeStream()

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()
	if _, err := p.readEventsJournalAsEvents(ctx); err != nil {
		log.Debug().Err(err).Msg("CASL: read_events недоступний під час формування активних тривог")
	}

	alarms := p.snapshotRealtimeAlarms()
	if len(alarms) == 0 {
		rows, err := p.readGeneralTapeItemRows(ctx)
		if err != nil {
			log.Debug().Err(err).Msg("CASL: get_general_tape_item недоступний під час формування активних тривог")
		} else if len(rows) > 0 {
			logCASLGeneralTapeItemRows(rows)
			p.updateRealtimeAlarmsFromRows(ctx, rows)
			alarms = p.snapshotRealtimeAlarms()
		}
	}
	if len(alarms) == 0 {
		return nil
	}
	sortCASLAlarms(alarms)
	return alarms
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
	log.Warn().Str("alarmID", id).Str("user", user).Msg("CASL: ProcessAlarm не підтримується API інтеграцією")
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

func (p *CASLCloudProvider) ensureRealtimeStream() {
	p.mu.RLock()
	wsURL := strings.TrimSpace(p.wsURL)
	p.mu.RUnlock()
	if wsURL == "" {
		ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
		_, _ = p.ensureToken(ctx)
		cancel()
		p.mu.RLock()
		wsURL = strings.TrimSpace(p.wsURL)
		p.mu.RUnlock()
	}
	if wsURL == "" {
		return
	}

	p.realtimeMu.Lock()
	if p.realtimeRunning {
		p.realtimeMu.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	p.realtimeCancel = cancel
	p.realtimeRunning = true
	p.realtimeMu.Unlock()

	go p.runRealtimeLoop(ctx)
}

func (p *CASLCloudProvider) runRealtimeLoop(ctx context.Context) {
	defer func() {
		p.realtimeMu.Lock()
		if p.realtimeCancel != nil {
			p.realtimeCancel = nil
		}
		p.realtimeRunning = false
		p.realtimeSubscribed = false
		p.realtimeMu.Unlock()
	}()

	backoff := time.Second
	lastLogAt := time.Time{}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := p.runRealtimeSession(ctx); err != nil && !errors.Is(err, context.Canceled) {
			now := time.Now()
			if lastLogAt.IsZero() || now.Sub(lastLogAt) >= 30*time.Second {
				log.Debug().Err(err).Msg("CASL realtime stream: reconnect")
				lastLogAt = now
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > caslRealtimeBackoff {
			backoff = caslRealtimeBackoff
		}
	}
}

func (p *CASLCloudProvider) runRealtimeSession(ctx context.Context) error {
	p.realtimeMu.Lock()
	p.realtimeSubscribed = false
	p.realtimeMu.Unlock()

	ensureCtx, ensureCancel := context.WithTimeout(ctx, caslHTTPTimeout)
	if _, err := p.ensureToken(ensureCtx); err != nil {
		ensureCancel()
		return err
	}
	ensureCancel()

	p.mu.RLock()
	wsURL := strings.TrimSpace(p.wsURL)
	userID := strings.TrimSpace(p.userID)
	p.mu.RUnlock()
	if wsURL == "" {
		return errors.New("casl realtime: empty ws_url")
	}

	origin := p.baseURL
	if parsed, err := url.Parse(wsURL); err == nil && parsed.Scheme == "wss" {
		origin = strings.Replace(origin, "http://", "https://", 1)
	}

	cfg, err := websocket.NewConfig(wsURL, origin)
	if err != nil {
		return fmt.Errorf("casl realtime config: %w", err)
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		return fmt.Errorf("casl realtime dial: %w", err)
	}
	defer conn.Close()
	if getIDErr := sendCASLRealtimeGetID(conn, userID); getIDErr != nil {
		log.Debug().Err(getIDErr).Msg("CASL realtime: get_id send failed")
	} else {
		log.Debug().Msg("CASL realtime: get_id sent")
	}

	rawCh := make(chan []byte, 64)
	errCh := make(chan error, 1)
	go func() {
		defer close(rawCh)
		for {
			var raw []byte
			if recvErr := websocket.Message.Receive(conn, &raw); recvErr != nil {
				select {
				case errCh <- recvErr:
				default:
				}
				return
			}
			if len(raw) == 0 {
				continue
			}
			msg := append([]byte(nil), raw...)
			select {
			case rawCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	connID := extractCASLConnIDFromWSURL(wsURL)
	if connID != "" {
		log.Debug().Str("conn_id", connID).Msg("CASL realtime conn_id extracted from ws_url")
	}
	subscribed := false
	subscribeTicker := time.NewTicker(1500 * time.Millisecond)
	defer subscribeTicker.Stop()
	lastSubscribeErrAt := time.Time{}
	lastSubscribeErrText := ""

	subscribeOnce := func(candidate string) {
		if subscribed {
			return
		}
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			return
		}

		subCtx, subCancel := context.WithTimeout(ctx, caslHTTPTimeout)
		subErr := p.subscribeRealtimeTags(subCtx, candidate)
		subCancel()
		if subErr != nil {
			errText := strings.TrimSpace(subErr.Error())
			if errText == "" {
				errText = "unknown"
			}
			now := time.Now()
			if errText != lastSubscribeErrText || lastSubscribeErrAt.IsZero() || now.Sub(lastSubscribeErrAt) >= 15*time.Second {
				log.Debug().Err(subErr).Str("conn_id", candidate).Msg("CASL realtime subscribe failed")
				lastSubscribeErrText = errText
				lastSubscribeErrAt = now
			}
			return
		}

		subscribed = true
		p.realtimeMu.Lock()
		p.realtimeSubscribed = true
		p.realtimeMu.Unlock()
		log.Debug().Str("conn_id", candidate).Msg("CASL realtime subscribed")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case recvErr := <-errCh:
			if recvErr == nil {
				return io.EOF
			}
			return recvErr
		case raw, ok := <-rawCh:
			if !ok {
				return io.EOF
			}

			if connID == "" {
				if extracted := extractCASLRealtimeConnID(raw); extracted != "" {
					connID = extracted
					log.Debug().Str("conn_id", connID).Msg("CASL realtime conn_id extracted")
				}
			}
			if !subscribed && connID != "" {
				subscribeOnce(connID)
			}

			rows := extractCASLRealtimeRows(raw)
			if len(rows) > 0 {
				if appendErr := p.appendRealtimeRows(ctx, rows); appendErr != nil {
					log.Debug().Err(appendErr).Msg("CASL realtime append failed")
				}
			}
		case <-subscribeTicker.C:
			if subscribed {
				continue
			}
			if connID == "" {
				continue
			}
			subscribeOnce(connID)
		}
	}
}

func (p *CASLCloudProvider) subscribeRealtimeTags(ctx context.Context, connID string) error {
	type tagSpec struct {
		name     string
		required bool
	}

	tags := []tagSpec{
		{name: "ppk_in", required: true},
		{name: "user_action", required: true},
		{name: "ppk_service", required: false},
		{name: "ppk_out", required: false},
		{name: "system_event", required: false},
		{name: "system_action", required: false},
		{name: "m3_in", required: false},
		{name: "mob_user_action", required: false},
		{name: "system_info", required: false},
		{name: "storage_change", required: false},
		{name: "notif", required: false},
		{name: "chat_action", required: false},
		{name: "rtsp_action", required: false},
		{name: "ping", required: false},
	}

	requiredSubscribed := 0
	totalSubscribed := 0
	for _, tag := range tags {
		if err := p.Subscribe(ctx, connID, tag.name); err != nil {
			if isCASLUnknownTagErr(err) {
				if tag.required {
					return fmt.Errorf("casl subscribe tag=%s: %w", tag.name, err)
				}
				continue
			}
			if !tag.required {
				log.Debug().Err(err).Str("tag", tag.name).Msg("CASL realtime subscribe: optional tag subscribe failed")
				continue
			}
			return fmt.Errorf("casl subscribe tag=%s: %w", tag.name, err)
		}
		totalSubscribed++
		if tag.required {
			requiredSubscribed++
		}
	}

	if totalSubscribed == 0 {
		return errors.New("casl subscribe: no tags were subscribed")
	}
	if requiredSubscribed == 0 {
		log.Debug().Msg("CASL realtime subscribe: required tags are unavailable, running on optional tags only")
	}

	return nil
}

func (p *CASLCloudProvider) appendRealtimeRows(ctx context.Context, rows []CASLObjectEvent) error {
	p.mu.RLock()
	startGate := p.eventsStartAtMs
	p.mu.RUnlock()

	events, maxEventTime := p.mapCASLRowsToEvents(ctx, rows, startGate)
	if len(events) == 0 {
		p.updateRealtimeAlarmsFromRows(ctx, rows)
		return nil
	}

	p.mu.Lock()
	added := p.mergeCachedEventsLocked(events)
	if maxEventTime > p.eventsCursorMs {
		p.eventsCursorMs = maxEventTime
	}
	if added > 0 {
		p.eventsRevision++
	}
	p.mu.Unlock()

	p.updateRealtimeAlarmsFromRows(ctx, rows)
	return nil
}

func (p *CASLCloudProvider) snapshotRealtimeAlarms() []models.Alarm {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.realtimeAlarmByObjID) == 0 {
		return nil
	}

	alarms := make([]models.Alarm, 0, len(p.realtimeAlarmByObjID))
	for _, alarm := range p.realtimeAlarmByObjID {
		alarms = append(alarms, alarm)
	}
	return alarms
}

func mergeCASLAlarms(primary []models.Alarm, secondary []models.Alarm) []models.Alarm {
	if len(secondary) == 0 {
		return append([]models.Alarm(nil), primary...)
	}

	out := append([]models.Alarm(nil), primary...)
	seen := make(map[string]int, len(out))
	for i := range out {
		seen[caslAlarmMergeKey(out[i])] = i
	}

	for _, alarm := range secondary {
		key := caslAlarmMergeKey(alarm)
		if idx, exists := seen[key]; exists {
			if alarm.Time.After(out[idx].Time) {
				out[idx] = alarm
			}
			continue
		}
		seen[key] = len(out)
		out = append(out, alarm)
	}

	return out
}

func caslAlarmMergeKey(alarm models.Alarm) string {
	return strconv.Itoa(alarm.ObjectID) + "|" + strconv.Itoa(alarm.ZoneNumber) + "|" + string(alarm.Type)
}

func sortCASLAlarms(alarms []models.Alarm) {
	sort.SliceStable(alarms, func(i, j int) bool {
		left := alarms[i].Time
		right := alarms[j].Time
		if left.Equal(right) {
			return alarms[i].ID > alarms[j].ID
		}
		return left.After(right)
	})
}

func canonicalCASLRealtimeObjectKey(rawObjID string, objectNum string, objectID int) string {
	if number := strings.TrimSpace(objectNum); number != "" {
		return number
	}
	if value := strings.TrimSpace(rawObjID); value != "" {
		return value
	}
	if objectID > 0 {
		return strconv.Itoa(objectID)
	}
	return ""
}

func canonicalCASLRealtimeAlarmKey(objectKey string, zoneNumber int) string {
	key := strings.TrimSpace(objectKey)
	if key == "" {
		return ""
	}
	if zoneNumber <= 0 {
		return key + "|z0"
	}
	return key + "|z" + strconv.Itoa(zoneNumber)
}

func deleteRealtimeAlarmsByObjectKey(cache map[string]models.Alarm, objectKey string) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return
	}
	prefix := objectKey + "|"
	for key := range cache {
		if strings.HasPrefix(key, prefix) || key == objectKey {
			delete(cache, key)
		}
	}
}

func isCASLEventAlarmCandidate(eventType models.EventType) bool {
	switch eventType {
	case models.EventFire,
		models.EventBurglary,
		models.EventPanic,
		models.EventMedical,
		models.EventGas,
		models.EventTamper,
		models.EventFault,
		models.EventPowerFail,
		models.EventBatteryLow,
		models.EventOffline:
		return true
	default:
		return false
	}
}

func (p *CASLCloudProvider) updateRealtimeAlarmsFromRows(ctx context.Context, rows []CASLObjectEvent) {
	if len(rows) == 0 {
		return
	}

	ppkFilter := make(map[int64]struct{}, len(rows))
	objFilter := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if row.PPKNum > 0 {
			ppkFilter[row.PPKNum] = struct{}{}
		}
		if objID := strings.TrimSpace(row.ObjID); objID != "" {
			objFilter[objID] = struct{}{}
		}
	}

	contextByPPK, err := p.loadEventContextsByPPK(ctx, ppkFilter)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити контексти ППК для realtime alarm cache")
	}
	contextByObject := p.loadEventContextsByObjectNum(ctx, objFilter, contextByPPK)
	dictMap := p.loadDictionaryMap(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, row := range rows {
		action := strings.ToUpper(strings.TrimSpace(row.Action))
		if action == "" {
			action = strings.ToUpper(strings.TrimSpace(row.Code))
		}
		if action == "" {
			continue
		}

		rawObjID := strings.TrimSpace(row.ObjID)
		ppkNum := row.PPKNum

		ctxItem, hasCtx := contextByPPK[ppkNum]
		if !hasCtx && rawObjID != "" {
			if objCtx, ok := contextByObject[rawObjID]; ok {
				ctxItem = objCtx
				hasCtx = true
			}
		}

		objectID := mapCASLObjectID(rawObjID, strconv.FormatInt(ppkNum, 10))
		objectName := strings.TrimSpace(row.ObjName)
		objectNum := preferredCASLObjectNumber(rawObjID, objectName, ppkNum)

		if hasCtx {
			objectID = ctxItem.ObjectID
			if strings.TrimSpace(ctxItem.ObjectNum) != "" {
				objectNum = strings.TrimSpace(ctxItem.ObjectNum)
			}
			if objectName == "" {
				objectName = strings.TrimSpace(ctxItem.ObjectName)
			}
		}

		if objectName == "" {
			if objectNum != "" {
				objectName = "Об'єкт #" + objectNum
			} else if ppkNum > 0 {
				objectName = "Об'єкт ППК #" + strconv.FormatInt(ppkNum, 10)
			} else {
				objectName = "Об'єкт"
			}
		}
		objectName = formatCASLJournalObjectName(objectNum, objectName)

		objectKey := canonicalCASLRealtimeObjectKey(rawObjID, objectNum, objectID)
		if objectKey == "" {
			continue
		}
		zoneNumber := int(row.Number)
		cacheKey := canonicalCASLRealtimeAlarmKey(objectKey, zoneNumber)

		switch action {
		case "GRD_OBJ_MGR_CANCEL", "GRD_OBJ_FINISH":
			deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			continue
		}

		deviceType := ""
		if hasCtx {
			deviceType = strings.TrimSpace(ctxItem.DeviceType)
		}

		details := buildCASLUserActionDetails(row)
		if details == "" {
			details = decodeCASLEventDescription(nil, dictMap, row.Code, row.ContactID, int(row.Number), deviceType)
		}
		if details == "" {
			details = strings.TrimSpace(row.Action)
		}
		if details == "" {
			details = strings.TrimSpace(row.Code)
		}
		if details == "" {
			details = "CASL подія"
		}

		classifierCode := strings.TrimSpace(row.Code)
		if classifierCode == "" {
			classifierCode = action
		}
		eventType := classifyCASLEventTypeWithContext(classifierCode, strings.TrimSpace(row.ContactID), strings.TrimSpace(row.Type), details)
		alarmType, include := mapEventTypeToAlarmType(eventType)
		if !include && action == "GRD_OBJ_NOTIF" {
			alarmType = models.AlarmFault
			include = true
		}

		alarmTime := time.Now()
		if row.Time > 0 {
			alarmTime = time.UnixMilli(row.Time).Local()
		}

		if action == "GRD_OBJ_NOTIF" {
			seed := action + "|" + strings.TrimSpace(row.AlarmType) + "|" + strconv.Itoa(zoneNumber)
			p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
				ID:         stableCASLEventID(cacheKey, alarmTime.UnixMilli(), seed, 0),
				ObjectID:   objectID,
				ObjectName: objectName,
				Address:    strings.TrimSpace(row.ObjAddr),
				Time:       alarmTime,
				Details:    details,
				Type:       alarmType,
				ZoneNumber: int(row.Number),
				SC1:        mapCASLEventSC1(eventType),
			}
			continue
		}

		if action == "GRD_OBJ_PICK" || action == "GRD_OBJ_ASS_MGR" {
			// Оновлюємо всі активні тривоги об'єкта даними про оператора/ГМР.
			for key, existing := range p.realtimeAlarmByObjID {
				if !strings.HasPrefix(key, objectKey+"|") {
					continue
				}
				existing.ObjectID = objectID
				existing.ObjectName = objectName
				if strings.TrimSpace(row.ObjAddr) != "" {
					existing.Address = strings.TrimSpace(row.ObjAddr)
				}
				existing.Time = alarmTime
				if details != "" {
					existing.Details = details
				}
				p.realtimeAlarmByObjID[key] = existing
			}
			continue
		}

		if eventType == models.EventRestore || eventType == models.EventPowerOK || eventType == models.EventOnline {
			if zoneNumber > 0 {
				delete(p.realtimeAlarmByObjID, cacheKey)
			} else {
				deleteRealtimeAlarmsByObjectKey(p.realtimeAlarmByObjID, objectKey)
			}
			continue
		}

		if !isCASLEventAlarmCandidate(eventType) || !include {
			continue
		}

		seed := strings.TrimSpace(row.Code) + "|" + strings.TrimSpace(row.ContactID) + "|" + strconv.Itoa(zoneNumber)
		p.realtimeAlarmByObjID[cacheKey] = models.Alarm{
			ID:         stableCASLEventID(cacheKey, alarmTime.UnixMilli(), seed, 0),
			ObjectID:   objectID,
			ObjectName: objectName,
			Address:    strings.TrimSpace(row.ObjAddr),
			Time:       alarmTime,
			Details:    details,
			Type:       alarmType,
			ZoneNumber: zoneNumber,
			SC1:        mapCASLEventSC1(eventType),
		}
	}
}

func (p *CASLCloudProvider) loadObjects(ctx context.Context) ([]caslGrdObject, error) {
	p.mu.RLock()
	cacheValid := len(p.cachedObjects) > 0 && time.Since(p.cachedObjectsAt) < caslObjectsCacheTTL
	if cacheValid {
		copied := append([]caslGrdObject(nil), p.cachedObjects...)
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	records, err := p.readGrdObjects(ctx)
	if err == nil && len(records) > 0 {
		for idx := range records {
			normalizeCASLObjectRecord(&records[idx], caslDevice{})
		}
		p.applyCASLCoreSnapshot(records, nil, nil)
		return append([]caslGrdObject(nil), records...), nil
	}

	connObjects, connDevices, connUsers, connErr := p.readConnectionsCoreSnapshot(ctx)
	if connErr == nil && len(connObjects) > 0 {
		p.applyCASLCoreSnapshot(connObjects, connDevices, connUsers)
		return append([]caslGrdObject(nil), connObjects...), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, connErr
}

func (p *CASLCloudProvider) loadDevices(ctx context.Context) ([]caslDevice, error) {
	p.mu.RLock()
	cacheValid := len(p.deviceByDeviceID) > 0 && time.Since(p.cachedDevicesAt) < caslObjectsCacheTTL
	if cacheValid {
		result := make([]caslDevice, 0, len(p.deviceByDeviceID))
		for _, item := range p.deviceByDeviceID {
			result = append(result, item)
		}
		p.mu.RUnlock()
		return result, nil
	}
	p.mu.RUnlock()

	devices, err := p.readDevices(ctx)
	if err == nil && len(devices) > 0 {
		p.applyCASLCoreSnapshot(nil, devices, nil)
		return append([]caslDevice(nil), devices...), nil
	}

	connObjects, connDevices, connUsers, connErr := p.readConnectionsCoreSnapshot(ctx)
	if connErr == nil && len(connDevices) > 0 {
		p.applyCASLCoreSnapshot(connObjects, connDevices, connUsers)
		return append([]caslDevice(nil), connDevices...), nil
	}

	if err != nil {
		return nil, err
	}
	return nil, connErr
}

func (p *CASLCloudProvider) resolveObjectRecord(ctx context.Context, internalID int) (caslGrdObject, bool, error) {
	p.mu.RLock()
	record, ok := p.objectByInternalID[internalID]
	p.mu.RUnlock()
	if ok {
		return record, true, nil
	}

	records, err := p.loadObjects(ctx)
	if err != nil {
		return caslGrdObject{}, false, err
	}
	for _, item := range records {
		id := mapCASLObjectID(item.ObjID, item.Name, strconv.FormatInt(item.DeviceNumber.Int64(), 10))
		if id == internalID {
			return item, true, nil
		}
	}
	return caslGrdObject{}, false, nil
}

func (p *CASLCloudProvider) resolveDeviceForObject(record caslGrdObject) (caslDevice, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	deviceID := strconv.FormatInt(record.DeviceID.Int64(), 10)
	if deviceID != "" && deviceID != "0" {
		if device, ok := p.deviceByDeviceID[deviceID]; ok {
			return device, true
		}
	}

	objID := strings.TrimSpace(record.ObjID)
	if objID != "" {
		if device, ok := p.deviceByObjectID[objID]; ok {
			return device, true
		}
	}

	deviceNumber := record.DeviceNumber.Int64()
	if deviceNumber > 0 {
		if device, ok := p.deviceByNumber[deviceNumber]; ok {
			return device, true
		}
	}

	return caslDevice{}, false
}

func (p *CASLCloudProvider) resolvePPKByDeviceID(ctx context.Context, deviceID string) (int64, bool) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return 0, false
	}

	p.mu.RLock()
	if device, ok := p.deviceByDeviceID[deviceID]; ok {
		ppkNum := device.Number.Int64()
		p.mu.RUnlock()
		if ppkNum > 0 {
			return ppkNum, true
		}
		return 0, false
	}
	p.mu.RUnlock()

	if _, err := p.loadDevices(ctx); err != nil {
		return 0, false
	}

	p.mu.RLock()
	device, ok := p.deviceByDeviceID[deviceID]
	p.mu.RUnlock()
	if !ok {
		return 0, false
	}

	ppkNum := device.Number.Int64()
	if ppkNum <= 0 {
		return 0, false
	}
	return ppkNum, true
}

func (p *CASLCloudProvider) loadUsers(ctx context.Context) (map[string]caslUser, error) {
	p.mu.RLock()
	cacheValid := len(p.cachedUsers) > 0 && time.Since(p.cachedUsersAt) < caslUsersCacheTTL
	if cacheValid {
		copied := make(map[string]caslUser, len(p.cachedUsers))
		for key, value := range p.cachedUsers {
			copied[key] = value
		}
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	objects, objectsErr := p.loadObjects(ctx)
	if objectsErr == nil && len(objects) > 0 {
		usersFromObjects := collectCASLUsersFromObjects(objects)
		if len(usersFromObjects) > 0 && hasDetailedCASLUsers(usersFromObjects) {
			p.applyCASLCoreSnapshot(nil, nil, usersFromObjects)
			copied := make(map[string]caslUser, len(usersFromObjects))
			for key, value := range usersFromObjects {
				copied[key] = value
			}
			return copied, nil
		}
	}

	users, err := p.readUsers(ctx)
	if err == nil && len(users) > 0 {
		index := make(map[string]caslUser, len(users))
		for _, user := range users {
			appendCASLUserIndex(index, user)
		}
		p.applyCASLCoreSnapshot(nil, nil, index)
		copied := make(map[string]caslUser, len(index))
		for key, value := range index {
			copied[key] = value
		}
		return copied, nil
	}

	connObjects, connDevices, connUsers, connErr := p.readConnectionsCoreSnapshot(ctx)
	if connErr == nil && len(connUsers) > 0 {
		p.applyCASLCoreSnapshot(connObjects, connDevices, connUsers)
		copied := make(map[string]caslUser, len(connUsers))
		for key, value := range connUsers {
			copied[key] = value
		}
		return copied, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, connErr
}

func (p *CASLCloudProvider) readConnectionsCoreSnapshot(ctx context.Context) ([]caslGrdObject, []caslDevice, map[string]caslUser, error) {
	rows, err := p.readConnections(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	if len(rows) == 0 {
		return nil, nil, nil, errors.New("casl: read_connections returned empty payload")
	}

	records := make([]caslGrdObject, 0, len(rows))
	devices := make([]caslDevice, 0, len(rows))
	users := make(map[string]caslUser, len(rows)*2)

	objectSeen := make(map[string]struct{}, len(rows))
	deviceSeen := make(map[string]struct{}, len(rows))

	for _, row := range rows {
		record := row.GuardedObject
		device := row.Device
		normalizeCASLObjectRecord(&record, device)

		if strings.TrimSpace(device.ObjID.String()) == "" && strings.TrimSpace(record.ObjID) != "" {
			device.ObjID = caslText(record.ObjID)
		}

		objectKey := strings.TrimSpace(record.ObjID)
		if objectKey == "" {
			objectKey = strings.TrimSpace(record.Name) + "|" + strconv.FormatInt(record.DeviceNumber.Int64(), 10)
		}
		if objectKey != "" {
			if _, exists := objectSeen[objectKey]; !exists {
				objectSeen[objectKey] = struct{}{}
				records = append(records, record)
			}
		}

		deviceKey := strings.TrimSpace(device.DeviceID.String())
		if deviceKey == "" {
			number := device.Number.Int64()
			if number > 0 {
				deviceKey = "num:" + strconv.FormatInt(number, 10)
			}
		}
		if deviceKey != "" {
			if _, exists := deviceSeen[deviceKey]; !exists {
				deviceSeen[deviceKey] = struct{}{}
				devices = append(devices, device)
			}
		}

		appendCASLUserIndex(users, record.Manager)
		for _, room := range record.Rooms {
			for _, roomUser := range room.Users {
				appendCASLUserIndex(users, roomUser)
			}
		}
	}

	if len(records) == 0 {
		return nil, nil, nil, errors.New("casl: read_connections payload has no objects")
	}

	return records, devices, users, nil
}

func (p *CASLCloudProvider) applyCASLCoreSnapshot(objects []caslGrdObject, devices []caslDevice, users map[string]caslUser) {
	now := time.Now()

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(objects) > 0 {
		copiedObjects := append([]caslGrdObject(nil), objects...)
		p.cachedObjects = copiedObjects
		p.cachedObjectsAt = now
		p.objectByInternalID = buildCASLObjectIndex(copiedObjects)
	}

	if len(devices) > 0 {
		copiedDevices := append([]caslDevice(nil), devices...)
		p.deviceByDeviceID, p.deviceByObjectID, p.deviceByNumber = buildCASLDeviceIndexes(copiedDevices)
		p.cachedDevicesAt = now
	}

	if len(users) > 0 {
		copiedUsers := make(map[string]caslUser, len(users))
		for key, value := range users {
			copiedUsers[key] = value
		}
		p.cachedUsers = copiedUsers
		p.cachedUsersAt = now
	}
}

func buildCASLObjectIndex(records []caslGrdObject) map[int]caslGrdObject {
	index := make(map[int]caslGrdObject, len(records))
	for _, record := range records {
		internalID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
		index[internalID] = record
	}
	return index
}

func buildCASLDeviceIndexes(devices []caslDevice) (map[string]caslDevice, map[string]caslDevice, map[int64]caslDevice) {
	byDeviceID := make(map[string]caslDevice, len(devices))
	byObjectID := make(map[string]caslDevice, len(devices))
	byNumber := make(map[int64]caslDevice, len(devices))

	for _, device := range devices {
		deviceID := strings.TrimSpace(device.DeviceID.String())
		if deviceID != "" {
			byDeviceID[deviceID] = device
		}
		objectID := strings.TrimSpace(device.ObjID.String())
		if objectID != "" {
			byObjectID[objectID] = device
		}
		number := device.Number.Int64()
		if number > 0 {
			byNumber[number] = device
		}
	}

	return byDeviceID, byObjectID, byNumber
}

func normalizeCASLObjectRecord(record *caslGrdObject, device caslDevice) {
	if record == nil {
		return
	}

	managerID := strings.TrimSpace(record.ManagerID)
	if managerID == "" {
		managerID = strings.TrimSpace(record.Manager.UserID)
	}
	record.ManagerID = managerID

	seen := make(map[string]struct{}, len(record.InCharge)+len(record.Rooms)*2)
	inCharge := make([]string, 0, len(record.InCharge)+len(record.Rooms)*2)
	for _, userID := range record.InCharge {
		inCharge = appendCASLUniqueID(inCharge, seen, userID)
	}
	for _, room := range record.Rooms {
		for _, roomUser := range room.Users {
			inCharge = appendCASLUniqueID(inCharge, seen, roomUser.UserID)
		}
	}
	if managerID != "" {
		filtered := inCharge[:0]
		for _, userID := range inCharge {
			if strings.TrimSpace(userID) == managerID {
				continue
			}
			filtered = append(filtered, userID)
		}
		inCharge = filtered
	}
	record.InCharge = inCharge

	if strings.TrimSpace(record.ObjID) == "" {
		record.ObjID = strings.TrimSpace(device.ObjID.String())
	}
	if record.DeviceID.Int64() <= 0 {
		if parsed := parseCASLAnyInt(device.DeviceID.String()); parsed > 0 {
			record.DeviceID = caslInt64(parsed)
		}
	}
	if record.DeviceNumber.Int64() <= 0 && device.Number.Int64() > 0 {
		record.DeviceNumber = device.Number
	}
}

func appendCASLUniqueID(dst []string, seen map[string]struct{}, raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return dst
	}
	if _, exists := seen[value]; exists {
		return dst
	}
	seen[value] = struct{}{}
	return append(dst, value)
}

func collectCASLUsersFromObjects(records []caslGrdObject) map[string]caslUser {
	index := make(map[string]caslUser, len(records)*2)
	for _, record := range records {
		appendCASLUserIndex(index, record.Manager)
		for _, room := range record.Rooms {
			for _, roomUser := range room.Users {
				appendCASLUserIndex(index, roomUser)
			}
		}
	}
	return index
}

func hasDetailedCASLUsers(users map[string]caslUser) bool {
	for _, user := range users {
		if hasCASLUserDetails(user) {
			return true
		}
	}
	return false
}

func hasCASLUserDetails(user caslUser) bool {
	if strings.TrimSpace(user.LastName) != "" ||
		strings.TrimSpace(user.FirstName) != "" ||
		strings.TrimSpace(user.MiddleName) != "" ||
		strings.TrimSpace(user.Email) != "" ||
		strings.TrimSpace(user.Role) != "" ||
		strings.TrimSpace(user.Tag.String()) != "" {
		return true
	}
	for _, phone := range user.PhoneNumbers {
		if strings.TrimSpace(phone.Number) != "" {
			return true
		}
	}
	return false
}

func appendCASLUserIndex(index map[string]caslUser, user caslUser) {
	userID := strings.TrimSpace(user.UserID)
	if userID == "" {
		return
	}
	if existing, ok := index[userID]; ok {
		index[userID] = mergeCASLUsers(existing, user)
		return
	}
	index[userID] = user
}

func mergeCASLUsers(base caslUser, incoming caslUser) caslUser {
	if strings.TrimSpace(base.Email) == "" {
		base.Email = incoming.Email
	}
	if strings.TrimSpace(base.LastName) == "" {
		base.LastName = incoming.LastName
	}
	if strings.TrimSpace(base.FirstName) == "" {
		base.FirstName = incoming.FirstName
	}
	if strings.TrimSpace(base.MiddleName) == "" {
		base.MiddleName = incoming.MiddleName
	}
	if strings.TrimSpace(base.Role) == "" {
		base.Role = incoming.Role
	}
	if strings.TrimSpace(base.Tag.String()) == "" {
		base.Tag = incoming.Tag
	}
	if len(base.PhoneNumbers) == 0 && len(incoming.PhoneNumbers) > 0 {
		base.PhoneNumbers = append([]caslPhoneNumber(nil), incoming.PhoneNumbers...)
	}
	return base
}

func (p *CASLCloudProvider) readGrdObjects(ctx context.Context) ([]caslGrdObject, error) {
	payload := map[string]any{"type": "read_grd_object", "skip": 0, "limit": caslReadLimit}

	var resp caslReadGrdObjectResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	return append([]caslGrdObject(nil), resp.Data...), nil
}

func (p *CASLCloudProvider) readUsers(ctx context.Context) ([]caslUser, error) {
	payload := map[string]any{"type": "read_user", "skip": 0, "limit": caslReadLimit}

	var resp caslReadUserResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	return append([]caslUser(nil), resp.Data...), nil
}

func (p *CASLCloudProvider) readDevices(ctx context.Context) ([]caslDevice, error) {
	payload := map[string]any{"type": "read_device", "skip": 0, "limit": caslReadLimit}

	var resp caslReadDeviceResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}

	return append([]caslDevice(nil), resp.Data...), nil
}

func (p *CASLCloudProvider) readConnections(ctx context.Context) ([]caslConnectionRecord, error) {
	payload := map[string]any{"type": "read_connections", "skip": 0, "limit": caslReadLimit}

	var resp struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
		Error  string          `json:"error"`
	}
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 || string(resp.Data) == "null" {
		return nil, nil
	}

	var rows []caslConnectionRecord
	if err := json.Unmarshal(resp.Data, &rows); err == nil {
		return rows, nil
	}

	var single caslConnectionRecord
	if err := json.Unmarshal(resp.Data, &single); err == nil {
		if single.hasPayload() {
			return []caslConnectionRecord{single}, nil
		}
	}

	return nil, fmt.Errorf("casl read_connections: unsupported payload format")
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

func (p *CASLCloudProvider) readBasketCount(ctx context.Context) (int, error) {
	if !p.hasAuthData() {
		return 0, nil
	}

	payload := map[string]any{"type": "read_count_in_basket"}

	var resp caslBasketResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return 0, err
	}
	return resp.Count, nil
}

func (p *CASLCloudProvider) readPultsPublic(ctx context.Context) ([]caslPult, error) {
	payload := map[string]any{"type": "read_pult", "skip": 0, "limit": caslReadLimit}

	var resp caslReadPultResponse
	if err := p.postCommand(ctx, payload, &resp, false); err != nil {
		return nil, err
	}

	return append([]caslPult(nil), resp.Data...), nil
}

func (p *CASLCloudProvider) postCommand(ctx context.Context, payload map[string]any, out any, requireAuth bool) error {
	return p.postCommandWithRetry(ctx, payload, out, requireAuth, true)
}

func (p *CASLCloudProvider) postCommandWithRetry(ctx context.Context, payload map[string]any, out any, requireAuth bool, allowRelogin bool) error {
	requestPayload := copyStringAnyMap(payload)
	if requireAuth {
		token, err := p.ensureToken(ctx)
		if err != nil {
			return err
		}
		if strings.TrimSpace(asString(requestPayload["token"])) == "" {
			requestPayload["token"] = token
		}
	}

	body, status, err := p.doJSONRequest(ctx, caslCommandPath, requestPayload)
	if err != nil {
		return err
	}

	if !statusIsOK(status.Status) {
		if requireAuth && allowRelogin && isCASLAuthError(status.Error) && p.canRelogin() {
			if reloginErr := p.refreshToken(ctx, true); reloginErr != nil {
				return fmt.Errorf("casl relogin failed: %w", reloginErr)
			}
			return p.postCommandWithRetry(ctx, payload, out, requireAuth, false)
		}
		return fmt.Errorf("casl command %q status=%q error=%q", asString(payload["type"]), status.Status, status.Error)
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("casl decode response: %w", err)
	}
	return nil
}

func (p *CASLCloudProvider) doJSONRequest(ctx context.Context, path string, payload any) ([]byte, caslStatusOnlyResponse, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl marshal payload: %w", err)
	}
	startedAt := time.Now()

	log.Debug().
		Str("method", http.MethodPost).
		Str("path", path).
		Msg("CASL HTTP request")
	logCASLHTTPBody(http.MethodPost, path, "request", requestBody)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(requestBody))
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl request failed: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, caslStatusOnlyResponse{}, fmt.Errorf("casl read response: %w", readErr)
	}

	log.Debug().
		Str("method", http.MethodPost).
		Str("path", path).
		Int("statusCode", resp.StatusCode).
		Dur("duration", time.Since(startedAt)).
		Msg("CASL HTTP response")
	logCASLHTTPBody(http.MethodPost, path, "response", body)

	var status caslStatusOnlyResponse
	_ = json.Unmarshal(body, &status)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		if strings.TrimSpace(status.Error) != "" {
			return nil, status, fmt.Errorf("casl http %d: %s", resp.StatusCode, status.Error)
		}
		return nil, status, fmt.Errorf("casl unexpected http status: %d", resp.StatusCode)
	}

	return body, status, nil
}

func (p *CASLCloudProvider) ensureToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	token := strings.TrimSpace(p.token)
	p.mu.RUnlock()
	if token != "" {
		return token, nil
	}

	if !p.canRelogin() {
		return "", errors.New("casl: token is empty and credentials are not configured")
	}

	if err := p.refreshToken(ctx, false); err != nil {
		return "", err
	}

	p.mu.RLock()
	token = strings.TrimSpace(p.token)
	p.mu.RUnlock()
	if token == "" {
		return "", errors.New("casl: login succeeded without token")
	}
	return token, nil
}

func (p *CASLCloudProvider) refreshToken(ctx context.Context, force bool) error {
	if !p.canRelogin() {
		return errors.New("casl: credentials are not configured")
	}

	p.authMu.Lock()
	defer p.authMu.Unlock()

	if force {
		p.mu.Lock()
		p.token = ""
		p.mu.Unlock()
	} else {
		p.mu.RLock()
		token := strings.TrimSpace(p.token)
		p.mu.RUnlock()
		if token != "" {
			return nil
		}
	}

	loginPultID := p.resolveLoginPultID(ctx)
	payload := map[string]any{"email": p.email, "pwd": p.pass, "pult_id": loginPultID, "captcha": ""}

	body, status, err := p.doJSONRequest(ctx, caslLoginPath, payload)
	if err != nil {
		return err
	}

	var resp caslLoginResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("casl decode login response: %w", err)
	}

	if !statusIsOK(resp.Status) && !statusIsOK(status.Status) {
		authErr := strings.TrimSpace(resp.Error)
		if authErr == "" {
			authErr = strings.TrimSpace(status.Error)
		}
		return fmt.Errorf("casl login status=%q error=%q", resp.Status, authErr)
	}
	if strings.TrimSpace(resp.Token) == "" {
		return errors.New("casl login did not return token")
	}

	p.mu.Lock()
	p.token = strings.TrimSpace(resp.Token)
	p.wsURL = strings.TrimSpace(resp.WSURL)
	p.userID = strings.TrimSpace(resp.UserID)
	if p.pultID <= 0 {
		if parsed := parseCASLID(loginPultID); parsed > 0 {
			p.pultID = int64(parsed)
		}
	}
	p.mu.Unlock()

	p.restartRealtimeStream()

	return nil
}

func (p *CASLCloudProvider) restartRealtimeStream() {
	p.realtimeMu.Lock()
	cancel := p.realtimeCancel
	p.realtimeCancel = nil
	p.realtimeRunning = false
	p.realtimeSubscribed = false
	p.realtimeMu.Unlock()

	if cancel != nil {
		cancel()
	}
	p.ensureRealtimeStream()
}

func (p *CASLCloudProvider) resolveLoginPultID(ctx context.Context) string {
	p.mu.RLock()
	if p.pultID > 0 {
		value := strconv.FormatInt(p.pultID, 10)
		p.mu.RUnlock()
		return value
	}
	p.mu.RUnlock()

	pults, err := p.readPultsPublic(ctx)
	if err == nil {
		for _, item := range pults {
			candidate := strings.TrimSpace(item.PultID)
			if candidate != "" {
				return candidate
			}
		}
	}

	return "1"
}

func (p *CASLCloudProvider) canRelogin() bool {
	return strings.TrimSpace(p.email) != "" && strings.TrimSpace(p.pass) != ""
}

func (p *CASLCloudProvider) hasAuthData() bool {
	p.mu.RLock()
	token := strings.TrimSpace(p.token)
	p.mu.RUnlock()
	return token != "" || p.canRelogin()
}

func (p *CASLCloudProvider) primaryObjectContext(ctx context.Context) (int, string) {
	records, err := p.loadObjects(ctx)
	if err == nil && len(records) > 0 {
		if p.pultID > 0 {
			for _, item := range records {
				if parseCASLID(item.ReactingPultID) == int(p.pultID) {
					obj := mapCASLGrdObjectToObject(item, nil)
					objectNum := preferredCASLObjectNumber(item.ObjID, item.Name, item.DeviceNumber.Int64())
					return obj.ID, formatCASLJournalObjectName(objectNum, obj.Name)
				}
			}
		}
		obj := mapCASLGrdObjectToObject(records[0], nil)
		objectNum := preferredCASLObjectNumber(records[0].ObjID, records[0].Name, records[0].DeviceNumber.Int64())
		return obj.ID, formatCASLJournalObjectName(objectNum, obj.Name)
	}

	pults, pErr := p.readPultsPublic(ctx)
	if pErr == nil && len(pults) > 0 {
		obj := mapCASLPultToObject(pults[0])
		return obj.ID, obj.Name
	}

	return 0, "CASL Cloud"
}

func mapCASLGrdObjectToObject(record caslGrdObject, device *caslDevice) models.Object {
	id := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))

	name := strings.TrimSpace(record.Name)
	if name == "" {
		name = "CASL Object #" + strings.TrimSpace(record.ObjID)
	}

	address := strings.TrimSpace(record.Address)
	if address == "" {
		address = formatCASLCoordinates(record.Lat, record.Long)
	}

	blocked := record.DeviceBlocked || strings.TrimSpace(record.BlockMessage.String()) != ""
	statusState := mapCASLObjectStatusState(record.Status, blocked)

	notes := strings.TrimSpace(record.Note)
	if notes == "" {
		notes = strings.TrimSpace(record.Description)
	}

	panelMark := ""
	if record.DeviceNumber.Int64() > 0 {
		panelMark = fmt.Sprintf("CASL #%d", record.DeviceNumber.Int64())
	}

	launchDate := ""
	if record.StartDate.Int64() > 0 {
		launchDate = time.UnixMilli(record.StartDate.Int64()).Format("02.01.2006")
	}

	deviceType := "—"
	sim1 := ""
	sim2 := ""
	if device != nil {
		if value := strings.TrimSpace(device.Type.String()); value != "" {
			deviceType = decodeCASLDeviceType(value)
		}
		sim1 = strings.TrimSpace(device.SIM1.String())
		sim2 = strings.TrimSpace(device.SIM2.String())
	}

	hasAssignment := len(normalizeContactIDs(record.InCharge, record.ManagerID)) > 0

	return models.Object{
		ID:             id,
		Name:           name,
		Address:        address,
		ContractNum:    strings.TrimSpace(record.Contract),
		Status:         statusState.Status,
		StatusText:     statusState.StatusText,
		AlarmState:     statusState.AlarmState,
		GuardState:     statusState.GuardState,
		TechAlarmState: statusState.TechAlarmState,
		IsConnState:    statusState.IsConnState,
		IsUnderGuard:   statusState.IsUnderGuard,
		IsConnOK:       statusState.IsConnState > 0,
		HasAssignment:  hasAssignment,
		SignalStrength: "н/д",
		DeviceType:     deviceType,
		PanelMark:      panelMark,
		SIM1:           sim1,
		SIM2:           sim2,
		ObjChan:        5,
		AutoTestHours:  24,
		Notes1:         notes,
		Location1:      address,
		LaunchDate:     launchDate,
		BlockedArmedOnOff: func() int16 {
			if blocked {
				return 1
			}
			return 0
		}(),
	}
}

func mapCASLPultToObject(item caslPult) models.Object {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.Nickname)
	}
	if name == "" {
		name = "CASL Pult"
	}

	id := mapCASLObjectID(item.PultID, item.Name, item.Nickname)

	address := ""
	if item.Lat != 0 || item.Lng != 0 {
		address = fmt.Sprintf("%.6f, %.6f", item.Lat, item.Lng)
	}

	return models.Object{
		ID:             id,
		Name:           name,
		Address:        address,
		ContractNum:    strings.TrimSpace(item.Nickname),
		Status:         models.StatusNormal,
		StatusText:     caslObjectStatusText,
		GuardState:     1,
		IsConnState:    1,
		IsUnderGuard:   true,
		IsConnOK:       true,
		HasAssignment:  true,
		SignalStrength: "н/д",
		DeviceType:     "CASL Pult",
		ObjChan:        5,
		AutoTestHours:  24,
	}
}

func (p *CASLCloudProvider) enrichCASLObjectWithDeviceMeta(ctx context.Context, obj *models.Object, hasDevice bool, device caslDevice) {
	if obj == nil || !hasDevice {
		return
	}

	rawType := strings.TrimSpace(device.Type.String())
	if rawType != "" {
		obj.DeviceType = p.resolveCASLDeviceTypeLabel(ctx, rawType)
	}

	deviceName := strings.TrimSpace(device.Name.String())
	if deviceName != "" {
		obj.Notes1 = deviceName
	}
}

func (p *CASLCloudProvider) resolveCASLDeviceTypeLabel(ctx context.Context, rawType string) string {
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return "—"
	}

	if value := p.lookupCASLDeviceTypeInDictionary(ctx, rawType); value != "" {
		return value
	}

	return decodeCASLDeviceType(rawType)
}

func (p *CASLCloudProvider) lookupCASLDeviceTypeInDictionary(ctx context.Context, rawType string) string {
	dict, ok := p.cachedDictionarySnapshot(ctx)
	if !ok || len(dict) == 0 {
		return ""
	}

	deviceTypes := extractCASLDeviceTypesMap(dict)
	if len(deviceTypes) == 0 {
		return ""
	}

	for _, key := range []string{rawType, strings.ToUpper(rawType), strings.ToLower(rawType)} {
		if text := strings.TrimSpace(deviceTypes[key]); text != "" {
			return text
		}
	}

	return ""
}

func (p *CASLCloudProvider) cachedDictionarySnapshot(ctx context.Context) (map[string]any, bool) {
	p.mu.RLock()
	cacheValid := len(p.cachedDictionary) > 0 && time.Since(p.cachedDictionaryAt) <= caslDictionaryTTL
	if cacheValid {
		snapshot := copyStringAnyMap(p.cachedDictionary)
		p.mu.RUnlock()
		return snapshot, true
	}
	p.mu.RUnlock()

	_ = p.loadDictionaryMap(ctx)

	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.cachedDictionary) == 0 {
		return nil, false
	}
	return copyStringAnyMap(p.cachedDictionary), true
}

func extractCASLDeviceTypesMap(value any) map[string]string {
	root, ok := value.(map[string]any)
	if !ok || len(root) == 0 {
		return nil
	}

	if raw, exists := root["device_types"]; exists {
		if mapped := flattenStringMap(raw); len(mapped) > 0 {
			return mapped
		}
	}

	if nestedRaw, exists := root["dictionary"]; exists {
		if nested, okNested := nestedRaw.(map[string]any); okNested {
			if raw, exists := nested["device_types"]; exists {
				if mapped := flattenStringMap(raw); len(mapped) > 0 {
					return mapped
				}
			}
		}
	}

	return nil
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
		ts := item.Time.Int64()
		eventTime := time.Now()
		if ts > 0 {
			eventTime = time.UnixMilli(ts).Local()
		}

		code := item.Code.String()
		if code == "" {
			code = "UNKNOWN"
		}

		zoneNumber := int(item.Number.Int64())
		contactID := strings.TrimSpace(item.ContactID.String())
		sourceType := strings.TrimSpace(item.Type)

		details := decodeCASLEventDescription(translator, dictMap, code, contactID, zoneNumber, deviceType)
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
		eventType := classifyCASLEventTypeWithContext(code, contactID, sourceType, details)

		if shouldAppendCASLLineDescription(code, contactID, details) {
			if lineName := strings.TrimSpace(lineNames[zoneNumber]); lineName != "" {
				details += " | Опис: " + lineName
			}
		}
		if sourceType != "" && !strings.EqualFold(sourceType, "ppk_event") {
			details += " | src=" + sourceType
		}

		result = append(result, models.Event{
			ID:         stableCASLEventID(record.ObjID, ts, code, idx),
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: zoneNumber,
			Details:    details,
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	return result
}

func classifyCASLEventType(code string) models.EventType {
	value := strings.ToUpper(strings.TrimSpace(code))
	valueLower := strings.ToLower(strings.TrimSpace(code))

	switch {
	case strings.Contains(value, "GRD_OBJ_NOTIF"):
		return models.EventBurglary
	case strings.Contains(value, "GRD_OBJ_MGR_CANCEL"), strings.Contains(value, "GRD_OBJ_FINISH"):
		return models.EventRestore
	case strings.Contains(value, "GRD_OBJ_PICK"), strings.Contains(value, "GRD_OBJ_ASS_MGR"), strings.Contains(value, "GRD_OBJ_"):
		return models.SystemEvent
	case strings.Contains(value, "PANIC"), strings.Contains(value, "COERCION"), strings.Contains(value, "ATTACK"), strings.Contains(value, "ALM_BTN_PRS"):
		return models.EventPanic
	case strings.Contains(valueLower, "тривожна кноп"), strings.Contains(valueLower, "кнопка тривог"), strings.Contains(valueLower, "напад"), strings.Contains(valueLower, "панік"):
		return models.EventPanic
	case strings.Contains(value, "MEDICAL"):
		return models.EventMedical
	case strings.Contains(valueLower, "медич"):
		return models.EventMedical
	case strings.Contains(value, "GAS_ALARM"), strings.Contains(value, "CO_GAS"), strings.Contains(value, "GAS_SUPERVISORY"):
		return models.EventGas
	case strings.Contains(valueLower, "газ"):
		return models.EventGas
	case strings.Contains(value, "BURGLARY"), strings.Contains(value, "INTRUSION"), strings.Contains(value, "BRUTFORS"), strings.Contains(value, "ZONE_ALM"), strings.Contains(value, "ALM_INNER_ZONE"):
		return models.EventBurglary
	case strings.Contains(valueLower, "проник"), strings.Contains(valueLower, "злом"), strings.Contains(valueLower, "охорон") && strings.Contains(valueLower, "тривог"):
		return models.EventBurglary
	case strings.Contains(value, "SABOTAGE"), strings.Contains(value, "TAMPER"), strings.Contains(value, "SENS_TAMP"), strings.Contains(value, "EXT_MOD_TAMP"), strings.Contains(value, "HUB_TAMP"):
		return models.EventTamper
	case strings.Contains(valueLower, "саботаж"), strings.Contains(valueLower, "тампер"):
		return models.EventTamper
	case strings.Contains(value, "FIRE"), strings.Contains(value, "SMOKE"), strings.Contains(value, "HEAT"):
		return models.EventFire
	case strings.Contains(valueLower, "пожеж"), strings.Contains(valueLower, "дим"), strings.Contains(valueLower, "тепл"):
		return models.EventFire
	case strings.Contains(value, "R402"),
		strings.Contains(value, "GROUP_ON"),
		strings.Contains(value, "GROUP_ON_USER"),
		strings.Contains(value, "ON_WITH_PPL"),
		strings.Contains(value, "ON_BFR_TIME"),
		strings.Contains(value, "ON_AFTR_TIME"),
		strings.Contains(value, "_ARMED"),
		value == "ARM":
		return models.EventArm
	case strings.Contains(valueLower, "взят"),
		strings.Contains(valueLower, "під охорон"),
		strings.Contains(valueLower, "взятие"),
		strings.Contains(valueLower, "постановк"):
		return models.EventArm
	case strings.Contains(value, "R401"),
		strings.Contains(value, "GROUP_OFF"),
		strings.Contains(value, "GROUP_OFF_USER"),
		strings.Contains(value, "OFF_WITH_PPL"),
		strings.Contains(value, "OFF_BFR_TIME"),
		strings.Contains(value, "OFF_AFTR_TIME"),
		strings.Contains(value, "_DISARM"),
		value == "DISARM":
		return models.EventDisarm
	case strings.Contains(valueLower, "знят"),
		strings.Contains(valueLower, "виключ"),
		strings.Contains(valueLower, "сняти"),
		strings.Contains(valueLower, "снятие"):
		return models.EventDisarm
	case strings.Contains(value, "ID_HOZ"),
		strings.Contains(value, "USER_ACCESS"),
		strings.Contains(valueLower, "ідентифікац"),
		strings.Contains(valueLower, "идентификац"),
		strings.Contains(valueLower, "користувач"),
		strings.Contains(valueLower, "пользовател"):
		return models.SystemEvent
	case value == "E627", value == "R627", value == "E628", value == "R628":
		return models.SystemEvent
	case strings.Contains(value, "UPD_START"), strings.Contains(value, "UPD_END"), strings.Contains(value, "FIRMWARE"),
		strings.Contains(valueLower, "оновлен"), strings.Contains(valueLower, "застосуван") && strings.Contains(valueLower, "налаштуван"):
		return models.SystemEvent
	case strings.Contains(value, "ALM_"),
		strings.Contains(value, "_ALARM"),
		strings.Contains(valueLower, "тривога"),
		strings.Contains(valueLower, "тревог"):
		return models.EventFault
	case strings.Contains(value, "ZONE_NORM"),
		strings.Contains(value, "NORM_"),
		strings.Contains(valueLower, "норма"),
		strings.Contains(valueLower, "віднов"),
		strings.Contains(valueLower, "восстанов"):
		return models.EventRestore
	case strings.Contains(value, "NO_CONN"), strings.Contains(value, "CONNECTION_LOST"), strings.Contains(value, "OFFLINE"), strings.Contains(value, "LOST"):
		return models.EventOffline
	case strings.Contains(valueLower, "нема зв"), strings.Contains(valueLower, "втрата зв"), strings.Contains(valueLower, "відсутн") && strings.Contains(valueLower, "зв"):
		return models.EventOffline
	case strings.Contains(value, "RECOVER"), strings.Contains(value, "RESTORE"),
		strings.HasPrefix(value, "OK_"), strings.Contains(value, "OK_220"), strings.Contains(value, "POWER_OK"),
		strings.HasSuffix(value, "_OK"), strings.HasPrefix(value, "R"):
		return models.EventRestore
	case strings.Contains(valueLower, "віднов"), strings.Contains(valueLower, "норма"):
		return models.EventRestore
	case (strings.Contains(value, "POWER") || strings.Contains(value, "NO_220") || strings.Contains(value, "MAIN_AC_LOSS")) &&
		!strings.Contains(value, "POWER_OK") &&
		!strings.Contains(value, "OK_220") &&
		!strings.HasSuffix(value, "_OK"):
		return models.EventPowerFail
	case strings.Contains(valueLower, "220") && strings.Contains(valueLower, "живлен") &&
		(strings.Contains(valueLower, "втра") || strings.Contains(valueLower, "пропаж")):
		return models.EventPowerFail
	case strings.Contains(value, "BATT"), strings.Contains(value, "BATTERY") && strings.Contains(value, "LOW"):
		return models.EventBatteryLow
	case strings.Contains(valueLower, "акб") && strings.Contains(valueLower, "розряд"):
		return models.EventBatteryLow
	case strings.Contains(value, "TEST"):
		return models.EventTest
	case strings.Contains(value, "POLL"), strings.Contains(value, "PING"), strings.Contains(value, "PONG"):
		return models.EventTest
	default:
		return models.EventFault
	}
}

func mapCASLTapeEventType(raw string) models.EventType {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "fire":
		return models.EventFire
	case "burglary":
		return models.EventBurglary
	case "panic":
		return models.EventPanic
	case "medical":
		return models.EventMedical
	case "gas":
		return models.EventGas
	case "tamper":
		return models.EventTamper
	case "fault":
		return models.EventFault
	case "restore":
		return models.EventRestore
	case "arm":
		return models.EventArm
	case "disarm":
		return models.EventDisarm
	case "test":
		return models.EventTest
	case "poll":
		return models.EventTest
	case "power_fail":
		return models.EventPowerFail
	case "power_ok":
		return models.EventPowerOK
	case "batt_low":
		return models.EventBatteryLow
	case "offline":
		return models.EventOffline
	case "online":
		return models.EventOnline
	case "system":
		return models.SystemEvent
	case "user_action":
		return models.SystemEvent
	case "ppk_action":
		return models.SystemEvent
	case "ppk_service":
		return models.SystemEvent
	case "system_event":
		return models.SystemEvent
	case "system_action":
		return models.SystemEvent
	case "m3_in":
		return models.SystemEvent
	case "mob_user_action":
		return models.SystemEvent
	default:
		return classifyCASLEventType(value)
	}
}

func (p *CASLCloudProvider) loadDictionaryMap(ctx context.Context) map[string]string {
	p.mu.RLock()
	if len(p.cachedDictionary) > 0 && time.Since(p.cachedDictionaryAt) <= caslDictionaryTTL {
		cached := flattenLocalizedDictionaryMap(p.cachedDictionary)
		p.mu.RUnlock()
		return cached
	}
	p.mu.RUnlock()

	dict, err := p.ReadDictionary(ctx)
	if err != nil || len(dict) == 0 {
		if err != nil {
			log.Debug().Err(err).Msg("CASL: не вдалося прочитати dictionary для розшифровки подій")
		}
		return nil
	}

	p.mu.Lock()
	p.cachedDictionary = copyStringAnyMap(dict)
	p.cachedDictionaryAt = time.Now()
	p.mu.Unlock()

	return flattenLocalizedDictionaryMap(dict)
}

func flattenLocalizedDictionaryMap(dict map[string]any) map[string]string {
	base := flattenStringMap(dict)
	uk := extractCASLDictionaryLanguageMap(dict, "uk")
	for key, value := range uk {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		base[key] = value
	}
	return base
}

func extractCASLDictionaryLanguageMap(dict map[string]any, lang string) map[string]string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if len(dict) == 0 || lang == "" {
		return nil
	}

	langCandidates := []string{lang, strings.ToUpper(lang), "uk-UA", "uk_ua", "ua", "UA"}
	resolveLangMap := func(node any) map[string]string {
		root, ok := node.(map[string]any)
		if !ok || len(root) == 0 {
			return nil
		}
		for _, key := range langCandidates {
			if nested, exists := root[key]; exists {
				flat := flattenStringMap(nested)
				if len(flat) > 0 {
					return flat
				}
			}
		}
		return nil
	}

	if nested, ok := dict["translate"]; ok {
		if out := resolveLangMap(nested); len(out) > 0 {
			return out
		}
	}
	if nestedRaw, ok := dict["dictionary"]; ok {
		if nested, okNested := nestedRaw.(map[string]any); okNested {
			if tr, exists := nested["translate"]; exists {
				if out := resolveLangMap(tr); len(out) > 0 {
					return out
				}
			}
		}
	}
	if out := resolveLangMap(dict); len(out) > 0 {
		return out
	}
	return nil
}

func (p *CASLCloudProvider) loadTranslatorMap(ctx context.Context, deviceType string) map[string]string {
	key := strings.TrimSpace(deviceType)
	if key == "" {
		return nil
	}

	p.mu.RLock()
	if !p.translatorDisabledUntil.IsZero() && time.Now().Before(p.translatorDisabledUntil) {
		p.mu.RUnlock()
		return nil
	}
	if cached, ok := p.cachedTranslators[key]; ok && time.Since(p.cachedTransAt[key]) <= caslTranslatorTTL {
		if len(cached) == 0 {
			p.mu.RUnlock()
			return nil
		}
		out := make(map[string]string, len(cached))
		for k, v := range cached {
			out[k] = v
		}
		p.mu.RUnlock()
		return out
	}
	p.mu.RUnlock()

	rawTranslator, err := p.GetMessageTranslatorByDeviceType(ctx, key)
	if err != nil {
		if isCASLWrongFormatErr(err) {
			// На частині інсталяцій CASL endpoint повертає WRONG_FORMAT, якщо передати device_type.
			// Пробуємо одноразово загальний виклик без device_type і витягаємо мапу по типу локально.
			rawAll, retryErr := p.GetMessageTranslatorByDeviceType(ctx, "")
			if retryErr == nil {
				flat := extractCASLTranslatorByType(rawAll, key)
				if len(flat) > 0 {
					p.mu.Lock()
					p.cachedTranslators[key] = flat
					p.cachedTransAt[key] = time.Now()
					p.mu.Unlock()

					out := make(map[string]string, len(flat))
					for k, v := range flat {
						out[k] = v
					}
					return out
				}
			}
		}

		p.mu.Lock()
		p.cachedTranslators[key] = map[string]string{}
		p.cachedTransAt[key] = time.Now()
		shouldLog := true
		if isCASLWrongFormatErr(err) {
			alreadyDisabled := !p.translatorDisabledUntil.IsZero() && time.Now().Before(p.translatorDisabledUntil)
			p.translatorDisabledUntil = time.Now().Add(caslTranslatorTTL)
			shouldLog = !alreadyDisabled
		}
		p.mu.Unlock()
		if shouldLog {
			log.Debug().Err(err).Str("deviceType", key).Msg("CASL: не вдалося отримати translator для типу пристрою")
		}
		return nil
	}

	flat := flattenCASLTranslatorMap(rawTranslator)
	if len(flat) == 0 {
		p.mu.Lock()
		p.cachedTranslators[key] = map[string]string{}
		p.cachedTransAt[key] = time.Now()
		p.mu.Unlock()
		return nil
	}

	p.mu.Lock()
	p.cachedTranslators[key] = flat
	p.cachedTransAt[key] = time.Now()
	p.mu.Unlock()

	out := make(map[string]string, len(flat))
	for k, v := range flat {
		out[k] = v
	}
	return out
}

func flattenStringMap(value any) map[string]string {
	result := make(map[string]string)

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for k := range typed {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				nested := typed[k]
				switch n := nested.(type) {
				case string:
					text := strings.TrimSpace(n)
					if text != "" {
						key := strings.TrimSpace(k)
						if key != "" {
							result[key] = text
						}
					}
				default:
					walk(n)
				}
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}

	walk(value)
	return result
}

func extractCASLTranslatorByType(raw any, deviceType string) map[string]string {
	flat := flattenCASLTranslatorMap(raw)
	if len(flat) == 0 {
		return nil
	}

	root, ok := raw.(map[string]any)
	if !ok {
		return flat
	}

	key := strings.TrimSpace(deviceType)
	if key == "" {
		return flat
	}

	candidates := []string{key, strings.ToUpper(key), strings.ToLower(key)}
	for _, candidate := range candidates {
		if nested, exists := root[candidate]; exists {
			if mapped := flattenCASLTranslatorMap(nested); len(mapped) > 0 {
				return mapped
			}
		}
	}

	return flat
}

func flattenCASLTranslatorMap(value any) map[string]string {
	result := make(map[string]string)

	setIfEmpty := func(key string, text string) {
		key = strings.TrimSpace(key)
		text = strings.TrimSpace(text)
		if key == "" || text == "" {
			return
		}
		if _, exists := result[key]; exists {
			return
		}
		result[key] = text
	}

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			if codes := extractCASLTranslatorCodes(typed); len(codes) > 0 {
				if text := extractCASLTranslatorText(typed); text != "" {
					for _, code := range codes {
						setIfEmpty(code, text)
					}
				}
			}

			for key, nested := range typed {
				candidate := strings.TrimSpace(key)
				if candidate == "" {
					continue
				}
				if looksLikeCASLTranslatorCode(candidate) {
					if text := extractCASLTranslatorText(nested); text != "" {
						setIfEmpty(candidate, text)
					}
				}
				walk(nested)
			}
		case []any:
			for _, nested := range typed {
				walk(nested)
			}
		}
	}

	walk(value)
	return result
}

func extractCASLTranslatorCode(entry map[string]any) string {
	codes := extractCASLTranslatorCodes(entry)
	if len(codes) > 0 {
		return codes[0]
	}
	return ""
}

func extractCASLTranslatorCodes(entry map[string]any) []string {
	base := ""
	for _, key := range []string{"contact_id", "contactId", "code", "event_code", "eventCode", "id", "key"} {
		if raw, ok := entry[key]; ok {
			code := strings.TrimSpace(asString(raw))
			if looksLikeCASLTranslatorCode(code) {
				base = code
				break
			}
		}
	}
	if base == "" {
		return nil
	}

	keys := []string{base}
	if isCASLTranslatorNumericCode(base) {
		if eventType := extractCASLTranslatorTypeEvent(entry); eventType != "" {
			keys = append([]string{eventType + base}, keys...)
		}
	}

	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func extractCASLTranslatorTypeEvent(entry map[string]any) string {
	for _, key := range []string{"typeEvent", "type_event", "eventType", "event_type"} {
		raw, ok := entry[key]
		if !ok {
			continue
		}
		value := strings.ToUpper(strings.TrimSpace(asString(raw)))
		if value == "" {
			continue
		}
		if len(value) == 1 && (value == "E" || value == "R" || value == "P") {
			return value
		}
		if len(value) > 1 && (value[0] == 'E' || value[0] == 'R' || value[0] == 'P') {
			isNumericTail := true
			for _, ch := range value[1:] {
				if ch < '0' || ch > '9' {
					isNumericTail = false
					break
				}
			}
			if isNumericTail {
				return string(value[0])
			}
		}
	}
	return ""
}

func isCASLTranslatorNumericCode(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func extractCASLTranslatorText(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case map[string]any:
		priority := []string{
			"msg", "message", "text", "description", "title", "template",
			"name", "label", "value", "lang_uk", "uk", "lang_ru", "ru", "lang_en", "en",
		}
		for _, key := range priority {
			if raw, ok := typed[key]; ok {
				if text := extractCASLTranslatorText(raw); text != "" {
					return text
				}
			}
		}

		for _, key := range []string{"lang", "langs", "translations", "translate"} {
			if raw, ok := typed[key]; ok {
				if text := extractCASLTranslatorText(raw); text != "" {
					return text
				}
			}
		}

		if len(typed) == 1 {
			for _, nested := range typed {
				if text := extractCASLTranslatorText(nested); text != "" {
					return text
				}
			}
		}
	}
	return ""
}

func looksLikeCASLTranslatorCode(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}

	upper := strings.ToUpper(key)
	if strings.HasPrefix(upper, "TYPE_DEVICE_") {
		return false
	}

	switch upper {
	case "MSG", "MESSAGE", "TEXT", "DESCRIPTION", "TITLE", "TEMPLATE", "NAME", "LABEL", "VALUE",
		"ISALARM", "ALARMCOLOR", "PRIORITY", "CODE", "EVENT_CODE", "CONTACT_ID", "LANG_UK", "LANG_RU", "LANG_EN",
		"UK", "RU", "EN", "STATUS", "DATA", "DICTIONARY", "TYPE":
		return false
	}

	if len(upper) >= 4 && (upper[0] == 'E' || upper[0] == 'R') {
		allDigits := true
		for _, ch := range upper[1:] {
			if ch < '0' || ch > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return true
		}
	}

	allDigits := len(upper) >= 3
	for _, ch := range upper {
		if ch < '0' || ch > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		return true
	}

	if strings.ContainsRune(upper, '_') {
		hasLetter := false
		for _, ch := range upper {
			if ch >= 'A' && ch <= 'Z' {
				hasLetter = true
				break
			}
		}
		return hasLetter
	}

	return false
}

var caslContactIDFallbackTemplates = map[string]string{
	"E110":  "Пожежна тривога",
	"R110":  "Відновлення після пожежної тривоги",
	"E120":  "Тривожна кнопка",
	"R120":  "Відновлення після тривожної кнопки",
	"E130":  "Тривога проникнення",
	"R130":  "Відновлення після тривоги проникнення",
	"E301":  "Втрата живлення 220В",
	"R301":  "Відновлення живлення 220В",
	"E302":  "Низький заряд АКБ",
	"R302":  "Відновлення АКБ",
	"E390":  "Не прийшло опитування за вказаний час",
	"R390":  "Відновлення опитування",
	"R401":  "Зняття групи № {number}",
	"R402":  "Взяття групи № {number}",
	"E627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"R627":  "Старт процесу оновлення чи застосування нових налаштувань",
	"E628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"R628":  "Завершення процесу оновлення чи застосування нових налаштувань",
	"61184": "Відповідь на опитування - норма шлейфа № {number}",
}

var caslMessageKeyFallbackTemplates = map[string]string{
	"GROUP_ON":        "Постановка групи {number}",
	"OO_GROUP_ON":     "Постановка групи {number}",
	"GROUP_OFF":       "Зняття групи № {number}",
	"OO_GROUP_OFF":    "Зняття групи № {number}",
	"LINE_BRK":        "Обрив шлейфа № {number}",
	"OO_LINE_BRK":     "Обрив шлейфа № {number}",
	"LINE_NORM":       "Норма шлейфа № {number}",
	"OO_LINE_NORM":    "Норма шлейфа № {number}",
	"LINE_KZ":         "Коротке замикання шлейфа № {number}",
	"OO_LINE_KZ":      "Коротке замикання шлейфа № {number}",
	"LINE_BAD":        "Несправність шлейфа № {number}",
	"OO_LINE_BAD":     "Несправність шлейфа № {number}",
	"ZONE_ALM":        "Тривога в зоні № {number}",
	"ZONE_NORM":       "Норма в зоні № {number}",
	"ALM_INNER_ZONE":  "Тривога внутрішньої зони № {number}",
	"NORM_INNER_ZONE": "Норма внутрішньої зони № {number}",
	"NORM_IO":         "Норма IO № {number}",
	"NO_220":          "Втрата живлення 220В",
	"OK_220":          "Відновлення живлення 220В",
	"PPK_NO_CONN":     "Немає зв'язку з ППК",
	"PPK_CONN_OK":     "Зв'язок з ППК відновлено",
	"ACC_BAD":         "Низький заряд АКБ",
	"ACC_OK":          "АКБ в нормі",
	"DOOR_OP":         "Відкриття корпусу/дверей",
	"DOOR_CL":         "Закриття корпусу/дверей",
	"CHECK_CONN":      "Перевірка зв'язку",
	"ENABLED":         "Прилад увімкнено",
	"DISABLED":        "Прилад вимкнено",
	"FULL_REBOOT":     "Повне перезавантаження ППК",
	"ID_HOZ":          "Ідентифікація користувача {number}",
	"PRIMUS":          "Ідентифікація користувача {number}",
	"UPD_START":       "Старт процесу оновлення чи застосування нових налаштувань",
	"UPD_END":         "Завершення процесу оновлення чи застосування нових налаштувань",
}

type caslProtocolModel int

const (
	caslProtocolModelOther caslProtocolModel = iota
	caslProtocolModelRcom
	caslProtocolModelSIA
	caslProtocolModelVBD4
	caslProtocolModelDozor
	caslProtocolModelD128
)

type caslDecodedEventCode struct {
	MessageKey string
	Number     int
	HasNumber  bool
}

func resolveCASLTemplate(source map[string]string, key string) string {
	if len(source) == 0 {
		return ""
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	candidates := []string{
		key,
		strings.ToUpper(key),
		strings.ToLower(key),
	}
	for _, candidate := range candidates {
		if value := strings.TrimSpace(source[candidate]); value != "" {
			return value
		}
	}
	return ""
}

func applyCASLNumberTemplate(template string, number int) string {
	if strings.TrimSpace(template) == "" {
		return ""
	}
	replacements := map[string]string{
		"{number}": strconv.Itoa(number),
		"{zone}":   strconv.Itoa(number),
		"{line}":   strconv.Itoa(number),
		"%number%": strconv.Itoa(number),
		"%zone%":   strconv.Itoa(number),
		"%line%":   strconv.Itoa(number),
	}
	out := template
	for from, to := range replacements {
		out = strings.ReplaceAll(out, from, to)
	}
	return strings.TrimSpace(out)
}

func caslTemplateHasNumberPlaceholder(template string) bool {
	template = strings.TrimSpace(template)
	if template == "" {
		return false
	}
	return strings.Contains(template, "{number}") ||
		strings.Contains(template, "{zone}") ||
		strings.Contains(template, "{line}") ||
		strings.Contains(template, "%number%") ||
		strings.Contains(template, "%zone%") ||
		strings.Contains(template, "%line%")
}

func caslMessageKeyNeedsNumberSuffix(key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	switch {
	case strings.Contains(key, "GROUP"),
		strings.Contains(key, "ZONE"),
		strings.Contains(key, "LINE"),
		strings.Contains(key, "ID_HOZ"):
		return true
	default:
		return false
	}
}

func finalizeCASLDecodedTemplate(template string, number int, messageKey string) string {
	out := applyCASLNumberTemplate(template, number)
	if out == "" {
		return ""
	}
	if number > 0 && caslMessageKeyNeedsNumberSuffix(messageKey) && !caslTemplateHasNumberPlaceholder(template) {
		return out + " № " + strconv.Itoa(number)
	}
	return out
}

func hasCyrillicChars(text string) bool {
	for _, r := range text {
		if (r >= 'А' && r <= 'я') || r == 'Ї' || r == 'ї' || r == 'Є' || r == 'є' || r == 'І' || r == 'і' || r == 'Ґ' || r == 'ґ' {
			return true
		}
	}
	return false
}

func shouldAppendCASLLineDescription(code string, contactID string, details string) bool {
	key := strings.ToUpper(strings.TrimSpace(code))
	if decoded, ok := decodeCASLProtocolCode(code, ""); ok {
		key = strings.ToUpper(strings.TrimSpace(decoded.MessageKey))
	}
	if key == "" {
		key = strings.ToUpper(strings.TrimSpace(contactID))
	}

	switch {
	case strings.Contains(key, "GROUP"),
		strings.Contains(key, "ID_HOZ"),
		strings.Contains(key, "USER_ACCESS"),
		strings.Contains(key, "PRIMUS"):
		return false
	}

	lowerDetails := strings.ToLower(strings.TrimSpace(details))
	switch {
	case strings.Contains(lowerDetails, "груп"),
		strings.Contains(lowerDetails, "користувач"),
		strings.Contains(lowerDetails, "ідентифікац"):
		return false
	}

	return true
}

func fallbackCASLContactIDTemplate(contactID string) string {
	value := strings.ToUpper(strings.TrimSpace(contactID))
	if len(value) < 4 {
		return ""
	}

	prefix := value[0]
	if prefix != 'E' && prefix != 'R' {
		return ""
	}

	for _, ch := range value[1:] {
		if ch < '0' || ch > '9' {
			return ""
		}
	}

	if prefix == 'R' {
		return "Відновлення ContactID " + value
	}
	return "Тривога ContactID " + value
}

func parseCASLCodeBytes(code string) (byte, byte, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, 0, false
	}
	value, err := strconv.ParseInt(code, 10, 64)
	if err != nil || value < 0 {
		return 0, 0, false
	}
	if value > 0xFFFF {
		value %= 0x10000
	}
	return byte((value >> 8) & 0xFF), byte(value & 0xFF), true
}

func caslProtocolModelFromDeviceType(deviceType string) caslProtocolModel {
	switch strings.TrimSpace(deviceType) {
	case "TYPE_DEVICE_Ajax_SIA", "TYPE_DEVICE_Bron_SIA":
		return caslProtocolModelSIA
	case "TYPE_DEVICE_Dunay_4_3", "TYPE_DEVICE_Dunay_4_3S", "TYPE_DEVICE_VBD4_ECOM", "TYPE_DEVICE_VBD_16":
		return caslProtocolModelVBD4
	case "TYPE_DEVICE_Dozor_4", "TYPE_DEVICE_Dozor_8", "TYPE_DEVICE_Dozor_8MG":
		return caslProtocolModelDozor
	case "TYPE_DEVICE_Dunay_16_32", "TYPE_DEVICE_Dunay_8_32", "TYPE_DEVICE_Dunay_PSPN_ECOM":
		return caslProtocolModelD128
	default:
		return caslProtocolModelRcom
	}
}

func decodedStatic(key string) (caslDecodedEventCode, bool) {
	if strings.TrimSpace(key) == "" {
		return caslDecodedEventCode{}, false
	}
	return caslDecodedEventCode{MessageKey: key}, true
}

func decodedWithOffset(key string, b2 byte, offset int) (caslDecodedEventCode, bool) {
	if strings.TrimSpace(key) == "" {
		return caslDecodedEventCode{}, false
	}
	return caslDecodedEventCode{
		MessageKey: key,
		Number:     int(b2) + offset,
		HasNumber:  true,
	}, true
}

func decodedWithSecondByte(key string, b2 byte) (caslDecodedEventCode, bool) {
	return decodedWithOffset(key, b2, 0)
}

func decodeCASLSystemCode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	switch b1 {
	case 0x00:
		switch b2 {
		case 0xB3:
			return decodedStatic("BAN_TIME")
		case 0xBD:
			return decodedStatic("REQUIRED_GROUP_ON")
		case 0x60:
			return decodedStatic("PPK_CONN_OK")
		case 0x66:
			return decodedStatic("SUSPICIOUS_ACTIVITY")
		case 0x67:
			return decodedStatic("SABOTAGE")
		}
	case 0x01:
		switch b2 {
		case 0x61:
			return decodedStatic("OO_NO_POLL")
		case 0x62:
			return decodedStatic("OO_NO_PING")
		}
	}
	return caslDecodedEventCode{}, false
}

func decodeCASLRcomSurgardCode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	if b1 == 0x3B {
		switch b2 {
		case 0x00:
			return decodedStatic("REP_FIRMW_4L")
		case 0x01:
			return decodedStatic("END_FIRMW_4L")
		case 0x02:
			return decodedStatic("REQ_REP_FIRMW_4L")
		case 0x03:
			return decodedStatic("REC_CONFIG_4L")
		case 0x04:
			return decodedStatic("END_CONFIG_4L")
		case 0x05:
			return decodedStatic("PPK_SIM_4L")
		case 0x06:
			return decodedStatic("PPK_IMEIL_4L")
		case 0x07:
			return decodedStatic("PPK_COORD_4L")
		case 0x08:
			return decodedStatic("PPK_CSQ_4L")
		case 0x09:
			return decodedStatic("CONTROL_4L")
		}
	}

	if b1 == 0x08 {
		switch b2 {
		case 0x27:
			return decodedWithOffset("ID_HOZ", b2, -0x0f)
		case 0x28:
			return decodedStatic("SET_INPUT_CONTROL")
		case 0x29:
			return decodedStatic("KEYPAD_PROGRAMMING")
		case 0x2A:
			return decodedStatic("PROGRAMMING_CP_USB")
		case 0x2B:
			return decodedStatic("PROGRAMMING_CP_INTERNET")
		case 0x2C:
			return decodedStatic("MANAGEMENT_FROM_DUNAY")
		case 0x2D:
			return decodedStatic("REMOTE_CONTROL")
		case 0x2E:
			return decodedStatic("KEYFOB_KEYBOARD")
		default:
			return decodedWithOffset("ID_HOZ", b2, -0x0f)
		}
	}

	switch b1 {
	case 0x00:
		switch b2 {
		case 0x02:
			return decodedStatic("CANNOT_AUTO_ARM")
		case 0x03:
			return decodedStatic("DEVICE_TEMPORARILY_DEACTIVATED")
		case 0x04:
			return decodedStatic("DEVICE_ACTIVE_AGAIN")
		case 0x05:
			return decodedStatic("TAMPER_ON")
		case 0x57:
			return decodedStatic("SERVER_CONNECTION_VIA_ETHERNET_LOST")
		case 0x58:
			return decodedStatic("SERVER_CONNECTION_VIA_ETHERNET_RESTORED")
		case 0x61:
			return decodedStatic("PPK_NO_CONN")
		case 0x63:
			return decodedStatic("PPK_BAD")
		case 0x64:
			return decodedStatic("ENABLED")
		case 0x65:
			return decodedStatic("DISABLED")
		case 0x68:
			return decodedStatic("NO_220")
		case 0x69:
			return decodedStatic("OK_220")
		case 0x6A:
			return decodedStatic("ACC_OK")
		case 0x6B:
			return decodedStatic("ACC_BAD")
		case 0x6C:
			return decodedStatic("DOOR_OP")
		case 0x6D:
			return decodedStatic("DOOR_CL")
		case 0x6E:
			return decodedStatic("SERVER_CONNECTION_VIA_CELLULAR_LOST")
		case 0x6F:
			return decodedStatic("SERVER_CONNECTION_VIA_CELLULAR_RESTORED")
		case 0x70:
			return decodedStatic("SERVER_CONNECTION_VIA_WI_FI_LOST")
		case 0x71:
			return decodedStatic("SERVER_CONNECTION_VIA_WI_FI_RESTORED")
		case 0x79:
			return decodedStatic("RING_DISCONNECTED")
		case 0x80:
			return decodedStatic("RING_CONNECTED")
		case 0xB9:
			return decodedStatic("FULL_REBOOT")
		}
	case 0x01:
		switch b2 {
		case 0x63:
			return decodedStatic("CHANGE_IP_OK")
		case 0x64:
			return decodedStatic("CHANGE_IP_FAIL")
		case 0x68:
			return decodedStatic("OO_NO_220")
		case 0x69:
			return decodedStatic("OO_OK_220")
		case 0x6A:
			return decodedStatic("OO_ACC_OK")
		case 0x6B:
			return decodedStatic("OO_ACC_BAD")
		case 0x6C:
			return decodedStatic("OO_DOOR_OP")
		case 0x6D:
			return decodedStatic("OO_DOOR_CL")
		}
	case 0x02:
		return decodedWithOffset("WL_ACC_OK", b2, 1)
	case 0x03:
		return decodedWithOffset("WL_ACC_BAD", b2, 1)
	case 0x04:
		return decodedWithOffset("WL_DOOR_CL", b2, 1)
	case 0x05:
		return decodedWithOffset("WL_DOOR_OP", b2, 1)
	case 0x06:
		return decodedWithOffset("WL_TROUBLE", b2, 1)
	case 0x07:
		return decodedWithOffset("WL_NORM", b2, 1)
	case 0x09:
		return decodedWithOffset("PRIMUS", b2, -0x0f)
	case 0x0A:
		return decodedWithOffset("ID_HOZ", b2, 0x10+1)
	case 0x0B:
		return decodedWithOffset("PRIMUS", b2, 0x10+1)
	case 0x0C:
		return decodedWithOffset("ID_HOZ", b2, 0x30+1)
	case 0x0D:
		return decodedWithOffset("PRIMUS", b2, 0x30+1)
	case 0x0E:
		return decodedWithOffset("ID_HOZ", b2, 0x50+1)
	case 0x0F:
		return decodedWithOffset("PRIMUS", b2, 0x50+1)
	case 0x30:
		return decodedWithOffset("AD_DOOR_OP", b2, -0x0f)
	case 0x31:
		return decodedWithOffset("OO_AD_DOOR_OP", b2, -0x0f)
	case 0x32:
		return decodedWithOffset("AD_DOOR_CL", b2, -0x0f)
	case 0x33:
		return decodedWithOffset("OO_AD_DOOR_CL", b2, -0x0f)
	case 0x34:
		return decodedWithOffset("AD_NO_CONN", b2, -0x0f)
	case 0x35:
		return decodedWithOffset("OO_AD_NO_CONN", b2, -0x0f)
	case 0x36:
		return decodedWithOffset("AD_CONN_OK", b2, -0x0f)
	case 0x37:
		return decodedWithOffset("OO_AD_CONN_OK", b2, -0x0f)
	case 0x38:
		return decodedWithOffset("AD_BAD_FOOD", b2, -0x0f)
	case 0x39:
		return decodedWithOffset("OO_ALM_AD_POWER", b2, -0x0f)
	case 0x3A:
		return decodedWithOffset("AD_FOOD_OK", b2, -0x0f)
	case 0x3B:
		return decodedWithOffset("OO_AD_POWER_OK", b2, -0x0f)
	case 0x3E:
		return decodedWithSecondByte("PPK_FW_VERSION", b2)
	case 0x3F:
		switch b2 {
		case 0x09, 0x8F:
			return decodedStatic("COERCION")
		case 0x10, 0x90:
			return decodedStatic("RESTART")
		case 0x11, 0x91:
			return decodedStatic("CHECK_CONN")
		case 0x12, 0x92:
			return decodedStatic("DECONCERV")
		case 0x13, 0x93:
			return decodedStatic("CONCERV")
		case 0x14, 0x94:
			return decodedStatic("EDIT_CONF")
		case 0x15, 0x95:
			return decodedStatic("ENABLED")
		case 0x16, 0x96:
			return decodedStatic("DISABLED")
		}
	case 0x40:
		return decodedWithOffset("GROUP_ON", b2, -0x0f)
	case 0x41:
		return decodedWithOffset("OO_GROUP_ON", b2, -0x0f)
	case 0x42:
		return decodedWithOffset("GROUP_ON", b2, 0x10+1)
	case 0x43:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x10+1)
	case 0x44:
		return decodedWithOffset("GROUP_ON", b2, 0x30+1)
	case 0x45:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x30+1)
	case 0x46:
		return decodedWithOffset("GROUP_ON", b2, 0x50+1)
	case 0x47:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x50+1)
	case 0x48:
		return decodedWithOffset("GROUP_OFF", b2, -0x0f)
	case 0x49:
		return decodedWithOffset("OO_GROUP_OFF", b2, -0x0f)
	case 0x4A:
		return decodedWithOffset("GROUP_OFF", b2, 0x10+1)
	case 0x4B:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x10+1)
	case 0x4C:
		return decodedWithOffset("GROUP_OFF", b2, 0x30+1)
	case 0x4D:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x30+1)
	case 0x4E:
		return decodedWithOffset("GROUP_OFF", b2, 0x50+1)
	case 0x4F:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x50+1)
	case 0x50:
		return decodedWithOffset("LINE_BRK", b2, -0x0f)
	case 0x51:
		return decodedWithOffset("OO_LINE_BRK", b2, -0x0f)
	case 0x52:
		return decodedWithOffset("LINE_BRK", b2, 17)
	case 0x53:
		return decodedWithOffset("OO_LINE_BRK", b2, 0x10+1)
	case 0x54:
		return decodedWithOffset("LINE_BRK", b2, 0x30+1)
	case 0x55:
		return decodedWithOffset("OO_LINE_BRK", b2, 0x30+1)
	case 0x56:
		return decodedWithOffset("LINE_BRK", b2, 81)
	case 0x57:
		return decodedWithOffset("OO_LINE_BRK", b2, 81)
	case 0x58:
		return decodedWithOffset("LINE_NORM", b2, -0x0f)
	case 0x59:
		return decodedWithOffset("OO_LINE_NORM", b2, -0x0f)
	case 0x5A:
		return decodedWithOffset("LINE_NORM", b2, 17)
	case 0x5B:
		return decodedWithOffset("OO_LINE_NORM", b2, 17)
	case 0x5C:
		return decodedWithOffset("LINE_NORM", b2, 0x30+1)
	case 0x5D:
		return decodedWithOffset("OO_LINE_NORM", b2, 0x30+1)
	case 0x5E:
		return decodedWithOffset("LINE_NORM", b2, 81)
	case 0x5F:
		return decodedWithOffset("OO_LINE_NORM", b2, 81)
	case 0x60:
		return decodedStatic("PPK_CONN_OK")
	case 0x61:
		return decodedStatic("PPK_NO_CONN")
	case 0x63:
		return decodedStatic("PPK_BAD")
	case 0x64:
		return decodedStatic("ENABL_PPK_OK")
	case 0x65:
		return decodedStatic("DISABL_PPK_OK")
	case 0x68:
		return decodedStatic("NO_220")
	case 0x69:
		return decodedStatic("OK_220")
	case 0x6A:
		return decodedStatic("ACC_OK")
	case 0x6B:
		return decodedStatic("ACC_BAD")
	case 0x6C:
		return decodedStatic("DOOR_OP")
	case 0x6D:
		return decodedStatic("DOOR_CL")
	case 0x6E:
		return decodedStatic("SABOTAGE")
	case 0x6F:
		return decodedStatic("ENABLED_DISABLED_ERROR")
	case 0x70:
		return decodedWithOffset("LINE_KZ", b2, -0x0f)
	case 0x71:
		return decodedWithOffset("OO_LINE_KZ", b2, -0x0f)
	case 0x72:
		return decodedWithOffset("LINE_KZ", b2, 0x10+1)
	case 0x73:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x10+1)
	case 0x74:
		return decodedWithOffset("LINE_KZ", b2, 0x30+1)
	case 0x75:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x30+1)
	case 0x76:
		return decodedWithOffset("LINE_KZ", b2, 0x50+1)
	case 0x77:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x50+1)
	case 0x78:
		return decodedWithOffset("LINE_BAD", b2, -0x0f)
	case 0x79:
		return decodedWithOffset("OO_LINE_BAD", b2, -0x0f)
	case 0x7A:
		return decodedWithOffset("LINE_BAD", b2, 0x10+1)
	case 0x7B:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x10+1)
	case 0x7C:
		return decodedWithOffset("LINE_BAD", b2, 0x30+1)
	case 0x7D:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x30+1)
	case 0x7E:
		return decodedWithOffset("LINE_BAD", b2, 0x50+1)
	case 0x7F:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x50+1)
	case 0x90:
		return decodedWithSecondByte("HIGH_TEMP_DETECTED", b2)
	case 0x91:
		return decodedWithSecondByte("TEMP_IS_OK", b2)
	case 0x92:
		return decodedWithSecondByte("LOW_TEMP_DETECTED", b2)
	case 0x93:
		return decodedWithSecondByte("TEMP_IS_OK_AFTER_LOW", b2)
	case 0x94:
		return decodedWithSecondByte("VIBRATION_DETECTED", b2)
	case 0x95:
		return decodedWithSecondByte("ZONE_MALFUNCTION", b2)
	case 0x96:
		return decodedWithSecondByte("ZONE_OK", b2)
	case 0x97:
		return decodedWithSecondByte("BOLT_LOCK_UNLOCKED", b2)
	case 0x98:
		return decodedWithSecondByte("BOLT_LOCK_LOCKED", b2)
	case 0xA0:
		return decodedWithSecondByte("SMOKE", b2)
	case 0xA1:
		return decodedWithSecondByte("HEAT", b2)
	case 0xA2:
		return decodedWithSecondByte("WATER", b2)
	case 0xA3:
		return decodedWithSecondByte("CO_GAS", b2)
	case 0xA4:
		return decodedWithSecondByte("BRUTFORS_CANCELLED", b2)
	case 0xA5:
		return decodedWithSecondByte("JAMMING", b2)
	case 0xA6:
		return decodedWithSecondByte("SENSOR_NO_CONN", b2)
	case 0xA7:
		return decodedWithSecondByte("AKSEL", b2)
	case 0xA8:
		return decodedWithSecondByte("BTTR_FAIL", b2)
	case 0xA9:
		return decodedWithSecondByte("HRDW_FAIL", b2)
	case 0xAA:
		return decodedWithSecondByte("DUST", b2)
	case 0xAB:
		return decodedWithSecondByte("FIRE_ALARM_FINISH", b2)
	case 0xAC:
		return decodedWithSecondByte("TMP_OK", b2)
	case 0xAD:
		return decodedWithSecondByte("GAS_ALARM", b2)
	case 0xAE:
		return decodedWithSecondByte("GAS_ALARM_FINISH", b2)
	case 0xAF:
		return decodedWithSecondByte("WATER_LEAK_FINISH", b2)
	case 0xB9:
		return decodedWithSecondByte("FULL_REBOOT", b2)
	case 0xD0:
		return decodedWithSecondByte("EMP_ON_TIME", b2)
	case 0xE0:
		return decodedWithSecondByte("NORM_24", b2)
	case 0xE1:
		return decodedWithSecondByte("ALM_IO", b2)
	case 0xE2:
		return decodedWithSecondByte("NORM_IO", b2)
	case 0xE5:
		return decodedWithSecondByte("GROUP_OFF_USER", b2)
	case 0xE6:
		return decodedWithSecondByte("GROUP_ON_USER", b2)
	case 0xE9:
		return decodedWithSecondByte("OFF_WITH_PPL", b2)
	case 0xEA:
		return decodedWithSecondByte("ON_WITH_PPL", b2)
	case 0xEF:
		return decodedWithSecondByte("EMP_OFF_TIME", b2)
	case 0xF0:
		return decodedWithSecondByte("STAYIN_HOME", b2)
	case 0xF1:
		return decodedWithSecondByte("OO_STAYIN_HOME", b2)
	case 0xF2:
		return decodedWithSecondByte("INGINEER_PL", b2)
	case 0xF3:
		return decodedWithSecondByte("ZONE_ALM", b2)
	case 0xF4:
		return decodedWithSecondByte("ALM_BTN_PRS", b2)
	case 0xF5:
		return decodedWithSecondByte("ALM_BTN_RLZ", b2)
	case 0xF6:
		return decodedWithSecondByte("ZONE_NORM", b2)
	case 0xF7:
		return decodedWithSecondByte("SENS_TAMP", b2)
	case 0xF8:
		return decodedWithSecondByte("SENS_TAMP_N", b2)
	case 0xF9:
		return decodedWithSecondByte("HUB_TAMP", b2)
	case 0xFA:
		return decodedWithSecondByte("HUB_TAMP_N", b2)
	case 0xFB:
		return decodedWithSecondByte("ALM_PERIM_ZONE", b2)
	case 0xFC:
		return decodedWithSecondByte("NORM_PERIM_ZONE", b2)
	case 0xFD:
		return decodedWithSecondByte("ALM_INNER_ZONE", b2)
	case 0xFE:
		return decodedWithSecondByte("NORM_INNER_ZONE", b2)
	case 0xFF:
		return decodedWithSecondByte("ALM_24_ZONE", b2)
	}

	return caslDecodedEventCode{}, false
}

func decodeCASLProtocolCode(code string, deviceType string) (caslDecodedEventCode, bool) {
	b1, b2, ok := parseCASLCodeBytes(code)
	if !ok {
		return caslDecodedEventCode{}, false
	}

	if decoded, ok := decodeCASLSystemCode(b1, b2); ok {
		return decoded, true
	}

	// Наразі використовуємо rcom/surgard як базовий декодер для всіх моделей.
	// Для SIA/VBD/Dozor/інших моделей він теж дає коректні результати для більшості
	// подій у CASL Cloud, а специфічні моделі можна додати окремими декодерами.
	_ = caslProtocolModelFromDeviceType(deviceType)
	return decodeCASLRcomSurgardCode(b1, b2)
}

func decodeCASLEventDescription(translator map[string]string, dictionary map[string]string, code string, contactID string, number int, deviceType ...string) string {
	code = strings.TrimSpace(code)
	contactID = strings.TrimSpace(contactID)
	resolvedNumber := number

	// Пріоритет 1: явні мапи по code.
	template := resolveCASLTemplate(translator, code)
	if template == "" {
		template = resolveCASLTemplate(dictionary, code)
	}
	fallbackTemplate := resolveCASLTemplate(caslMessageKeyFallbackTemplates, code)
	if fallbackTemplate != "" {
		// Для ключових message key використовуємо канонічні українські тексти.
		template = fallbackTemplate
	}
	if template == "" {
		template = resolveCASLTemplate(caslContactIDFallbackTemplates, code)
	}
	if template != "" {
		return applyCASLNumberTemplate(template, resolvedNumber)
	}

	// Пріоритет 2: байтовий декодер протоколу (code -> msg key -> template).
	rawDeviceType := ""
	if len(deviceType) > 0 {
		rawDeviceType = strings.TrimSpace(deviceType[0])
	}
	if decoded, ok := decodeCASLProtocolCode(code, rawDeviceType); ok {
		if resolvedNumber <= 0 && decoded.HasNumber {
			resolvedNumber = decoded.Number
		}
		template = resolveCASLTemplate(translator, decoded.MessageKey)
		if template == "" {
			template = resolveCASLTemplate(dictionary, decoded.MessageKey)
		}
		fallbackTemplate = resolveCASLTemplate(caslMessageKeyFallbackTemplates, decoded.MessageKey)
		if fallbackTemplate != "" {
			template = fallbackTemplate
		}
		if template == "" {
			template = resolveCASLTemplate(caslContactIDFallbackTemplates, decoded.MessageKey)
		}
		if template == "" {
			template = fallbackTemplate
		}
		if template == "" {
			template = strings.TrimSpace(decoded.MessageKey)
		}
		if template != "" {
			return finalizeCASLDecodedTemplate(template, resolvedNumber, decoded.MessageKey)
		}
	}

	// Пріоритет 3: fallback по contact_id.
	template = resolveCASLTemplate(translator, contactID)
	if template == "" {
		template = resolveCASLTemplate(dictionary, contactID)
	}
	if template == "" {
		template = resolveCASLTemplate(caslContactIDFallbackTemplates, contactID)
	}
	if template == "" {
		template = fallbackCASLContactIDTemplate(contactID)
	}
	if template == "" {
		return ""
	}
	return applyCASLNumberTemplate(template, resolvedNumber)
}

func buildCASLUserActionDetails(row CASLObjectEvent) string {
	action := strings.ToUpper(strings.TrimSpace(row.Action))
	if action == "" {
		action = strings.ToUpper(strings.TrimSpace(row.Code))
	}
	if action == "" {
		return ""
	}

	objectLabel := strings.TrimSpace(row.ObjName)
	if objectLabel == "" {
		if objID := strings.TrimSpace(row.ObjID); objID != "" {
			objectLabel = "Об'єкт #" + objID
		}
	}

	switch action {
	case "GRD_OBJ_NOTIF":
		parts := []string{"Нова тривога"}
		if objectLabel != "" {
			parts = append(parts, objectLabel)
		}
		return strings.Join(parts, " | ")
	case "GRD_OBJ_PICK":
		base := "Тривогу взято в роботу оператором"
		who := strings.TrimSpace(row.UserFIO)
		if who == "" {
			who = strings.TrimSpace(row.UserID)
		}
		if who != "" {
			return base + ": " + who
		}
		return base
	case "GRD_OBJ_ASS_MGR":
		base := "На тривогу призначено ГМР"
		mgrID := strings.TrimSpace(row.MgrID)
		if mgrID != "" {
			return base + " #" + mgrID
		}
		return base
	case "GRD_OBJ_MGR_CANCEL":
		base := "Тривогу скасовано оператором"
		mgrID := strings.TrimSpace(row.MgrID)
		if mgrID != "" {
			return base + " (ГМР #" + mgrID + ")"
		}
		return base
	default:
		return ""
	}
}

func buildCASLLineNameIndex(lines []caslDeviceLine) map[int]string {
	if len(lines) == 0 {
		return nil
	}

	index := make(map[int]string, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line.Name.String())
		if name == "" {
			continue
		}
		if number := int(line.ID.Int64()); number > 0 {
			index[number] = name
			continue
		}
		if number := int(line.Number.Int64()); number > 0 {
			index[number] = name
		}
	}
	return index
}

type caslEventContext struct {
	ObjectID   int
	ObjectNum  string
	ObjectName string
	DeviceType string
	Translator map[string]string
	LineNames  map[int]string
}

func (p *CASLCloudProvider) loadEventContextsByPPK(ctx context.Context, ppkFilter map[int64]struct{}) (map[int64]caslEventContext, error) {
	records, err := p.loadObjects(ctx)
	if err != nil {
		return nil, err
	}
	_, devicesErr := p.loadDevices(ctx)
	if devicesErr != nil {
		log.Debug().Err(devicesErr).Msg("CASL: read_device недоступний для event context")
	}

	contexts := make(map[int64]caslEventContext, len(records))
	for _, record := range records {
		ppkNum := record.DeviceNumber.Int64()
		if ppkNum <= 0 {
			continue
		}
		if len(ppkFilter) > 0 {
			if _, ok := ppkFilter[ppkNum]; !ok {
				continue
			}
		}

		objectNum := preferredCASLObjectNumber(record.ObjID, record.Name, ppkNum)
		ctxItem := caslEventContext{
			ObjectID:  mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(ppkNum, 10)),
			ObjectNum: objectNum,
		}
		ctxItem.ObjectName = strings.TrimSpace(record.Name)
		if ctxItem.ObjectName == "" {
			ctxItem.ObjectName = "Об'єкт #" + objectNum
		}
		ctxItem.ObjectName = formatCASLJournalObjectName(ctxItem.ObjectNum, ctxItem.ObjectName)

		device, hasDevice := p.resolveDeviceForObject(record)
		if hasDevice {
			ctxItem.DeviceType = strings.TrimSpace(device.Type.String())
			ctxItem.LineNames = buildCASLLineNameIndex(device.Lines)
			ctxItem.Translator = p.loadTranslatorMap(ctx, ctxItem.DeviceType)
		}
		contexts[ppkNum] = ctxItem
	}

	return contexts, nil
}

func classifyCASLEventTypeWithContext(code string, contactID string, sourceType string, details string) models.EventType {
	normalizedType := strings.TrimSpace(sourceType)
	if normalizedType != "" && !strings.EqualFold(normalizedType, "user_action") && !strings.EqualFold(normalizedType, "mob_user_action") {
		if mapped := mapCASLTapeEventType(normalizedType); mapped != models.EventFault || strings.EqualFold(normalizedType, "fault") {
			return mapped
		}
	}

	if byCode := classifyCASLEventType(code); byCode != models.EventFault {
		return byCode
	}

	if decoded, ok := decodeCASLProtocolCode(code, ""); ok {
		if byDecoded := classifyCASLEventType(decoded.MessageKey); byDecoded != models.EventFault {
			return byDecoded
		}
	}

	if byContact := classifyCASLEventType(contactID); byContact != models.EventFault {
		return byContact
	}

	if byDetails := classifyCASLEventType(details); byDetails != models.EventFault {
		return byDetails
	}

	return models.EventFault
}

func mapEventTypeToAlarmType(eventType models.EventType) (models.AlarmType, bool) {
	switch eventType {
	case models.EventFire:
		return models.AlarmFire, true
	case models.EventBurglary:
		return models.AlarmBurglary, true
	case models.EventPanic:
		return models.AlarmPanic, true
	case models.EventMedical:
		return models.AlarmMedical, true
	case models.EventGas:
		return models.AlarmGas, true
	case models.EventTamper:
		return models.AlarmTamper, true
	case models.EventPowerFail:
		return models.AlarmPowerFail, true
	case models.EventBatteryLow:
		return models.AlarmBatteryLow, true
	case models.EventOffline:
		return models.AlarmOffline, true
	case models.SystemEvent:
		return models.AlarmSystemEvent, true
	case models.EventFault:
		return models.AlarmFault, true
	default:
		return "", false
	}
}

func parseCASLAnyInt(value any) int {
	switch v := value.(type) {
	case nil:
		return 0
	case int:
		return v
	case int64:
		return int(v)
	case int32:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
		return 0
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return 0
		}
		if i, err := strconv.Atoi(text); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return int(f)
		}
		return 0
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" {
			return 0
		}
		if i, err := strconv.Atoi(text); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return int(f)
		}
		return 0
	}
}

func parseCASLAnyTime(value any) time.Time {
	parseEpoch := func(epoch int64) time.Time {
		if epoch == 0 {
			return time.Time{}
		}
		if epoch > 1_000_000_000_000 || epoch < -1_000_000_000_000 {
			return time.UnixMilli(epoch).Local()
		}
		if epoch > 1_000_000_000 || epoch < -1_000_000_000 {
			return time.Unix(epoch, 0).Local()
		}
		return time.Time{}
	}

	switch v := value.(type) {
	case nil:
		return time.Time{}
	case time.Time:
		return v.Local()
	case int64:
		return parseEpoch(v)
	case int:
		return parseEpoch(int64(v))
	case float64:
		return parseEpoch(int64(v))
	case float32:
		return parseEpoch(int64(v))
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return parseEpoch(i)
		}
		if f, err := v.Float64(); err == nil {
			return parseEpoch(int64(f))
		}
		return time.Time{}
	case string:
		text := strings.TrimSpace(v)
		if text == "" {
			return time.Time{}
		}
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.Local()
		}
		if parsed, err := time.Parse(time.RFC3339, text); err == nil {
			return parsed.Local()
		}
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return parseEpoch(i)
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return parseEpoch(int64(f))
		}
		return time.Time{}
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" {
			return time.Time{}
		}
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return parseEpoch(i)
		}
		if f, err := strconv.ParseFloat(text, 64); err == nil {
			return parseEpoch(int64(f))
		}
		return time.Time{}
	}
}

func mapCASLEventSC1(eventType models.EventType) int {
	switch eventType {
	case models.EventFire:
		return 1
	case models.EventBurglary:
		return 22
	case models.EventPanic:
		return 21
	case models.EventMedical:
		return 23
	case models.EventGas:
		return 24
	case models.EventTamper:
		return 25
	case models.EventRestore, models.EventPowerOK:
		return 5
	case models.EventArm:
		return 10
	case models.EventDisarm:
		return 14
	case models.EventOffline:
		return 12
	case models.EventTest, models.SystemEvent:
		return 6
	default:
		return 2
	}
}

type caslObjectStatusState struct {
	Status         models.ObjectStatus
	StatusText     string
	AlarmState     int64
	GuardState     int64
	TechAlarmState int64
	IsConnState    int64
	IsUnderGuard   bool
}

func mapCASLObjectStatus(statusRaw string, blocked bool) (models.ObjectStatus, string, bool) {
	state := mapCASLObjectStatusState(statusRaw, blocked)
	return state.Status, state.StatusText, state.IsUnderGuard
}

func mapCASLObjectStatusState(statusRaw string, blocked bool) caslObjectStatusState {
	if blocked {
		return caslObjectStatusState{
			Status:         models.StatusFault,
			StatusText:     "ЗАБЛОКОВАНО",
			GuardState:     0,
			TechAlarmState: 1,
			IsConnState:    1,
			IsUnderGuard:   false,
		}
	}

	statusText := strings.TrimSpace(statusRaw)
	if statusText == "" {
		return caslObjectStatusState{
			Status:       models.StatusNormal,
			StatusText:   caslObjectStatusText,
			GuardState:   1,
			IsConnState:  1,
			IsUnderGuard: true,
		}
	}

	lower := strings.ToLower(statusText)
	isOffline := strings.Contains(lower, "нема") ||
		strings.Contains(lower, "offline") ||
		strings.Contains(lower, "зв'язк") ||
		strings.Contains(lower, "без зв") ||
		strings.Contains(lower, "lost")
	isAlarm := strings.Contains(lower, "трив") ||
		strings.Contains(lower, "alarm") ||
		strings.Contains(lower, "пожеж")
	isFault := strings.Contains(lower, "несправ") ||
		strings.Contains(lower, "fault") ||
		strings.Contains(lower, "error") ||
		strings.Contains(lower, "problem")
	isDisarmed := strings.Contains(lower, "виключ") ||
		strings.Contains(lower, "знято") ||
		strings.Contains(lower, "disarm") ||
		strings.Contains(lower, "off")
	isArmed := strings.Contains(lower, "включ") ||
		strings.Contains(lower, "під охор") ||
		strings.Contains(lower, "armed") ||
		strings.Contains(lower, "on")

	state := caslObjectStatusState{
		Status:       models.StatusNormal,
		StatusText:   statusText,
		GuardState:   1,
		IsConnState:  1,
		IsUnderGuard: true,
	}

	if isDisarmed {
		state.GuardState = 0
		state.IsUnderGuard = false
	} else if isArmed {
		state.GuardState = 1
		state.IsUnderGuard = true
	}

	if isOffline {
		state.Status = models.StatusOffline
		state.IsConnState = 0
	}
	if isFault {
		state.Status = models.StatusFault
		state.TechAlarmState = 1
	}
	if isAlarm {
		state.Status = models.StatusFire
		state.AlarmState = 1
	}

	switch {
	case isAlarm, isFault, isOffline, isDisarmed, isArmed:
		return state
	default:
		return state
	}
}

func normalizeContactIDs(inCharge []string, managerID string) []string {
	seen := make(map[string]struct{}, len(inCharge)+1)
	ids := make([]string, 0, len(inCharge)+1)

	appendID := func(raw string) {
		value := strings.TrimSpace(raw)
		if value == "" {
			return
		}
		if _, exists := seen[value]; exists {
			return
		}
		seen[value] = struct{}{}
		ids = append(ids, value)
	}

	for _, userID := range inCharge {
		appendID(userID)
	}
	appendID(managerID)

	return ids
}

func mapCASLDeviceGroupsToObjectGroups(rawGroups any, rooms []caslRoom) []models.ObjectGroup {
	candidates := collectCASLGroupCandidates(rawGroups)
	if len(candidates) == 0 {
		return nil
	}

	roomNames := make(map[string]string, len(rooms))
	for _, room := range rooms {
		roomID := strings.TrimSpace(room.RoomID)
		roomName := strings.TrimSpace(room.Name)
		if roomID != "" && roomName != "" {
			roomNames[roomID] = roomName
		}
	}

	groups := make([]models.ObjectGroup, 0, len(candidates))
	for idx, candidate := range candidates {
		group := models.ObjectGroup{
			Number: parseCASLID(candidate.key),
		}
		if group.Number <= 0 {
			group.Number = extractGroupNumber(candidate.value, idx+1)
		}

		applyGroupValue(&group, candidate.value)
		if group.StateText == "" {
			if group.Armed {
				group.StateText = "ПІД ОХОРОНОЮ"
			} else {
				group.StateText = "ЗНЯТО"
			}
		}

		if group.RoomName == "" {
			group.RoomName = roomNames[group.RoomID]
		}
		groups = append(groups, group)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Number == groups[j].Number {
			return groups[i].RoomName < groups[j].RoomName
		}
		return groups[i].Number < groups[j].Number
	})

	return groups
}

type caslGroupCandidate struct {
	key   string
	value any
}

func collectCASLGroupCandidates(raw any) []caslGroupCandidate {
	result := make([]caslGroupCandidate, 0, 8)
	collectCASLGroupCandidatesRecursive("", raw, 0, &result)
	return result
}

func collectCASLGroupCandidatesRecursive(keyHint string, raw any, depth int, out *[]caslGroupCandidate) {
	if out == nil || depth > 8 || raw == nil {
		return
	}

	switch typed := raw.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return
		}

		if isCASLGroupPayloadMap(typed) {
			*out = append(*out, caslGroupCandidate{key: keyHint, value: typed})
			return
		}

		hasNumericChildren := false
		for key, value := range typed {
			if parseCASLID(key) > 0 {
				hasNumericChildren = true
				collectCASLGroupCandidatesRecursive(key, value, depth+1, out)
			}
		}
		if hasNumericChildren {
			return
		}

		for _, wrapperKey := range []string{"groups", "group", "items", "list", "data", "result", "values"} {
			if nested, ok := typed[wrapperKey]; ok {
				collectCASLGroupCandidatesRecursive(keyHint, nested, depth+1, out)
				return
			}
		}

		hasNested := false
		for key, value := range typed {
			switch value.(type) {
			case map[string]any, []any:
				hasNested = true
				collectCASLGroupCandidatesRecursive(key, value, depth+1, out)
			}
		}
		if hasNested {
			return
		}

		*out = append(*out, caslGroupCandidate{key: keyHint, value: typed})
	case []any:
		for idx, item := range typed {
			collectCASLGroupCandidatesRecursive(strconv.Itoa(idx+1), item, depth+1, out)
		}
	default:
		if strings.TrimSpace(keyHint) == "" {
			return
		}
		*out = append(*out, caslGroupCandidate{key: keyHint, value: raw})
	}
}

func isCASLGroupPayloadMap(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}

	for _, key := range []string{
		"group",
		"group_id",
		"group_number",
		"number",
		"id",
		"state",
		"status",
		"group_state",
		"groupStatus",
		"is_on",
		"is_armed",
		"armed",
		"guard",
		"on",
		"group_on",
		"room",
		"room_id",
		"room_name",
		"name_room",
	} {
		if _, ok := payload[key]; ok {
			return true
		}
	}

	return false
}

func applyGroupValue(group *models.ObjectGroup, value any) {
	if group == nil {
		return
	}

	switch typed := value.(type) {
	case bool:
		group.Armed = typed
	case string:
		setGroupStateFromString(group, typed)
	case float64:
		setGroupStateFromInt(group, int64(typed))
	case int:
		setGroupStateFromInt(group, int64(typed))
	case int64:
		setGroupStateFromInt(group, typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			setGroupStateFromInt(group, parsed)
		} else if parsedF, errF := typed.Float64(); errF == nil {
			setGroupStateFromInt(group, int64(parsedF))
		}
	case map[string]any:
		applyGroupMap(group, typed)
	case []any:
		if len(typed) > 0 {
			applyGroupValue(group, typed[0])
		}
	default:
		group.StateText = strings.TrimSpace(asString(typed))
	}
}

func applyGroupMap(group *models.ObjectGroup, payload map[string]any) {
	if group == nil {
		return
	}

	if n := extractGroupNumber(payload, group.Number); n > 0 {
		group.Number = n
	}

	if roomName := strings.TrimSpace(asString(payload["room_name"])); roomName != "" {
		group.RoomName = roomName
	}
	if roomName := strings.TrimSpace(asString(payload["name_room"])); roomName != "" {
		group.RoomName = roomName
	}
	if roomID := strings.TrimSpace(asString(payload["room_id"])); roomID != "" {
		group.RoomID = roomID
	}
	if roomID := strings.TrimSpace(asString(payload["roomId"])); roomID != "" {
		group.RoomID = roomID
	}

	if room, ok := payload["room"].(map[string]any); ok {
		if roomName := strings.TrimSpace(asString(room["name"])); roomName != "" {
			group.RoomName = roomName
		}
		if roomID := strings.TrimSpace(asString(room["room_id"])); roomID != "" {
			group.RoomID = roomID
		}
		if roomID := strings.TrimSpace(asString(room["id"])); roomID != "" && group.RoomID == "" {
			group.RoomID = roomID
		}
	}

	for _, key := range []string{"is_on", "is_armed", "armed", "guard", "on", "group_on"} {
		if raw, ok := payload[key]; ok {
			if armed, known := boolFromAny(raw); known {
				group.Armed = armed
				if armed {
					group.StateText = "ПІД ОХОРОНОЮ"
				} else {
					group.StateText = "ЗНЯТО"
				}
				return
			}
		}
	}

	for _, key := range []string{"state", "status", "group_state", "groupStatus"} {
		if raw, ok := payload[key]; ok {
			switch v := raw.(type) {
			case string:
				setGroupStateFromString(group, v)
				return
			case map[string]any:
				applyGroupMap(group, v)
				return
			default:
				if armed, known := boolFromAny(v); known {
					group.Armed = armed
					if armed {
						group.StateText = "ПІД ОХОРОНОЮ"
					} else {
						group.StateText = "ЗНЯТО"
					}
					return
				}
			}
		}
	}
}

func setGroupStateFromInt(group *models.ObjectGroup, value int64) {
	if group == nil {
		return
	}
	group.Armed = value > 0
	if group.Armed {
		group.StateText = "ПІД ОХОРОНОЮ"
	} else {
		group.StateText = "ЗНЯТО"
	}
}

func setGroupStateFromString(group *models.ObjectGroup, raw string) {
	if group == nil {
		return
	}

	value := strings.TrimSpace(raw)
	if value == "" {
		return
	}

	lower := strings.ToLower(value)
	switch {
	case strings.Contains(lower, "on"),
		strings.Contains(lower, "group_on"),
		strings.Contains(lower, "guard"),
		strings.Contains(lower, "arm"),
		strings.Contains(lower, "взят"),
		strings.Contains(lower, "включ"),
		strings.Contains(lower, "під охор"):
		group.Armed = true
		group.StateText = "ПІД ОХОРОНОЮ"
	case strings.Contains(lower, "off"),
		strings.Contains(lower, "group_off"),
		strings.Contains(lower, "disarm"),
		strings.Contains(lower, "знят"),
		strings.Contains(lower, "виключ"):
		group.Armed = false
		group.StateText = "ЗНЯТО"
	default:
		group.StateText = value
	}
}

func extractGroupNumber(value any, fallback int) int {
	switch typed := value.(type) {
	case int:
		if typed > 0 {
			return typed
		}
	case int64:
		if typed > 0 {
			return int(typed)
		}
	case float64:
		if typed > 0 {
			return int(typed)
		}
	case json.Number:
		if parsed, err := typed.Int64(); err == nil && parsed > 0 {
			return int(parsed)
		}
	case map[string]any:
		for _, key := range []string{"number", "group_number", "group", "id", "group_id", "groupId", "groupNum"} {
			if raw, ok := typed[key]; ok {
				if parsed := parseCASLID(asString(raw)); parsed > 0 {
					return parsed
				}
				if parsed := parseCASLAnyInt(raw); parsed > 0 {
					return parsed
				}
			}
		}
	case string:
		if parsed := parseCASLID(typed); parsed > 0 {
			return parsed
		}
	}

	if fallback > 0 {
		return fallback
	}
	return 1
}

func boolFromAny(value any) (bool, bool) {
	switch typed := value.(type) {
	case bool:
		return typed, true
	case int:
		return typed > 0, true
	case int64:
		return typed > 0, true
	case float64:
		return typed > 0, true
	case string:
		raw := strings.TrimSpace(strings.ToLower(typed))
		switch raw {
		case "1", "true", "on", "armed", "guard", "group_on", "взято":
			return true, true
		case "0", "false", "off", "disarmed", "not_guard", "group_off", "знято":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func formatCASLCoordinates(lat, lng string) string {
	lat = strings.TrimSpace(lat)
	lng = strings.TrimSpace(lng)
	if lat == "" || lng == "" {
		return ""
	}
	return lat + ", " + lng
}

func parseCASLID(raw string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func mapCASLObjectID(parts ...string) int {
	base := 0
	if len(parts) > 0 {
		base = parseCASLID(parts[0])
	}
	if base == 0 {
		base = stableCASLID(parts...)
	}
	return caslObjectIDNamespaceStart + (base % caslObjectIDNamespaceSize)
}

func preferredCASLObjectNumber(rawObjID string, name string, ppkNum int64) string {
	if ppkNum > 0 {
		return strconv.FormatInt(ppkNum, 10)
	}

	if fromName := extractLeadingCASLNumber(name); fromName != "" {
		return fromName
	}

	return strings.TrimSpace(rawObjID)
}

func extractLeadingCASLNumber(name string) string {
	value := strings.TrimSpace(name)
	if value == "" {
		return ""
	}

	start := 0
	for start < len(value) && value[start] == ' ' {
		start++
	}
	end := start
	for end < len(value) && value[end] >= '0' && value[end] <= '9' {
		end++
	}
	if end == start {
		return ""
	}
	return strings.TrimSpace(value[start:end])
}

func formatCASLJournalObjectName(objNum string, name string) string {
	number := strings.TrimSpace(objNum)
	title := strings.TrimSpace(name)
	if number == "" {
		return title
	}
	if title == "" {
		return "Об'єкт #" + number
	}

	prefixes := []string{
		number + " |",
		number + " ",
		"Об'єкт #" + number,
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(title, prefix) {
			if strings.HasPrefix(title, number+" | ") {
				rest := strings.TrimSpace(strings.TrimPrefix(title, number+" | "))
				if strings.HasPrefix(rest, number+" |") {
					rest = strings.TrimSpace(strings.TrimPrefix(rest, number+" |"))
					if rest != "" {
						return number + " | " + rest
					}
				}
				if strings.HasPrefix(rest, number+" ") {
					rest = strings.TrimSpace(strings.TrimPrefix(rest, number+" "))
					if rest != "" {
						return number + " | " + rest
					}
				}
			}
			return title
		}
	}

	return number + " | " + title
}

var caslDeviceTypeDisplayNames = map[string]string{
	"TYPE_DEVICE_CASL":                    "CASL",
	"TYPE_DEVICE_Dunay_8L":                "Дунай-8L",
	"TYPE_DEVICE_Dunay_16L":               "Дунай-16L",
	"TYPE_DEVICE_Dunay_4L":                "Дунай-4L",
	"TYPE_DEVICE_Lun":                     "Лунь",
	"TYPE_DEVICE_Ajax":                    "Ajax",
	"TYPE_DEVICE_Ajax_SIA":                "Ajax(SIA)",
	"TYPE_DEVICE_Bron_SIA":                "Bron(SIA)",
	"TYPE_DEVICE_CASL_PLUS":               "CASL+",
	"TYPE_DEVICE_Dozor_4":                 "Дозор-4",
	"TYPE_DEVICE_Dozor_8":                 "Дозор-8",
	"TYPE_DEVICE_Dozor_8MG":               "Дозор-8MG",
	"TYPE_DEVICE_Dunay_8_32":              "Дунай-8/32",
	"TYPE_DEVICE_Dunay_16_32":             "Дунай-16/32",
	"TYPE_DEVICE_Dunay_4_3":               "Дунай-4.3",
	"TYPE_DEVICE_Dunay_4_3S":              "Дунай-4.3.1S",
	"TYPE_DEVICE_Dunay_8(16)32_Dunay_G1R": "128 + G1R",
	"TYPE_DEVICE_Dunay_STK":               "Дунай-СТК",
	"TYPE_DEVICE_Dunay_4.2":               "4.2 + G1R",
	"TYPE_DEVICE_VBDb_2":                  "ВБД6-2 + G1R",
	"TYPE_DEVICE_VBD4":                    "ВБД4 + G1R",
	"TYPE_DEVICE_Dunay_PSPN":              "ПСПН (R.COM)",
	"TYPE_DEVICE_Dunay_PSPN_ECOM":         "ПСПН (ECOM)",
	"TYPE_DEVICE_VBD4_ECOM":               "ВБД4",
	"TYPE_DEVICE_VBD_16":                  "ВБД6-16",
}

func decodeCASLDeviceType(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "—"
	}
	if translated, ok := caslDeviceTypeDisplayNames[value]; ok {
		return translated
	}
	return value
}

func selectCASLDevice(ok bool, device caslDevice) *caslDevice {
	if !ok {
		return nil
	}
	value := device
	return &value
}

func normalizeCASLAlarmState(raw int64) int64 {
	if raw == 0 {
		return 0
	}
	return 1
}

func isCASLObjectID(id int) bool {
	return id >= caslObjectIDNamespaceStart && id <= caslObjectIDNamespaceEnd
}

func stableCASLID(parts ...string) int {
	h := fnv.New32a()
	for _, part := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(part)))
		_, _ = h.Write([]byte{0})
	}
	id := int(h.Sum32() & 0x7fffffff)
	if id == 0 {
		return 1
	}
	return id
}

func stableCASLEventID(objID string, ts int64, code string, index int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(objID)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.FormatInt(ts, 10)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(code)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(index)))

	value := int(h.Sum32() & 0x7fffffff)
	if value == 0 {
		return nextCASLEventID()
	}
	return value
}

func normalizeCASLBaseURL(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		value = caslDefaultBaseURL
	}
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	return strings.TrimRight(value, "/")
}

func nextCASLEventID() int {
	return int(time.Now().UnixMilli() & 0x7fffffff)
}

func statusIsOK(status string) bool {
	value := strings.ToLower(strings.TrimSpace(status))
	return value == "" || value == "ok"
}

func isCASLAuthError(raw string) bool {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return false
	}
	if value == "WRONG_FORMAT" {
		return true
	}
	return strings.Contains(value, "TOKEN") || strings.Contains(value, "AUTH") || strings.Contains(value, "UNAUTHORIZED")
}

func isCASLWrongFormatErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), "WRONG_FORMAT")
}

func isCASLUnknownTagErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToUpper(err.Error()), "UNKNOWN_TAG")
}

func copyStringAnyMap(source map[string]any) map[string]any {
	if source == nil {
		return map[string]any{}
	}
	target := make(map[string]any, len(source))
	for key, value := range source {
		target[key] = value
	}
	return target
}

func caslBodyForDebugLog(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}

	body := append([]byte(nil), raw...)
	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		maskCASLSensitiveFields(decoded)
		if normalized, marshalErr := json.Marshal(decoded); marshalErr == nil {
			var pretty bytes.Buffer
			if indentErr := json.Indent(&pretty, normalized, "", "  "); indentErr == nil {
				body = pretty.Bytes()
			} else {
				body = normalized
			}
		}
	}

	text := strings.TrimSpace(string(body))
	if len(text) <= caslDebugBodyLimit {
		return text
	}
	return text[:caslDebugBodyLimit] + "...(truncated)"
}

func logCASLHTTPBody(method string, path string, kind string, body []byte) {
	formatted := caslBodyForDebugLog(body)
	if strings.TrimSpace(formatted) == "" {
		return
	}

	log.Debug().
		Str("method", method).
		Str("path", path).
		Msgf("CASL HTTP %s body:\n%s", kind, formatted)
}

func maskCASLSensitiveFields(value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			if isCASLSensitiveKey(key) {
				typed[key] = "***"
				continue
			}
			maskCASLSensitiveFields(nested)
		}
	case []any:
		for idx := range typed {
			maskCASLSensitiveFields(typed[idx])
		}
	}
}

func isCASLSensitiveKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "token", "pwd", "password", "pass", "authorization":
		return true
	default:
		return false
	}
}

func asString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatInt(int64(v), 10)
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

type caslInt64 int64

func (v caslInt64) Int64() int64 { return int64(v) }

func (v *caslInt64) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}

	if strings.HasPrefix(raw, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return nil
		}
		value = strings.TrimSpace(value)
		if value == "" {
			*v = 0
			return nil
		}
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			*v = caslInt64(i)
			return nil
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			*v = caslInt64(int64(f))
			return nil
		}
		*v = 0
		return nil
	}

	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		*v = caslInt64(i)
		return nil
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		*v = caslInt64(int64(f))
		return nil
	}

	*v = 0
	return nil
}

type caslText string

func (v caslText) String() string { return strings.TrimSpace(string(v)) }

func (v *caslText) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = caslText(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*v = caslText(number.String())
		return nil
	}

	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		if boolean {
			*v = "true"
		} else {
			*v = "false"
		}
		return nil
	}

	*v = caslText(strings.Trim(raw, "\""))
	return nil
}

type caslPult struct {
	PultID   string   `json:"pult_id"`
	Name     string   `json:"name"`
	Nickname string   `json:"nickname"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Zoom     int      `json:"zoom"`
	Users    []string `json:"users"`
}

type caslRoom struct {
	RoomID      string     `json:"room_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RTSP        string     `json:"rtsp"`
	Users       []caslUser `json:"users"`
}

type caslGrdObject struct {
	ObjID          string     `json:"obj_id"`
	Name           string     `json:"name"`
	Address        string     `json:"address"`
	Lat            string     `json:"lat"`
	Long           string     `json:"long"`
	Description    string     `json:"description"`
	ReactingPultID string     `json:"reacting_pult_id"`
	Contract       string     `json:"contract"`
	Note           string     `json:"note"`
	StartDate      caslInt64  `json:"start_date"`
	Status         string     `json:"status"`
	ObjectType     string     `json:"object_type"`
	DeviceNumber   caslInt64  `json:"device_number"`
	DeviceBlocked  bool       `json:"device_blocked"`
	DeviceID       caslInt64  `json:"device_id"`
	BlockMessage   caslText   `json:"block_message"`
	TimeUnblock    caslText   `json:"time_unblock"`
	ManagerID      string     `json:"manager_id"`
	Manager        caslUser   `json:"manager"`
	InCharge       []string   `json:"in_charge"`
	Rooms          []caslRoom `json:"rooms"`
}

type caslDevice struct {
	DeviceID caslText         `json:"device_id"`
	ObjID    caslText         `json:"obj_id"`
	Number   caslInt64        `json:"number"`
	Name     caslText         `json:"name"`
	Type     caslText         `json:"type"`
	SIM1     caslText         `json:"sim1"`
	SIM2     caslText         `json:"sim2"`
	Lines    []caslDeviceLine `json:"lines"`
}

type caslConnectionRecord struct {
	GuardedObject caslGrdObject
	Device        caslDevice
}

func (r *caslConnectionRecord) UnmarshalJSON(data []byte) error {
	type rawConnection struct {
		GuardedObject    json.RawMessage `json:"guardedObject"`
		GuardedObjectAlt json.RawMessage `json:"guarded_object"`
		Device           json.RawMessage `json:"device"`
	}

	var raw rawConnection
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	guardedObjectRaw := raw.GuardedObject
	if len(guardedObjectRaw) == 0 {
		guardedObjectRaw = raw.GuardedObjectAlt
	}
	if len(guardedObjectRaw) == 0 {
		guardedObjectRaw = data
	}
	_ = json.Unmarshal(guardedObjectRaw, &r.GuardedObject)

	if len(raw.Device) > 0 {
		_ = json.Unmarshal(raw.Device, &r.Device)
	}

	normalizeCASLObjectRecord(&r.GuardedObject, r.Device)
	if strings.TrimSpace(r.Device.ObjID.String()) == "" && strings.TrimSpace(r.GuardedObject.ObjID) != "" {
		r.Device.ObjID = caslText(r.GuardedObject.ObjID)
	}
	return nil
}

func (r caslConnectionRecord) hasPayload() bool {
	if strings.TrimSpace(r.GuardedObject.ObjID) != "" || strings.TrimSpace(r.GuardedObject.Name) != "" {
		return true
	}
	if strings.TrimSpace(r.Device.DeviceID.String()) != "" || r.Device.Number.Int64() > 0 {
		return true
	}
	return false
}

func (d *caslDevice) UnmarshalJSON(data []byte) error {
	type rawDevice struct {
		DeviceID caslText        `json:"device_id"`
		ObjID    caslText        `json:"obj_id"`
		Number   caslInt64       `json:"number"`
		Name     caslText        `json:"name"`
		Type     caslText        `json:"type"`
		SIM1     caslText        `json:"sim1"`
		SIM2     caslText        `json:"sim2"`
		Lines    json.RawMessage `json:"lines"`
	}

	var raw rawDevice
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.DeviceID = raw.DeviceID
	d.ObjID = raw.ObjID
	d.Number = raw.Number
	d.Name = raw.Name
	d.Type = raw.Type
	d.SIM1 = raw.SIM1
	d.SIM2 = raw.SIM2
	d.Lines = decodeCASLDeviceLines(raw.Lines)
	return nil
}

type caslDeviceLine struct {
	ID     caslInt64 `json:"id"`
	Name   caslText  `json:"name"`
	Number caslInt64 `json:"number"`
	Type   caslText  `json:"type"`
}

func decodeCASLDeviceLines(raw json.RawMessage) []caslDeviceLine {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		return nil
	}

	if body[0] == '[' {
		var lines []caslDeviceLine
		if err := json.Unmarshal(body, &lines); err == nil {
			return lines
		}
		return nil
	}

	if body[0] != '{' {
		return nil
	}

	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		return nil
	}
	if len(source) == 0 {
		return nil
	}

	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]caslDeviceLine, 0, len(keys))
	for _, key := range keys {
		line, ok := decodeCASLDeviceLineFromAny(source[key], key)
		if !ok {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}

func decodeCASLDeviceLineFromAny(value any, fallbackKey string) (caslDeviceLine, bool) {
	var line caslDeviceLine
	fallbackNum := parseCASLID(fallbackKey)
	if fallbackNum > 0 {
		line.Number = caslInt64(fallbackNum)
		line.ID = caslInt64(fallbackNum)
	}

	switch typed := value.(type) {
	case string:
		line.Name = caslText(strings.TrimSpace(typed))
		return line, strings.TrimSpace(line.Name.String()) != ""
	case map[string]any:
		if encoded, err := json.Marshal(typed); err == nil {
			_ = json.Unmarshal(encoded, &line)
		}
		if line.ID.Int64() <= 0 && fallbackNum > 0 {
			line.ID = caslInt64(fallbackNum)
		}
		if line.Number.Int64() <= 0 && fallbackNum > 0 {
			line.Number = caslInt64(fallbackNum)
		}
		if line.Name.String() == "" {
			if text := strings.TrimSpace(asString(typed["description"])); text != "" {
				line.Name = caslText(text)
			}
		}
		return line, line.Name.String() != "" || line.ID.Int64() > 0 || line.Number.Int64() > 0
	default:
		return line, false
	}
}

type caslPhoneNumber struct {
	Active bool   `json:"active"`
	Number string `json:"number"`
}

type caslUser struct {
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	LastName     string            `json:"last_name"`
	FirstName    string            `json:"first_name"`
	MiddleName   string            `json:"middle_name"`
	Role         string            `json:"role"`
	Tag          caslText          `json:"tag"`
	PhoneNumbers []caslPhoneNumber `json:"phone_numbers"`
}

func (u caslUser) FullName() string {
	parts := []string{strings.TrimSpace(u.LastName), strings.TrimSpace(u.FirstName), strings.TrimSpace(u.MiddleName)}
	filtered := make([]string, 0, 3)
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return "Користувач #" + strings.TrimSpace(u.UserID)
	}
	return strings.Join(filtered, " ")
}

func (u caslUser) PrimaryPhone() string {
	for _, phone := range u.PhoneNumbers {
		if phone.Active && strings.TrimSpace(phone.Number) != "" {
			return strings.TrimSpace(phone.Number)
		}
	}
	for _, phone := range u.PhoneNumbers {
		if strings.TrimSpace(phone.Number) != "" {
			return strings.TrimSpace(phone.Number)
		}
	}
	if strings.TrimSpace(u.Tag.String()) != "" {
		return strings.TrimSpace(u.Tag.String())
	}
	return ""
}

type caslObjectEvent struct {
	PPKNum    caslInt64 `json:"ppk_num"`
	DeviceID  caslText  `json:"device_id"`
	ObjID     caslText  `json:"obj_id"`
	ObjName   caslText  `json:"obj_name"`
	ObjAddr   caslText  `json:"obj_address"`
	Action    caslText  `json:"action"`
	AlarmType caslText  `json:"alarm_type"`
	MgrID     caslText  `json:"mgr_id"`
	UserID    caslText  `json:"user_id"`
	UserFIO   caslText  `json:"user_fio"`
	Time      caslInt64 `json:"time"`
	Code      caslText  `json:"code"`
	Type      string    `json:"type"`
	Number    caslInt64 `json:"number"`
	ContactID caslText  `json:"contact_id"`
	HozUserID caslText  `json:"hoz_user_id"`
}

type caslDeviceState struct {
	Power        caslInt64 `json:"power"`
	Accum        caslInt64 `json:"accum"`
	Door         caslInt64 `json:"door"`
	Online       caslInt64 `json:"online"`
	LastPingDate caslInt64 `json:"lastPingDate"`
	Lines        any       `json:"lines"`
	Groups       any       `json:"groups"`
	Adapters     any       `json:"adapters"`
}

type caslStatsAlarmsData struct {
	DeviceID            string    `json:"device_id"`
	ObjectID            string    `json:"obj_id"`
	ResponseFrequencies caslInt64 `json:"responseFrequencies"`
	CommunicQuality     caslInt64 `json:"communicQuality"`
	PowerFailure        caslInt64 `json:"powerFailure"`
	Criminogenicity     caslInt64 `json:"criminogenicity"`
	CustomWins          caslInt64 `json:"customWins"`
}

type caslLoginResponse struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	FIO    string `json:"fio"`
	Token  string `json:"token"`
	WSURL  string `json:"ws_url"`
	Error  string `json:"error"`
}

type caslReadPultResponse struct {
	Status string     `json:"status"`
	Data   []caslPult `json:"data"`
	Error  string     `json:"error"`
}

type caslReadGrdObjectResponse struct {
	Status string          `json:"status"`
	Data   []caslGrdObject `json:"data"`
	Error  string          `json:"error"`
}

type caslReadUserResponse struct {
	Status string     `json:"status"`
	Data   []caslUser `json:"data"`
	Error  string     `json:"error"`
}

type caslReadDeviceResponse struct {
	Status string       `json:"status"`
	Data   []caslDevice `json:"data"`
	Error  string       `json:"error"`
}

type caslReadEventsByIDResponse struct {
	Status string            `json:"status"`
	Data   []caslObjectEvent `json:"data"`
	Events []caslObjectEvent `json:"events"`
	Error  string            `json:"error"`
}

type caslReadDeviceStateResponse struct {
	Status string          `json:"status"`
	State  caslDeviceState `json:"state"`
	Error  string          `json:"error"`
}

type caslGetStatisticResponse struct {
	Status string              `json:"status"`
	Data   caslStatsAlarmsData `json:"data"`
	Error  string              `json:"error"`
}

type caslBasketResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error"`
}

type caslStatusOnlyResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}
