//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
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

func TestEventRowColorsUseSemanticPalette(t *testing.T) {
	tests := []struct {
		name  string
		event models.Event
		code  int
	}{
		{name: "critical overrides SC1", event: models.Event{Type: models.EventBurglary, SC1: 6}, code: 1},
		{name: "warning overrides SC1", event: models.Event{Type: models.EventPowerFail, SC1: 6}, code: 4},
		{name: "info overrides SC1", event: models.Event{Type: models.EventNotification, SC1: 1}, code: 6},
		{name: "normal preserves SC1", event: models.Event{Type: models.EventDisarm, SC1: 11}, code: 11},
		{name: "normal without SC1", event: models.Event{Type: models.EventArm}, code: 10},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotText, gotRow := eventRowColors(test.event)
			wantText, wantRow := utils.SelectColorNRGBA(test.code)
			if gotText != wantText || gotRow != wantRow {
				t.Fatalf(
					"eventRowColors() = text %+v, row %+v; want text %+v, row %+v",
					gotText,
					gotRow,
					wantText,
					wantRow,
				)
			}
		})
	}
}
