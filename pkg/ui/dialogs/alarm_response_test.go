package dialogs

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestAlarmResponseOperatorAndTakeoverLabels(t *testing.T) {
	alarm := models.Alarm{
		IsInProgress: true,
		InProgressBy: "Оператор 2",
		CanTakeOver:  true,
	}

	if got := alarmOperatorStateText(alarm); got != "У роботі: Оператор 2" {
		t.Fatalf("alarmOperatorStateText() = %q", got)
	}
	if got := alarmTakeButtonText(alarm); got != "Перехопити тривогу" {
		t.Fatalf("alarmTakeButtonText() = %q", got)
	}
}

func TestResponseGroupSelectableAllowsFreeAndCurrentGroups(t *testing.T) {
	if !responseGroupSelectable(contracts.FrontendResponseGroup{
		ID:     "free",
		Status: contracts.ResponseGroupStatusFree,
	}, "") {
		t.Fatal("free response group must be selectable")
	}
	if !responseGroupSelectable(contracts.FrontendResponseGroup{
		ID:     "current",
		Status: contracts.ResponseGroupStatusArrived,
	}, "current") {
		t.Fatal("currently assigned response group must remain selectable")
	}
	if responseGroupSelectable(contracts.FrontendResponseGroup{
		ID:     "busy",
		Status: contracts.ResponseGroupStatusDispatched,
	}, "current") {
		t.Fatal("response group dispatched to another object must not be selectable")
	}
}

func TestAlarmResponseActionsRequireCASLOrPhoenixOwnership(t *testing.T) {
	if alarmResponseActionsAllowed(models.Alarm{ObjectID: ids.CASLObjectIDNamespaceStart}) {
		t.Fatal("unowned CASL alarm must not allow response actions")
	}
	if alarmResponseActionsAllowed(models.Alarm{ObjectID: ids.PhoenixObjectIDNamespaceStart}) {
		t.Fatal("unowned Phoenix alarm must not allow response actions")
	}
	if !alarmResponseActionsAllowed(models.Alarm{
		ObjectID:    ids.CASLObjectIDNamespaceStart,
		IsOwnedByMe: true,
	}) {
		t.Fatal("owned CASL alarm must allow response actions")
	}
}
