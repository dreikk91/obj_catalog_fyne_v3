package viewmodels

import (
	"errors"
	"testing"
)

func TestObjectWizardInitViewModel_Initialize_Success(t *testing.T) {
	vm := NewObjectWizardInitViewModel()
	loadRefsCalled := false
	fillDefaultsCalled := false

	result := vm.Initialize(ObjectWizardInitInput{
		LoadReferenceData: func() error {
			loadRefsCalled = true
			return nil
		},
		FillDefaults: func() {
			fillDefaultsCalled = true
		},
	})

	if !loadRefsCalled {
		t.Fatalf("expected reference loading")
	}
	if !fillDefaultsCalled {
		t.Fatalf("expected defaults initialization")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(result.Issues))
	}
}

func TestObjectWizardInitViewModel_Initialize_LoadReferencesError(t *testing.T) {
	vm := NewObjectWizardInitViewModel()
	expectedErr := errors.New("refs unavailable")
	fillDefaultsCalled := false

	result := vm.Initialize(ObjectWizardInitInput{
		LoadReferenceData: func() error {
			return expectedErr
		},
		FillDefaults: func() {
			fillDefaultsCalled = true
		},
	})

	if !fillDefaultsCalled {
		t.Fatalf("defaults must be initialized even when references loading fails")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected one issue, got %d", len(result.Issues))
	}
	if !errors.Is(result.Issues[0].Err, expectedErr) {
		t.Fatalf("unexpected issue error: %v", result.Issues[0].Err)
	}
	if result.Issues[0].StatusMessage != "Не вдалося завантажити довідники" {
		t.Fatalf("unexpected status message: %q", result.Issues[0].StatusMessage)
	}
	if !result.Issues[0].ShowErrorDialog {
		t.Fatalf("expected error dialog flag")
	}
}
