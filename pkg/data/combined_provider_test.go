package data

import (
	"strconv"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type combinedStubProvider struct {
	objects      []models.Object
	zones        map[string][]models.Zone
	employees    map[string][]models.Contact
	events       []models.Event
	objectEvents map[string][]models.Event
	alarms       []models.Alarm
	testMessages map[string][]models.TestMessage
	latestID     int64
	latestErr    error
}

func (s *combinedStubProvider) GetObjects() []models.Object {
	return append([]models.Object(nil), s.objects...)
}

func (s *combinedStubProvider) GetObjectByID(id string) *models.Object {
	for i := range s.objects {
		if strconv.Itoa(s.objects[i].ID) == id {
			obj := s.objects[i]
			return &obj
		}
	}
	return nil
}

func (s *combinedStubProvider) GetZones(objectID string) []models.Zone {
	return append([]models.Zone(nil), s.zones[objectID]...)
}

func (s *combinedStubProvider) GetEmployees(objectID string) []models.Contact {
	return append([]models.Contact(nil), s.employees[objectID]...)
}

func (s *combinedStubProvider) GetEvents() []models.Event {
	return append([]models.Event(nil), s.events...)
}

func (s *combinedStubProvider) GetObjectEvents(objectID string) []models.Event {
	if src, ok := s.objectEvents[objectID]; ok {
		return append([]models.Event(nil), src...)
	}
	return nil
}

func (s *combinedStubProvider) GetAlarms() []models.Alarm {
	return append([]models.Alarm(nil), s.alarms...)
}

func (s *combinedStubProvider) ProcessAlarm(id string, user string, note string) {}

func (s *combinedStubProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	return "", "", time.Time{}, time.Time{}
}

func (s *combinedStubProvider) GetTestMessages(objectID string) []models.TestMessage {
	return append([]models.TestMessage(nil), s.testMessages[objectID]...)
}

func (s *combinedStubProvider) GetLatestEventID() (int64, error) {
	return s.latestID, s.latestErr
}

func TestCombinedDataProvider_MergesObjectsAndAlarms(t *testing.T) {
	t.Parallel()

	now := time.Now()
	secondaryObjID := caslObjectIDNamespaceStart + 1

	primary := &combinedStubProvider{
		objects: []models.Object{
			{ID: 10, Name: "DB object"},
		},
		alarms: []models.Alarm{
			{ID: 10, ObjectID: 10, Time: now.Add(-2 * time.Minute)},
		},
	}
	secondary := &combinedStubProvider{
		objects: []models.Object{
			{ID: secondaryObjID, Name: "CASL object"},
		},
		alarms: []models.Alarm{
			{ID: secondaryObjID, ObjectID: secondaryObjID, Time: now.Add(-1 * time.Minute)},
		},
	}

	provider := NewCombinedDataProvider(primary, secondary)

	objects := provider.GetObjects()
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
	if objects[0].ID != 10 || objects[1].ID != secondaryObjID {
		t.Fatalf("unexpected merged objects order/ids: %+v", objects)
	}

	alarms := provider.GetAlarms()
	if len(alarms) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(alarms))
	}
	if alarms[0].ObjectID != secondaryObjID {
		t.Fatalf("latest alarm should be CASL alarm")
	}
}

func TestCombinedDataProvider_RoutesByObjectIDNamespace(t *testing.T) {
	t.Parallel()

	secondaryObjID := caslObjectIDNamespaceStart + 2
	secondaryObjIDStr := strconv.Itoa(secondaryObjID)

	primary := &combinedStubProvider{
		zones: map[string][]models.Zone{
			"42": {{Number: 42, Name: "DB zone"}},
		},
	}
	secondary := &combinedStubProvider{
		zones: map[string][]models.Zone{
			secondaryObjIDStr: {{Number: 2, Name: "CASL zone"}},
		},
	}

	provider := NewCombinedDataProvider(primary, secondary)

	dbZones := provider.GetZones("42")
	if len(dbZones) != 1 || dbZones[0].Name != "DB zone" {
		t.Fatalf("unexpected DB zones: %+v", dbZones)
	}

	caslZones := provider.GetZones(secondaryObjIDStr)
	if len(caslZones) != 1 || caslZones[0].Name != "CASL zone" {
		t.Fatalf("unexpected CASL zones: %+v", caslZones)
	}
}

func TestCombinedDataProvider_GetLatestEventID_ChangesWhenAnySourceChanges(t *testing.T) {
	t.Parallel()

	primary := &combinedStubProvider{latestID: 10}
	secondary := &combinedStubProvider{latestID: 20}
	provider := NewCombinedDataProvider(primary, secondary)

	first, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	second, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first != second {
		t.Fatalf("cursor must be stable when sources unchanged: %d != %d", first, second)
	}

	secondary.latestID = 21
	third, err := provider.GetLatestEventID()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if third == second {
		t.Fatalf("cursor must change when secondary source changes: %d == %d", third, second)
	}
}

func TestCombinedDataProvider_MergesBridgePhoenixAndCASLAlarms(t *testing.T) {
	t.Parallel()

	now := time.Now()
	phoenixObjID := phoenixObjectIDNamespaceStart + 10
	caslObjID := caslObjectIDNamespaceStart + 20

	provider := NewMultiSourceDataProvider(
		ProviderSource{
			Name: "bridge",
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 101, ObjectID: 101, ObjectNumber: "101", ObjectName: "Bridge object", Time: now.Add(-3 * time.Minute)},
				},
			},
		},
		ProviderSource{
			Name:         "phoenix",
			OwnsObjectID: IsPhoenixObjectID,
			OwnsAlarmID:  IsPhoenixObjectID,
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 201, ObjectID: phoenixObjID, ObjectNumber: "L00028", ObjectName: "Phoenix object", Time: now.Add(-2 * time.Minute)},
				},
			},
		},
		ProviderSource{
			Name:         "casl",
			OwnsObjectID: IsCASLObjectID,
			OwnsAlarmID:  IsCASLObjectID,
			Provider: &combinedStubProvider{
				alarms: []models.Alarm{
					{ID: 301, ObjectID: caslObjID, ObjectNumber: "1004", ObjectName: "CASL object", Time: now.Add(-1 * time.Minute)},
				},
			},
		},
	)

	alarms := provider.GetAlarms()
	if len(alarms) != 3 {
		t.Fatalf("expected 3 merged alarms, got %d", len(alarms))
	}
	if alarms[0].ObjectID != caslObjID {
		t.Fatalf("latest alarm must be CASL, got objectID=%d", alarms[0].ObjectID)
	}
	if alarms[1].ObjectID != phoenixObjID {
		t.Fatalf("second alarm must be Phoenix, got objectID=%d", alarms[1].ObjectID)
	}
	if alarms[2].ObjectID != 101 {
		t.Fatalf("third alarm must be Bridge, got objectID=%d", alarms[2].ObjectID)
	}
}
