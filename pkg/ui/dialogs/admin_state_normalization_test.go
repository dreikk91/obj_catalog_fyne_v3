package dialogs

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
	"obj_catalog_fyne_v3/pkg/utils"
)

func TestAdminDisplayBlockObjectColorsUsesNormalizedStatuses(t *testing.T) {
	text, row := adminDisplayBlockObjectColors(adminv1.DisplayBlockObject{
		BlockMode:        adminv1.DisplayBlockModeTemporaryOff,
		MonitoringStatus: frontendv1.MonitoringStatusBlocked,
	}, false)
	wantText, wantRow := utils.SelectObjectColorNRGBA(utils.ObjectColorBlocked)
	if text != wantText || row != wantRow {
		t.Fatalf("blocked colors mismatch: text=%+v row=%+v want text=%+v row=%+v", text, row, wantText, wantRow)
	}

	offlineText, offlineRow := adminDisplayBlockObjectColors(adminv1.DisplayBlockObject{
		ConnectionStatus: frontendv1.ConnectionStatusOffline,
		GuardStatus:      frontendv1.GuardStatusGuarded,
	}, false)
	wantOfflineText, wantOfflineRow := utils.SelectObjectColorNRGBA(4)
	if offlineText != wantOfflineText || offlineRow != wantOfflineRow {
		t.Fatalf("offline colors mismatch: text=%+v row=%+v want text=%+v row=%+v", offlineText, offlineRow, wantOfflineText, wantOfflineRow)
	}
	if text == offlineText && row == offlineRow {
		t.Fatalf("blocked and offline colors must differ: text=%+v row=%+v", text, row)
	}

	disarmedText, disarmedRow := adminDisplayBlockObjectColors(adminv1.DisplayBlockObject{
		ConnectionStatus: frontendv1.ConnectionStatusOnline,
		GuardStatus:      frontendv1.GuardStatusDisarmed,
	}, false)
	wantDisarmedText, wantDisarmedRow := utils.SelectObjectColorNRGBA(utils.ObjectColorDisarmed)
	if disarmedText != wantDisarmedText || disarmedRow != wantDisarmedRow {
		t.Fatalf("disarmed colors mismatch: text=%+v row=%+v want text=%+v row=%+v", disarmedText, disarmedRow, wantDisarmedText, wantDisarmedRow)
	}
	if offlineText == disarmedText && offlineRow == disarmedRow {
		t.Fatalf("offline and disarmed colors must differ: text=%+v row=%+v", offlineText, offlineRow)
	}
}

func TestStatisticsCaptionsUseNormalizedStatuses(t *testing.T) {
	if got := guardStatusCaption(frontendv1.GuardStatusDisarmed, 0); got != "0 (знято)" {
		t.Fatalf("guardStatusCaption() = %q", got)
	}
	if got := connectionStatusCaption(frontendv1.ConnectionStatusOnline); got != "є зв'язок" {
		t.Fatalf("connectionStatusCaption() = %q", got)
	}
	if got := adminGuardSortRank(frontendv1.GuardStatusGuarded); got != 1 {
		t.Fatalf("adminGuardSortRank() = %d", got)
	}
	if got := adminConnectionSortRank(frontendv1.ConnectionStatusOffline); got != 0 {
		t.Fatalf("adminConnectionSortRank() = %d", got)
	}
}
