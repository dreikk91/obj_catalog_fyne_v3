// Package models - структура події для журналу
package models

import "time"

// EventType визначає тип події
type EventType string

const (
	EventFire              EventType = "fire"               // Пожежа
	EventBurglary          EventType = "burglary"           // Проникнення/охоронна тривога
	EventPanic             EventType = "panic"              // Тривожна кнопка/напад
	EventMedical           EventType = "medical"            // Медична тривога
	EventGas               EventType = "gas"                // Газова тривога
	EventTamper            EventType = "tamper"             // Саботаж/тампер
	EventFault             EventType = "fault"              // Несправність
	EventRestore           EventType = "restore"            // Відновлення
	EventArm               EventType = "arm"                // Постановка під охорону
	EventDisarm            EventType = "disarm"             // Зняття з охорони
	EventTest              EventType = "test"               // Тестовий сигнал
	EventPowerFail         EventType = "power_fail"         // Втрата 220В
	EventPowerOK           EventType = "power_ok"           // Відновлення 220В
	EventBatteryLow        EventType = "batt_low"           // Низький заряд АКБ
	EventOnline            EventType = "online"             // Прилад на зв'язку
	EventOffline           EventType = "offline"            // Втрата зв'язку
	SystemEvent            EventType = "system"             // Системна подія
	EventNotification      EventType = "notification"       // Повідомлення
	EventAlarmNotification EventType = "alarm_notification" // Потрапляння тривоги в стрічку
	EventOperatorAction    EventType = "operator_action"    // Дія оператора
	EventManagerAssigned   EventType = "manager_assigned"   // Призначено МГР
	EventManagerArrived    EventType = "manager_arrived"    // Прибуття МГР
	EventManagerCanceled   EventType = "manager_canceled"   // Скасування виїзду МГР
	EventAlarmFinished     EventType = "alarm_finished"     // Завершення відпрацювання
	EventDeviceBlocked     EventType = "device_blocked"     // Пристрій заблоковано
	EventDeviceUnblocked   EventType = "device_unblocked"   // Пристрій розблоковано
	EventService           EventType = "service"            // Сервісна / системна дія
)

// Event представляє подію в журналі
type Event struct {
	ID           int       // Унікальний ID події
	Time         time.Time // Час події
	ObjectID     int       // ID об'єкта
	ObjectNumber string    // Номер об'єкта
	ObjectName   string    // Назва об'єкта
	Type         EventType // Тип події
	ZoneNumber int       // Номер зони (якщо застосовно)
	Details    string    // Додаткові деталі
	UserName   string    // Користувач (для постановки/зняття)
	SC1        int       // Код кольору з БД
}

// GetTypeDisplay повертає текстовий опис типу події українською
func (e *Event) GetTypeDisplay() string {
	switch e.Type {
	case EventFire:
		return "ПОЖЕЖА"
	case EventBurglary:
		return "ПРОНИКНЕННЯ"
	case EventPanic:
		return "ТРИВОЖНА КНОПКА"
	case EventMedical:
		return "МЕДИЧНА ТРИВОГА"
	case EventGas:
		return "ГАЗОВА ТРИВОГА"
	case EventTamper:
		return "ТАМПЕР/САБОТАЖ"
	case EventFault:
		return "НЕСПРАВНІСТЬ"
	case EventRestore:
		return "ВІДНОВЛЕННЯ"
	case EventArm:
		return "ПОСТАНОВКА"
	case EventDisarm:
		return "ЗНЯТТЯ"
	case EventTest:
		return "ТЕСТ"
	case EventPowerFail:
		return "ВТРАТА 220В"
	case EventPowerOK:
		return "ВІДНОВЛЕННЯ 220В"
	case EventBatteryLow:
		return "НИЗЬКИЙ ЗАРЯД АКБ"
	case EventOnline:
		return "НА ЗВ'ЯЗКУ"
	case EventOffline:
		return "ВТРАТА ЗВ'ЯЗКУ"
	case SystemEvent:
		return "СИСТЕМА"
	case EventAlarmNotification:
		return "ТРИВОГА В СТРІЧЦІ"
	case EventOperatorAction:
		return "ОПЕРАТОР"
	case EventManagerAssigned:
		return "ПРИЗНАЧЕНО МГР"
	case EventManagerArrived:
		return "ПРИБУТТЯ МГР"
	case EventManagerCanceled:
		return "СКАСУВАННЯ МГР"
	case EventAlarmFinished:
		return "ЗАВЕРШЕНО"
	case EventDeviceBlocked:
		return "БЛОКУВАННЯ ПРИСТРОЮ"
	case EventDeviceUnblocked:
		return "РОЗБЛОКУВАННЯ ПРИСТРОЮ"
	case EventService:
		return "СЕРВІС"
	default:
		return "ПОДІЯ"
	}
}

// GetTimeDisplay повертає форматований час
func (e *Event) GetTimeDisplay() string {
	return e.Time.Format("15:04:05")
}

// GetDateTimeDisplay повертає повну дату і час
func (e *Event) GetDateTimeDisplay() string {
	return e.Time.Format("02.01.2006 15:04:05")
}

// IsCritical повертає true якщо подія критична (пожежа, несправність)
func (e *Event) IsCritical() bool {
	switch e.Type {
	case EventFire, EventBurglary, EventPanic, EventMedical, EventGas, EventTamper, EventFault, EventOffline, EventAlarmNotification, EventDeviceBlocked:
		return true
	default:
		return false
	}
}

// IsWarning повертає true якщо подія є попередженням
func (e *Event) IsWarning() bool {
	switch e.Type {
	case EventPowerFail, EventBatteryLow, EventManagerAssigned, EventManagerCanceled, EventService:
		return true
	default:
		return false
	}
}
