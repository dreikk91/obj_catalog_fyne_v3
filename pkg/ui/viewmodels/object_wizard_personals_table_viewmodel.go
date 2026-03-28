package viewmodels

import (
	"fmt"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
)

// ObjectWizardPersonalsTableViewModel формує тексти таблиці та статуси кроку "В/О" в майстрі.
type ObjectWizardPersonalsTableViewModel struct{}

func NewObjectWizardPersonalsTableViewModel() *ObjectWizardPersonalsTableViewModel {
	return &ObjectWizardPersonalsTableViewModel{}
}

func (vm *ObjectWizardPersonalsTableViewModel) HeaderText(col int) string {
	switch col {
	case 0:
		return "№"
	case 1:
		return "ПІБ"
	case 2:
		return "Телефон"
	case 3:
		return "Посада"
	case 4:
		return "Доступ"
	default:
		return "Примітка"
	}
}

func (vm *ObjectWizardPersonalsTableViewModel) CellText(item contracts.AdminObjectPersonal, fullName string, col int) string {
	switch col {
	case 0:
		return strconv.FormatInt(item.Number, 10)
	case 1:
		return strings.TrimSpace(fullName)
	case 2:
		return strings.TrimSpace(item.Phones)
	case 3:
		return strings.TrimSpace(item.Position)
	case 4:
		if item.Access1 > 0 {
			return "Адмін"
		}
		return "Оператор"
	default:
		return strings.TrimSpace(item.Notes)
	}
}

func (vm *ObjectWizardPersonalsTableViewModel) StatusAdded(total int) string {
	return fmt.Sprintf("Додано В/О. Всього: %d", total)
}

func (vm *ObjectWizardPersonalsTableViewModel) StatusUpdated() string {
	return "В/О оновлено"
}

func (vm *ObjectWizardPersonalsTableViewModel) StatusSelectionRequired() string {
	return "Виберіть В/О у таблиці"
}

func (vm *ObjectWizardPersonalsTableViewModel) DeleteConfirmText(fullName string) string {
	return fmt.Sprintf("Видалити В/О \"%s\"?", strings.TrimSpace(fullName))
}

func (vm *ObjectWizardPersonalsTableViewModel) StatusDeleted(remaining int) string {
	return fmt.Sprintf("В/О видалено. Залишилось: %d", remaining)
}
