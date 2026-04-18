package dialogs

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

func TestAdminDisplayBlockObjectColorsUsesNormalizedStatuses(t *testing.T) {
	text, row := adminDisplayBlockObjectColors(adminv1.DisplayBlockObject{
		BlockMode:        adminv1.DisplayBlockModeTemporaryOff,
		MonitoringStatus: frontendv1.MonitoringStatusBlocked,
	}, false)
	if text.R != 255 || row.R != 144 {
		t.Fatalf("blocked colors mismatch: text=%+v row=%+v", text, row)
	}

	_, offlineRow := adminDisplayBlockObjectColors(adminv1.DisplayBlockObject{
		ConnectionStatus: frontendv1.ConnectionStatusOffline,
		GuardStatus:      frontendv1.GuardStatusGuarded,
	}, false)
	if offlineRow.G != 235 {
		t.Fatalf("offline colors mismatch: row=%+v", offlineRow)
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
