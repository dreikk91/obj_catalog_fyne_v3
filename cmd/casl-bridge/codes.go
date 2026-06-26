package main

import "obj_catalog_fyne_v3/pkg/models"

// ppk_in numeric codes in CASL rcomSurgard encoding.
// All codes are in the 0x00XX range (firstByte=0x00) so they resolve
// to rcomSurgardDictionary[0x00][secondByte] and never return undefined
// from translateMsg, which would crash EventsOrm.saveDeviceEvent.
const (
	codeFire      = 0x00A0 // 160 → SMOKE
	codeBurglary  = 0x00B0 // 176 → ALM_GENERAL
	codePanic     = 0x00B0 // 176 → ALM_GENERAL
	codeMedical   = 0x00B7 // 183 → MEDICAL_ALARM
	codeGas       = 0x00A3 // 163 → CO_GAS
	codeTamper    = 0x0005 // 5   → TAMPER_ON
	codeFault     = 0x00BB // 187 → LOST_CONNECTION
	codeRestore   = 0x00B1 // 177 → NORM_GENERAL
	codeArm       = 0x0064 // 100 → ENABLED
	codeDisarm    = 0x0065 // 101 → DISABLED
	codePowerFail = 0x0068 // 104 → NO_220
	codePowerOK   = 0x0069 // 105 → OK_220
	codeBattLow   = 0x00A8 // 168 → BTTR_FAIL
	codeTest      = 0x0069 // 105 → OK_220  (periodic alive signal)
	codeOffline   = 0x0061 // 97  → PPK_NO_CONN
	codeOnline    = 0x0069 // 105 → OK_220
	codeSystem    = 0x00B0 // 176 → ALM_GENERAL
	codeUnknown   = 0x00B0 // 176 → ALM_GENERAL
)

// eventCode maps a models.EventType to a numeric ppk_in code.
// Returns (code, isRestore).
func eventCode(t models.EventType) (int, bool) {
	switch t {
	case models.EventFire:
		return codeFire, false
	case models.EventBurglary:
		return codeBurglary, false
	case models.EventPanic:
		return codePanic, false
	case models.EventMedical:
		return codeMedical, false
	case models.EventGas:
		return codeGas, false
	case models.EventTamper:
		return codeTamper, false
	case models.EventFault:
		return codeFault, false
	case models.EventRestore:
		return codeRestore, true
	case models.EventArm:
		return codeArm, false
	case models.EventDisarm:
		return codeDisarm, true
	case models.EventPowerFail:
		return codePowerFail, false
	case models.EventPowerOK:
		return codePowerOK, true
	case models.EventBatteryLow:
		return codeBattLow, false
	case models.EventTest:
		return codeTest, false
	case models.EventOffline:
		return codeOffline, false
	case models.EventOnline:
		return codeOnline, true
	case models.SystemEvent, models.EventService:
		return codeSystem, false
	default:
		return codeUnknown, false
	}
}
