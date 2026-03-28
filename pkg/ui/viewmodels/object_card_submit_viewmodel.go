package viewmodels

import "obj_catalog_fyne_v3/pkg/contracts"

// ObjectCardSubmitInput описує залежності submit-операції діалогу картки об'єкта.
type ObjectCardSubmitInput struct {
	BuildCard    func() (contracts.AdminObjectCard, error)
	Persistence  ObjectCardPersistence
	EditObjN     *int64
	OnSaveResult func(result ObjectCardSaveResult)
}

// ObjectCardSubmitOutput містить дані для реакції View після submit.
type ObjectCardSubmitOutput struct {
	Result          ObjectCardSaveResult
	StatusMessage   string
	ShowErrorDialog bool
}

// ObjectCardSubmitViewModel інкапсулює submit-сценарій create/update для картки об'єкта.
type ObjectCardSubmitViewModel struct {
	dialogVM *ObjectCardDialogViewModel
}

func NewObjectCardSubmitViewModel(dialogVM *ObjectCardDialogViewModel) *ObjectCardSubmitViewModel {
	return &ObjectCardSubmitViewModel{dialogVM: dialogVM}
}

func (vm *ObjectCardSubmitViewModel) Submit(input ObjectCardSubmitInput) (ObjectCardSubmitOutput, error) {
	if input.BuildCard == nil {
		return ObjectCardSubmitOutput{StatusMessage: "Не вдалося підготувати картку"}, nil
	}
	if input.Persistence == nil {
		return ObjectCardSubmitOutput{StatusMessage: "Не вдалося зберегти картку"}, nil
	}
	if vm.dialogVM == nil {
		return ObjectCardSubmitOutput{StatusMessage: "Не вдалося зберегти картку"}, nil
	}

	card, err := input.BuildCard()
	if err != nil {
		return ObjectCardSubmitOutput{
			StatusMessage: err.Error(),
		}, err
	}

	result, err := vm.dialogVM.SaveObject(input.Persistence, input.EditObjN, card)
	out := ObjectCardSubmitOutput{
		Result:        result,
		StatusMessage: result.StatusMessage,
	}
	if err != nil {
		out.ShowErrorDialog = true
		return out, err
	}
	if input.OnSaveResult != nil {
		input.OnSaveResult(result)
	}
	return out, nil
}
