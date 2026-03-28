package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardPersonalsStateViewModel_CRUDAndSelection(t *testing.T) {
	vm := NewObjectWizardPersonalsStateViewModel()

	if idx := vm.Add(contracts.AdminObjectPersonal{Name: "A"}); idx != 0 {
		t.Fatalf("unexpected add index: %d", idx)
	}
	if vm.Count() != 1 {
		t.Fatalf("unexpected count: %d", vm.Count())
	}
	if !vm.SetSelected(0) {
		t.Fatalf("expected selection success")
	}
	selected, ok := vm.At(vm.Selected())
	if !ok || selected.Number != 1 {
		t.Fatalf("unexpected selected item: %+v", selected)
	}

	if !vm.Update(0, contracts.AdminObjectPersonal{Number: 0, Surname: "Petrenko", Name: "Ivan"}) {
		t.Fatalf("expected update success")
	}
	updated, _ := vm.At(0)
	if updated.Number != 1 {
		t.Fatalf("expected preserved number, got %d", updated.Number)
	}
	if vm.FullName(updated) != "Petrenko Ivan" {
		t.Fatalf("unexpected full name: %q", vm.FullName(updated))
	}

	if !vm.Delete(0) || vm.Count() != 0 {
		t.Fatalf("delete must clear list")
	}
}
