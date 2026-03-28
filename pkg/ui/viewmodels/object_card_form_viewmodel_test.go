package viewmodels

import "testing"

type objectCardFormReferencesStub struct {
	typeIDs      map[string]int64
	regionIDs    map[string]int64
	ppkIDs       map[string]int64
	subServerMap map[string]string
}

func (s *objectCardFormReferencesStub) ObjectTypeID(label string) int64 {
	return s.typeIDs[label]
}

func (s *objectCardFormReferencesStub) RegionID(label string) int64 {
	return s.regionIDs[label]
}

func (s *objectCardFormReferencesStub) PPKID(label string) int64 {
	return s.ppkIDs[label]
}

func (s *objectCardFormReferencesStub) SubServerBind(label string) string {
	return s.subServerMap[label]
}

func TestObjectCardFormViewModel_Defaults(t *testing.T) {
	vm := NewObjectCardFormViewModel()
	defaults := vm.Defaults()

	if defaults.ChannelCode != 1 {
		t.Fatalf("unexpected default channel: %d", defaults.ChannelCode)
	}
	if !defaults.TestControlEnabled {
		t.Fatalf("expected test control enabled by default")
	}
	if defaults.TestIntervalMinRaw != "9" {
		t.Fatalf("unexpected default test interval: %q", defaults.TestIntervalMinRaw)
	}
	if defaults.RegionID != 1 {
		t.Fatalf("unexpected default region id: %d", defaults.RegionID)
	}
}

func TestObjectCardFormViewModel_BuildInput(t *testing.T) {
	vm := NewObjectCardFormViewModel()
	refs := &objectCardFormReferencesStub{
		typeIDs: map[string]int64{"Тип [2]": 2},
		regionIDs: map[string]int64{
			"Район [1]": 1,
		},
		ppkIDs: map[string]int64{"ППК [11]": 11},
		subServerMap: map[string]string{
			"A [a]": "a",
			"B [b]": "b",
		},
	}

	input, err := vm.BuildInput(
		ObjectCardFormSnapshot{
			ObjNRaw:            "101",
			ShortName:          "Obj",
			FullName:           "Obj Full",
			Address:            "Address",
			Phones:             "123",
			Contract:           "C",
			StartDate:          "28.03.2026",
			Location:           "Loc",
			Notes:              "N",
			GSMPhone1:          "111",
			GSMPhone2:          "222",
			GSMHiddenNRaw:      "999",
			ChannelLabel:       "1 - Автододзвон",
			TestControlEnabled: true,
			TestIntervalMinRaw: "9",
			ObjectTypeLabel:    "Тип [2]",
			RegionLabel:        "Район [1]",
			PPKLabel:           "ППК [11]",
			SubServerALabel:    "A [a]",
			SubServerBLabel:    "B [b]",
		},
		refs,
		map[string]int64{
			"1 - Автододзвон": 1,
			"5 - GPRS":        5,
		},
	)
	if err != nil {
		t.Fatalf("unexpected build input error: %v", err)
	}
	if input.ChannelCode != 1 {
		t.Fatalf("unexpected channel code: %d", input.ChannelCode)
	}
	if input.ObjTypeID != 2 {
		t.Fatalf("unexpected object type id: %d", input.ObjTypeID)
	}
	if input.PPKID != 11 {
		t.Fatalf("unexpected ppk id: %d", input.PPKID)
	}
	if input.SubServerA != "a" || input.SubServerB != "b" {
		t.Fatalf("unexpected subserver binds: %q / %q", input.SubServerA, input.SubServerB)
	}
}

func TestObjectCardFormViewModel_BuildInput_ChannelRequired(t *testing.T) {
	vm := NewObjectCardFormViewModel()
	refs := &objectCardFormReferencesStub{
		typeIDs:      map[string]int64{},
		regionIDs:    map[string]int64{},
		ppkIDs:       map[string]int64{},
		subServerMap: map[string]string{},
	}

	_, err := vm.BuildInput(
		ObjectCardFormSnapshot{ChannelLabel: "unknown"},
		refs,
		map[string]int64{"1 - Автододзвон": 1},
	)
	if err == nil {
		t.Fatalf("expected missing channel error")
	}
}
