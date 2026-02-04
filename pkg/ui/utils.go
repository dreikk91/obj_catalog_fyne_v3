// pkg/ui/utils.go
package ui

import (
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/theme"

	"fyne.io/fyne/v2"
)

// itoa - проста конвертація int в string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

func getStatusIcon(status models.ObjectStatus) string {
	switch status {
	case models.StatusFire:
		return "🔴"
	case models.StatusFault, models.StatusOffline:
		return "🟡"
	case models.StatusNormal:
		return "🟢"
	default:
		return "🔵"
	}
}

// GetStatusColor повертає колір для статусу
func GetStatusColor(status models.ObjectStatus) interface{ RGBA() (r, g, b, a uint32) } {
	switch status {
	case models.StatusFire:
		return theme.ColorFire
	case models.StatusFault, models.StatusOffline:
		return theme.ColorFault
	case models.StatusNormal:
		return theme.ColorNormal
	default:
		return theme.ColorInfo
	}
}

// IsDarkMode перевіряє чи зараз активна темна тема
func IsDarkMode() bool {
	t := fyne.CurrentApp().Settings().Theme()
	_, ok := t.(*theme.DarkTheme)
	return ok
}
