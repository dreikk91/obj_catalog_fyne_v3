package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardPersonalsFlowViewModel_SelectAndPrepareEdit(t *testing.T) {
	state := NewObjectWizardPersonalsStateViewModel()
	vm := NewObjectWizardPersonalsFlowViewModel(NewObjectWizardPersonalsTableViewModel())

	vm.SelectTableRow(state, 0)
	prompt := vm.PrepareEdit(state)
	if prompt.CanEdit {
		t.Fatalf("must not allow edit without selection")
	}
	if prompt.StatusText != "Виберіть В/О у таблиці" {
		t.Fatalf("unexpected status: %q", prompt.StatusText)
	}

	vm.ApplyAdd(state, contracts.AdminObjectPersonal{Number: 1, Name: "Іван"})
	vm.SelectTableRow(state, 1)
	prompt = vm.PrepareEdit(state)
	if !prompt.CanEdit {
		t.Fatalf("expected edit prompt")
	}
	if prompt.SelectedIdx != 0 {
		t.Fatalf("unexpected selected idx: %d", prompt.SelectedIdx)
	}
}

func TestObjectWizardPersonalsFlowViewModel_AddUpdateDelete(t *testing.T) {
	state := NewObjectWizardPersonalsStateViewModel()
	vm := NewObjectWizardPersonalsFlowViewModel(NewObjectWizardPersonalsTableViewModel())

	add := vm.ApplyAdd(state, contracts.AdminObjectPersonal{Number: 1, Name: "Іван"})
	if add.StatusText != "Додано В/О. Всього: 1" {
		t.Fatalf("unexpected add status: %q", add.StatusText)
	}
	if !add.RefreshTable {
		t.Fatalf("add must request table refresh")
	}

	editPrompt := vm.PrepareEdit(state)
	if !editPrompt.CanEdit {
		t.Fatalf("expected edit prompt")
	}
	updated := editPrompt.Initial
	updated.Name = "Петро"
	upd := vm.ApplyUpdate(state, editPrompt.SelectedIdx, updated)
	if upd.StatusText != "В/О оновлено" {
		t.Fatalf("unexpected update status: %q", upd.StatusText)
	}
	if !upd.RefreshTable {
		t.Fatalf("update must request table refresh")
	}

	delPrompt := vm.PrepareDelete(state)
	if !delPrompt.CanDelete {
		t.Fatalf("expected delete prompt")
	}
	if delPrompt.ConfirmText == "" {
		t.Fatalf("confirm text must be populated")
	}
	del := vm.ApplyDelete(state, delPrompt.SelectedIdx)
	if del.StatusText != "В/О видалено. Залишилось: 0" {
		t.Fatalf("unexpected delete status: %q", del.StatusText)
	}
	if !del.RefreshTable {
		t.Fatalf("delete must request table refresh")
	}
}

func TestObjectWizardPersonalsFlowViewModel_NextNumber(t *testing.T) {
	state := NewObjectWizardPersonalsStateViewModel()
	vm := NewObjectWizardPersonalsFlowViewModel(NewObjectWizardPersonalsTableViewModel())

	if got := vm.NextNumber(state); got != 1 {
		t.Fatalf("unexpected next number: %d", got)
	}

	vm.ApplyAdd(state, contracts.AdminObjectPersonal{Number: 7})
	if got := vm.NextNumber(state); got != 8 {
		t.Fatalf("unexpected next number after add: %d", got)
	}
}
