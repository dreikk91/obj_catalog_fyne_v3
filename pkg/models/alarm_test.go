package models

import "testing"

func TestAlarmGetObjectNumberDisplay(t *testing.T) {
	t.Run("uses object number when present", func(t *testing.T) {
		alarm := &Alarm{
			ObjectID:     1_500_000_000,
			ObjectNumber: "1004",
		}

		if got := alarm.GetObjectNumberDisplay(); got != "1004" {
			t.Fatalf("GetObjectNumberDisplay() = %q, want %q", got, "1004")
		}
	})

	t.Run("falls back to object id", func(t *testing.T) {
		alarm := &Alarm{ObjectID: 42}

		if got := alarm.GetObjectNumberDisplay(); got != "42" {
			t.Fatalf("GetObjectNumberDisplay() = %q, want %q", got, "42")
		}
	})

	t.Run("handles nil receiver", func(t *testing.T) {
		var alarm *Alarm

		if got := alarm.GetObjectNumberDisplay(); got != "" {
			t.Fatalf("GetObjectNumberDisplay() = %q, want empty string", got)
		}
	})
}
