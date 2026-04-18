package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

func eventRowColors(sc1 int) (textColor, rowColor color.NRGBA) {
	if IsDarkMode() {
		return utils.SelectColorNRGBADark(sc1)
	}
	return utils.SelectColorNRGBA(sc1)
}

func eventRowColorsBySeverity(severity models.VisualSeverity, sc1 int) (textColor, rowColor color.NRGBA) {
	if severity != models.VisualSeverityUnknown {
		sc1 = sc1FromVisualSeverity(severity, sc1)
	}
	if IsDarkMode() {
		return utils.SelectColorNRGBADark(sc1)
	}
	return utils.SelectColorNRGBA(sc1)
}

func sc1FromVisualSeverity(severity models.VisualSeverity, fallback int) int {
	switch severity {
	case models.VisualSeverityCritical:
		return 1
	case models.VisualSeverityWarning:
		return 2
	case models.VisualSeverityInfo:
		return 10
	case models.VisualSeverityNormal:
		if fallback != 0 {
			return fallback
		}
		return 10
	default:
		return fallback
	}
}

func updateSelectPreservingValue(sel *widget.Select, options []string, current string) {
	if sel == nil || len(options) == 0 {
		return
	}

	sel.Options = options
	target := options[0]
	for _, option := range options {
		if strings.HasPrefix(option, current+" (") || option == current {
			target = option
			break
		}
	}

	handler := sel.OnChanged
	sel.OnChanged = nil
	sel.SetSelected(target)
	sel.OnChanged = handler
	sel.Refresh()
}
