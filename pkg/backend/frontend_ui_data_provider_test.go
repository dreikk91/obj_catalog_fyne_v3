package backend

import (
	"context"
	"errors"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type frontendUIBackendStub struct {
	objectSummaries []contracts.FrontendObjectSummary
	objectDetails   contracts.FrontendObjectDetails
	events          []contracts.FrontendEventItem
	alarms          []contracts.FrontendAlarmItem
	objectDetailErr error
}

func (s *frontendUIBackendStub) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	return contracts.FrontendCapabilities{}, nil
}

func (s *frontendUIBackendStub) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	return s.objectSummaries, nil
}

func (s *frontendUIBackendStub) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	return s.alarms, nil
}

func (s *frontendUIBackendStub) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	return s.events, nil
}

func (s *frontendUIBackendStub) GetObjectDetails(context.Context, int) (contracts.FrontendObjectDetails, error) {
	if s.objectDetailErr != nil {
		return contracts.FrontendObjectDetails{}, s.objectDetailErr
	}
	return s.objectDetails, nil
}

func (s *frontendUIBackendStub) CreateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

func (s *frontendUIBackendStub) UpdateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

type frontendUIFallbackStub struct {
	objects      []models.Object
	objectByID   map[string]models.Object
	zones        []models.Zone
	contacts     []models.Contact
	events       []models.Event
	objectEvents []models.Event
	alarms       []models.Alarm

	processAlarmCalls int
	testMessages      []models.TestMessage
}

func (s *frontendUIFallbackStub) GetObjects() []models.Object {
	return s.objects
}

func (s *frontendUIFallbackStub) GetObjectByID(id string) *models.Object {
	object, ok := s.objectByID[id]
	if !ok {
		return nil
	}
	copy := object
	return &copy
}

func (s *frontendUIFallbackStub) GetZones(string) []models.Zone {
	return s.zones
}

func (s *frontendUIFallbackStub) GetEmployees(string) []models.Contact {
	return s.contacts
}

func (s *frontendUIFallbackStub) GetExternalData(string) (string, string, time.Time, time.Time) {
	return "fallback-signal", "fallback-test", time.Time{}, time.Time{}
}

func (s *frontendUIFallbackStub) GetEvents() []models.Event {
	return s.events
}

func (s *frontendUIFallbackStub) GetObjectEvents(string) []models.Event {
	return s.objectEvents
}

func (s *frontendUIFallbackStub) GetAlarms() []models.Alarm {
	return s.alarms
}

func (s *frontendUIFallbackStub) ProcessAlarm(string, string, string) error {
	s.processAlarmCalls++
	return nil
}

func (s *frontendUIFallbackStub) GetTestMessages(string) []models.TestMessage {
	return s.testMessages
}

func TestFrontendUIDataProviderGetObjectsUsesFrontendAndKeepsLegacyFlags(t *testing.T) {
	frontend := &frontendUIBackendStub{
		objectSummaries: []contracts.FrontendObjectSummary{
			{
				ID:               101,
				DisplayNumber:    "101",
				Name:             "Нова назва",
				Address:          "Нова адреса",
				StatusCode:       "fault",
				StatusText:       "НЕСПРАВНІСТЬ",
				ContractNumber:   "CNT-1",
				GuardStatus:      contracts.FrontendGuardStatusDisarmed,
				ConnectionStatus: contracts.FrontendConnectionStatusOnline,
				MonitoringStatus: contracts.FrontendMonitoringStatusDebug,
				HasAssignment:    true,
			},
		},
	}
	fallback := &frontendUIFallbackStub{
		objects: []models.Object{
			{
				ID:   101,
				Name: "Стара назва",
			},
		},
	}

	provider := NewFrontendUIDataProvider(frontend, fallback)
	objects := provider.GetObjects()
	if len(objects) != 1 {
		t.Fatalf("len(GetObjects()) = %d, want 1", len(objects))
	}
	if objects[0].Name != "Нова назва" {
		t.Fatalf("object name = %q, want frontend name", objects[0].Name)
	}
	if objects[0].GuardState != 0 || objects[0].IsConnState != 1 || objects[0].BlockedArmedOnOff != 2 {
		t.Fatalf("normalized states were not derived: %+v", objects[0])
	}
	if objects[0].Status != models.StatusFault {
		t.Fatalf("status = %v, want %v", objects[0].Status, models.StatusFault)
	}
	if !objects[0].HasAssignment || objects[0].TechAlarmState != 1 {
		t.Fatalf("derived assignment/fault state mismatch: %+v", objects[0])
	}
}

func TestFrontendUIDataProviderGetObjectByIDUsesFrontendDetailsAndPreservesGroups(t *testing.T) {
	lastTest := time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC)
	lastMessage := time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC)

	frontend := &frontendUIBackendStub{
		objectDetails: contracts.FrontendObjectDetails{
			Summary: contracts.FrontendObjectSummary{
				ID:               202,
				DisplayNumber:    "202",
				Name:             "Школа",
				Address:          "Київ",
				StatusCode:       "normal",
				StatusText:       "НОРМА",
				Phone:            "380001",
				DeviceType:       "Tiras",
				PanelMark:        "Mark",
				SignalStrength:   "-61 dBm",
				LastTestTime:     lastTest,
				LastMessageTime:  lastMessage,
				GuardStatus:      contracts.FrontendGuardStatusGuarded,
				ConnectionStatus: contracts.FrontendConnectionStatusOnline,
				MonitoringStatus: contracts.FrontendMonitoringStatusActive,
				HasAssignment:    true,
			},
			GSMLevel:            88,
			PowerSource:         "battery",
			AutoTestHours:       24,
			ChannelCode:         5,
			AKBState:            1,
			PowerFault:          0,
			TestControl:         true,
			TestIntervalMin:     60,
			Phones:              "380001",
			Notes:               "Примітка",
			Location:            "Підвал",
			LaunchDate:          "01.04.2026",
			ExternalSignal:      "OK",
			ExternalTestMessage: "TEST",
			ExternalLastTest:    lastTest,
			ExternalLastMessage: lastMessage,
			Zones: []contracts.FrontendZone{
				{Number: 1, Name: "Зона 1", SensorType: "Дим", Status: "fire", GroupID: "g1", GroupNumber: 1, GroupName: "Група 1", GroupStateText: "Під охороною"},
			},
			Contacts: []contracts.FrontendContact{
				{Name: "Черговий", Phone: "380002", GroupID: "g1", GroupNumber: 1, GroupName: "Група 1", GroupStateText: "Під охороною"},
			},
			Events: []contracts.FrontendEventItem{
				{ID: 1, ObjectID: 202, ObjectNumber: "202", ObjectName: "Школа", Time: lastMessage, TypeCode: "fire", TypeText: "ПОЖЕЖА", ZoneNumber: 1, Details: "Спрацювання", VisualSeverity: contracts.FrontendVisualSeverityCritical},
			},
		},
	}
	fallback := &frontendUIFallbackStub{
		objectByID: map[string]models.Object{
			"202": {ID: 202},
		},
		zones: []models.Zone{{Number: 9, Name: "fallback"}},
		contacts: []models.Contact{
			{Name: "fallback"},
		},
	}

	provider := NewFrontendUIDataProvider(frontend, fallback)
	object := provider.GetObjectByID("202")
	if object == nil {
		t.Fatal("GetObjectByID() returned nil")
	}
	if object.Name != "Школа" || object.Location1 != "Підвал" || object.Notes1 != "Примітка" {
		t.Fatalf("object details were not mapped: %+v", *object)
	}
	if object.PowerSource != models.PowerBattery || object.TestControl != 1 || object.TestTime != 60 {
		t.Fatalf("device state was not mapped: %+v", *object)
	}
	if object.GuardState != 1 || object.IsConnState != 1 || object.BlockedArmedOnOff != 0 || !object.HasAssignment {
		t.Fatalf("normalized state mismatch: %+v", *object)
	}
	if len(object.Groups) != 1 || object.Groups[0].ID != "g1" {
		t.Fatalf("groups were not preserved: %+v", object.Groups)
	}

	zones := provider.GetZones("202")
	if len(zones) != 1 || zones[0].Status != models.ZoneFire {
		t.Fatalf("zones = %+v, want mapped frontend zones", zones)
	}

	contacts := provider.GetEmployees("202")
	if len(contacts) != 1 || contacts[0].Name != "Черговий" {
		t.Fatalf("contacts = %+v, want mapped frontend contacts", contacts)
	}

	events := provider.GetObjectEvents("202")
	if len(events) != 1 || events[0].SC1 != 1 || events[0].Type != models.EventFire {
		t.Fatalf("object events = %+v, want frontend event with fallback color", events)
	}

	signal, testMsg, gotLastTest, gotLastMessage := provider.GetExternalData("202")
	if signal != "OK" || testMsg != "TEST" || !gotLastTest.Equal(lastTest) || !gotLastMessage.Equal(lastMessage) {
		t.Fatalf("external data = (%q, %q, %v, %v)", signal, testMsg, gotLastTest, gotLastMessage)
	}
}

func TestFrontendUIDataProviderGetEventsAndAlarmsUseFrontendAndDelegateFallbackOperations(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	frontend := &frontendUIBackendStub{
		events: []contracts.FrontendEventItem{
			{ID: 10, ObjectID: 1, ObjectNumber: "1", ObjectName: "Obj", Time: now, TypeCode: "offline", TypeText: "НЕМАЄ ЗВ'ЯЗКУ", Details: "det", VisualSeverity: contracts.FrontendVisualSeverityWarning},
		},
		alarms: []contracts.FrontendAlarmItem{
			{ID: 20, ObjectID: 1, ObjectNumber: "1", ObjectName: "Obj", Address: "Addr", Time: now, TypeCode: "fire", TypeText: "ПОЖЕЖА", Details: "alarm", ZoneNumber: 2, VisualSeverity: contracts.FrontendVisualSeverityCritical},
		},
	}
	fallback := &frontendUIFallbackStub{
		alarms: []models.Alarm{
			{ID: 20, SC1: 6, SourceMsgs: []models.AlarmMsg{{Details: "src"}}},
		},
		testMessages: []models.TestMessage{{Info: "T1"}},
	}

	provider := NewFrontendUIDataProvider(frontend, fallback)

	events := provider.GetEvents()
	if len(events) != 1 || events[0].SC1 != 2 || events[0].Type != models.EventOffline {
		t.Fatalf("events = %+v, want frontend event merged with normalized SC1", events)
	}

	alarms := provider.GetAlarms()
	if len(alarms) != 1 || alarms[0].SC1 != 6 || alarms[0].Type != models.AlarmFire || len(alarms[0].SourceMsgs) != 1 {
		t.Fatalf("alarms = %+v, want frontend alarm merged with fallback fields", alarms)
	}

	if err := provider.ProcessAlarm("20", "user", "note"); err != nil {
		t.Fatalf("ProcessAlarm() error = %v", err)
	}
	if fallback.processAlarmCalls != 1 {
		t.Fatalf("ProcessAlarm calls = %d, want 1", fallback.processAlarmCalls)
	}

	messages := provider.GetTestMessages("1")
	if len(messages) != 1 || messages[0].Info != "T1" {
		t.Fatalf("test messages = %+v, want delegated fallback messages", messages)
	}
}

func TestFrontendUIDataProviderFallsBackWhenFrontendFails(t *testing.T) {
	frontend := &frontendUIBackendStub{objectDetailErr: errors.New("boom")}
	fallback := &frontendUIFallbackStub{
		objectByID: map[string]models.Object{
			"303": {ID: 303, Name: "fallback"},
		},
	}

	provider := NewFrontendUIDataProvider(frontend, fallback)
	object := provider.GetObjectByID("303")
	if object == nil || object.Name != "fallback" {
		t.Fatalf("fallback object = %+v, want fallback data", object)
	}
}
