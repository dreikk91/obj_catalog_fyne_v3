package ui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectListRowColors_PriorityBlockedOverAlarm(t *testing.T) {
	item := models.Object{
		BlockedArmedOnOff: 1,
		AlarmState:        1,
		Status:            models.StatusFire,
	}

	text, row := objectListRowColors(item, false)
	if text.R != 255 || row.R != 144 {
		t.Fatalf("unexpected blocked colors (light): text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_PhoenixBlockedUsesDedicatedPalette(t *testing.T) {
	item := models.Object{
		ID:                1000000077,
		BlockedArmedOnOff: 1,
		Status:            models.StatusNormal,
	}

	text, row := objectListRowColors(item, false)
	if text.R != 255 || row.R != 79 || row.G != 109 || row.B != 135 {
		t.Fatalf("unexpected phoenix blocked colors (light): text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_PhoenixDisarmedUsesDedicatedPalette(t *testing.T) {
	item := models.Object{
		ID:         1000000078,
		GuardState: 0,
		Status:     models.StatusNormal,
	}

	text, row := objectListRowColors(item, false)
	if text.R != 255 || row.R != 67 || row.G != 156 || row.B != 199 {
		t.Fatalf("unexpected phoenix disarmed colors (light): text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_OfflinePriority(t *testing.T) {
	item := models.Object{
		IsConnState: 0,
		Status:      models.StatusOffline,
	}

	_, row := objectListRowColors(item, false)
	if row.G != 235 {
		t.Fatalf("unexpected offline row color: %+v", row)
	}
}

func TestObjectListRowColors_CASLAssignmentWarning(t *testing.T) {
	item := models.Object{
		ID:            1500000010,
		HasAssignment: false,
		IsConnState:   1,
		Status:        models.StatusNormal,
	}

	text, row := objectListRowColors(item, false)
	if text.R != 255 || row.B != 168 {
		t.Fatalf("unexpected casl assignment colors: text=%+v row=%+v", text, row)
	}
}
