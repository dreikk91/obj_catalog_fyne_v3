package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardPersonalsStateViewModel_CRUD(t *testing.T) {
	vm := NewObjectWizardPersonalsStateViewModel()

	added := vm.Add(contracts.AdminObjectPersonal{Name: "Ivan"})
	if added != 0 {
		t.Fatalf("unexpected added index: %d", added)
	}
	if vm.Count() != 1 {
		t.Fatalf("unexpected personal count: %d", vm.Count())
	}
	item, ok := vm.At(0)
	if !ok {
		t.Fatalf("expected personal at index 0")
	}
	if item.Number != 1 {
		t.Fatalf("expected auto number 1, got %d", item.Number)
	}

	updated := vm.Update(0, contracts.AdminObjectPersonal{
		Number:  0,
		Surname: "Petrenko",
		Name:    "Ivan",
	})
	if !updated {
		t.Fatalf("expected update success")
	}
	item, _ = vm.At(0)
	if item.Number != 1 {
		t.Fatalf("expected existing number to be preserved, got %d", item.Number)
	}
	if vm.FullName(item) != "Petrenko Ivan" {
		t.Fatalf("unexpected full name: %q", vm.FullName(item))
	}

	if !vm.Delete(0) {
		t.Fatalf("expected delete success")
	}
	if vm.Count() != 0 {
		t.Fatalf("expected empty state after delete")
	}
}

func TestObjectWizardPersonalsStateViewModel_NextNumberUsesMaxExisting(t *testing.T) {
	vm := NewObjectWizardPersonalsStateViewModel()
	vm.Add(contracts.AdminObjectPersonal{Number: 5})
	vm.Add(contracts.AdminObjectPersonal{Number: 2})

	if next := vm.NextNumber(); next != 6 {
		t.Fatalf("unexpected next number: %d", next)
	}
}

func TestObjectWizardZonesStateViewModel_Flow(t *testing.T) {
	vm := NewObjectWizardZonesStateViewModel()

	if err := vm.EnsureExists(2, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if err := vm.EnsureExists(1, "Main"); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if vm.Count() != 2 {
		t.Fatalf("unexpected zone count: %d", vm.Count())
	}
	if vm.EffectiveNumberAt(0) != 1 {
		t.Fatalf("expected sorted zones, got first=%d", vm.EffectiveNumberAt(0))
	}

	z, ok := vm.At(0)
	if !ok {
		t.Fatalf("expected zone at index 0")
	}
	z.Description = "Updated"
	if !vm.Update(0, z) {
		t.Fatalf("expected update success")
	}
	updated, _ := vm.At(0)
	if updated.Description != "Updated" {
		t.Fatalf("unexpected description: %q", updated.Description)
	}
	if vm.MaxNumber() != 2 {
		t.Fatalf("unexpected max zone: %d", vm.MaxNumber())
	}
	if !vm.Delete(1) {
		t.Fatalf("expected delete success")
	}
	if vm.Count() != 1 {
		t.Fatalf("unexpected zone count after delete: %d", vm.Count())
	}
}

func TestObjectWizardZonesStateViewModel_NextZoneNumberForAdd_UsesSelectedOrMax(t *testing.T) {
	vm := NewObjectWizardZonesStateViewModel()
	if next := vm.NextNumberForAdd(); next != 1 {
		t.Fatalf("unexpected next zone for empty state: %d", next)
	}

	if err := vm.EnsureExists(3, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if err := vm.EnsureExists(1, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if next := vm.NextNumberForAdd(); next != 4 {
		t.Fatalf("expected max+1, got %d", next)
	}
	if !vm.SelectByNumber(1) {
		t.Fatalf("expected zone selection")
	}
	if next := vm.NextNumberForAdd(); next != 2 {
		t.Fatalf("expected selected+1, got %d", next)
	}
}

func TestObjectWizardZonesStateViewModel_EnsureFirstZoneAndFillZones(t *testing.T) {
	vm := NewObjectWizardZonesStateViewModel()

	zoneNumber, err := vm.EnsureFirst("Custom")
	if err != nil {
		t.Fatalf("unexpected ensure first error: %v", err)
	}
	if zoneNumber != 1 {
		t.Fatalf("unexpected first zone number: %d", zoneNumber)
	}
	if vm.SelectedLabel() != "Зона: #1" {
		t.Fatalf("unexpected selected label: %q", vm.SelectedLabel())
	}

	if err := vm.Fill(3); err != nil {
		t.Fatalf("unexpected fill error: %v", err)
	}
	if vm.Count() != 3 {
		t.Fatalf("unexpected zone count after fill: %d", vm.Count())
	}
	first, ok := vm.At(0)
	if !ok || first.Description != "Custom" {
		t.Fatalf("expected to preserve first description, got %+v", first)
	}
}
