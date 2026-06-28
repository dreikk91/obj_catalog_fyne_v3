//go:build qt

package qtapp

import (
	"fmt"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/version"
)

const maxDiagnosticsSamples = 30

func (a *Application) showDiagnostics() {
	if a == nil || a.ui == nil {
		return
	}
	a.ui.ShowText("Діагностика", a.diagnosticsText())
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
	writeDiagnosticsSummary(&b, samples)
	start := len(samples) - maxDiagnosticsSamples
	if start < 0 {
		start = 0
	}
	b.WriteString("\nОстанні виміри:\n")
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

type diagnosticsOperationStat struct {
	count int
	total time.Duration
	max   time.Duration
}

func writeDiagnosticsSummary(b *strings.Builder, samples []qtPerformanceSample) {
	if b == nil || len(samples) == 0 {
		return
	}
	operations := make([]string, 0, 8)
	stats := make(map[string]diagnosticsOperationStat, 8)
	for _, sample := range samples {
		stat := stats[sample.Operation]
		if stat.count == 0 {
			operations = append(operations, sample.Operation)
		}
		stat.count++
		stat.total += sample.Elapsed
		if sample.Elapsed > stat.max {
			stat.max = sample.Elapsed
		}
		stats[sample.Operation] = stat
	}

	b.WriteString("Підсумок:\n")
	for _, operation := range operations {
		stat := stats[operation]
		average := time.Duration(0)
		if stat.count > 0 {
			average = stat.total / time.Duration(stat.count)
		}
		fmt.Fprintf(
			b,
			"%-22s count %3d  avg %4d мс  max %4d мс\n",
			operation,
			stat.count,
			average.Milliseconds(),
			stat.max.Milliseconds(),
		)
	}
}
