package viewmodels

import "obj_catalog_fyne_v3/pkg/contracts"

// ObjectWizardStateViewModel - сумісний фасад над під-станами майстра (В/О та зони).
type ObjectWizardStateViewModel struct {
	personals *ObjectWizardPersonalsStateViewModel
	zones     *ObjectWizardZonesStateViewModel
}

func NewObjectWizardStateViewModel() *ObjectWizardStateViewModel {
	return &ObjectWizardStateViewModel{
		personals: NewObjectWizardPersonalsStateViewModel(),
		zones:     NewObjectWizardZonesStateViewModel(),
	}
}

func (vm *ObjectWizardStateViewModel) ResetPersonals() {
	vm.personals.Reset()
}

func (vm *ObjectWizardStateViewModel) ResetZones() {
	vm.zones.Reset()
}

func (vm *ObjectWizardStateViewModel) PersonalCount() int {
	return vm.personals.Count()
}

func (vm *ObjectWizardStateViewModel) Personals() []contracts.AdminObjectPersonal {
	return vm.personals.Items()
}

func (vm *ObjectWizardStateViewModel) Zones() []contracts.AdminObjectZone {
	return vm.zones.Items()
}

func (vm *ObjectWizardStateViewModel) PersonalAt(idx int) (contracts.AdminObjectPersonal, bool) {
	return vm.personals.At(idx)
}

func (vm *ObjectWizardStateViewModel) SelectedPersonal() int {
	return vm.personals.Selected()
}

func (vm *ObjectWizardStateViewModel) SelectedZone() int {
	return vm.zones.Selected()
}

func (vm *ObjectWizardStateViewModel) SetSelectedPersonal(idx int) bool {
	return vm.personals.SetSelected(idx)
}

func (vm *ObjectWizardStateViewModel) SetSelectedZone(idx int) bool {
	return vm.zones.SetSelected(idx)
}

func (vm *ObjectWizardStateViewModel) SelectZoneByNumber(zoneNumber int64) bool {
	return vm.zones.SelectByNumber(zoneNumber)
}

func (vm *ObjectWizardStateViewModel) NextPersonalNumber() int64 {
	return vm.personals.NextNumber()
}

func (vm *ObjectWizardStateViewModel) ZoneCount() int {
	return vm.zones.Count()
}

func (vm *ObjectWizardStateViewModel) ZoneAt(idx int) (contracts.AdminObjectZone, bool) {
	return vm.zones.At(idx)
}

func (vm *ObjectWizardStateViewModel) SelectedZoneData() (idx int, zone contracts.AdminObjectZone, zoneNumber int64, ok bool) {
	return vm.zones.SelectedData()
}

func (vm *ObjectWizardStateViewModel) SelectedZoneDescription() string {
	return vm.zones.SelectedDescription()
}

func (vm *ObjectWizardStateViewModel) AddPersonal(item contracts.AdminObjectPersonal) int {
	return vm.personals.Add(item)
}

func (vm *ObjectWizardStateViewModel) UpdatePersonal(idx int, item contracts.AdminObjectPersonal) bool {
	return vm.personals.Update(idx, item)
}

func (vm *ObjectWizardStateViewModel) DeletePersonal(idx int) bool {
	return vm.personals.Delete(idx)
}

func (vm *ObjectWizardStateViewModel) EffectiveZoneNumberAt(idx int) int64 {
	return vm.zones.EffectiveNumberAt(idx)
}

func (vm *ObjectWizardStateViewModel) FindZoneRowByNumber(zoneNumber int64) int {
	return vm.zones.FindRowByNumber(zoneNumber)
}

func (vm *ObjectWizardStateViewModel) EnsureZoneExists(zoneNumber int64, defaultDescription string) error {
	return vm.zones.EnsureExists(zoneNumber, defaultDescription)
}

func (vm *ObjectWizardStateViewModel) UpdateZone(idx int, zone contracts.AdminObjectZone) bool {
	return vm.zones.Update(idx, zone)
}

func (vm *ObjectWizardStateViewModel) SaveSelectedZoneAndEnsureNext(description string) (currentZone int64, nextZone int64, err error) {
	return vm.zones.SaveSelectedAndEnsureNext(description)
}

func (vm *ObjectWizardStateViewModel) NextZoneNumberForAdd() int64 {
	return vm.zones.NextNumberForAdd()
}

func (vm *ObjectWizardStateViewModel) SelectFirstZone() bool {
	return vm.zones.SelectFirst()
}

func (vm *ObjectWizardStateViewModel) EnsureFirstZone(defaultDescription string) (int64, error) {
	return vm.zones.EnsureFirst(defaultDescription)
}

func (vm *ObjectWizardStateViewModel) FillZones(count int64) error {
	return vm.zones.Fill(count)
}

func (vm *ObjectWizardStateViewModel) DeleteSelectedZone() (int64, bool) {
	return vm.zones.DeleteSelected()
}

func (vm *ObjectWizardStateViewModel) SelectedZoneLabel() string {
	return vm.zones.SelectedLabel()
}

func (vm *ObjectWizardStateViewModel) DeleteZone(idx int) bool {
	return vm.zones.Delete(idx)
}

func (vm *ObjectWizardStateViewModel) MaxZoneNumber() int64 {
	return vm.zones.MaxNumber()
}

func (vm *ObjectWizardStateViewModel) PersonalFullName(item contracts.AdminObjectPersonal) string {
	return vm.personals.FullName(item)
}
