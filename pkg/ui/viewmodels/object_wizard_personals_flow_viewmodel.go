package viewmodels

// ObjectWizardPersonalsState описує мінімальний стан/операції В/О для flow-кроку майстра.
type ObjectWizardPersonalsState interface {
	Count() int
	Selected() int
	SetSelected(idx int) bool
	At(idx int) (ObjectPersonal, bool)
	Add(item ObjectPersonal) int
	Update(idx int, item ObjectPersonal) bool
	Delete(idx int) bool
	FullName(item ObjectPersonal) string
	NextNumber() int64
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
	Initial      ObjectPersonal
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
	return state.NextNumber()
}

func (vm *ObjectWizardPersonalsFlowViewModel) SelectTableRow(state ObjectWizardPersonalsState, row int) {
	if row <= 0 {
		state.SetSelected(-1)
		return
	}
	idx := row - 1
	state.SetSelected(idx)
}

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyAdd(state ObjectWizardPersonalsState, item ObjectPersonal) ObjectWizardPersonalsActionResult {
	state.Add(item)
	return ObjectWizardPersonalsActionResult{
		StatusText:   vm.tableVM.StatusAdded(state.Count()),
		RefreshTable: true,
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) PrepareEdit(state ObjectWizardPersonalsState) ObjectWizardPersonalsEditPrompt {
	selectedIdx := state.Selected()
	initial, ok := state.At(selectedIdx)
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

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyUpdate(state ObjectWizardPersonalsState, idx int, item ObjectPersonal) ObjectWizardPersonalsActionResult {
	if !state.Update(idx, item) {
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
	selectedIdx := state.Selected()
	target, ok := state.At(selectedIdx)
	if !ok {
		return ObjectWizardPersonalsDeletePrompt{
			CanDelete:  false,
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsDeletePrompt{
		CanDelete:   true,
		SelectedIdx: selectedIdx,
		ConfirmText: vm.tableVM.DeleteConfirmText(state.FullName(target)),
	}
}

func (vm *ObjectWizardPersonalsFlowViewModel) ApplyDelete(state ObjectWizardPersonalsState, idx int) ObjectWizardPersonalsActionResult {
	if !state.Delete(idx) {
		return ObjectWizardPersonalsActionResult{
			StatusText: vm.tableVM.StatusSelectionRequired(),
		}
	}
	return ObjectWizardPersonalsActionResult{
		StatusText:   vm.tableVM.StatusDeleted(state.Count()),
		RefreshTable: true,
	}
}
