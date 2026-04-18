package dialogs

import "testing"

func TestAdminEventOverrideMapping(t *testing.T) {
	if got := adminEventOverrideLabelFromSC1(i64(2)); got != "Тривога техн." {
		t.Fatalf("adminEventOverrideLabelFromSC1() = %q", got)
	}
	if got := adminEventOverrideSC1FromLabel("Подію заборонено"); got == nil || *got != 12 {
		t.Fatalf("adminEventOverrideSC1FromLabel() = %+v", got)
	}
}

func TestAdminEventSemanticFamiliesAndPalette(t *testing.T) {
	if got := adminEventTypeLabel(i64(29)); got != "офлайн" {
		t.Fatalf("adminEventTypeLabel() = %q", got)
	}
	if !adminEventMatchesFamily(i64(16), "test") {
		t.Fatal("adminEventMatchesFamily() expected test family match")
	}
	if got := adminEventPaletteCode(i64(18)); got != 14 {
		t.Fatalf("adminEventPaletteCode() = %d, want 14", got)
	}
}
