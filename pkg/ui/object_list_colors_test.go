package ui

import (
	"image/color"
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/utils"
)

func TestObjectListRowColors_PriorityBlockedOverAlarm(t *testing.T) {
	item := models.Object{
		BlockedArmedOnOff: 1,
		AlarmState:        1,
		Status:            models.StatusFire,
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
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

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
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

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	if text.R != 255 || row.R != 67 || row.G != 156 || row.B != 199 {
		t.Fatalf("unexpected phoenix disarmed colors (light): text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_OfflinePriority(t *testing.T) {
	item := models.Object{
		IsConnState: 0,
		Status:      models.StatusOffline,
	}

	vm := viewmodels.NewObjectListViewModel()
	_, row := vm.GetRowColors(item, false)
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

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	if text.R != 255 || row.B != 168 {
		t.Fatalf("unexpected casl assignment colors: text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_DoNotFollowCustomizedEventPalette_Light(t *testing.T) {
	utils.ResetEventColorsToDefault(false)
	t.Cleanup(func() {
		utils.ResetEventColorsToDefault(false)
	})

	item := models.Object{
		AlarmState: 1,
		Status:     models.StatusFire,
	}
	vm := viewmodels.NewObjectListViewModel()
	wantText, wantRow := vm.GetRowColors(item, false)

	utils.SetEventTextColor(1, false, color.NRGBA{R: 1, G: 2, B: 3, A: 255})
	utils.SetEventRowColor(1, false, color.NRGBA{R: 4, G: 5, B: 6, A: 255})

	gotText, gotRow := vm.GetRowColors(item, false)
	if gotText != wantText || gotRow != wantRow {
		t.Fatalf("object colors must ignore customized event palette: got text=%+v row=%+v want text=%+v row=%+v", gotText, gotRow, wantText, wantRow)
	}
}

func TestObjectListRowColors_DoNotFollowCustomizedEventPalette_Dark(t *testing.T) {
	utils.ResetEventColorsToDefault(true)
	t.Cleanup(func() {
		utils.ResetEventColorsToDefault(true)
	})

	item := models.Object{
		AlarmState: 1,
		Status:     models.StatusFire,
	}
	vm := viewmodels.NewObjectListViewModel()
	wantText, wantRow := vm.GetRowColors(item, true)

	utils.SetEventTextColor(1, true, color.NRGBA{R: 11, G: 12, B: 13, A: 255})
	utils.SetEventRowColor(1, true, color.NRGBA{R: 14, G: 15, B: 16, A: 255})

	gotText, gotRow := vm.GetRowColors(item, true)
	if gotText != wantText || gotRow != wantRow {
		t.Fatalf("dark object colors must ignore customized event palette: got text=%+v row=%+v want text=%+v row=%+v", gotText, gotRow, wantText, wantRow)
	}
}
