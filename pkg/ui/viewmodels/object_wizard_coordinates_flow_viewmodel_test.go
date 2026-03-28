package viewmodels

import "testing"

func TestObjectWizardCoordinatesFlowViewModel_PreparePickerInput(t *testing.T) {
	vm := NewObjectWizardCoordinatesFlowViewModel()

	lat, lon := vm.PreparePickerInput(" 50.4501 ", " 30.5234 ")
	if lat != "50.4501" || lon != "30.5234" {
		t.Fatalf("unexpected picker input: %q, %q", lat, lon)
	}
}

func TestObjectWizardCoordinatesFlowViewModel_ApplyPicked(t *testing.T) {
	vm := NewObjectWizardCoordinatesFlowViewModel()

	out := vm.ApplyPicked(" 50.4501 ", " 30.5234 ")
	if out.Latitude != "50.4501" || out.Longitude != "30.5234" {
		t.Fatalf("unexpected coords: %+v", out)
	}
	if out.Status != "Координати вибрано на карті" {
		t.Fatalf("unexpected status: %q", out.Status)
	}
}

func TestObjectWizardCoordinatesFlowViewModel_Clear(t *testing.T) {
	vm := NewObjectWizardCoordinatesFlowViewModel()

	out := vm.Clear()
	if out.Latitude != "" || out.Longitude != "" {
		t.Fatalf("coordinates must be cleared: %+v", out)
	}
	if out.Status != "Координати очищено" {
		t.Fatalf("unexpected status: %q", out.Status)
	}
}
