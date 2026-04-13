package data

import (
	"database/sql"
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestPhoenixBuildObjects_PopulatesSIMNumbersFromListRows(t *testing.T) {
	provider := &PhoenixDataProvider{
		panelByID: make(map[int]string),
		idByPanel: make(map[string]int),
	}

	objects := provider.buildObjects([]phoenixObjectGroupRow{
		{
			PanelID:     "L00028",
			GroupNo:     1,
			CompanyName: sql.NullString{String: "Phoenix One", Valid: true},
			Sim1Number:  sql.NullString{String: "380661234567", Valid: true},
			Sim2Number:  sql.NullString{String: "380671112233", Valid: true},
		},
	})

	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].SIM1 != "380661234567" {
		t.Fatalf("unexpected SIM1: %q", objects[0].SIM1)
	}
	if objects[0].SIM2 != "380671112233" {
		t.Fatalf("unexpected SIM2: %q", objects[0].SIM2)
	}
}

func TestPhoenixChannelInfoUsesTelephonNumPriority(t *testing.T) {
	row := phoenixChannelRow{
		Sim1Number: sql.NullString{String: "380671783262", Valid: true},
		Sim2Number: sql.NullString{String: "380676341887", Valid: true},
	}
	obj := &models.Object{}

	provider := &PhoenixDataProvider{}
	provider.applyChannelInfo(obj, row)

	if obj.SIM1 != "380671783262" {
		t.Fatalf("unexpected SIM1: %q", obj.SIM1)
	}
	if obj.SIM2 != "380676341887" {
		t.Fatalf("unexpected SIM2: %q", obj.SIM2)
	}
}
