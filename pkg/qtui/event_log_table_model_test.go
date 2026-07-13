//go:build qt

package qtui

import (
	"reflect"
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestEventLogTableIncludesZoneColumn(t *testing.T) {
	wantHeaders := []string{"Час", "№", "Подія", "Зона", "Об'єкт", "Опис", "Джерело"}
	if got := eventLogHeaders(); !reflect.DeepEqual(got, wantHeaders) {
		t.Fatalf("eventLogHeaders() = %v, want %v", got, wantHeaders)
	}

	event := models.Event{ZoneNumber: 17}
	if got := eventLogCellText(event, 3); got != "17" {
		t.Fatalf("zone cell = %q, want %q", got, "17")
	}
	if got := eventLogCellText(event, 5); got != "Зона 17" {
		t.Fatalf("details cell = %q, want %q", got, "Зона 17")
	}
}
