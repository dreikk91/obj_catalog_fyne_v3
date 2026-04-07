package viewmodels

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

type alarmListUseCaseStub struct {
	alarms []models.Alarm
}

func (s *alarmListUseCaseStub) FetchAlarms() []models.Alarm {
	return append([]models.Alarm(nil), s.alarms...)
}

func TestAlarmListViewModel_LoadAlarms(t *testing.T) {
	vm := NewAlarmListViewModel()
	alarms := vm.LoadAlarms(&alarmListUseCaseStub{
		alarms: []models.Alarm{{ID: 1}, {ID: 2}},
	})

	if len(alarms) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(alarms))
	}
}

func TestAlarmListViewModel_BuildRefreshOutput(t *testing.T) {
	vm := NewAlarmListViewModel()
	input := AlarmRefreshInput{
		Alarms: []models.Alarm{
			{ID: 10, Type: models.AlarmFire, IsProcessed: false},
			{ID: 11, ObjectID: ids.PhoenixObjectIDNamespaceStart + 11, Type: models.AlarmFault, IsProcessed: false},
		},
		LastKnownIDs: map[int]struct{}{
			11: {},
		},
	}

	out := vm.BuildRefreshOutput(input)
	if out.Total != 2 {
		t.Fatalf("unexpected total: %d", out.Total)
	}
	if out.CriticalCount != 2 {
		t.Fatalf("unexpected critical count: %d", out.CriticalCount)
	}
	if !out.HasNewCritical || out.NewCritical.ID != 10 {
		t.Fatalf("expected new critical alarm, got %+v", out)
	}
	if _, ok := out.KnownIDs[10]; !ok {
		t.Fatalf("known ids must include alarm 10")
	}
	if out.CountAll != 2 || out.CountBridge != 1 || out.CountPhoenix != 1 || out.CountCASL != 0 {
		t.Fatalf("unexpected source counters: all=%d bridge=%d phoenix=%d casl=%d", out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}

func TestAlarmListViewModel_BuildRefreshOutput_BySource(t *testing.T) {
	vm := NewAlarmListViewModel()
	caslObjectID := ids.CASLObjectIDNamespaceStart + 100
	phoenixObjectID := ids.PhoenixObjectIDNamespaceStart + 50

	out := vm.BuildRefreshOutput(AlarmRefreshInput{
		Alarms: []models.Alarm{
			{ID: 1, ObjectID: 22, Type: models.AlarmFire, IsProcessed: false},
			{ID: 2, ObjectID: phoenixObjectID, Type: models.AlarmFire, IsProcessed: false},
			{ID: 3, ObjectID: caslObjectID, Type: models.AlarmFire, IsProcessed: false},
			{ID: 4, ObjectID: caslObjectID, Type: models.AlarmFault, IsProcessed: false},
		},
		LastKnownIDs:   map[int]struct{}{},
		SelectedSource: ObjectSourceCASL,
	})

	if len(out.FilteredAlarms) != 2 {
		t.Fatalf("expected 2 CASL alarms in filtered list, got %d", len(out.FilteredAlarms))
	}
	if out.CountAll != 4 || out.CountBridge != 1 || out.CountPhoenix != 1 || out.CountCASL != 2 {
		t.Fatalf("unexpected source counters: all=%d bridge=%d phoenix=%d casl=%d", out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
	}
}
