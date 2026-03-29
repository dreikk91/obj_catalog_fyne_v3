package viewmodels

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectPersonalsTabViewModel керує станом вкладки відповідальних осіб (В/О).
type ObjectPersonalsTabViewModel struct {
	items       []contracts.AdminObjectPersonal
	selectedRow int
}

func NewObjectPersonalsTabViewModel() *ObjectPersonalsTabViewModel {
	return &ObjectPersonalsTabViewModel{
		selectedRow: -1,
	}
}

func (vm *ObjectPersonalsTabViewModel) SetItems(items []contracts.AdminObjectPersonal) {
	vm.items = append([]contracts.AdminObjectPersonal(nil), items...)
	vm.selectedRow = -1
}

func (vm *ObjectPersonalsTabViewModel) Count() int {
	return len(vm.items)
}

func (vm *ObjectPersonalsTabViewModel) CountStatusText() string {
	return fmt.Sprintf("В/О: %d запис(ів)", vm.Count())
}

func (vm *ObjectPersonalsTabViewModel) ItemAt(idx int) (contracts.AdminObjectPersonal, bool) {
	if idx < 0 || idx >= len(vm.items) {
		return contracts.AdminObjectPersonal{}, false
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

func (vm *ObjectPersonalsTabViewModel) SelectedItem() (contracts.AdminObjectPersonal, bool) {
	return vm.ItemAt(vm.selectedRow)
}

func (vm *ObjectPersonalsTabViewModel) FullName(item contracts.AdminObjectPersonal) string {
	parts := []string{
		strings.TrimSpace(item.Surname),
		strings.TrimSpace(item.Name),
		strings.TrimSpace(item.SecName),
	}
	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) == 0 {
		return "(без ПІБ)"
	}
	return strings.Join(filtered, " ")
}

func (vm *ObjectPersonalsTabViewModel) PrepareUpdatedItem(
	original contracts.AdminObjectPersonal,
	edited contracts.AdminObjectPersonal,
) contracts.AdminObjectPersonal {
	edited.ID = original.ID
	if strings.TrimSpace(edited.CreatedAt) == "" {
		edited.CreatedAt = original.CreatedAt
	}
	return edited
}
