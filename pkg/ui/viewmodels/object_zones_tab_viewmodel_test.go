package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestObjectZonesTabViewModel_SelectAndEffectiveZoneNumber(t *testing.T) {
	vm := NewObjectZonesTabViewModel()
	vm.SetItems([]contracts.AdminObjectZone{
		{ID: 1, ZoneNumber: 0, Description: "A"},
		{ID: 2, ZoneNumber: 5, Description: "B"},
	})

	if vm.EffectiveZoneNumberAt(0) != 1 {
		t.Fatalf("unexpected effective zone for first row")
	}
	if vm.EffectiveZoneNumberAt(1) != 5 {
		t.Fatalf("unexpected effective zone for second row")
	}
	if !vm.SelectByTableRow(2) {
		t.Fatalf("expected selection by row to succeed")
	}
	if got, ok := vm.SelectedZoneNumber(); !ok || got != 5 {
		t.Fatalf("unexpected selected zone: %d, ok=%v", got, ok)
	}
}

func TestObjectZonesTabViewModel_SelectZoneByNumber(t *testing.T) {
	vm := NewObjectZonesTabViewModel()
	vm.SetItems([]contracts.AdminObjectZone{
		{ID: 1, ZoneNumber: 10},
		{ID: 2, ZoneNumber: 20},
	})

	if !vm.SelectZoneByNumber(20) {
		t.Fatalf("expected select by number")
	}
	if vm.SelectedRow() != 1 {
		t.Fatalf("unexpected selected row: %d", vm.SelectedRow())
	}
}

func TestObjectZonesTabViewModel_NextZoneNumberForAdd(t *testing.T) {
	vm := NewObjectZonesTabViewModel()
	vm.SetItems([]contracts.AdminObjectZone{
		{ZoneNumber: 1},
		{ZoneNumber: 4},
	})

	if got := vm.NextZoneNumberForAdd(); got != 5 {
		t.Fatalf("unexpected next zone without selected row: %d", got)
	}
	vm.SelectByTableRow(1)
	if got := vm.NextZoneNumberForAdd(); got != 2 {
		t.Fatalf("unexpected next zone from selection: %d", got)
	}
}

func TestObjectZonesTabViewModel_BuildZoneForCreate(t *testing.T) {
	vm := NewObjectZonesTabViewModel()
	zone, err := vm.BuildZoneForCreate(3, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if zone.ZoneNumber != 3 || zone.Description != "Шлейф 3" || zone.ZoneType != 1 {
		t.Fatalf("unexpected created zone: %+v", zone)
	}
}

func TestObjectZonesTabViewModel_PrepareSelectedZoneForSave(t *testing.T) {
	vm := NewObjectZonesTabViewModel()
	vm.SetItems([]contracts.AdminObjectZone{
		{ID: 7, ZoneNumber: 0, Description: "  old  "},
	})
	vm.SelectByTableRow(1)

	zone, zoneNumber, ok := vm.PrepareSelectedZoneForSave("  New Name ")
	if !ok {
		t.Fatalf("expected prepare success")
	}
	if zone.ID != 7 {
		t.Fatalf("unexpected zone id: %d", zone.ID)
	}
	if zoneNumber != 1 || zone.ZoneNumber != 1 {
		t.Fatalf("unexpected zone number: %d, zone=%+v", zoneNumber, zone)
	}
	if zone.Description != "New Name" {
		t.Fatalf("description must be trimmed, got %q", zone.Description)
	}
}
