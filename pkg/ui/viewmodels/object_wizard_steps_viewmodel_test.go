package viewmodels

import "testing"

func TestObjectWizardStepsViewModel_InitialState(t *testing.T) {
	vm := NewObjectWizardStepsViewModel([]string{"one", "two", "three"})

	if vm.CurrentStep() != 0 {
		t.Fatalf("unexpected initial step: %d", vm.CurrentStep())
	}
	if vm.TotalSteps() != 3 {
		t.Fatalf("unexpected total steps: %d", vm.TotalSteps())
	}
	if vm.CanGoBack() {
		t.Fatalf("must not allow back on first step")
	}
	if !vm.CanGoNext() {
		t.Fatalf("must allow next on first step")
	}
	if vm.CanCreate() {
		t.Fatalf("must not allow create on first step")
	}
}

func TestObjectWizardStepsViewModel_Navigation(t *testing.T) {
	vm := NewObjectWizardStepsViewModel([]string{"one", "two", "three"})

	if !vm.GoNext() || vm.CurrentStep() != 1 {
		t.Fatalf("expected step 1 after first next")
	}
	if !vm.GoNext() || vm.CurrentStep() != 2 {
		t.Fatalf("expected step 2 after second next")
	}
	if vm.GoNext() {
		t.Fatalf("must not go next on last step")
	}
	if !vm.CanCreate() {
		t.Fatalf("must allow create on last step")
	}
	if !vm.GoBack() || vm.CurrentStep() != 1 {
		t.Fatalf("expected step 1 after back")
	}
}

func TestObjectWizardStepsViewModel_StatusText(t *testing.T) {
	vm := NewObjectWizardStepsViewModel([]string{"дані", "параметри"})
	if vm.StatusText() != "Крок 1/2: дані" {
		t.Fatalf("unexpected initial status: %q", vm.StatusText())
	}
	vm.GoNext()
	if vm.StatusText() != "Крок 2/2: параметри" {
		t.Fatalf("unexpected second status: %q", vm.StatusText())
	}
}
