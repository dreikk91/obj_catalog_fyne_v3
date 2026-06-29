//go:build qt

package qtui

import (
	"testing"

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
