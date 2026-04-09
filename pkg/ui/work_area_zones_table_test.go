package ui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	fyneTheme "fyne.io/fyne/v2/theme"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestWrappedTextLineCount_WrapsLongZoneName(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	textSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	style := fyne.TextStyle{}
	width := fyne.MeasureText("Довга назва", textSize, style).Width
	text := strings.Repeat("Дуже довга назва зони ", 4)

	got := wrappedTextLineCount(text, width, textSize, style)
	if got < 2 {
		t.Fatalf("wrappedTextLineCount() = %d, want at least 2 lines", got)
	}
}

func TestZoneTableRowHeight_GrowsForWrappedContent(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	textSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	shortZone := models.Zone{
		Name:       "Вхід",
		SensorType: "PIR",
	}
	longZone := models.Zone{
		Name:       strings.Repeat("Дуже довга назва зони ", 5),
		SensorType: "Нетипізована тривожна зона",
	}

	shortHeight := zoneTableRowHeight(shortZone, 220, zoneTableTypeColumnWidth, textSize)
	longHeight := zoneTableRowHeight(longZone, 120, zoneTableTypeColumnWidth, textSize)
	if longHeight <= shortHeight {
		t.Fatalf("zoneTableRowHeight() long=%v must be greater than short=%v", longHeight, shortHeight)
	}
}
