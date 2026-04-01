package data

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// PhoenixDataProvider реалізує інтерфейс DataProvider для БД Phoenix (MSSQL)
type PhoenixDataProvider struct {
	db *sqlx.DB
}

func NewPhoenixDataProvider(db *sqlx.DB) *PhoenixDataProvider {
	log.Debug().Msg("PhoenixDataProvider ініціалізовано")
	return &PhoenixDataProvider{db: db}
}

// GetObjects отримує список об'єктів з Phoenix MSSQL
func (p *PhoenixDataProvider) GetObjects() []models.Object {
	// TODO: Реалізувати запит до vwRealPanel
	return nil
}

// GetObjectByID отримує детальну інформацію про об'єкт Phoenix
func (p *PhoenixDataProvider) GetObjectByID(id string) *models.Object {
	// TODO: Реалізувати запит до vwRealPanel та супутніх таблиць
	return nil
}

// GetZones отримує зони об'єкта Phoenix
func (p *PhoenixDataProvider) GetZones(objectID string) []models.Zone {
	// TODO: Реалізувати запит до таблиці zones
	return nil
}

// GetEmployees отримує персонал об'єкта Phoenix
func (p *PhoenixDataProvider) GetEmployees(objectID string) []models.Contact {
	// TODO: Реалізувати запит до ResponsiblesList
	return nil
}

// GetEvents отримує глобальний журнал подій Phoenix
func (p *PhoenixDataProvider) GetEvents() []models.Event {
	// TODO: Реалізувати інкрементальне завантаження з vwArchives
	return nil
}

// GetObjectEvents отримує журнал подій для конкретного об'єкта Phoenix
func (p *PhoenixDataProvider) GetObjectEvents(objectID string) []models.Event {
	// TODO: Реалізувати запит до vwArchives за panel_id
	return nil
}

// GetAlarms отримує список активних тривог Phoenix
func (p *PhoenixDataProvider) GetAlarms() []models.Alarm {
	// TODO: Реалізувати запит до CurrentAlarms
	return nil
}

// ProcessAlarm обробляє тривогу в системі Phoenix
func (p *PhoenixDataProvider) ProcessAlarm(id string, user string, note string) {
	// TODO: Реалізувати логіку підтвердження тривоги в MSSQL
}

// GetExternalData отримує зовнішні дані Phoenix (сигнал, тестування)
func (p *PhoenixDataProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	// TODO: Реалізувати запит до таблиць Mphone/Central
	return "", "", time.Time{}, time.Time{}
}

// GetTestMessages отримує тестові повідомлення об'єкта Phoenix
func (p *PhoenixDataProvider) GetTestMessages(objectID string) []models.TestMessage {
	// TODO: Реалізувати запит до TRPMSG або аналогічної таблиці Phoenix
	return nil
}

// Перевірка імплементації інтерфейсу
var _ contracts.DataProvider = (*PhoenixDataProvider)(nil)
