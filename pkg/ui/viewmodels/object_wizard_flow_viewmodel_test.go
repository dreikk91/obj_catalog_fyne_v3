package viewmodels

import (
	"errors"
	"testing"
)

func TestObjectWizardFlowViewModel_InitialState(t *testing.T) {
	vm := NewObjectWizardFlowViewModel([]string{"one", "two"})

	if vm.CurrentStep() != 0 {
		t.Fatalf("unexpected current step: %d", vm.CurrentStep())
	}
	if vm.TotalSteps() != 2 {
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

func TestObjectWizardFlowViewModel_GoNext_WithValidation(t *testing.T) {
	vm := NewObjectWizardFlowViewModel([]string{"one", "two"})
	calledWith := -1

	moved, err := vm.GoNext(func(step int) error {
		calledWith = step
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if !moved {
		t.Fatalf("expected move to next step")
	}
	if calledWith != 0 {
		t.Fatalf("validator called with unexpected step: %d", calledWith)
	}
	if vm.CurrentStep() != 1 {
		t.Fatalf("unexpected current step after next: %d", vm.CurrentStep())
	}
}

func TestObjectWizardFlowViewModel_GoNext_ValidationError(t *testing.T) {
	vm := NewObjectWizardFlowViewModel([]string{"one", "two"})
	expectedErr := errors.New("boom")

	moved, err := vm.GoNext(func(step int) error {
		if step != 0 {
			t.Fatalf("unexpected step: %d", step)
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected validation error, got: %v", err)
	}
	if moved {
		t.Fatalf("must not move on validation error")
	}
	if vm.CurrentStep() != 0 {
		t.Fatalf("step must remain unchanged on validation error")
	}
}

func TestObjectWizardFlowViewModel_ValidateCreate(t *testing.T) {
	vm := NewObjectWizardFlowViewModel([]string{"one", "two", "three"})
	calledWith := -1

	if err := vm.ValidateCreate(func(step int) error {
		calledWith = step
		return nil
	}); err != nil {
		t.Fatalf("unexpected create validation error: %v", err)
	}
	if calledWith != 2 {
		t.Fatalf("validator must be called for last step, got: %d", calledWith)
	}
}
