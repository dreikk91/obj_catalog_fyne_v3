package viewmodels

import "strings"

// ObjectWizardZonesState описує мінімальний стан/операції зон, потрібні для flow-кроку майстра.
type ObjectWizardZonesState interface {
	SelectedZone() int
	ZoneCount() int
	SelectZoneByNumber(zoneNumber int64) bool
	EnsureFirstZone(defaultDescription string) (int64, error)
	SaveSelectedZoneAndEnsureNext(description string) (currentZone int64, nextZone int64, err error)
	NextZoneNumberForAdd() int64
	EnsureZoneExists(zoneNumber int64, defaultDescription string) error
	EffectiveZoneNumberAt(idx int) int64
	DeleteSelectedZone() (int64, bool)
	FillZones(count int64) error
	ResetZones()
	MaxZoneNumber() int64
}

// ObjectWizardZonesActionResult описує результат команди flow для UI.
type ObjectWizardZonesActionResult struct {
	StatusText       string
	RefreshTable     bool
	TargetZoneNumber int64
	FocusQuickName   bool
	ShowErrorDialog  bool
	Err              error
}

// ObjectWizardZonesDeletePrompt містить дані підтвердження видалення.
type ObjectWizardZonesDeletePrompt struct {
	CanDelete        bool
	TargetZoneNumber int64
	ConfirmText      string
	StatusText       string
}

// ObjectWizardZonesFlowViewModel інкапсулює сценарії дій кроку "Зони".
type ObjectWizardZonesFlowViewModel struct {
	stepVM *ObjectWizardZonesStepViewModel
}

func NewObjectWizardZonesFlowViewModel(stepVM *ObjectWizardZonesStepViewModel) *ObjectWizardZonesFlowViewModel {
	if stepVM == nil {
		stepVM = NewObjectWizardZonesStepViewModel()
	}
	return &ObjectWizardZonesFlowViewModel{stepVM: stepVM}
}

func (vm *ObjectWizardZonesFlowViewModel) MoveToNext(state ObjectWizardZonesState, description string) ObjectWizardZonesActionResult {
	selectedZone := state.SelectedZone()
	if selectedZone < 0 || selectedZone >= state.ZoneCount() {
		if state.ZoneCount() == 0 {
			if _, err := state.EnsureFirstZone(strings.TrimSpace(description)); err != nil {
				return ObjectWizardZonesActionResult{
					StatusText:      vm.stepVM.StatusAddFirstFailed(),
					ShowErrorDialog: true,
					Err:             err,
				}
			}
			return ObjectWizardZonesActionResult{
				StatusText:       vm.stepVM.StatusFirstAdded(),
				RefreshTable:     true,
				TargetZoneNumber: 1,
				FocusQuickName:   true,
			}
		}
		_ = state.SelectZoneByNumber(0)
	}

	selectedZone = state.SelectedZone()
	if selectedZone < 0 || selectedZone >= state.ZoneCount() {
		return ObjectWizardZonesActionResult{
			StatusText: vm.stepVM.StatusSelectionRequired(),
		}
	}

	currentZoneNumber, nextZoneNumber, err := state.SaveSelectedZoneAndEnsureNext(description)
	if err != nil {
		return ObjectWizardZonesActionResult{
			StatusText:      vm.stepVM.StatusAddNextFailed(),
			ShowErrorDialog: true,
			Err:             err,
		}
	}

	return ObjectWizardZonesActionResult{
		StatusText:       vm.stepVM.StatusSavedAndMoved(currentZoneNumber, nextZoneNumber),
		RefreshTable:     true,
		TargetZoneNumber: nextZoneNumber,
		FocusQuickName:   true,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) AddZone(state ObjectWizardZonesState) ObjectWizardZonesActionResult {
	nextZoneNumber := state.NextZoneNumberForAdd()
	if err := state.EnsureZoneExists(nextZoneNumber, ""); err != nil {
		return ObjectWizardZonesActionResult{
			StatusText:      vm.stepVM.StatusAddFailed(),
			ShowErrorDialog: true,
			Err:             err,
		}
	}
	return ObjectWizardZonesActionResult{
		StatusText:       vm.stepVM.StatusReadyForInput(nextZoneNumber),
		RefreshTable:     true,
		TargetZoneNumber: nextZoneNumber,
		FocusQuickName:   true,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) StartEdit(state ObjectWizardZonesState) ObjectWizardZonesActionResult {
	if state.ZoneCount() == 0 {
		if _, err := state.EnsureFirstZone(""); err != nil {
			return ObjectWizardZonesActionResult{
				StatusText:      vm.stepVM.StatusCreateFirstFailed(),
				ShowErrorDialog: true,
				Err:             err,
			}
		}
		return ObjectWizardZonesActionResult{
			StatusText:       vm.stepVM.StatusCreatedFirst(),
			RefreshTable:     true,
			TargetZoneNumber: 1,
			FocusQuickName:   true,
		}
	}

	selectedZone := state.SelectedZone()
	if selectedZone < 0 || selectedZone >= state.ZoneCount() {
		targetZone := state.EffectiveZoneNumberAt(0)
		if targetZone <= 0 {
			targetZone = 1
		}
		_ = state.SelectZoneByNumber(targetZone)
		return ObjectWizardZonesActionResult{
			StatusText:       vm.stepVM.StatusSelectAndInput(),
			RefreshTable:     true,
			TargetZoneNumber: targetZone,
			FocusQuickName:   true,
		}
	}

	targetZone := state.EffectiveZoneNumberAt(selectedZone)
	return ObjectWizardZonesActionResult{
		StatusText:       vm.stepVM.StatusEditingPrompt(targetZone),
		RefreshTable:     true,
		TargetZoneNumber: targetZone,
		FocusQuickName:   true,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) PrepareDelete(state ObjectWizardZonesState) ObjectWizardZonesDeletePrompt {
	selectedZone := state.SelectedZone()
	if selectedZone < 0 || selectedZone >= state.ZoneCount() {
		return ObjectWizardZonesDeletePrompt{
			CanDelete:  false,
			StatusText: vm.stepVM.StatusSelectionRequired(),
		}
	}
	targetZone := state.EffectiveZoneNumberAt(selectedZone)
	return ObjectWizardZonesDeletePrompt{
		CanDelete:        true,
		TargetZoneNumber: targetZone,
		ConfirmText:      vm.stepVM.DeleteConfirmText(targetZone),
	}
}

func (vm *ObjectWizardZonesFlowViewModel) ApplyDelete(state ObjectWizardZonesState, targetZone int64) ObjectWizardZonesActionResult {
	state.DeleteSelectedZone()
	return ObjectWizardZonesActionResult{
		StatusText:   vm.stepVM.StatusDeleted(targetZone),
		RefreshTable: true,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) Fill(state ObjectWizardZonesState, count int64) ObjectWizardZonesActionResult {
	if err := state.FillZones(count); err != nil {
		return ObjectWizardZonesActionResult{
			StatusText:      vm.stepVM.StatusFillFailed(),
			ShowErrorDialog: true,
			Err:             err,
		}
	}
	return ObjectWizardZonesActionResult{
		StatusText:       vm.stepVM.StatusFilledTo(count),
		RefreshTable:     true,
		TargetZoneNumber: 1,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) Clear(state ObjectWizardZonesState) ObjectWizardZonesActionResult {
	state.ResetZones()
	return ObjectWizardZonesActionResult{
		StatusText:   vm.stepVM.StatusCleared(),
		RefreshTable: true,
	}
}

func (vm *ObjectWizardZonesFlowViewModel) DefaultFillCount(state ObjectWizardZonesState) int64 {
	maxZone := state.MaxZoneNumber()
	if maxZone > 0 {
		return maxZone
	}
	return 24
}

func (vm *ObjectWizardZonesFlowViewModel) ClearConfirmText() string {
	return vm.stepVM.ClearConfirmText()
}

func (vm *ObjectWizardZonesFlowViewModel) RefreshStatus(state ObjectWizardZonesState) string {
	return vm.stepVM.StatusCount(state.ZoneCount())
}
