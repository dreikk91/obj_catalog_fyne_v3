//go:build qt

package qtui

import (
	"strconv"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func (panel *WorkAreaPanel) showCurrentObjectTestMessages() {
	if panel == nil || panel.currentObject == nil || panel.dataProvider == nil {
		return
	}
	showTestMessagesDialog(
		panel.QWidget,
		panel.dataProvider,
		strconv.Itoa(panel.currentObject.ID),
	)
}

func showTestMessagesDialog(parent *qt.QWidget, provider contracts.TestMessageProvider, objectID string) {
	messages := provider.GetTestMessages(objectID)

	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Тестові повідомлення: " + objectID)
	dialog.Resize(820, 480)
	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(qt.NewQLabel3("Останні тестові повідомлення для об'єкта №" + objectID).QWidget)

	model := qt.NewQStandardItemModel2(0, 3)
	headers := []string{"Дата і час", "Повідомлення", "Деталі"}
	model.SetHorizontalHeaderLabels(headers)
	for _, message := range messages {
		addReadOnlyRow(model, []string{
			message.Time.Format("02.01.2006 15:04:05"),
			message.Info,
			message.Details,
		})
	}
	table := newTable(model, headers)
	table.SetWordWrap(false)
	table.ResizeColumnsToContents()
	table.HorizontalHeader().SetStretchLastSection(true)
	layout.AddWidget(table.QWidget)

	if len(messages) == 0 {
		layout.AddWidget(qt.NewQLabel3("Тестових повідомлень не знайдено або джерело недоступне.").QWidget)
	}

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	buttons.OnRejected(func() { dialog.Reject() })
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}
