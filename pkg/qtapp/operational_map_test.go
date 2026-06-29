//go:build qt

package qtapp

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

func TestApplyObjectLocationsUpdatesOnlyMatchingObjects(t *testing.T) {
	objects := []models.Object{{ID: 1}, {ID: 2, Latitude: "old"}}
	applyObjectLocations(objects, []contracts.ObjectLocation{
		{ObjectID: 2, Latitude: "49.8", Longitude: "24.0"},
	})
	if objects[0].Latitude != "" {
		t.Fatalf("unmatched object changed: %+v", objects[0])
	}
	if objects[1].Latitude != "49.8" || objects[1].Longitude != "24.0" {
		t.Fatalf("matching object coordinates = %+v", objects[1])
	}
}
