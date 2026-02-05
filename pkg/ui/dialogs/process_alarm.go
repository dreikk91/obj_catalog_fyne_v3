// Package dialogs містить модальні вікна додатку
// Цей файл: діалог обробки тривоги
package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
)

// ProcessAlarmResult містить результат обробки тривоги
type ProcessAlarmResult struct {
	Action string // Дія (виклик пожежників, помилкова, тощо)
	Note   string // Примітка диспетчера
}

// ShowProcessAlarmDialog показує діалог обробки тривоги
func ShowProcessAlarmDialog(parent fyne.Window, alarm models.Alarm, onConfirm func(result ProcessAlarmResult)) {
	// Варіанти дій
	actions := []string{
		"Виклик пожежників",
		"Виклик ГШР",
		"Помилкова тривога",
		"Технічна несправність",
		"Контрольна перевірка",
		"Інше",
	}

	// Вибір дії
	actionSelect := widget.NewSelect(actions, nil)
	actionSelect.SetSelected(actions[0])

	// Поле примітки
	noteEntry := widget.NewMultiLineEntry()
	noteEntry.SetPlaceHolder("Введіть примітку...")
	noteEntry.SetMinRowsVisible(3)

	// Інформація про тривогу
	infoLabel := widget.NewLabel(
		"Об'єкт: " + alarm.ObjectName + "\n" +
			"Адреса: " + alarm.Address + "\n" +
			"Тип: " + alarm.GetTypeDisplay() + "\n" +
			"Час: " + alarm.GetDateTimeDisplay(),
	)

	// Форма
	form := container.NewVBox(
		widget.NewLabel("Інформація про тривогу:"),
		widget.NewSeparator(),
		infoLabel,
		widget.NewSeparator(),
		widget.NewLabel("Результат обробки:"),
		actionSelect,
		widget.NewLabel("Примітка:"),
		noteEntry,
	)

	// Діалог
	d := dialog.NewCustomConfirm(
		"Обробка тривоги",
		"Підтвердити",
		"Скасувати",
		form,
		func(confirmed bool) {
			if confirmed && onConfirm != nil {
				result := ProcessAlarmResult{
					Action: actionSelect.Selected,
					Note:   noteEntry.Text,
				}
				onConfirm(result)
			}
		},
		parent,
	)

	// Встановлюємо розмір діалогу
	d.Resize(fyne.NewSize(400, 350))
	d.Show()
}
