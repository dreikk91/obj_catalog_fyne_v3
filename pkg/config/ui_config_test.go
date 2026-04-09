package config

import "testing"

func TestNormalizeBridgeAlarmHistoryMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default empty", input: "", want: BridgeAlarmHistoryModeActiveOnly},
		{name: "active explicit", input: BridgeAlarmHistoryModeActiveOnly, want: BridgeAlarmHistoryModeActiveOnly},
		{name: "legacy explicit", input: BridgeAlarmHistoryModeLegacy, want: BridgeAlarmHistoryModeLegacy},
		{name: "unknown fallback", input: "other", want: BridgeAlarmHistoryModeActiveOnly},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeBridgeAlarmHistoryMode(tt.input); got != tt.want {
				t.Fatalf("NormalizeBridgeAlarmHistoryMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClampSchedulerIntervalSec(t *testing.T) {
	t.Parallel()

	if got := clampSchedulerIntervalSec(0, 15); got != 15 {
		t.Fatalf("clampSchedulerIntervalSec(0, 15) = %d, want 15", got)
	}
	if got := clampSchedulerIntervalSec(-10, 15); got != 15 {
		t.Fatalf("clampSchedulerIntervalSec(-10, 15) = %d, want 15", got)
	}
	if got := clampSchedulerIntervalSec(4000, 15); got != 3600 {
		t.Fatalf("clampSchedulerIntervalSec(4000, 15) = %d, want 3600", got)
	}
	if got := clampSchedulerIntervalSec(30, 15); got != 30 {
		t.Fatalf("clampSchedulerIntervalSec(30, 15) = %d, want 30", got)
	}
}
