// pkg/ui/utils.go
package ui

import (
	"image/color"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/theme"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

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
func GetStatusColor(status models.ObjectStatus) color.Color {
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

// ShowToast показує коротке ненав'язливе повідомлення внизу вікна.
// Використовується як зворотний зв'язок для дій (копіювання, навігація, тощо).
func ShowToast(win fyne.Window, message string) {
	if win == nil || win.Canvas() == nil || message == "" {
		return
	}

	bg := canvas.NewRectangle(color.NRGBA{R: 20, G: 20, B: 20, A: 210})
	txt := canvas.NewText(message, color.White)
	txt.TextStyle = fyne.TextStyle{Bold: true}
	content := container.NewPadded(txt)
	pop := widget.NewPopUp(container.NewStack(bg, content), win.Canvas())

	// Позиція: по центру внизу з відступом.
	size := pop.Content.MinSize()
	canvasSize := win.Canvas().Size()
	marginBottom := float32(20)
	pos := fyne.NewPos(
		(canvasSize.Width-size.Width)/2,
		canvasSize.Height-size.Height-marginBottom,
	)
	pop.Resize(size)
	pop.Move(pos)
	pop.Show()

	time.AfterFunc(1400*time.Millisecond, func() {
		fyne.Do(func() {
			pop.Hide()
		})
	})
}
