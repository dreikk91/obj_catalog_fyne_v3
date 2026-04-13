// Package dialogs містить модальні вікна додатку.
// Цей файл: діалог відпрацювання тривоги.
package dialogs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

var legacyAlarmProcessActions = []string{
	"Помилкова тривога",
	"Виклик пожежників",
	"Виклик ГШР",
	"Технічна несправність",
	"Контрольна перевірка",
	"Інше",
}

// ShowProcessAlarmDialog показує діалог відпрацювання тривоги.
// CASL-модальник використовується тільки для CASL-тривог.
// Для інших джерел лишається простий legacy-діалог.
func ShowProcessAlarmDialog(parent fyne.Window, alarm models.Alarm, provider contracts.AlarmProvider, user string, onSuccess func()) {
	if parent == nil || provider == nil {
		return
	}
	if !ids.IsCASLObjectID(alarm.ObjectID) {
		showLegacyProcessAlarmDialog(parent, alarm, provider, user, onSuccess)
		return
	}

	noteEntry := widget.NewMultiLineEntry()
	noteEntry.SetPlaceHolder("Коментар диспетчера")
	noteEntry.SetMinRowsVisible(4)

	statusLabel := widget.NewLabel("")
	statusLabel.Wrapping = fyne.TextWrapWord

	loading := widget.NewProgressBarInfinite()
	loading.Hide()

	reasonSelect := widget.NewSelect(nil, nil)
	reasonSelect.PlaceHolder = "Оберіть причину відпрацювання"
	reasonSelect.Disable()
	reasonSelect.Hide()

	labelToCode := make(map[string]string)
	advanced, hasAdvanced := provider.(contracts.AlarmProcessingProvider)
	if !hasAdvanced {
		showLegacyProcessAlarmDialog(parent, alarm, provider, user, onSuccess)
		return
	}
	inProgress := false
	optionsReady := false

	var dlg dialog.Dialog
	var submitBtn *widget.Button

	setBusy := func(busy bool) {
		inProgress = busy
		if busy {
			noteEntry.Disable()
			reasonSelect.Disable()
			loading.Show()
		} else {
			noteEntry.Enable()
			loading.Hide()
			if optionsReady {
				reasonSelect.Enable()
			} else {
				reasonSelect.Disable()
			}
		}
		if submitBtn != nil {
			if busy {
				submitBtn.Disable()
				return
			}
			if optionsReady && strings.TrimSpace(reasonSelect.Selected) != "" {
				submitBtn.Enable()
			} else {
				submitBtn.Disable()
			}
		}
	}

	cancelBtn := widget.NewButton("Скасувати", func() {
		if inProgress {
			return
		}
		if dlg != nil {
			dlg.Hide()
		}
	})

	submitBtn = widget.NewButton("Відпрацювати", func() {
		if inProgress {
			return
		}

		request := contracts.AlarmProcessingRequest{
			Note: strings.TrimSpace(noteEntry.Text),
		}
		request.CauseCode = strings.TrimSpace(labelToCode[reasonSelect.Selected])
		if request.CauseCode == "" {
			statusLabel.SetText("Не вибрано причину відпрацювання.")
			setBusy(false)
			return
		}

		statusLabel.SetText("Виконується відпрацювання тривоги...")
		setBusy(true)

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			err := advanced.ProcessAlarmWithRequest(ctx, alarm, user, request)

			fyne.Do(func() {
				if err != nil {
					statusLabel.SetText("Помилка відпрацювання тривоги.")
					setBusy(false)
					dialog.ShowError(err, parent)
					return
				}
				if dlg != nil {
					dlg.Hide()
				}
				if onSuccess != nil {
					onSuccess()
				}
			})
		}()
	})

	summary := widget.NewCard("Тривога", "", buildAlarmSummaryContent(alarm))
	formItems := []fyne.CanvasObject{
		summary,
		widget.NewSeparator(),
	}
	formItems = append(formItems,
		widget.NewLabel("Причина відпрацювання"),
		reasonSelect,
	)
	formItems = append(formItems,
		widget.NewLabel("Коментар"),
		noteEntry,
		loading,
		statusLabel,
		container.NewHBox(layout.NewSpacer(), cancelBtn, submitBtn),
	)

	content := container.NewVBox(formItems...)
	dlg = dialog.NewCustom("Відпрацювання тривоги", "Закрити", content, parent)
	dlg.Resize(fyne.NewSize(520, 430))

	statusLabel.SetText("Завантаження причин відпрацювання...")
	loading.Show()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		options, err := advanced.GetAlarmProcessingOptions(ctx, alarm)
		fyne.Do(func() {
			if err != nil {
				statusLabel.SetText("Не вдалося завантажити причини відпрацювання.")
				dialog.ShowError(err, parent)
				setBusy(false)
				return
			}

			labelToCode = make(map[string]string, len(options))
			labels := make([]string, 0, len(options))
			usedLabels := make(map[string]struct{}, len(options))
			for _, option := range options {
				code := strings.TrimSpace(option.Code)
				if code == "" {
					continue
				}
				label := strings.TrimSpace(option.Label)
				if label == "" {
					label = code
				}
				if _, exists := usedLabels[label]; exists {
					label = fmt.Sprintf("%s (%s)", label, code)
				}
				usedLabels[label] = struct{}{}
				labelToCode[label] = code
				labels = append(labels, label)
			}

			reasonSelect.Options = labels
			if len(labels) > 0 {
				reasonSelect.SetSelected(labels[0])
				statusLabel.SetText("")
			} else {
				statusLabel.SetText("CASL не повернув жодної причини відпрацювання.")
			}
			reasonSelect.Show()
			loading.Hide()
			optionsReady = true
			setBusy(false)
		})
	}()

	dlg.Show()
}

func showLegacyProcessAlarmDialog(parent fyne.Window, alarm models.Alarm, provider contracts.AlarmProvider, user string, onSuccess func()) {
	actionSelect := widget.NewSelect(legacyAlarmProcessActions, nil)
	actionSelect.SetSelected(legacyAlarmProcessActions[0])

	noteEntry := widget.NewMultiLineEntry()
	noteEntry.SetPlaceHolder("Введіть примітку...")
	noteEntry.SetMinRowsVisible(3)

	form := container.NewVBox(
		widget.NewLabel("Інформація про тривогу:"),
		widget.NewSeparator(),
		buildAlarmSummaryContent(alarm),
		widget.NewSeparator(),
		widget.NewLabel("Результат обробки:"),
		actionSelect,
		widget.NewLabel("Примітка:"),
		noteEntry,
	)

	dlg := dialog.NewCustomConfirm(
		"Обробка тривоги",
		"Підтвердити",
		"Скасувати",
		form,
		func(confirmed bool) {
			if !confirmed {
				return
			}
			provider.ProcessAlarm(fmt.Sprintf("%d", alarm.ID), user, strings.TrimSpace(noteEntry.Text))
			if onSuccess != nil {
				onSuccess()
			}
		},
		parent,
	)
	dlg.Resize(fyne.NewSize(400, 350))
	dlg.Show()
}

func buildAlarmSummaryContent(alarm models.Alarm) fyne.CanvasObject {
	lines := []string{
		fmt.Sprintf("Об'єкт: %s", formatAlarmObjectLabel(alarm)),
		fmt.Sprintf("Адреса: %s", fallbackAlarmField(alarm.Address)),
		fmt.Sprintf("Тип: %s", alarm.GetTypeDisplay()),
		fmt.Sprintf("Час: %s", alarm.GetDateTimeDisplay()),
	}
	if alarm.ZoneNumber > 0 {
		lines = append(lines, fmt.Sprintf("Зона: %d", alarm.ZoneNumber))
	}
	if zoneName := strings.TrimSpace(alarm.ZoneName); zoneName != "" {
		lines = append(lines, fmt.Sprintf("Назва зони: %s", zoneName))
	}
	if details := strings.TrimSpace(alarm.Details); details != "" {
		lines = append(lines, fmt.Sprintf("Деталі: %s", details))
	}
	return widget.NewLabel(strings.Join(lines, "\n"))
}

func formatAlarmObjectLabel(alarm models.Alarm) string {
	number := strings.TrimSpace(alarm.GetObjectNumberDisplay())
	name := strings.TrimSpace(alarm.ObjectName)
	switch {
	case number != "" && name != "":
		return fmt.Sprintf("№%s %s", number, name)
	case name != "":
		return name
	case number != "":
		return "№" + number
	default:
		return fmt.Sprintf("ID %d", alarm.ObjectID)
	}
}

func fallbackAlarmField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}
	return value
}
