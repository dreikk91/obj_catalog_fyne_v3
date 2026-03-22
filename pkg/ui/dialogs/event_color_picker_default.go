//go:build !windows

package dialogs

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
)

func showEventColorPicker(
	win fyne.Window,
	title string,
	description string,
	current color.NRGBA,
	onPicked func(color.NRGBA),
) {
	picker := dialog.NewColorPicker(
		title,
		description,
		func(c color.Color) {
			if c == nil || onPicked == nil {
				return
			}
			onPicked(color.NRGBAModel.Convert(c).(color.NRGBA))
		},
		win,
	)
	picker.Advanced = true
	picker.SetColor(current)
	picker.Show()
}
