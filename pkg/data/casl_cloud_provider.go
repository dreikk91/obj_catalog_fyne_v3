package data

import (
	"context"
	"fmt"
	"obj_catalog_fyne_v3/pkg/data/casl"
	"obj_catalog_fyne_v3/pkg/data/casl/protocol"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// CASLCloudProvider реалізує DataProvider для CASL Cloud API.
type CASLCloudProvider struct {
	client   *casl.APIClient
	mapper   *casl.Mapper
	realtime *casl.RealtimeService

	mu sync.RWMutex

	cachedObjects      []casl.GrdObject
	cachedObjectsAt    time.Time
	objectByInternalID map[int]casl.GrdObject
	deviceByDeviceID   map[string]casl.Device
	deviceByObjectID   map[string]casl.Device
	deviceByNumber     map[int64]casl.Device
	cachedDevicesAt    time.Time

	cachedUsers   map[string]casl.User
	cachedUsersAt time.Time

	cachedEvents    []models.Event
	eventsRevision  int64

	cachedDictionary   map[string]any
	cachedDictionaryAt time.Time

	cachedTranslators map[string]map[string]string

	realtimeAlarmByObjID map[string]models.Alarm
}

func NewCASLCloudProvider(baseURL string, token string, pultID int64, credentials ...string) *CASLCloudProvider {
	client := casl.NewAPIClient(baseURL, token, pultID, credentials...)
	mapper := casl.NewMapper()
	realtime := casl.NewRealtimeService(client)

	p := &CASLCloudProvider{
		client:               client,
		mapper:               mapper,
		realtime:             realtime,
		objectByInternalID:   make(map[int]casl.GrdObject),
		deviceByDeviceID:     make(map[string]casl.Device),
		deviceByObjectID:     make(map[string]casl.Device),
		deviceByNumber:       make(map[int64]casl.Device),
		cachedUsers:          make(map[string]casl.User),
		cachedTranslators:    make(map[string]map[string]string),
		realtimeAlarmByObjID: make(map[string]models.Alarm),
	}

	realtime.SetEventHandler(p.handleRealtimeEvents)
	return p
}

func (p *CASLCloudProvider) GetObjects() []models.Object {
	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	records, err := p.loadObjects(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("CASL: read_grd_object недоступний, fallback на read_pult")
		pults, pErr := p.client.ReadPults(ctx, 0, casl.ReadLimit)
		if pErr != nil {
			log.Error().Err(pErr).Msg("CASL: не вдалося завантажити об'єкти")
			return nil
		}
		objects := make([]models.Object, 0, len(pults))
		for _, item := range pults {
			objects = append(objects, p.mapper.ToPultObject(item))
		}
		return objects
	}

	_, _ = p.loadDevices(ctx)

	objects := make([]models.Object, 0, len(records))
	for _, record := range records {
		device, _ := p.resolveDeviceForObject(record)
		obj := p.mapper.ToObject(record, &device)
		objects = append(objects, obj)
	}
	return objects
}

func (p *CASLCloudProvider) GetObjectByID(idStr string) *models.Object {
	objectID, err := strconv.Atoi(idStr)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, objectID)
	if err != nil || !found {
		return nil
	}

	_, _ = p.loadDevices(ctx)

	device, hasDevice := p.resolveDeviceForObject(record)
	obj := p.mapper.ToObject(record, &device)

	if state, stateErr := p.client.ReadDeviceState(ctx, strconv.FormatInt(record.DeviceID.Int64(), 10)); stateErr == nil {
		obj.IsConnState = int64(state.Online.Int64())
		obj.IsConnOK = obj.IsConnState > 0
		if state.LastPingDate.Int64() > 0 {
			obj.LastMessageTime = time.UnixMilli(state.LastPingDate.Int64()).Local()
		}
		obj.Groups = p.mapper.MapObjectGroups(state.Groups, record.Rooms)
	}

	if hasDevice {
		p.mapper.EnrichObjectWithDeviceMeta(&obj, &device, "")
	}

	return &obj
}

func (p *CASLCloudProvider) GetZones(objectID string) []models.Zone {
	internalID, err := strconv.Atoi(objectID)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	_, _ = p.loadDevices(ctx)
	device, _ := p.resolveDeviceForObject(record)

	return p.mapper.ToZones(record, &device)
}

func (p *CASLCloudProvider) GetEmployees(objectID string) []models.Contact {
	internalID, err := strconv.Atoi(objectID)
	if err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found {
		return nil
	}

	users, _ := p.loadUsers(ctx)
	return p.mapper.ToContacts(record, users)
}

func (p *CASLCloudProvider) GetEvents() []models.Event {
	p.realtime.Start(context.Background())
	p.mu.RLock()
	defer p.mu.RUnlock()
	return append([]models.Event(nil), p.cachedEvents...)
}

func (p *CASLCloudProvider) GetObjectEvents(objectID string) []models.Event {
	internalID, err := strconv.Atoi(objectID)
	if err != nil { return nil }

	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found { return nil }

	var resp casl.ReadEventsByIDResponse
	payload := map[string]any{
		"type": "read_events_by_id",
		"objIds": []string{record.ObjID},
	}
	if err := p.client.PostCommand(ctx, payload, &resp, true); err != nil {
		return nil
	}

	rows := resp.Data
	if len(rows) == 0 { rows = resp.Events }

	deviceType := ""
	if dev, ok := p.deviceByObjectID[record.ObjID]; ok {
		deviceType = dev.Type.String()
	}

	events := make([]models.Event, 0, len(rows))
	for _, row := range rows {
		events = append(events, p.mapRowToEvent(row, deviceType))
	}
	return events
}

func (p *CASLCloudProvider) GetAlarms() []models.Alarm {
	p.realtime.Start(context.Background())
	p.mu.RLock()
	defer p.mu.RUnlock()

	alarms := make([]models.Alarm, 0, len(p.realtimeAlarmByObjID))
	for _, alarm := range p.realtimeAlarmByObjID {
		alarms = append(alarms, alarm)
	}
	return alarms
}

func (p *CASLCloudProvider) ProcessAlarm(id string, user string, note string) {
	log.Warn().Str("alarmID", id).Str("user", user).Msg("CASL: ProcessAlarm не підтримується API інтеграцією")
}

func (p *CASLCloudProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	internalID, err := strconv.Atoi(objectID)
	if err != nil { return "н/д", "н/д", time.Time{}, time.Time{} }

	ctx, cancel := context.WithTimeout(context.Background(), casl.HTTPTimeout)
	defer cancel()

	record, found, err := p.resolveObjectRecord(ctx, internalID)
	if err != nil || !found { return "н/д", "н/д", time.Time{}, time.Time{} }

	state, stateErr := p.client.ReadDeviceState(ctx, strconv.FormatInt(record.DeviceID.Int64(), 10))
	if stateErr == nil {
		if state.LastPingDate.Int64() > 0 {
			lastMsg = time.UnixMilli(state.LastPingDate.Int64()).Local()
			lastTest = lastMsg
		}
	}

	testParts := []string{"н/д"}
	if stats, statsErr := p.client.ReadStatsAlarms(ctx, strconv.FormatInt(record.DeviceID.Int64(), 10), record.ObjID, time.Now().Add(-casl.StatsSpan).UnixMilli(), time.Now().UnixMilli()); statsErr == nil {
		testParts = []string{
			fmt.Sprintf("freq=%d", stats.ResponseFrequencies.Int64()),
			fmt.Sprintf("quality=%d", stats.CommunicQuality.Int64()),
			fmt.Sprintf("alarms=%d", stats.CustomWins.Int64()),
			fmt.Sprintf("power=%d", stats.PowerFailure.Int64()),
		}
	}

	return "н/д", strings.Join(testParts, "; "), lastTest, lastMsg
}

func (p *CASLCloudProvider) GetTestMessages(objectID string) []models.TestMessage {
	events := p.GetObjectEvents(objectID)
	if len(events) == 0 { return nil }

	messages := make([]models.TestMessage, 0, 32)
	for _, ev := range events {
		if ev.Type == models.EventTest || strings.Contains(strings.ToUpper(ev.Details), "TEST") {
			messages = append(messages, models.TestMessage{Time: ev.Time, Info: ev.GetTypeDisplay(), Details: ev.Details})
		}
		if len(messages) >= 200 { break }
	}
	return messages
}

func (p *CASLCloudProvider) GetLatestEventID() (int64, error) {
	p.realtime.Start(context.Background())
	p.mu.RLock()
	revision := p.eventsRevision
	p.mu.RUnlock()
	return revision, nil
}

func (p *CASLCloudProvider) handleRealtimeEvents(rows []casl.ObjectEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, row := range rows {
		// Resolve device type for decoding
		deviceType := ""
		if dev, ok := p.deviceByObjectID[row.ObjID.String()]; ok {
			deviceType = dev.Type.String()
		}

		ev := p.mapRowToEvent(row, deviceType)
		p.cachedEvents = append([]models.Event{ev}, p.cachedEvents...)
		if len(p.cachedEvents) > casl.MaxCachedEvents {
			p.cachedEvents = p.cachedEvents[:casl.MaxCachedEvents]
		}
		p.eventsRevision++

		action := strings.ToUpper(row.Action.String())
		if action == "GRD_OBJ_MGR_CANCEL" || action == "GRD_OBJ_FINISH" || ev.Type == models.EventRestore || ev.Type == models.EventPowerOK || ev.Type == models.EventOnline {
			delete(p.realtimeAlarmByObjID, row.ObjID.String())
		} else if isAlarmType(ev.Type) {
			p.realtimeAlarmByObjID[row.ObjID.String()] = models.Alarm{
				ID: ev.ID,
				ObjectID: ev.ObjectID,
				ObjectName: ev.ObjectName,
				Type: mapEventToAlarmType(ev.Type),
				Time: ev.Time,
				Details: ev.Details,
			}
		}
	}
}

func (p *CASLCloudProvider) mapRowToEvent(row casl.ObjectEvent, deviceType string) models.Event {
	eventType := protocol.ClassifyEventTypeWithContext(row.Code.String(), row.ContactID.String(), row.Type, row.ObjName.String())

	dict := p.loadDictionaryMap(context.Background())
	trans := p.loadTranslatorMap(context.Background(), deviceType)

	return models.Event{
		ID: casl.StableEventID(row.ObjID.String(), row.Time.Int64(), row.Code.String(), 0),
		Time: time.UnixMilli(row.Time.Int64()),
		ObjectID: casl.MapObjectID(row.ObjID.String()),
		ObjectName: row.ObjName.String(),
		Type: eventType,
		Details: protocol.DecodeEventDescription(trans, dict, row.Code.String(), row.ContactID.String(), int(row.Number.Int64()), deviceType),
		SC1: protocol.MapEventSC1(eventType),
	}
}

func (p *CASLCloudProvider) loadDictionaryMap(ctx context.Context) map[string]string {
	p.mu.RLock()
	if len(p.cachedDictionary) > 0 && time.Since(p.cachedDictionaryAt) <= casl.DictionaryTTL {
		res := casl.FlattenDictionaryMap(p.cachedDictionary)
		p.mu.RUnlock()
		return res
	}
	p.mu.RUnlock()

	dict, err := p.client.ReadDictionary(ctx)
	if err != nil { return nil }

	p.mu.Lock()
	p.cachedDictionary = dict
	p.cachedDictionaryAt = time.Now()
	p.mu.Unlock()

	return casl.FlattenDictionaryMap(dict)
}

func (p *CASLCloudProvider) loadTranslatorMap(ctx context.Context, deviceType string) map[string]string {
	if deviceType == "" { return nil }
	p.mu.RLock()
	if m, ok := p.cachedTranslators[deviceType]; ok {
		p.mu.RUnlock()
		return m
	}
	p.mu.RUnlock()

	raw, err := p.client.GetMessageTranslatorByDeviceType(ctx, deviceType)
	if err != nil { return nil }

	m := casl.FlattenTranslatorMap(raw)
	p.mu.Lock()
	p.cachedTranslators[deviceType] = m
	p.mu.Unlock()

	return m
}

func (p *CASLCloudProvider) loadObjects(ctx context.Context) ([]casl.GrdObject, error) {
	p.mu.RLock()
	if len(p.cachedObjects) > 0 && time.Since(p.cachedObjectsAt) < casl.ObjectsCacheTTL {
		copied := append([]casl.GrdObject(nil), p.cachedObjects...)
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	records, err := p.client.ReadGuardObjects(ctx, 0, casl.ReadLimit)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	p.cachedObjects = records
	p.cachedObjectsAt = time.Now()
	for i := range records {
		p.mapper.NormalizeObjectRecord(&records[i], casl.Device{})
		internalID := casl.MapObjectID(records[i].ObjID, records[i].Name, strconv.FormatInt(records[i].DeviceNumber.Int64(), 10))
		p.objectByInternalID[internalID] = records[i]
	}
	p.mu.Unlock()

	return records, nil
}

func (p *CASLCloudProvider) loadDevices(ctx context.Context) ([]casl.Device, error) {
	p.mu.RLock()
	if len(p.deviceByDeviceID) > 0 && time.Since(p.cachedDevicesAt) < casl.ObjectsCacheTTL {
		result := make([]casl.Device, 0, len(p.deviceByDeviceID))
		for _, item := range p.deviceByDeviceID {
			result = append(result, item)
		}
		p.mu.RUnlock()
		return result, nil
	}
	p.mu.RUnlock()

	devices, err := p.client.ReadDevices(ctx, 0, casl.ReadLimit)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	for _, device := range devices {
		p.deviceByDeviceID[device.DeviceID.String()] = device
		p.deviceByObjectID[device.ObjID.String()] = device
		p.deviceByNumber[device.Number.Int64()] = device
	}
	p.cachedDevicesAt = time.Now()
	p.mu.Unlock()

	return devices, nil
}

func (p *CASLCloudProvider) resolveDeviceForObject(record casl.GrdObject) (casl.Device, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if dev, ok := p.deviceByDeviceID[strconv.FormatInt(record.DeviceID.Int64(), 10)]; ok {
		return dev, true
	}
	if dev, ok := p.deviceByObjectID[record.ObjID]; ok {
		return dev, true
	}
	if dev, ok := p.deviceByNumber[record.DeviceNumber.Int64()]; ok {
		return dev, true
	}
	return casl.Device{}, false
}

func (p *CASLCloudProvider) resolveObjectRecord(ctx context.Context, internalID int) (casl.GrdObject, bool, error) {
	p.mu.RLock()
	record, ok := p.objectByInternalID[internalID]
	p.mu.RUnlock()
	if ok {
		return record, true, nil
	}

	records, err := p.loadObjects(ctx)
	if err != nil {
		return casl.GrdObject{}, false, err
	}
	for _, item := range records {
		id := casl.MapObjectID(item.ObjID, item.Name, strconv.FormatInt(item.DeviceNumber.Int64(), 10))
		if id == internalID {
			return item, true, nil
		}
	}
	return casl.GrdObject{}, false, nil
}

func (p *CASLCloudProvider) loadUsers(ctx context.Context) (map[string]casl.User, error) {
	p.mu.RLock()
	if len(p.cachedUsers) > 0 && time.Since(p.cachedUsersAt) < casl.UsersCacheTTL {
		copied := make(map[string]casl.User, len(p.cachedUsers))
		for k, v := range p.cachedUsers {
			copied[k] = v
		}
		p.mu.RUnlock()
		return copied, nil
	}
	p.mu.RUnlock()

	users, err := p.client.ReadUsers(ctx, 0, casl.ReadLimit)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	for _, user := range users {
		p.cachedUsers[user.UserID] = user
	}
	p.cachedUsersAt = time.Now()
	p.mu.Unlock()

	return p.cachedUsers, nil
}

func (p *CASLCloudProvider) SessionInfo() casl.SessionInfo {
	token, wsURL, userID, pultID := p.client.GetSessionInfo()

	return casl.SessionInfo{
		Token:  token,
		WSURL:  wsURL,
		UserID: userID,
		PultID: pultID,
	}
}

func (p *CASLCloudProvider) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	return p.client.GetStatisticReport(ctx, name, limit)
}

func isCASLObjectID(id int) bool {
	return id >= casl.ObjectIDNamespaceStart && id <= casl.ObjectIDNamespaceEnd
}

func isAlarmType(t models.EventType) bool {
	return t == models.EventFire || t == models.EventBurglary || t == models.EventPanic || t == models.EventMedical || t == models.EventGas || t == models.EventTamper
}

func mapEventToAlarmType(t models.EventType) models.AlarmType {
	switch t {
	case models.EventFire: return models.AlarmFire
	case models.EventBurglary: return models.AlarmBurglary
	case models.EventPanic: return models.AlarmPanic
	case models.EventMedical: return models.AlarmMedical
	case models.EventGas: return models.AlarmGas
	case models.EventTamper: return models.AlarmTamper
	}
	return models.AlarmFault
}
