package viewmodels

import (
	"obj_catalog_fyne_v3/pkg/utils"
	"slices"
)

// ObjectWizardPersonalsStateViewModel керує чернетками відповідальних осіб у майстрі.
type ObjectWizardPersonalsStateViewModel struct {
	pending  []ObjectPersonal
	selected int
}

func NewObjectWizardPersonalsStateViewModel() *ObjectWizardPersonalsStateViewModel {
	return &ObjectWizardPersonalsStateViewModel{
		selected: -1,
	}
}

func (vm *ObjectWizardPersonalsStateViewModel) Reset() {
	vm.pending = nil
	vm.selected = -1
}

func (vm *ObjectWizardPersonalsStateViewModel) Count() int {
	return len(vm.pending)
}

func (vm *ObjectWizardPersonalsStateViewModel) Items() []ObjectPersonal {
	return slices.Clone(vm.pending)
}

func (vm *ObjectWizardPersonalsStateViewModel) At(idx int) (ObjectPersonal, bool) {
	if idx < 0 || idx >= len(vm.pending) {
		return ObjectPersonal{}, false
	}
	return vm.pending[idx], true
}

func (vm *ObjectWizardPersonalsStateViewModel) Selected() int {
	return vm.selected
}

func (vm *ObjectWizardPersonalsStateViewModel) SetSelected(idx int) bool {
	if idx < 0 || idx >= len(vm.pending) {
		vm.selected = -1
		return false
	}
	vm.selected = idx
	return true
}

func (vm *ObjectWizardPersonalsStateViewModel) NextNumber() int64 {
	maxVal := int64(0)
	for _, item := range vm.pending {
		if item.Number > maxVal {
			maxVal = item.Number
		}
	}
	return maxVal + 1
}

func (vm *ObjectWizardPersonalsStateViewModel) Add(item ObjectPersonal) int {
	if item.Number <= 0 {
		item.Number = vm.NextNumber()
	}
	vm.pending = append(vm.pending, item)
	vm.selected = len(vm.pending) - 1
	return vm.selected
}

func (vm *ObjectWizardPersonalsStateViewModel) Update(idx int, item ObjectPersonal) bool {
	if idx < 0 || idx >= len(vm.pending) {
		return false
	}
	if item.Number <= 0 {
		item.Number = vm.pending[idx].Number
	}
	vm.pending[idx] = item
	vm.selected = idx
	return true
}

func (vm *ObjectWizardPersonalsStateViewModel) Delete(idx int) bool {
	if idx < 0 || idx >= len(vm.pending) {
		return false
	}
	vm.pending = append(vm.pending[:idx], vm.pending[idx+1:]...)
	vm.selected = -1
	return true
}

func (vm *ObjectWizardPersonalsStateViewModel) FullName(item ObjectPersonal) string {
	fullName := utils.JoinTrimmedNonEmpty(item.Surname, item.Name, item.SecName)
	if fullName == "" {
		return "(без ПІБ)"
	}
	return fullName
}
