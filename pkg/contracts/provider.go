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

// WorkAreaDetailsProvider визначає мінімальний набір даних для правої панелі об'єкта.
type WorkAreaDetailsProvider interface {
	GetObjectByID(id string) *models.Object
	GetZones(objectID string) []models.Zone
	GetEmployees(objectID string) []models.Contact
	GetObjectEvents(objectID string) []models.Event
	GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time)
}

// TestMessageProvider визначає доступ до тестових повідомлень об'єкта.
type TestMessageProvider interface {
	GetTestMessages(objectID string) []models.TestMessage
}

// WorkAreaProvider об'єднує мінімальні залежності WorkArea UI.
type WorkAreaProvider interface {
	WorkAreaDetailsProvider
	TestMessageProvider
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
