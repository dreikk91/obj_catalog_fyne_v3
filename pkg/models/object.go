package models

import (
	"time"
)

// ObjectStatus РІРёР·РЅР°С‡Р°С” СЃС‚Р°РЅ РѕР±'С”РєС‚Р°
type ObjectStatus int

const (
	StatusNormal  ObjectStatus = iota // РќРѕСЂРјР° (Р—РµР»РµРЅРёР№)
	StatusFire                        // РџРѕР¶РµР¶Р° (Р§РµСЂРІРѕРЅРёР№)
	StatusFault                       // РќРµСЃРїСЂР°РІРЅС–СЃС‚СЊ (Р–РѕРІС‚РёР№)
	StatusOffline                     // РќРµРјР°С” Р·РІ'СЏР·РєСѓ (РЎС–СЂРёР№)
)

// PowerSource РІРёР·РЅР°С‡Р°С” РґР¶РµСЂРµР»Рѕ Р¶РёРІР»РµРЅРЅСЏ
type PowerSource int

const (
	PowerMains   PowerSource = iota // 220Р’
	PowerBattery                    // РђРљР‘
)

// Object - РѕСЃРЅРѕРІРЅР° СЃС‚СЂСѓРєС‚СѓСЂР° РѕР±'С”РєС‚Р°
type Object struct {
	ID          int
	Name        string
	Address     string
	ContractNum string
	Phone       string
	Status      ObjectStatus
	StatusText  string

	// Р”РµС‚Р°Р»СЊРЅС– СЃС‚Р°РЅРё РґР»СЏ РІС–РґРѕР±СЂР°Р¶РµРЅРЅСЏ РІ СЃРїРёСЃРєСѓ
	AlarmState        int64
	GuardState        int64
	TechAlarmState    int64
	IsConnState       int64
	BlockedArmedOnOff int16

	// Р†РЅС„РѕСЂРјР°С†С–СЏ РїСЂРѕ РїСЂРёР»Р°Рґ
	DeviceType      string      // РўРёРї РїСЂРёР»Р°РґСѓ (РЅР°РїСЂ. "РўС–СЂР°СЃ-16Рџ")
	PanelMark       string      // РњР°СЂРєР° РџРџРљ (РЅР°РїСЂ. "РўС–СЂР°СЃ-8Рџ")
	SignalStrength  string      // Р С–РІРµРЅСЊ СЃРёРіРЅР°Р»Сѓ (РЅР°РїСЂ. "[-61 dBm]" Р°Р±Рѕ "AVD")
	GSMLevel        int         // Р С–РІРµРЅСЊ GSM СЃРёРіРЅР°Р»Сѓ (0-100%)
	LastTestTime    time.Time   // Р§Р°СЃ РѕСЃС‚Р°РЅРЅСЊРѕРіРѕ С‚РµСЃС‚Сѓ
	LastMessageTime time.Time   // Р§Р°СЃ РѕСЃС‚Р°РЅРЅСЊРѕС— РїРѕРґС–С—
	PowerSource     PowerSource // РџРѕС‚РѕС‡РЅРµ РґР¶РµСЂРµР»Рѕ Р¶РёРІР»РµРЅРЅСЏ
	AutoTestHours   int         // РџРµСЂС–РѕРґ Р°РІС‚РѕС‚РµСЃС‚Сѓ РІ РіРѕРґРёРЅР°С…

	SIM1        string // РќРѕРјРµСЂ SIM 1
	SIM2        string // РќРѕРјРµСЂ SIM 2
	SubServerA  string // Підсервер A (SBSA)
	SubServerB  string // Підсервер B (SBSB)
	ObjChan     int    // РљР°РЅР°Р» Р·РІ'СЏР·РєСѓ (1=РђРІС‚РѕРґРѕРґР·РІРѕРЅ, 5=GPRS, С–РЅС€Рµ=Р†РЅС€РёР№)
	AkbState    int64  // РЎС‚Р°РЅ РђРљР‘
	PowerFault  int64  // РќРµСЃРїСЂР°РІРЅС–СЃС‚СЊ 220Р’ (0=РћРљ, >0=РўСЂРёРІРѕРіР°)
	TestControl int64  // РљРѕРЅС‚СЂРѕР»СЊ С‚РµСЃС‚Сѓ
	TestTime    int64  // Р§Р°СЃ С‚РµСЃС‚Сѓ (РїРµСЂС–РѕРґ)
	Phones1     string // РўРµР»РµС„РѕРЅ РЅР° РѕР±'С”РєС‚С–
	Notes1      string // Р”РѕРґР°С‚РєРѕРІР° С–РЅС„РѕСЂРјР°С†С–СЏ
	Location1   string // Розташування
	LaunchDate  string // Дата запуску (OBJECTS_INFO.RESERVTEXT)

	// РўРµС…РЅС–С‡РЅС– СЃС‚Р°РЅРё
	IsUnderGuard bool
	IsConnOK     bool

	// РЎРїРёСЃРєРё (РјРѕР¶СѓС‚СЊ Р·Р°РІР°РЅС‚Р°Р¶СѓРІР°С‚РёСЃСЊ Р»С–РЅРёРІРѕ)
	Zones    []Zone
	Contacts []Contact
}

// GetStatusDisplay РїРѕРІРµСЂС‚Р°С” С‚РµРєСЃС‚РѕРІРёР№ СЃС‚Р°С‚СѓСЃ РѕР±'С”РєС‚Р°
func (o *Object) GetStatusDisplay() string {
	if o.StatusText != "" {
		return o.StatusText
	}
	switch o.Status {
	case StatusNormal:
		return "РќРћР РњРђ"
	case StatusFire:
		return "РџРћР–Р•Р–Рђ"
	case StatusFault:
		return "РќР•РЎРџР РђР’РќР†РЎРўР¬"
	case StatusOffline:
		return "РќР•РњРђР„ Р—Р’'РЇР—РљРЈ"
	default:
		return "РќР•Р’Р†Р”РћРњРћ"
	}
}

// Contact - РІС–РґРїРѕРІС–РґР°Р»СЊРЅР° РѕСЃРѕР±Р°
type Contact struct {
	Name     string
	Position string
	Phone    string
	Priority int
	CodeWord string // РљРѕРґРѕРІРµ СЃР»РѕРІРѕ
}

// LastEvent Р·Р±РµСЂС–РіР°С” С‡Р°СЃ РѕСЃС‚Р°РЅРЅСЊРѕС— РїРѕРґС–С— РґР»СЏ РІС–РґРѕР±СЂР°Р¶РµРЅРЅСЏ
type LastEvent struct {
	Time time.Time
	Text string
}

// TestMessage РїСЂРµРґСЃС‚Р°РІР»СЏС” С‚РµСЃС‚РѕРІРµ РїРѕРІС–РґРѕРјР»РµРЅРЅСЏ Р· TRPMSG
type TestMessage struct {
	Time    time.Time
	Info    string
	Details string
}
