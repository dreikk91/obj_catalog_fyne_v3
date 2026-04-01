// Package models - структура тривоги
package models

import "time"

// AlarmType визначає тип тривоги
type AlarmType string

const (
	AlarmFire        AlarmType = "fire"         // Пожежа
	// AlarmBurglary    AlarmType = "burglary"     // Проникнення/охоронна тривога
	AlarmPanic       AlarmType = "panic"        // Тривожна кнопка/напад
	AlarmMedical     AlarmType = "medical"      // Медична тривога
	AlarmGas         AlarmType = "gas"          // Газова тривога
	AlarmTamper      AlarmType = "tamper"       // Саботаж/тампер
	AlarmFault       AlarmType = "fault"        // Несправність
	AlarmPowerFail   AlarmType = "power_fail"   // Втрата 220В
	AlarmBatteryLow  AlarmType = "battery_low"  // Низький заряд АКБ
	AlarmOffline     AlarmType = "offline"      // Втрата зв'язку
	AlarmSystemEvent AlarmType = "system_event" // Системна подія
	AlarmOperator    AlarmType = "ALARM_TYPE_OPERATOR"     // Операторська тривога
	AlarmDevice      AlarmType = "ALARM_TYPE_DEVICE"     // Пристроєва тривога
	AlarmMobile      AlarmType = "ALARM_TYPE_MOBILE"     // Мобільна тривога
	AlarmBurglary    AlarmType = "BURGLARY_ALARM"    // Охоронна тривога
	AlarmEliminated  AlarmType = "ALARM_ELIMINATED"  // Ліквідована тривога
	AlarmAcTrouble   AlarmType = "AC_TROUBLE"      // Проблеми з живленням
	AlarmExit        AlarmType = "EXIT_ALARM"        // Вихідна тривога
	// AlarmFire        AlarmType = "FIRE_ALARM"        // Пожежна тривога
	AlarmFireTrouble AlarmType = "FIRE_TROUBLE"    // Проблеми з пожежною сигналізацією
	AlarmNotification AlarmType = "notification" // Повідомлення
	
)

// Alarm представляє активну тривогу, що потребує обробки
type Alarm struct {
	ID           int       // Унікальний ID тривоги
	ObjectID     int       // ID об'єкта
	ObjectNumber string    // Номер об'єкта (людський формат)
	ObjectName   string    // Назва об'єкта (для швидкого відображення)
	Address     string    // Адреса об'єкта
	Time        time.Time // Час виникнення
	Details     string    // Деталі тривоги
	Type        AlarmType // Тип тривоги
	ZoneNumber  int       // Номер зони (шлейфу)
	ZoneName    string    // Назва зони
	SC1         int       // Код кольору з БД
	IsProcessed bool      // Чи оброблена тривога
	ProcessedBy string    // Ким оброблена
	ProcessNote string    // Примітка при обробці
}

// GetTypeDisplay повертає текстовий опис типу тривоги українською
func (a *Alarm) GetTypeDisplay() string {
	switch a.Type {
	case AlarmFire:
		return "ПОЖЕЖА"
	case AlarmBurglary:
		return "ПРОНИКНЕННЯ"
	case AlarmPanic:
		return "ТРИВОЖНА КНОПКА"
	case AlarmMedical:
		return "МЕДИЧНА ТРИВОГА"
	case AlarmGas:
		return "ГАЗОВА ТРИВОГА"
	case AlarmTamper:
		return "ТАМПЕР/САБОТАЖ"
	case AlarmFault:
		return "НЕСПРАВНІСТЬ"
	case AlarmPowerFail:
		return "ВТРАТА 220В"
	case AlarmBatteryLow:
		return "НИЗЬКИЙ ЗАРЯД АКБ"
	case AlarmOffline:
		return "ВТРАТА ЗВ'ЯЗКУ"
	case AlarmSystemEvent:
		return "СИСТЕМА"
	case AlarmOperator:
		return "Тривога від оператора"
	case AlarmDevice:
		return "Тривога від пристрою"
	case AlarmMobile:
		return "Тривога від мобільного додатку"
	case AlarmEliminated:
		return "Ліквідована тривога"
	case AlarmAcTrouble:
		return "Проблеми з живленням"
	case AlarmExit:
		return "Вихідна тривога"
	case AlarmFireTrouble:
		return "Проблеми з пожежною сигналізацією"
	case AlarmNotification:
		return "Попадання тривоги в стрічку"
	default:
		return "ПОДІЯ"
	}
}

// GetTimeDisplay повертає форматований час
func (a *Alarm) GetTimeDisplay() string {
	return a.Time.Format("15:04:05")
}

// GetDateTimeDisplay повертає повну дату і час
func (a *Alarm) GetDateTimeDisplay() string {
	return a.Time.Format("02.01.2006 15:04:05")
}

// IsCritical повертає true якщо тривога критична і має підсвічуватись як пріоритетна.
func (a *Alarm) IsCritical() bool {
	switch a.Type {
	case AlarmFire, AlarmBurglary, AlarmPanic, AlarmMedical, AlarmGas, AlarmTamper, AlarmFault, AlarmOffline, AlarmOperator, AlarmDevice, AlarmMobile:
		return true
	default:
		return false
	}
}
