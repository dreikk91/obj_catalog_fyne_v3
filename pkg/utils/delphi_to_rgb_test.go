package utils

import "testing"

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
