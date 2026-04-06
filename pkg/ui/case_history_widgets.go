package ui

import (
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/utils"
)

func buildCaseHistoryEventList(group viewmodels.WorkAreaCaseHistoryGroup) fyne.CanvasObject {
	if len(group.Events) <= 1 {
		label := widget.NewLabel("Після початку тривоги додаткових подій поки немає.")
		label.Wrapping = fyne.TextWrapWord
		return container.NewPadded(label)
	}

	rows := make([]fyne.CanvasObject, 0, (len(group.Events)-1)*2)
	for idx, event := range group.Events[1:] {
		rows = append(rows, buildCaseHistoryEventLine(event))
		if idx < len(group.Events)-2 {
			rows = append(rows, widget.NewSeparator())
		}
	}

	return container.NewPadded(container.NewVBox(rows...))
}

func buildCaseHistoryEventLine(event models.Event) fyne.CanvasObject {
	textColor := caseHistoryEventTextColor(event.SC1)
	text := canvas.NewText(caseHistoryEventText(event), textColor)
	text.TextSize = fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	text.TextStyle = fyne.TextStyle{Bold: event.IsCritical()}

	return container.NewPadded(text)
}

func caseHistoryEventText(event models.Event) string {
	parts := []string{event.GetDateTimeDisplay()}
	if icon := getEventIcon(event.Type); icon != "" {
		parts = append(parts, icon)
	}
	parts = append(parts, strings.TrimSpace(event.GetTypeDisplay()))

	line := strings.Join(parts, " ")
	if event.ZoneNumber > 0 {
		line += " | Зона " + strconv.Itoa(event.ZoneNumber)
	}
	if user := strings.TrimSpace(event.UserName); user != "" {
		line += " | " + user
	}
	if details := strings.TrimSpace(event.Details); details != "" {
		line += " — " + details
	}
	return line
}

func caseHistoryEventTextColor(sc1 int) color.Color {
	if IsDarkMode() {
		text, _ := utils.SelectColorNRGBADark(sc1)
		return text
	}
	text, _ := utils.SelectColorNRGBA(sc1)
	return text
}
