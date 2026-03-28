package usecases

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type eventLogRepoStub struct {
	events []models.Event
}

func (s *eventLogRepoStub) GetEvents() []models.Event {
	return append([]models.Event(nil), s.events...)
}

func TestEventLogUseCase_FetchEventsReturnsCopy(t *testing.T) {
	stub := &eventLogRepoStub{
		events: []models.Event{
			{ID: 1, Time: time.Now()},
			{ID: 2, Time: time.Now()},
		},
	}
	uc := NewEventLogUseCase(stub)

	got := uc.FetchEvents()
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	got[0].ID = 999
	if stub.events[0].ID != 1 {
		t.Fatalf("use case must return copy, repository data changed")
	}
}

func TestEventLogUseCase_FetchEventsNilRepository(t *testing.T) {
	uc := NewEventLogUseCase(nil)
	got := uc.FetchEvents()
	if len(got) != 0 {
		t.Fatalf("expected empty result for nil repository, got %d", len(got))
	}
}
