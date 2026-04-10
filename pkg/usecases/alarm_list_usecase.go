package usecases

import (
	"slices"

	"obj_catalog_fyne_v3/pkg/models"
)

// AlarmListRepository описує мінімальне джерело тривог для use case.
type AlarmListRepository interface {
	GetAlarms() []models.Alarm
}

// AlarmListUseCase інкапсулює сценарій отримання активних тривог.
type AlarmListUseCase struct {
	repository AlarmListRepository
}

func NewAlarmListUseCase(repository AlarmListRepository) *AlarmListUseCase {
	return &AlarmListUseCase{repository: repository}
}

// FetchAlarms повертає копію масиву тривог для подальшої обробки у ViewModel.
func (uc *AlarmListUseCase) FetchAlarms() []models.Alarm {
	if uc == nil || uc.repository == nil {
		return nil
	}
	alarms := uc.repository.GetAlarms()
	return slices.Clone(alarms)
}
