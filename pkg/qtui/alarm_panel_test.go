//go:build qt

package qtui

import (
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
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

func TestAlarmResponseStateShowsArrivedGroup(t *testing.T) {
	got := alarmResponseState(models.Alarm{
		ResponseGroupID:           "17",
		IsResponseGroupDispatched: true,
		IsResponseGroupArrived:    true,
	})
	if got != "МГР прибула (17)" {
		t.Fatalf("alarmResponseState() = %q", got)
	}
}

func TestResponseGroupLabelIncludesOperationalContacts(t *testing.T) {
	got := responseGroupLabel(contracts.FrontendResponseGroup{
		ID:       "7",
		Name:     "Група Захід",
		Callsign: "Беркут",
		Phone:    "+380671234567",
	})
	want := "Група Захід | позивний Беркут | +380671234567"
	if got != want {
		t.Fatalf("responseGroupLabel() = %q, want %q", got, want)
	}
}

func TestAlarmResponseHistoryHTMLIncludesCaseMessages(t *testing.T) {
	history := []models.AlarmMsg{
		{
			Time:    time.Date(2026, 6, 28, 14, 5, 0, 0, time.Local),
			Code:    "FIRE",
			Details: "Пожежа у зоні 3",
			SC1:     1,
			IsAlarm: true,
		},
	}
	html := alarmResponseHistoryHTML(models.Alarm{
		ObjectID:   101,
		Time:       history[0].Time,
		ZoneNumber: 3,
	}, history)
	if !strings.Contains(html, "Пожежа у зоні 3") {
		t.Fatalf("history HTML does not contain message: %s", html)
	}
	if !strings.Contains(html, "font-weight:bold") {
		t.Fatalf("alarm history row is not emphasized: %s", html)
	}
}

func TestAlarmResponseHistoryHTMLShowsEmptyState(t *testing.T) {
	html := alarmResponseHistoryHTML(models.Alarm{}, nil)
	if !strings.Contains(html, "Додаткових подій кейсу немає") {
		t.Fatalf("empty history HTML = %q", html)
	}
}

func TestAlarmActionCapabilitiesRequireEverySelectedAlarm(t *testing.T) {
	alarms := []models.Alarm{
		{CanProcess: true},
		{CanProcess: false},
	}
	if canProcessAlarms(alarms) {
		t.Fatal("processing must be disabled when one selected alarm cannot be processed")
	}

	alarms = []models.Alarm{
		{IsInProgress: false},
		{IsInProgress: true, CanTakeOver: true},
	}
	if !canTakeAlarms(alarms) {
		t.Fatal("taking must be enabled when every selected alarm is new or can be taken over")
	}
}

func TestAlarmPickActionShowsCASLTakeover(t *testing.T) {
	alarms := []models.Alarm{{
		IsInProgress: true,
		IsOwnedByMe:  false,
		CanTakeOver:  true,
	}}
	if got := alarmPickActionVerb(alarms); got != "Перехопити" {
		t.Fatalf("alarmPickActionVerb() = %q", got)
	}
	if !alarmsRequireTakeover(alarms) {
		t.Fatal("foreign CASL alarm must require takeover confirmation")
	}
}

func TestSelectableResponseGroupsExcludesBusyGroups(t *testing.T) {
	groups := []contracts.FrontendResponseGroup{
		{ID: "free", Status: contracts.ResponseGroupStatusFree},
		{ID: "busy", Status: contracts.ResponseGroupStatusDispatched},
		{ID: "unknown", Status: contracts.ResponseGroupStatusUnknown},
	}
	got := selectableResponseGroups(models.Alarm{}, groups)
	if len(got) != 2 || got[0].ID != "free" || got[1].ID != "unknown" {
		t.Fatalf("selectable groups = %+v", got)
	}
}
