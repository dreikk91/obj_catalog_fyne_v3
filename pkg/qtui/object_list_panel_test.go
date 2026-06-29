//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectListClipboardTextUsesVisibleFields(t *testing.T) {
	object := models.Object{
		ID:            42,
		DisplayNumber: "A-42",
		Name:          "  Office  ",
		Address:       " Main st. 1 ",
	}

	got := objectListClipboardText(object)
	want := "№A-42 | Office | Main st. 1"
	if got != want {
		t.Fatalf("objectListClipboardText() = %q, want %q", got, want)
	}
}

func TestObjectListClipboardTextFallsBackToNumericDisplayNumber(t *testing.T) {
	got := objectListClipboardText(models.Object{ID: 77})

	if got != "№77" {
		t.Fatalf("objectListClipboardText() = %q, want numeric display number", got)
	}
}

func TestBridgeDisplayBlockModeUsesMonitoringStatus(t *testing.T) {
	tests := []struct {
		status models.MonitoringStatus
		want   contracts.DisplayBlockMode
	}{
		{status: models.MonitoringStatusActive, want: contracts.DisplayBlockNone},
		{status: models.MonitoringStatusBlocked, want: contracts.DisplayBlockTemporaryOff},
		{status: models.MonitoringStatusDebug, want: contracts.DisplayBlockDebug},
	}
	for _, test := range tests {
		object := models.Object{ID: 10001, MonitoringStatus: test.status}
		if got := bridgeDisplayBlockMode(object); got != test.want {
			t.Fatalf("bridgeDisplayBlockMode(%q) = %d, want %d", test.status, got, test.want)
		}
	}
}

func TestObjectRowsSignatureIncludesMonitoringStatus(t *testing.T) {
	active := []models.Object{{ID: 10001, MonitoringStatus: models.MonitoringStatusActive}}
	blocked := []models.Object{{ID: 10001, MonitoringStatus: models.MonitoringStatusBlocked}}
	if objectRowsSignature(active) == objectRowsSignature(blocked) {
		t.Fatal("monitoring status must change the object rows signature")
	}
}
