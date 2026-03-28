package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardSubmitViewModel_Submit_Success(t *testing.T) {
	flowVM := NewObjectWizardFlowViewModel([]string{"one"})
	wizardVM := NewObjectWizardViewModel()
	submitVM := NewObjectWizardSubmitViewModel(flowVM, wizardVM)
	stub := &objectWizardPersistenceStub{}

	out, err := submitVM.Submit(ObjectWizardSubmitInput{
		ValidateStep: func(step int) error { return nil },
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: 1501}, nil
		},
		Persistence: stub,
		Personals:   []contracts.AdminObjectPersonal{{Number: 1}},
		Zones:       []contracts.AdminObjectZone{{ZoneNumber: 1}},
		Coordinates: contracts.AdminObjectCoordinates{Latitude: "49.1", Longitude: "24.1"},
	})
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if out.StatusMessage != "Новий об'єкт створено" {
		t.Fatalf("unexpected status: %q", out.StatusMessage)
	}
	if out.ShowErrorDialog {
		t.Fatalf("must not show error dialog on success")
	}
	if out.WarningMessage != "" {
		t.Fatalf("unexpected warning message: %q", out.WarningMessage)
	}
	if out.Result.ObjN != 1501 {
		t.Fatalf("unexpected objn in result: %d", out.Result.ObjN)
	}
}

func TestObjectWizardSubmitViewModel_Submit_ValidateError(t *testing.T) {
	flowVM := NewObjectWizardFlowViewModel([]string{"one"})
	wizardVM := NewObjectWizardViewModel()
	submitVM := NewObjectWizardSubmitViewModel(flowVM, wizardVM)
	stub := &objectWizardPersistenceStub{}
	validateErr := errors.New("validation failed")
	buildCalled := false

	out, err := submitVM.Submit(ObjectWizardSubmitInput{
		ValidateStep: func(step int) error { return validateErr },
		BuildCard: func() (contracts.AdminObjectCard, error) {
			buildCalled = true
			return contracts.AdminObjectCard{ObjN: 1502}, nil
		},
		Persistence: stub,
	})
	if !errors.Is(err, validateErr) {
		t.Fatalf("expected validate error, got: %v", err)
	}
	if buildCalled {
		t.Fatalf("build card must not be called when validation fails")
	}
	if out.ShowErrorDialog {
		t.Fatalf("must not show error dialog for validation error")
	}
	if out.StatusMessage != "validation failed" {
		t.Fatalf("unexpected status for validation error: %q", out.StatusMessage)
	}
}

func TestObjectWizardSubmitViewModel_Submit_BuildCardError(t *testing.T) {
	flowVM := NewObjectWizardFlowViewModel([]string{"one"})
	wizardVM := NewObjectWizardViewModel()
	submitVM := NewObjectWizardSubmitViewModel(flowVM, wizardVM)
	stub := &objectWizardPersistenceStub{}
	buildErr := errors.New("build failed")

	out, err := submitVM.Submit(ObjectWizardSubmitInput{
		ValidateStep: func(step int) error { return nil },
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{}, buildErr
		},
		Persistence: stub,
	})
	if !errors.Is(err, buildErr) {
		t.Fatalf("expected build error, got: %v", err)
	}
	if out.ShowErrorDialog {
		t.Fatalf("must not show error dialog for build error")
	}
	if out.StatusMessage != "build failed" {
		t.Fatalf("unexpected status for build error: %q", out.StatusMessage)
	}
}

func TestObjectWizardSubmitViewModel_Submit_CreateError(t *testing.T) {
	flowVM := NewObjectWizardFlowViewModel([]string{"one"})
	wizardVM := NewObjectWizardViewModel()
	submitVM := NewObjectWizardSubmitViewModel(flowVM, wizardVM)
	createErr := errors.New("create failed")
	stub := &objectWizardPersistenceStub{
		createErr: createErr,
	}

	out, err := submitVM.Submit(ObjectWizardSubmitInput{
		ValidateStep: func(step int) error { return nil },
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: 1503}, nil
		},
		Persistence: stub,
	})
	if !errors.Is(err, createErr) {
		t.Fatalf("expected create error, got: %v", err)
	}
	if !out.ShowErrorDialog {
		t.Fatalf("must show error dialog for create error")
	}
	if out.StatusMessage != "Не вдалося створити об'єкт" {
		t.Fatalf("unexpected status for create error: %q", out.StatusMessage)
	}
}

func TestObjectWizardSubmitViewModel_Submit_Warnings(t *testing.T) {
	flowVM := NewObjectWizardFlowViewModel([]string{"one"})
	wizardVM := NewObjectWizardViewModel()
	submitVM := NewObjectWizardSubmitViewModel(flowVM, wizardVM)
	stub := &objectWizardPersistenceStub{
		addPersErrs: []error{errors.New("p1")},
	}

	out, err := submitVM.Submit(ObjectWizardSubmitInput{
		ValidateStep: func(step int) error { return nil },
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: 1504}, nil
		},
		Persistence: stub,
		Personals:   []contracts.AdminObjectPersonal{{Number: 1}},
	})
	if err != nil {
		t.Fatalf("unexpected error with warnings: %v", err)
	}
	if out.WarningMessage == "" {
		t.Fatalf("expected warning message to be prepared")
	}
}
