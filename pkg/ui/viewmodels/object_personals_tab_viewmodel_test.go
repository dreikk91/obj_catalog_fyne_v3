package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectPersonalsTabViewModel_SetItemsAndSelect(t *testing.T) {
	vm := NewObjectPersonalsTabViewModel()
	vm.SetItems([]contracts.AdminObjectPersonal{
		{ID: 1, Name: "Іван"},
		{ID: 2, Name: "Петро"},
	})

	if vm.Count() != 2 {
		t.Fatalf("unexpected count: %d", vm.Count())
	}
	if !vm.SelectByTableRow(2) {
		t.Fatalf("expected selection success")
	}
	item, ok := vm.SelectedItem()
	if !ok {
		t.Fatalf("expected selected item")
	}
	if item.ID != 2 {
		t.Fatalf("unexpected selected id: %d", item.ID)
	}
}

func TestObjectPersonalsTabViewModel_SelectInvalid(t *testing.T) {
	vm := NewObjectPersonalsTabViewModel()
	vm.SetItems([]contracts.AdminObjectPersonal{{ID: 1}})

	if vm.SelectByTableRow(0) {
		t.Fatalf("header row must not be selectable")
	}
	if vm.SelectByTableRow(99) {
		t.Fatalf("out-of-range row must not be selectable")
	}
	if _, ok := vm.SelectedItem(); ok {
		t.Fatalf("must have no selected item")
	}
}

func TestObjectPersonalsTabViewModel_FullName(t *testing.T) {
	vm := NewObjectPersonalsTabViewModel()
	got := vm.FullName(contracts.AdminObjectPersonal{
		Surname: " Петренко ",
		Name:    " Іван ",
		SecName: " Іванович ",
	})
	if got != "Петренко Іван Іванович" {
		t.Fatalf("unexpected full name: %q", got)
	}
}

func TestObjectPersonalsTabViewModel_PrepareUpdatedItem(t *testing.T) {
	vm := NewObjectPersonalsTabViewModel()
	original := contracts.AdminObjectPersonal{
		ID:        77,
		CreatedAt: "2026-01-02 03:04:05",
		Name:      "Old",
	}
	edited := contracts.AdminObjectPersonal{
		Name: "New",
	}

	prepared := vm.PrepareUpdatedItem(original, edited)
	if prepared.ID != 77 {
		t.Fatalf("id must be kept from original")
	}
	if prepared.CreatedAt != original.CreatedAt {
		t.Fatalf("created_at must be kept from original when blank")
	}
	if prepared.Name != "New" {
		t.Fatalf("name must come from edited item")
	}
}
