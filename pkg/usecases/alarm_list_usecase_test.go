package usecases

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

type alarmListRepoStub struct {
	alarms []models.Alarm
}

func (s *alarmListRepoStub) GetAlarms() []models.Alarm {
	return append([]models.Alarm(nil), s.alarms...)
}

func TestAlarmListUseCase_FetchAlarmsReturnsCopy(t *testing.T) {
	stub := &alarmListRepoStub{
		alarms: []models.Alarm{
			{ID: 1, ObjectID: 100},
			{ID: 2, ObjectID: 101},
		},
	}
	uc := NewAlarmListUseCase(stub)

	got := uc.FetchAlarms()
	if len(got) != 2 {
		t.Fatalf("expected 2 alarms, got %d", len(got))
	}
	got[0].ObjectID = 999
	if stub.alarms[0].ObjectID != 100 {
		t.Fatalf("use case must return copy, repository data changed")
	}
}

func TestAlarmListUseCase_FetchAlarmsNilRepository(t *testing.T) {
	uc := NewAlarmListUseCase(nil)
	got := uc.FetchAlarms()
	if len(got) != 0 {
		t.Fatalf("expected empty result for nil repository, got %d", len(got))
	}
}
