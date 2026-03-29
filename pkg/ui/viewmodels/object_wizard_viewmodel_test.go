package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectWizardPersistenceStub struct {
	createErr    error
	addPersErrs  []error
	addZoneErrs  []error
	coordsErr    error
	createCalled bool
}

func (s *objectWizardPersistenceStub) CreateObject(card contracts.AdminObjectCard) error {
	s.createCalled = true
	return s.createErr
}

func (s *objectWizardPersistenceStub) AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	if len(s.addPersErrs) == 0 {
		return nil
	}
	err := s.addPersErrs[0]
	s.addPersErrs = s.addPersErrs[1:]
	return err
}

func (s *objectWizardPersistenceStub) AddObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	if len(s.addZoneErrs) == 0 {
		return nil
	}
	err := s.addZoneErrs[0]
	s.addZoneErrs = s.addZoneErrs[1:]
	return err
}

func (s *objectWizardPersistenceStub) SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error {
	return s.coordsErr
}

func TestObjectWizardViewModel_CreateObjectWithRelatedData_Success(t *testing.T) {
	vm := NewObjectWizardViewModel()
	stub := &objectWizardPersistenceStub{}

	result, err := vm.CreateObjectWithRelatedData(
		stub,
		contracts.AdminObjectCard{ObjN: 111},
		[]contracts.AdminObjectPersonal{{Number: 1}},
		[]contracts.AdminObjectZone{{ZoneNumber: 1}},
		contracts.AdminObjectCoordinates{Latitude: "49.1", Longitude: "24.1"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.createCalled {
		t.Fatalf("expected create call")
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("expected no warnings, got %d", len(result.Warnings))
	}
	if result.StatusMessage != "Новий об'єкт створено" {
		t.Fatalf("unexpected status: %q", result.StatusMessage)
	}
}

func TestObjectWizardViewModel_CreateObjectWithRelatedData_CreateFails(t *testing.T) {
	vm := NewObjectWizardViewModel()
	stub := &objectWizardPersistenceStub{
		createErr: errors.New("create failed"),
	}

	result, err := vm.CreateObjectWithRelatedData(
		stub,
		contracts.AdminObjectCard{ObjN: 112},
		nil,
		nil,
		contracts.AdminObjectCoordinates{},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if result.StatusMessage != "Не вдалося створити об'єкт" {
		t.Fatalf("unexpected status: %q", result.StatusMessage)
	}
}

func TestObjectWizardViewModel_CreateObjectWithRelatedData_Warnings(t *testing.T) {
	vm := NewObjectWizardViewModel()
	stub := &objectWizardPersistenceStub{
		addPersErrs: []error{errors.New("p1")},
		addZoneErrs: []error{errors.New("z1")},
		coordsErr:   errors.New("coords"),
	}

	result, err := vm.CreateObjectWithRelatedData(
		stub,
		contracts.AdminObjectCard{ObjN: 113},
		[]contracts.AdminObjectPersonal{{Number: 1}},
		[]contracts.AdminObjectZone{{ZoneNumber: 1}},
		contracts.AdminObjectCoordinates{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d", len(result.Warnings))
	}
}

func TestObjectWizardViewModel_ValidateStep_RequiredFields(t *testing.T) {
	vm := NewObjectWizardViewModel()

	err := vm.ValidateStep(ObjectWizardStepValidationInput{
		Step:              0,
		ObjNRaw:           " ",
		ShortName:         "Obj",
		SelectedObjTypeID: 1,
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestObjectWizardViewModel_ValidateStep_PropagatesCardBuildError(t *testing.T) {
	vm := NewObjectWizardViewModel()
	cardErr := errors.New("card invalid")

	err := vm.ValidateStep(ObjectWizardStepValidationInput{
		Step:              0,
		ObjNRaw:           "1001",
		ShortName:         "Obj",
		SelectedObjTypeID: 1,
		CardBuildErr:      cardErr,
	})
	if err == nil || err.Error() != "card invalid" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestObjectWizardViewModel_ValidateStep_Success(t *testing.T) {
	vm := NewObjectWizardViewModel()

	err := vm.ValidateStep(ObjectWizardStepValidationInput{
		Step:              0,
		ObjNRaw:           "1001",
		ShortName:         "Obj",
		SelectedObjTypeID: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
