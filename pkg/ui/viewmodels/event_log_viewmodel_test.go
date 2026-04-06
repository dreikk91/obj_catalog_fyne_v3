package viewmodels

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type eventLogUseCaseStub struct {
	events []models.Event
}

func (s *eventLogUseCaseStub) FetchEvents() []models.Event {
	return append([]models.Event(nil), s.events...)
}

func TestEventLogViewModel_LoadEvents(t *testing.T) {
	vm := NewEventLogViewModel()
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.Local)
	events := vm.LoadEvents(&eventLogUseCaseStub{
		events: []models.Event{
			{ID: 1, Time: now.Add(-2 * time.Minute)},
			{ID: 2, Time: now.Add(-1 * time.Minute)},
		},
	})

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != 2 || events[1].ID != 1 {
		t.Fatalf("expected sorted events, got %+v", events)
	}
}

func TestEventLogViewModel_ApplyFiltersByPeriod(t *testing.T) {
	vm := NewEventLogViewModel()
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.Local)
	out := vm.ApplyFilters(EventLogFilterInput{
		AllEvents: []models.Event{
			{ID: 1, Time: now.Add(-10 * time.Minute), Type: models.EventFire},
			{ID: 2, ObjectID: phoenixObjectIDNamespaceStart + 12, Time: now.Add(-45 * time.Minute), Type: models.EventArm},
			{ID: 3, Time: now.Add(-2 * time.Hour), Type: models.EventFault},
		},
		Period: "Остання година",
		Now:    now,
	})

	if out.Count != 2 {
		t.Fatalf("expected 2 events in last hour, got %d", out.Count)
	}
	if out.Filtered[0].ID != 1 || out.Filtered[1].ID != 2 {
		t.Fatalf("unexpected filtered order: %+v", out.Filtered)
	}
	if out.CountAll != 2 || out.CountBridge != 1 || out.CountPhoenix != 1 || out.CountCASL != 0 {
		t.Fatalf("unexpected source counters: all=%d bridge=%d phoenix=%d casl=%d", out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}

func TestEventLogViewModel_ApplyFiltersImportantCurrentAndLimit(t *testing.T) {
	vm := NewEventLogViewModel()
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.Local)
	out := vm.ApplyFilters(EventLogFilterInput{
		AllEvents: []models.Event{
			{ID: 1, ObjectID: 10, Time: now.Add(-5 * time.Minute), Type: models.EventFire},
			{ID: 2, ObjectID: phoenixObjectIDNamespaceStart + 20, Time: now.Add(-6 * time.Minute), Type: models.EventFault},
			{ID: 3, ObjectID: 10, Time: now.Add(-7 * time.Minute), Type: models.EventArm},
			{ID: 4, ObjectID: 20, Time: now.Add(-8 * time.Minute), Type: models.EventBatteryLow},
		},
		Period:             "Всі",
		ImportantOnly:      true,
		ShowForCurrentOnly: true,
		CurrentObjectID:    10,
		HasCurrentObject:   true,
		MaxEvents:          1,
		Now:                now,
	})

	if out.Count != 1 {
		t.Fatalf("expected limit=1 to cut result, got %d", out.Count)
	}
	if out.Filtered[0].ID != 1 {
		t.Fatalf("expected first matching critical event, got %+v", out.Filtered[0])
	}
	if out.CountAll != 1 || out.CountBridge != 1 || out.CountPhoenix != 0 || out.CountCASL != 0 {
		t.Fatalf("unexpected source counters: all=%d bridge=%d phoenix=%d casl=%d", out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}

func TestEventLogViewModel_ApplyFiltersBySource(t *testing.T) {
	vm := NewEventLogViewModel()
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.Local)
	caslID := caslObjectIDNamespaceStart + 24
	phoenixID := phoenixObjectIDNamespaceStart + 5

	out := vm.ApplyFilters(EventLogFilterInput{
		AllEvents: []models.Event{
			{ID: 1, ObjectID: 11, Time: now.Add(-2 * time.Minute), Type: models.EventFire},
			{ID: 2, ObjectID: phoenixID, Time: now.Add(-3 * time.Minute), Type: models.EventArm},
			{ID: 3, ObjectID: caslID, Time: now.Add(-4 * time.Minute), Type: models.EventFire},
			{ID: 4, ObjectID: caslID, Time: now.Add(-5 * time.Minute), Type: models.EventFault},
		},
		Period:         "Всі",
		SelectedSource: ObjectSourceCASL,
		Now:            now,
	})

	if out.Count != 2 {
		t.Fatalf("expected 2 CASL events, got %d", out.Count)
	}
	if out.CountAll != 4 || out.CountBridge != 1 || out.CountPhoenix != 1 || out.CountCASL != 2 {
		t.Fatalf("unexpected source counters: all=%d bridge=%d phoenix=%d casl=%d", out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}

func TestEventLogViewModel_ApplyFiltersSortsNewestFirst(t *testing.T) {
	vm := NewEventLogViewModel()
	now := time.Date(2026, 3, 29, 12, 0, 0, 0, time.Local)

	out := vm.ApplyFilters(EventLogFilterInput{
		AllEvents: []models.Event{
			{ID: 1, ObjectID: 11, Time: now.Add(-20 * time.Minute), Type: models.EventFire},
			{ID: 2, ObjectID: 11, Time: now.Add(-5 * time.Minute), Type: models.EventArm},
			{ID: 3, ObjectID: 11, Time: now.Add(-10 * time.Minute), Type: models.EventFault},
		},
		Period: "Всі",
		Now:    now,
	})

	if len(out.Filtered) != 3 {
		t.Fatalf("expected 3 events, got %d", len(out.Filtered))
	}
	if out.Filtered[0].ID != 2 || out.Filtered[1].ID != 3 || out.Filtered[2].ID != 1 {
		t.Fatalf("unexpected sorted order: %+v", out.Filtered)
	}
}
