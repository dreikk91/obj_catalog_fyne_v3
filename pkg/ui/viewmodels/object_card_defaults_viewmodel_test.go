package viewmodels

import "testing"

type objectCardDefaultsRefsStub struct{}

func (s objectCardDefaultsRefsStub) ObjectTypeLabelByID(id int64) string {
	if id == 7 {
		return "Тип 7"
	}
	return ""
}

func (s objectCardDefaultsRefsStub) RegionLabelByID(id int64) string {
	if id == 3 {
		return "Район 3"
	}
	return ""
}

func (s objectCardDefaultsRefsStub) SubServerLabelByBind(bind string) string {
	if bind == "gprs-a" {
		return "SBS A"
	}
	return ""
}

func TestObjectCardDefaultsViewModel_BuildPresentation(t *testing.T) {
	vm := NewObjectCardDefaultsViewModel()
	defaults := ObjectCardFormDefaults{
		ChannelCode:        5,
		ObjectTypeID:       7,
		RegionID:           3,
		SubServerBind:      "gprs-a",
		TestControlEnabled: true,
		TestIntervalMinRaw: "11",
	}

	p := vm.BuildPresentation(defaults, objectCardDefaultsRefsStub{}, map[int64]string{
		1: "1 - Автододзвон",
		5: "5 - GPRS",
	}, "28.03.2026")

	if p.StartDateText != "28.03.2026" {
		t.Fatalf("unexpected start date: %q", p.StartDateText)
	}
	if p.ChannelCode != 5 || p.ChannelLabel != "5 - GPRS" {
		t.Fatalf("unexpected channel: %+v", p)
	}
	if p.TestIntervalMinText != "11" || !p.TestControlEnabled {
		t.Fatalf("unexpected test controls: %+v", p)
	}
	if p.ObjectTypeLabel != "Тип 7" || p.RegionLabel != "Район 3" {
		t.Fatalf("unexpected reference labels: %+v", p)
	}
	if p.SubServerALabel != "SBS A" || p.SubServerBLabel != "SBS A" {
		t.Fatalf("unexpected subserver labels: %+v", p)
	}
	if p.ShortName != "" || p.FullName != "" || p.ObjNText != "" {
		t.Fatalf("defaults must clear text fields: %+v", p)
	}
}

func TestObjectCardDefaultsViewModel_BuildPresentation_ChannelFallback(t *testing.T) {
	vm := NewObjectCardDefaultsViewModel()
	defaults := ObjectCardFormDefaults{ChannelCode: 99}

	p := vm.BuildPresentation(defaults, objectCardDefaultsRefsStub{}, map[int64]string{
		1: "1 - Автододзвон",
	}, "01.01.2026")

	if p.ChannelCode != 1 || p.ChannelLabel != "1 - Автододзвон" {
		t.Fatalf("unexpected channel fallback: %+v", p)
	}
}
