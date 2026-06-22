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
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Відпрацювання тривоги")
	dialog.Resize(520, 360)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)

	form.AddRow3("Об'єкт", qt.NewQLabel3(fmt.Sprintf("№%s %s", alarm.GetObjectNumberDisplay(), strings.TrimSpace(alarm.ObjectName))).QWidget)
	form.AddRow3("Тривога", qt.NewQLabel3(alarm.GetTypeDisplay()).QWidget)
	form.AddRow3("Час", qt.NewQLabel3(alarm.GetTimeDisplay()).QWidget)

	cause := qt.NewQComboBox2()
	for _, option := range options {
		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = strings.TrimSpace(option.Code)
		}
		if label == "" {
			continue
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
	if index >= 0 && index < len(options) {
		input.CauseCode = strings.TrimSpace(options[index].Code)
	}
	return input, true
}
