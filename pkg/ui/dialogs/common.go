package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func ShowInfoDialog(win fyne.Window, title, msg string) {
	dialog.ShowInformation(title, msg, win)
}

func ShowErrorDialog(win fyne.Window, title string, err error) {
	dialog.ShowError(err, win)
}
