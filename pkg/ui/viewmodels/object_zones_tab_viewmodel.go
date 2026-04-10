package viewmodels

import (
	"fmt"
	"slices"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectZonesTabViewModel керує станом вкладки зон у картці об'єкта.
type ObjectZonesTabViewModel struct {
	items       []contracts.AdminObjectZone
	selectedRow int
}

func NewObjectZonesTabViewModel() *ObjectZonesTabViewModel {
	return &ObjectZonesTabViewModel{
		selectedRow: -1,
	}
}

func (vm *ObjectZonesTabViewModel) SetItems(items []contracts.AdminObjectZone) {
	vm.items = slices.Clone(items)
	vm.selectedRow = -1
}

func (vm *ObjectZonesTabViewModel) Items() []contracts.AdminObjectZone {
	return slices.Clone(vm.items)
}

func (vm *ObjectZonesTabViewModel) Count() int {
	return len(vm.items)
}

func (vm *ObjectZonesTabViewModel) CountStatusText() string {
	return fmt.Sprintf("Зони: %d запис(ів)", vm.Count())
}

func (vm *ObjectZonesTabViewModel) ItemAt(idx int) (contracts.AdminObjectZone, bool) {
	if idx < 0 || idx >= len(vm.items) {
		return contracts.AdminObjectZone{}, false
	}
	return vm.items[idx], true
}

func (vm *ObjectZonesTabViewModel) SelectedItem() (contracts.AdminObjectZone, bool) {
	return vm.ItemAt(vm.selectedRow)
}

func (vm *ObjectZonesTabViewModel) SelectedRow() int {
	return vm.selectedRow
}

func (vm *ObjectZonesTabViewModel) SelectByTableRow(row int) bool {
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

func (vm *ObjectZonesTabViewModel) SelectZoneByNumber(zoneNumber int64) bool {
	if len(vm.items) == 0 {
		vm.selectedRow = -1
		return false
	}

	targetRow := 0
	if zoneNumber > 0 {
		if row := vm.FindRowByZoneNumber(zoneNumber); row >= 0 {
			targetRow = row
		}
	}
	vm.selectedRow = targetRow
	return true
}

func (vm *ObjectZonesTabViewModel) EffectiveZoneNumberAt(idx int) int64 {
	if idx < 0 || idx >= len(vm.items) {
		return 0
	}
	if vm.items[idx].ZoneNumber > 0 {
		return vm.items[idx].ZoneNumber
	}
	return int64(idx) + 1
}

func (vm *ObjectZonesTabViewModel) FindRowByZoneNumber(zoneNumber int64) int {
	if zoneNumber <= 0 {
		return -1
	}
	for i := range vm.items {
		if vm.EffectiveZoneNumberAt(i) == zoneNumber {
			return i
		}
	}
	return -1
}

func (vm *ObjectZonesTabViewModel) SelectedZoneNumber() (int64, bool) {
	if vm.selectedRow < 0 || vm.selectedRow >= len(vm.items) {
		return 0, false
	}
	return vm.EffectiveZoneNumberAt(vm.selectedRow), true
}

func (vm *ObjectZonesTabViewModel) SelectedZoneDescription() string {
	item, ok := vm.SelectedItem()
	if !ok {
		return ""
	}
	return strings.TrimSpace(item.Description)
}

func (vm *ObjectZonesTabViewModel) SelectedZoneLabel() string {
	if zoneNumber, ok := vm.SelectedZoneNumber(); ok {
		return fmt.Sprintf("Зона: #%d", zoneNumber)
	}
	return "Зона: —"
}

func (vm *ObjectZonesTabViewModel) NextZoneNumberForAdd() int64 {
	if currentZone, ok := vm.SelectedZoneNumber(); ok {
		return currentZone + 1
	}
	if len(vm.items) > 0 {
		lastZone := vm.EffectiveZoneNumberAt(len(vm.items) - 1)
		if lastZone > 0 {
			return lastZone + 1
		}
	}
	return 1
}

func (vm *ObjectZonesTabViewModel) BuildZoneForCreate(zoneNumber int64, defaultDescription string) (contracts.AdminObjectZone, error) {
	if zoneNumber <= 0 {
		return contracts.AdminObjectZone{}, fmt.Errorf("invalid zone number")
	}
	description := strings.TrimSpace(defaultDescription)
	if description == "" {
		description = fmt.Sprintf("Шлейф %d", zoneNumber)
	}
	return contracts.AdminObjectZone{
		ZoneNumber:    zoneNumber,
		ZoneType:      1,
		Description:   description,
		EntryDelaySec: 0,
	}, nil
}

func (vm *ObjectZonesTabViewModel) PrepareSelectedZoneForSave(description string) (contracts.AdminObjectZone, int64, bool) {
	current, ok := vm.SelectedItem()
	if !ok {
		return contracts.AdminObjectZone{}, 0, false
	}
	currentZoneNumber, ok := vm.SelectedZoneNumber()
	if !ok {
		return contracts.AdminObjectZone{}, 0, false
	}
	if current.ZoneNumber <= 0 {
		current.ZoneNumber = currentZoneNumber
	}
	current.Description = strings.TrimSpace(description)
	return current, current.ZoneNumber, true
}
