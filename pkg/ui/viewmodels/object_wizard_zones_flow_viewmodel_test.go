package viewmodels

import "testing"

func TestObjectWizardZonesFlowViewModel_MoveToNext_FromEmpty(t *testing.T) {
	state := NewObjectWizardStateViewModel()
	vm := NewObjectWizardZonesFlowViewModel(NewObjectWizardZonesStepViewModel())

	out := vm.MoveToNext(state, "  Склад  ")
	if out.Err != nil {
		t.Fatalf("unexpected error: %v", out.Err)
	}
	if out.StatusText != "Додано зону #1" {
		t.Fatalf("unexpected status: %q", out.StatusText)
	}
	if !out.RefreshTable || !out.FocusQuickName || out.TargetZoneNumber != 1 {
		t.Fatalf("unexpected UI action: %+v", out)
	}
	if state.ZoneCount() != 1 {
		t.Fatalf("expected 1 zone, got %d", state.ZoneCount())
	}
}

func TestObjectWizardZonesFlowViewModel_MoveToNext_SavesAndCreatesNext(t *testing.T) {
	state := NewObjectWizardStateViewModel()
	if _, err := state.EnsureFirstZone(""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	vm := NewObjectWizardZonesFlowViewModel(NewObjectWizardZonesStepViewModel())

	out := vm.MoveToNext(state, "Пожежна")
	if out.Err != nil {
		t.Fatalf("unexpected error: %v", out.Err)
	}
	if out.StatusText != "Збережено зону #1, перехід на #2" {
		t.Fatalf("unexpected status: %q", out.StatusText)
	}
	if !out.RefreshTable || out.TargetZoneNumber != 2 {
		t.Fatalf("unexpected UI action: %+v", out)
	}
	if state.ZoneCount() != 2 {
		t.Fatalf("expected 2 zones, got %d", state.ZoneCount())
	}
}

func TestObjectWizardZonesFlowViewModel_PrepareDelete(t *testing.T) {
	state := NewObjectWizardStateViewModel()
	vm := NewObjectWizardZonesFlowViewModel(NewObjectWizardZonesStepViewModel())

	prompt := vm.PrepareDelete(state)
	if prompt.CanDelete {
		t.Fatalf("must not allow delete without selection")
	}
	if prompt.StatusText != "Виберіть зону у таблиці" {
		t.Fatalf("unexpected status: %q", prompt.StatusText)
	}

	if _, err := state.EnsureFirstZone(""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	prompt = vm.PrepareDelete(state)
	if !prompt.CanDelete {
		t.Fatalf("expected delete prompt")
	}
	if prompt.TargetZoneNumber != 1 {
		t.Fatalf("unexpected target zone: %d", prompt.TargetZoneNumber)
	}
	if prompt.ConfirmText != "Видалити зону #1?" {
		t.Fatalf("unexpected confirm text: %q", prompt.ConfirmText)
	}
}

func TestObjectWizardZonesFlowViewModel_FillAndClear(t *testing.T) {
	state := NewObjectWizardStateViewModel()
	vm := NewObjectWizardZonesFlowViewModel(NewObjectWizardZonesStepViewModel())

	fill := vm.Fill(state, 3)
	if fill.Err != nil {
		t.Fatalf("unexpected fill error: %v", fill.Err)
	}
	if fill.StatusText != "Зони заповнено до #3" {
		t.Fatalf("unexpected fill status: %q", fill.StatusText)
	}
	if state.ZoneCount() != 3 {
		t.Fatalf("expected 3 zones after fill, got %d", state.ZoneCount())
	}

	clear := vm.Clear(state)
	if clear.StatusText != "Зони очищено" {
		t.Fatalf("unexpected clear status: %q", clear.StatusText)
	}
	if !clear.RefreshTable {
		t.Fatalf("clear must request table refresh")
	}
	if state.ZoneCount() != 0 {
		t.Fatalf("expected empty zones after clear, got %d", state.ZoneCount())
	}
}

func TestObjectWizardZonesFlowViewModel_DefaultFillCount(t *testing.T) {
	state := NewObjectWizardStateViewModel()
	vm := NewObjectWizardZonesFlowViewModel(NewObjectWizardZonesStepViewModel())

	if got := vm.DefaultFillCount(state); got != 24 {
		t.Fatalf("unexpected default fill count: %d", got)
	}

	if err := state.EnsureZoneExists(30, ""); err != nil {
		t.Fatalf("unexpected ensure error: %v", err)
	}
	if got := vm.DefaultFillCount(state); got != 30 {
		t.Fatalf("unexpected fill count from state max: %d", got)
	}
}
