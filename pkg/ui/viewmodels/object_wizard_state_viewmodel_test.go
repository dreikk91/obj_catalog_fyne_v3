package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardStateViewModel_PersonalsCRUD(t *testing.T) {
	vm := NewObjectWizardStateViewModel()

	added := vm.AddPersonal(contracts.AdminObjectPersonal{Name: "Ivan"})
	if added != 0 {
		t.Fatalf("unexpected added index: %d", added)
	}
	if vm.PersonalCount() != 1 {
		t.Fatalf("unexpected personal count: %d", vm.PersonalCount())
	}
	item, ok := vm.PersonalAt(0)
	if !ok {
		t.Fatalf("expected personal at index 0")
	}
	if item.Number != 1 {
		t.Fatalf("expected auto number 1, got %d", item.Number)
	}

	updated := vm.UpdatePersonal(0, contracts.AdminObjectPersonal{
		Number:  0,
		Surname: "Petrenko",
		Name:    "Ivan",
	})
	if !updated {
		t.Fatalf("expected update success")
	}
	item, _ = vm.PersonalAt(0)
	if item.Number != 1 {
		t.Fatalf("expected existing number to be preserved, got %d", item.Number)
	}
	if vm.PersonalFullName(item) != "Petrenko Ivan" {
		t.Fatalf("unexpected full name: %q", vm.PersonalFullName(item))
	}

	if !vm.DeletePersonal(0) {
		t.Fatalf("expected delete success")
	}
	if vm.PersonalCount() != 0 {
		t.Fatalf("expected empty state after delete")
	}
}

func TestObjectWizardStateViewModel_NextNumberUsesMaxExisting(t *testing.T) {
	vm := NewObjectWizardStateViewModel()
	vm.AddPersonal(contracts.AdminObjectPersonal{Number: 5})
	vm.AddPersonal(contracts.AdminObjectPersonal{Number: 2})

	if next := vm.NextPersonalNumber(); next != 6 {
		t.Fatalf("unexpected next number: %d", next)
	}
}

func TestObjectWizardStateViewModel_ZonesFlow(t *testing.T) {
	vm := NewObjectWizardStateViewModel()

	if err := vm.EnsureZoneExists(2, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if err := vm.EnsureZoneExists(1, "Main"); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if vm.ZoneCount() != 2 {
		t.Fatalf("unexpected zone count: %d", vm.ZoneCount())
	}
	if vm.EffectiveZoneNumberAt(0) != 1 {
		t.Fatalf("expected sorted zones, got first=%d", vm.EffectiveZoneNumberAt(0))
	}

	z, ok := vm.ZoneAt(0)
	if !ok {
		t.Fatalf("expected zone at index 0")
	}
	z.Description = "Updated"
	if !vm.UpdateZone(0, z) {
		t.Fatalf("expected update success")
	}
	updated, _ := vm.ZoneAt(0)
	if updated.Description != "Updated" {
		t.Fatalf("unexpected description: %q", updated.Description)
	}
	if vm.MaxZoneNumber() != 2 {
		t.Fatalf("unexpected max zone: %d", vm.MaxZoneNumber())
	}
	if !vm.DeleteZone(1) {
		t.Fatalf("expected delete success")
	}
	if vm.ZoneCount() != 1 {
		t.Fatalf("unexpected zone count after delete: %d", vm.ZoneCount())
	}
}

func TestObjectWizardStateViewModel_SaveSelectedZoneAndEnsureNext(t *testing.T) {
	vm := NewObjectWizardStateViewModel()
	if err := vm.EnsureZoneExists(1, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if !vm.SelectZoneByNumber(1) {
		t.Fatalf("expected zone selection")
	}

	current, next, err := vm.SaveSelectedZoneAndEnsureNext("New name")
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	if current != 1 || next != 2 {
		t.Fatalf("unexpected transition: %d -> %d", current, next)
	}
	if vm.ZoneCount() != 2 {
		t.Fatalf("expected auto-created next zone")
	}
	if vm.SelectedZoneLabel() != "Зона: #2" {
		t.Fatalf("unexpected selected label: %q", vm.SelectedZoneLabel())
	}
}

func TestObjectWizardStateViewModel_NextZoneNumberForAdd_UsesSelectedOrMax(t *testing.T) {
	vm := NewObjectWizardStateViewModel()
	if next := vm.NextZoneNumberForAdd(); next != 1 {
		t.Fatalf("unexpected next zone for empty state: %d", next)
	}

	if err := vm.EnsureZoneExists(3, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if err := vm.EnsureZoneExists(1, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if next := vm.NextZoneNumberForAdd(); next != 4 {
		t.Fatalf("expected max+1, got %d", next)
	}
	if !vm.SelectZoneByNumber(1) {
		t.Fatalf("expected zone selection")
	}
	if next := vm.NextZoneNumberForAdd(); next != 2 {
		t.Fatalf("expected selected+1, got %d", next)
	}
}

func TestObjectWizardStateViewModel_EnsureFirstZoneAndFillZones(t *testing.T) {
	vm := NewObjectWizardStateViewModel()

	zoneNumber, err := vm.EnsureFirstZone("Custom")
	if err != nil {
		t.Fatalf("unexpected ensure first error: %v", err)
	}
	if zoneNumber != 1 {
		t.Fatalf("unexpected first zone number: %d", zoneNumber)
	}
	if vm.SelectedZoneLabel() != "Зона: #1" {
		t.Fatalf("unexpected selected label: %q", vm.SelectedZoneLabel())
	}

	if err := vm.FillZones(3); err != nil {
		t.Fatalf("unexpected fill error: %v", err)
	}
	if vm.ZoneCount() != 3 {
		t.Fatalf("unexpected zone count after fill: %d", vm.ZoneCount())
	}
	first, ok := vm.ZoneAt(0)
	if !ok || first.Description != "Custom" {
		t.Fatalf("expected to preserve first description, got %+v", first)
	}
}
