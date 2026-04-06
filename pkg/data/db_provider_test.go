package data

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestFormatDBObjectName(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "Офіс Регіональної служби ветконтролю на кордоні"
	got := formatDBObjectName(&number, &title)
	want := "Офіс Регіональної служби ветконтролю на кордоні"
	if got != want {
		t.Fatalf("unexpected formatted object name: got %q, want %q", got, want)
	}
}

func TestFormatDBObjectName_AlreadyPrefixed(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "Офіс"
	got := formatDBObjectName(&number, &title)
	if got != title {
		t.Fatalf("must keep already prefixed name, got %q", got)
	}
}

func TestReverseDBEvents(t *testing.T) {
	t.Parallel()

	events := []models.Event{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	reverseDBEvents(events)

	if events[0].ID != 3 || events[1].ID != 2 || events[2].ID != 1 {
		t.Fatalf("unexpected reverse order: %+v", events)
	}
}

func TestMaxDBEventRowID(t *testing.T) {
	t.Parallel()

	rows := []database.EventRow{
		{ID: 101},
		{ID: 150},
		{ID: 120},
	}

	if got := maxDBEventRowID(rows, 99); got != 150 {
		t.Fatalf("maxDBEventRowID() = %d, want 150", got)
	}
}

func TestMapDBEventRowsPreservesInputOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.Local)
	objn1 := int64(1001)
	objn2 := int64(1002)
	sc1 := 1

	rows := []database.EventRow{
		{ID: 20, ObjN: &objn2, EvTime1: ptrTime(now.Add(-1 * time.Minute)), Sc1: &sc1},
		{ID: 10, ObjN: &objn1, EvTime1: ptrTime(now.Add(-2 * time.Minute)), Sc1: &sc1},
	}

	events := mapDBEventRows(rows, 0)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != 20 || events[1].ID != 10 {
		t.Fatalf("mapDBEventRows() changed order: %+v", events)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
