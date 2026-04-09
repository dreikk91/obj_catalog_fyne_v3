package ui

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestPrepareSourceMessagesForDisplay_BridgeActiveModeKeepsWholeChronology(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	alarm := models.Alarm{
		ObjectID:   1001,
		Time:       base,
		ZoneNumber: 1,
	}
	sourceMsgs := []models.AlarmMsg{
		{Time: base.Add(-1 * time.Minute), Details: "Старіша"},
		{Time: base, Details: "Поточна"},
	}

	out := prepareSourceMessagesForDisplay(alarm, sourceMsgs, config.BridgeAlarmHistoryModeActiveOnly)
	if len(out) != 2 {
		t.Fatalf("expected 2 source messages, got %d", len(out))
	}
	if out[0].Details != "Старіша" {
		t.Fatalf("first message = %q, want %q", out[0].Details, "Старіша")
	}
}

func TestPrepareSourceMessagesForDisplay_BridgeLegacyModeFiltersOlderMessages(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	alarm := models.Alarm{
		ObjectID:   1001,
		Time:       base,
		ZoneNumber: 1,
	}
	sourceMsgs := []models.AlarmMsg{
		{Time: base.Add(-1 * time.Minute), Details: "Старіша"},
		{Time: base, Details: "Поточна"},
	}

	out := prepareSourceMessagesForDisplay(alarm, sourceMsgs, config.BridgeAlarmHistoryModeLegacy)
	if len(out) != 1 {
		t.Fatalf("expected 1 source message, got %d", len(out))
	}
	if out[0].Details != "Поточна" {
		t.Fatalf("remaining message = %q, want %q", out[0].Details, "Поточна")
	}
}

func TestPrepareSourceMessagesForDisplay_PhoenixIgnoresBridgeActiveMode(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	alarm := models.Alarm{
		ObjectID: ids.PhoenixObjectIDNamespaceStart + 1,
		Time:     base,
	}
	sourceMsgs := []models.AlarmMsg{
		{Time: base.Add(-1 * time.Minute), Details: "Старіша"},
		{Time: base, Details: "Поточна"},
	}

	out := prepareSourceMessagesForDisplay(alarm, sourceMsgs, config.BridgeAlarmHistoryModeActiveOnly)
	if len(out) != 1 {
		t.Fatalf("expected 1 source message after legacy-style filtering, got %d", len(out))
	}
	if out[0].Details != "Поточна" {
		t.Fatalf("remaining message = %q, want %q", out[0].Details, "Поточна")
	}
}

func TestPrepareSourceMessagesForDisplay_CASLKeepsWholeCaseChronology(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	alarm := models.Alarm{
		ObjectID: ids.CASLObjectIDNamespaceStart + 1,
		Time:     base,
	}
	sourceMsgs := []models.AlarmMsg{
		{Time: base.Add(-1 * time.Minute), Details: "Початкова причина"},
		{Time: base, Details: "GRD_OBJ_NOTIF"},
		{Time: base.Add(1 * time.Minute), Details: "GRD_OBJ_PICK"},
	}

	out := prepareSourceMessagesForDisplay(alarm, sourceMsgs, config.BridgeAlarmHistoryModeActiveOnly)
	if len(out) != 3 {
		t.Fatalf("expected 3 source messages, got %d", len(out))
	}
	if out[0].Details != "Початкова причина" {
		t.Fatalf("first message = %q, want %q", out[0].Details, "Початкова причина")
	}
}
