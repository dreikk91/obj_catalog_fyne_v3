package viewmodels

import (
	"testing"

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
			{ID: 11, Type: models.AlarmFault, IsProcessed: false},
		},
		LastKnownIDs: map[int]struct{}{
			11: {},
		},
	}

	out := vm.BuildRefreshOutput(input)
	if out.Total != 2 {
		t.Fatalf("unexpected total: %d", out.Total)
	}
	if out.FireCount != 1 {
		t.Fatalf("unexpected fire count: %d", out.FireCount)
	}
	if !out.HasNewCritical || out.NewCritical.ID != 10 {
		t.Fatalf("expected new critical alarm, got %+v", out)
	}
	if _, ok := out.KnownIDs[10]; !ok {
		t.Fatalf("known ids must include alarm 10")
	}
}
