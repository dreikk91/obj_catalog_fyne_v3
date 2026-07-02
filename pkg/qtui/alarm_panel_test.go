//go:build qt

package qtui

import (
	"strings"
	"testing"
	"time"

	qt "github.com/mappu/miqt/qt6"

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

func TestAlarmMessageHistoryTreeRowSeparatesColumns(t *testing.T) {
	msg := models.AlarmMsg{
		Time:      time.Date(2026, 7, 2, 10, 21, 58, 0, time.Local),
		Code:      "GRD_OBJ_FINISH",
		ContactID: "401",
		Number:    7,
		Details:   "Завершення відпрацювання тривоги",
		IsAlarm:   true,
	}

	row := alarmMessageHistoryTreeRow(msg, "#111111", "#ffffff")

	if row.Time != "02.07.2026 10:21:58" {
		t.Fatalf("row.Time = %q", row.Time)
	}
	if row.Event != "Тривога — Завершення відпрацювання тривоги" {
		t.Fatalf("row.Event = %q", row.Event)
	}
	if row.Context != "GRD_OBJ_FINISH · CID 401" || row.Zone != "7" {
		t.Fatalf("unexpected context columns: %+v", row)
	}
}

func TestHistoryTreeCountLabelUsesUkrainianPlural(t *testing.T) {
	tests := map[int]string{
		1:  "1 подія",
		2:  "2 події",
		5:  "5 подій",
		11: "11 подій",
		21: "21 подія",
	}
	for count, want := range tests {
		if got := historyTreeCountLabel(count); got != want {
			t.Fatalf("historyTreeCountLabel(%d) = %q, want %q", count, got, want)
		}
	}
}

func TestCaseHistorySplitterSizesMakesHistoryVisible(t *testing.T) {
	got := caseHistorySplitterSizes([]int{900, 0})
	if len(got) != 2 || got[0] != 600 || got[1] != 300 {
		t.Fatalf("caseHistorySplitterSizes() = %v, want [600 300]", got)
	}

	got = caseHistorySplitterSizes(nil)
	if len(got) != 2 || got[1] <= 0 {
		t.Fatalf("fallback splitter sizes must expose history: %v", got)
	}
}

func TestAlarmPanelAlarmAtIndexReturnsExactTreeChild(t *testing.T) {
	model := qt.NewQStandardItemModel2(0, 1)
	parentItem := newReadOnlyItem("Об'єкт")
	parentItem.SetData(qt.NewQVariant14("bridge:101"), int(qt.UserRole))
	parentItem.SetData(qt.NewQVariant4(0), int(qt.UserRole)+1)
	childItem := newReadOnlyItem("Тривога 2")
	childItem.SetData(qt.NewQVariant14("bridge:101"), int(qt.UserRole))
	childItem.SetData(qt.NewQVariant4(2), int(qt.UserRole)+1)
	parentItem.AppendRowWithItem(childItem)
	model.AppendRowWithItem(parentItem)

	primary := models.Alarm{ID: 1, ObjectID: 101}
	exact := models.Alarm{ID: 2, ObjectID: 101, ZoneNumber: 7}
	panel := &AlarmPanel{
		model:      model,
		alarmsByID: map[int]models.Alarm{1: primary, 2: exact},
		groupsByKey: map[string]alarmGroup{
			"bridge:101": {Key: "bridge:101", Primary: primary, Alarms: []models.Alarm{primary, exact}},
		},
	}

	parentIndex := parentItem.Index()
	childIndex := model.Index(0, 0, parentIndex)
	got, ok := panel.alarmAtIndex(childIndex)
	if !ok || got.ID != 2 || got.ZoneNumber != 7 {
		t.Fatalf("alarmAtIndex(child) = %+v, %v; want exact alarm 2", got, ok)
	}

	got, ok = panel.alarmAtIndex(parentIndex)
	if !ok || got.ID != 1 {
		t.Fatalf("alarmAtIndex(parent) = %+v, %v; want primary alarm 1", got, ok)
	}
}

func TestAlarmTreeChildValuesShowsEventDetails(t *testing.T) {
	alarm := models.Alarm{
		ObjectID:     101,
		ZoneNumber:   7,
		Type:         models.AlarmFire,
		Details:      "Пожежа у серверній",
		IsInProgress: true,
		InProgressBy: "Оператор",
	}
	values := alarmTreeChildValues(alarm)
	if len(values) != len(alarmGroupHeaders()) {
		t.Fatalf("child columns = %d, want %d", len(values), len(alarmGroupHeaders()))
	}
	if values[2] != "↳ Зона 7" || !strings.Contains(values[3], "Пожежа у серверній") || values[4] != "Оператор" {
		t.Fatalf("unexpected child values: %v", values)
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

func TestAlarmPickActionShowsTakeover(t *testing.T) {
	alarms := []models.Alarm{{
		IsInProgress: true,
		IsOwnedByMe:  false,
		CanTakeOver:  true,
	}}
	if got := alarmPickActionVerb(alarms); got != "Перехопити" {
		t.Fatalf("alarmPickActionVerb() = %q", got)
	}
	if !alarmsRequireTakeover(alarms) {
		t.Fatal("foreign alarm must require takeover confirmation")
	}
}

func TestAlarmGroupOperatorTextShowsCurrentOwner(t *testing.T) {
	got := alarmGroupOperatorText(alarmGroup{
		Alarms: []models.Alarm{{
			IsInProgress: true,
			InProgressBy: "Оператор 4",
		}},
	})
	if got != "Оператор 4" {
		t.Fatalf("alarmGroupOperatorText() = %q", got)
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
