package data

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/database"
)

func TestBuildDBActiveAlarmSourceMessages_PreservesWholeActiveChronology(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 4, 9, 12, 0, 0, 0, time.Local)
	objn := int64(1501)
	scFire := 1
	scFault := 2

	rows := []database.ActAlarmsRow{
		{
			EvTime1: ptrTime(base.Add(2 * time.Minute)),
			ObjN:    &objn,
			Zonen:   ptrInt64(2),
			Info1:   ptrString("Грп.02"),
			Ukr1:    ptrString("Несправність"),
			Sc1:     &scFault,
		},
		{
			EvTime1: ptrTime(base),
			ObjN:    &objn,
			Zonen:   ptrInt64(1),
			Info1:   ptrString("Грп.01"),
			Ukr1:    ptrString("Пожежа"),
			Sc1:     &scFire,
		},
	}

	msgs := buildDBActiveAlarmSourceMessages(rows)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 active source messages, got %d", len(msgs))
	}
	if msgs[0].Details != "Несправність (Грп.02)" {
		t.Fatalf("first details = %q, want %q", msgs[0].Details, "Несправність (Грп.02)")
	}
	if msgs[1].Details != "Пожежа (Грп.01)" {
		t.Fatalf("second details = %q, want %q", msgs[1].Details, "Пожежа (Грп.01)")
	}
}

func ptrString(v string) *string {
	return &v
}

func ptrInt64(v int64) *int64 {
	return &v
}
