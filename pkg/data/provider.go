// pkg/data/provider.go
package data

import (
	"obj_catalog_fyne_v3/pkg/models"
	// "strconv"
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

// MockDataProvider адаптує MockData до інтерфейсу DataProvider
// Це дозволяє легко замінити MockData на реальну БД пізніше.
// type MockDataProvider struct {
// 	mock *MockData
// }

// func NewMockDataProvider(mock *MockData) *MockDataProvider {
// 	return &MockDataProvider{mock: mock}
// }

// func (p *MockDataProvider) GetObjects() []models.Object {
// 	return p.mock.GetObjects()
// }

// func (p *MockDataProvider) GetObjectByID(id string) *models.Object {
// 	return p.mock.GetObjectByIDStr(id)
// }

// func (p *MockDataProvider) GetEvents() []models.Event {
// 	return p.mock.GetEvents()
// }

// func (p *MockDataProvider) GetObjectEvents(objectID string) []models.Event {
// 	id, _ := strconv.Atoi(objectID)
// 	return p.mock.GetObjectEvents(id)
// }

// func (p *MockDataProvider) GetAlarms() []models.Alarm {
// 	return p.mock.GetAlarms()
// }

// func (p *MockDataProvider) ProcessAlarm(id string, user string, note string) {
// 	p.mock.ProcessAlarmStr(id, user, note)
// }
// func (p *MockDataProvider) GetZones(objectID string) []models.Zone {
// 	obj := p.mock.GetObjectByIDStr(objectID)
// 	if obj != nil {
// 		return obj.Zones
// 	}
// 	return nil
// }

// func (p *MockDataProvider) GetEmployees(objectID string) []models.Contact {
// 	obj := p.mock.GetObjectByIDStr(objectID)
// 	if obj != nil {
// 		return obj.Contacts
// 	}
// 	return nil
// }

// func (p *MockDataProvider) GetTestMessages(objectID string) []models.TestMessage {
// 	return nil // Mock not implemented
// }

// func (p *MockDataProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
// 	return "Mock Signal", "Mock Test", time.Now(), time.Now()
// }
