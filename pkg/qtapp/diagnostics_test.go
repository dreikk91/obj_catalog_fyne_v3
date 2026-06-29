//go:build qt

package qtapp

import (
	"strings"
	"testing"

	"obj_catalog_fyne_v3/pkg/qtui"
)

func TestDiagnosticsAssessmentIdentifiesSlowObjectCard(t *testing.T) {
	summary := []qtui.DiagnosticsOperation{
		{Operation: "Оновлення списку об'єктів", AverageMS: 392, MaximumMS: 558},
		{Operation: "Завантаження картки об'єкта", AverageMS: 4992, MaximumMS: 6211},
		{Operation: "Відображення картки об'єкта", AverageMS: 7, MaximumMS: 27},
	}
	got := diagnosticsAssessment(summary)
	if !strings.Contains(got, "Завантаження картки об'єкта") || !strings.Contains(got, "5.0 с") {
		t.Fatalf("diagnosticsAssessment() = %q", got)
	}
}

func TestDiagnosticsSettingsStorageLabelsWindowsRegistry(t *testing.T) {
	got := diagnosticsSettingsStorage(`\HKEY_CURRENT_USER\Software\MOST\ObjCatalogQt`)
	if !strings.HasPrefix(got, "Реєстр Windows:") {
		t.Fatalf("diagnosticsSettingsStorage() = %q", got)
	}
}
