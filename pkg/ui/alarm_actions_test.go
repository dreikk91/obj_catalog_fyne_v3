package ui

import (
	"strings"
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestFormatAlarmListTextShowsCurrentOperator(t *testing.T) {
	text := formatAlarmListText(models.Alarm{
		ObjectID:     1,
		ObjectNumber: "101",
		IsInProgress: true,
		InProgressBy: "Оператор 7",
	})

	if !strings.Contains(text, "У роботі: Оператор 7") {
		t.Fatalf("formatAlarmListText() = %q, want current operator", text)
	}
}

func TestAlarmTakeActionTextUsesTakeoverForForeignOwner(t *testing.T) {
	text := alarmTakeActionText(models.Alarm{
		IsInProgress: true,
		IsOwnedByMe:  false,
		CanTakeOver:  true,
	})

	if text != "Перехопити тривогу" {
		t.Fatalf("alarmTakeActionText() = %q, want takeover action", text)
	}
}
