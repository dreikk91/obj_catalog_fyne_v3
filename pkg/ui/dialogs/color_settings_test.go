package dialogs

import (
	"image/color"
	"testing"

	"obj_catalog_fyne_v3/pkg/utils"
)

func TestEventColorOptions_GroupDisarmCodesTogether(t *testing.T) {
	options := eventColorOptions()

	var disarm eventColorOption
	found := false
	for _, option := range options {
		if option.Label == "Зняття з охорони" {
			disarm = option
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected disarm group in color settings")
	}

	want := []int{11, 14, 18}
	if len(disarm.Codes) != len(want) {
		t.Fatalf("unexpected disarm group size: got=%v want=%v", disarm.Codes, want)
	}
	for idx, code := range want {
		if disarm.Codes[idx] != code {
			t.Fatalf("unexpected disarm codes: got=%v want=%v", disarm.Codes, want)
		}
	}
}

func TestApplyEventRowColor_UpdatesAllCodesInGroup(t *testing.T) {
	utils.ResetEventColorsToDefault(false)
	t.Cleanup(func() {
		utils.ResetEventColorsToDefault(false)
	})

	newRow := color.NRGBA{R: 12, G: 34, B: 56, A: 255}
	applyEventRowColor([]int{11, 14, 18}, false, newRow)

	for _, code := range []int{11, 14, 18} {
		if got := utils.GetEventRowColor(code, false); got != newRow {
			t.Fatalf("row color for SC1=%d = %+v, want %+v", code, got, newRow)
		}
	}

	if got := utils.GetEventRowColor(1, false); got == newRow {
		t.Fatalf("alarm group color changed unexpectedly: %+v", got)
	}
}

func TestApplyEventTextColor_UpdatesAllCodesInGroup(t *testing.T) {
	utils.ResetEventColorsToDefault(true)
	t.Cleanup(func() {
		utils.ResetEventColorsToDefault(true)
	})

	newText := color.NRGBA{R: 101, G: 102, B: 103, A: 255}
	applyEventTextColor([]int{1, 21, 22, 23, 24, 25}, true, newText)

	for _, code := range []int{1, 21, 22, 23, 24, 25} {
		if got := utils.GetEventTextColor(code, true); got != newText {
			t.Fatalf("text color for SC1=%d = %+v, want %+v", code, got, newText)
		}
	}

	if got := utils.GetEventTextColor(11, true); got == newText {
		t.Fatalf("disarm group text changed unexpectedly: %+v", got)
	}
}
