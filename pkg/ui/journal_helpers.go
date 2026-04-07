package ui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/utils"
)

func eventRowColors(sc1 int) (textColor, rowColor color.NRGBA) {
	if IsDarkMode() {
		return utils.SelectColorNRGBADark(sc1)
	}
	return utils.SelectColorNRGBA(sc1)
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
