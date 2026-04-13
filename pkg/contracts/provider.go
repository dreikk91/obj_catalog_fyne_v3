// Package contracts defines stable GUI-backend interfaces.
package contracts

import (
	"context"
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

// AlarmHistoryProvider defines optional lazy chronology loading for a single alarm.
type AlarmHistoryProvider interface {
	GetAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg
}

// ActiveAlarmHistoryProvider defines optional chronology loading from currently active alarm rows only.
type ActiveAlarmHistoryProvider interface {
	GetActiveAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg
}

// AlarmProvider визначає інтерфейс для отримання тривог
type AlarmProvider interface {
	GetAlarms() []models.Alarm
	ProcessAlarm(id string, user string, note string)
}

// AlarmProcessingOption описує одну причину відпрацювання тривоги.
type AlarmProcessingOption struct {
	Code  string
	Label string
}

// AlarmProcessingRequest описує параметри відпрацювання тривоги.
type AlarmProcessingRequest struct {
	CauseCode string
	Note      string
}

// AlarmProcessingProvider описує розширене відпрацювання тривоги
// з причиною відпрацювання, як у CASL.
type AlarmProcessingProvider interface {
	GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]AlarmProcessingOption, error)
	ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request AlarmProcessingRequest) error
}

// DataProvider об'єднує всі інтерфейси даних
type DataProvider interface {
	ObjectProvider
	EventProvider
	AlarmProvider
	DetailProvider
}

// ShutdownProvider описує провайдер, який має довгоживучі фонові ресурси
// (goroutines, realtime streams, reconnect loops) і повинен бути явно зупинений.
type ShutdownProvider interface {
	Shutdown()
}
