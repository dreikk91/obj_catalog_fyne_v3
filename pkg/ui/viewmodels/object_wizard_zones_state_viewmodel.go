package viewmodels

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectWizardZonesStateViewModel керує чернетками зон у майстрі.
type ObjectWizardZonesStateViewModel struct {
	pending  []contracts.AdminObjectZone
	selected int
}

func NewObjectWizardZonesStateViewModel() *ObjectWizardZonesStateViewModel {
	return &ObjectWizardZonesStateViewModel{
		selected: -1,
	}
}

func (vm *ObjectWizardZonesStateViewModel) Reset() {
	vm.pending = nil
	vm.selected = -1
}

func (vm *ObjectWizardZonesStateViewModel) Count() int {
	return len(vm.pending)
}

func (vm *ObjectWizardZonesStateViewModel) Items() []contracts.AdminObjectZone {
	return slices.Clone(vm.pending)
}

func (vm *ObjectWizardZonesStateViewModel) At(idx int) (contracts.AdminObjectZone, bool) {
	if idx < 0 || idx >= len(vm.pending) {
		return contracts.AdminObjectZone{}, false
	}
	return vm.pending[idx], true
}

func (vm *ObjectWizardZonesStateViewModel) Selected() int {
	return vm.selected
}

func (vm *ObjectWizardZonesStateViewModel) SetSelected(idx int) bool {
	if idx < 0 || idx >= len(vm.pending) {
		vm.selected = -1
		return false
	}
	vm.selected = idx
	return true
}

func (vm *ObjectWizardZonesStateViewModel) EffectiveNumberAt(idx int) int64 {
	if idx < 0 || idx >= len(vm.pending) {
		return 0
	}
	if vm.pending[idx].ZoneNumber > 0 {
		return vm.pending[idx].ZoneNumber
	}
	return int64(idx) + 1
}

func (vm *ObjectWizardZonesStateViewModel) FindRowByNumber(zoneNumber int64) int {
	if zoneNumber <= 0 {
		return -1
	}
	for i := range vm.pending {
		if vm.EffectiveNumberAt(i) == zoneNumber {
			return i
		}
	}
	return -1
}

func (vm *ObjectWizardZonesStateViewModel) SelectByNumber(zoneNumber int64) bool {
	if len(vm.pending) == 0 {
		vm.selected = -1
		return false
	}
	target := 0
	if zoneNumber > 0 {
		if row := vm.FindRowByNumber(zoneNumber); row >= 0 {
			target = row
		}
	}
	vm.selected = target
	return true
}

func (vm *ObjectWizardZonesStateViewModel) SelectedData() (idx int, zone contracts.AdminObjectZone, zoneNumber int64, ok bool) {
	idx = vm.selected
	zone, ok = vm.At(idx)
	if !ok {
		return -1, contracts.AdminObjectZone{}, 0, false
	}
	zoneNumber = vm.EffectiveNumberAt(idx)
	return idx, zone, zoneNumber, true
}

func (vm *ObjectWizardZonesStateViewModel) SelectedDescription() string {
	_, zone, _, ok := vm.SelectedData()
	if !ok {
		return ""
	}
	return strings.TrimSpace(zone.Description)
}

func (vm *ObjectWizardZonesStateViewModel) EnsureExists(zoneNumber int64, defaultDescription string) error {
	if zoneNumber <= 0 {
		return fmt.Errorf("некоректний номер зони")
	}
	if vm.FindRowByNumber(zoneNumber) >= 0 {
		return nil
	}
	description := strings.TrimSpace(defaultDescription)
	if description == "" {
		description = fmt.Sprintf("Шлейф %d", zoneNumber)
	}
	vm.pending = append(vm.pending, contracts.AdminObjectZone{
		ZoneNumber:    zoneNumber,
		ZoneType:      1,
		Description:   description,
		EntryDelaySec: 0,
	})
	vm.sortPending()
	return nil
}

func (vm *ObjectWizardZonesStateViewModel) Update(idx int, zone contracts.AdminObjectZone) bool {
	if idx < 0 || idx >= len(vm.pending) {
		return false
	}
	vm.pending[idx] = zone
	vm.selected = idx
	vm.sortPending()
	return true
}

func (vm *ObjectWizardZonesStateViewModel) SaveSelectedAndEnsureNext(description string) (currentZone int64, nextZone int64, err error) {
	idx, zone, zoneNumber, ok := vm.SelectedData()
	if !ok {
		return 0, 0, fmt.Errorf("зону не вибрано")
	}

	if zone.ZoneNumber <= 0 {
		zone.ZoneNumber = zoneNumber
	}
	zone.Description = strings.TrimSpace(description)
	if !vm.Update(idx, zone) {
		return 0, 0, fmt.Errorf("не вдалося оновити зону")
	}

	currentZone = zone.ZoneNumber
	nextZone = currentZone + 1
	if err := vm.EnsureExists(nextZone, ""); err != nil {
		return 0, 0, err
	}
	vm.SelectByNumber(nextZone)
	return currentZone, nextZone, nil
}

func (vm *ObjectWizardZonesStateViewModel) NextNumberForAdd() int64 {
	if _, _, zoneNumber, ok := vm.SelectedData(); ok {
		return zoneNumber + 1
	}
	if maxZone := vm.MaxNumber(); maxZone > 0 {
		return maxZone + 1
	}
	return 1
}

func (vm *ObjectWizardZonesStateViewModel) SelectFirst() bool {
	return vm.SelectByNumber(0)
}

func (vm *ObjectWizardZonesStateViewModel) EnsureFirst(defaultDescription string) (int64, error) {
	if err := vm.EnsureExists(1, defaultDescription); err != nil {
		return 0, err
	}
	if !vm.SelectByNumber(1) {
		return 0, fmt.Errorf("не вдалося вибрати першу зону")
	}
	return 1, nil
}

func (vm *ObjectWizardZonesStateViewModel) Fill(count int64) error {
	if count <= 0 {
		return fmt.Errorf("кількість зон має бути більше 0")
	}
	existingDescriptions := make(map[int64]string, vm.Count())
	for i := 0; i < vm.Count(); i++ {
		zoneNumber := vm.EffectiveNumberAt(i)
		if zone, ok := vm.At(i); ok {
			existingDescriptions[zoneNumber] = strings.TrimSpace(zone.Description)
		}
	}
	for zoneNumber := int64(1); zoneNumber <= count; zoneNumber++ {
		if err := vm.EnsureExists(zoneNumber, existingDescriptions[zoneNumber]); err != nil {
			return err
		}
	}
	vm.SelectByNumber(1)
	return nil
}

func (vm *ObjectWizardZonesStateViewModel) DeleteSelected() (int64, bool) {
	idx, _, zoneNumber, ok := vm.SelectedData()
	if !ok {
		return 0, false
	}
	return zoneNumber, vm.Delete(idx)
}

func (vm *ObjectWizardZonesStateViewModel) SelectedLabel() string {
	if _, _, zoneNumber, ok := vm.SelectedData(); ok {
		return fmt.Sprintf("Зона: #%d", zoneNumber)
	}
	return "Зона: —"
}

func (vm *ObjectWizardZonesStateViewModel) Delete(idx int) bool {
	if idx < 0 || idx >= len(vm.pending) {
		return false
	}
	vm.pending = append(vm.pending[:idx], vm.pending[idx+1:]...)
	vm.selected = -1
	vm.sortPending()
	return true
}

func (vm *ObjectWizardZonesStateViewModel) MaxNumber() int64 {
	maxZone := int64(0)
	for i := range vm.pending {
		if zoneNumber := vm.EffectiveNumberAt(i); zoneNumber > maxZone {
			maxZone = zoneNumber
		}
	}
	return maxZone
}

func (vm *ObjectWizardZonesStateViewModel) sortPending() {
	sort.SliceStable(vm.pending, func(i, j int) bool {
		left := vm.pending[i].ZoneNumber
		right := vm.pending[j].ZoneNumber
		if left <= 0 {
			left = int64(i) + 1
		}
		if right <= 0 {
			right = int64(j) + 1
		}
		return left < right
	})
}
