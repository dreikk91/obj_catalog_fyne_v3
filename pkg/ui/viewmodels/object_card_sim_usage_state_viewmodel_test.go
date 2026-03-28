package viewmodels

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestObjectCardSIMUsageStateViewModel_DefaultState(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewObjectCardSIMUsageStateViewModel()
	sim1, _ := vm.SIM1Binding().Get()
	sim2, _ := vm.SIM2Binding().Get()

	if sim1 != "" {
		t.Fatalf("unexpected default SIM1 text: %q", sim1)
	}
	if sim2 != "" {
		t.Fatalf("unexpected default SIM2 text: %q", sim2)
	}
}

func TestObjectCardSIMUsageStateViewModel_SetAndClear(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	vm := NewObjectCardSIMUsageStateViewModel()
	vm.SetSIM1("used by #1")
	vm.SetSIM2("used by #2")

	sim1, _ := vm.SIM1Binding().Get()
	sim2, _ := vm.SIM2Binding().Get()
	if sim1 != "used by #1" {
		t.Fatalf("unexpected SIM1 text: %q", sim1)
	}
	if sim2 != "used by #2" {
		t.Fatalf("unexpected SIM2 text: %q", sim2)
	}

	vm.Clear()
	sim1, _ = vm.SIM1Binding().Get()
	sim2, _ = vm.SIM2Binding().Get()
	if sim1 != "" || sim2 != "" {
		t.Fatalf("texts must be cleared, got SIM1=%q SIM2=%q", sim1, sim2)
	}
}
