package dialogs

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/data"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ShowTestMessagesDialog відкриває вікно з тестовими повідомленнями об'єкта
func ShowTestMessagesDialog(parent fyne.Window, provider data.DataProvider, objectID string) {
	win := fyne.CurrentApp().NewWindow("Тестові повідомлення: " + objectID)
	win.Resize(fyne.NewSize(700, 400))

	loading := widget.NewProgressBarInfinite()
	content := container.NewStack(loading)
	win.SetContent(content)
	win.Show()

	go func() {
		messages := provider.GetTestMessages(objectID)

		fyne.Do(func() {
			if len(messages) == 0 {
				win.SetContent(container.NewCenter(widget.NewLabel("Тестових повідомлень не знайдено або помилка доступу")))
				return
			}

			table := widget.NewTable(
				func() (int, int) {
					return len(messages), 3
				},
				func() fyne.CanvasObject {
					return widget.NewLabel("Cell")
				},
				func(id widget.TableCellID, obj fyne.CanvasObject) {
					label := obj.(*widget.Label)
					msg := messages[id.Row]
					switch id.Col {
					case 0:
						label.SetText(msg.Time.Format("02.01.2006 15:04:05"))
					case 1:
						label.SetText(msg.Info)
					case 2:
						label.SetText(msg.Details)
					}
				},
			)

			table.SetColumnWidth(0, 150)
			table.SetColumnWidth(1, 250)
			table.SetColumnWidth(2, 280)

			win.SetContent(container.NewBorder(
				widget.NewLabel(fmt.Sprintf("Останні тестові повідомлення для об'єкта №%s", objectID)),
				widget.NewButton("Закрити", func() { win.Close() }),
				nil, nil,
				table,
			))
		})
	}()
}
