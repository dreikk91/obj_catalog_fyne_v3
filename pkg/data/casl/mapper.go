package casl

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"obj_catalog_fyne_v3/pkg/models"
	"sort"
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

func (m *Mapper) EnrichObjectWithDeviceMeta(obj *models.Object, device *Device, deviceTypeLabel string) {
	if obj == nil || device == nil {
		return
	}

	if deviceTypeLabel != "" {
		obj.DeviceType = deviceTypeLabel
	}

	deviceName := strings.TrimSpace(device.Name.String())
	if deviceName != "" {
		obj.Notes1 = deviceName
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
		sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
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
	sort.SliceStable(zones, func(i, j int) bool { return zones[i].Number < zones[j].Number })
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
	candidates := collectGroupCandidates(rawGroups)
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

func (m *Mapper) BuildLineNameIndex(lines []DeviceLine) map[int]string {
	if len(lines) == 0 {
		return nil
	}
	index := make(map[int]string, len(lines))
	for _, line := range lines {
		name := strings.TrimSpace(line.Name.String())
		if name == "" { continue }
		if num := int(line.ID.Int64()); num > 0 {
			index[num] = name
			continue
		}
		if num := int(line.Number.Int64()); num > 0 {
			index[num] = name
		}
	}
	return index
}

func (m *Mapper) NormalizeObjectRecord(record *GrdObject, device Device) {
	if record == nil { return }
	if record.ManagerID == "" {
		record.ManagerID = strings.TrimSpace(record.Manager.UserID)
	}
	if record.ObjID == "" {
		record.ObjID = device.ObjID.String()
	}
	if record.DeviceID.Int64() <= 0 {
		if i, err := strconv.ParseInt(device.DeviceID.String(), 10, 64); err == nil {
			record.DeviceID = Int64(i)
		}
	}
}

// Internal recursive helpers for groups

type groupCandidate struct {
	key   string
	value any
}

func collectGroupCandidates(raw any) []groupCandidate {
	result := make([]groupCandidate, 0, 8)
	collectGroupCandidatesRecursive("", raw, 0, &result)
	return result
}

func collectGroupCandidatesRecursive(keyHint string, raw any, depth int, out *[]groupCandidate) {
	if out == nil || depth > 8 || raw == nil {
		return
	}

	switch typed := raw.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return
		}

		if isGroupPayloadMap(typed) {
			*out = append(*out, groupCandidate{key: keyHint, value: typed})
			return
		}

		hasNumericChildren := false
		for key, value := range typed {
			if parseCASLID(key) > 0 {
				hasNumericChildren = true
				collectGroupCandidatesRecursive(key, value, depth+1, out)
			}
		}
		if hasNumericChildren {
			return
		}

		for _, wrapperKey := range []string{"groups", "group", "items", "list", "data", "result", "values"} {
			if nested, ok := typed[wrapperKey]; ok {
				collectGroupCandidatesRecursive(keyHint, nested, depth+1, out)
				return
			}
		}

		hasNested := false
		for key, value := range typed {
			switch value.(type) {
			case map[string]any, []any:
				hasNested = true
				collectGroupCandidatesRecursive(key, value, depth+1, out)
			}
		}
		if hasNested {
			return
		}

		*out = append(*out, groupCandidate{key: keyHint, value: typed})
	case []any:
		for idx, item := range typed {
			collectGroupCandidatesRecursive(strconv.Itoa(idx+1), item, depth+1, out)
		}
	default:
		if strings.TrimSpace(keyHint) == "" {
			return
		}
		*out = append(*out, groupCandidate{key: keyHint, value: raw})
	}
}

func isGroupPayloadMap(payload map[string]any) bool {
	if len(payload) == 0 {
		return false
	}

	for _, key := range []string{
		"group", "group_id", "group_number", "number", "id", "state", "status", "group_state", "groupStatus",
		"is_on", "is_armed", "armed", "guard", "on", "group_on", "room", "room_id", "room_name", "name_room",
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
	case strings.Contains(lower, "on"), strings.Contains(lower, "group_on"), strings.Contains(lower, "guard"), strings.Contains(lower, "arm"), strings.Contains(lower, "взят"), strings.Contains(lower, "включ"), strings.Contains(lower, "під охор"):
		group.Armed = true
		group.StateText = "ПІД ОХОРОНОЮ"
	case strings.Contains(lower, "off"), strings.Contains(lower, "group_off"), strings.Contains(lower, "disarm"), strings.Contains(lower, "знят"), strings.Contains(lower, "виключ"):
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
		base = parseCASLID(parts[0])
	}
	if base == 0 {
		base = stableID(parts...)
	}
	return ObjectIDNamespaceStart + (base % int(ObjectIDNamespaceSize))
}

func stableID(parts ...string) int {
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

func parseCASLID(raw string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
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
