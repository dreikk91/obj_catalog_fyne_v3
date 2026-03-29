package viewmodels

import "obj_catalog_fyne_v3/pkg/contracts"

// ObjectWizardPersonalsState описує мінімальний стан/операції В/О для flow-кроку майстра.
type ObjectWizardPersonalsState interface {
	PersonalCount() int
	SelectedPersonal() int
	SetSelectedPersonal(idx int) bool
	PersonalAt(idx int) (contracts.AdminObjectPersonal, bool)
	AddPersonal(item contracts.AdminObjectPersonal) int
	UpdatePersonal(idx int, item contracts.AdminObjectPersonal) bool
	DeletePersonal(idx int) bool
	PersonalFullName(item contracts.AdminObjectPersonal) string
	NextPersonalNumber() int64
}

// ObjectWizardPersonalsActionResult описує результат команди flow для UI.
type ObjectWizardPersonalsActionResult struct {
	StatusText   string
	RefreshTable bool
}

// ObjectWizardPersonalsEditPrompt містить дані для запуску редагування В/О.
type ObjectWizardPersonalsEditPrompt struct {
	CanEdit      bool
	SelectedIdx  int
	Initial      contracts.AdminObjectPersonal
	StatusText   string
	RefreshTable bool
}

// ObjectWizardPersonalsDeletePrompt містить дані підтвердження видалення В/О.
type ObjectWizardPersonalsDeletePrompt struct {
	CanDelete    bool
	SelectedIdx  int
	ConfirmText  string
	StatusText   string
	RefreshTable bool
}

// ObjectWizardPersonalsFlowViewModel інкапсулює сценарії кроку "В/О".
type ObjectWizardPersonalsFlowViewModel struct {
	tableVM *ObjectWizardPersonalsTableViewModel
}

func NewObjectWizardPersonalsFlowViewModel(tableVM *ObjectWizardPersonalsTableViewModel) *ObjectWizardPersonalsFlowViewModel {
	if tableVM == nil {
		tableVM = NewObjectWizardPersonalsTableViewModel()
	}
	return &ObjectWizardPersonalsFlowViewModel{tableVM: tableVM}
}

func (vm *ObjectWizardPersonalsFlowViewModel) NextNumber(state ObjectWizardPersonalsState) int64 {
	return state.NextPersonalNumber()
}

func (vm *ObjectWizardPersonalsFlowViewModel) SelectTableRow(state ObjectWizardPersonalsState, row int) {
	if row <= 0 {
		state.SetSelectedPersonal(-1)
		return
	}
	idx := row - 1
	state.SetSelectedPersonal(idx)
}

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyAdd(state ObjectWizardPersonalsState, item contracts.AdminObjectPersonal) ObjectWizardPersonalsActionResult {
	state.AddPersonal(item)
	return ObjectWizardPersonalsActionResult{
		StatusText:   vm.tableVM.StatusAdded(state.PersonalCount()),
		RefreshTable: true,
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) PrepareEdit(state ObjectWizardPersonalsState) ObjectWizardPersonalsEditPrompt {
	selectedIdx := state.SelectedPersonal()
	initial, ok := state.PersonalAt(selectedIdx)
	if !ok {
		return ObjectWizardPersonalsEditPrompt{
			CanEdit:    false,
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsEditPrompt{
		CanEdit:     true,
		SelectedIdx: selectedIdx,
		Initial:     initial,
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyUpdate(state ObjectWizardPersonalsState, idx int, item contracts.AdminObjectPersonal) ObjectWizardPersonalsActionResult {
	if !state.UpdatePersonal(idx, item) {
		return ObjectWizardPersonalsActionResult{
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsActionResult{
		StatusText:   vm.tableVM.StatusUpdated(),
		RefreshTable: true,
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) PrepareDelete(state ObjectWizardPersonalsState) ObjectWizardPersonalsDeletePrompt {
	selectedIdx := state.SelectedPersonal()
	target, ok := state.PersonalAt(selectedIdx)
	if !ok {
		return ObjectWizardPersonalsDeletePrompt{
			CanDelete:  false,
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsDeletePrompt{
		CanDelete:   true,
		SelectedIdx: selectedIdx,
		ConfirmText: vm.tableVM.DeleteConfirmText(state.PersonalFullName(target)),
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyDelete(state ObjectWizardPersonalsState, idx int) ObjectWizardPersonalsActionResult {
	if !state.DeletePersonal(idx) {
		return ObjectWizardPersonalsActionResult{
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsActionResult{
		StatusText:   vm.tableVM.StatusDeleted(state.PersonalCount()),
		RefreshTable: true,
	}
}
