//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

func TestNormalizeTopSplitterSizesKeepsUserObjectListWidth(t *testing.T) {
	got := normalizeTopSplitterSizes([]int{700, 580}, 1280)
	if got[0] != 700 || got[1] != 580 {
		t.Fatalf("normalizeTopSplitterSizes() = %v, want [700 580]", got)
	}
}

func TestNormalizeTopSplitterSizesPreservesReasonableWidth(t *testing.T) {
	got := normalizeTopSplitterSizes([]int{320, 1040}, 1360)
	if got[0] != 320 || got[1] != 1040 {
		t.Fatalf("normalizeTopSplitterSizes() = %v, want [320 1040]", got)
	}
}

func TestConstrainWindowSizeUsesAvailableLaptopGeometry(t *testing.T) {
	width, height := constrainWindowSize(1920, 1080, 1280, 760)
	if width != 1280 || height != 760 {
		t.Fatalf("constrainWindowSize() = %dx%d, want 1280x760", width, height)
	}
}

func TestJournalDockHeightScalesForLaptop(t *testing.T) {
	if got := journalDockHeight(760); got != 182 {
		t.Fatalf("journalDockHeight(760) = %d, want 182", got)
	}
}

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

func TestEventRowSignatureIncludesZone(t *testing.T) {
	base := models.Event{ID: 1, ZoneNumber: 3}
	changed := base
	changed.ZoneNumber = 4
	if eventRowSignature(base) == eventRowSignature(changed) {
		t.Fatal("eventRowSignature() must change when the zone changes")
	}
}

func TestEventDetailsTextIncludesZoneNameForZoneAlarm(t *testing.T) {
	event := models.Event{Type: models.EventFire, ZoneNumber: 12, ZoneName: "Склад", Details: "Пожежна тривога"}
	if got := eventDetailsText(event); got != "Пожежна тривога | Склад" {
		t.Fatalf("eventDetailsText() = %q", got)
	}
}

func TestEventDetailsTextSkipsZoneForBatteryFault(t *testing.T) {
	event := models.Event{Type: models.EventFault, ZoneNumber: 12, ZoneName: "Склад", Details: "Несправність АКБ"}
	if got := eventDetailsText(event); got != "Несправність АКБ" {
		t.Fatalf("eventDetailsText() = %q", got)
	}
}

func TestEventDetailsTextIncludesGroupForArmDisarm(t *testing.T) {
	event := models.Event{Type: models.EventArm, GroupName: "Нічна охорона", Details: "Постановка"}
	if got := eventDetailsText(event); got != "Постановка | Група: Нічна охорона" {
		t.Fatalf("eventDetailsText() = %q", got)
	}
}

func TestEventDetailsTextCollapsesBridgePadding(t *testing.T) {
	event := models.Event{Details: "Пожежа (Грп.№01 Тірас 16 оптичний                         )"}
	if got := eventDetailsText(event); got != "Пожежа (Грп.№01 Тірас 16 оптичний)" {
		t.Fatalf("eventDetailsText() = %q", got)
	}
}

func TestMinimumColumnWidthsMatchEventTables(t *testing.T) {
	if got := len(minimumColumnWidths("events")); got != len(eventLogHeaders()) {
		t.Fatalf("events minimum widths = %d, headers = %d", got, len(eventLogHeaders()))
	}
	if got := len(minimumColumnWidths("object_events")); got != 3 {
		t.Fatalf("object events minimum widths = %d, want 3", got)
	}
}
