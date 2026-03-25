package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func ShowAdminPlaceholderDialog(win fyne.Window, title string, message string) {
	content := container.NewVBox(
		widget.NewLabel(message),
		widget.NewSeparator(),
		widget.NewLabel("Цей пункт уже винесено в адмін-меню та буде реалізований наступним кроком."),
	)

	d := dialog.NewCustom(title, "Закрити", content, win)
	d.Resize(fyne.NewSize(620, 220))
	d.Show()
}
