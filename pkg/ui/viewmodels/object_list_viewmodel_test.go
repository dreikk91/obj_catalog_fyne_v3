package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

type objectListUseCaseStub struct {
	objects []models.Object
}

func (s *objectListUseCaseStub) FetchObjects() []models.Object {
	return append([]models.Object(nil), s.objects...)
}

func TestObjectListViewModel_LoadObjects(t *testing.T) {
	vm := NewObjectListViewModel()
	objects := vm.LoadObjects(&objectListUseCaseStub{
		objects: []models.Object{{ID: 1}, {ID: 2}},
	})

	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
}

func TestObjectListViewModel_NormalizeFilter(t *testing.T) {
	vm := NewObjectListViewModel()
	if got := vm.NormalizeFilter("Є тривоги (12)"); got != "Є тривоги" {
		t.Fatalf("unexpected normalized filter: %q", got)
	}
}

func TestObjectListViewModel_ApplyFilters(t *testing.T) {
	vm := NewObjectListViewModel()
	all := []models.Object{
		{ID: 1, Name: "Альфа", Status: models.StatusNormal, GuardState: 1, IsConnState: 1},
		{ID: caslObjectIDNamespaceStart + 2, Name: "Бета", Status: models.StatusFire, GuardState: 1, IsConnState: 1},
		{ID: 3, Name: "Гамма", Status: models.StatusNormal, GuardState: 0, IsConnState: 0},
	}

	out := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:           all,
		Query:                "",
		CurrentFilter:        "Є тривоги",
		PreviousSelectedID:   1,
		HadPreviousSelection: true,
		LastNotifiedID:       1,
		HasNotifiedSelection: true,
	})

	if out.CountAll != 3 || out.CountAlarm != 1 || out.CountDisarmed != 1 {
		t.Fatalf("unexpected counters: %+v", out)
	}
	if len(out.Filtered) != 1 || out.Filtered[0].ID != caslObjectIDNamespaceStart+2 {
		t.Fatalf("unexpected filtered result: %+v", out.Filtered)
	}
	if out.NewSelectedRow != 0 || !out.HasSelectedObject || out.SelectedObject.ID != caslObjectIDNamespaceStart+2 {
		t.Fatalf("unexpected selected object: %+v", out)
	}
	if !out.ShouldNotifySelection {
		t.Fatalf("must notify selection on auto-selected different object")
	}
	if out.CountCASL != 1 || out.CountBridge != 2 {
		t.Fatalf("unexpected source counters: bridge=%d casl=%d", out.CountBridge, out.CountCASL)
	}
}

func TestObjectListViewModel_BuildFilterOptions(t *testing.T) {
	vm := NewObjectListViewModel()
	opts := vm.BuildFilterOptions(10, 2, 3, 4)
	if len(opts) != 4 {
		t.Fatalf("expected 4 options, got %d", len(opts))
	}
	if opts[0] != "Всі (10)" {
		t.Fatalf("unexpected option: %q", opts[0])
	}
}

func TestObjectListViewModel_ApplyFilters_BySourceAndSIMSearch(t *testing.T) {
	vm := NewObjectListViewModel()
	all := []models.Object{
		{ID: 10, Name: "Bridge One", SIM1: "+380501112233"},
		{ID: caslObjectIDNamespaceStart + 10, Name: "CASL One", SIM1: "+380671234567"},
	}

	outBySource := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: "Всі",
		CurrentSource: ObjectSourceCASL,
	})
	if len(outBySource.Filtered) != 1 || !IsCASLObjectID(outBySource.Filtered[0].ID) {
		t.Fatalf("expected only CASL objects, got %+v", outBySource.Filtered)
	}

	outBySIM := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: "Всі",
		CurrentSource: ObjectSourceAll,
		Query:         "sim:1234567",
	})
	if len(outBySIM.Filtered) != 1 || outBySIM.Filtered[0].Name != "CASL One" {
		t.Fatalf("expected CASL One by sim search, got %+v", outBySIM.Filtered)
	}

	outBySourceToken := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: "Всі",
		CurrentSource: ObjectSourceAll,
		Query:         "src:casl",
	})
	if len(outBySourceToken.Filtered) != 1 || !IsCASLObjectID(outBySourceToken.Filtered[0].ID) {
		t.Fatalf("expected src:casl to filter only CASL objects, got %+v", outBySourceToken.Filtered)
	}
}
