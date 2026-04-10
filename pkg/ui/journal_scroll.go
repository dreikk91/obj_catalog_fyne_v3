package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	journalListMinWidth         float32 = 320
	journalListHorizontalGutter float32 = 40
)

func newHorizontalJournalScroll(list *widget.List) (*container.Scroll, *canvas.Rectangle) {
	guide := canvas.NewRectangle(color.Transparent)
	guide.SetMinSize(fyne.NewSize(journalListMinWidth, 1))
	return container.NewHScroll(container.NewStack(list, guide)), guide
}

func ensureJournalListMinWidth(guide *canvas.Rectangle, texts []string, fontSize float32, style fyne.TextStyle) {
	if guide == nil {
		return
	}
	if fontSize <= 0 {
		fontSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	}

	maxWidth := journalListMinWidth
	for _, text := range texts {
		if width := measureJournalTextWidth(text, fontSize, style) + journalListHorizontalGutter; width > maxWidth {
			maxWidth = width
		}
	}

	guide.SetMinSize(fyne.NewSize(maxWidth, 1))
	guide.Refresh()
}

func measureJournalTextWidth(text string, fontSize float32, style fyne.TextStyle) float32 {
	sample := canvas.NewText(text, color.White)
	sample.TextSize = fontSize
	sample.TextStyle = style
	return sample.MinSize().Width
}
