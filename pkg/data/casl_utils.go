package data

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"

	"github.com/rs/zerolog/log"
)

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

func hasCyrillicChars(text string) bool {
	return utils.HasCyrillicChars(text)
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

func parseCASLAnyInt(value any) int {
	return utils.ParseAnyInt(value)
}

func tryParseCASLAnyInt(value any) (int, bool) {
	switch typed := value.(type) {
	case nil:
		return 0, false
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case int32:
		return int(typed), true
	case float64:
		return int(typed), true
	case float32:
		return int(typed), true
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed), true
		}
		if parsed, err := typed.Float64(); err == nil {
			return int(parsed), true
		}
		return 0, false
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, false
		}
		if parsed, err := strconv.Atoi(text); err == nil {
			return parsed, true
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			return int(parsed), true
		}
		return 0, false
	default:
		return 0, false
	}
}

func parseCASLAnyTime(value any) time.Time {
	return utils.ParseAnyTime(value)
}

func tryParseCASLAnyTime(value any) (time.Time, bool) {
	parseEpoch := func(epoch int64) (time.Time, bool) {
		if epoch == 0 {
			return time.Time{}, false
		}
		if epoch > 1_000_000_000_000 || epoch < -1_000_000_000_000 {
			return time.UnixMilli(epoch).Local(), true
		}
		if epoch > 1_000_000_000 || epoch < -1_000_000_000 {
			return time.Unix(epoch, 0).Local(), true
		}
		return time.Time{}, false
	}

	switch typed := value.(type) {
	case nil:
		return time.Time{}, false
	case time.Time:
		if typed.IsZero() {
			return time.Time{}, false
		}
		return typed.Local(), true
	case int64:
		return parseEpoch(typed)
	case int:
		return parseEpoch(int64(typed))
	case float64:
		return parseEpoch(int64(typed))
	case float32:
		return parseEpoch(int64(typed))
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parseEpoch(parsed)
		}
		if parsed, err := typed.Float64(); err == nil {
			return parseEpoch(int64(parsed))
		}
		return time.Time{}, false
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed.Local(), true
		}
		if parsed, err := time.Parse(time.RFC3339, text); err == nil {
			return parsed.Local(), true
		}
		if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
			return parseEpoch(parsed)
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			return parseEpoch(int64(parsed))
		}
		return time.Time{}, false
	default:
		return time.Time{}, false
	}
}

func firstCASLTextValue(values ...any) string {
	for _, value := range values {
		text := strings.TrimSpace(asString(value))
		if text != "" {
			return text
		}
	}
	return ""
}

func firstCASLIntValue(values ...any) (int, bool) {
	for _, value := range values {
		if parsed, ok := tryParseCASLAnyInt(value); ok {
			return parsed, true
		}
	}
	return 0, false
}

func firstCASLTimeValue(values ...any) (time.Time, bool) {
	for _, value := range values {
		if parsed, ok := tryParseCASLAnyTime(value); ok {
			return parsed, true
		}
	}
	return time.Time{}, false
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
			Source: "casl",
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
		if group.Name == "" {
			group.Name = group.RoomName
		}
		if group.PremiseID == "" {
			group.PremiseID = group.RoomID
		}
		if group.PremiseName == "" {
			group.PremiseName = group.RoomName
		}
		if group.ID == "" {
			group.ID = fmt.Sprintf("casl:group=%d", group.Number)
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

func alignCASLGroupsWithDeviceLines(
	groups []models.ObjectGroup,
	lines []caslDeviceLine,
	rooms []caslRoom,
) []models.ObjectGroup {
	if len(groups) == 0 && len(lines) == 0 {
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

	lineGroupsByRoom := make(map[string][]int)
	lineGroupNumbers := make([]int, 0, len(lines))
	for _, line := range lines {
		number := caslLineGroupNumber(line)
		if number <= 0 {
			continue
		}
		lineGroupNumbers = appendCASLUniqueInt(lineGroupNumbers, number)

		roomID := strings.TrimSpace(line.RoomID.String())
		if roomID == "" {
			continue
		}
		lineGroupsByRoom[roomID] = appendCASLUniqueInt(lineGroupsByRoom[roomID], number)
	}

	mergedByNumber := make(map[int]models.ObjectGroup, len(groups)+len(lineGroupNumbers))
	order := make([]int, 0, len(groups)+len(lineGroupNumbers))
	placeholdersByRoom := make(map[string]models.ObjectGroup)

	appendGroup := func(group models.ObjectGroup) {
		group = normalizeCASLObjectGroup(group, roomNames)
		if group.Number <= 0 {
			return
		}

		if existing, ok := mergedByNumber[group.Number]; ok {
			mergedByNumber[group.Number] = mergeCASLObjectGroup(existing, group)
			return
		}

		mergedByNumber[group.Number] = group
		order = append(order, group.Number)
	}

	for _, group := range groups {
		roomID := roomIDFromCASLObjectGroup(group)
		roomNumbers := lineGroupsByRoom[roomID]
		if roomID != "" && len(roomNumbers) > 0 && !slices.Contains(roomNumbers, group.Number) {
			if _, exists := placeholdersByRoom[roomID]; !exists {
				placeholdersByRoom[roomID] = normalizeCASLObjectGroup(group, roomNames)
			}

			for _, number := range roomNumbers {
				clone := group
				clone.Number = number
				clone.ID = fmt.Sprintf("casl:group=%d", number)
				appendGroup(clone)
			}
			continue
		}

		appendGroup(group)
	}

	for _, number := range lineGroupNumbers {
		if _, exists := mergedByNumber[number]; exists {
			continue
		}

		group := models.ObjectGroup{
			ID:     fmt.Sprintf("casl:group=%d", number),
			Source: "casl",
			Number: number,
		}

		for roomID, roomNumbers := range lineGroupsByRoom {
			if !slices.Contains(roomNumbers, number) {
				continue
			}
			if placeholder, ok := placeholdersByRoom[roomID]; ok {
				placeholder.Number = number
				placeholder.ID = group.ID
				group = mergeCASLObjectGroup(group, normalizeCASLObjectGroup(placeholder, roomNames))
			} else {
				group.RoomID = roomID
				group.PremiseID = roomID
				group.RoomName = roomNames[roomID]
				group.PremiseName = roomNames[roomID]
			}
			break
		}

		appendGroup(group)
	}

	if len(mergedByNumber) == 0 {
		return nil
	}

	result := make([]models.ObjectGroup, 0, len(mergedByNumber))
	for _, number := range order {
		group := normalizeCASLObjectGroup(mergedByNumber[number], roomNames)
		result = append(result, group)
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Number == result[j].Number {
			return result[i].RoomName < result[j].RoomName
		}
		return result[i].Number < result[j].Number
	})

	return result
}

func collectCASLGroupCandidates(raw any) []caslGroupCandidate {
	result := make([]caslGroupCandidate, 0, 8)
	collectCASLGroupCandidatesRecursive("", raw, 0, &result)
	return result
}

func roomIDFromCASLObjectGroup(group models.ObjectGroup) string {
	if roomID := strings.TrimSpace(group.RoomID); roomID != "" {
		return roomID
	}
	return strings.TrimSpace(group.PremiseID)
}

func normalizeCASLObjectGroup(group models.ObjectGroup, roomNames map[string]string) models.ObjectGroup {
	if group.Number > 0 && strings.TrimSpace(group.ID) == "" {
		group.ID = fmt.Sprintf("casl:group=%d", group.Number)
	}
	if strings.TrimSpace(group.Source) == "" {
		group.Source = "casl"
	}

	roomID := roomIDFromCASLObjectGroup(group)
	if roomID != "" {
		group.RoomID = roomID
		if group.PremiseID == "" {
			group.PremiseID = roomID
		}
		if group.RoomName == "" {
			group.RoomName = roomNames[roomID]
		}
		if group.PremiseName == "" {
			group.PremiseName = group.RoomName
		}
	}

	if group.Name == "" {
		group.Name = group.RoomName
	}
	if group.StateText == "" {
		if group.Armed {
			group.StateText = "ПІД ОХОРОНОЮ"
		} else {
			group.StateText = "ЗНЯТО"
		}
	}

	return group
}

func mergeCASLObjectGroup(dst, src models.ObjectGroup) models.ObjectGroup {
	if dst.Number <= 0 {
		dst.Number = src.Number
	}
	if strings.TrimSpace(dst.ID) == "" {
		dst.ID = strings.TrimSpace(src.ID)
	}
	if strings.TrimSpace(dst.Source) == "" {
		dst.Source = strings.TrimSpace(src.Source)
	}
	if strings.TrimSpace(dst.Name) == "" {
		dst.Name = strings.TrimSpace(src.Name)
	}
	if strings.TrimSpace(dst.StateText) == "" {
		dst.StateText = strings.TrimSpace(src.StateText)
	}
	if !dst.Armed && src.Armed {
		dst.Armed = true
	}
	if strings.TrimSpace(dst.RoomID) == "" {
		dst.RoomID = strings.TrimSpace(src.RoomID)
	}
	if strings.TrimSpace(dst.RoomName) == "" {
		dst.RoomName = strings.TrimSpace(src.RoomName)
	}
	if strings.TrimSpace(dst.PremiseID) == "" {
		dst.PremiseID = strings.TrimSpace(src.PremiseID)
	}
	if strings.TrimSpace(dst.PremiseName) == "" {
		dst.PremiseName = strings.TrimSpace(src.PremiseName)
	}

	return dst
}

func appendCASLUniqueInt(dst []int, value int) []int {
	if value <= 0 || slices.Contains(dst, value) {
		return dst
	}
	return append(dst, value)
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
		if group.Name == "" {
			group.Name = roomName
		}
	}
	if roomName := strings.TrimSpace(asString(payload["name_room"])); roomName != "" {
		group.RoomName = roomName
		if group.Name == "" {
			group.Name = roomName
		}
	}
	if roomID := strings.TrimSpace(asString(payload["room_id"])); roomID != "" {
		group.RoomID = roomID
		if group.PremiseID == "" {
			group.PremiseID = roomID
		}
	}
	if roomID := strings.TrimSpace(asString(payload["roomId"])); roomID != "" {
		group.RoomID = roomID
		if group.PremiseID == "" {
			group.PremiseID = roomID
		}
	}

	if room, ok := payload["room"].(map[string]any); ok {
		if roomName := strings.TrimSpace(asString(room["name"])); roomName != "" {
			group.RoomName = roomName
			if group.Name == "" {
				group.Name = roomName
			}
		}
		if roomID := strings.TrimSpace(asString(room["room_id"])); roomID != "" {
			group.RoomID = roomID
			if group.PremiseID == "" {
				group.PremiseID = roomID
			}
		}
		if roomID := strings.TrimSpace(asString(room["id"])); roomID != "" && group.RoomID == "" {
			group.RoomID = roomID
			if group.PremiseID == "" {
				group.PremiseID = roomID
			}
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
	return utils.BoolFromAny(value)
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
	return ids.CASLObjectIDNamespaceStart + (base % ids.CASLObjectIDNamespaceSize)
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

func normalizeCASLAlarmState(raw int64) int64 {
	if raw == 0 {
		return 0
	}
	return 1
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

func stableCASLEventID(objID string, ts int64, seed string, index int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(objID)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.FormatInt(ts, 10)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(seed)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.Itoa(index)))

	base := int(h.Sum32() & 0x7fffffff)
	if base == 0 {
		return nextCASLEventID()
	}
	return ids.CASLObjectIDNamespaceStart + (base % ids.CASLObjectIDNamespaceSize)
}

func stableCASLAlarmSeed(code string, contactID string, zoneNumber int) string {
	return strings.TrimSpace(code) + "|" + strings.TrimSpace(contactID) + "|" + strconv.Itoa(zoneNumber)
}

func stableCASLAlarmID(objKey string, ts int64, seed string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(objKey)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strconv.FormatInt(ts, 10)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(seed)))

	base := int(h.Sum32() & 0x7fffffff)
	if base == 0 {
		return nextCASLEventID()
	}
	return ids.CASLObjectIDNamespaceStart + (base % ids.CASLObjectIDNamespaceSize)
}

func nextCASLEventID() int {
	base := int(time.Now().UnixMilli() & 0x7fffffff)
	return ids.CASLObjectIDNamespaceStart + (base % ids.CASLObjectIDNamespaceSize)
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
	return maps.Clone(source)
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
	return utils.AsString(value)
}
