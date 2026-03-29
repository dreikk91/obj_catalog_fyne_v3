package data

import (
	"context"
	"obj_catalog_fyne_v3/pkg/data/casl"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
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
		realtimeAlarmByObjID: make(map[string]models.Alarm),
	}

	realtime.SetEventHandler(p.handleRealtimeEvent)
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

	device, _ := p.resolveDeviceForObject(record)
	obj := p.mapper.ToObject(record, &device)

	// Fetch device state for more info
	var state casl.DeviceState
	if err := p.client.PostCommand(ctx, map[string]any{"type": "read_device_state", "device_id": strconv.FormatInt(record.DeviceID.Int64(), 10)}, &state, true); err == nil {
		// Enriched data mapping here...
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
	return nil
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
	return "н/д", "н/д", time.Time{}, time.Time{}
}

func (p *CASLCloudProvider) GetTestMessages(objectID string) []models.TestMessage {
	return nil
}

func (p *CASLCloudProvider) GetLatestEventID() (int64, error) {
	p.mu.RLock()
	revision := p.eventsRevision
	p.mu.RUnlock()
	return revision, nil
}

func (p *CASLCloudProvider) handleRealtimeEvent(event casl.ObjectEvent) {
	p.mu.Lock()
	defer p.mu.Unlock()
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
	for _, obj := range records {
		internalID := casl.MapObjectID(obj.ObjID, obj.Name, strconv.FormatInt(obj.DeviceNumber.Int64(), 10))
		p.objectByInternalID[internalID] = obj
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
