//go:build qt

package qtapp

import (
	"fmt"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/qtui"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/version"
)

const maxDiagnosticsSamples = 30

func (a *Application) showDiagnostics() {
	if a == nil || a.ui == nil {
		return
	}
	a.ui.ShowDiagnostics(a.diagnosticsReport())
}

func (a *Application) diagnosticsReport() qtui.DiagnosticsReport {
	report := qtui.DiagnosticsReport{
		Version: version.Current().String(),
		Sources: "не ініціалізовано",
	}
	if report.Version == "dev" {
		report.Version = "dev (локальна збірка)"
	}
	if a.runtime != nil {
		report.Sources = strings.TrimPrefix(backendStatusText(a.runtime), "Джерела даних: ")
	}
	if a.ui != nil {
		if qtPrefs, ok := a.ui.Preferences().(*config.QtPreferences); ok {
			report.SettingsStorage = diagnosticsSettingsStorage(qtPrefs.FileName())
		}
	}
	if a.currentObject != nil {
		report.CurrentObject = fmt.Sprintf(
			"№%s | %s | %s",
			viewmodels.ObjectDisplayNumber(*a.currentObject),
			strings.TrimSpace(a.currentObject.Name),
			viewmodels.ObjectSourceByID(a.currentObject.ID),
		)
		report.CardState = fmt.Sprintf("зон %d, відповідальних %d", a.currentObjectZones, a.currentObjectContacts)
	} else {
		report.CurrentObject = "не вибрано"
		report.CardState = "немає даних"
	}
	report.UIState = fmt.Sprintf(
		"об'єктів %d, тривог %d, подій журналу завантажено %d",
		a.currentObjectsCount,
		a.currentAlarmsCount,
		a.currentEventsCount,
	)

	samples := snapshotQtPerformance()
	report.Summary = diagnosticsSummary(samples)
	start := len(samples) - maxDiagnosticsSamples
	if start < 0 {
		start = 0
	}
	for index := len(samples) - 1; index >= start; index-- {
		sample := samples[index]
		report.Samples = append(report.Samples, qtui.DiagnosticsSample{
			Time:      sample.At.Format("15:04:05"),
			Operation: diagnosticsOperationName(sample.Operation),
			ElapsedMS: sample.Elapsed.Milliseconds(),
		})
	}
	report.Assessment = diagnosticsAssessment(report.Summary)
	report.RawText = diagnosticsRawText(report)
	return report
}

func diagnosticsSummary(samples []qtPerformanceSample) []qtui.DiagnosticsOperation {
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
	result := make([]qtui.DiagnosticsOperation, 0, len(operations))
	for _, operation := range operations {
		stat := stats[operation]
		average := time.Duration(0)
		if stat.count > 0 {
			average = stat.total / time.Duration(stat.count)
		}
		result = append(result, qtui.DiagnosticsOperation{
			Operation: diagnosticsOperationName(operation),
			Count:     stat.count,
			AverageMS: average.Milliseconds(),
			MaximumMS: stat.max.Milliseconds(),
		})
	}
	return result
}

func diagnosticsAssessment(summary []qtui.DiagnosticsOperation) string {
	var slowest qtui.DiagnosticsOperation
	for _, operation := range summary {
		if operation.AverageMS > slowest.AverageMS {
			slowest = operation
		}
	}
	if slowest.Operation == "" {
		return "Ще недостатньо вимірів для оцінки."
	}
	switch {
	case slowest.AverageMS >= 2000:
		return fmt.Sprintf(
			"Проблема продуктивності: «%s» займає в середньому %.1f с, максимум %.1f с.",
			slowest.Operation,
			float64(slowest.AverageMS)/1000,
			float64(slowest.MaximumMS)/1000,
		)
	case slowest.AverageMS >= 500:
		return fmt.Sprintf("Потребує уваги: «%s» у середньому %d мс.", slowest.Operation, slowest.AverageMS)
	default:
		return "Критично повільних операцій у поточній вибірці немає."
	}
}

func diagnosticsOperationName(operation string) string {
	switch operation {
	case "refreshData":
		return "Запуск загального оновлення"
	case "refreshObjects":
		return "Оновлення списку об'єктів"
	case "refreshAlarms":
		return "Оновлення тривог"
	case "refreshEvents":
		return "Оновлення журналу"
	case "selectObject":
		return "Вибір об'єкта"
	case "loadObjectDetails":
		return "Завантаження картки об'єкта"
	case "applyObjectDetails":
		return "Відображення картки об'єкта"
	default:
		return operation
	}
}

func diagnosticsSettingsStorage(value string) string {
	value = strings.TrimSpace(value)
	if strings.Contains(strings.ToUpper(value), "HKEY_CURRENT_USER") {
		return "Реєстр Windows: " + value
	}
	if value == "" {
		return "не визначено"
	}
	return value
}

func diagnosticsRawText(report qtui.DiagnosticsReport) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "Версія: %s\n", report.Version)
	fmt.Fprintf(&builder, "Джерела даних: %s\n", report.Sources)
	fmt.Fprintf(&builder, "Сховище налаштувань: %s\n", report.SettingsStorage)
	fmt.Fprintf(&builder, "Поточний об'єкт: %s\n", report.CurrentObject)
	fmt.Fprintf(&builder, "Картка об'єкта: %s\n", report.CardState)
	fmt.Fprintf(&builder, "Дані в UI: %s\n", report.UIState)
	fmt.Fprintf(&builder, "Оцінка: %s\n\n", report.Assessment)
	builder.WriteString("Підсумок:\n")
	for _, row := range report.Summary {
		fmt.Fprintf(&builder, "%s | count %d | avg %d мс | max %d мс\n", row.Operation, row.Count, row.AverageMS, row.MaximumMS)
	}
	builder.WriteString("\nОстанні виміри:\n")
	for _, row := range report.Samples {
		fmt.Fprintf(&builder, "%s | %s | %d мс\n", row.Time, row.Operation, row.ElapsedMS)
	}
	return builder.String()
}

type diagnosticsOperationStat struct {
	count int
	total time.Duration
	max   time.Duration
}
