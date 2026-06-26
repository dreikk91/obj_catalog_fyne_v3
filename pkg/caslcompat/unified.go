package caslcompat

import "fmt"

type UnifiedFixture struct {
	Admin          UnifiedUser
	Responders     []UnifiedUser
	Managers       []UnifiedManager
	Pults          []UnifiedPult
	Objects        []UnifiedObject
	Devices        []UnifiedDevice
	DeviceTypes    []UnifiedDeviceType
	EventTypes     []UnifiedEventType
	ActiveAlarms   []UnifiedAlarm
	JournalEvents  []UnifiedEvent
	Disconnected   []UnifiedDisconnectedDevice
	CountOfRooms   int
	TotalObjects   int
	OfflineObjects int
	ActiveAlarmsN  int
}

type UnifiedUser struct {
	ID          string
	Email       string
	Role        string
	FirstName   string
	LastName    string
	MiddleName  string
	PultID      int
	PhoneNumber string
	BasketID    int
}

type UnifiedManager struct {
	ID          int
	Name        string
	Number      int
	PhoneNumber string
	UserIDs     []string
}

type UnifiedPult struct {
	ID      int
	Name    string
	Number  int
	UserIDs []string
}

type UnifiedObject struct {
	ID             int
	DisplayNumber  string
	Name           string
	Address        string
	Lat            string
	Long           string
	Description    string
	Contract       string
	Status         string
	ObjectType     string
	ReactingPultID int
	ResponsibleIDs []string
	Room           UnifiedRoom
	Rooms          []UnifiedRoom
}

type UnifiedRoom struct {
	ID          string
	Name        string
	Description string
	RTSP        string
	GroupNumber int
}

type UnifiedDevice struct {
	ID           int
	ObjectID     int
	Number       int
	Name         string
	Type         string
	SignalLevel  int
	Timeout      int
	SIM1         string
	SIM2         string
	TechnicianID string
	Enabled      int64
	Offline      int64
	Disconnected bool
	Lines        []UnifiedLine
}

type UnifiedLine struct {
	ID            int
	Number        int
	AdapterType   string
	LineType      string
	Description   string
	GroupNumber   int
	RoomID        string
	AdapterNumber int
	IsBroken      int
	IsBlocked     bool
}

type UnifiedDeviceType struct {
	Type      string
	NameUK    string
	NameRU    string
	NameEN    string
	MaxLines  int
	MaxGroups int
}

type UnifiedEventType struct {
	DeviceType     string
	Code           int
	TypeEvent      string
	AdditionalType int
	EventByUser    string
	IsAlarm        int
	LangUK         string
	LangRU         string
	LangEN         string
}

type UnifiedAlarm struct {
	ObjectID       int
	DeviceID       int
	DeviceNumber   int
	Time           int64
	UserID         string
	AlarmType      string
	EventCode      int
	EventType      string
	AdditionalType int
	EventName      string
	Details        string
	LineNumber     int
	LastAction     string
	SourceEvents   []UnifiedEvent
}

type UnifiedEvent struct {
	ObjectID       int
	DeviceID       int
	DeviceNumber   int
	Time           int64
	EventCode      int
	EventType      string
	AdditionalType int
	EventName      string
	LineNumber     int
	Details        string
	IsAlarm        bool
}

type UnifiedDisconnectedDevice struct {
	ObjectID     int
	DeviceID     int
	Number       int
	Offline      int64
	Disconnected bool
}

func defaultUnifiedFixture() UnifiedFixture {
	const (
		phoenixType = "PHOENIXDB_GENERIC"
		mostType    = "MOST_GENERIC"
	)

	return UnifiedFixture{
		Admin: UnifiedUser{
			ID:         "100",
			Email:      "operator@example.com",
			Role:       "ADMIN",
			FirstName:  "Адміністратор",
			LastName:   "Fixtures",
			MiddleName: "",
			PultID:     1,
			BasketID:   100,
		},
		Responders: []UnifiedUser{
			{ID: "700101", Role: "IN_CHARGE", FirstName: "Відповідальний", LastName: "PhoenixDB", PhoneNumber: "+380000000101", BasketID: 700101},
			{ID: "800201", Role: "IN_CHARGE", FirstName: "Відповідальний", LastName: "Мост", PhoneNumber: "+380000000201", BasketID: 800201},
		},
		Managers: []UnifiedManager{
			{ID: 1, Name: "МГР fixture", Number: 1, UserIDs: []string{"100"}},
		},
		Pults: []UnifiedPult{
			{ID: 1, Name: "Fixture ARC", Number: 1, UserIDs: []string{"100"}},
		},
		Objects: []UnifiedObject{
			{
				ID:             7001001,
				DisplayNumber:  "7001001",
				Name:           "PhoenixDB: Магазин 1",
				Address:        "Київ, вул. Тестова, 1",
				Lat:            "50.4501",
				Long:           "30.5234",
				Description:    "Fixture object from PhoenixDB projection",
				Contract:       "PHX-001",
				ObjectType:     "commercial",
				ReactingPultID: 1,
				ResponsibleIDs: []string{"700101"},
				Room:           UnifiedRoom{ID: "700100101", Name: "Fixture room", Description: "Fixture room"},
			},
			{
				ID:             8002001,
				DisplayNumber:  "8002001",
				Name:           "Мост: Склад 2",
				Address:        "Львів, вул. Прикладна, 2",
				Description:    "Fixture object from Мост projection",
				Contract:       "MOST-002",
				ObjectType:     "warehouse",
				ReactingPultID: 1,
				ResponsibleIDs: []string{"800201"},
				Room:           UnifiedRoom{ID: "800200101", Name: "Fixture room", Description: "Fixture room"},
			},
		},
		Devices: []UnifiedDevice{
			{
				ID:           7101001,
				ObjectID:     7001001,
				Number:       7001001,
				Name:         "PhoenixDB virtual device",
				Type:         phoenixType,
				Timeout:      240,
				Enabled:      -1,
				Offline:      -1713970000000,
				Disconnected: false,
				Lines: []UnifiedLine{
					{ID: 1, Number: 1, AdapterType: "SYS", LineType: "ALM_BTN", Description: "Тривожна кнопка", GroupNumber: 1, RoomID: "700100101"},
					{ID: 2, Number: 2, AdapterType: "SYS", LineType: "FIRE", Description: "Пожежна зона", GroupNumber: 1, RoomID: "700100101"},
				},
			},
			{
				ID:           8102001,
				ObjectID:     8002001,
				Number:       8002001,
				Name:         "Мост virtual device",
				Type:         mostType,
				Timeout:      240,
				Enabled:      -1,
				Offline:      1713970500000,
				Disconnected: false,
				Lines: []UnifiedLine{
					{ID: 1, Number: 1, AdapterType: "SYS", LineType: "TECH", Description: "Технічна зона", GroupNumber: 1, RoomID: "800200101"},
				},
			},
		},
		DeviceTypes: []UnifiedDeviceType{
			{Type: phoenixType, NameUK: "PhoenixDB", NameRU: "PhoenixDB", NameEN: "PhoenixDB", MaxLines: 999, MaxGroups: 999},
			{Type: mostType, NameUK: "Мост", NameRU: "Мост", NameEN: "Most", MaxLines: 999, MaxGroups: 999},
		},
		EventTypes: buildFixtureEventTypes(phoenixType, mostType),
		ActiveAlarms: []UnifiedAlarm{
			{ObjectID: 7001001, DeviceID: 7101001, DeviceNumber: 7001001, Time: 1777058079975, UserID: "0", AlarmType: "ALARM_TYPE_DEVICE", EventCode: 110, EventType: "E", AdditionalType: 1, EventName: "PHOENIX_FIRE_ALARM", LineNumber: 2, LastAction: "GRD_OBJ_NOTIF"},
		},
		Disconnected:   []UnifiedDisconnectedDevice{{ObjectID: 8002001, DeviceID: 8102001, Number: 8002001, Offline: 1713970500000}},
		CountOfRooms:   2,
		TotalObjects:   2,
		OfflineObjects: 1,
		ActiveAlarmsN:  1,
	}
}

func buildFixtureFromUnified(source UnifiedFixture) Fixture {
	users := make([]FixtureUser, 0, 1+len(source.Responders))
	users = append(users, fixtureUserFromUnified(source.Admin))
	for _, user := range source.Responders {
		users = append(users, fixtureUserFromUnified(user))
	}

	objects := make([]FixtureObject, 0, len(source.Objects))
	for _, object := range source.Objects {
		objects = append(objects, FixtureObject{
			ObjID:          object.ID,
			DisplayNumber:  object.DisplayNumber,
			Name:           object.Name,
			Address:        object.Address,
			Lat:            object.Lat,
			Long:           object.Long,
			Description:    object.Description,
			Contract:       object.Contract,
			Status:         object.Status,
			ObjectType:     object.ObjectType,
			ReactingPultID: object.ReactingPultID,
		})
	}

	devices := make([]FixtureDevice, 0, len(source.Devices))
	for _, device := range source.Devices {
		devices = append(devices, fixtureDeviceFromUnified(device))
	}

	translators := messageTranslatorsFromUnified(source.EventTypes)
	dictionaryAdd := dictionaryAddFromUnified(source.EventTypes)

	return Fixture{
		User:                fixtureUserFromUnified(source.Admin),
		Users:               users,
		Managers:            fixtureManagersFromUnified(source.Managers),
		Pults:               fixturePultsFromUnified(source.Pults),
		Objects:             objects,
		Devices:             devices,
		Connections:         fixtureConnections(objects, devices),
		Rooms:               fixtureRoomsFromUnified(source.Objects, source.Devices),
		Dictionary:          fixtureDictionaryFromUnified(source.DeviceTypes, source.EventTypes, dictionaryAdd, translators),
		MessageTranslators:  translators,
		GeneralTape:         fixtureGeneralTapeFromUnified(source.ActiveAlarms, source.Objects),
		GeneralTapeItems:    fixtureGeneralTapeItemsFromUnified(source.ActiveAlarms, source.JournalEvents),
		DisconnectedDevices: fixtureDisconnectedFromUnified(source.Disconnected),
		Statistics: map[string]any{
			"groupStatistics": fixtureGroupStatisticsFromUnified(source.Devices),
			"countOfRooms":    source.CountOfRooms,
			"total":           source.TotalObjects,
			"offline":         source.OfflineObjects,
			"active_alarm":    source.ActiveAlarmsN,
		},
		AlarmEvents: fixtureAlarmEventsFromUnified(source.EventTypes),
	}
}

func fixtureUserFromUnified(user UnifiedUser) FixtureUser {
	phoneNumber := user.PhoneNumber
	return FixtureUser{
		UserID:     user.ID,
		Email:      user.Email,
		Role:       user.Role,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		MiddleName: user.MiddleName,
		PultID:     user.PultID,
		Images:     []any{nil},
		PhoneNumbers: []map[string]any{
			{"active": true, "number": phoneNumber},
		},
		Tag:       "",
		DeviceIDs: []any{},
		OneboxID:  "",
		UserNotif: map[string]any{},
		BasketID:  user.BasketID,
	}
}

func fixtureDeviceFromUnified(device UnifiedDevice) FixtureDevice {
	lines := make(map[string]FixtureLine, len(device.Lines))
	for _, line := range device.Lines {
		lines[fmt.Sprintf("%d", line.Number)] = FixtureLine{
			LineID:        line.ID,
			LineNumber:    line.Number,
			AdapterType:   line.AdapterType,
			LineType:      line.LineType,
			Description:   line.Description,
			GroupNumber:   line.GroupNumber,
			RoomID:        line.RoomID,
			AdapterNumber: line.AdapterNumber,
			IsBroken:      line.IsBroken,
			IsBlocked:     line.IsBlocked,
		}
	}
	return FixtureDevice{
		DeviceID:          device.ID,
		ObjID:             device.ObjectID,
		Number:            device.Number,
		Name:              device.Name,
		Type:              device.Type,
		DeviceType:        device.Type,
		SignalLevel:       device.SignalLevel,
		Timeout:           device.Timeout,
		Sim1:              device.SIM1,
		Sim2:              device.SIM2,
		TechnicianID:      device.TechnicianID,
		MoreAlarmTime:     []map[string]any{},
		IgnoringAlarmTime: []map[string]any{},
		Enabled:           device.Enabled,
		Offline:           device.Offline,
		Disconnected:      device.Disconnected,
		Lines:             lines,
	}
}

func fixtureManagersFromUnified(managers []UnifiedManager) []map[string]any {
	result := make([]map[string]any, 0, len(managers))
	for _, manager := range managers {
		result = append(result, map[string]any{
			"mgr_id":       manager.ID,
			"name":         manager.Name,
			"number":       manager.Number,
			"phone_number": manager.PhoneNumber,
			"users":        manager.UserIDs,
		})
	}
	return result
}

func fixturePultsFromUnified(pults []UnifiedPult) []map[string]any {
	result := make([]map[string]any, 0, len(pults))
	for _, pult := range pults {
		result = append(result, map[string]any{
			"pult_id": pult.ID,
			"name":    pult.Name,
			"number":  pult.Number,
			"users":   pult.UserIDs,
		})
	}
	return result
}

func fixtureConnections(objects []FixtureObject, devices []FixtureDevice) []FixtureConnection {
	result := make([]FixtureConnection, 0, len(devices))
	for _, device := range devices {
		for _, object := range objects {
			if object.ObjID == device.ObjID {
				result = append(result, FixtureConnection{GuardedObject: object, Device: device})
				break
			}
		}
	}
	return result
}

func fixtureRoomsFromUnified(objects []UnifiedObject, devices []UnifiedDevice) []FixtureRoom {
	rooms := make([]FixtureRoom, 0, len(objects))
	for _, object := range objects {
		objectRooms := object.Rooms
		if len(objectRooms) == 0 {
			objectRooms = []UnifiedRoom{object.Room}
		}
		for _, unifiedRoom := range objectRooms {
			roomID := unifiedRoom.ID
			if roomID == "" {
				roomID = fmt.Sprintf("%d%02d", object.ID, maxInt(1, unifiedRoom.GroupNumber))
			}
			room := FixtureRoom{
				RoomID:      roomID,
				ObjID:       fmt.Sprintf("%d", object.ID),
				Name:        unifiedRoom.Name,
				Description: unifiedRoom.Description,
				RTSP:        unifiedRoom.RTSP,
				Images:      []any{},
				Lines:       map[string]FixtureRoomLine{},
				Users:       fixtureRoomUsersFromUnified(object.ResponsibleIDs),
			}
			if room.Name == "" {
				room.Name = "Fixture room"
			}
			if room.Description == "" {
				room.Description = room.Name
			}

			for _, device := range devices {
				if device.ObjectID != object.ID {
					continue
				}
				for _, line := range device.Lines {
					lineRoomID := line.RoomID
					if lineRoomID == "" {
						lineRoomID = roomID
					}
					if lineRoomID != roomID {
						continue
					}
					room.Lines[fmt.Sprintf("%d", line.Number)] = FixtureRoomLine{
						LineID:        line.ID,
						LineNumber:    line.Number,
						AdapterType:   line.AdapterType,
						LineType:      line.LineType,
						Description:   line.Description,
						GroupNumber:   line.GroupNumber,
						RoomID:        lineRoomID,
						DeviceID:      device.ID,
						DeviceNumber:  device.Number,
						AdapterNumber: line.AdapterNumber,
						IsBroken:      line.IsBroken,
						IsBlocked:     line.IsBlocked,
					}
				}
			}
			rooms = append(rooms, room)
		}
	}
	return rooms
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func fixtureRoomUsersFromUnified(userIDs []string) []FixtureRoomUser {
	users := make([]FixtureRoomUser, 0, len(userIDs))
	for index, userID := range userIDs {
		users = append(users, FixtureRoomUser{
			UserID:   userID,
			Priority: index + 1,
			HozNum:   fmt.Sprintf("%d", index+1),
		})
	}
	return users
}

func messageTranslatorsFromUnified(events []UnifiedEventType) map[string][]FixtureTranslatorRow {
	result := make(map[string][]FixtureTranslatorRow)
	for _, event := range events {
		result[event.DeviceType] = append(result[event.DeviceType], FixtureTranslatorRow{
			TypeProtocol:   event.DeviceType,
			Code:           event.Code,
			TypeEvent:      event.TypeEvent,
			AdditionalType: event.AdditionalType,
			EventByUser:    event.EventByUser,
			IsAlarm:        event.IsAlarm,
		})
	}
	return result
}

func dictionaryAddFromUnified(events []UnifiedEventType) []map[string]string {
	result := make([]map[string]string, 0, len(events))
	seen := map[string]struct{}{}
	for _, event := range events {
		if _, ok := seen[event.EventByUser]; ok {
			continue
		}
		seen[event.EventByUser] = struct{}{}
		result = append(result, map[string]string{
			"name":    event.EventByUser,
			"lang_en": event.LangEN,
			"lang_ru": event.LangRU,
			"lang_uk": event.LangUK,
		})
	}
	return result
}

func fixtureDictionaryFromUnified(deviceTypes []UnifiedDeviceType, events []UnifiedEventType, dictionaryAdd []map[string]string, translators map[string][]FixtureTranslatorRow) map[string]any {
	devices := make([]map[string]any, 0, len(deviceTypes))
	deviceTypeIDs := make([]string, 0, len(deviceTypes))
	for _, deviceType := range deviceTypes {
		devices = append(devices, map[string]any{
			"type":       deviceType.Type,
			"max_lines":  deviceType.MaxLines,
			"max_groups": deviceType.MaxGroups,
			"lang_uk":    deviceType.NameUK,
			"lang_ru":    deviceType.NameRU,
			"lang_en":    deviceType.NameEN,
		})
		deviceTypeIDs = append(deviceTypeIDs, deviceType.Type)
	}

	translate := map[string]map[string]string{"uk": {}, "ru": {}, "en": {}}
	for _, deviceType := range deviceTypes {
		translate["uk"][deviceType.Type] = deviceType.NameUK
		translate["ru"][deviceType.Type] = deviceType.NameRU
		translate["en"][deviceType.Type] = deviceType.NameEN
	}
	for _, item := range dictionaryAdd {
		name := item["name"]
		translate["uk"][name] = item["lang_uk"]
		translate["ru"][name] = item["lang_ru"]
		translate["en"][name] = item["lang_en"]
	}

	userMsgs := make([]string, 0, len(dictionaryAdd))
	for _, item := range dictionaryAdd {
		userMsgs = append(userMsgs, item["name"])
	}

	return map[string]any{
		"devices":           devices,
		"device_types":      deviceTypeIDs,
		"user_device_types": deviceTypeIDs,
		"adapter_types":     []string{"SYS", "AD3L"},
		"user_msgs":         userMsgs,
		"line_types":        []string{"NORMAL", "ALM_BTN", "FIRE", "TECH"},
		"alarm_causes": []string{
			"CAUSES_CUSTOM_WINS",
			"CAUSES_ELECT_WORK",
			"CAUSES_HARDWARE_FAILED",
			"CAUSES_NO_POWER",
			"CAUSES_OTHER",
			"CAUSES_PENETRATION",
			"CAUSES_UNKNOWN",
		},
		"block_causes":      []string{"GROUP_ON", "GROUP_OFF", "REQUIRED_GROUP_ON", "REQUIRED_GROUP_ON_PLUS_GRUOP_OFF"},
		"off_hours_causes":  []string{"IGNORE_ALL_ALARM_EVENT", "IGNORE_KZ_LINEBRK"},
		"user_roles":        []string{"ADMIN", "BC_USER", "ENGINEER", "IN_CHARGE", "MANAGER", "MGR", "OPERATOR", "SENIOR_OPERATOR", "TECHNICIAN"},
		"more_alarm_time":   []string{},
		"ignore_alarm_time": []string{},
		"translate":         translate,
		"dictionary_add":    dictionaryAdd,
		"msg_translator":    translators,
	}
}

func fixtureGeneralTapeFromUnified(alarms []UnifiedAlarm, objects []UnifiedObject) []FixtureTapeRow {
	result := make([]FixtureTapeRow, 0, len(alarms))
	for _, alarm := range alarms {
		object, ok := findUnifiedObject(objects, alarm.ObjectID)
		if !ok {
			continue
		}
		result = append(result, FixtureTapeRow{
			Time:        alarm.Time,
			UserID:      alarm.UserID,
			ObjID:       alarm.ObjectID,
			DeviceID:    alarm.DeviceID,
			AlarmType:   alarm.AlarmType,
			MgrID:       nil,
			Name:        object.Name,
			Address:     object.Address,
			PultID:      fmt.Sprintf("%d", object.ReactingPultID),
			Description: object.Description,
			ReasonAlarm: fmt.Sprintf(`{"msg":"%s","num":%d,"additionalInfo":%d,"time":%d}`, firstUnifiedString(alarm.Details, alarm.EventName), alarm.LineNumber, alarm.AdditionalType, alarm.Time),
			LastAct:     alarm.LastAction,
		})
	}
	return result
}

func fixtureGeneralTapeItemsFromUnified(alarms []UnifiedAlarm, journalEvents []UnifiedEvent) map[string][]FixtureEvent {
	result := make(map[string][]FixtureEvent, len(alarms)+len(journalEvents))
	seen := make(map[string]struct{}, len(alarms)+len(journalEvents))
	for _, event := range journalEvents {
		appendFixtureEvent(result, seen, fixtureEventFromUnified(event))
	}
	for _, alarm := range alarms {
		key := fmt.Sprintf("%d", alarm.ObjectID)
		if len(alarm.SourceEvents) == 0 {
			appendFixtureEvent(result, seen, fixtureEventFromUnified(UnifiedEvent{
				ObjectID:       alarm.ObjectID,
				DeviceID:       alarm.DeviceID,
				DeviceNumber:   alarm.DeviceNumber,
				Time:           alarm.Time,
				EventCode:      alarm.EventCode,
				EventType:      alarm.EventType,
				AdditionalType: alarm.AdditionalType,
				EventName:      alarm.EventName,
				Details:        alarm.Details,
				LineNumber:     alarm.LineNumber,
				IsAlarm:        true,
			}))
		}
		for _, event := range alarm.SourceEvents {
			appendFixtureEvent(result, seen, fixtureEventFromUnified(event))
		}
		result[key] = append(result[key], FixtureEvent{
			ObjID:     alarm.ObjectID,
			DeviceID:  alarm.DeviceID,
			Time:      alarm.Time + 40429,
			Type:      "user_action",
			DictName:  alarm.LastAction,
			UserID:    alarm.UserID,
			Number:    nil,
			HozUserID: nil,
			ContactID: nil,
		})
	}
	return result
}

func appendFixtureEvent(items map[string][]FixtureEvent, seen map[string]struct{}, event FixtureEvent) {
	key := fmt.Sprintf("%d", event.ObjID)
	seenKey := fmt.Sprintf("%d:%d:%d:%d:%s", event.ObjID, event.DeviceID, event.Time, event.Code, event.TypeEvent)
	if _, ok := seen[seenKey]; ok {
		return
	}
	seen[seenKey] = struct{}{}
	items[key] = append(items[key], event)
}

func fixtureEventFromUnified(event UnifiedEvent) FixtureEvent {
	lineNumber := event.LineNumber
	var number any
	if lineNumber != 0 {
		number = lineNumber
	}
	msg := event.EventName
	if event.Details != "" {
		msg = event.Details
	}
	return FixtureEvent{
		ObjID:          event.ObjectID,
		DeviceID:       event.DeviceID,
		PPKNum:         event.DeviceNumber,
		Time:           event.Time,
		Code:           event.EventCode,
		Type:           "ppk_event",
		TypeEvent:      event.EventType,
		AdditionalType: event.AdditionalType,
		Msg:            msg,
		Line:           lineNumber,
		LineNumber:     lineNumber,
		Number:         number,
		HozUserID:      nil,
		ContactID:      nil,
	}
}

func firstUnifiedString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func fixtureDisconnectedFromUnified(disconnected []UnifiedDisconnectedDevice) []FixtureDisconnectedDevice {
	result := make([]FixtureDisconnectedDevice, 0, len(disconnected))
	for _, item := range disconnected {
		result = append(result, FixtureDisconnectedDevice{
			ObjID:        item.ObjectID,
			DeviceID:     item.DeviceID,
			Number:       item.Number,
			Offline:      item.Offline,
			Disconnected: item.Disconnected,
		})
	}
	return result
}

func fixtureGroupStatisticsFromUnified(devices []UnifiedDevice) map[string]map[string]int {
	result := make(map[string]map[string]int, len(devices))
	for _, device := range devices {
		groups := map[string]int{}
		for _, line := range device.Lines {
			if line.LineType == "FIRE" {
				groups[fmt.Sprintf("%d", line.GroupNumber)] = 1
				continue
			}
			if _, ok := groups[fmt.Sprintf("%d", line.GroupNumber)]; !ok {
				groups[fmt.Sprintf("%d", line.GroupNumber)] = 0
			}
		}
		result[fmt.Sprintf("%d", device.ObjectID)] = groups
	}
	return result
}

func fixtureAlarmEventsFromUnified(events []UnifiedEventType) []map[string]any {
	result := make([]map[string]any, 0)
	seen := map[string]struct{}{}
	for _, event := range events {
		if event.IsAlarm == 0 {
			continue
		}
		if _, ok := seen[event.EventByUser]; ok {
			continue
		}
		seen[event.EventByUser] = struct{}{}
		result = append(result, map[string]any{
			"code":              event.EventByUser,
			"name":              event.EventByUser,
			"is_alarm":          event.IsAlarm,
			"is_alarm_in_start": event.IsAlarm,
		})
	}
	return result
}

func findUnifiedObject(objects []UnifiedObject, objID int) (UnifiedObject, bool) {
	for _, object := range objects {
		if object.ID == objID {
			return object, true
		}
	}
	return UnifiedObject{}, false
}
