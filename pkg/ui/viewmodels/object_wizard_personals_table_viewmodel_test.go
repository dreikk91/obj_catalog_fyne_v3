package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectWizardPersonalsTableViewModel_HeaderText(t *testing.T) {
	vm := NewObjectWizardPersonalsTableViewModel()

	want := []string{"№", "ПІБ", "Телефон", "Посада", "Доступ", "Примітка"}
	for col, expected := range want {
		if got := vm.HeaderText(col); got != expected {
			t.Fatalf("col %d: expected %q, got %q", col, expected, got)
		}
	}
}

func TestObjectWizardPersonalsTableViewModel_CellText(t *testing.T) {
	vm := NewObjectWizardPersonalsTableViewModel()
	item := contracts.AdminObjectPersonal{
		Number:   7,
		Phones:   " 0501234567 ",
		Position: " Інженер ",
		Access1:  1,
		Notes:    " Примітка ",
	}

	if got := vm.CellText(item, " Іванов Іван ", 0); got != "7" {
		t.Fatalf("unexpected number cell: %q", got)
	}
	if got := vm.CellText(item, " Іванов Іван ", 1); got != "Іванов Іван" {
		t.Fatalf("unexpected full name cell: %q", got)
	}
	if got := vm.CellText(item, " Іванов Іван ", 2); got != "0501234567" {
		t.Fatalf("unexpected phones cell: %q", got)
	}
	if got := vm.CellText(item, " Іванов Іван ", 3); got != "Інженер" {
		t.Fatalf("unexpected position cell: %q", got)
	}
	if got := vm.CellText(item, " Іванов Іван ", 4); got != "Адмін" {
		t.Fatalf("unexpected access cell: %q", got)
	}
	if got := vm.CellText(item, " Іванов Іван ", 5); got != "Примітка" {
		t.Fatalf("unexpected notes cell: %q", got)
	}
}

func TestObjectWizardPersonalsTableViewModel_StatusTexts(t *testing.T) {
	vm := NewObjectWizardPersonalsTableViewModel()

	if got := vm.StatusAdded(3); got != "Додано В/О. Всього: 3" {
		t.Fatalf("unexpected added status: %q", got)
	}
	if got := vm.StatusUpdated(); got != "В/О оновлено" {
		t.Fatalf("unexpected updated status: %q", got)
	}
	if got := vm.StatusSelectionRequired(); got != "Виберіть В/О у таблиці" {
		t.Fatalf("unexpected selection status: %q", got)
	}
	if got := vm.DeleteConfirmText(" Петров П.П. "); got != "Видалити В/О \"Петров П.П.\"?" {
		t.Fatalf("unexpected delete confirm text: %q", got)
	}
	if got := vm.StatusDeleted(1); got != "В/О видалено. Залишилось: 1" {
		t.Fatalf("unexpected deleted status: %q", got)
	}
}
