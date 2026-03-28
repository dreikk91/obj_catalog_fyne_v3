package viewmodels

import (
	"fmt"
	"strconv"
	"strings"
)

// ObjectWizardZonesStepViewModel формує тексти таблиці та статуси кроку "Зони" в майстрі.
type ObjectWizardZonesStepViewModel struct{}

func NewObjectWizardZonesStepViewModel() *ObjectWizardZonesStepViewModel {
	return &ObjectWizardZonesStepViewModel{}
}

func (vm *ObjectWizardZonesStepViewModel) HeaderText(col int) string {
	switch col {
	case 0:
		return "ZONEN"
	case 1:
		return "Тип"
	default:
		return "Опис"
	}
}

func (vm *ObjectWizardZonesStepViewModel) CellText(zoneNumber int64, description string, col int) string {
	switch col {
	case 0:
		return strconv.FormatInt(zoneNumber, 10)
	case 1:
		return "пож."
	default:
		return strings.TrimSpace(description)
	}
}

func (vm *ObjectWizardZonesStepViewModel) StatusAddFirstFailed() string {
	return "Не вдалося додати першу зону"
}

func (vm *ObjectWizardZonesStepViewModel) StatusFirstAdded() string {
	return "Додано зону #1"
}

func (vm *ObjectWizardZonesStepViewModel) StatusSelectionRequired() string {
	return "Виберіть зону у таблиці"
}

func (vm *ObjectWizardZonesStepViewModel) StatusAddNextFailed() string {
	return "Не вдалося додати наступну зону"
}

func (vm *ObjectWizardZonesStepViewModel) StatusSavedAndMoved(currentZone int64, nextZone int64) string {
	return fmt.Sprintf("Збережено зону #%d, перехід на #%d", currentZone, nextZone)
}

func (vm *ObjectWizardZonesStepViewModel) StatusAddFailed() string {
	return "Не вдалося додати зону"
}

func (vm *ObjectWizardZonesStepViewModel) StatusReadyForInput(nextZone int64) string {
	return fmt.Sprintf("Готово до введення зони #%d", nextZone)
}

func (vm *ObjectWizardZonesStepViewModel) StatusCreateFirstFailed() string {
	return "Не вдалося створити першу зону"
}

func (vm *ObjectWizardZonesStepViewModel) StatusCreatedFirst() string {
	return "Створено зону #1, можна вводити назву"
}

func (vm *ObjectWizardZonesStepViewModel) StatusSelectAndInput() string {
	return "Виберіть зону і вводьте назву"
}

func (vm *ObjectWizardZonesStepViewModel) StatusEditingPrompt(zoneNumber int64) string {
	return fmt.Sprintf("Редагування зони #%d: введіть назву і натисніть Enter", zoneNumber)
}

func (vm *ObjectWizardZonesStepViewModel) DeleteConfirmText(zoneNumber int64) string {
	return fmt.Sprintf("Видалити зону #%d?", zoneNumber)
}

func (vm *ObjectWizardZonesStepViewModel) StatusDeleted(zoneNumber int64) string {
	return fmt.Sprintf("Зону #%d видалено", zoneNumber)
}

func (vm *ObjectWizardZonesStepViewModel) StatusFillFailed() string {
	return "Не вдалося заповнити зони"
}

func (vm *ObjectWizardZonesStepViewModel) StatusFilledTo(count int64) string {
	return fmt.Sprintf("Зони заповнено до #%d", count)
}

func (vm *ObjectWizardZonesStepViewModel) ClearConfirmText() string {
	return "Видалити всі зони, додані в майстрі?"
}

func (vm *ObjectWizardZonesStepViewModel) StatusCleared() string {
	return "Зони очищено"
}

func (vm *ObjectWizardZonesStepViewModel) StatusCount(count int) string {
	return fmt.Sprintf("Зони: %d запис(ів)", count)
}
