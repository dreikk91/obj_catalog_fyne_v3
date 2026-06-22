//go:build qt

package qtui

import (
	"fmt"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type AlarmProcessInput struct {
	CauseCode string
	Note      string
}

func ShowAlarmProcessDialog(parent *qt.QWidget, alarm models.Alarm, options []contracts.AlarmProcessingOption) (AlarmProcessInput, bool) {
	return ShowAlarmProcessDialogForAlarms(parent, []models.Alarm{alarm}, options)
}

func ShowAlarmProcessDialogForAlarms(parent *qt.QWidget, alarms []models.Alarm, options []contracts.AlarmProcessingOption) (AlarmProcessInput, bool) {
	if len(alarms) == 0 {
		return AlarmProcessInput{}, false
	}

	dialog := qt.NewQDialog(parent)
	if len(alarms) == 1 {
		dialog.SetWindowTitle("Відпрацювання тривоги")
		dialog.Resize(520, 360)
	} else {
		dialog.SetWindowTitle("Групове відпрацювання тривог")
		dialog.Resize(620, 460)
	}

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)

	if len(alarms) == 1 {
		alarm := alarms[0]
		form.AddRow3("Об'єкт", qt.NewQLabel3(fmt.Sprintf("№%s %s", alarm.GetObjectNumberDisplay(), strings.TrimSpace(alarm.ObjectName))).QWidget)
		form.AddRow3("Тривога", qt.NewQLabel3(alarm.GetTypeDisplay()).QWidget)
		form.AddRow3("Час", qt.NewQLabel3(alarm.GetTimeDisplay()).QWidget)
	} else {
		countLabel := qt.NewQLabel3(fmt.Sprintf("Буде відпрацьовано: %d", len(alarms)))
		countLabel.SetStyleSheet("font-weight: 600;")
		form.AddRow3("Тривоги", countLabel.QWidget)

		summary := qt.NewQTextEdit3(alarmProcessSummary(alarms))
		summary.SetReadOnly(true)
		summary.SetMinimumHeight(140)
		form.AddRow3("Список", summary.QWidget)
	}

	cause := qt.NewQComboBox2()
	validOptions := normalizeDialogAlarmOptions(options)
	for _, option := range validOptions {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = strings.TrimSpace(option.Code)
		}
		cause.AddItem(label)
	}
	if cause.CurrentIndex() < 0 {
		cause.AddItem("Відпрацювати")
	}
	form.AddRow3("Причина", cause.QWidget)

	note := qt.NewQTextEdit2()
	note.SetMinimumHeight(96)
	form.AddRow3("Примітка", note.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)

	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return AlarmProcessInput{}, false
	}

	input := AlarmProcessInput{
		Note: strings.TrimSpace(note.ToPlainText()),
	}
	index := cause.CurrentIndex()
	if index >= 0 && index < len(validOptions) {
		input.CauseCode = strings.TrimSpace(validOptions[index].Code)
	}
	return input, true
}

func normalizeDialogAlarmOptions(options []contracts.AlarmProcessingOption) []contracts.AlarmProcessingOption {
	result := make([]contracts.AlarmProcessingOption, 0, len(options))
	for _, option := range options {
		code := strings.TrimSpace(option.Code)
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = code
		}
		if label == "" {
			continue
		}
		result = append(result, contracts.AlarmProcessingOption{
			Code:  code,
			Label: label,
		})
	}
	return result
}

func alarmProcessSummary(alarms []models.Alarm) string {
	lines := make([]string, 0, len(alarms))
	for _, alarm := range alarms {
		object := strings.TrimSpace(fmt.Sprintf("№%s %s", alarm.GetObjectNumberDisplay(), strings.TrimSpace(alarm.ObjectName)))
		if object == "№" {
			object = fmt.Sprintf("ID %d", alarm.ObjectID)
		}
		line := fmt.Sprintf("%s | %s | %s", object, alarm.GetTypeDisplay(), alarm.GetTimeDisplay())
		if zone := alarmZoneSummary(alarm); zone != "" {
			line += " | " + zone
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func alarmZoneSummary(alarm models.Alarm) string {
	parts := make([]string, 0, 2)
	if alarm.ZoneNumber > 0 {
		parts = append(parts, fmt.Sprintf("зона %d", alarm.ZoneNumber))
	}
	if zoneName := strings.TrimSpace(alarm.ZoneName); zoneName != "" {
		parts = append(parts, zoneName)
	}
	return strings.Join(parts, " ")
}
