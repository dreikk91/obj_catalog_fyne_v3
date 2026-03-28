package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectCardLoadRefsStub struct{}

func (s objectCardLoadRefsStub) ObjectTypeLabelByID(id int64) string {
	if id == 10 {
		return "Тип 10"
	}
	return ""
}

func (s objectCardLoadRefsStub) RegionLabelByID(id int64) string {
	if id == 1 {
		return "Район 1"
	}
	if id == 2 {
		return "Район 2"
	}
	return ""
}

func (s objectCardLoadRefsStub) SubServerLabelByBind(bind string) string {
	if bind == "a" {
		return "SBS A"
	}
	if bind == "b" {
		return "SBS B"
	}
	return ""
}

func TestObjectCardLoadViewModel_BuildPresentation(t *testing.T) {
	vm := NewObjectCardLoadViewModel()
	card := contracts.AdminObjectCard{
		ObjN:               123,
		ShortName:          "short",
		FullName:           "full",
		Address:            "addr",
		Phones:             "phones",
		Contract:           "contract",
		StartDate:          "01.01.2026",
		Location:           "loc",
		Notes:              "notes",
		ChannelCode:        5,
		PPKID:              42,
		GSMPhone1:          "0501111111",
		GSMPhone2:          "0502222222",
		GSMHiddenN:         321,
		TestControlEnabled: true,
		TestIntervalMin:    15,
		ObjTypeID:          10,
		ObjRegID:           2,
		SubServerA:         "a",
		SubServerB:         "b",
	}
	p := vm.BuildPresentation(card, objectCardLoadRefsStub{}, map[int64]string{
		1: "1 - Автододзвон",
		5: "5 - GPRS",
	})

	if p.ObjNText != "123" || p.ChannelLabel != "5 - GPRS" || p.ChannelCode != 5 {
		t.Fatalf("unexpected channel/object presentation: %+v", p)
	}
	if p.GSMHiddenNText != "321" {
		t.Fatalf("unexpected hidden n text: %q", p.GSMHiddenNText)
	}
	if p.TestIntervalMinText != "15" {
		t.Fatalf("unexpected test interval: %q", p.TestIntervalMinText)
	}
	if p.ObjectTypeLabel != "Тип 10" || p.RegionLabel != "Район 2" {
		t.Fatalf("unexpected refs labels: %+v", p)
	}
	if p.SubServerALabel != "SBS A" || p.SubServerBLabel != "SBS B" {
		t.Fatalf("unexpected subserver labels: %+v", p)
	}
}

func TestObjectCardLoadViewModel_BuildPresentation_Fallbacks(t *testing.T) {
	vm := NewObjectCardLoadViewModel()
	card := contracts.AdminObjectCard{
		ObjN:            1,
		ChannelCode:     99,
		GSMHiddenN:      0,
		TestIntervalMin: 0,
		ObjRegID:        0,
	}
	p := vm.BuildPresentation(card, objectCardLoadRefsStub{}, map[int64]string{
		1: "1 - Автододзвон",
	})

	if p.ChannelCode != 1 || p.ChannelLabel != "1 - Автододзвон" {
		t.Fatalf("unexpected channel fallback: %+v", p)
	}
	if p.GSMHiddenNText != "" {
		t.Fatalf("hidden n must be empty by default: %q", p.GSMHiddenNText)
	}
	if p.TestIntervalMinText != "9" {
		t.Fatalf("unexpected interval fallback: %q", p.TestIntervalMinText)
	}
	if p.RegionLabel != "Район 1" {
		t.Fatalf("unexpected region fallback label: %q", p.RegionLabel)
	}
}
