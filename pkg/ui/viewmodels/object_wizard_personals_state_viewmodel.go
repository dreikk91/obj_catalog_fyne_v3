package viewmodels

import (
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectWizardPersonalsStateViewModel керує чернетками відповідальних осіб у майстрі.
type ObjectWizardPersonalsStateViewModel struct {
	pending  []contracts.AdminObjectPersonal
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

func (vm *ObjectWizardPersonalsStateViewModel) Items() []contracts.AdminObjectPersonal {
	return append([]contracts.AdminObjectPersonal(nil), vm.pending...)
}

func (vm *ObjectWizardPersonalsStateViewModel) At(idx int) (contracts.AdminObjectPersonal, bool) {
	if idx < 0 || idx >= len(vm.pending) {
		return contracts.AdminObjectPersonal{}, false
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

func (vm *ObjectWizardPersonalsStateViewModel) Add(item contracts.AdminObjectPersonal) int {
	if item.Number <= 0 {
		item.Number = vm.NextNumber()
	}
	vm.pending = append(vm.pending, item)
	vm.selected = len(vm.pending) - 1
	return vm.selected
}

func (vm *ObjectWizardPersonalsStateViewModel) Update(idx int, item contracts.AdminObjectPersonal) bool {
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

func (vm *ObjectWizardPersonalsStateViewModel) FullName(item contracts.AdminObjectPersonal) string {
	parts := []string{
		strings.TrimSpace(item.Surname),
		strings.TrimSpace(item.Name),
		strings.TrimSpace(item.SecName),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return "(без ПІБ)"
	}
	return strings.Join(filtered, " ")
}
