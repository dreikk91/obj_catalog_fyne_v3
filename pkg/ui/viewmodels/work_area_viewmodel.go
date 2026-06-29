package viewmodels

import (
	"slices"
	"strconv"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

// WorkAreaDataProvider описує мінімальні дані, потрібні для завантаження деталей об'єкта.
type WorkAreaDataProvider interface {
	GetObjectByID(id string) *models.Object
	GetZones(objectID string) []models.Zone
	GetEmployees(objectID string) []models.Contact
	GetObjectEvents(objectID string) []models.Event
}

// WorkAreaBaseDetailsProvider can return object, zones, and contacts with one backend request.
type WorkAreaBaseDetailsProvider interface {
	GetObjectBaseDetails(objectID string) (fullObject *models.Object, zones []models.Zone, contacts []models.Contact)
}

// WorkAreaExternalDataProvider описує мінімальні зовнішні дані для полів сигналу/тестів.
type WorkAreaExternalDataProvider interface {
	GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time)
}

// WorkAreaDetails містить агреговані деталі об'єкта для правої панелі.
type WorkAreaDetails struct {
	FullObject *models.Object
	Zones      []models.Zone
	Contacts   []models.Contact
	Events     []models.Event
}

// WorkAreaExternalData містить дані зовнішнього статусу об'єкта.
type WorkAreaExternalData struct {
	Signal      string
	TestMessage string
	LastTest    time.Time
	LastMessage time.Time
}

// WorkAreaViewModel інкапсулює завантаження та підготовку деталей об'єкта.
type WorkAreaViewModel struct{}

func NewWorkAreaViewModel() *WorkAreaViewModel {
	return &WorkAreaViewModel{}
}

func (vm *WorkAreaViewModel) CanApplyDetails(currentObject *models.Object, requestedObjectID int) bool {
	return currentObject != nil && currentObject.ID == requestedObjectID
}

func (vm *WorkAreaViewModel) LoadObjectBaseDetails(provider WorkAreaDataProvider, objectID int) WorkAreaDetails {
	idStr := strconv.Itoa(objectID)

	if optimized, ok := provider.(WorkAreaBaseDetailsProvider); ok {
		fullObj, zones, contacts := optimized.GetObjectBaseDetails(idStr)
		return buildWorkAreaDetails(fullObj, zones, contacts)
	}

	fullObj := provider.GetObjectByID(idStr)
	zones := provider.GetZones(idStr)
	contacts := provider.GetEmployees(idStr)

	return buildWorkAreaDetails(fullObj, zones, contacts)
}

func buildWorkAreaDetails(fullObj *models.Object, zones []models.Zone, contacts []models.Contact) WorkAreaDetails {
	details := WorkAreaDetails{
		Zones:    slices.Clone(zones),
		Contacts: slices.Clone(contacts),
	}
	if fullObj != nil {
		clone := *fullObj
		details.FullObject = &clone
	}
	return details
}

// LoadObjectEvents повертає лише журнал подій об'єкта з урахуванням ліміту.
func (vm *WorkAreaViewModel) LoadObjectEvents(provider WorkAreaDataProvider, objectID int, eventLimit int) []models.Event {
	return vm.LoadObjectEventsRange(provider, objectID, eventLimit, time.Time{}, time.Time{})
}

// LoadObjectEventsRange loads and sorts an object's journal for the selected time range.
func (vm *WorkAreaViewModel) LoadObjectEventsRange(
	provider WorkAreaDataProvider,
	objectID int,
	eventLimit int,
	from time.Time,
	to time.Time,
) []models.Event {
	idStr := strconv.Itoa(objectID)
	var events []models.Event
	if ranged, ok := provider.(interface {
		GetObjectEventsRange(string, time.Time, time.Time) []models.Event
	}); ok && (!from.IsZero() || !to.IsZero()) {
		events = ranged.GetObjectEventsRange(idStr, from, to)
	} else {
		events = provider.GetObjectEvents(idStr)
	}
	events = filterWorkAreaEventsRange(events, from, to)
	events = sortEventsByTimeDesc(events)

	if eventLimit > 0 && len(events) > eventLimit {
		events = events[:eventLimit]
	}

	return slices.Clone(events)
}

func filterWorkAreaEventsRange(events []models.Event, from time.Time, to time.Time) []models.Event {
	if from.IsZero() && to.IsZero() {
		return events
	}
	result := make([]models.Event, 0, len(events))
	for _, event := range events {
		if !from.IsZero() && event.Time.Before(from) {
			continue
		}
		if !to.IsZero() && event.Time.After(to) {
			continue
		}
		result = append(result, event)
	}
	return result
}

func (vm *WorkAreaViewModel) LoadExternalData(provider WorkAreaExternalDataProvider, objectID int) WorkAreaExternalData {
	idStr := strconv.Itoa(objectID)
	signal, testMsg, lastTest, lastMessage := provider.GetExternalData(idStr)

	return WorkAreaExternalData{
		Signal:      signal,
		TestMessage: testMsg,
		LastTest:    lastTest,
		LastMessage: lastMessage,
	}
}
