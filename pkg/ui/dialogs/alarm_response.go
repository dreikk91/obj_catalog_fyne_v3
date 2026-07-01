package dialogs

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

// AlarmResponseAction identifies an action selected in the alarm response card.
type AlarmResponseAction int

const (
	AlarmResponseNone AlarmResponseAction = iota
	AlarmResponseTake
	AlarmResponseProcess
	AlarmResponseAssign
	AlarmResponseArrived
	AlarmResponseCancel
)

// ShowAlarmResponseDialog displays ownership and response-group actions for an alarm.
func ShowAlarmResponseDialog(
	parent fyne.Window,
	alarm models.Alarm,
	groups []contracts.FrontendResponseGroup,
	onAction func(AlarmResponseAction, string),
) {
	if parent == nil {
		return
	}

	operatorLabel := widget.NewLabel(alarmOperatorStateText(alarm))
	operatorLabel.Wrapping = fyne.TextWrapWord
	responseLabel := widget.NewLabel(alarmResponseStateText(alarm))
	responseLabel.Wrapping = fyne.TextWrapWord

	groupIDs := make(map[string]string)
	groupLabels := make([]string, 0, len(groups))
	currentGroupID := strings.TrimSpace(alarm.ResponseGroupID)
	for _, group := range groups {
		if !responseGroupSelectable(group, currentGroupID) {
			continue
		}
		label := responseGroupLabelText(group)
		groupLabels = append(groupLabels, label)
		groupIDs[label] = strings.TrimSpace(group.ID)
	}
	groupSelect := widget.NewSelect(groupLabels, nil)
	groupSelect.PlaceHolder = "Оберіть ГМР"
	if len(groupLabels) > 0 {
		groupSelect.SetSelected(groupLabels[0])
	} else {
		groupSelect.Disable()
	}

	var dlg dialog.Dialog
	run := func(action AlarmResponseAction, groupID string) {
		if dlg != nil {
			dlg.Hide()
		}
		if onAction != nil {
			onAction(action, strings.TrimSpace(groupID))
		}
	}

	takeButton := widget.NewButton(alarmTakeButtonText(alarm), func() {
		run(AlarmResponseTake, "")
	})
	if alarm.IsInProgress && !alarm.CanTakeOver {
		takeButton.Disable()
	}

	assignButton := widget.NewButton("Призначити ГМР", func() {
		run(AlarmResponseAssign, groupIDs[groupSelect.Selected])
	})
	if len(groupLabels) == 0 || alarm.IsResponseGroupDispatched {
		assignButton.Disable()
	}

	arrivedButton := widget.NewButton("ГМР прибула", func() {
		run(AlarmResponseArrived, "")
	})
	if !alarm.IsResponseGroupDispatched || alarm.IsResponseGroupArrived {
		arrivedButton.Disable()
	}

	cancelButton := widget.NewButton("Зняти ГМР", func() {
		run(AlarmResponseCancel, "")
	})
	if !alarm.IsResponseGroupDispatched {
		cancelButton.Disable()
	}

	processButton := widget.NewButton("Причина / завершити тривогу", func() {
		run(AlarmResponseProcess, "")
	})
	if !alarm.CanProcess {
		processButton.Disable()
	}

	content := container.NewVBox(
		widget.NewCard("Тривога", "", buildAlarmSummaryContent(alarm)),
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewLabel("Оператор"),
			operatorLabel,
			widget.NewLabel("Реагування"),
			responseLabel,
		),
		widget.NewSeparator(),
		widget.NewLabel("Група реагування"),
		groupSelect,
		container.NewGridWithColumns(3, takeButton, assignButton, arrivedButton),
		container.NewHBox(cancelButton, layout.NewSpacer(), processButton),
	)

	dlg = dialog.NewCustom("Картка реагування", "Закрити", content, parent)
	dlg.Resize(fyne.NewSize(720, 480))
	dlg.Show()
}

func alarmOperatorStateText(alarm models.Alarm) string {
	if !alarm.IsInProgress {
		return "Не взята в роботу"
	}
	operator := strings.TrimSpace(alarm.InProgressBy)
	switch {
	case alarm.IsOwnedByMe && operator != "":
		return "У роботі у вас: " + operator
	case alarm.IsOwnedByMe:
		return "У роботі у вас"
	case operator != "":
		return "У роботі: " + operator
	default:
		return "У роботі в іншого оператора"
	}
}

func alarmTakeButtonText(alarm models.Alarm) string {
	switch {
	case alarm.IsInProgress && !alarm.IsOwnedByMe && alarm.CanTakeOver:
		return "Перехопити тривогу"
	case alarm.IsOwnedByMe:
		return "Тривога вже у вас"
	default:
		return "Взяти в роботу"
	}
}

func alarmResponseStateText(alarm models.Alarm) string {
	switch {
	case alarm.IsResponseGroupArrived:
		return "ГМР прибула" + responseGroupIDSuffix(alarm.ResponseGroupID)
	case alarm.IsResponseGroupDispatched:
		return "ГМР направлена" + responseGroupIDSuffix(alarm.ResponseGroupID)
	default:
		return "ГМР не призначена"
	}
}

func responseGroupIDSuffix(groupID string) string {
	if groupID = strings.TrimSpace(groupID); groupID != "" {
		return " (" + groupID + ")"
	}
	return ""
}

func responseGroupSelectable(group contracts.FrontendResponseGroup, currentID string) bool {
	if strings.TrimSpace(group.ID) == currentID && currentID != "" {
		return true
	}
	return group.Status == "" ||
		group.Status == contracts.ResponseGroupStatusUnknown ||
		group.Status == contracts.ResponseGroupStatusFree
}

func responseGroupLabelText(group contracts.FrontendResponseGroup) string {
	name := strings.TrimSpace(group.Name)
	if name == "" {
		name = "ГМР " + strings.TrimSpace(group.ID)
	}
	parts := []string{name}
	if callsign := strings.TrimSpace(group.Callsign); callsign != "" {
		parts = append(parts, "позивний "+callsign)
	}
	if phone := strings.TrimSpace(group.Phone); phone != "" {
		parts = append(parts, phone)
	}
	if status := strings.TrimSpace(group.StatusText); status != "" {
		parts = append(parts, status)
	}
	if id := strings.TrimSpace(group.ID); id != "" {
		parts = append(parts, fmt.Sprintf("ID %s", id))
	}
	return strings.Join(parts, " | ")
}
