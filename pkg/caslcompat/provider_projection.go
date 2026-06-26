package caslcompat

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"time"
)

const defaultProviderDeviceType = "UNIFIED_PROVIDER_GENERIC"

type ProviderFixtureOptions struct {
	SourceName string
	Admin      UnifiedUser
	Managers   []UnifiedManager
	Pults      []UnifiedPult
	DeviceType UnifiedDeviceType
}

func BuildFixtureFromDataProvider(provider contracts.DataProvider, options ProviderFixtureOptions) Fixture {
	return buildFixtureFromUnified(BuildUnifiedFixtureFromDataProvider(provider, options))
}

func BuildUnifiedFixtureFromDataProvider(provider contracts.DataProvider, options ProviderFixtureOptions) UnifiedFixture {
	options = normalizeProviderFixtureOptions(options)
	if provider == nil {
		return UnifiedFixture{
			Admin:       options.Admin,
			Managers:    options.Managers,
			Pults:       options.Pults,
			DeviceTypes: []UnifiedDeviceType{options.DeviceType},
			EventTypes:  defaultProviderEventTypes(options.DeviceType.Type),
		}
	}

	objects := provider.GetObjects()
	publicObjectIDs := providerPublicObjectIDs(objects)
	unifiedObjects := make([]UnifiedObject, 0, len(objects))
	unifiedDevices := make([]UnifiedDevice, 0, len(objects))
	responders := make([]UnifiedUser, 0)
	respondersByKey := map[string]string{}
	deviceTypes := map[string]UnifiedDeviceType{}
	deviceTypes[options.DeviceType.Type] = options.DeviceType

	for _, object := range objects {
		objectID := object.ID
		if objectID == 0 {
			continue
		}
		publicObjectID := publicObjectIDs[objectID]
		roomID := providerRoomID(publicObjectID)
		contacts := provider.GetEmployees(strconv.Itoa(objectID))
		responsibleIDs := make([]string, 0, len(contacts))
		for index, contact := range contacts {
			userID := providerResponsibleID(options.SourceName, objectID, index+1, contact)
			key := userID
			if contact.Phone != "" {
				key = strings.TrimSpace(contact.Phone)
			}
			if existingID, ok := respondersByKey[key]; ok {
				responsibleIDs = append(responsibleIDs, existingID)
				continue
			}
			respondersByKey[key] = userID
			responsibleIDs = append(responsibleIDs, userID)
			responders = append(responders, providerResponder(userID, contact, options.Admin.PultID))
		}

		unifiedObjects = append(unifiedObjects, UnifiedObject{
			ID:             publicObjectID,
			DisplayNumber:  providerObjectNumber(object),
			Name:           providerObjectName(object),
			Address:        strings.TrimSpace(object.Address),
			Description:    providerObjectDescription(object),
			Contract:       strings.TrimSpace(object.ContractNum),
			Status:         object.GetStatusDisplay(),
			ObjectType:     providerObjectType(options.SourceName),
			ReactingPultID: options.Admin.PultID,
			ResponsibleIDs: responsibleIDs,
			Room: UnifiedRoom{
				ID:          roomID,
				Name:        providerRoomName(object),
				Description: providerRoomDescription(object),
			},
			Rooms: providerRoomsFromObject(object, publicObjectID, roomID),
		})

		zones := provider.GetZones(strconv.Itoa(objectID))
		deviceType := providerDeviceType(object, options.DeviceType)
		deviceTypes[deviceType.Type] = deviceType
		unifiedDevices = append(unifiedDevices, UnifiedDevice{
			ID:           providerDeviceID(publicObjectID),
			ObjectID:     publicObjectID,
			Number:       providerDeviceNumber(object),
			Name:         providerDeviceName(object),
			Type:         deviceType.Type,
			SignalLevel:  providerSignalLevel(object),
			Timeout:      providerDeviceTimeout(object),
			SIM1:         strings.TrimSpace(object.SIM1),
			SIM2:         strings.TrimSpace(object.SIM2),
			TechnicianID: providerTechnicianID(object),
			Enabled:      -1,
			Offline:      providerOfflineMS(object),
			Disconnected: providerDisconnected(object),
			Lines:        providerLinesFromZones(zones, object, publicObjectID, roomID),
		})
	}

	activeAlarms := providerActiveAlarms(provider.GetAlarms(), publicObjectIDs)
	journalEvents := providerJournalEvents(provider.GetEvents(), publicObjectIDs)
	unifiedDeviceTypes := make([]UnifiedDeviceType, 0, len(deviceTypes))
	for _, deviceType := range deviceTypes {
		unifiedDeviceTypes = append(unifiedDeviceTypes, deviceType)
	}
	eventTypes := make([]UnifiedEventType, 0, len(unifiedDeviceTypes)*5)
	for _, deviceType := range unifiedDeviceTypes {
		eventTypes = append(eventTypes, defaultProviderEventTypes(deviceType.Type)...)
	}
	return UnifiedFixture{
		Admin:          options.Admin,
		Responders:     responders,
		Managers:       options.Managers,
		Pults:          options.Pults,
		Objects:        unifiedObjects,
		Devices:        unifiedDevices,
		DeviceTypes:    unifiedDeviceTypes,
		EventTypes:     eventTypes,
		ActiveAlarms:   activeAlarms,
		JournalEvents:  journalEvents,
		Disconnected:   providerDisconnectedDevices(unifiedDevices),
		CountOfRooms:   len(unifiedObjects),
		TotalObjects:   len(unifiedObjects),
		OfflineObjects: providerOfflineObjectCount(unifiedDevices),
		ActiveAlarmsN:  len(activeAlarms),
	}
}

func normalizeProviderFixtureOptions(options ProviderFixtureOptions) ProviderFixtureOptions {
	if strings.TrimSpace(options.SourceName) == "" {
		options.SourceName = "provider"
	}
	if strings.TrimSpace(options.Admin.ID) == "" {
		options.Admin = UnifiedUser{
			ID:        "100",
			Email:     "operator@example.com",
			Role:      "ADMIN",
			FirstName: "Адміністратор",
			LastName:  "Provider",
			PultID:    1,
			BasketID:  100,
		}
	}
	if options.Admin.PultID == 0 {
		options.Admin.PultID = 1
	}
	if len(options.Managers) == 0 {
		options.Managers = []UnifiedManager{{ID: 1, Name: "МГР", Number: 1, UserIDs: []string{options.Admin.ID}}}
	}
	if len(options.Pults) == 0 {
		options.Pults = []UnifiedPult{{ID: options.Admin.PultID, Name: "Unified ARC", Number: options.Admin.PultID, UserIDs: []string{options.Admin.ID}}}
	}
	if strings.TrimSpace(options.DeviceType.Type) == "" {
		options.DeviceType = UnifiedDeviceType{
			Type:      defaultProviderDeviceType,
			NameUK:    "Уніфіковане джерело",
			NameRU:    "Унифицированный источник",
			NameEN:    "Unified provider",
			MaxLines:  999,
			MaxGroups: 999,
		}
	}
	if options.DeviceType.MaxLines == 0 {
		options.DeviceType.MaxLines = 999
	}
	if options.DeviceType.MaxGroups == 0 {
		options.DeviceType.MaxGroups = 999
	}
	return options
}

func providerObjectName(object models.Object) string {
	return strings.TrimSpace(object.Name)
}

func providerObjectNumber(object models.Object) string {
	number := strings.TrimSpace(object.DisplayNumber)
	if number != "" {
		return number
	}
	return strconv.Itoa(object.ID)
}

func providerPublicObjectIDs(objects []models.Object) map[int]int {
	result := make(map[int]int, len(objects))
	used := make(map[int]struct{}, len(objects))
	next := 1
	for _, object := range objects {
		if object.ID == 0 {
			continue
		}
		candidate := providerPublicObjectIDCandidate(object)
		if candidate <= 0 {
			candidate = next
		}
		for {
			if _, occupied := used[candidate]; !occupied {
				break
			}
			candidate++
		}
		result[object.ID] = candidate
		used[candidate] = struct{}{}
		if candidate >= next {
			next = candidate + 1
		}
	}
	return result
}

func providerPublicObjectIDCandidate(object models.Object) int {
	if number, err := strconv.Atoi(strings.TrimSpace(object.DisplayNumber)); err == nil && number > 0 {
		return number
	}
	if digits := onlyDigits(object.DisplayNumber); digits != "" {
		if number, err := strconv.Atoi(strings.TrimLeft(digits, "0")); err == nil && number > 0 {
			return 100000 + number
		}
	}
	if object.ID > 0 && object.ID < 1_000_000_000 {
		return object.ID
	}
	return 0
}

func providerObjectDescription(object models.Object) string {
	parts := []string{}
	for _, value := range []string{object.Description1, object.Notes1, object.Phone, object.Phones1} {
		value = strings.TrimSpace(value)
		if value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "\n")
}

func providerObjectType(sourceName string) string {
	sourceName = strings.ToLower(strings.TrimSpace(sourceName))
	if sourceName == "" {
		return "provider"
	}
	return sourceName
}

func providerRoomID(objectID int) string {
	return fmt.Sprintf("%d01", objectID)
}

func providerRoomName(object models.Object) string {
	if len(object.Groups) == 1 && strings.TrimSpace(object.Groups[0].Name) != "" {
		return strings.TrimSpace(object.Groups[0].Name)
	}
	return "Основне приміщення"
}

func providerRoomDescription(object models.Object) string {
	if location := strings.TrimSpace(object.Location1); location != "" {
		return location
	}
	return providerRoomName(object)
}

func providerRoomsFromObject(object models.Object, publicObjectID int, fallbackRoomID string) []UnifiedRoom {
	if len(object.Groups) == 0 {
		return []UnifiedRoom{{
			ID:          fallbackRoomID,
			Name:        providerRoomName(object),
			Description: providerRoomDescription(object),
			GroupNumber: 1,
		}}
	}
	rooms := make([]UnifiedRoom, 0, len(object.Groups))
	for _, group := range object.Groups {
		groupNumber := group.Number
		if groupNumber == 0 {
			groupNumber = len(rooms) + 1
		}
		roomID := strings.TrimSpace(group.RoomID)
		if roomID == "" {
			roomID = providerRoomIDForGroup(publicObjectID, groupNumber)
		}
		name := strings.TrimSpace(group.RoomName)
		if name == "" {
			name = strings.TrimSpace(group.Name)
		}
		if name == "" {
			name = fmt.Sprintf("Група %d", groupNumber)
		}
		rooms = append(rooms, UnifiedRoom{
			ID:          roomID,
			Name:        name,
			Description: name,
			GroupNumber: groupNumber,
		})
	}
	return rooms
}

func providerRoomIDForGroup(objectID int, groupNumber int) string {
	if groupNumber <= 0 {
		groupNumber = 1
	}
	return fmt.Sprintf("%d%02d", objectID, groupNumber)
}

func providerDeviceID(objectID int) int {
	return objectID*10 + 1
}

func providerDeviceNumber(object models.Object) int {
	if number, err := strconv.Atoi(strings.TrimSpace(object.DisplayNumber)); err == nil && number > 0 {
		return number
	}
	if digits := onlyDigits(object.DisplayNumber); digits != "" {
		if number, err := strconv.Atoi(strings.TrimLeft(digits, "0")); err == nil && number > 0 {
			return 100000 + number
		}
	}
	if object.ID >= 1_000_000_000 {
		return providerPublicObjectIDCandidate(object)
	}
	return object.ID
}

func providerDeviceType(object models.Object, fallback UnifiedDeviceType) UnifiedDeviceType {
	name := strings.TrimSpace(object.DeviceType)
	if name == "" {
		name = strings.TrimSpace(object.PanelMark)
	}
	if name == "" {
		return fallback
	}
	return UnifiedDeviceType{
		Type:      name,
		NameUK:    name,
		NameRU:    name,
		NameEN:    name,
		MaxLines:  fallback.MaxLines,
		MaxGroups: fallback.MaxGroups,
	}
}

func providerDeviceName(object models.Object) string {
	for _, value := range []string{object.DeviceType, object.PanelMark, object.DisplayNumber} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return fmt.Sprintf("Device %d", object.ID)
}

func providerDeviceTimeout(object models.Object) int {
	if object.AutoTestHours > 0 {
		return object.AutoTestHours * 3600
	}
	if object.TestTime > 0 {
		return int(object.TestTime * 60)
	}
	return 240
}

func providerSignalLevel(object models.Object) int {
	if object.GSMLevel > 0 {
		return clampPercent(object.GSMLevel)
	}
	if value, ok := firstIntInText(object.SignalStrength); ok {
		if value < 0 {
			return clampPercent((value + 113) * 100 / 62)
		}
		return clampPercent(value)
	}
	return 0
}

func firstIntInText(value string) (int, bool) {
	value = strings.TrimSpace(value)
	sign := 1
	number := 0
	inNumber := false
	for _, r := range value {
		switch {
		case r == '-' && !inNumber:
			sign = -1
			inNumber = true
		case r >= '0' && r <= '9':
			number = number*10 + int(r-'0')
			inNumber = true
		case inNumber:
			return sign * number, true
		}
	}
	if inNumber {
		return sign * number, true
	}
	return 0, false
}

func clampPercent(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func providerTechnicianID(object models.Object) string {
	for _, value := range []string{object.TechnicianID, object.TechnicianName} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func providerDisconnected(object models.Object) bool {
	return object.ConnectionStatusValue() == models.ConnectionStatusOffline || object.Status == models.StatusOffline
}

func providerOfflineMS(object models.Object) int64 {
	if !providerDisconnected(object) {
		return -1
	}
	if !object.LastMessageTime.IsZero() {
		return object.LastMessageTime.UnixMilli()
	}
	return time.Now().UnixMilli()
}

func providerLinesFromZones(zones []models.Zone, object models.Object, publicObjectID int, fallbackRoomID string) []UnifiedLine {
	if len(zones) == 0 {
		return []UnifiedLine{{
			ID:          1,
			Number:      1,
			AdapterType: "SYS",
			LineType:    providerDefaultLineType(object),
			Description: "Віртуальна зона",
			GroupNumber: 1,
			RoomID:      fallbackRoomID,
		}}
	}
	lines := make([]UnifiedLine, 0, len(zones))
	for index, zone := range zones {
		number := zone.Number
		if number == 0 {
			number = index + 1
		}
		groupNumber := zone.GroupNumber
		if groupNumber == 0 {
			groupNumber = 1
		}
		description := strings.TrimSpace(zone.Name)
		if description == "" {
			description = fmt.Sprintf("Зона %d", number)
		}
		lines = append(lines, UnifiedLine{
			ID:          number,
			Number:      number,
			AdapterType: "SYS",
			LineType:    providerLineType(zone, object),
			Description: description,
			GroupNumber: groupNumber,
			RoomID:      providerRoomIDForGroup(publicObjectID, groupNumber),
			IsBroken:    providerLineBroken(zone),
			IsBlocked:   zone.IsBypassed,
		})
	}
	return lines
}

func providerLineType(zone models.Zone, object models.Object) string {
	if providerObjectIsFire(object) {
		return "FIRE"
	}
	text := strings.ToLower(zone.SensorType + " " + zone.Name)
	switch {
	case strings.Contains(text, "трив") || strings.Contains(text, "panic"):
		return "ALM_BTN"
	case strings.Contains(text, "пож") || strings.Contains(text, "fire") || strings.Contains(text, "дим"):
		return "FIRE"
	default:
		return "NORMAL"
	}
}

func providerDefaultLineType(object models.Object) string {
	if providerObjectIsFire(object) {
		return "FIRE"
	}
	return "NORMAL"
}

func providerObjectIsFire(object models.Object) bool {
	text := strings.ToLower(strings.Join([]string{
		object.DeviceType,
		object.PanelMark,
		object.Name,
		object.Description1,
		object.Notes1,
	}, " "))
	return strings.Contains(text, "пож") ||
		strings.Contains(text, "fire") ||
		strings.Contains(text, "дим")
}

func providerLineBroken(zone models.Zone) int {
	switch zone.Status {
	case models.ZoneBreak, models.ZoneShort:
		return 1
	default:
		return 0
	}
}

func providerResponsibleID(sourceName string, objectID, index int, contact models.Contact) string {
	phone := onlyDigits(contact.Phone)
	if phone != "" {
		return fmt.Sprintf("%s-%s", strings.ToLower(strings.TrimSpace(sourceName)), phone)
	}
	return fmt.Sprintf("%s-%d-%d", strings.ToLower(strings.TrimSpace(sourceName)), objectID, index)
}

func providerResponder(userID string, contact models.Contact, pultID int) UnifiedUser {
	firstName, lastName := splitContactName(contact.Name)
	return UnifiedUser{
		ID:          userID,
		Role:        "IN_CHARGE",
		FirstName:   firstName,
		LastName:    lastName,
		PultID:      pultID,
		PhoneNumber: strings.TrimSpace(contact.Phone),
		BasketID:    stablePositiveID(userID),
	}
}

func splitContactName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "Відповідальний", ""
	}
	parts := strings.Fields(name)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return strings.Join(parts[1:], " "), parts[0]
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, r := range value {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func stablePositiveID(value string) int {
	hash := 0
	for _, r := range value {
		hash = hash*31 + int(r)
	}
	if hash < 0 {
		hash = -hash
	}
	if hash == 0 {
		return 1
	}
	return hash
}

func providerActiveAlarms(alarms []models.Alarm, publicObjectIDs map[int]int) []UnifiedAlarm {
	result := make([]UnifiedAlarm, 0, len(alarms))
	for _, alarm := range alarms {
		if alarm.ObjectID == 0 || alarm.IsProcessed {
			continue
		}
		publicObjectID := publicObjectIDs[alarm.ObjectID]
		if publicObjectID == 0 {
			publicObjectID = alarm.ObjectID
		}
		eventName, code, typeEvent, additionalType := providerAlarmEvent(alarm.Type)
		lineNumber := alarm.ZoneNumber
		if lineNumber == 0 {
			lineNumber = 1
		}
		timeMS := time.Now().UnixMilli()
		if !alarm.Time.IsZero() {
			timeMS = alarm.Time.UnixMilli()
		}
		lastAction := "GRD_OBJ_NOTIF"
		userID := "0"
		if alarm.IsInProgress {
			lastAction = "GRD_OBJ_PICK"
			userID = strings.TrimSpace(alarm.InProgressUser)
			if userID == "" {
				userID = strings.TrimSpace(alarm.InProgressBy)
			}
			if userID == "" {
				userID = "0"
			}
		}
		result = append(result, UnifiedAlarm{
			ObjectID:       publicObjectID,
			DeviceID:       providerDeviceID(publicObjectID),
			DeviceNumber:   providerAlarmDeviceNumber(alarm, publicObjectID),
			Time:           timeMS,
			UserID:         userID,
			AlarmType:      "ALARM_TYPE_DEVICE",
			EventCode:      code,
			EventType:      typeEvent,
			AdditionalType: additionalType,
			EventName:      eventName,
			Details:        providerAlarmDetails(alarm),
			LineNumber:     lineNumber,
			LastAction:     lastAction,
			SourceEvents:   providerAlarmSourceEvents(alarm, publicObjectID, providerDeviceID(publicObjectID), providerAlarmDeviceNumber(alarm, publicObjectID)),
		})
	}
	return result
}

func providerJournalEvents(events []models.Event, publicObjectIDs map[int]int) []UnifiedEvent {
	result := make([]UnifiedEvent, 0, len(events))
	for _, event := range events {
		if event.ObjectID == 0 {
			continue
		}
		publicObjectID := publicObjectIDs[event.ObjectID]
		if publicObjectID == 0 {
			publicObjectID = event.ObjectID
		}
		timeMS := time.Now().UnixMilli()
		if !event.Time.IsZero() {
			timeMS = event.Time.UnixMilli()
		}
		eventName, code, typeEvent, additionalType, isAlarm := providerEventType(event.Type)
		lineNumber := event.ZoneNumber
		result = append(result, UnifiedEvent{
			ObjectID:       publicObjectID,
			DeviceID:       providerDeviceID(publicObjectID),
			DeviceNumber:   providerEventDeviceNumber(event, publicObjectID),
			Time:           timeMS,
			EventCode:      code,
			EventType:      typeEvent,
			AdditionalType: additionalType,
			EventName:      eventName,
			LineNumber:     lineNumber,
			Details:        strings.TrimSpace(firstProviderString(event.Details, event.GetTypeDisplay())),
			IsAlarm:        isAlarm,
		})
	}
	return result
}

func providerAlarmSourceEvents(alarm models.Alarm, publicObjectID int, deviceID int, deviceNumber int) []UnifiedEvent {
	if len(alarm.SourceMsgs) == 0 {
		return nil
	}
	result := make([]UnifiedEvent, 0, len(alarm.SourceMsgs))
	for _, msg := range alarm.SourceMsgs {
		eventName, code, typeEvent, additionalType := providerAlarmEvent(alarm.Type)
		lineNumber := msg.Number
		if lineNumber == 0 {
			lineNumber = alarm.ZoneNumber
		}
		timeMS := time.Now().UnixMilli()
		if !msg.Time.IsZero() {
			timeMS = msg.Time.UnixMilli()
		}
		details := strings.TrimSpace(msg.Details)
		if details == "" {
			details = providerAlarmDetails(alarm)
		}
		result = append(result, UnifiedEvent{
			ObjectID:       publicObjectID,
			DeviceID:       deviceID,
			DeviceNumber:   deviceNumber,
			Time:           timeMS,
			EventCode:      code,
			EventType:      typeEvent,
			AdditionalType: additionalType,
			EventName:      eventName,
			LineNumber:     lineNumber,
			Details:        details,
			IsAlarm:        msg.IsAlarm,
		})
	}
	return result
}

func providerAlarmDetails(alarm models.Alarm) string {
	for _, value := range []string{
		alarm.Details,
		alarm.ZoneName,
	} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return (&alarm).GetTypeDisplay()
}

func providerAlarmDeviceNumber(alarm models.Alarm, fallback int) int {
	if number, err := strconv.Atoi(strings.TrimSpace(alarm.ObjectNumber)); err == nil && number > 0 {
		return number
	}
	if digits := onlyDigits(alarm.ObjectNumber); digits != "" {
		if number, err := strconv.Atoi(strings.TrimLeft(digits, "0")); err == nil && number > 0 {
			return number
		}
	}
	return fallback
}

func providerEventDeviceNumber(event models.Event, fallback int) int {
	if number, err := strconv.Atoi(strings.TrimSpace(event.ObjectNumber)); err == nil && number > 0 {
		return number
	}
	if digits := onlyDigits(event.ObjectNumber); digits != "" {
		if number, err := strconv.Atoi(strings.TrimLeft(digits, "0")); err == nil && number > 0 {
			return number
		}
	}
	return fallback
}

func firstProviderString(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func providerAlarmEvent(alarmType models.AlarmType) (string, int, string, int) {
	switch alarmType {
	case models.AlarmFire, models.AlarmFireTrouble:
		return "UNIFIED_FIRE_ALARM", 110, "E", 1
	case models.AlarmBurglary:
		return "UNIFIED_BURGLARY_ALARM", 130, "E", 1
	case models.AlarmPanic:
		return "UNIFIED_PANIC_ALARM", 120, "E", 1
	case models.AlarmTamper:
		return "UNIFIED_TAMPER_ALARM", 383, "E", 1
	case models.AlarmMedical:
		return "UNIFIED_MEDICAL_ALARM", 100, "E", 1
	case models.AlarmGas:
		return "UNIFIED_GAS_ALARM", 151, "E", 1
	case models.AlarmOffline:
		return "UNIFIED_DEVICE_OFFLINE", 359, "E", 0
	case models.AlarmPowerFail, models.AlarmAcTrouble:
		return "UNIFIED_AC_LOSS", 301, "E", 0
	default:
		return "UNIFIED_GENERIC_ALARM", 999, "E", 1
	}
}

func providerEventType(eventType models.EventType) (string, int, string, int, bool) {
	switch eventType {
	case models.EventFire:
		return "UNIFIED_FIRE_ALARM", 110, "E", 1, true
	case models.EventBurglary:
		return "UNIFIED_BURGLARY_ALARM", 130, "E", 1, true
	case models.EventPanic:
		return "UNIFIED_PANIC_ALARM", 120, "E", 1, true
	case models.EventTamper:
		return "UNIFIED_TAMPER_ALARM", 383, "E", 1, true
	case models.EventMedical:
		return "UNIFIED_MEDICAL_ALARM", 100, "E", 1, true
	case models.EventGas:
		return "UNIFIED_GAS_ALARM", 151, "E", 1, true
	case models.EventOffline:
		return "UNIFIED_DEVICE_OFFLINE", 359, "E", 0, true
	case models.EventOnline:
		return "UNIFIED_DEVICE_ONLINE", 359, "R", 0, false
	case models.EventPowerFail:
		return "UNIFIED_AC_LOSS", 301, "E", 0, false
	case models.EventPowerOK:
		return "UNIFIED_AC_RESTORE", 301, "R", 0, false
	case models.EventRestore:
		return "UNIFIED_FIRE_RESTORE", 110, "R", 1, false
	case models.EventArm:
		return "UNIFIED_ARM", 401, "E", 0, false
	case models.EventDisarm:
		return "UNIFIED_DISARM", 402, "E", 0, false
	case models.EventTest:
		return "UNIFIED_TEST", 602, "E", 0, false
	case models.EventFault, models.EventBatteryLow, models.EventDeviceBlocked:
		return "UNIFIED_DEVICE_OFFLINE", 359, "E", 0, true
	default:
		return "UNIFIED_GENERIC_ALARM", 999, "E", 1, true
	}
}

func defaultProviderEventTypes(deviceType string) []UnifiedEventType {
	return cidEventTypes(deviceType)
}

func providerDisconnectedDevices(devices []UnifiedDevice) []UnifiedDisconnectedDevice {
	result := make([]UnifiedDisconnectedDevice, 0)
	for _, device := range devices {
		if !device.Disconnected {
			continue
		}
		result = append(result, UnifiedDisconnectedDevice{
			ObjectID:     device.ObjectID,
			DeviceID:     device.ID,
			Number:       device.Number,
			Offline:      device.Offline,
			Disconnected: true,
		})
	}
	return result
}

func providerOfflineObjectCount(devices []UnifiedDevice) int {
	count := 0
	for _, device := range devices {
		if device.Disconnected {
			count++
		}
	}
	return count
}
