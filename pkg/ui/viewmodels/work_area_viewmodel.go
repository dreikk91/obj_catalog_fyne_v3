package viewmodels

import (
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

	fullObj := provider.GetObjectByID(idStr)
	zones := provider.GetZones(idStr)
	contacts := provider.GetEmployees(idStr)

	details := WorkAreaDetails{
		Zones:    append([]models.Zone(nil), zones...),
		Contacts: append([]models.Contact(nil), contacts...),
	}
	if fullObj != nil {
		clone := *fullObj
		details.FullObject = &clone
	}
	return details
}

// LoadObjectEvents повертає лише журнал подій об'єкта з урахуванням ліміту.
func (vm *WorkAreaViewModel) LoadObjectEvents(provider WorkAreaDataProvider, objectID int, eventLimit int) []models.Event {
	idStr := strconv.Itoa(objectID)
	events := sortEventsByTimeDesc(provider.GetObjectEvents(idStr))

	if eventLimit > 0 && len(events) > eventLimit {
		events = events[:eventLimit]
	}

	return append([]models.Event(nil), events...)
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
