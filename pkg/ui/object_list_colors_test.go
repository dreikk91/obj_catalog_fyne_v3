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
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected blocked colors (light): text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}
}

func TestObjectListRowColors_BridgeDisarmedUsesSemanticPalette(t *testing.T) {
	item := models.Object{
		ID:               1559,
		GuardStatus:      models.GuardStatusDisarmed,
		ConnectionStatus: models.ConnectionStatusOnline,
		MonitoringStatus: models.MonitoringStatusActive,
		Status:           models.StatusNormal,
		SubServerA:       "A",
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected disarmed colors: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}
}

func TestObjectListRowColors_BridgeGreenRequiresConnectedAndGuarded(t *testing.T) {
	vm := viewmodels.NewObjectListViewModel()
	item := models.Object{
		ID:               1559,
		ConnectionStatus: models.ConnectionStatusUnknown,
		GuardStatus:      models.GuardStatusGuarded,
		MonitoringStatus: models.MonitoringStatusActive,
		Status:           models.StatusNormal,
		SubServerA:       "A",
	}

	_, row := vm.GetRowColors(item, false)
	_, normalRow := utils.SelectObjectColorNRGBA(10)
	if row == normalRow {
		t.Fatalf("bridge object with unknown connection must not be green: %+v", row)
	}
}

func TestObjectListRowColors_PhoenixBlockedUsesSemanticPalette(t *testing.T) {
	item := models.Object{
		ID:               1000000077,
		MonitoringStatus: models.MonitoringStatusBlocked,
		Status:           models.StatusNormal,
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected phoenix blocked colors: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}
}

func TestObjectListRowColors_PhoenixDisarmedUsesSemanticPalette(t *testing.T) {
	item := models.Object{
		ID:               1000000078,
		MonitoringStatus: models.MonitoringStatusActive,
		GuardStatus:      models.GuardStatusDisarmed,
		Status:           models.StatusNormal,
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected phoenix disarmed colors: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}
}

func TestObjectListRowColors_PhoenixOnlineGuardedDoesNotUseYellow(t *testing.T) {
	item := models.Object{
		ID:               1000000079,
		ConnectionStatus: models.ConnectionStatusOnline,
		IsConnState:      1,
		IsConnOK:         true,
		GuardStatus:      models.GuardStatusGuarded,
		Status:           models.StatusNormal,
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	_, warningRow := utils.SelectObjectColorNRGBA(4)
	if row == warningRow {
		t.Fatalf("Phoenix online guarded object got colored yellow as if offline: text=%+v row=%+v", text, row)
	}
}

func TestObjectListRowColors_OfflinePriority(t *testing.T) {
	item := models.Object{
		ConnectionStatus: models.ConnectionStatusOffline,
		Status:           models.StatusOffline,
	}

	vm := viewmodels.NewObjectListViewModel()
	text, row := vm.GetRowColors(item, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected offline colors: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}
}

func TestObjectListRowColors_CASLAssignmentWarning(t *testing.T) {
	item := models.Object{
		ID:               1500000010,
		HasAssignment:    false,
		ConnectionStatus: models.ConnectionStatusOnline,
		Status:           models.StatusNormal,
	}

	text, row := viewmodels.NewObjectListViewModel().GetRowColors(item, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected CASL assignment colors: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
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

func TestObjectListRowColors_UsesNormalizedStatuses(t *testing.T) {
	vm := viewmodels.NewObjectListViewModel()

	text, row := vm.GetRowColors(models.Object{
		MonitoringStatus: models.MonitoringStatusBlocked,
		Status:           models.StatusNormal,
	}, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(4)
	if text != wantText || row != wantRow {
		t.Fatalf("unexpected blocked colors from normalized state: text=%+v row=%+v", text, row)
	}

	offlineText, offlineRow := vm.GetRowColors(models.Object{
		ConnectionStatus: models.ConnectionStatusOffline,
		GuardStatus:      models.GuardStatusGuarded,
		Status:           models.StatusNormal,
	}, false)
	wantOfflineText, wantOfflineRow := utils.SelectObjectColorNRGBA(4)
	if offlineText != wantOfflineText || offlineRow != wantOfflineRow {
		t.Fatalf(
			"unexpected offline colors from normalized state: text=%+v row=%+v want text=%+v row=%+v",
			offlineText,
			offlineRow,
			wantOfflineText,
			wantOfflineRow,
		)
	}
}
