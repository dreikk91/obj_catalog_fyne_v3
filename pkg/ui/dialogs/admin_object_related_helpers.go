package dialogs

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func showZoneFillDialog(parent fyne.Window, defaultCount int64, onApply func(count int64), statusLabel *widget.Label) {
	entry := widget.NewEntry()
	if defaultCount <= 0 {
		defaultCount = 24
	}
	entry.SetText(strconv.FormatInt(defaultCount, 10))
	entry.SetPlaceHolder("Кількість зон")

	form := widget.NewForm(
		widget.NewFormItem("Кількість зон:", entry),
	)

	dlg := dialog.NewCustomConfirm("Заповнення зон", "Застосувати", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}
		count, err := strconv.ParseInt(strings.TrimSpace(entry.Text), 10, 64)
		if err != nil {
			statusLabel.SetText("Некоректна кількість зон")
			return
		}
		onApply(count)
	}, parent)
	dlg.Show()
}

func suggestZoneFillCount(provider contracts.AdminObjectZonesTabProvider, objn int64, current []contracts.AdminObjectZone) int64 {
	maxZone := int64(0)
	for _, z := range current {
		if z.ZoneNumber > maxZone {
			maxZone = z.ZoneNumber
		}
	}

	card, err := provider.GetObjectCard(objn)
	if err == nil && card.PPKID > 0 {
		ppkItems, ppkErr := provider.ListPPKConstructor()
		if ppkErr == nil {
			for _, it := range ppkItems {
				if it.ID == card.PPKID && it.ZoneCount > 0 {
					return it.ZoneCount
				}
			}
		}
	}

	if maxZone > 0 {
		return maxZone
	}
	return 24
}

func focusIfOnCanvas(parent fyne.Window, target fyne.Focusable) {
	if parent == nil || target == nil {
		return
	}
	canvas := parent.Canvas()
	if canvas == nil {
		return
	}
	root := canvas.Content()
	if root == nil {
		return
	}
	targetObj, ok := target.(fyne.CanvasObject)
	if !ok {
		return
	}
	if !containsCanvasObject(root, targetObj) {
		return
	}
	canvas.Focus(target)
}

func containsCanvasObject(root fyne.CanvasObject, target fyne.CanvasObject) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	cont, ok := root.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range cont.Objects {
		if containsCanvasObject(child, target) {
			return true
		}
	}
	return false
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
