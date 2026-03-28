package usecases

import "obj_catalog_fyne_v3/pkg/models"

// EventLogRepository описує мінімальне джерело подій для use case журналу.
type EventLogRepository interface {
	GetEvents() []models.Event
}

// EventLogUseCase інкапсулює сценарій отримання подій журналу.
type EventLogUseCase struct {
	repository EventLogRepository
}

func NewEventLogUseCase(repository EventLogRepository) *EventLogUseCase {
	return &EventLogUseCase{repository: repository}
}

// FetchEvents повертає копію списку подій для подальшої обробки у ViewModel.
func (uc *EventLogUseCase) FetchEvents() []models.Event {
	if uc == nil || uc.repository == nil {
		return nil
	}
	events := uc.repository.GetEvents()
	return append([]models.Event(nil), events...)
}
