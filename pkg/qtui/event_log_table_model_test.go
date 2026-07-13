//go:build qt

package qtui

import (
	"reflect"
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestEventLogTableIncludesZoneColumn(t *testing.T) {
	wantHeaders := []string{"Час", "№", "Подія", "Об'єкт", "Опис", "Джерело"}
	if got := eventLogHeaders(); !reflect.DeepEqual(got, wantHeaders) {
		t.Fatalf("eventLogHeaders() = %v, want %v", got, wantHeaders)
	}

	event := models.Event{Type: models.EventFire, ZoneNumber: 17, ZoneName: "Склад"}
	if got := eventLogCellText(event, 3); got != "" {
		t.Fatalf("object cell = %q, want empty string", got)
	}
	if got := eventLogCellText(event, 4); got != "Склад" {
		t.Fatalf("details cell = %q, want %q", got, "Склад")
	}
}
