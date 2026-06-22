//go:build qt

package qtui

import (
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestNormalizeDialogAlarmOptionsSkipsInvisibleOptions(t *testing.T) {
	got := normalizeDialogAlarmOptions([]contracts.AlarmProcessingOption{
		{Code: " 10 ", Label: ""},
		{Code: "", Label: "Manual reason"},
		{Code: "", Label: ""},
	})

	if len(got) != 2 {
		t.Fatalf("len(options) = %d, want 2: %+v", len(got), got)
	}
	if got[0].Code != "10" || got[0].Label != "10" {
		t.Fatalf("first option = %+v, want trimmed code as fallback label", got[0])
	}
	if got[1].Code != "" || got[1].Label != "Manual reason" {
		t.Fatalf("second option = %+v, want label-only option", got[1])
	}
}

func TestAlarmProcessSummaryIncludesEachSelectedAlarm(t *testing.T) {
	alarms := []models.Alarm{
		{
			ObjectID:     101,
			ObjectNumber: "101",
			ObjectName:   "Alpha",
			Type:         models.AlarmFire,
			Time:         time.Date(2026, 6, 22, 8, 30, 0, 0, time.Local),
			ZoneNumber:   7,
			ZoneName:     "Kitchen",
		},
		{
			ObjectID:     102,
			ObjectNumber: "102",
			ObjectName:   "Beta",
			Type:         models.AlarmFault,
			Time:         time.Date(2026, 6, 22, 8, 31, 0, 0, time.Local),
		},
	}

	got := alarmProcessSummary(alarms)
	for _, want := range []string{"№101 Alpha", "Kitchen", "№102 Beta"} {
		if !strings.Contains(got, want) {
			t.Fatalf("summary %q does not contain %q", got, want)
		}
	}
	if lines := strings.Count(got, "\n") + 1; lines != 2 {
		t.Fatalf("summary line count = %d, want 2: %q", lines, got)
	}
}
