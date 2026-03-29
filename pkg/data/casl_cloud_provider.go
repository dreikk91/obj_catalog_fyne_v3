package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

const (
	caslCommandPath    = "/command"
	caslLoginPath      = "/login"
	caslDefaultBaseURL = "http://127.0.0.1:50003"

	caslHTTPTimeout      = 12 * time.Second
	caslObjectsCacheTTL  = 20 * time.Second
	caslUsersCacheTTL    = 5 * time.Minute
	caslObjectEventsTTL  = 10 * time.Second
	caslObjectEventsSpan = 7 * 24 * time.Hour
	caslStatsSpan        = 30 * 24 * time.Hour

	caslMaxCachedEvents = 2000
	caslReadLimit       = 100000

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

	cachedUsers   map[string]caslUser
	cachedUsersAt time.Time

	cachedObjectEvents   map[int][]models.Event
	cachedObjectEventsAt map[int]time.Time

	cachedEvents    []models.Event
	lastBasketCount int
	hasBasketCount  bool
}

func NewCASLCloudProvider(baseURL string, token string, pultID int64, credentials ...string) *CASLCloudProvider {
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
		cachedUsers:          make(map[string]caslUser),
		cachedObjectEvents:   make(map[int][]models.Event),
		cachedObjectEventsAt: make(map[int]time.Time),
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

	objects := make([]models.Object, 0, len(records))
	for _, record := range records {
		objects = append(objects, mapCASLGrdObjectToObject(record))
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

	obj := mapCASLGrdObjectToObject(record)
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
	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	tapeEvents, tapeErr := p.readGeneralTapeAsEvents(ctx)
	if tapeErr == nil && len(tapeEvents) > 0 {
		p.mu.Lock()
		p.cachedEvents = append([]models.Event(nil), tapeEvents...)
		p.mu.Unlock()
		return tapeEvents
	}
	if tapeErr != nil {
		log.Debug().Err(tapeErr).Msg("CASL: get_general_tape_objects недоступний, fallback на basket-модель")
	}

	return p.getEventsFromBasketFallback(ctx)
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

		eventTime := parseCASLAnyTime(row["time"])
		if eventTime.IsZero() {
			eventTime = time.Now()
		}

		eventType := mapCASLTapeEventType(asString(row["event_type"]))
		details := strings.TrimSpace(asString(row["description"]))
		if details == "" {
			reason := strings.TrimSpace(asString(row["reasonAlarm"]))
			if reason != "" {
				details = "Причина: " + reason
			}
		}

		eventID := parseCASLAnyInt(row["event_id"])
		if eventID <= 0 {
			eventID = stableCASLEventID(rawObjID, eventTime.UnixMilli(), asString(row["event_type"]), idx)
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

	events := mapCASLObjectEvents(record, rawEvents)
	sortEvents(events)

	p.mu.Lock()
	p.cachedObjectEvents[internalID] = append([]models.Event(nil), events...)
	p.cachedObjectEventsAt[internalID] = now
	p.mu.Unlock()

	return events
}

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	basketCount, err := p.readBasketCount(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося прочитати тривоги")
		return nil
	}
	if basketCount <= 0 {
		return nil
	}

	objectID, objectName := p.primaryObjectContext(ctx)
	if objectName == "" {
		objectName = "CASL Cloud"
	}

	return []models.Alarm{{
		ID:         objectID,
		ObjectID:   objectID,
		ObjectName: objectName,
		Address:    "CASL Cloud API",
		Time:       time.Now(),
		Details:    fmt.Sprintf("У кошику CASL Cloud: %d активних тривог", basketCount),
		Type:       models.AlarmFault,
		SC1:        2,
	}}
}

func (p *CASLCloudProvider) ProcessAlarm(id string, user string, note string) {
	log.Warn().Str("alarmID", id).Str("user", user).Msg("CASL: ProcessAlarm не підтримується API інтеграцією")
}

func (p *CASLCloudProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	internalID, ok := parseObjectID(objectID)
	if !ok {
		return "CASL Cloud", "н/д", time.Time{}, time.Time{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return "CASL Cloud", "н/д", time.Time{}, time.Time{}
	}

	state, stateErr := p.readDeviceState(ctx, record)
	stats, statsErr := p.readStatsAlarms(ctx, record)

	signalParts := []string{"CASL"}
	testParts := make([]string, 0, 4)

	if stateErr == nil {
		signalParts = append(signalParts,
			fmt.Sprintf("online=%d", state.Online.Int64()),
			fmt.Sprintf("power=%d", state.Power.Int64()),
			fmt.Sprintf("accum=%d", state.Accum.Int64()),
		)
		if state.LastPingDate.Int64() > 0 {
			lastMsg = time.UnixMilli(state.LastPingDate.Int64()).Local()
			lastTest = lastMsg
		}
	}

	if statsErr == nil {
		testParts = append(testParts,
			fmt.Sprintf("freq=%d", stats.ResponseFrequencies.Int64()),
			fmt.Sprintf("quality=%d", stats.CommunicQuality.Int64()),
			fmt.Sprintf("power=%d", stats.PowerFailure.Int64()),
			fmt.Sprintf("custom=%d", stats.CustomWins.Int64()),
		)
		if stats.ResponseFrequencies.Int64() > 0 {
			signalParts = append(signalParts, fmt.Sprintf("alarms=%d", stats.ResponseFrequencies.Int64()))
		}
	}

	if stateErr != nil && statsErr != nil {
		log.Debug().Err(stateErr).Msg("CASL: не вдалося отримати read_device_state")
		log.Debug().Err(statsErr).Msg("CASL: не вдалося отримати stats_alarms")
		signalParts = append(signalParts, "н/д")
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
	ctx, cancel := context.WithTimeout(context.Background(), caslHTTPTimeout)
	defer cancel()

	count, err := p.readBasketCount(ctx)
	if err != nil {
		return 0, err
	}
	return int64(count), nil
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
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cachedObjects = append([]caslGrdObject(nil), records...)
	p.cachedObjectsAt = time.Now()
	p.objectByInternalID = make(map[int]caslGrdObject, len(records))
	for _, record := range records {
		internalID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
		p.objectByInternalID[internalID] = record
	}
	p.mu.Unlock()

	return append([]caslGrdObject(nil), records...), nil
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

	users, err := p.readUsers(ctx)
	if err != nil {
		return nil, err
	}

	index := make(map[string]caslUser, len(users))
	for _, user := range users {
		userID := strings.TrimSpace(user.UserID)
		if userID == "" {
			continue
		}
		index[userID] = user
	}

	p.mu.Lock()
	p.cachedUsers = index
	p.cachedUsersAt = time.Now()
	p.mu.Unlock()

	copied := make(map[string]caslUser, len(index))
	for key, value := range index {
		copied[key] = value
	}
	return copied, nil
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

	return nil
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
					obj := mapCASLGrdObjectToObject(item)
					return obj.ID, obj.Name
				}
			}
		}
		obj := mapCASLGrdObjectToObject(records[0])
		return obj.ID, obj.Name
	}

	pults, pErr := p.readPultsPublic(ctx)
	if pErr == nil && len(pults) > 0 {
		obj := mapCASLPultToObject(pults[0])
		return obj.ID, obj.Name
	}

	return 0, "CASL Cloud"
}

func mapCASLGrdObjectToObject(record caslGrdObject) models.Object {
	id := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))

	name := strings.TrimSpace(record.Name)
	if name == "" {
		name = "CASL Object #" + strings.TrimSpace(record.ObjID)
	}

	address := strings.TrimSpace(record.Address)
	if address == "" {
		address = formatCASLCoordinates(record.Lat, record.Long)
	}

	objectStatus, statusText, isUnderGuard := mapCASLObjectStatus(record.Status, record.DeviceBlocked)
	guardState := int64(0)
	if isUnderGuard {
		guardState = 1
	}

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

	return models.Object{
		ID:             id,
		Name:           name,
		Address:        address,
		ContractNum:    strings.TrimSpace(record.Contract),
		Status:         objectStatus,
		StatusText:     statusText,
		GuardState:     guardState,
		IsConnState:    1,
		IsUnderGuard:   isUnderGuard,
		IsConnOK:       true,
		SignalStrength: "CASL Cloud",
		DeviceType:     strings.TrimSpace(record.ObjectType),
		PanelMark:      panelMark,
		ObjChan:        5,
		AutoTestHours:  24,
		Notes1:         notes,
		Location1:      address,
		LaunchDate:     launchDate,
		BlockedArmedOnOff: func() int16 {
			if record.DeviceBlocked {
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
		SignalStrength: "CASL Cloud",
		DeviceType:     "CASL Pult",
		ObjChan:        5,
		AutoTestHours:  24,
	}
}

func mapCASLObjectEvents(record caslGrdObject, raw []caslObjectEvent) []models.Event {
	if len(raw) == 0 {
		return nil
	}

	result := make([]models.Event, 0, len(raw))
	objectID := mapCASLObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))
	objectName := strings.TrimSpace(record.Name)

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

		eventType := classifyCASLEventType(code)
		zoneNumber := int(item.Number.Int64())

		detailsParts := []string{code}
		if contactID := strings.TrimSpace(item.ContactID.String()); contactID != "" {
			detailsParts = append(detailsParts, "contact="+contactID)
		}
		if sourceType := strings.TrimSpace(item.Type); sourceType != "" {
			detailsParts = append(detailsParts, "src="+sourceType)
		}

		result = append(result, models.Event{
			ID:         stableCASLEventID(record.ObjID, ts, code, idx),
			Time:       eventTime,
			ObjectID:   objectID,
			ObjectName: objectName,
			Type:       eventType,
			ZoneNumber: zoneNumber,
			Details:    strings.Join(detailsParts, " | "),
			SC1:        mapCASLEventSC1(eventType),
		})
	}

	return result
}

func classifyCASLEventType(code string) models.EventType {
	value := strings.ToUpper(strings.TrimSpace(code))

	switch {
	case strings.Contains(value, "FIRE"), strings.Contains(value, "SMOKE"), strings.Contains(value, "HEAT"):
		return models.EventFire
	case strings.Contains(value, "GROUP_ON"), strings.Contains(value, "ARM"), strings.HasPrefix(value, "ON_"):
		return models.EventArm
	case strings.Contains(value, "GROUP_OFF"), strings.Contains(value, "DISARM"), strings.HasPrefix(value, "OFF_"):
		return models.EventDisarm
	case strings.Contains(value, "POWER"), strings.Contains(value, "NO_220"), strings.Contains(value, "MAIN_AC_LOSS"):
		return models.EventPowerFail
	case strings.Contains(value, "RECOVER"), strings.Contains(value, "RESTORE"), strings.HasPrefix(value, "OK_"), strings.HasSuffix(value, "_OK"):
		return models.EventRestore
	case strings.Contains(value, "TEST"):
		return models.EventTest
	case strings.Contains(value, "NO_CONN"), strings.Contains(value, "CONNECTION_LOST"), strings.Contains(value, "OFFLINE"), strings.Contains(value, "LOST"):
		return models.EventOffline
	default:
		return models.EventFault
	}
}

func mapCASLTapeEventType(raw string) models.EventType {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "fire":
		return models.EventFire
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
	default:
		return classifyCASLEventType(value)
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

func mapCASLObjectStatus(statusRaw string, blocked bool) (models.ObjectStatus, string, bool) {
	if blocked {
		return models.StatusFault, "ЗАБЛОКОВАНО", false
	}

	statusText := strings.TrimSpace(statusRaw)
	if statusText == "" {
		return models.StatusNormal, caslObjectStatusText, true
	}

	lower := strings.ToLower(statusText)
	switch {
	case strings.Contains(lower, "включ"):
		return models.StatusNormal, statusText, true
	case strings.Contains(lower, "виключ"), strings.Contains(lower, "знято"):
		return models.StatusNormal, statusText, false
	case strings.Contains(lower, "нема"), strings.Contains(lower, "offline"), strings.Contains(lower, "зв'язк"):
		return models.StatusOffline, statusText, false
	case strings.Contains(lower, "трив"), strings.Contains(lower, "alarm"), strings.Contains(lower, "несправ"):
		return models.StatusFault, statusText, false
	default:
		return models.StatusNormal, statusText, true
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
	RoomID      string `json:"room_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RTSP        string `json:"rtsp"`
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
	InCharge       []string   `json:"in_charge"`
	Rooms          []caslRoom `json:"rooms"`
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
	Time      caslInt64 `json:"time"`
	Code      caslText  `json:"code"`
	Type      string    `json:"type"`
	Number    caslInt64 `json:"number"`
	ContactID caslText  `json:"contact_id"`
	HozUserID caslText  `json:"hoz_user_id"`
}

type caslDeviceState struct {
	Power        caslInt64      `json:"power"`
	Accum        caslInt64      `json:"accum"`
	Door         caslInt64      `json:"door"`
	Online       caslInt64      `json:"online"`
	LastPingDate caslInt64      `json:"lastPingDate"`
	Lines        map[string]any `json:"lines"`
	Groups       map[string]any `json:"groups"`
	Adapters     map[string]any `json:"adapters"`
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
