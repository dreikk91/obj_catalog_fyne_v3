package viewmodels

import "testing"

func TestObjectWizardZonesStateViewModel_SaveSelectedAndEnsureNext(t *testing.T) {
	vm := NewObjectWizardZonesStateViewModel()
	if err := vm.EnsureExists(1, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if !vm.SelectByNumber(1) {
		t.Fatalf("expected select by number")
	}

	current, next, err := vm.SaveSelectedAndEnsureNext("Zone 1")
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	if current != 1 || next != 2 {
		t.Fatalf("unexpected transition: %d -> %d", current, next)
	}
	if vm.Count() != 2 {
		t.Fatalf("expected auto-created next zone")
	}
	if vm.SelectedLabel() != "Зона: #2" {
		t.Fatalf("unexpected selected label: %q", vm.SelectedLabel())
	}
}

func TestObjectWizardZonesStateViewModel_FillAndDeleteSelected(t *testing.T) {
	vm := NewObjectWizardZonesStateViewModel()
	if _, err := vm.EnsureFirst("Custom"); err != nil {
		t.Fatalf("unexpected ensure first error: %v", err)
	}
	if err := vm.Fill(3); err != nil {
		t.Fatalf("unexpected fill error: %v", err)
	}
	if vm.Count() != 3 {
		t.Fatalf("unexpected count after fill: %d", vm.Count())
	}
	if zone, ok := vm.At(0); !ok || zone.Description != "Custom" {
		t.Fatalf("expected to preserve description, got %+v", zone)
	}

	if _, deleted := vm.DeleteSelected(); !deleted {
		t.Fatalf("expected selected zone delete success")
	}
	if vm.Count() != 2 {
		t.Fatalf("unexpected count after delete: %d", vm.Count())
	}
}
