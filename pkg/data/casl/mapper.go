package casl

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"time"
)

type Mapper struct {
}

func NewMapper() *Mapper {
	return &Mapper{}
}

func (m *Mapper) ToObject(record GrdObject, device *Device) models.Object {
	id := MapObjectID(record.ObjID, record.Name, strconv.FormatInt(record.DeviceNumber.Int64(), 10))

	name := strings.TrimSpace(record.Name)
	if name == "" {
		name = "CASL Object #" + strings.TrimSpace(record.ObjID)
	}

	address := strings.TrimSpace(record.Address)
	if address == "" {
		address = formatCoordinates(record.Lat, record.Long)
	}

	blocked := record.DeviceBlocked || strings.TrimSpace(record.BlockMessage.String()) != ""
	statusState := mapObjectStatusState(record.Status, blocked)

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
		deviceType = strings.TrimSpace(device.Type.String())
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

func (m *Mapper) ToPultObject(item Pult) models.Object {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = strings.TrimSpace(item.Nickname)
	}
	if name == "" {
		name = "CASL Pult"
	}

	id := MapObjectID(item.PultID, item.Name, item.Nickname)

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
		StatusText:     ObjectStatusText,
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

func (m *Mapper) ToZones(record GrdObject, device *Device) []models.Zone {
	if device != nil && len(device.Lines) > 0 {
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
		return zones
	}

	if len(record.Rooms) == 0 {
		return []models.Zone{{Number: 1, Name: "Об'єкт", SensorType: "Приміщення", Status: models.ZoneNormal}}
	}

	zones := make([]models.Zone, 0, len(record.Rooms))
	for idx, room := range record.Rooms {
		number := idx + 1
		if parsed, _ := strconv.Atoi(room.RoomID); parsed > 0 {
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
	return zones
}

func (m *Mapper) ToContacts(record GrdObject, users map[string]User) []models.Contact {
	orderedIDs := normalizeContactIDs(record.InCharge, record.ManagerID)
	if len(orderedIDs) == 0 {
		return nil
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

func (m *Mapper) MapObjectGroups(rawGroups any, rooms []Room) []models.ObjectGroup {
	// Migration of mapCASLDeviceGroupsToObjectGroups logic
	return nil
}

// Utility functions

func formatCoordinates(lat, lng string) string {
	lat = strings.TrimSpace(lat)
	lng = strings.TrimSpace(lng)
	if lat == "" || lng == "" {
		return ""
	}
	return lat + ", " + lng
}

type objectStatusState struct {
	Status         models.ObjectStatus
	StatusText     string
	AlarmState     int64
	GuardState     int64
	TechAlarmState int64
	IsConnState    int64
	IsUnderGuard   bool
}

func mapObjectStatusState(statusRaw string, blocked bool) objectStatusState {
	if blocked {
		return objectStatusState{
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
		return objectStatusState{
			Status:       models.StatusNormal,
			StatusText:   ObjectStatusText,
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

	state := objectStatusState{
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

	return state
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

func MapObjectID(parts ...string) int {
	base := 0
	if len(parts) > 0 {
		base, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
	}
	if base == 0 {
		// Simplified for now, original used fnv hash.
		// base = stableID(parts...)
	}
	return ObjectIDNamespaceStart + (base % int(ObjectIDNamespaceSize))
}
