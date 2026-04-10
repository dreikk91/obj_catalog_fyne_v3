package viewmodels

import (
	"slices"

	"obj_catalog_fyne_v3/pkg/models"
)

// AlarmListUseCase описує мінімальний use case для завантаження тривог.
type AlarmListUseCase interface {
	FetchAlarms() []models.Alarm
}

// AlarmRefreshInput описує вхідні дані для оновлення стану панелі тривог.
type AlarmRefreshInput struct {
	Alarms         []models.Alarm
	LastKnownIDs   map[int]struct{}
	SelectedSource string
}

// AlarmRefreshOutput описує результат обробки списку тривог для UI.
type AlarmRefreshOutput struct {
	CurrentAlarms  []models.Alarm
	FilteredAlarms []models.Alarm
	KnownIDs       map[int]struct{}
	Total          int
	CriticalCount  int
	CountAll       int
	CountBridge    int
	CountPhoenix   int
	CountCASL      int
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
	return slices.Clone(alarms)
}

func (vm *AlarmListViewModel) BuildRefreshOutput(input AlarmRefreshInput) AlarmRefreshOutput {
	out := AlarmRefreshOutput{
		CurrentAlarms:  slices.Clone(input.Alarms),
		FilteredAlarms: make([]models.Alarm, 0, len(input.Alarms)),
		KnownIDs:       make(map[int]struct{}, len(input.Alarms)),
		Total:          len(input.Alarms),
	}

	for i := range input.Alarms {
		alarm := input.Alarms[i]
		source := ObjectSourceByID(alarm.ObjectID)
		out.CountAll++
		switch source {
		case ObjectSourceCASL:
			out.CountCASL++
		case ObjectSourcePhoenix:
			out.CountPhoenix++
		default:
			out.CountBridge++
		}
		if sourceMatchesFilter(source, input.SelectedSource) {
			out.FilteredAlarms = append(out.FilteredAlarms, alarm)
		}
		if alarm.IsCritical() && !alarm.IsProcessed {
			out.CriticalCount++
		}
		if _, ok := input.LastKnownIDs[alarm.ID]; !ok {
			if !out.HasNewCritical && alarm.IsCritical() && !alarm.IsProcessed {
				out.NewCritical = alarm
				out.HasNewCritical = true
			}
		}
		out.KnownIDs[alarm.ID] = struct{}{}
	}

	return out
}
