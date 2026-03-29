package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectCardPersistenceStub struct {
	getCardResult contracts.AdminObjectCard
	getCardErr    error
	createErr     error
	updateErr     error

	createCalled bool
	updateCalled bool
	lastCard     contracts.AdminObjectCard
}

func (s *objectCardPersistenceStub) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	return s.getCardResult, s.getCardErr
}

func (s *objectCardPersistenceStub) CreateObject(card contracts.AdminObjectCard) error {
	s.createCalled = true
	s.lastCard = card
	return s.createErr
}

func (s *objectCardPersistenceStub) UpdateObject(card contracts.AdminObjectCard) error {
	s.updateCalled = true
	s.lastCard = card
	return s.updateErr
}

func TestObjectCardDialogViewModel_SaveObject_Create(t *testing.T) {
	vm := NewObjectCardDialogViewModel()
	stub := &objectCardPersistenceStub{}

	result, err := vm.SaveObject(stub, nil, contracts.AdminObjectCard{ObjN: 1001})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.createCalled {
		t.Fatalf("expected create call")
	}
	if result.ObjN != 1001 {
		t.Fatalf("unexpected objn: %d", result.ObjN)
	}
	if result.StatusMessage != "Новий об'єкт створено" {
		t.Fatalf("unexpected status: %q", result.StatusMessage)
	}
}

func TestObjectCardDialogViewModel_SaveObject_Update(t *testing.T) {
	vm := NewObjectCardDialogViewModel()
	stub := &objectCardPersistenceStub{
		getCardResult: contracts.AdminObjectCard{ObjUIN: 777},
	}
	objn := int64(2002)

	result, err := vm.SaveObject(stub, &objn, contracts.AdminObjectCard{ObjN: objn})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !stub.updateCalled {
		t.Fatalf("expected update call")
	}
	if stub.lastCard.ObjUIN != 777 {
		t.Fatalf("expected objuin to be propagated, got %d", stub.lastCard.ObjUIN)
	}
	if result.StatusMessage != "Картку об'єкта оновлено" {
		t.Fatalf("unexpected status: %q", result.StatusMessage)
	}
}

func TestObjectCardDialogViewModel_SaveObject_UpdateReloadError(t *testing.T) {
	vm := NewObjectCardDialogViewModel()
	stub := &objectCardPersistenceStub{getCardErr: errors.New("db down")}
	objn := int64(3003)

	result, err := vm.SaveObject(stub, &objn, contracts.AdminObjectCard{ObjN: objn})
	if err == nil {
		t.Fatalf("expected error")
	}
	if result.StatusMessage != "Не вдалося перезавантажити картку" {
		t.Fatalf("unexpected status: %q", result.StatusMessage)
	}
}
