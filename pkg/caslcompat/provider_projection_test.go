package caslcompat

import (
	"errors"
	"obj_catalog_fyne_v3/pkg/models"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBuildUnifiedFixtureFromDataProviderMapsObjectsRoomsDevicesAndAlarms(t *testing.T) {
	alarmTime := time.Date(2026, 4, 25, 12, 30, 0, 0, time.UTC)
	sourceMsgTime := alarmTime.Add(-time.Minute)
	provider := stubDataProvider{
		objects: []models.Object{
			{
				ID:            101,
				DisplayNumber: "50101",
				Name:          "Магазин",
				Address:       "Київ",
				ContractNum:   "P-101",
				DeviceType:    "Phoenix panel",
				PanelMark:     "Lun",
				TechnicianID:  "engineer-1",
				GSMLevel:      77,
				SIM1:          "0501112233",
				SIM2:          "0671112233",
				TestTime:      30,
				IsConnOK:      true,
				Status:        models.StatusNormal,
				Groups: []models.ObjectGroup{
					{Number: 1, Name: "Торговий зал"},
					{Number: 2, Name: "Склад"},
				},
			},
		},
		zones: map[string][]models.Zone{
			"101": {
				{Number: 1, Name: "Кнопка", SensorType: "Тривожна кнопка", Status: models.ZoneNormal, GroupNumber: 1},
				{Number: 2, Name: "Звичайний", SensorType: "Нормальний", Status: models.ZoneNormal, GroupNumber: 2},
			},
		},
		contacts: map[string][]models.Contact{
			"101": {
				{Name: "Петренко Іван", Phone: "+380501112233", Priority: 1},
			},
		},
		alarms: []models.Alarm{
			{
				ID:           77,
				ObjectID:     101,
				ObjectNumber: "50101",
				ObjectName:   "Магазин",
				Time:         alarmTime,
				Type:         models.AlarmFire,
				Details:      "Пожежна тривога",
				ZoneNumber:   2,
				SourceMsgs: []models.AlarmMsg{
					{Time: sourceMsgTime, Code: "E130", Number: 2, Details: "Пожежа склад", IsAlarm: true},
				},
			},
		},
		events: []models.Event{
			{ID: 88, ObjectID: 101, ObjectNumber: "50101", ObjectName: "Магазин", Time: alarmTime.Add(-2 * time.Minute), Type: models.EventArm, Details: "Постановка"},
		},
	}

	unified := BuildUnifiedFixtureFromDataProvider(provider, ProviderFixtureOptions{
		SourceName: "phoenixdb",
		DeviceType: UnifiedDeviceType{
			Type:      "PHOENIXDB_GENERIC",
			NameUK:    "PhoenixDB",
			NameRU:    "PhoenixDB",
			NameEN:    "PhoenixDB",
			MaxLines:  999,
			MaxGroups: 999,
		},
	})

	if len(unified.Objects) != 1 || unified.Objects[0].ID != 50101 {
		t.Fatalf("objects = %#v", unified.Objects)
	}
	if unified.Objects[0].DisplayNumber != "50101" || unified.Objects[0].Name != "Магазин" {
		t.Fatalf("object display fields = %#v", unified.Objects[0])
	}
	if unified.Objects[0].ResponsibleIDs[0] != "phoenixdb-380501112233" {
		t.Fatalf("responsible ids = %#v", unified.Objects[0].ResponsibleIDs)
	}
	if len(unified.Responders) != 1 || unified.Responders[0].PhoneNumber != "+380501112233" {
		t.Fatalf("responders = %#v", unified.Responders)
	}
	if len(unified.Devices) != 1 || unified.Devices[0].Number != 50101 {
		t.Fatalf("devices = %#v", unified.Devices)
	}
	if unified.Devices[0].ObjectID != 50101 || unified.Devices[0].Type != "Phoenix panel" {
		t.Fatalf("device identity = %#v", unified.Devices[0])
	}
	if unified.Devices[0].SIM1 != "0501112233" || unified.Devices[0].SIM2 != "0671112233" {
		t.Fatalf("device sims = %#v", unified.Devices[0])
	}
	if unified.Devices[0].Timeout != 1800 {
		t.Fatalf("device timeout = %#v", unified.Devices[0].Timeout)
	}
	if unified.Devices[0].TechnicianID != "engineer-1" {
		t.Fatalf("device technician = %#v", unified.Devices[0].TechnicianID)
	}
	if unified.Devices[0].SignalLevel != 77 {
		t.Fatalf("device signal level = %#v", unified.Devices[0].SignalLevel)
	}
	if got := unified.Devices[0].Lines[0]; got.LineType != "ALM_BTN" || got.RoomID != "5010101" {
		t.Fatalf("line 1 = %#v", got)
	}
	if got := unified.Devices[0].Lines[1]; got.LineType != "NORMAL" || got.RoomID != "5010102" {
		t.Fatalf("line 2 = %#v", got)
	}
	if len(unified.ActiveAlarms) != 1 || unified.ActiveAlarms[0].EventName != "UNIFIED_FIRE_ALARM" {
		t.Fatalf("active alarms = %#v", unified.ActiveAlarms)
	}
	if unified.ActiveAlarms[0].Details != "Пожежна тривога" {
		t.Fatalf("active alarm details = %#v", unified.ActiveAlarms[0])
	}
	if len(unified.ActiveAlarms[0].SourceEvents) != 1 || unified.ActiveAlarms[0].SourceEvents[0].Details != "Пожежа склад" {
		t.Fatalf("alarm source events = %#v", unified.ActiveAlarms[0].SourceEvents)
	}
	if unified.ActiveAlarms[0].SourceEvents[0].EventCode != 110 {
		t.Fatalf("source event code should stay mapped to known CASL code: %#v", unified.ActiveAlarms[0].SourceEvents[0])
	}
	if len(unified.JournalEvents) != 1 || unified.JournalEvents[0].EventName != "UNIFIED_ARM" {
		t.Fatalf("journal events = %#v", unified.JournalEvents)
	}

	fixture := BuildFixtureFromDataProvider(provider, ProviderFixtureOptions{SourceName: "phoenixdb"})
	if len(fixture.Rooms) != 2 || len(fixture.Rooms[0].Users) != 1 {
		t.Fatalf("fixture rooms = %#v", fixture.Rooms)
	}
	if len(fixture.Rooms[0].Lines) != 1 || len(fixture.Rooms[1].Lines) != 1 {
		t.Fatalf("fixture room lines = %#v", fixture.Rooms)
	}
	if len(fixture.GeneralTape) != 1 || fixture.GeneralTape[0].ObjID != 50101 {
		t.Fatalf("general tape = %#v", fixture.GeneralTape)
	}
	if fixture.GeneralTape[0].ReasonAlarm == "" || strings.Contains(fixture.GeneralTape[0].ReasonAlarm, "UNIFIED_") {
		t.Fatalf("general tape reason should use human text: %#v", fixture.GeneralTape[0].ReasonAlarm)
	}
	items := fixture.GeneralTapeItems["50101"]
	if len(items) < 3 {
		t.Fatalf("general tape items = %#v", items)
	}
	if items[0].Msg != "Постановка" {
		t.Fatalf("journal item = %#v", items[0])
	}
	if items[1].Msg != "Пожежа склад" {
		t.Fatalf("alarm source item = %#v", items[1])
	}
}

func TestBuildUnifiedFixtureFromDataProviderUsesPublicObjectID(t *testing.T) {
	provider := stubDataProvider{
		objects: []models.Object{
			{
				ID:            1006827206,
				DisplayNumber: "L00005",
				Name:          "Phoenix object",
				IsConnOK:      true,
			},
		},
	}

	unified := BuildUnifiedFixtureFromDataProvider(provider, ProviderFixtureOptions{SourceName: "phoenixdb"})
	if len(unified.Objects) != 1 {
		t.Fatalf("objects = %#v", unified.Objects)
	}
	if unified.Objects[0].ID != 100005 {
		t.Fatalf("public obj_id = %d, want 100005; object = %#v", unified.Objects[0].ID, unified.Objects[0])
	}
	if len(unified.Devices) != 1 || unified.Devices[0].ObjectID != 100005 {
		t.Fatalf("devices = %#v", unified.Devices)
	}
	if unified.Devices[0].Number != 100005 {
		t.Fatalf("device number = %#v", unified.Devices[0])
	}
}

func TestBuildUnifiedFixtureFromDataProviderUsesSixDigitPublicObjectIDForPhoenixMarks(t *testing.T) {
	provider := stubDataProvider{
		objects: []models.Object{
			{ID: 1006827206, DisplayNumber: "L00027", Name: "Phoenix object", IsConnOK: true},
		},
	}

	unified := BuildUnifiedFixtureFromDataProvider(provider, ProviderFixtureOptions{SourceName: "phoenixdb"})
	if len(unified.Objects) != 1 {
		t.Fatalf("objects = %#v", unified.Objects)
	}
	if unified.Objects[0].ID != 100027 {
		t.Fatalf("public obj_id = %d, want 100027; object = %#v", unified.Objects[0].ID, unified.Objects[0])
	}
	if len(unified.Devices) != 1 || unified.Devices[0].Number != 100027 {
		t.Fatalf("devices = %#v", unified.Devices)
	}
}

func TestBuildUnifiedFixtureFromDataProviderPrefersRealDeviceNameOverPanelMark(t *testing.T) {
	provider := stubDataProvider{
		objects: []models.Object{
			{
				ID:            1006827206,
				DisplayNumber: "L00027",
				Name:          "Phoenix object",
				DeviceType:    "Лунь-11",
				PanelMark:     "L00027",
				IsConnOK:      true,
			},
		},
	}

	unified := BuildUnifiedFixtureFromDataProvider(provider, ProviderFixtureOptions{SourceName: "phoenixdb"})
	if len(unified.Devices) != 1 {
		t.Fatalf("devices = %#v", unified.Devices)
	}
	if unified.Devices[0].Name != "Лунь-11" {
		t.Fatalf("device name = %q, want %q", unified.Devices[0].Name, "Лунь-11")
	}
	if unified.Devices[0].Type != "Лунь-11" {
		t.Fatalf("device type = %q, want %q", unified.Devices[0].Type, "Лунь-11")
	}
}

func TestBuildUnifiedFixtureFromDataProviderMapsFireObjectLinesAsFire(t *testing.T) {
	provider := stubDataProvider{
		objects: []models.Object{
			{
				ID:            202,
				DisplayNumber: "F202",
				Name:          "Пожежний об'єкт",
				DeviceType:    "Пожежна сигналізація",
				IsConnOK:      true,
				Groups:        []models.ObjectGroup{{Number: 1, Name: "Будівля"}},
			},
		},
		zones: map[string][]models.Zone{
			"202": {
				{Number: 1, Name: "Зона 1", SensorType: "Нормальний", GroupNumber: 1},
				{Number: 2, Name: "Кнопка", SensorType: "Тривожна кнопка", GroupNumber: 1},
			},
		},
	}

	unified := BuildUnifiedFixtureFromDataProvider(provider, ProviderFixtureOptions{SourceName: "phoenixdb"})
	if len(unified.Devices) != 1 {
		t.Fatalf("devices = %#v", unified.Devices)
	}
	for _, line := range unified.Devices[0].Lines {
		if line.LineType != "FIRE" {
			t.Fatalf("fire object line = %#v", line)
		}
	}
}

type stubDataProvider struct {
	objects  []models.Object
	zones    map[string][]models.Zone
	contacts map[string][]models.Contact
	events   []models.Event
	alarms   []models.Alarm
}

func (p stubDataProvider) GetObjects() []models.Object {
	return append([]models.Object(nil), p.objects...)
}

func (p stubDataProvider) GetObjectByID(id string) *models.Object {
	for _, object := range p.objects {
		if id == "" || id == "0" || id == object.DisplayNumber || id == strconv.Itoa(object.ID) {
			obj := object
			return &obj
		}
	}
	return nil
}

func (p stubDataProvider) GetZones(objectID string) []models.Zone {
	return append([]models.Zone(nil), p.zones[objectID]...)
}

func (p stubDataProvider) GetEmployees(objectID string) []models.Contact {
	return append([]models.Contact(nil), p.contacts[objectID]...)
}

func (p stubDataProvider) GetTestMessages(string) []models.TestMessage {
	return nil
}

func (p stubDataProvider) GetExternalData(string) (string, string, time.Time, time.Time) {
	return "", "", time.Time{}, time.Time{}
}

func (p stubDataProvider) GetEvents() []models.Event {
	return append([]models.Event(nil), p.events...)
}

func (p stubDataProvider) GetObjectEvents(string) []models.Event {
	return append([]models.Event(nil), p.events...)
}

func (p stubDataProvider) GetAlarms() []models.Alarm {
	return append([]models.Alarm(nil), p.alarms...)
}

func (p stubDataProvider) ProcessAlarm(string, string, string) error {
	return errors.New("not implemented")
}
