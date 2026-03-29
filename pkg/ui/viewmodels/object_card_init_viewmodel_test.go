package viewmodels

import (
	"errors"
	"testing"
)

func TestObjectCardInitViewModel_Initialize_CreateMode(t *testing.T) {
	vm := NewObjectCardInitViewModel()
	loadRefsCalled := false
	fillDefaultsCalled := false
	prepareEditCalled := false
	loadCardCalled := false

	result := vm.Initialize(ObjectCardInitInput{
		LoadReferenceData: func() error {
			loadRefsCalled = true
			return nil
		},
		PrepareEditMode: func() {
			prepareEditCalled = true
		},
		LoadCard: func(objn int64) error {
			loadCardCalled = true
			return nil
		},
		FillDefaults: func() {
			fillDefaultsCalled = true
		},
	})

	if !loadRefsCalled {
		t.Fatalf("expected references loading in create mode")
	}
	if !fillDefaultsCalled {
		t.Fatalf("expected defaults fill in create mode")
	}
	if prepareEditCalled {
		t.Fatalf("must not prepare edit mode in create mode")
	}
	if loadCardCalled {
		t.Fatalf("must not load card in create mode")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(result.Issues))
	}
}

func TestObjectCardInitViewModel_Initialize_EditMode(t *testing.T) {
	vm := NewObjectCardInitViewModel()
	objn := int64(2201)
	prepareEditCalled := false
	loadCardCalled := false
	fillDefaultsCalled := false
	var loadedObjN int64

	result := vm.Initialize(ObjectCardInitInput{
		EditObjN: &objn,
		LoadReferenceData: func() error {
			return nil
		},
		PrepareEditMode: func() {
			prepareEditCalled = true
		},
		LoadCard: func(targetObjN int64) error {
			loadCardCalled = true
			loadedObjN = targetObjN
			return nil
		},
		FillDefaults: func() {
			fillDefaultsCalled = true
		},
	})

	if !prepareEditCalled {
		t.Fatalf("expected edit mode preparation")
	}
	if !loadCardCalled {
		t.Fatalf("expected card loading in edit mode")
	}
	if loadedObjN != objn {
		t.Fatalf("unexpected objn passed to LoadCard: %d", loadedObjN)
	}
	if fillDefaultsCalled {
		t.Fatalf("must not fill defaults in edit mode")
	}
	if len(result.Issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(result.Issues))
	}
}

func TestObjectCardInitViewModel_Initialize_ReferenceErrorStillContinues(t *testing.T) {
	vm := NewObjectCardInitViewModel()
	loadCardCalled := false
	fillDefaultsCalled := false
	refErr := errors.New("refs down")

	result := vm.Initialize(ObjectCardInitInput{
		LoadReferenceData: func() error {
			return refErr
		},
		LoadCard: func(targetObjN int64) error {
			loadCardCalled = true
			return nil
		},
		FillDefaults: func() {
			fillDefaultsCalled = true
		},
	})

	if !fillDefaultsCalled {
		t.Fatalf("expected defaults fill even after refs error in create mode")
	}
	if loadCardCalled {
		t.Fatalf("must not load card in create mode")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected one issue, got %d", len(result.Issues))
	}
	if !errors.Is(result.Issues[0].Err, refErr) {
		t.Fatalf("unexpected refs issue error: %v", result.Issues[0].Err)
	}
	if result.Issues[0].StatusMessage != "Не вдалося завантажити довідники" {
		t.Fatalf("unexpected refs issue status: %q", result.Issues[0].StatusMessage)
	}
}

func TestObjectCardInitViewModel_Initialize_EditLoadError(t *testing.T) {
	vm := NewObjectCardInitViewModel()
	objn := int64(3301)
	loadCardErr := errors.New("load card failed")

	result := vm.Initialize(ObjectCardInitInput{
		EditObjN: &objn,
		LoadReferenceData: func() error {
			return nil
		},
		PrepareEditMode: func() {},
		LoadCard: func(targetObjN int64) error {
			if targetObjN != objn {
				t.Fatalf("unexpected objn: %d", targetObjN)
			}
			return loadCardErr
		},
		FillDefaults: func() {
			t.Fatalf("must not fill defaults in edit mode")
		},
	})

	if len(result.Issues) != 1 {
		t.Fatalf("expected one issue, got %d", len(result.Issues))
	}
	if !errors.Is(result.Issues[0].Err, loadCardErr) {
		t.Fatalf("unexpected load card issue error: %v", result.Issues[0].Err)
	}
	if result.Issues[0].StatusMessage != "Не вдалося завантажити об'єкт для редагування" {
		t.Fatalf("unexpected issue status: %q", result.Issues[0].StatusMessage)
	}
	if !result.Issues[0].ShowErrorDialog {
		t.Fatalf("expected show error dialog flag")
	}
}
