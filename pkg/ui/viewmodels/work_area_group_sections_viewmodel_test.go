package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestWorkAreaGroupSectionsViewModel_BuildZoneSections(t *testing.T) {
	t.Parallel()

	vm := NewWorkAreaGroupSectionsViewModel()
	object := &models.Object{
		Groups: []models.ObjectGroup{
			{ID: "group:1", Number: 1, Name: "Офіс", StateText: "ПІД ОХОРОНОЮ"},
			{ID: "group:2", Number: 2, Name: "Склад", StateText: "ЗНЯТО"},
		},
	}
	zones := []models.Zone{
		{Number: 2, Name: "Двері", SensorType: "Магніт", GroupID: "group:1", GroupNumber: 1},
		{Number: 1, Name: "Хол", SensorType: "Рух", GroupID: "group:1", GroupNumber: 1},
		{Number: 3, Name: "Комора", SensorType: "Дим", GroupID: "group:2", GroupNumber: 2},
	}

	sections := vm.BuildZoneSections(object, zones)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if got := vm.FormatSectionTitle(sections[0].Group); got != "Група 1 | Офіс | ПІД ОХОРОНОЮ" {
		t.Fatalf("unexpected first section title: %q", got)
	}
	if len(sections[0].Zones) != 2 || sections[0].Zones[0].Number != 1 || sections[0].Zones[1].Number != 2 {
		t.Fatalf("unexpected first section zones: %+v", sections[0].Zones)
	}
	if len(sections[1].Zones) != 1 || sections[1].Zones[0].Name != "Комора" {
		t.Fatalf("unexpected second section zones: %+v", sections[1].Zones)
	}
}

func TestWorkAreaGroupSectionsViewModel_BuildContactSections(t *testing.T) {
	t.Parallel()

	vm := NewWorkAreaGroupSectionsViewModel()
	object := &models.Object{
		Groups: []models.ObjectGroup{
			{ID: "group:1", Number: 1, Name: "Офіс", StateText: "ПІД ОХОРОНОЮ"},
			{ID: "group:2", Number: 2, Name: "Склад", StateText: "ЗНЯТО"},
		},
	}
	contacts := []models.Contact{
		{Name: "Іван", Priority: 2, GroupID: "group:1", GroupNumber: 1},
		{Name: "Петро", Priority: 1, GroupID: "group:1", GroupNumber: 1},
		{Name: "Марія", Priority: 1, GroupID: "group:2", GroupNumber: 2},
	}

	sections := vm.BuildContactSections(object, contacts)
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if len(sections[0].Contacts) != 2 || sections[0].Contacts[0].Name != "Петро" {
		t.Fatalf("unexpected first section contacts: %+v", sections[0].Contacts)
	}
	if len(sections[1].Contacts) != 1 || sections[1].Contacts[0].Name != "Марія" {
		t.Fatalf("unexpected second section contacts: %+v", sections[1].Contacts)
	}
}

func TestWorkAreaGroupSectionsViewModel_ShouldUseGroupedLayout(t *testing.T) {
	t.Parallel()

	vm := NewWorkAreaGroupSectionsViewModel()
	object := &models.Object{
		Groups: []models.ObjectGroup{
			{ID: "group:1", Number: 1},
			{ID: "group:2", Number: 2},
		},
	}
	if !vm.ShouldUseGroupedZones(object, nil) {
		t.Fatal("expected grouped zones layout for multi-group object")
	}
	if !vm.ShouldUseGroupedContacts(object, nil) {
		t.Fatal("expected grouped contacts layout for multi-group object")
	}
	if vm.ShouldUseGroupedZones(&models.Object{}, []models.Zone{{Number: 1}}) {
		t.Fatal("did not expect grouped zones layout for plain bridge object")
	}
}
