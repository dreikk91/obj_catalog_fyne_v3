//go:build qt

package qtui

import (
	"strings"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestParseOperationalCoordinates(t *testing.T) {
	lat, lon, ok := parseOperationalCoordinates("49,8397", "24.0297")
	if !ok || lat != 49.8397 || lon != 24.0297 {
		t.Fatalf("coordinates = %v/%v ok=%v", lat, lon, ok)
	}
	if _, _, ok := parseOperationalCoordinates("999", "24"); ok {
		t.Fatal("out-of-range coordinates must be rejected")
	}
}

func TestBuildOperationalMapItemsUsesObjectCoordinatesForAlarm(t *testing.T) {
	objects := []models.Object{{ID: 7, DisplayNumber: "1007", Name: "Офіс", Latitude: "49.8", Longitude: "24.0"}}
	alarms := []models.Alarm{{ObjectID: 7, ObjectNumber: "1007", Type: models.AlarmFire}}
	groups := []contracts.FrontendResponseGroup{{ID: "1", Name: "МГР 1", Latitude: "49.9", Longitude: "24.1"}}
	got := buildOperationalMapItems(objects, alarms, groups, false, true, true)
	if len(got) != 2 || got[0].Kind != operationalMapAlarm || got[1].Kind != operationalMapGroup {
		t.Fatalf("map items = %+v", got)
	}
}

func TestBuildOperationalMapItemsGroupsAlarmsByObject(t *testing.T) {
	objects := []models.Object{{ID: 7, DisplayNumber: "1007", Name: "Офіс", Latitude: "49.8", Longitude: "24.0"}}
	alarms := []models.Alarm{
		{ObjectID: 7, ObjectNumber: "1007", Type: models.AlarmPowerFail, Time: time.Date(2026, 6, 29, 15, 0, 0, 0, time.Local)},
		{ObjectID: 7, ObjectNumber: "1007", Type: models.AlarmFire, Time: time.Date(2026, 6, 29, 14, 0, 0, 0, time.Local)},
		{ObjectID: 7, ObjectNumber: "1007", Type: models.AlarmBatteryLow, Time: time.Date(2026, 6, 29, 16, 0, 0, 0, time.Local)},
	}

	got := buildOperationalMapItems(objects, alarms, nil, false, true, false)
	if len(got) != 1 {
		t.Fatalf("map items count = %d, want 1: %+v", len(got), got)
	}
	if !strings.Contains(got[0].Details, "ПОЖЕЖА") || !strings.Contains(got[0].Details, "3 активних подій") {
		t.Fatalf("grouped alarm details = %q", got[0].Details)
	}
}
