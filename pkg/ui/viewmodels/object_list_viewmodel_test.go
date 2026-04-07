package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/ids"
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

func TestNormalizeObjectListFilter(t *testing.T) {
	if got := NormalizeObjectListFilter(FilterAlarm + " (12)"); got != FilterAlarm {
		t.Fatalf("unexpected normalized filter: %q", got)
	}
	if got := NormalizeObjectListFilter("unknown"); got != FilterAll {
		t.Fatalf("unexpected fallback filter: %q", got)
	}
}

func TestObjectListViewModel_ApplyFilters(t *testing.T) {
	vm := NewObjectListViewModel()
	all := []models.Object{
		{ID: 1, Name: "Альфа", Status: models.StatusNormal, GuardState: 1, IsConnState: 1},
		{ID: ids.PhoenixObjectIDNamespaceStart + 10, DisplayNumber: "L00028", Name: "Phoenix", Status: models.StatusNormal, GuardState: 1, IsConnState: 1},
		{ID: ids.CASLObjectIDNamespaceStart + 2, Name: "Бета", Status: models.StatusFire, GuardState: 1, IsConnState: 1},
		{ID: 3, Name: "Гамма", Status: models.StatusNormal, GuardState: 0, IsConnState: 0},
	}

	out := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:           all,
		Query:                "",
		CurrentFilter:        FilterAlarm,
		PreviousSelectedID:   1,
		HadPreviousSelection: true,
		LastNotifiedID:       1,
		HasNotifiedSelection: true,
	})

	if out.CountAll != 4 || out.CountAlarm != 1 || out.CountMonitoringOff != 1 || out.CountDebug != 0 {
		t.Fatalf("unexpected counters: %+v", out)
	}
	if len(out.Filtered) != 1 || out.Filtered[0].ID != ids.CASLObjectIDNamespaceStart+2 {
		t.Fatalf("unexpected filtered result: %+v", out.Filtered)
	}
	if out.NewSelectedRow != 0 || !out.HasSelectedObject || out.SelectedObject.ID != ids.CASLObjectIDNamespaceStart+2 {
		t.Fatalf("unexpected selected object: %+v", out)
	}
	if !out.ShouldNotifySelection {
		t.Fatalf("must notify selection on auto-selected different object")
	}
	if out.CountCASL != 1 || out.CountBridge != 2 || out.CountPhoenix != 1 {
		t.Fatalf("unexpected source counters: bridge=%d phoenix=%d casl=%d", out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}

func TestObjectListViewModel_BuildFilterOptions(t *testing.T) {
	vm := NewObjectListViewModel()
	opts := vm.BuildFilterOptions(10, 2, 3, 4, 1)
	if len(opts) != 5 {
		t.Fatalf("expected 5 options, got %d", len(opts))
	}
	if opts[0] != FilterAll+" (10)" {
		t.Fatalf("unexpected option: %q", opts[0])
	}
	if opts[3] != "Знято зі спостереження (4)" {
		t.Fatalf("unexpected monitoring-off option: %q", opts[3])
	}
	if opts[4] != "В режимі налагодження (1)" {
		t.Fatalf("unexpected debug option: %q", opts[4])
	}
}

func TestObjectListViewModel_ApplyFilters_BySourceAndSIMSearch(t *testing.T) {
	vm := NewObjectListViewModel()
	all := []models.Object{
		{ID: 10, Name: "Bridge One", SIM1: "+380501112233"},
		{ID: ids.PhoenixObjectIDNamespaceStart + 20, DisplayNumber: "L00028", Name: "Phoenix One", SIM1: "+380661234567"},
		{ID: ids.CASLObjectIDNamespaceStart + 10, Name: "CASL One", SIM1: "+380671234567"},
	}

	outBySource := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterAll,
		CurrentSource: ObjectSourceCASL,
	})
	if len(outBySource.Filtered) != 1 || !ids.IsCASLObjectID(outBySource.Filtered[0].ID) {
		t.Fatalf("expected only CASL objects, got %+v", outBySource.Filtered)
	}

	outBySIM := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterAll,
		CurrentSource: ObjectSourceAll,
		Query:         "sim:671234567",
	})
	if len(outBySIM.Filtered) != 1 || outBySIM.Filtered[0].Name != "CASL One" {
		t.Fatalf("expected CASL One by sim search, got %+v", outBySIM.Filtered)
	}

	outBySourceToken := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterAll,
		CurrentSource: ObjectSourceAll,
		Query:         "src:casl",
	})
	if len(outBySourceToken.Filtered) != 1 || !ids.IsCASLObjectID(outBySourceToken.Filtered[0].ID) {
		t.Fatalf("expected src:casl to filter only CASL objects, got %+v", outBySourceToken.Filtered)
	}

	outByPhoenixSource := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterAll,
		CurrentSource: ObjectSourcePhoenix,
	})
	if len(outByPhoenixSource.Filtered) != 1 || !ids.IsPhoenixObjectID(outByPhoenixSource.Filtered[0].ID) {
		t.Fatalf("expected only Phoenix objects, got %+v", outByPhoenixSource.Filtered)
	}

	outByDisplayNumber := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterAll,
		CurrentSource: ObjectSourceAll,
		Query:         "L00028",
	})
	if len(outByDisplayNumber.Filtered) != 1 || outByDisplayNumber.Filtered[0].Name != "Phoenix One" {
		t.Fatalf("expected Phoenix One by display number search, got %+v", outByDisplayNumber.Filtered)
	}
}

func TestObjectListViewModel_ApplyFilters_MonitoringOffAndDebug(t *testing.T) {
	vm := NewObjectListViewModel()
	all := []models.Object{
		{ID: 10, Name: "Bridge Off", GuardState: 0, IsConnState: 1},
		{ID: 11, Name: "Bridge Debug", GuardState: 1, IsConnState: 1, BlockedArmedOnOff: 2},
		{ID: ids.PhoenixObjectIDNamespaceStart + 20, Name: "Phoenix Blocked", GuardState: 1, IsConnState: 1, BlockedArmedOnOff: 1},
		{ID: ids.PhoenixObjectIDNamespaceStart + 21, Name: "Phoenix Stand", GuardState: 1, IsConnState: 1, BlockedArmedOnOff: 2},
		{ID: ids.PhoenixObjectIDNamespaceStart + 22, Name: "Phoenix Disarmed", GuardState: 0, IsConnState: 1},
		{ID: ids.CASLObjectIDNamespaceStart + 30, Name: "CASL Blocked", GuardState: 0, IsConnState: 1, BlockedArmedOnOff: 1},
	}

	monitoringOff := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterMonitoringOff,
		CurrentSource: ObjectSourceAll,
	})
	if len(monitoringOff.Filtered) != 3 {
		t.Fatalf("expected 3 monitoring-off objects, got %+v", monitoringOff.Filtered)
	}
	if monitoringOff.CountMonitoringOff != 3 {
		t.Fatalf("unexpected monitoring-off count: %d", monitoringOff.CountMonitoringOff)
	}

	debug := vm.ApplyFilters(ObjectListFilterInput{
		AllObjects:    all,
		CurrentFilter: FilterDebug,
		CurrentSource: ObjectSourceAll,
	})
	if len(debug.Filtered) != 2 {
		t.Fatalf("expected 2 debug objects, got %+v", debug.Filtered)
	}
	if debug.CountDebug != 2 {
		t.Fatalf("unexpected debug count: %d", debug.CountDebug)
	}
}
