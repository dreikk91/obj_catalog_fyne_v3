package viewmodels

import "testing"

func TestObjectAdditionalTabViewModel_AddressFromObjectTab(t *testing.T) {
	vm := NewObjectAdditionalTabViewModel()

	address, ok := vm.AddressFromObjectTab(func() string { return "  Львів, вул. Зелена 1  " })
	if !ok {
		t.Fatalf("expected address from callback")
	}
	if address != "Львів, вул. Зелена 1" {
		t.Fatalf("unexpected address: %q", address)
	}
}

func TestObjectAdditionalTabViewModel_RequireLookupAddress(t *testing.T) {
	vm := NewObjectAdditionalTabViewModel()

	if _, err := vm.RequireLookupAddress("   "); err == nil {
		t.Fatalf("expected error for empty address")
	}
	address, err := vm.RequireLookupAddress("  Київ, Хрещатик 1 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if address != "Київ, Хрещатик 1" {
		t.Fatalf("unexpected normalized address: %q", address)
	}
}

func TestObjectAdditionalTabViewModel_CachedDistrictHintsForAddress(t *testing.T) {
	vm := NewObjectAdditionalTabViewModel()
	vm.RememberGeocode("Львів, вул. Шевченка 1", []string{"Шевченківський", "Львів"})

	hints, ok := vm.CachedDistrictHintsForAddress(" львів, вул. шевченка 1 ")
	if !ok {
		t.Fatalf("expected cached hints")
	}
	if len(hints) != 2 || hints[0] != "Шевченківський" {
		t.Fatalf("unexpected hints: %+v", hints)
	}
	// Перевіряємо, що повертається копія.
	hints[0] = "Changed"
	hints2, ok := vm.CachedDistrictHintsForAddress("Львів, вул. Шевченка 1")
	if !ok || hints2[0] != "Шевченківський" {
		t.Fatalf("cached hints must be immutable copy: %+v", hints2)
	}
}

func TestObjectAdditionalTabViewModel_BuildCoordinates(t *testing.T) {
	vm := NewObjectAdditionalTabViewModel()
	coords := vm.BuildCoordinates(" 49.84 ", " 24.03 ")
	if coords.Latitude != "49.84" || coords.Longitude != "24.03" {
		t.Fatalf("unexpected coords: %+v", coords)
	}
}
