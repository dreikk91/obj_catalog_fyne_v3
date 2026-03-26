// Package contracts defines stable GUI-backend interfaces.
package contracts

import (
	"obj_catalog_fyne_v3/pkg/models"
	"time"
)

// ObjectProvider визначає інтерфейс для отримання об'єктів
type ObjectProvider interface {
	GetObjects() []models.Object
	GetObjectByID(id string) *models.Object
}

type DetailProvider interface {
	GetZones(objectID string) []models.Zone
	GetEmployees(objectID string) []models.Contact
	GetTestMessages(objectID string) []models.TestMessage
	GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time)
}

// EventProvider визначає інтерфейс для отримання подій
type EventProvider interface {
	GetEvents() []models.Event
	GetObjectEvents(objectID string) []models.Event
}

// AlarmProvider визначає інтерфейс для отримання тривог
type AlarmProvider interface {
	GetAlarms() []models.Alarm
	ProcessAlarm(id string, user string, note string)
}

// DataProvider об'єднує всі інтерфейси даних
type DataProvider interface {
	ObjectProvider
	EventProvider
	AlarmProvider
	DetailProvider
}
