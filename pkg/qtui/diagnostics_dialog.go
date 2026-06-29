//go:build qt

package qtui

import (
	"fmt"

	qt "github.com/mappu/miqt/qt6"
)

type DiagnosticsOperation struct {
	Operation string
	Count     int
	AverageMS int64
	MaximumMS int64
}

type DiagnosticsSample struct {
	Time      string
	Operation string
	ElapsedMS int64
}

type DiagnosticsReport struct {
	Version         string
	Sources         string
	SettingsStorage string
	CurrentObject   string
	CardState       string
	UIState         string
	Assessment      string
	Summary         []DiagnosticsOperation
	Samples         []DiagnosticsSample
	RawText         string
}

func ShowDiagnosticsDialog(parent *qt.QWidget, report DiagnosticsReport) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Діагностика")
	dialog.Resize(920, 680)

	overview := qt.NewQWidget2()
	overviewForm := qt.NewQFormLayout2()
	overviewForm.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	addDiagnosticsValue(overviewForm, "Версія", report.Version)
	addDiagnosticsValue(overviewForm, "Джерела даних", report.Sources)
	addDiagnosticsValue(overviewForm, "Налаштування", report.SettingsStorage)
	addDiagnosticsValue(overviewForm, "Поточний об'єкт", report.CurrentObject)
	addDiagnosticsValue(overviewForm, "Картка", report.CardState)
	addDiagnosticsValue(overviewForm, "Дані в UI", report.UIState)
	overview.SetLayout(overviewForm.QLayout)

	assessment := qt.NewQLabel3(report.Assessment)
	assessment.SetWordWrap(true)
	assessment.SetStyleSheet(diagnosticsAssessmentStyle(report.Summary))

	tabs := qt.NewQTabWidget2()
	tabs.AddTab(buildDiagnosticsSummaryTable(report.Summary).QWidget, "Підсумок")
	tabs.AddTab(buildDiagnosticsSamplesTable(report.Samples).QWidget, "Останні виміри")

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	copyButton := buttons.AddButton2("Копіювати звіт", qt.QDialogButtonBox__ActionRole)
	copyButton.OnClicked(func() {
		setClipboardText(report.RawText)
	})
	buttons.OnRejected(dialog.Reject)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddWidget(overview)
	layout.AddWidget(assessment.QWidget)
	layout.AddWidget(tabs.QWidget)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)
	dialog.Exec()
}

func addDiagnosticsValue(form *qt.QFormLayout, label string, value string) {
	field := qt.NewQLabel3(value)
	field.SetWordWrap(true)
	field.SetTextInteractionFlags(qt.TextSelectableByMouse)
	form.AddRow3(label, field.QWidget)
}

func buildDiagnosticsSummaryTable(rows []DiagnosticsOperation) *qt.QTableWidget {
	table := newDiagnosticsTable(len(rows), []string{"Операція", "Кількість", "Середнє", "Максимум", "Оцінка"})
	for row, operation := range rows {
		level, color := diagnosticsPerformanceLevel(operation.AverageMS)
		values := []string{
			operation.Operation,
			fmt.Sprintf("%d", operation.Count),
			diagnosticsDuration(operation.AverageMS),
			diagnosticsDuration(operation.MaximumMS),
			level,
		}
		setDiagnosticsTableRow(table, row, values, color)
	}
	for column, width := range []int{310, 90, 120, 120, 130} {
		table.SetColumnWidth(column, width)
	}
	table.HorizontalHeader().SetStretchLastSection(true)
	return table
}

func buildDiagnosticsSamplesTable(rows []DiagnosticsSample) *qt.QTableWidget {
	table := newDiagnosticsTable(len(rows), []string{"Час", "Операція", "Тривалість", "Оцінка"})
	for row, sample := range rows {
		level, color := diagnosticsPerformanceLevel(sample.ElapsedMS)
		values := []string{sample.Time, sample.Operation, diagnosticsDuration(sample.ElapsedMS), level}
		setDiagnosticsTableRow(table, row, values, color)
	}
	for column, width := range []int{100, 420, 130, 140} {
		table.SetColumnWidth(column, width)
	}
	table.HorizontalHeader().SetStretchLastSection(true)
	return table
}

func newDiagnosticsTable(rowCount int, headers []string) *qt.QTableWidget {
	table := qt.NewQTableWidget3(rowCount, len(headers))
	table.SetHorizontalHeaderLabels(headers)
	table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	table.SetAlternatingRowColors(true)
	table.VerticalHeader().SetVisible(false)
	return table
}

func setDiagnosticsTableRow(table *qt.QTableWidget, row int, values []string, statusColor *qt.QColor) {
	for column, value := range values {
		item := qt.NewQTableWidgetItem2(value)
		if column == len(values)-1 && statusColor != nil {
			item.SetForeground(qt.NewQBrush3(statusColor))
		}
		table.SetItem(row, column, item)
	}
}

func diagnosticsPerformanceLevel(milliseconds int64) (string, *qt.QColor) {
	switch {
	case milliseconds >= 2000:
		return "Критично", qt.NewQColor3(190, 35, 35)
	case milliseconds >= 500:
		return "Повільно", qt.NewQColor3(190, 105, 0)
	case milliseconds >= 150:
		return "Увага", qt.NewQColor3(155, 120, 0)
	default:
		return "Норма", qt.NewQColor3(25, 125, 70)
	}
}

func diagnosticsDuration(milliseconds int64) string {
	if milliseconds >= 1000 {
		return fmt.Sprintf("%.2f с", float64(milliseconds)/1000)
	}
	return fmt.Sprintf("%d мс", milliseconds)
}

func diagnosticsAssessmentStyle(summary []DiagnosticsOperation) string {
	maximum := int64(0)
	for _, operation := range summary {
		if operation.AverageMS > maximum {
			maximum = operation.AverageMS
		}
	}
	switch {
	case maximum >= 2000:
		return "padding: 8px; border: 1px solid #c62828; color: #a61b1b; font-weight: 600;"
	case maximum >= 500:
		return "padding: 8px; border: 1px solid #c07800; color: #9a6200; font-weight: 600;"
	default:
		return "padding: 8px; border: 1px solid #27804c; color: #1f7041; font-weight: 600;"
	}
}
