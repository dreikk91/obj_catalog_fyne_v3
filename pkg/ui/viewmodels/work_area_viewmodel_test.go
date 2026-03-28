package viewmodels

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type workAreaDataProviderStub struct {
	lastRequestedID string

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
	return s.fullObject
}

func (s *workAreaDataProviderStub) GetZones(objectID string) []models.Zone {
	s.lastRequestedID = objectID
	return s.zones
}

func (s *workAreaDataProviderStub) GetEmployees(objectID string) []models.Contact {
	s.lastRequestedID = objectID
	return s.contacts
}

func (s *workAreaDataProviderStub) GetObjectEvents(objectID string) []models.Event {
	s.lastRequestedID = objectID
	return s.events
}

func (s *workAreaDataProviderStub) GetExternalData(objectID string) (string, string, time.Time, time.Time) {
	s.lastRequestedID = objectID
	return s.signal, s.testMessage, s.lastTest, s.lastMessage
}

func TestWorkAreaViewModel_LoadObjectDetails(t *testing.T) {
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

	details := vm.LoadObjectDetails(stub, 42, 2)

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
	if len(details.Events) != 2 {
		t.Fatalf("expected event limit to apply, got %d", len(details.Events))
	}

	details.Events[0].ID = 999
	if stub.events[0].ID == 999 {
		t.Fatalf("details must not alias source events slice")
	}
}

func TestWorkAreaViewModel_LoadObjectDetails_WithoutLimit(t *testing.T) {
	vm := NewWorkAreaViewModel()
	stub := &workAreaDataProviderStub{
		events: []models.Event{
			{ID: 1},
			{ID: 2},
		},
	}

	details := vm.LoadObjectDetails(stub, 7, 0)
	if len(details.Events) != 2 {
		t.Fatalf("expected all events without limit, got %d", len(details.Events))
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
