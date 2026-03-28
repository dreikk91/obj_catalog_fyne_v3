package viewmodels

import "obj_catalog_fyne_v3/pkg/models"

// AlarmListUseCase описує мінімальний use case для завантаження тривог.
type AlarmListUseCase interface {
	FetchAlarms() []models.Alarm
}

// AlarmRefreshInput описує вхідні дані для оновлення стану панелі тривог.
type AlarmRefreshInput struct {
	Alarms       []models.Alarm
	LastKnownIDs map[int]struct{}
}

// AlarmRefreshOutput описує результат обробки списку тривог для UI.
type AlarmRefreshOutput struct {
	CurrentAlarms  []models.Alarm
	KnownIDs       map[int]struct{}
	Total          int
	FireCount      int
	NewCritical    models.Alarm
	HasNewCritical bool
}

// AlarmListViewModel інкапсулює обчислення стану панелі тривог.
type AlarmListViewModel struct{}

func NewAlarmListViewModel() *AlarmListViewModel {
	return &AlarmListViewModel{}
}

func (vm *AlarmListViewModel) LoadAlarms(useCase AlarmListUseCase) []models.Alarm {
	if useCase == nil {
		return nil
	}
	alarms := useCase.FetchAlarms()
	return append([]models.Alarm(nil), alarms...)
}

func (vm *AlarmListViewModel) BuildRefreshOutput(input AlarmRefreshInput) AlarmRefreshOutput {
	out := AlarmRefreshOutput{
		CurrentAlarms: append([]models.Alarm(nil), input.Alarms...),
		KnownIDs:      make(map[int]struct{}, len(input.Alarms)),
		Total:         len(input.Alarms),
	}

	for i := range input.Alarms {
		alarm := input.Alarms[i]
		if alarm.Type == models.AlarmFire && !alarm.IsProcessed {
			out.FireCount++
		}
		if _, ok := input.LastKnownIDs[alarm.ID]; !ok {
			if !out.HasNewCritical && alarm.Type == models.AlarmFire && !alarm.IsProcessed {
				out.NewCritical = alarm
				out.HasNewCritical = true
			}
		}
		out.KnownIDs[alarm.ID] = struct{}{}
	}

	return out
}
