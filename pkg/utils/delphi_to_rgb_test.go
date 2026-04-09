package utils

import (
	"image/color"
	"testing"
)

func TestDefaultDisarmPaletteDiffersFromDebugPalette_Light(t *testing.T) {
	ResetEventColorsToDefault(false)

	_, disarm := SelectColorNRGBA(11)
	_, partial := SelectColorNRGBA(14)

	if disarm == partial {
		t.Fatalf("expected disarm palette to differ from debug-like olive palette, got %+v", disarm)
	}
}

func TestDefaultDisarmPaletteDiffersFromDebugPalette_Dark(t *testing.T) {
	ResetEventColorsToDefault(true)

	_, disarm := SelectColorNRGBADark(11)
	_, partial := SelectColorNRGBADark(14)

	if disarm == partial {
		t.Fatalf("expected dark disarm palette to differ from debug-like olive palette, got %+v", disarm)
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
