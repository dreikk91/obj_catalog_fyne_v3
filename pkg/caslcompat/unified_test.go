package caslcompat

import "testing"

func TestBuildFixtureFromUnifiedProjectsCASLContracts(t *testing.T) {
	source := UnifiedFixture{
		Admin: UnifiedUser{ID: "1", Email: "admin@example.com", Role: "ADMIN", FirstName: "Admin", LastName: "Fixture", PultID: 1, BasketID: 1},
		Managers: []UnifiedManager{
			{ID: 7, Name: "МГР 7", Number: 7, UserIDs: []string{"1"}},
		},
		Pults: []UnifiedPult{
			{ID: 1, Name: "Pult", Number: 1, UserIDs: []string{"1"}},
		},
		Objects: []UnifiedObject{
			{ID: 42, DisplayNumber: "0042", Name: "Object 42", Address: "Address", Description: "Description", Contract: "C-42", ReactingPultID: 1, ResponsibleIDs: []string{"2"}, Room: UnifiedRoom{ID: "4201", Name: "Room", Description: "Room"}},
		},
		Devices: []UnifiedDevice{
			{ID: 43, ObjectID: 42, Number: 420042, Name: "Device", Type: "TEST_DEVICE", Timeout: 60, Enabled: -1, Lines: []UnifiedLine{
				{ID: 1, Number: 1, AdapterType: "SYS", LineType: "NORMAL", Description: "Zone", GroupNumber: 1, RoomID: "4201"},
			}},
		},
		DeviceTypes: []UnifiedDeviceType{
			{Type: "TEST_DEVICE", NameUK: "Тест", NameRU: "Тест", NameEN: "Test", MaxLines: 16, MaxGroups: 4},
		},
		EventTypes: []UnifiedEventType{
			{DeviceType: "TEST_DEVICE", Code: 101, TypeEvent: "E", AdditionalType: 1, EventByUser: "TEST_ALARM", IsAlarm: 1, LangUK: "Тестова тривога", LangRU: "Тестовая тревога", LangEN: "Test alarm"},
		},
		ActiveAlarms: []UnifiedAlarm{
			{ObjectID: 42, DeviceID: 43, DeviceNumber: 420042, Time: 1777058079975, UserID: "0", AlarmType: "ALARM_TYPE_DEVICE", EventCode: 101, EventType: "E", AdditionalType: 1, EventName: "TEST_ALARM", LineNumber: 1, LastAction: "GRD_OBJ_NOTIF"},
		},
		CountOfRooms:  1,
		TotalObjects:  1,
		ActiveAlarmsN: 1,
	}

	fixture := buildFixtureFromUnified(source)

	if fixture.User.Role != "ADMIN" {
		t.Fatalf("admin role = %q", fixture.User.Role)
	}
	if len(fixture.Objects) != 1 || fixture.Objects[0].ObjID != 42 {
		t.Fatalf("objects = %#v", fixture.Objects)
	}
	if fixture.Objects[0].DisplayNumber != "0042" {
		t.Fatalf("object display number = %#v", fixture.Objects[0].DisplayNumber)
	}
	if len(fixture.Devices) != 1 || fixture.Devices[0].Lines["1"].AdapterType != "SYS" {
		t.Fatalf("devices = %#v", fixture.Devices)
	}
	if len(fixture.Connections) != 1 || fixture.Connections[0].GuardedObject.ObjID != 42 {
		t.Fatalf("connections = %#v", fixture.Connections)
	}
	if len(fixture.Rooms) != 1 || fixture.Rooms[0].RoomID != "4201" {
		t.Fatalf("rooms = %#v", fixture.Rooms)
	}
	if got := fixture.Rooms[0].Users; len(got) != 1 || got[0].UserID != "2" {
		t.Fatalf("room users = %#v", got)
	}
	if got := fixture.Rooms[0].Lines["1"]; got.DeviceID != 43 || got.RoomID != "4201" {
		t.Fatalf("room line = %#v", got)
	}
	if len(fixture.GeneralTape) != 1 || fixture.GeneralTape[0].LastAct != "GRD_OBJ_NOTIF" {
		t.Fatalf("general tape = %#v", fixture.GeneralTape)
	}
	if rows := fixture.GeneralTapeItems["42"]; len(rows) != 2 || rows[0].Msg != "TEST_ALARM" {
		t.Fatalf("general tape items = %#v", rows)
	}
	dictionary := fixture.Dictionary
	devices := dictionary["devices"].([]map[string]any)
	if devices[0]["type"] != "TEST_DEVICE" || devices[0]["max_lines"] != 16 {
		t.Fatalf("dictionary devices = %#v", devices)
	}
	translators := fixture.MessageTranslators["TEST_DEVICE"]
	if len(translators) != 1 || translators[0].EventByUser != "TEST_ALARM" {
		t.Fatalf("translators = %#v", translators)
	}
	if len(fixture.AlarmEvents) != 1 || fixture.AlarmEvents[0]["code"] != "TEST_ALARM" {
		t.Fatalf("alarm events = %#v", fixture.AlarmEvents)
	}
}
