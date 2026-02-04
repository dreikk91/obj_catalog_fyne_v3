package data

import (
	"sync"

	"obj_catalog_fyne_v3/pkg/models"

	"github.com/jmoiron/sqlx"
)

// Data зберігає всі дані для відображення
type Data struct {
	Objects []models.Object
	Alarms  []models.Alarm
	Events  []models.Event

	// Для потокобезпечного доступу
	mutex sync.RWMutex

	// Лічильники для генерації ID
	nextAlarmID int
	nextEventID int
	db *sqlx.DB
}

func NewData(db *sqlx.DB) *Data {
	return &Data{
		nextAlarmID: 100,
		nextEventID: 1000,
		db: db,
	}
}

func (d *Data) AddObject(obj models.Object) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.Objects = append(d.Objects, obj)
}

func (d *Data) AddAlarm(alarm models.Alarm) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.Alarms = append(d.Alarms, alarm)
}

func (d *Data) AddEvent(event models.Event) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	d.Events = append(d.Events, event)
}


func (d *Data) GetObjects() []models.Object {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	// Створюємо копію для безпеки
	objects := make([]models.Object, len(d.Objects))
	
	return objects
}

func (d *Data) GetAlarms() []models.Alarm {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.Alarms
}

func (d *Data) GetEvents() []models.Event {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.Events
}
