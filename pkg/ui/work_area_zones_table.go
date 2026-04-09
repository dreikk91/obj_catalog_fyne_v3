package ui

import (
	"math"
	"strings"

	"fyne.io/fyne/v2"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
)

const (
	zoneTableNumberColumnWidth  float32 = 50
	zoneTableNameDefaultWidth   float32 = 260
	zoneTableNameMinWidth       float32 = 190
	zoneTableTypeColumnWidth    float32 = 140
	zoneTableStatusColumnWidth  float32 = 100
	zoneTableCopyColumnWidth    float32 = 40
	zoneTableMinRowHeight       float32 = 36
	zoneTableVerticalPadding    float32 = 12
	groupedZoneTableNameWidth   float32 = 300
	groupedZoneTableTypeWidth   float32 = 140
	groupedZoneTableStatusWidth float32 = 110
)

func zoneTableNameColumnWidth(totalWidth float32) float32 {
	if totalWidth <= 0 {
		return zoneTableNameDefaultWidth
	}

	fixedWidth := zoneTableNumberColumnWidth + zoneTableTypeColumnWidth + zoneTableStatusColumnWidth + zoneTableCopyColumnWidth
	available := totalWidth - fixedWidth - 10
	if available < zoneTableNameMinWidth {
		return zoneTableNameMinWidth
	}
	return available
}

func updateZoneTableRowHeights(table *widget.Table, zones []models.Zone, nameWidth float32) {
	if table == nil {
		return
	}

	textSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	for row, zone := range zones {
		table.SetRowHeight(row, zoneTableRowHeight(zone, nameWidth, zoneTableTypeColumnWidth, textSize))
	}
}

func updateGroupedZoneTableRowHeights(table *widget.Table, zones []models.Zone) {
	if table == nil {
		return
	}

	textSize := fyne.CurrentApp().Settings().Theme().Size(fyneTheme.SizeNameText)
	for row, zone := range zones {
		table.SetRowHeight(row, zoneTableRowHeight(zone, groupedZoneTableNameWidth, groupedZoneTableTypeWidth, textSize))
	}
}

func zoneTableRowHeight(zone models.Zone, nameWidth float32, typeWidth float32, textSize float32) float32 {
	style := fyne.TextStyle{}
	lineHeight := fyne.MeasureText("Ag", textSize, style).Height
	lines := maxWrappedTextLineCount(
		wrappedTextLineCount(zone.Name, nameWidth, textSize, style),
		wrappedTextLineCount(zone.SensorType, typeWidth, textSize, style),
	)

	height := float32(lines)*lineHeight + zoneTableVerticalPadding
	if height < zoneTableMinRowHeight {
		return zoneTableMinRowHeight
	}
	return height
}

func wrappedTextLineCount(text string, width float32, textSize float32, style fyne.TextStyle) int {
	if width <= 0 {
		return 1
	}

	blocks := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	totalLines := 0
	spaceWidth := fyne.MeasureText(" ", textSize, style).Width

	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			totalLines++
			continue
		}

		words := strings.Fields(block)
		if len(words) == 0 {
			totalLines++
			continue
		}

		lines := 1
		currentWidth := float32(0)
		for _, word := range words {
			wordWidth := fyne.MeasureText(word, textSize, style).Width
			if wordWidth > width {
				if currentWidth > 0 {
					lines++
					currentWidth = 0
				}
				chunks := int(math.Ceil(float64(wordWidth / width)))
				lines += chunks - 1
				currentWidth = float32(math.Mod(float64(wordWidth), float64(width)))
				if currentWidth == 0 {
					currentWidth = width
				}
				continue
			}

			if currentWidth == 0 {
				currentWidth = wordWidth
				continue
			}

			if currentWidth+spaceWidth+wordWidth > width {
				lines++
				currentWidth = wordWidth
				continue
			}

			currentWidth += spaceWidth + wordWidth
		}

		totalLines += lines
	}

	if totalLines < 1 {
		return 1
	}
	return totalLines
}

func maxWrappedTextLineCount(values ...int) int {
	max := 1
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}
