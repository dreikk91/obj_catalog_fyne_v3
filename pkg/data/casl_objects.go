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

func (p *CASLCloudProvider) readGrdObjects(ctx context.Context) ([]caslGrdObject, error) {
	payload := map[string]any{"type": "read_grd_object", "skip": 0, "limit": caslReadLimit}

	var resp caslReadGrdObjectResponse
	if err := p.postCommand(ctx, payload, &resp, true); err != nil {
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
	if device != nil {
		if value := strings.TrimSpace(device.Type.String()); value != "" {
			deviceType = decodeCASLDeviceType(value)
		}
		sim1 = strings.TrimSpace(device.SIM1.String())
		sim2 = strings.TrimSpace(device.SIM2.String())
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
