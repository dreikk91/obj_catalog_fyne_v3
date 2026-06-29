package utils

import (
	"image/color"
	"testing"
)

func TestSelectColorNRGBA_SemanticLevelsAreDifferent_Light(t *testing.T) {
	ResetEventColorsToDefault(false)

	// Verify that all 5 semantic levels have distinct row colors.
	_, criticalRow := SelectColorNRGBA(1) // Critical
	_, alarmRow := SelectColorNRGBA(2)    // Alarm
	_, warningRow := SelectColorNRGBA(4)  // Warning
	_, normalRow := SelectColorNRGBA(10)  // Normal
	_, infoRow := SelectColorNRGBA(6)     // Info

	rows := map[string]color.NRGBA{
		"critical": criticalRow,
		"alarm":    alarmRow,
		"warning":  warningRow,
		"normal":   normalRow,
		"info":     infoRow,
	}

	names := []string{"critical", "alarm", "warning", "normal", "info"}
	for i, a := range names {
		for j, b := range names {
			if i >= j {
				continue
			}
			if rows[a] == rows[b] {
				t.Errorf("semantic levels %q and %q share the same row color: %+v", a, b, rows[a])
			}
		}
	}
}

func TestSelectColorNRGBA_SemanticLevelsAreDifferent_Dark(t *testing.T) {
	ResetEventColorsToDefault(true)

	_, criticalRow := SelectColorNRGBADark(1)
	_, alarmRow := SelectColorNRGBADark(2)
	_, warningRow := SelectColorNRGBADark(4)
	_, normalRow := SelectColorNRGBADark(10)
	_, infoRow := SelectColorNRGBADark(6)

	rows := map[string]color.NRGBA{
		"critical": criticalRow,
		"alarm":    alarmRow,
		"warning":  warningRow,
		"normal":   normalRow,
		"info":     infoRow,
	}

	names := []string{"critical", "alarm", "warning", "normal", "info"}
	for i, a := range names {
		for j, b := range names {
			if i >= j {
				continue
			}
			if rows[a] == rows[b] {
				t.Errorf("semantic levels %q and %q share the same row color: %+v", a, b, rows[a])
			}
		}
	}
}

func TestSelectColorNRGBA_SameGroupSharesColor_Light(t *testing.T) {
	ResetEventColorsToDefault(false)

	// All critical codes should share the same row color.
	_, alarm1 := SelectColorNRGBA(1)
	_, panic21 := SelectColorNRGBA(21)
	_, burglary22 := SelectColorNRGBA(22)
	_, medical23 := SelectColorNRGBA(23)

	if alarm1 != panic21 {
		t.Errorf("alarm (1) and panic (21) should share row color, got %+v vs %+v", alarm1, panic21)
	}
	if alarm1 != burglary22 {
		t.Errorf("alarm (1) and burglary (22) should share row color, got %+v vs %+v", alarm1, burglary22)
	}
	if alarm1 != medical23 {
		t.Errorf("alarm (1) and medical (23) should share row color, got %+v vs %+v", alarm1, medical23)
	}
}

func TestSetEventTextColor_ChangesOnlyText_Light(t *testing.T) {
	ResetEventColorsToDefault(false)

	beforeText, beforeRow := SelectColorNRGBA(1)
	newText := color.NRGBA{R: 10, G: 20, B: 30, A: 255}
	SetEventTextColor(1, false, newText)
	afterText, afterRow := SelectColorNRGBA(1)

	if afterText != newText {
		t.Fatalf("text = %+v, want %+v", afterText, newText)
	}
	if afterRow != beforeRow {
		t.Fatalf("row changed unexpectedly: before=%+v after=%+v", beforeRow, afterRow)
	}
	if beforeText == afterText {
		t.Fatalf("text color did not change")
	}
}

func TestSetEventTextColor_ChangesOnlyText_Dark(t *testing.T) {
	ResetEventColorsToDefault(true)

	_, beforeRow := SelectColorNRGBADark(21)
	newText := color.NRGBA{R: 200, G: 210, B: 220, A: 255}
	SetEventTextColor(21, true, newText)
	afterText, afterRow := SelectColorNRGBADark(21)

	if afterText != newText {
		t.Fatalf("text = %+v, want %+v", afterText, newText)
	}
	if afterRow != beforeRow {
		t.Fatalf("row changed unexpectedly: before=%+v after=%+v", beforeRow, afterRow)
	}
}
