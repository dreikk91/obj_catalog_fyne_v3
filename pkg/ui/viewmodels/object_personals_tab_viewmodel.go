package viewmodels

import (
	"fmt"
	"slices"
	"strings"

	"obj_catalog_fyne_v3/pkg/utils"
)

// ObjectPersonalsTabViewModel керує станом вкладки відповідальних осіб (В/О).
type ObjectPersonalsTabViewModel struct {
	items       []ObjectPersonal
	selectedRow int
}

func NewObjectPersonalsTabViewModel() *ObjectPersonalsTabViewModel {
	return &ObjectPersonalsTabViewModel{
		selectedRow: -1,
	}
}

func (vm *ObjectPersonalsTabViewModel) SetItems(items []ObjectPersonal) {
	vm.items = slices.Clone(items)
	vm.selectedRow = -1
}

func (vm *ObjectPersonalsTabViewModel) Count() int {
	return len(vm.items)
}

func (vm *ObjectPersonalsTabViewModel) CountStatusText() string {
	return fmt.Sprintf("В/О: %d запис(ів)", vm.Count())
}

func (vm *ObjectPersonalsTabViewModel) ItemAt(idx int) (ObjectPersonal, bool) {
	if idx < 0 || idx >= len(vm.items) {
		return ObjectPersonal{}, false
	}
	return vm.items[idx], true
}

func (vm *ObjectPersonalsTabViewModel) SelectByTableRow(row int) bool {
	if row <= 0 {
		vm.selectedRow = -1
		return false
	}
	itemIdx := row - 1
	if itemIdx < 0 || itemIdx >= len(vm.items) {
		vm.selectedRow = -1
		return false
	}
	vm.selectedRow = itemIdx
	return true
}

func (vm *ObjectPersonalsTabViewModel) SelectedItem() (ObjectPersonal, bool) {
	return vm.ItemAt(vm.selectedRow)
}

func (vm *ObjectPersonalsTabViewModel) FullName(item ObjectPersonal) string {
	fullName := utils.JoinTrimmedNonEmpty(item.Surname, item.Name, item.SecName)
	if fullName == "" {
		return "(без ПІБ)"
	}
	return fullName
}

func (vm *ObjectPersonalsTabViewModel) PrepareUpdatedItem(
	original ObjectPersonal,
	edited ObjectPersonal,
) ObjectPersonal {
	edited.ID = original.ID
	if strings.TrimSpace(edited.CreatedAt) == "" {
		edited.CreatedAt = original.CreatedAt
	}
	return edited
}
