package data

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestBuildAlarmSourceMessagesFromEvents_SelectsMatchingCase(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 10, 0, 0, 0, time.Local)
	events := []models.Event{
		{
			ID:         1,
			Time:       base,
			Type:       models.EventFire,
			ZoneNumber: 1,
			Details:    "Перша пожежа",
			SC1:        1,
		},
		{
			ID:         2,
			Time:       base.Add(1 * time.Minute),
			Type:       models.EventRestore,
			ZoneNumber: 1,
			Details:    "Відновлення 1",
			SC1:        5,
		},
		{
			ID:         3,
			Time:       base.Add(10 * time.Minute),
			Type:       models.EventFire,
			ZoneNumber: 2,
			Details:    "Друга пожежа",
			SC1:        1,
		},
		{
			ID:         4,
			Time:       base.Add(11 * time.Minute),
			Type:       models.EventFault,
			ZoneNumber: 2,
			Details:    "Несправність після пожежі",
			SC1:        2,
		},
	}

	alarm := models.Alarm{
		Time:       base.Add(10 * time.Minute),
		Type:       models.AlarmFire,
		ZoneNumber: 2,
	}

	msgs := buildAlarmSourceMessagesFromEvents(alarm, events)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 source messages, got %d", len(msgs))
	}
	if got := msgs[0].Details; got != "Несправність після пожежі" {
		t.Fatalf("newest message details = %q, want %q", got, "Несправність після пожежі")
	}
	if got := msgs[1].Details; got != "Друга пожежа" {
		t.Fatalf("oldest message details = %q, want %q", got, "Друга пожежа")
	}
}

func TestBuildAlarmSourceMessagesFromEvents_FallsBackToAllEventsWithoutGroups(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 11, 0, 0, 0, time.Local)
	events := []models.Event{
		{ID: 1, Time: base, Type: models.EventRestore, Details: "Відновлення", SC1: 5},
		{ID: 2, Time: base.Add(1 * time.Minute), Type: models.SystemEvent, Details: "Системна", SC1: 30},
	}

	msgs := buildAlarmSourceMessagesFromEvents(models.Alarm{}, events)
	if len(msgs) != 2 {
		t.Fatalf("expected fallback to keep all events, got %d", len(msgs))
	}
	if got := msgs[0].Details; got != "Системна" {
		t.Fatalf("newest fallback message = %q, want %q", got, "Системна")
	}
}

func TestBuildAlarmSourceMessagesFromEvents_FiltersOutOlderThanAlarm(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	events := []models.Event{
		{
			ID:         1,
			Time:       base.Add(-1 * time.Minute),
			Type:       models.EventFire,
			ZoneNumber: 3,
			Details:    "Стара тривога",
			SC1:        1,
		},
		{
			ID:         2,
			Time:       base,
			Type:       models.EventFire,
			ZoneNumber: 3,
			Details:    "Поточна тривога",
			SC1:        1,
		},
		{
			ID:         3,
			Time:       base.Add(1 * time.Minute),
			Type:       models.EventFault,
			ZoneNumber: 3,
			Details:    "Подія після тривоги",
			SC1:        2,
		},
	}

	alarm := models.Alarm{
		Time:       base,
		Type:       models.AlarmFire,
		ZoneNumber: 3,
	}

	msgs := buildAlarmSourceMessagesFromEvents(alarm, events)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 filtered source messages, got %d", len(msgs))
	}
	if got := msgs[0].Details; got != "Подія після тривоги" {
		t.Fatalf("newest filtered message = %q, want %q", got, "Подія після тривоги")
	}
	if got := msgs[1].Details; got != "Поточна тривога" {
		t.Fatalf("oldest filtered message = %q, want %q", got, "Поточна тривога")
	}
}
