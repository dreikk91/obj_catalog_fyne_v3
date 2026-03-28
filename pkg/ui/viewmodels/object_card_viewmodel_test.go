package viewmodels

import "testing"

func TestObjectCardViewModel_OnShortNameChanged_SyncsFullName(t *testing.T) {
	vm := NewObjectCardViewModel()
	vm.ResetNameSync("Alpha", "Alpha")

	fullName, changed := vm.OnShortNameChanged("Beta ")
	if !changed {
		t.Fatalf("expected full name sync to be applied")
	}
	if fullName != "Beta" {
		t.Fatalf("unexpected full name: %q", fullName)
	}
}

func TestObjectCardViewModel_OnShortNameChanged_DoesNotSyncWhenFullNameWasEdited(t *testing.T) {
	vm := NewObjectCardViewModel()
	vm.ResetNameSync("Alpha", "Custom Name")

	_, changed := vm.OnShortNameChanged("Beta")
	if changed {
		t.Fatalf("expected sync to be disabled after custom full name")
	}
}

func TestObjectCardViewModel_ValidateAndBuildCard_Channel5RequiresHiddenNumber(t *testing.T) {
	vm := NewObjectCardViewModel()

	_, err := vm.ValidateAndBuildCard(ObjectCardInput{
		ObjNRaw:       "1234",
		ShortName:     "Obj",
		FullName:      "Obj",
		ChannelCode:   5,
		ObjTypeID:     1,
		GSMHiddenNRaw: "",
	})
	if err == nil {
		t.Fatalf("expected validation error for hidden number")
	}
}

func TestObjectCardViewModel_ValidateAndBuildCard_ParsesAndBuilds(t *testing.T) {
	vm := NewObjectCardViewModel()

	card, err := vm.ValidateAndBuildCard(ObjectCardInput{
		ObjNRaw:            "1234",
		ShortName:          "Obj",
		FullName:           "Obj full",
		Address:            "Address",
		ChannelCode:        1,
		ObjTypeID:          2,
		ObjRegID:           3,
		PPKID:              4,
		TestControlEnabled: true,
		TestIntervalMinRaw: "9",
		SubServerA:         "A",
		SubServerB:         "B",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card.ObjN != 1234 {
		t.Fatalf("unexpected objn: %d", card.ObjN)
	}
	if card.TestIntervalMin != 9 {
		t.Fatalf("unexpected test interval: %d", card.TestIntervalMin)
	}
	if card.ObjTypeID != 2 {
		t.Fatalf("unexpected object type id: %d", card.ObjTypeID)
	}
}
