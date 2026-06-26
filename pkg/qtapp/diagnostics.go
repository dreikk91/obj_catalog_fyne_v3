//go:build qt

package qtapp

import (
	"fmt"
	"strings"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/version"
)

const maxDiagnosticsSamples = 30

func (a *Application) showDiagnostics() {
	if a == nil || a.ui == nil {
		return
	}
	a.ui.ShowInfo("Діагностика", a.diagnosticsText())
}

func (a *Application) diagnosticsText() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Версія: %s\n", version.Current().String())
	if a.runtime != nil {
		fmt.Fprintf(&b, "%s\n", backendStatusText(a.runtime))
	} else {
		b.WriteString("Джерела даних: не ініціалізовано\n")
	}
	if a.ui != nil {
		if qtPrefs, ok := a.ui.Preferences().(*config.QtPreferences); ok {
			if filename := strings.TrimSpace(qtPrefs.FileName()); filename != "" {
				fmt.Fprintf(&b, "Файл налаштувань: %s\n", filename)
			}
		}
	}
	if a.currentObject != nil {
		fmt.Fprintf(&b, "Поточний об'єкт: %d | %s\n", a.currentObject.ID, strings.TrimSpace(a.currentObject.Name))
		fmt.Fprintf(&b, "Картка об'єкта: зон %d, відповідальних %d\n", a.currentObjectZones, a.currentObjectContacts)
	} else {
		b.WriteString("Поточний об'єкт: не вибрано\n")
	}
	fmt.Fprintf(
		&b,
		"Дані в UI: об'єктів %d, тривог %d, подій журналу %d\n",
		a.currentObjectsCount,
		a.currentAlarmsCount,
		a.currentEventsCount,
	)

	samples := snapshotQtPerformance()
	b.WriteString("\nОстанні операції Qt:\n")
	if len(samples) == 0 {
		b.WriteString("немає вимірів\n")
		return b.String()
	}
	start := len(samples) - maxDiagnosticsSamples
	if start < 0 {
		start = 0
	}
	for i := len(samples) - 1; i >= start; i-- {
		sample := samples[i]
		fmt.Fprintf(
			&b,
			"%s  %-22s %4d мс\n",
			sample.At.Format("15:04:05"),
			sample.Operation,
			sample.Elapsed.Milliseconds(),
		)
	}
	return b.String()
}
