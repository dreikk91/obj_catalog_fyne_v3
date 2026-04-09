package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestSelectDBAlarmMessage_PrefersLatestAlarmOverNewerRestore(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 8, 12, 0, 0, 0, time.Local)
	messages := []dbAlarmMessage{
		{
			Time:      base.Add(2 * time.Minute),
			EventType: models.EventRestore,
			IsAlarm:   false,
			Details:   "Відновлення",
		},
		{
			Time:      base.Add(1 * time.Minute),
			EventType: models.EventFault,
			IsAlarm:   true,
			Details:   "Несправність",
		},
	}
	sortDBAlarmMessages(messages)

	selected, ok := selectDBAlarmMessage(messages)
	if !ok {
		t.Fatalf("expected selected message")
	}
	if selected.EventType != models.EventFault {
		t.Fatalf("selected type = %s, want %s", selected.EventType, models.EventFault)
	}
	if selected.Details != "Несправність" {
		t.Fatalf("selected details = %q, want %q", selected.Details, "Несправність")
	}
}

func TestSelectDBAlarmMessage_PrefersPrimaryAlarmOverNewerFault(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 8, 13, 0, 0, 0, time.Local)
	messages := []dbAlarmMessage{
		{
			Time:      base.Add(3 * time.Minute),
			EventType: models.EventFault,
			IsAlarm:   true,
			Details:   "Несправність",
		},
		{
			Time:      base.Add(2 * time.Minute),
			EventType: models.EventRestore,
			IsAlarm:   false,
			Details:   "Відновлення",
		},
		{
			Time:      base.Add(1 * time.Minute),
			EventType: models.EventFire,
			IsAlarm:   true,
			Details:   "Перша тривога",
		},
	}
	sortDBAlarmMessages(messages)

	selected, ok := selectDBAlarmMessage(messages)
	if !ok {
		t.Fatalf("expected selected message")
	}
	if selected.Details != "Перша тривога" {
		t.Fatalf("selected details = %q, want %q", selected.Details, "Перша тривога")
	}
}

func TestResolveDBGroupedAlarmSC1_StaysFireWhenLatestIsFaultAfterAlarm(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 8, 13, 30, 0, 0, time.Local)
	messages := []dbAlarmMessage{
		{
			Time:      base.Add(2 * time.Minute),
			EventType: models.EventFault,
			SC1:       2,
		},
		{
			Time:      base.Add(1 * time.Minute),
			EventType: models.EventFire,
			SC1:       1,
		},
	}
	sortDBAlarmMessages(messages)

	if got := resolveDBGroupedAlarmSC1(messages, 0); got != 1 {
		t.Fatalf("resolveDBGroupedAlarmSC1() = %d, want 1", got)
	}
}

func TestMapDBAlarmMessagesToSourceMsgs_PreservesOrderAndFlags(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 8, 14, 0, 0, 0, time.Local)
	messages := []dbAlarmMessage{
		{
			Time:       base.Add(2 * time.Minute),
			Details:    "Актуальна",
			IsAlarm:    true,
			ZoneNumber: 4,
			SC1:        1,
		},
		{
			Time:       base.Add(1 * time.Minute),
			Details:    "Стара",
			IsAlarm:    false,
			ZoneNumber: 4,
			SC1:        5,
		},
	}

	source := mapDBAlarmMessagesToSourceMsgs(messages)
	if len(source) != 2 {
		t.Fatalf("expected 2 source messages, got %d", len(source))
	}
	if !source[0].IsAlarm || source[0].Details != "Актуальна" {
		t.Fatalf("unexpected first source message: %+v", source[0])
	}
	if source[1].IsAlarm || source[1].Details != "Стара" {
		t.Fatalf("unexpected second source message: %+v", source[1])
	}
	if source[0].SC1 != 1 || source[1].SC1 != 5 {
		t.Fatalf("unexpected source SC1 values: %+v", source)
	}
}

func TestMapDBSC1ToEventType_PowerAndBattery(t *testing.T) {
	t.Parallel()

	if got := mapDBSC1ToEventType(3); got != models.EventPowerFail {
		t.Fatalf("mapDBSC1ToEventType(3) = %s, want %s", got, models.EventPowerFail)
	}
	if got := mapDBSC1ToEventType(4); got != models.EventBatteryLow {
		t.Fatalf("mapDBSC1ToEventType(4) = %s, want %s", got, models.EventBatteryLow)
	}
}

func TestIsDBQueryContextCanceled(t *testing.T) {
	t.Parallel()

	if !isDBQueryContextCanceled(context.DeadlineExceeded) {
		t.Fatal("expected deadline exceeded to be treated as canceled")
	}
	if !isDBQueryContextCanceled(context.Canceled) {
		t.Fatal("expected canceled context to be treated as canceled")
	}
	err := errors.New("failed to select active alarm events: operation was cancelled")
	if !isDBQueryContextCanceled(err) {
		t.Fatal("expected operation was cancelled to be treated as canceled")
	}
	if isDBQueryContextCanceled(errors.New("some other db error")) {
		t.Fatal("unexpected canceled detection for generic error")
	}
}

func TestFormatDBObjectName(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "Офіс Регіональної служби ветконтролю на кордоні"
	got := formatDBObjectName(&number, &title)
	want := "Офіс Регіональної служби ветконтролю на кордоні"
	if got != want {
		t.Fatalf("unexpected formatted object name: got %q, want %q", got, want)
	}
}

func TestFormatDBObjectName_AlreadyPrefixed(t *testing.T) {
	t.Parallel()

	number := int64(1003)
	title := "Офіс"
	got := formatDBObjectName(&number, &title)
	if got != title {
		t.Fatalf("must keep already prefixed name, got %q", got)
	}
}

func TestReverseDBEvents(t *testing.T) {
	t.Parallel()

	events := []models.Event{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	reverseDBEvents(events)

	if events[0].ID != 3 || events[1].ID != 2 || events[2].ID != 1 {
		t.Fatalf("unexpected reverse order: %+v", events)
	}
}

func TestMaxDBEventRowID(t *testing.T) {
	t.Parallel()

	rows := []database.EventRow{
		{ID: 101},
		{ID: 150},
		{ID: 120},
	}

	if got := maxDBEventRowID(rows, 99); got != 150 {
		t.Fatalf("maxDBEventRowID() = %d, want 150", got)
	}
}

func TestMapDBEventRowsPreservesInputOrder(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 6, 10, 0, 0, 0, time.Local)
	objn1 := int64(1001)
	objn2 := int64(1002)
	sc1 := 1

	rows := []database.EventRow{
		{ID: 20, ObjN: &objn2, EvTime1: ptrTime(now.Add(-1 * time.Minute)), Sc1: &sc1},
		{ID: 10, ObjN: &objn1, EvTime1: ptrTime(now.Add(-2 * time.Minute)), Sc1: &sc1},
	}

	events := mapDBEventRows(rows, 0)

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].ID != 20 || events[1].ID != 10 {
		t.Fatalf("mapDBEventRows() changed order: %+v", events)
	}
}

func TestMapObjectRowToModel_PreservesSIMPhones(t *testing.T) {
	t.Parallel()

	sim1 := "+380501234567"
	sim2 := "+380671234567"
	row := database.ObjectInfoRow{
		Objn:      1001,
		GsmPhone:  &sim1,
		GsmPhone2: &sim2,
	}

	obj := mapObjectRowToModel(row)
	if obj.SIM1 != sim1 {
		t.Fatalf("SIM1 = %q, want %q", obj.SIM1, sim1)
	}
	if obj.SIM2 != sim2 {
		t.Fatalf("SIM2 = %q, want %q", obj.SIM2, sim2)
	}
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
