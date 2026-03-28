package viewmodels

import (
	"errors"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectCardSubmitViewModel_Submit_BuildCardError(t *testing.T) {
	dialogVM := NewObjectCardDialogViewModel()
	submitVM := NewObjectCardSubmitViewModel(dialogVM)
	expectedErr := errors.New("invalid card")

	out, err := submitVM.Submit(ObjectCardSubmitInput{
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{}, expectedErr
		},
		Persistence: &objectCardPersistenceStub{},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected build error, got: %v", err)
	}
	if out.StatusMessage != "invalid card" {
		t.Fatalf("unexpected status: %q", out.StatusMessage)
	}
	if out.ShowErrorDialog {
		t.Fatalf("must not show error dialog for build error")
	}
}

func TestObjectCardSubmitViewModel_Submit_SaveError(t *testing.T) {
	dialogVM := NewObjectCardDialogViewModel()
	submitVM := NewObjectCardSubmitViewModel(dialogVM)
	expectedErr := errors.New("create failed")

	out, err := submitVM.Submit(ObjectCardSubmitInput{
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: 1201}, nil
		},
		Persistence: &objectCardPersistenceStub{
			createErr: expectedErr,
		},
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected save error, got: %v", err)
	}
	if !out.ShowErrorDialog {
		t.Fatalf("must show error dialog for save error")
	}
	if out.StatusMessage != "Не вдалося створити об'єкт" {
		t.Fatalf("unexpected status: %q", out.StatusMessage)
	}
}

func TestObjectCardSubmitViewModel_Submit_SuccessCreate(t *testing.T) {
	dialogVM := NewObjectCardDialogViewModel()
	submitVM := NewObjectCardSubmitViewModel(dialogVM)
	stub := &objectCardPersistenceStub{}

	var callbackObjN int64
	out, err := submitVM.Submit(ObjectCardSubmitInput{
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: 1202}, nil
		},
		Persistence: stub,
		OnSaveResult: func(result ObjectCardSaveResult) {
			callbackObjN = result.ObjN
		},
	})
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if !stub.createCalled {
		t.Fatalf("expected create call")
	}
	if callbackObjN != 1202 {
		t.Fatalf("unexpected callback objn: %d", callbackObjN)
	}
	if out.StatusMessage != "Новий об'єкт створено" {
		t.Fatalf("unexpected status: %q", out.StatusMessage)
	}
	if out.ShowErrorDialog {
		t.Fatalf("must not show error dialog on success")
	}
}

func TestObjectCardSubmitViewModel_Submit_SuccessUpdate(t *testing.T) {
	dialogVM := NewObjectCardDialogViewModel()
	submitVM := NewObjectCardSubmitViewModel(dialogVM)
	objn := int64(1303)
	stub := &objectCardPersistenceStub{
		getCardResult: contracts.AdminObjectCard{ObjUIN: 91},
	}

	out, err := submitVM.Submit(ObjectCardSubmitInput{
		BuildCard: func() (contracts.AdminObjectCard, error) {
			return contracts.AdminObjectCard{ObjN: objn}, nil
		},
		Persistence: stub,
		EditObjN:    &objn,
	})
	if err != nil {
		t.Fatalf("unexpected submit error: %v", err)
	}
	if !stub.updateCalled {
		t.Fatalf("expected update call")
	}
	if stub.lastCard.ObjUIN != 91 {
		t.Fatalf("expected objuin propagation")
	}
	if out.StatusMessage != "Картку об'єкта оновлено" {
		t.Fatalf("unexpected status: %q", out.StatusMessage)
	}
}
