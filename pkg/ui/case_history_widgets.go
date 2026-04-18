package ui

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const caseHistoryVisibleEventRows = 5

func buildCaseHistoryEventList(group viewmodels.WorkAreaCaseHistoryGroup) fyne.CanvasObject {
	if len(group.Events) <= 1 {
		label := widget.NewLabel("Після початку тривоги додаткових подій поки немає.")
		label.Wrapping = fyne.TextWrapWord
		return container.NewPadded(label)
	}

	rows := make([]fyne.CanvasObject, 0, (len(group.Events)-1)*2)
	eventLines := make([]fyne.CanvasObject, 0, len(group.Events)-1)
	for idx, event := range group.Events[1:] {
		line := buildCaseHistoryEventLine(event)
		eventLines = append(eventLines, line)
		rows = append(rows, line)
		if idx < len(group.Events)-2 {
			rows = append(rows, widget.NewSeparator())
		}
	}

	content := container.NewPadded(container.NewVBox(rows...))
	if len(eventLines) <= caseHistoryVisibleEventRows {
		return content
	}

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(0, caseHistoryEventViewportHeight(rows)))
	return scroll
}

func buildCaseHistoryEventLine(event models.Event) fyne.CanvasObject {
	textColor, rowColor := eventRowColorsBySeverity(event.VisualSeverityValue(), event.SC1)
	text := canvas.NewText(caseHistoryEventText(event), textColor)
	text.TextSize = fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	text.TextStyle = fyne.TextStyle{Bold: event.IsCritical()}

	bg := canvas.NewRectangle(rowColor)
	bg.CornerRadius = 6

	return container.NewPadded(
		container.NewStack(
			bg,
			container.NewPadded(text),
		),
	)
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

func caseHistoryEventViewportHeight(rows []fyne.CanvasObject) float32 {
	if len(rows) == 0 {
		return 0
	}

	firstVisibleLineIdx := len(rows) - (caseHistoryVisibleEventRows*2 - 1)
	if firstVisibleLineIdx < 0 {
		firstVisibleLineIdx = 0
	}

	height := float32(0)
	for _, row := range rows[firstVisibleLineIdx:] {
		height += row.MinSize().Height
	}

	padding := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNamePadding) * 2
	return height + padding
}

func scrollCaseHistoryToBottom(obj fyne.CanvasObject) {
	switch typed := obj.(type) {
	case *container.Scroll:
		typed.ScrollToBottom()
	case *fyne.Container:
		for _, child := range typed.Objects {
			scrollCaseHistoryToBottom(child)
		}
	}
}
