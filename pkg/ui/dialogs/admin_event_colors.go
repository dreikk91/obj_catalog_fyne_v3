package dialogs

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/utils"
)

func isDialogsDarkMode() bool {
	if fyne.CurrentApp() == nil || fyne.CurrentApp().Settings() == nil {
		return false
	}
	_, ok := fyne.CurrentApp().Settings().Theme().(*appTheme.DarkTheme)
	return ok
}

func mapSC1ToPaletteCode(sc1 *int64) int {
	if sc1 == nil {
		return 6
	}
	switch *sc1 {
	case 1:
		return 1
	case 2, 3:
		return 2
	case 5, 9, 13, 17:
		return 5
	case 10:
		return 10
	case 11, 14, 18:
		return 14
	case 12:
		return 12
	default:
		return int(*sc1)
	}
}

func messageRowColors(sc1 *int64) (text color.NRGBA, row color.NRGBA) {
	code := mapSC1ToPaletteCode(sc1)
	if isDialogsDarkMode() {
		return utils.SelectColorNRGBADark(code)
	}
	return utils.SelectColorNRGBA(code)
}

func newColoredTableCell() fyne.CanvasObject {
	bg := canvas.NewRectangle(color.Transparent)
	txt := canvas.NewText("", color.Black)
	return container.NewStack(bg, container.NewPadded(txt))
}

func updateColoredMessageCell(obj fyne.CanvasObject, text string, sc1 *int64, selected bool) {
	stack := obj.(*fyne.Container)
	bg := stack.Objects[0].(*canvas.Rectangle)
	padded := stack.Objects[1].(*fyne.Container)
	txt := padded.Objects[0].(*canvas.Text)

	if selected {
		bg.FillColor = appTheme.ColorSelection
		txt.Color = color.White
	} else {
		textColor, rowColor := messageRowColors(sc1)
		bg.FillColor = rowColor
		txt.Color = textColor
	}

	txt.Text = text
	txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	bg.Show()
	bg.Refresh()
	txt.Refresh()
}
