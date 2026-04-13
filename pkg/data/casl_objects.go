package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/rs/zerolog/log"
)

func (p *CASLCloudProvider) GetObjects() []models.Object {
	p.ensureRealtimeStream()

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
	disconnected := p.loadDisconnectedDevicesIndex(ctx)
	activeAlarms := p.loadCASLActiveAlarmIndex(ctx)

	objects := make([]models.Object, 0, len(records))
	for _, record := range records {
		device, hasDevice := p.resolveDeviceForObject(record)
		obj := mapCASLGrdObjectToObject(record, selectCASLDevice(hasDevice, device))
		p.enrichCASLObjectWithDeviceMeta(ctx, &obj, hasDevice, device)
		if hasDevice {
			applyCASLObjectDeviceConnectivityState(&obj, device)
		}
		if disconnected.match(record, selectCASLDevice(hasDevice, device)) {
			applyCASLObjectDisconnectedState(&obj, disconnected.lastSeen(record, selectCASLDevice(hasDevice, device)))
		}
		if obj.IsConnState > 0 {
			if alarm, ok := activeAlarms[obj.ID]; ok {
				applyCASLObjectAlarmState(&obj, alarm)
			}
		}
		objects = append(objects, obj)
	}
	// sortCASLObjectsByNumber(objects)
	return objects
}

// func sortCASLObjectsByNumber(objects []models.Object) {
// 	sort.SliceStable(objects, func(i, j int) bool {
// 		ni := extractLeadingCASLNumber(objects[i].Name)
// 		nj := extractLeadingCASLNumber(objects[j].Name)
// 		if ni != "" && nj != "" {
// 			vi, _ := strconv.Atoi(ni)
// 			vj, _ := strconv.Atoi(nj)
// 			if vi != vj {
// 				return vi < vj
// 			}
// 		}
// 		if ni != "" && nj == "" {
// 			return true
// 		}
// 		if ni == "" && nj != "" {
// 			return false
// 		}
// 		return objects[i].Name < objects[j].Name
// 	})
// }

func (p *CASLCloudProvider) GetDisplayNumber(internalID int) string {
	p.mu.RLock()
	record, ok := p.objectByInternalID[internalID]
	p.mu.RUnlock()

	if !ok {
		return ""
	}

	return preferredCASLObjectNumber(record.ObjID, record.Name, record.DeviceNumber.Int64())
}

func (p *CASLCloudProvider) GetObjectByID(idStr string) *models.Object {
	objectID, ok := parseObjectID(idStr)
	if !ok {
		return nil
	}

	p.ensureRealtimeStream()

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
	if hasDevice {
		applyCASLObjectDeviceConnectivityState(&obj, device)
	}
	if state, stateErr := p.readDeviceState(ctx, record); stateErr == nil {
		obj.PowerFault = state.Power.Int64()
		obj.AkbState = state.Accum.Int64()
		obj.PowerSource = models.PowerMains
		if obj.PowerFault == 0 {
			obj.PowerSource = models.PowerBattery
		}
		if state.Online.Int64() > 0 {
			obj.IsConnState = 1
			obj.IsConnOK = true
		} else {
			applyCASLObjectDisconnectedState(&obj, time.UnixMilli(state.LastPingDate.Int64()).Local())
		}
		if state.LastPingDate.Int64() > 0 {
			msgTime := time.UnixMilli(state.LastPingDate.Int64()).Local()
			obj.LastMessageTime = msgTime
			obj.LastTestTime = msgTime
		}
		obj.Groups = p.buildCASLObjectGroups(ctx, record, state.Groups)
	}
	if hasDevice {
		applyCASLObjectDeviceConnectivityState(&obj, device)
	}
	if obj.IsConnState > 0 {
		disconnected := p.loadDisconnectedDevicesIndex(ctx)
		if disconnected.match(record, selectCASLDevice(hasDevice, device)) {
			applyCASLObjectDisconnectedState(&obj, disconnected.lastSeen(record, selectCASLDevice(hasDevice, device)))
		}
	}
	if obj.IsConnState > 0 {
		if alarm, ok := p.loadCASLActiveAlarmIndex(ctx)[obj.ID]; ok {
			applyCASLObjectAlarmState(&obj, alarm)
		}
	}

	return &obj
}

type caslDisconnectedDevicesIndex struct {
	byDeviceID map[string]time.Time
	byObjID    map[string]time.Time
	byNumber   map[int64]time.Time
}

func (p *CASLCloudProvider) loadDisconnectedDevicesIndex(ctx context.Context) caslDisconnectedDevicesIndex {
	rows, err := p.GetDisconnectedDevices(ctx)
	if err != nil {
		log.Debug().Err(err).Msg("CASL: не вдалося завантажити get_disconnected_devices")
		return caslDisconnectedDevicesIndex{}
	}
	index := caslDisconnectedDevicesIndex{
		byDeviceID: make(map[string]time.Time, len(rows)),
		byObjID:    make(map[string]time.Time, len(rows)),
		byNumber:   make(map[int64]time.Time, len(rows)),
	}
	for _, row := range rows {
		seenAt, _ := firstCASLTimeValue(row["offline"], row["last"], row["lastPingDate"], row["date"])
		if deviceID := strings.TrimSpace(firstCASLString(row["device_id"], row["id"])); deviceID != "" {
			index.byDeviceID[deviceID] = seenAt
		}
		if objID := strings.TrimSpace(asString(row["obj_id"])); objID != "" {
			index.byObjID[objID] = seenAt
		}
		if number, ok := firstCASLIntValue(row["number"], row["device_number"], row["ppk_num"]); ok && number > 0 {
			index.byNumber[int64(number)] = seenAt
		}
	}
	return index
}

func (i caslDisconnectedDevicesIndex) match(record caslGrdObject, device *caslDevice) bool {
	_, ok := i.lastSeenWithState(record, device)
	return ok
}

func (i caslDisconnectedDevicesIndex) lastSeen(record caslGrdObject, device *caslDevice) time.Time {
	lastSeen, _ := i.lastSeenWithState(record, device)
	return lastSeen
}

func (i caslDisconnectedDevicesIndex) lastSeenWithState(record caslGrdObject, device *caslDevice) (time.Time, bool) {
	if device != nil {
		if deviceID := strings.TrimSpace(device.DeviceID.String()); deviceID != "" {
			if value, ok := i.byDeviceID[deviceID]; ok {
				return value, true
			}
		}
		if objID := strings.TrimSpace(device.ObjID.String()); objID != "" {
			if value, ok := i.byObjID[objID]; ok {
				return value, true
			}
		}
		if number := device.Number.Int64(); number > 0 {
			if value, ok := i.byNumber[number]; ok {
				return value, true
			}
		}
	}
	if objID := strings.TrimSpace(record.ObjID); objID != "" {
		if value, ok := i.byObjID[objID]; ok {
			return value, true
		}
	}
	if number := record.DeviceNumber.Int64(); number > 0 {
		if value, ok := i.byNumber[number]; ok {
			return value, true
		}
	}
	return time.Time{}, false
}

func applyCASLObjectDisconnectedState(obj *models.Object, lastSeen time.Time) {
	if obj == nil {
		return
	}
	if obj.BlockedArmedOnOff == 1 {
		if !lastSeen.IsZero() && lastSeen.After(obj.LastMessageTime) {
			obj.LastMessageTime = lastSeen
			if obj.LastTestTime.IsZero() {
				obj.LastTestTime = lastSeen
			}
		}
		return
	}
	obj.Status = models.StatusOffline
	obj.StatusText = "НЕМАЄ ЗВ'ЯЗКУ"
	obj.AlarmState = 0
	obj.TechAlarmState = 0
	obj.BlockedArmedOnOff = 0
	obj.IsConnState = 0
	obj.IsConnOK = false
	if !lastSeen.IsZero() {
		obj.LastMessageTime = lastSeen
		if obj.LastTestTime.IsZero() {
			obj.LastTestTime = lastSeen
		}
	}
}

func applyCASLObjectDeviceConnectivityState(obj *models.Object, device caslDevice) {
	if obj == nil {
		return
	}

	if device.Disconnected {
		lastSeen := time.Time{}
		if device.LastPingDate.Int64() > 0 {
			lastSeen = time.UnixMilli(device.LastPingDate.Int64()).Local()
		}
		applyCASLObjectDisconnectedState(obj, lastSeen)
		return
	}

	if device.Offline.Int64() > 0 {
		applyCASLObjectDisconnectedState(obj, time.UnixMilli(device.Offline.Int64()).Local())
	}
}

func (p *CASLCloudProvider) loadCASLActiveAlarmIndex(ctx context.Context) map[int]models.Alarm {
	_ = ctx
	alarms := p.snapshotRealtimeAlarms()
	if len(alarms) == 0 {
		return nil
	}

	index := make(map[int]models.Alarm, len(alarms))
	for _, alarm := range alarms {
		if alarm.ObjectID <= 0 {
			continue
		}
		existing, exists := index[alarm.ObjectID]
		if !exists ||
			caslObjectAlarmPriority(alarm) > caslObjectAlarmPriority(existing) ||
			(caslObjectAlarmPriority(alarm) == caslObjectAlarmPriority(existing) && alarm.Time.After(existing.Time)) {
			index[alarm.ObjectID] = alarm
		}
	}
	return index
}

func applyCASLObjectAlarmState(obj *models.Object, alarm models.Alarm) {
	if obj == nil || obj.BlockedArmedOnOff == 1 || obj.IsConnState == 0 {
		return
	}

	obj.Status = models.StatusFire
	obj.StatusText = alarm.GetTypeDisplay()
	if strings.TrimSpace(obj.StatusText) == "" {
		obj.StatusText = "ТРИВОГА"
	}
	obj.AlarmState = 1
	obj.TechAlarmState = 0
	obj.IsConnState = 1
	obj.IsConnOK = true
	if !alarm.Time.IsZero() && alarm.Time.After(obj.LastMessageTime) {
		obj.LastMessageTime = alarm.Time
	}
}

func caslObjectAlarmPriority(alarm models.Alarm) int {
	switch alarm.Type {
	case models.AlarmFire:
		return 120
	case models.AlarmPanic:
		return 115
	case models.AlarmBurglary:
		return 110
	case models.AlarmMedical:
		return 105
	case models.AlarmGas:
		return 100
	case models.AlarmTamper:
		return 95
	case models.AlarmDevice:
		return 90
	case models.AlarmMobile:
		return 85
	case models.AlarmOperator:
		return 80
	case models.AlarmPowerFail:
		return 70
	case models.AlarmBatteryLow:
		return 65
	case models.AlarmOffline:
		return 60
	case models.AlarmFault:
		return 55
	case models.AlarmNotification:
		return 50
	default:
		return 10
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

func normalizeCASLObjectRecord(record *caslGrdObject, device caslDevice) {
	if record == nil {
		return
	}

	managerID := strings.TrimSpace(record.ManagerID)
	if managerID == "" {
		managerID = strings.TrimSpace(record.Manager.UserID)
	}
	record.ManagerID = managerID

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

func (p *CASLCloudProvider) readGrdObjects(ctx context.Context) ([]caslGrdObject, error) {
	payload := map[string]any{"type": "read_grd_object", "skip": 0, "limit": caslReadLimit}

	var resp caslReadGrdObjectResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
		return nil, err
	}
	if err := validateCASLGuardObjects(resp.Data); err != nil {
		return nil, err
	}

	return append([]caslGrdObject(nil), resp.Data...), nil
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
	rowsErr := json.Unmarshal(resp.Data, &rows)
	if rowsErr == nil {
		if validateErr := validateCASLConnections(rows); validateErr != nil {
			return nil, validateErr
		}
		return rows, nil
	}

	var single caslConnectionRecord
	singleErr := json.Unmarshal(resp.Data, &single)
	if singleErr == nil {
		if single.hasPayload() {
			if validateErr := validateCASLConnections([]caslConnectionRecord{single}); validateErr != nil {
				return nil, validateErr
			}
			return []caslConnectionRecord{single}, nil
		}
	}

	if rowsErr != nil {
		return nil, fmt.Errorf("casl read_connections: decode rows: %w", rowsErr)
	}
	if singleErr != nil {
		return nil, fmt.Errorf("casl read_connections: decode record: %w", singleErr)
	}

	return nil, fmt.Errorf("casl read_connections: unsupported payload format")
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

	blocked := record.DeviceBlocked || strings.TrimSpace(record.BlockMessage.String()) != "" || (device != nil && device.Blocked)
	statusState := mapCASLObjectStatusState(record.Status, blocked)

	notes := strings.TrimSpace(record.Note)
	if notes == "" {
		notes = strings.TrimSpace(record.Description)
	}

	// displayNubmer := ""
	// if record.DeviceNumber.Int64() > 0 {
	// displayNubmer = strconv.FormatInt(record.DeviceNumber.Int64(), 10)
	// }

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
	autoTestHours := 24
	testControl := int64(1)
	testTime := int64(0)
	if device != nil {
		if value := strings.TrimSpace(device.Type.String()); value != "" {
			deviceType = decodeCASLDeviceType(value)
		}
		sim1 = strings.TrimSpace(device.SIM1.String())
		sim2 = strings.TrimSpace(device.SIM2.String())
		if timeoutMinutes := caslTimeoutMinutes(device.Timeout.Int64()); timeoutMinutes > 0 {
			testControl = 1
			testTime = timeoutMinutes
			autoTestHours = 0
			if timeoutMinutes%60 == 0 {
				autoTestHours = int(timeoutMinutes / 60)
			}
		}
	}
	lastTestTime := time.Time{}
	if device != nil && device.LastPingDate.Int64() > 0 {
		lastTestTime = time.UnixMilli(device.LastPingDate.Int64()).Local()
	}

	hasAssignment := len(normalizeContactIDs(record.InCharge, record.ManagerID)) > 0
	objectNum := preferredCASLObjectNumber(record.ObjID, record.Name, record.DeviceNumber.Int64())

	return models.Object{
		ID:             id,
		Name:           name,
		DisplayNumber:  objectNum,
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
		TestControl:    testControl,
		TestTime:       testTime,
		LastTestTime:   lastTestTime,
		SIM1:           sim1,
		SIM2:           sim2,
		ObjChan:        5,
		AutoTestHours:  autoTestHours,
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

func caslTimeoutMinutes(timeoutSeconds int64) int64 {
	if timeoutSeconds <= 0 {
		return 0
	}
	return (timeoutSeconds + 59) / 60
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
