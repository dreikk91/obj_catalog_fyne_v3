//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestContextMenuAlarmsUsesSelectionWhenClickedAlarmIsSelected(t *testing.T) {
	selected := []models.Alarm{{ID: 1}, {ID: 2}, {ID: 3}}

	got := contextMenuAlarms([]models.Alarm{{ID: 2}, {ID: 4}}, selected)

	if len(got) != len(selected) {
		t.Fatalf("len(context alarms) = %d, want %d", len(got), len(selected))
	}
	for i := range selected {
		if got[i].ID != selected[i].ID {
			t.Fatalf("alarm[%d].ID = %d, want %d", i, got[i].ID, selected[i].ID)
		}
	}
}

func TestContextMenuAlarmsFallsBackToClickedAlarmOutsideSelection(t *testing.T) {
	got := contextMenuAlarms([]models.Alarm{{ID: 9}, {ID: 10}}, []models.Alarm{{ID: 1}, {ID: 2}})

	if len(got) != 2 || got[0].ID != 9 || got[1].ID != 10 {
		t.Fatalf("context alarms = %+v, want clicked group alarms 9 and 10", got)
	}
}

func TestAlarmActionTextIncludesCountForGroup(t *testing.T) {
	got := alarmActionText("Відпрацювати", []models.Alarm{{ID: 1}, {ID: 2}})

	if got != "Відпрацювати (2)" {
		t.Fatalf("alarmActionText() = %q, want group count", got)
	}
}
