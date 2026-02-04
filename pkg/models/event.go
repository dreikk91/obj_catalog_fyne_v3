// Package models - структура події для журналу
package models

import "time"

// EventType визначає тип події
type EventType string

const (
	EventFire       EventType = "fire"       // Пожежа
	EventFault      EventType = "fault"      // Несправність
	EventRestore    EventType = "restore"    // Відновлення
	EventArm        EventType = "arm"        // Постановка під охорону
	EventDisarm     EventType = "disarm"     // Зняття з охорони
	EventTest       EventType = "test"       // Тестовий сигнал
	EventPowerFail  EventType = "power_fail" // Втрата 220В
	EventPowerOK    EventType = "power_ok"   // Відновлення 220В
	EventBatteryLow EventType = "batt_low"   // Низький заряд АКБ
	EventOnline     EventType = "online"     // Прилад на зв'язку
	EventOffline    EventType = "offline"    // Втрата зв'язку
)

// Event представляє подію в журналі
type Event struct {
	ID         int       // Унікальний ID події
	Time       time.Time // Час події
	ObjectID   int       // ID об'єкта
	ObjectName string    // Назва об'єкта
	Type       EventType // Тип події
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
	return e.Type == EventFire || e.Type == EventFault || e.Type == EventOffline
}

// IsWarning повертає true якщо подія є попередженням
func (e *Event) IsWarning() bool {
	return e.Type == EventPowerFail || e.Type == EventBatteryLow
}
