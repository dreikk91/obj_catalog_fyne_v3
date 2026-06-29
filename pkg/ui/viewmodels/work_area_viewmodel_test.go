package viewmodels

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type workAreaDataProviderStub struct {
	lastRequestedID string
	objectRequests  int
	zoneRequests    int
	contactRequests int
	eventRequests   int

	fullObject *models.Object
	zones      []models.Zone
	contacts   []models.Contact
	events     []models.Event

	signal      string
	testMessage string
	lastTest    time.Time
	lastMessage time.Time
}

func (s *workAreaDataProviderStub) GetObjectByID(id string) *models.Object {
	s.lastRequestedID = id
	s.objectRequests++
	return s.fullObject
}

func (s *workAreaDataProviderStub) GetZones(objectID string) []models.Zone {
	s.lastRequestedID = objectID
	s.zoneRequests++
	return s.zones
}

func (s *workAreaDataProviderStub) GetEmployees(objectID string) []models.Contact {
	s.lastRequestedID = objectID
	s.contactRequests++
	return s.contacts
}

func (s *workAreaDataProviderStub) GetObjectEvents(objectID string) []models.Event {
	s.lastRequestedID = objectID
	s.eventRequests++
	return s.events
}

func (s *workAreaDataProviderStub) GetExternalData(objectID string) (string, string, time.Time, time.Time) {
	s.lastRequestedID = objectID
	return s.signal, s.testMessage, s.lastTest, s.lastMessage
}

type optimizedWorkAreaDataProviderStub struct {
	workAreaDataProviderStub
	baseRequests int
}

type rangedWorkAreaDataProviderStub struct {
	workAreaDataProviderStub
	rangeRequests int
	rangeFrom     time.Time
	rangeTo       time.Time
}

func (s *rangedWorkAreaDataProviderStub) GetObjectEventsRange(objectID string, from time.Time, to time.Time) []models.Event {
	s.lastRequestedID = objectID
	s.rangeRequests++
	s.rangeFrom = from
	s.rangeTo = to
	return s.events
}

func (s *optimizedWorkAreaDataProviderStub) GetObjectBaseDetails(objectID string) (*models.Object, []models.Zone, []models.Contact) {
	s.lastRequestedID = objectID
	s.baseRequests++
	return s.fullObject, s.zones, s.contacts
}

func TestWorkAreaViewModel_LoadObjectBaseDetails(t *testing.T) {
	vm := NewWorkAreaViewModel()
	stub := &workAreaDataProviderStub{
		fullObject: &models.Object{ID: 42, Name: "Obj 42"},
		zones: []models.Zone{
			{Number: 1, Name: "Zone 1"},
		},
		contacts: []models.Contact{
			{Name: "John"},
		},
		events: []models.Event{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
	}

	details := vm.LoadObjectBaseDetails(stub, 42)

	if stub.lastRequestedID != "42" {
		t.Fatalf("expected string object id 42, got %q", stub.lastRequestedID)
	}
	if details.FullObject == nil || details.FullObject.ID != 42 {
		t.Fatalf("unexpected full object: %#v", details.FullObject)
	}
	if len(details.Zones) != 1 {
		t.Fatalf("unexpected zones count: %d", len(details.Zones))
	}
	if len(details.Contacts) != 1 {
		t.Fatalf("unexpected contacts count: %d", len(details.Contacts))
	}
	if len(details.Events) != 0 {
		t.Fatalf("base details must not preload events, got %d", len(details.Events))
	}
	if stub.eventRequests != 0 {
		t.Fatalf("base details must not request events, got %d requests", stub.eventRequests)
	}
}

func TestWorkAreaViewModel_LoadObjectBaseDetailsUsesOptimizedProvider(t *testing.T) {
	vm := NewWorkAreaViewModel()
	stub := &optimizedWorkAreaDataProviderStub{
		workAreaDataProviderStub: workAreaDataProviderStub{
			fullObject: &models.Object{ID: 42, Name: "Obj 42"},
			zones:      []models.Zone{{Number: 1, Name: "Zone 1"}},
			contacts:   []models.Contact{{Name: "John"}},
		},
	}

	details := vm.LoadObjectBaseDetails(stub, 42)

	if stub.baseRequests != 1 {
		t.Fatalf("optimized base requests = %d, want 1", stub.baseRequests)
	}
	if stub.objectRequests != 0 || stub.zoneRequests != 0 || stub.contactRequests != 0 {
		t.Fatalf("fallback requests must not be used: object=%d zones=%d contacts=%d", stub.objectRequests, stub.zoneRequests, stub.contactRequests)
	}
	if details.FullObject == nil || details.FullObject.ID != 42 {
		t.Fatalf("unexpected full object: %#v", details.FullObject)
	}
	if len(details.Zones) != 1 || len(details.Contacts) != 1 {
		t.Fatalf("unexpected details: zones=%d contacts=%d", len(details.Zones), len(details.Contacts))
	}
}

func TestWorkAreaViewModel_LoadObjectEvents(t *testing.T) {
	vm := NewWorkAreaViewModel()
	stub := &workAreaDataProviderStub{
		events: []models.Event{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
	}

	events := vm.LoadObjectEvents(stub, 7, 2)
	if stub.lastRequestedID != "7" {
		t.Fatalf("expected string object id 7, got %q", stub.lastRequestedID)
	}
	if len(events) != 2 {
		t.Fatalf("expected event limit to apply, got %d", len(events))
	}

	events[0].ID = 999
	if stub.events[0].ID == 999 {
		t.Fatalf("events must not alias source events slice")
	}
}

func TestWorkAreaViewModel_LoadObjectEventsRange(t *testing.T) {
	vm := NewWorkAreaViewModel()
	from := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	to := from.Add(72 * time.Hour)
	stub := &rangedWorkAreaDataProviderStub{
		workAreaDataProviderStub: workAreaDataProviderStub{
			events: []models.Event{
				{ID: 1, Time: from.Add(-time.Minute)},
				{ID: 2, Time: from.Add(time.Hour)},
				{ID: 3, Time: to.Add(time.Minute)},
			},
		},
	}

	events := vm.LoadObjectEventsRange(stub, 7, 100, from, to)
	if stub.rangeRequests != 1 || stub.eventRequests != 0 {
		t.Fatalf("range requests = %d, fallback requests = %d", stub.rangeRequests, stub.eventRequests)
	}
	if !stub.rangeFrom.Equal(from) || !stub.rangeTo.Equal(to) {
		t.Fatalf("requested range = %v..%v", stub.rangeFrom, stub.rangeTo)
	}
	if len(events) != 1 || events[0].ID != 2 {
		t.Fatalf("filtered events = %+v", events)
	}
}

func TestWorkAreaViewModel_CanApplyDetails(t *testing.T) {
	vm := NewWorkAreaViewModel()
	current := &models.Object{ID: 15}

	if !vm.CanApplyDetails(current, 15) {
		t.Fatalf("expected apply for same object id")
	}
	if vm.CanApplyDetails(current, 16) {
		t.Fatalf("must not apply for different object id")
	}
	if vm.CanApplyDetails(nil, 15) {
		t.Fatalf("must not apply for nil current object")
	}
}

func TestWorkAreaViewModel_LoadExternalData(t *testing.T) {
	vm := NewWorkAreaViewModel()
	lastTest := time.Date(2026, 3, 28, 10, 0, 0, 0, time.Local)
	lastMessage := time.Date(2026, 3, 28, 10, 5, 0, 0, time.Local)
	stub := &workAreaDataProviderStub{
		signal:      "85%",
		testMessage: "OK",
		lastTest:    lastTest,
		lastMessage: lastMessage,
	}

	external := vm.LoadExternalData(stub, 99)
	if stub.lastRequestedID != "99" {
		t.Fatalf("expected string object id 99, got %q", stub.lastRequestedID)
	}
	if external.Signal != "85%" || external.TestMessage != "OK" {
		t.Fatalf("unexpected external data payload: %#v", external)
	}
	if !external.LastTest.Equal(lastTest) {
		t.Fatalf("unexpected last test time")
	}
	if !external.LastMessage.Equal(lastMessage) {
		t.Fatalf("unexpected last message time")
	}
}
