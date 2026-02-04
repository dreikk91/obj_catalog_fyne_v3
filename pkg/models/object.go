package models

import (
	"time"
)

// ObjectStatus визначає стан об'єкта
type ObjectStatus int

const (
	StatusNormal  ObjectStatus = iota // Норма (Зелений)
	StatusFire                        // Пожежа (Червоний)
	StatusFault                       // Несправність (Жовтий)
	StatusOffline                     // Немає зв'язку (Сірий)
)

// PowerSource визначає джерело живлення
type PowerSource int

const (
	PowerMains   PowerSource = iota // 220В
	PowerBattery                    // АКБ
)

// Object - основна структура об'єкта
type Object struct {
	ID          int
	Name        string
	Address     string
	ContractNum string
	Phone       string
	Status      ObjectStatus
	StatusText  string

	// Детальні стани для відображення в списку
	AlarmState     int64
	GuardState     int64
	TechAlarmState int64
	IsConnState    int64

	// Інформація про прилад
	DeviceType      string      // Тип приладу (напр. "Тірас-16П")
	PanelMark       string      // Марка ППК (напр. "Тірас-8П")
	SignalStrength  string      // Рівень сигналу (напр. "[-61 dBm]" або "AVD")
	GSMLevel        int         // Рівень GSM сигналу (0-100%)
	LastTestTime    time.Time   // Час останнього тесту
	LastMessageTime time.Time   // Час останньої події
	PowerSource     PowerSource // Поточне джерело живлення
	AutoTestHours   int         // Період автотесту в годинах

	SIM1        string // Номер SIM 1
	SIM2        string // Номер SIM 2
	ObjChan     int    // Канал зв'язку (1=Автододзвон, 5=GPRS, інше=Інший)
	AkbState    int64  // Стан АКБ
	PowerFault  int64  // Несправність 220В (0=ОК, >0=Тривога)
	TestControl int64  // Контроль тесту
	TestTime    int64  // Час тесту (період)
	Phones1     string // Телефон на об'єкті
	Notes1      string // Додаткова інформація
	Location1   string // Розташування

	// Технічні стани
	IsUnderGuard bool
	IsConnOK     bool

	// Списки (можуть завантажуватись ліниво)
	Zones    []Zone
	Contacts []Contact
}

// GetStatusDisplay повертає текстовий статус об'єкта
func (o *Object) GetStatusDisplay() string {
	if o.StatusText != "" {
		return o.StatusText
	}
	switch o.Status {
	case StatusNormal:
		return "НОРМА"
	case StatusFire:
		return "ПОЖЕЖА"
	case StatusFault:
		return "НЕСПРАВНІСТЬ"
	case StatusOffline:
		return "НЕМАЄ ЗВ'ЯЗКУ"
	default:
		return "НЕВІДОМО"
	}
}

// Contact - відповідальна особа
type Contact struct {
	Name     string
	Position string
	Phone    string
	Priority int
	CodeWord string // Кодове слово
}

// LastEvent зберігає час останньої події для відображення
type LastEvent struct {
	Time time.Time
	Text string
}

// TestMessage представляє тестове повідомлення з TRPMSG
type TestMessage struct {
	Time    time.Time
	Info    string
	Details string
}
