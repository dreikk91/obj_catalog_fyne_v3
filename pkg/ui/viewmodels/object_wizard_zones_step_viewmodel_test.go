package viewmodels

import "testing"

func TestObjectWizardZonesStepViewModel_HeaderAndCellText(t *testing.T) {
	vm := NewObjectWizardZonesStepViewModel()

	if got := vm.HeaderText(0); got != "ZONEN" {
		t.Fatalf("unexpected header col0: %q", got)
	}
	if got := vm.HeaderText(1); got != "Тип" {
		t.Fatalf("unexpected header col1: %q", got)
	}
	if got := vm.HeaderText(2); got != "Опис" {
		t.Fatalf("unexpected header col2: %q", got)
	}
	if got := vm.CellText(12, "  Склад  ", 0); got != "12" {
		t.Fatalf("unexpected cell col0: %q", got)
	}
	if got := vm.CellText(12, "  Склад  ", 1); got != "пож." {
		t.Fatalf("unexpected cell col1: %q", got)
	}
	if got := vm.CellText(12, "  Склад  ", 2); got != "Склад" {
		t.Fatalf("unexpected cell col2: %q", got)
	}
}

func TestObjectWizardZonesStepViewModel_StatusMessages(t *testing.T) {
	vm := NewObjectWizardZonesStepViewModel()

	if got := vm.StatusAddFirstFailed(); got != "Не вдалося додати першу зону" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.StatusFirstAdded(); got != "Додано зону #1" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.StatusSelectionRequired(); got != "Виберіть зону у таблиці" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.StatusSavedAndMoved(3, 4); got != "Збережено зону #3, перехід на #4" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.StatusReadyForInput(7); got != "Готово до введення зони #7" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.StatusEditingPrompt(2); got != "Редагування зони #2: введіть назву і натисніть Enter" {
		t.Fatalf("unexpected status: %q", got)
	}
	if got := vm.DeleteConfirmText(9); got != "Видалити зону #9?" {
		t.Fatalf("unexpected confirm text: %q", got)
	}
	if got := vm.StatusDeleted(9); got != "Зону #9 видалено" {
		t.Fatalf("unexpected deleted status: %q", got)
	}
	if got := vm.StatusFilledTo(24); got != "Зони заповнено до #24" {
		t.Fatalf("unexpected fill status: %q", got)
	}
	if got := vm.ClearConfirmText(); got != "Видалити всі зони, додані в майстрі?" {
		t.Fatalf("unexpected clear confirm: %q", got)
	}
	if got := vm.StatusCount(5); got != "Зони: 5 запис(ів)" {
		t.Fatalf("unexpected count status: %q", got)
	}
}
