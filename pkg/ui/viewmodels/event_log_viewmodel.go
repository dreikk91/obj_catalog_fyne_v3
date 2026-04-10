package viewmodels

import (
	"slices"
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
	Filtered     []models.Event
	Count        int
	CountAll     int
	CountBridge  int
	CountPhoenix int
	CountCASL    int
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
	return sortEventsByTimeDesc(events)
}

func (vm *EventLogViewModel) ApplyFilters(input EventLogFilterInput) EventLogFilterOutput {
	now := input.Now
	if now.IsZero() {
		now = time.Now()
	}
	year, month, day := now.Date()

	orderedEvents := sortEventsByTimeDesc(input.AllEvents)
	filtered := make([]models.Event, 0, len(orderedEvents))
	countAll := 0
	countBridge := 0
	countPhoenix := 0
	countCASL := 0
eventLoop:
	for _, event := range orderedEvents {
		switch input.Period {
		case "Остання година":
			if now.Sub(event.Time) > time.Hour {
				// Події приходять у порядку від нових до старих,
				// тому далі теж буде поза періодом.
				break eventLoop
			}
		case "Сьогодні":
			y, m, d := event.Time.Date()
			if y != year || m != month || d != day {
				break eventLoop
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
		switch source {
		case ObjectSourceCASL:
			countCASL++
		case ObjectSourcePhoenix:
			countPhoenix++
		default:
			countBridge++
		}

		if !sourceMatchesFilter(source, input.SelectedSource) {
			continue
		}

		filtered = append(filtered, event)
	}
	if input.MaxEvents > 0 && len(filtered) > input.MaxEvents {
		filtered = filtered[:input.MaxEvents]
	}

	return EventLogFilterOutput{
		Filtered:     filtered,
		Count:        len(filtered),
		CountAll:     countAll,
		CountBridge:  countBridge,
		CountPhoenix: countPhoenix,
		CountCASL:    countCASL,
	}
}

func sortEventsByTimeDesc(events []models.Event) []models.Event {
	ordered := slices.Clone(events)
	slices.SortStableFunc(ordered, func(left, right models.Event) int {
		leftTime := left.Time
		rightTime := right.Time
		if leftTime.Equal(rightTime) {
			switch {
			case left.ID > right.ID:
				return -1
			case left.ID < right.ID:
				return 1
			default:
				return 0
			}
		}
		switch {
		case leftTime.After(rightTime):
			return -1
		case leftTime.Before(rightTime):
			return 1
		default:
			return 0
		}
	})
	return ordered
}
