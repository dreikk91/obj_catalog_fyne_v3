package viewmodels

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectWizardSubmitInput описує всі залежності та дані для submit-операції майстра.
type ObjectWizardSubmitInput struct {
	ValidateStep func(step int) error
	BuildCard    func() (contracts.AdminObjectCard, error)
	Persistence  ObjectWizardPersistence
	Personals    []contracts.AdminObjectPersonal
	Zones        []contracts.AdminObjectZone
	Coordinates  contracts.AdminObjectCoordinates
}

// ObjectWizardSubmitOutput містить результат submit-операції для UI шару.
type ObjectWizardSubmitOutput struct {
	Result          ObjectWizardCreateResult
	StatusMessage   string
	ShowErrorDialog bool
	WarningMessage  string
}

// ObjectWizardSubmitViewModel інкапсулює submit-флоу майстра створення об'єкта.
type ObjectWizardSubmitViewModel struct {
	flowVM   *ObjectWizardFlowViewModel
	wizardVM *ObjectWizardViewModel
}

func NewObjectWizardSubmitViewModel(flowVM *ObjectWizardFlowViewModel, wizardVM *ObjectWizardViewModel) *ObjectWizardSubmitViewModel {
	return &ObjectWizardSubmitViewModel{
		flowVM:   flowVM,
		wizardVM: wizardVM,
	}
}

func (vm *ObjectWizardSubmitViewModel) Submit(input ObjectWizardSubmitInput) (ObjectWizardSubmitOutput, error) {
	if vm.flowVM == nil {
		return ObjectWizardSubmitOutput{}, fmt.Errorf("flow view model не ініціалізовано")
	}
	if vm.wizardVM == nil {
		return ObjectWizardSubmitOutput{}, fmt.Errorf("wizard view model не ініціалізовано")
	}
	if input.Persistence == nil {
		return ObjectWizardSubmitOutput{}, fmt.Errorf("persistence не ініціалізовано")
	}
	if input.BuildCard == nil {
		return ObjectWizardSubmitOutput{}, fmt.Errorf("buildCard не ініціалізовано")
	}

	if err := vm.flowVM.ValidateCreate(input.ValidateStep); err != nil {
		return ObjectWizardSubmitOutput{
			StatusMessage: err.Error(),
		}, err
	}

	card, err := input.BuildCard()
	if err != nil {
		return ObjectWizardSubmitOutput{
			StatusMessage: err.Error(),
		}, err
	}

	result, err := vm.wizardVM.CreateObjectWithRelatedData(
		input.Persistence,
		card,
		input.Personals,
		input.Zones,
		input.Coordinates,
	)

	out := ObjectWizardSubmitOutput{
		Result:        result,
		StatusMessage: result.StatusMessage,
	}
	if len(result.Warnings) > 0 {
		out.WarningMessage = "Об'єкт створено, але частина додаткових даних не збережена:\n\n" + strings.Join(result.Warnings, "\n")
	}
	if err != nil {
		out.ShowErrorDialog = true
		return out, err
	}
	return out, nil
}
