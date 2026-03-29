package viewmodels

import (
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

// EventLogUseCase описує мінімальний use case для завантаження подій журналу.
type EventLogUseCase interface {
	FetchEvents() []models.Event
}

// EventLogFilterInput описує вхідні дані для фільтрації журналу подій.
type EventLogFilterInput struct {
	AllEvents          []models.Event
	Period             string
	SelectedSource     string
	ImportantOnly      bool
	ShowForCurrentOnly bool
	CurrentObjectID    int
	HasCurrentObject   bool
	MaxEvents          int
	Now                time.Time
}

// EventLogFilterOutput описує результат фільтрації журналу подій.
type EventLogFilterOutput struct {
	Filtered    []models.Event
	Count       int
	CountAll    int
	CountBridge int
	CountCASL   int
}

// EventLogViewModel інкапсулює бізнес-правила формування журналу подій.
type EventLogViewModel struct{}

func NewEventLogViewModel() *EventLogViewModel {
	return &EventLogViewModel{}
}

func (vm *EventLogViewModel) LoadEvents(useCase EventLogUseCase) []models.Event {
	if useCase == nil {
		return nil
	}
	events := useCase.FetchEvents()
	return append([]models.Event(nil), events...)
}

func (vm *EventLogViewModel) ApplyFilters(input EventLogFilterInput) EventLogFilterOutput {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	year, month, day := now.Date()

	filtered := make([]models.Event, 0, len(input.AllEvents))
	countAll := 0
	countBridge := 0
	countCASL := 0
	for _, event := range input.AllEvents {
		switch input.Period {
		case "Остання година":
			if now.Sub(event.Time) > time.Hour {
				// Події приходять у порядку від нових до старих,
				// тому далі теж буде поза періодом.
				goto done
			}
		case "Сьогодні":
			y, m, d := event.Time.Date()
			if y != year || m != month || d != day {
				goto done
			}
		}

		if input.ImportantOnly && !(event.IsCritical() || event.IsWarning()) {
			continue
		}

		if input.ShowForCurrentOnly && input.HasCurrentObject && event.ObjectID != input.CurrentObjectID {
			continue
		}

		source := ObjectSourceByID(event.ObjectID)
		countAll++
		if source == ObjectSourceCASL {
			countCASL++
		} else {
			countBridge++
		}

		if !sourceMatchesFilter(source, input.SelectedSource) {
			continue
		}

		filtered = append(filtered, event)
	}

done:
	if input.MaxEvents > 0 && len(filtered) > input.MaxEvents {
		filtered = filtered[:input.MaxEvents]
	}

	return EventLogFilterOutput{
		Filtered:    filtered,
		Count:       len(filtered),
		CountAll:    countAll,
		CountBridge: countBridge,
		CountCASL:   countCASL,
	}
}
