// Package contracts defines stable GUI-backend interfaces.
package contracts

import (
	"context"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

// ObjectProvider визначає інтерфейс для отримання об'єктів
type ObjectProvider interface {
	GetObjects() []models.Object
	GetObjectByID(id string) *models.Object
}

// ContextObjectProvider supports cancellation of potentially blocking object reads.
type ContextObjectProvider interface {
	GetObjectsContext(ctx context.Context) []models.Object
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

// ContextEventProvider supports cancellation of potentially blocking event reads.
type ContextEventProvider interface {
	GetEventsContext(ctx context.Context) []models.Event
}

// ObjectEventsRangeProvider optionally loads an object's events for an explicit time range.
type ObjectEventsRangeProvider interface {
	GetObjectEventsRange(objectID string, from time.Time, to time.Time) []models.Event
}

type ObjectMediaKind string

const (
	ObjectMediaImage  ObjectMediaKind = "image"
	ObjectMediaCamera ObjectMediaKind = "camera"
)

// ObjectMedia describes an object photo, scheme or camera endpoint.
type ObjectMedia struct {
	ID       string
	Kind     ObjectMediaKind
	Title    string
	RoomName string
	URL      string
}

// ObjectMediaProvider loads object media lazily.
type ObjectMediaProvider interface {
	GetObjectMedia(ctx context.Context, objectID int) ([]ObjectMedia, error)
	FetchObjectMedia(ctx context.Context, media ObjectMedia) ([]byte, error)
}

type ObjectLocation struct {
	ObjectID  int
	Latitude  string
	Longitude string
}

// ObjectLocationProvider loads coordinates only when an operational map is opened.
type ObjectLocationProvider interface {
	ListObjectLocations(ctx context.Context) ([]ObjectLocation, error)
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
	ProcessAlarm(id string, user string, note string) error
}

// ContextAlarmProvider supports cancellation of potentially blocking alarm reads.
type ContextAlarmProvider interface {
	GetAlarmsContext(ctx context.Context) []models.Alarm
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

// AlarmTakeoverProvider описує взяття/перехоплення тривоги в роботу.
type AlarmTakeoverProvider interface {
	PickAlarm(ctx context.Context, alarm models.Alarm, user string) error
}

type ResponseGroupStatus string

const (
	ResponseGroupStatusUnknown    ResponseGroupStatus = "unknown"
	ResponseGroupStatusFree       ResponseGroupStatus = "free"
	ResponseGroupStatusDispatched ResponseGroupStatus = "dispatched"
	ResponseGroupStatusArrived    ResponseGroupStatus = "arrived"
)

// ResponseGroup описує групу реагування (МГР).
type ResponseGroup struct {
	ID              string
	Name            string
	Callsign        string
	Phone           string
	Source          FrontendSource
	Status          ResponseGroupStatus
	StatusText      string
	ObjectNumber    string
	ObjectName      string
	Latitude        string
	Longitude       string
	StatusChangedAt time.Time
}

// AlarmGroupProcessProvider описує групове завершення тривог (МІСТ).
type AlarmGroupProcessProvider interface {
	GroupProcessAlarm(ctx context.Context, alarm models.Alarm, user string) error
}

// ResponseGroupProvider описує отримання та дії з групами реагування.
type ResponseGroupProvider interface {
	ListResponseGroups(ctx context.Context) ([]ResponseGroup, error)
	AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error
	NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error
	CancelResponseGroup(ctx context.Context, alarm models.Alarm) error
}

// AlarmResponseGroupProvider loads response groups only from the source that owns an alarm.
type AlarmResponseGroupProvider interface {
	ListResponseGroupsForAlarm(ctx context.Context, alarm models.Alarm) ([]ResponseGroup, error)
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
