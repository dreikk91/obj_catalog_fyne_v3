//go:build qt

package qtapp

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestIntersectAlarmProcessingOptionsKeepsCommonCodesInFirstOrder(t *testing.T) {
	got := intersectAlarmProcessingOptions(
		[]contracts.AlarmProcessingOption{
			{Code: "10", Label: "False alarm"},
			{Code: "20", Label: "Fire brigade"},
			{Code: "30", Label: "Other"},
		},
		[]contracts.AlarmProcessingOption{
			{Code: "20", Label: "Brigade"},
			{Code: "10", Label: "False"},
		},
	)

	if len(got) != 2 {
		t.Fatalf("len(intersection) = %d, want 2: %+v", len(got), got)
	}
	if got[0].Code != "10" || got[0].Label != "False alarm" {
		t.Fatalf("first option = %+v, want code 10 with label from first set", got[0])
	}
	if got[1].Code != "20" || got[1].Label != "Fire brigade" {
		t.Fatalf("second option = %+v, want code 20 with label from first set", got[1])
	}
}

func TestIntersectAlarmProcessingOptionsNormalizesAndSkipsEmptyCodes(t *testing.T) {
	got := intersectAlarmProcessingOptions(
		[]contracts.AlarmProcessingOption{
			{Code: " 10 ", Label: " "},
			{Code: "", Label: "empty"},
			{Code: "10", Label: "duplicate"},
		},
		[]contracts.AlarmProcessingOption{
			{Code: "10", Label: "same"},
		},
	)

	if len(got) != 1 {
		t.Fatalf("len(intersection) = %d, want 1: %+v", len(got), got)
	}
	if got[0].Code != "10" || got[0].Label != "10" {
		t.Fatalf("option = %+v, want trimmed code and fallback label", got[0])
	}
}

func TestSameAlarmProcessingSourceRejectsMixedSources(t *testing.T) {
	alarms := []models.Alarm{
		{ObjectID: 100},
		{ObjectID: ids.StablePhoenixID("L00028")},
	}

	if sameAlarmProcessingSource(alarms) {
		t.Fatal("sameAlarmProcessingSource() = true, want false for mixed Bridge/Phoenix alarms")
	}
}

func TestSameAlarmProcessingSourceAllowsSameSource(t *testing.T) {
	alarms := []models.Alarm{
		{ObjectID: ids.CASLObjectIDNamespaceStart + 1},
		{ObjectID: ids.CASLObjectIDNamespaceStart + 2},
	}

	if !sameAlarmProcessingSource(alarms) {
		t.Fatal("sameAlarmProcessingSource() = false, want true for same CASL source")
	}
}
