//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestSC1FromVisualSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity models.VisualSeverity
		fallback int
		want     int
	}{
		{name: "critical", severity: models.VisualSeverityCritical, fallback: 30, want: 1},
		{name: "warning", severity: models.VisualSeverityWarning, fallback: 30, want: 4},
		{name: "info", severity: models.VisualSeverityInfo, fallback: 30, want: 6},
		{name: "normal uses fallback", severity: models.VisualSeverityNormal, fallback: 28, want: 28},
		{name: "normal default", severity: models.VisualSeverityNormal, want: 10},
		{name: "unknown uses fallback", severity: models.VisualSeverityUnknown, fallback: 25, want: 25},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sc1FromVisualSeverity(test.severity, test.fallback); got != test.want {
				t.Fatalf("sc1FromVisualSeverity(%q, %d) = %d, want %d", test.severity, test.fallback, got, test.want)
			}
		})
	}
}

func TestObjectPowerStatusCardState(t *testing.T) {
	tests := []struct {
		name   string
		object models.Object
		want   string
	}{
		{name: "normal", object: models.Object{}, want: "220В та АКБ в нормі"},
		{name: "mains fault", object: models.Object{PowerFault: 1}, want: "Аварія 220В"},
		{name: "battery fault", object: models.Object{AkbState: 1}, want: "Несправність АКБ"},
		{name: "unknown", object: models.Object{PowerFault: -1, AkbState: -1}, want: "Невідомо"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _, _ := objectPowerStatusCardState(test.object)
			if got != test.want {
				t.Fatalf("objectPowerStatusCardState() = %q, want %q", got, test.want)
			}
		})
	}
}
