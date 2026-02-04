package data

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

// MockData зберігає всі тестові дані
type MockData struct {
	Objects []models.Object
	Alarms  []models.Alarm
	Events  []models.Event

	// Для потокобезпечного доступу
	mutex sync.RWMutex

	// Лічильники для генерації ID
	nextAlarmID int
	nextEventID int
}

// NewMockData створює нові тестові дані
func NewMockData() *MockData {
	data := &MockData{
		nextAlarmID: 100,
		nextEventID: 1000,
	}
	data.generateObjects()
	data.generateAlarms()
	data.generateEvents()
	fmt.Printf("DEBUG: MockData Created. Objects: %d, Alarms: %d, Events: %d\n", len(data.Objects), len(data.Alarms), len(data.Events))
	return data
}

// generateObjects створює тестові об'єкти
func (d *MockData) generateObjects() {
	d.Objects = []models.Object{
		{
			ID:            1001,
			Name:          "ТОВ \"Ромашка\"",
			Address:       "вул. Шевченка, 15",
			ContractNum:   "ПС-2024-001",
			Phone:         "+380501234567",
			Status:        models.StatusNormal,
			IsUnderGuard:  true,
			DeviceType:    "Тірас-16П",
			GSMLevel:      85,
			PowerSource:   models.PowerMains,
			AutoTestHours: 24,
			SIM1:          "+380671111111",
			SIM2:          "+380672222222",
			Zones: []models.Zone{
				{Number: 1, Name: "Склад 1 поверх", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Офіс", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 3, Name: "Коридор", SensorType: "Теплові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Петренко Іван Васильович", Position: "Директор", Phone: "+380501234567", Priority: 1, CodeWord: "Сонце"},
				{Name: "Коваленко Марія Олексіївна", Position: "Охоронець", Phone: "+380671234567", Priority: 2, CodeWord: "Місяць"},
			},
		},
		{
			ID:            1002,
			Name:          "Супермаркет \"Продукти\"",
			Address:       "пр. Перемоги, 100",
			ContractNum:   "ПС-2024-002",
			Phone:         "+380509876543",
			Status:        models.StatusFire,
			StatusText:    "ПОЖЕЖА В ШЛЕЙФІ 2",
			IsUnderGuard:  true,
			DeviceType:    "Тірас-8П",
			GSMLevel:      72,
			PowerSource:   models.PowerMains,
			AutoTestHours: 24,
			SIM1:          "+380673333333",
			SIM2:          "",
			Zones: []models.Zone{
				{Number: 1, Name: "Торговий зал", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Склад продуктів", SensorType: "Димові", Status: models.ZoneFire},
				{Number: 3, Name: "Підсобка", SensorType: "Теплові", Status: models.ZoneNormal},
				{Number: 4, Name: "Кабінет директора", SensorType: "Димові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Сидоренко Олег Петрович", Position: "Директор", Phone: "+380509876543", Priority: 1, CodeWord: "Зірка"},
				{Name: "Бондаренко Анна Ігорівна", Position: "Бухгалтер", Phone: "+380679876543", Priority: 2, CodeWord: "Небо"},
			},
		},
		{
			ID:            1003,
			Name:          "Школа №25",
			Address:       "вул. Лесі Українки, 42",
			ContractNum:   "ПС-2024-003",
			Phone:         "+380442223344",
			Status:        models.StatusFault,
			StatusText:    "НЕСПРАВНІСТЬ ШЛЕЙФУ 1",
			IsUnderGuard:  true,
			DeviceType:    "Тірас-32П",
			GSMLevel:      45,
			PowerSource:   models.PowerBattery,
			AutoTestHours: 12,
			SIM1:          "+380674444444",
			SIM2:          "+380675555555",
			Zones: []models.Zone{
				{Number: 1, Name: "Спортзал", SensorType: "Теплові", Status: models.ZoneBreak},
				{Number: 2, Name: "Їдальня", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 3, Name: "Актова зала", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 4, Name: "Бібліотека", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 5, Name: "Коридор 1 поверх", SensorType: "Теплові", Status: models.ZoneNormal},
				{Number: 6, Name: "Коридор 2 поверх", SensorType: "Теплові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Мельник Наталія Андріївна", Position: "Директор", Phone: "+380442223344", Priority: 1, CodeWord: "Освіта"},
				{Name: "Кравченко Сергій Юрійович", Position: "Завгосп", Phone: "+380674445566", Priority: 2, CodeWord: "Школа"},
			},
		},
		{
			ID:            1004,
			Name:          "Аптека \"Здоров'я\"",
			Address:       "вул. Хрещатик, 5",
			ContractNum:   "ПС-2024-004",
			Phone:         "+380445556677",
			Status:        models.StatusNormal,
			IsUnderGuard:  true,
			DeviceType:    "Тірас-4П",
			GSMLevel:      92,
			PowerSource:   models.PowerMains,
			AutoTestHours: 24,
			SIM1:          "+380676666666",
			SIM2:          "",
			Zones: []models.Zone{
				{Number: 1, Name: "Торговий зал", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Склад ліків", SensorType: "Димові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Лисенко Тетяна Миколаївна", Position: "Завідуюча", Phone: "+380445556677", Priority: 1, CodeWord: "Ліки"},
			},
		},
		{
			ID:            1005,
			Name:          "Готель \"Центральний\"",
			Address:       "пл. Незалежності, 1",
			ContractNum:   "ПС-2024-005",
			Phone:         "+380447778899",
			Status:        models.StatusOffline,
			StatusText:    "НЕМАЄ ЗВ'ЯЗКУ",
			IsUnderGuard:  true,
			DeviceType:    "Тірас-64П",
			GSMLevel:      0,
			PowerSource:   models.PowerMains,
			AutoTestHours: 6,
			SIM1:          "+380677777777",
			SIM2:          "+380678888888",
			Zones: []models.Zone{
				{Number: 1, Name: "Рецепція", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Ресторан", SensorType: "Теплові", Status: models.ZoneNormal},
				{Number: 3, Name: "Конференц-зал", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 4, Name: "Поверх 1", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 5, Name: "Поверх 2", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 6, Name: "Поверх 3", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 7, Name: "Поверх 4", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 8, Name: "Підвал", SensorType: "Теплові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Гончаренко Віктор Андрійович", Position: "Директор", Phone: "+380447778899", Priority: 1, CodeWord: "Готель"},
				{Name: "Савченко Олена Вікторівна", Position: "Адміністратор", Phone: "+380679998877", Priority: 2, CodeWord: "Номер"},
				{Name: "Козак Дмитро Олексійович", Position: "Охоронець", Phone: "+380671112233", Priority: 3, CodeWord: "Безпека"},
			},
		},
		{
			ID:            1006,
			Name:          "Бізнес-центр \"Олімп\"",
			Address:       "вул. Грушевського, 28",
			ContractNum:   "ПС-2024-006",
			Phone:         "+380441112233",
			Status:        models.StatusNormal,
			IsUnderGuard:  false,
			DeviceType:    "Тірас-48П",
			GSMLevel:      88,
			PowerSource:   models.PowerMains,
			AutoTestHours: 12,
			SIM1:          "+380679999999",
			SIM2:          "+380670000000",
			Zones: []models.Zone{
				{Number: 1, Name: "Холл", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Офіс 101", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 3, Name: "Офіс 102", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 4, Name: "Офіс 201", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 5, Name: "Офіс 202", SensorType: "Димові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Романенко Андрій Павлович", Position: "Керуючий", Phone: "+380441112233", Priority: 1, CodeWord: "Олімп"},
			},
		},
		{
			ID:            1007,
			Name:          "Дитячий садок \"Веселка\"",
			Address:       "вул. Квіткова, 10",
			ContractNum:   "ПС-2024-007",
			Phone:         "+380443334455",
			Status:        models.StatusNormal,
			IsUnderGuard:  true,
			DeviceType:    "Тірас-16П",
			GSMLevel:      78,
			PowerSource:   models.PowerMains,
			AutoTestHours: 24,
			SIM1:          "+380671234321",
			SIM2:          "",
			Zones: []models.Zone{
				{Number: 1, Name: "Група \"Сонечко\"", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Група \"Зірочка\"", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 3, Name: "Кухня", SensorType: "Теплові", Status: models.ZoneNormal},
				{Number: 4, Name: "Музичний зал", SensorType: "Димові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Іваненко Світлана Петрівна", Position: "Завідуюча", Phone: "+380443334455", Priority: 1, CodeWord: "Веселка"},
				{Name: "Демченко Оксана Ігорівна", Position: "Методист", Phone: "+380671234321", Priority: 2, CodeWord: "Діти"},
			},
		},
		{
			ID:            1008,
			Name:          "Ресторан \"Козак\"",
			Address:       "вул. Саксаганського, 55",
			ContractNum:   "ПС-2024-008",
			Phone:         "+380445556688",
			Status:        models.StatusNormal,
			IsUnderGuard:  true,
			DeviceType:    "Тірас-8П",
			GSMLevel:      95,
			PowerSource:   models.PowerMains,
			AutoTestHours: 24,
			SIM1:          "+380679876512",
			SIM2:          "",
			Zones: []models.Zone{
				{Number: 1, Name: "Зал для гостей", SensorType: "Димові", Status: models.ZoneNormal},
				{Number: 2, Name: "Кухня", SensorType: "Теплові", Status: models.ZoneNormal},
				{Number: 3, Name: "Бар", SensorType: "Димові", Status: models.ZoneNormal},
			},
			Contacts: []models.Contact{
				{Name: "Шевченко Богдан Тарасович", Position: "Власник", Phone: "+380445556688", Priority: 1, CodeWord: "Козак"},
			},
		},
	}
}

// generateAlarms створює тестові активні тривоги
func (d *MockData) generateAlarms() {
	now := time.Now()
	d.Alarms = []models.Alarm{
		{
			ID:          1,
			ObjectID:    1002,
			ObjectName:  "Супермаркет \"Продукти\"",
			Address:     "пр. Перемоги, 100",
			Time:        now.Add(-2 * time.Minute),
			Type:        models.AlarmFire,
			ZoneNumber:  2,
			ZoneName:    "Склад продуктів",
			IsProcessed: false,
		},
		{
			ID:          2,
			ObjectID:    1003,
			ObjectName:  "Школа №25",
			Address:     "вул. Лесі Українки, 42",
			Time:        now.Add(-15 * time.Minute),
			Type:        models.AlarmFault,
			ZoneNumber:  1,
			ZoneName:    "Спортзал",
			IsProcessed: false,
		},
		{
			ID:          3,
			ObjectID:    1005,
			ObjectName:  "Готель \"Центральний\"",
			Address:     "пл. Незалежності, 1",
			Time:        now.Add(-30 * time.Minute),
			Type:        models.AlarmFault,
			ZoneNumber:  0,
			ZoneName:    "",
			IsProcessed: false,
		},
	}
}

// generateEvents створює тестові події для журналу
func (d *MockData) generateEvents() {
	now := time.Now()
	d.Events = []models.Event{
		// Останні події
		{ID: 1, Time: now.Add(-1 * time.Minute), ObjectID: 1002, ObjectName: "Супермаркет \"Продукти\"", Type: models.EventFire, ZoneNumber: 2, Details: "Склад продуктів"},
		{ID: 2, Time: now.Add(-3 * time.Minute), ObjectID: 1002, ObjectName: "Супермаркет \"Продукти\"", Type: models.EventPowerFail, Details: "Перехід на АКБ"},
		{ID: 3, Time: now.Add(-5 * time.Minute), ObjectID: 1001, ObjectName: "ТОВ \"Ромашка\"", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 4, Time: now.Add(-8 * time.Minute), ObjectID: 1003, ObjectName: "Школа №25", Type: models.EventFault, ZoneNumber: 1, Details: "Обрив шлейфу"},
		{ID: 5, Time: now.Add(-10 * time.Minute), ObjectID: 1004, ObjectName: "Аптека \"Здоров'я\"", Type: models.EventArm, Details: "Постановка під охорону", UserName: "Лисенко Т.М."},
		{ID: 6, Time: now.Add(-12 * time.Minute), ObjectID: 1001, ObjectName: "ТОВ \"Ромашка\"", Type: models.EventOnline, Details: "Відновлення зв'язку"},
		{ID: 7, Time: now.Add(-15 * time.Minute), ObjectID: 1007, ObjectName: "Дитячий садок \"Веселка\"", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 8, Time: now.Add(-18 * time.Minute), ObjectID: 1005, ObjectName: "Готель \"Центральний\"", Type: models.EventOffline, Details: "Втрата зв'язку"},
		{ID: 9, Time: now.Add(-20 * time.Minute), ObjectID: 1008, ObjectName: "Ресторан \"Козак\"", Type: models.EventDisarm, Details: "Зняття з охорони", UserName: "Шевченко Б.Т."},
		{ID: 10, Time: now.Add(-22 * time.Minute), ObjectID: 1006, ObjectName: "Бізнес-центр \"Олімп\"", Type: models.EventPowerFail, Details: "Перехід на АКБ"},
		{ID: 11, Time: now.Add(-25 * time.Minute), ObjectID: 1006, ObjectName: "Бізнес-центр \"Олімп\"", Type: models.EventPowerOK, Details: "Відновлення 220В"},
		{ID: 12, Time: now.Add(-28 * time.Minute), ObjectID: 1001, ObjectName: "ТОВ \"Ромашка\"", Type: models.EventArm, Details: "Постановка під охорону", UserName: "Петренко І.В."},
		{ID: 13, Time: now.Add(-30 * time.Minute), ObjectID: 1003, ObjectName: "Школа №25", Type: models.EventBatteryLow, Details: "Низький заряд АКБ"},
		{ID: 14, Time: now.Add(-35 * time.Minute), ObjectID: 1004, ObjectName: "Аптека \"Здоров'я\"", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 15, Time: now.Add(-40 * time.Minute), ObjectID: 1007, ObjectName: "Дитячий садок \"Веселка\"", Type: models.EventDisarm, Details: "Зняття з охорони", UserName: "Іваненко С.П."},
		{ID: 16, Time: now.Add(-45 * time.Minute), ObjectID: 1008, ObjectName: "Ресторан \"Козак\"", Type: models.EventArm, Details: "Постановка під охорону", UserName: "Шевченко Б.Т."},
		{ID: 17, Time: now.Add(-50 * time.Minute), ObjectID: 1002, ObjectName: "Супермаркет \"Продукти\"", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 18, Time: now.Add(-55 * time.Minute), ObjectID: 1005, ObjectName: "Готель \"Центральний\"", Type: models.EventOnline, Details: "Відновлення зв'язку"},
		{ID: 19, Time: now.Add(-60 * time.Minute), ObjectID: 1001, ObjectName: "ТОВ \"Ромашка\"", Type: models.EventRestore, ZoneNumber: 2, Details: "Відновлення норми"},
		{ID: 20, Time: now.Add(-65 * time.Minute), ObjectID: 1006, ObjectName: "Бізнес-центр \"Олімп\"", Type: models.EventDisarm, Details: "Зняття з охорони", UserName: "Романенко А.П."},
		// Старіші події
		{ID: 21, Time: now.Add(-70 * time.Minute), ObjectID: 1003, ObjectName: "Школа №25", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 22, Time: now.Add(-80 * time.Minute), ObjectID: 1007, ObjectName: "Дитячий садок \"Веселка\"", Type: models.EventArm, Details: "Постановка під охорону", UserName: "Демченко О.І."},
		{ID: 23, Time: now.Add(-90 * time.Minute), ObjectID: 1004, ObjectName: "Аптека \"Здоров'я\"", Type: models.EventDisarm, Details: "Зняття з охорони", UserName: "Лисенко Т.М."},
		{ID: 24, Time: now.Add(-100 * time.Minute), ObjectID: 1008, ObjectName: "Ресторан \"Козак\"", Type: models.EventTest, Details: "Автоматичний тест"},
		{ID: 25, Time: now.Add(-120 * time.Minute), ObjectID: 1002, ObjectName: "Супермаркет \"Продукти\"", Type: models.EventArm, Details: "Постановка під охорону", UserName: "Сидоренко О.П."},
	}
}

// GetAlarms повертає активні тривоги (реалізація інтерфейсу)
func (d *MockData) GetAlarms() []models.Alarm {
	return d.GetActiveAlarms()
}

// GetObjects повертає список об'єктів (реалізація інтерфейсу)
func (d *MockData) GetObjects() []models.Object {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.Objects // В реальному коді краще копіювати
}

// GetEvents повертає список подій (реалізація інтерфейсу)
func (d *MockData) GetEvents() []models.Event {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.Events
}

// GetActiveAlarms повертає тільки необроблені тривоги
func (d *MockData) GetActiveAlarms() []models.Alarm {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var active []models.Alarm
	for _, alarm := range d.Alarms {
		if !alarm.IsProcessed {
			active = append(active, alarm)
		}
	}
	return active
}

// ProcessAlarm позначає тривогу як оброблену (реалізація інтерфейсу з конвертацією ID)
func (d *MockData) ProcessAlarmStr(id string, processedBy string, note string) bool {
	alarmID, _ := strconv.Atoi(id)
	return d.ProcessAlarm(alarmID, processedBy, note)
}

// ProcessAlarm позначає тривогу як оброблену
func (d *MockData) ProcessAlarm(alarmID int, processedBy string, note string) bool {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	for i := range d.Alarms {
		if d.Alarms[i].ID == alarmID {
			d.Alarms[i].IsProcessed = true
			d.Alarms[i].ProcessedBy = processedBy
			d.Alarms[i].ProcessNote = note
			return true
		}
	}
	return false
}

// GetObjectByIDStr повертає об'єкт за ID (string)
func (d *MockData) GetObjectByIDStr(id string) *models.Object {
	objID, _ := strconv.Atoi(id)
	return d.GetObjectByID(objID)
}

// GetObjectByID повертає об'єкт за ID
func (d *MockData) GetObjectByID(id int) *models.Object {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	for i := range d.Objects {
		if d.Objects[i].ID == id {
			return &d.Objects[i]
		}
	}
	return nil
}

// GetObjectEvents повертає події конкретного об'єкта
func (d *MockData) GetObjectEvents(objectID int) []models.Event {
	d.mutex.RLock()
	defer d.mutex.RUnlock()

	var events []models.Event
	for _, event := range d.Events {
		if event.ObjectID == objectID {
			events = append(events, event)
		}
	}
	return events
}

// AddEvent додає нову подію (для симуляції real-time)
func (d *MockData) AddEvent(event models.Event) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.nextEventID++
	event.ID = d.nextEventID
	event.Time = time.Now()
	// Додаємо на початок списку
	d.Events = append([]models.Event{event}, d.Events...)
}

// SimulateRandomEvent генерує випадкову подію для демонстрації
func (d *MockData) SimulateRandomEvent() models.Event {
	d.mutex.RLock()
	obj := d.Objects[rand.Intn(len(d.Objects))]
	d.mutex.RUnlock()

	eventTypes := []models.EventType{
		models.EventTest,
		models.EventOnline,
		models.EventArm,
		models.EventDisarm,
	}

	event := models.Event{
		ObjectID:   obj.ID,
		ObjectName: obj.Name,
		Type:       eventTypes[rand.Intn(len(eventTypes))],
		Details:    "Симульована подія",
	}

	d.AddEvent(event)
	return event
}

// SimulateObjectChange змінює випадковий параметр випадкового об'єкта
func (d *MockData) SimulateObjectChange() *models.Object {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	idx := rand.Intn(len(d.Objects))
	obj := &d.Objects[idx]

	// Змінюємо GSM рівень
	change := rand.Intn(11) - 5 // -5 до +5
	obj.GSMLevel += change
	if obj.GSMLevel > 100 {
		obj.GSMLevel = 100
	} else if obj.GSMLevel < 0 {
		obj.GSMLevel = 0
	}

	// Рідко змінюємо статус (1 з 10 разів)
	if rand.Intn(10) == 0 {
		statuses := []models.ObjectStatus{models.StatusNormal, models.StatusOffline, models.StatusFault}
		// Якщо зараз Normal, є шанс стати іншим. Якщо не Normal, великий шанс стати Normal.
		if obj.Status == models.StatusNormal {
			if rand.Intn(5) == 0 { // 20% шанс проблеми
				obj.Status = statuses[rand.Intn(len(statuses))]
			}
		} else {
			if rand.Intn(2) == 0 { // 50% шанс відновлення
				obj.Status = models.StatusNormal
			}
		}
	}

	return obj
}

// SimulateNewAlarm генерує нову тривогу або видаляє стару
func (d *MockData) SimulateNewAlarm() (bool, *models.Alarm) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	// 30% шанс видалити стару оброблену, 70% створити нову
	if len(d.Alarms) > 0 && rand.Intn(10) < 3 {
		// Видаляємо випадкову стару (просто для демо, щоб список не ріс вічно)
		if len(d.Alarms) > 5 {
			d.Alarms = d.Alarms[1:]
			return true, nil // список змінився
		}
	}

	// Створюємо нову лише якщо їх мало
	if len(d.Alarms) < 10 && rand.Intn(10) < 4 { // 40% шанс
		idx := rand.Intn(len(d.Objects))
		obj := d.Objects[idx]

		d.nextAlarmID++
		alarm := models.Alarm{
			ID:          d.nextAlarmID,
			ObjectID:    obj.ID,
			ObjectName:  obj.Name,
			Address:     obj.Address,
			Time:        time.Now(),
			Type:        models.AlarmFire,
			ZoneNumber:  rand.Intn(5) + 1,
			IsProcessed: false,
		}
		if rand.Intn(2) == 0 {
			alarm.Type = models.AlarmFault
		}

		d.Alarms = append([]models.Alarm{alarm}, d.Alarms...)
		return true, &alarm
	}

	return false, nil
}
