package data

import (
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/models"
)

func classifyCASLEventType(code string) models.EventType {
	value := strings.ToUpper(strings.TrimSpace(code))
	valueLower := strings.ToLower(strings.TrimSpace(code))

	switch {
	case strings.Contains(value, "GRD_OBJ_NOTIF"):
		return models.EventType(models.AlarmBurglary)
	case strings.Contains(value, "GRD_OBJ_MGR_CANCEL"), strings.Contains(value, "GRD_OBJ_FINISH"):
		return models.EventRestore
	case strings.Contains(value, "GRD_OBJ_PICK"), strings.Contains(value, "GRD_OBJ_ASS_MGR"), strings.Contains(value, "GRD_OBJ_"):
		return models.SystemEvent
	case strings.Contains(value, "PANIC"), strings.Contains(value, "COERCION"), strings.Contains(value, "ATTACK"), strings.Contains(value, "ALM_BTN_PRS"):
		return models.EventPanic
	case strings.Contains(valueLower, "тривожна кноп"), strings.Contains(valueLower, "кнопка тривог"), strings.Contains(valueLower, "напад"), strings.Contains(valueLower, "панік"):
		return models.EventPanic
	case strings.Contains(value, "MEDICAL"):
		return models.EventMedical
	case strings.Contains(valueLower, "медич"):
		return models.EventMedical
	case strings.Contains(value, "GAS_ALARM"), strings.Contains(value, "CO_GAS"), strings.Contains(value, "GAS_SUPERVISORY"):
		return models.EventGas
	case strings.Contains(valueLower, "газ"):
		return models.EventGas
	case strings.Contains(value, "BURGLARY"), strings.Contains(value, "INTRUSION"), strings.Contains(value, "BRUTFORS"), strings.Contains(value, "ZONE_ALM"), strings.Contains(value, "ALM_INNER_ZONE"):
		return models.EventBurglary
	case strings.Contains(valueLower, "проник"), strings.Contains(valueLower, "злом"), strings.Contains(valueLower, "охорон") && strings.Contains(valueLower, "тривог"):
		return models.EventBurglary
	case strings.Contains(value, "SABOTAGE"), strings.Contains(value, "TAMPER"), strings.Contains(value, "SENS_TAMP"), strings.Contains(value, "EXT_MOD_TAMP"), strings.Contains(value, "HUB_TAMP"):
		return models.EventTamper
	case strings.Contains(valueLower, "саботаж"), strings.Contains(valueLower, "тампер"):
		return models.EventTamper
	case strings.Contains(value, "FIRE"), strings.Contains(value, "SMOKE"), strings.Contains(value, "HEAT"):
		return models.EventFire
	case strings.Contains(valueLower, "пожеж"), strings.Contains(valueLower, "дим"), strings.Contains(valueLower, "тепл"):
		return models.EventFire
	case strings.Contains(value, "R402"),
		strings.Contains(value, "GROUP_ON"),
		strings.Contains(value, "GROUP_ON_USER"),
		strings.Contains(value, "ON_WITH_PPL"),
		strings.Contains(value, "ON_BFR_TIME"),
		strings.Contains(value, "ON_AFTR_TIME"),
		strings.Contains(value, "_ARMED"),
		value == "ARM":
		return models.EventArm
	case strings.Contains(valueLower, "взят"),
		strings.Contains(valueLower, "під охорон"),
		strings.Contains(valueLower, "взятие"),
		strings.Contains(valueLower, "постановк"):
		return models.EventArm
	case strings.Contains(value, "R401"),
		strings.Contains(value, "GROUP_OFF"),
		strings.Contains(value, "GROUP_OFF_USER"),
		strings.Contains(value, "OFF_WITH_PPL"),
		strings.Contains(value, "OFF_BFR_TIME"),
		strings.Contains(value, "OFF_AFTR_TIME"),
		strings.Contains(value, "_DISARM"),
		value == "DISARM":
		return models.EventDisarm
	case strings.Contains(valueLower, "знят"),
		strings.Contains(valueLower, "виключ"),
		strings.Contains(valueLower, "сняти"),
		strings.Contains(valueLower, "снятие"):
		return models.EventDisarm
	case strings.Contains(value, "ID_HOZ"),
		strings.Contains(value, "USER_ACCESS"),
		strings.Contains(valueLower, "ідентифікац"),
		strings.Contains(valueLower, "идентификац"),
		strings.Contains(valueLower, "користувач"),
		strings.Contains(valueLower, "пользовател"):
		return models.SystemEvent
	case value == "E627", value == "R627", value == "E628", value == "R628":
		return models.SystemEvent
	case strings.Contains(value, "UPD_START"), strings.Contains(value, "UPD_END"), strings.Contains(value, "FIRMWARE"),
		strings.Contains(valueLower, "оновлен"), strings.Contains(valueLower, "застосуван") && strings.Contains(valueLower, "налаштуван"):
		return models.SystemEvent
	case strings.Contains(value, "ALM_"),
		strings.Contains(value, "_ALARM"),
		strings.Contains(valueLower, "тривога"),
		strings.Contains(valueLower, "тревог"):
		return models.EventFault
	case strings.Contains(value, "ZONE_NORM"),
		strings.Contains(value, "NORM_"),
		strings.Contains(valueLower, "норма"),
		strings.Contains(valueLower, "віднов"),
		strings.Contains(valueLower, "восстанов"):
		return models.EventRestore
	case strings.Contains(value, "NO_CONN"), strings.Contains(value, "CONNECTION_LOST"), strings.Contains(value, "OFFLINE"), strings.Contains(value, "LOST"):
		return models.EventOffline
	case strings.Contains(valueLower, "нема зв"), strings.Contains(valueLower, "втрата зв"), strings.Contains(valueLower, "відсутн") && strings.Contains(valueLower, "зв"):
		return models.EventOffline
	case strings.Contains(value, "RECOVER"), strings.Contains(value, "RESTORE"),
		strings.HasPrefix(value, "OK_"), strings.Contains(value, "OK_220"), strings.Contains(value, "POWER_OK"),
		strings.HasSuffix(value, "_OK"), strings.HasPrefix(value, "R"):
		return models.EventRestore
	case strings.Contains(valueLower, "віднов"), strings.Contains(valueLower, "норма"):
		return models.EventRestore
	case (strings.Contains(value, "POWER") || strings.Contains(value, "NO_220") || strings.Contains(value, "MAIN_AC_LOSS")) &&
		!strings.Contains(value, "POWER_OK") &&
		!strings.Contains(value, "OK_220") &&
		!strings.HasSuffix(value, "_OK"):
		return models.EventPowerFail
	case strings.Contains(valueLower, "220") && strings.Contains(valueLower, "живлен") &&
		(strings.Contains(valueLower, "втра") || strings.Contains(valueLower, "пропаж")):
		return models.EventPowerFail
	case strings.Contains(value, "BATT"), strings.Contains(value, "BATTERY") && strings.Contains(value, "LOW"):
		return models.EventBatteryLow
	case strings.Contains(valueLower, "акб") && strings.Contains(valueLower, "розряд"):
		return models.EventBatteryLow
	case strings.Contains(value, "TEST"):
		return models.EventTest
	case strings.Contains(value, "POLL"), strings.Contains(value, "PING"), strings.Contains(value, "PONG"):
		return models.EventTest
	default:
		return models.EventFault
	}
}

func mapCASLTapeEventType(raw string) models.EventType {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "fire", "FIRE_ALARM":
		return models.EventFire
	case "burglary", "BURGLARY_ALARM":
		return models.EventBurglary
	case "panic":
		return models.EventPanic
	case "medical":
		return models.EventMedical
	case "gas":
		return models.EventGas
	case "tamper":
		return models.EventTamper
	case "fault":
		return models.EventFault
	case "restore":
		return models.EventRestore
	case "arm":
		return models.EventArm
	case "disarm":
		return models.EventDisarm
	case "test":
		return models.EventTest
	case "poll":
		return models.EventTest
	case "power_fail", "AC_TROUBLE":
		return models.EventPowerFail
	case "power_ok":
		return models.EventPowerOK
	case "batt_low":
		return models.EventBatteryLow
	case "offline":
		return models.EventOffline
	case "online":
		return models.EventOnline
	case "system", "ppk_action", "ppk_service", "system_event", "system_action", "m3_in":
		return models.SystemEvent
	case "user_action":
		return models.EventOperator
	case "mob_user_action", "ALARM_TYPE_MOBILE":
		return models.EventMobile
	case "ALARM_TYPE_OPERATOR":
		return models.EventOperator
	case "ALARM_ELIMINATED":
		return models.EventEliminated
	case "FIRE_TROUBLE":
		return models.EventFireTrouble
	default:
		return classifyCASLEventType(value)
	}
}

func resolveCASLTemplate(source map[string]string, key string) string {
	if len(source) == 0 {
		return ""
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}

	upper := strings.ToUpper(key)

	candidates := []string{key, upper, strings.ToLower(key)}

	if len(upper) >= 2 && (upper[0] == 'E' || upper[0] == 'R') {
		tail := upper[1:]
		isNumericTail := len(tail) >= 1
		for _, ch := range tail {
			if ch < '0' || ch > '9' {
				isNumericTail = false
				break
			}
		}
		if isNumericTail {

			candidates = append(candidates, tail)

			if upper[0] == 'E' {
				candidates = append(candidates, "R"+tail)
			} else {
				candidates = append(candidates, "E"+tail)
			}
		}
	} else {

		isAllDigits := len(upper) >= 1
		for _, ch := range upper {
			if ch < '0' || ch > '9' {
				isAllDigits = false
				break
			}
		}
		if isAllDigits {
			candidates = append(candidates, "E"+upper, "R"+upper)
		}
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, dup := seen[candidate]; dup {
			continue
		}
		seen[candidate] = struct{}{}
		if value := strings.TrimSpace(source[candidate]); value != "" {
			return value
		}
	}
	return ""
}

func applyCASLNumberTemplate(template string, number int) string {
	if strings.TrimSpace(template) == "" {
		return ""
	}
	replacements := map[string]string{
		"{number}": strconv.Itoa(number),
		"{zone}":   strconv.Itoa(number),
		"{line}":   strconv.Itoa(number),
		"%number%": strconv.Itoa(number),
		"%zone%":   strconv.Itoa(number),
		"%line%":   strconv.Itoa(number),
	}
	out := template
	for from, to := range replacements {
		out = strings.ReplaceAll(out, from, to)
	}
	return strings.TrimSpace(out)
}

func caslTemplateHasNumberPlaceholder(template string) bool {
	template = strings.TrimSpace(template)
	if template == "" {
		return false
	}
	return strings.Contains(template, "{number}") ||
		strings.Contains(template, "{zone}") ||
		strings.Contains(template, "{line}") ||
		strings.Contains(template, "%number%") ||
		strings.Contains(template, "%zone%") ||
		strings.Contains(template, "%line%")
}

func caslMessageKeyNeedsNumberSuffix(key string) bool {
	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		return false
	}
	switch {
	case strings.Contains(key, "GROUP"),
		strings.Contains(key, "ZONE"),
		strings.Contains(key, "LINE"),
		strings.Contains(key, "ID_HOZ"):
		return true
	default:
		return false
	}
}

func finalizeCASLDecodedTemplate(template string, number int, messageKey string) string {
	out := applyCASLNumberTemplate(template, number)
	if out == "" {
		return ""
	}
	if number > 0 && caslMessageKeyNeedsNumberSuffix(messageKey) && !caslTemplateHasNumberPlaceholder(template) {
		return out + " № " + strconv.Itoa(number)
	}
	return out
}

func shouldAppendCASLLineDescription(code string, contactID string, details string) bool {
	key := strings.ToUpper(strings.TrimSpace(code))
	if decoded, ok := decodeCASLProtocolCode(code, ""); ok {
		key = strings.ToUpper(strings.TrimSpace(decoded.MessageKey))
	}
	if key == "" {
		key = strings.ToUpper(strings.TrimSpace(contactID))
	}

	switch {
	case strings.Contains(key, "GROUP"),
		strings.Contains(key, "ID_HOZ"),
		strings.Contains(key, "USER_ACCESS"),
		strings.Contains(key, "PRIMUS"):
		return false
	}

	lowerDetails := strings.ToLower(strings.TrimSpace(details))
	switch {
	case strings.Contains(lowerDetails, "груп"),
		strings.Contains(lowerDetails, "користувач"),
		strings.Contains(lowerDetails, "ідентифікац"):
		return false
	}

	return true
}

func parseCASLCodeBytes(code string) (byte, byte, bool) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, 0, false
	}
	value, err := strconv.ParseInt(code, 10, 64)
	if err != nil || value < 0 {
		return 0, 0, false
	}
	if value > 0xFFFF {
		value %= 0x10000
	}
	return byte((value >> 8) & 0xFF), byte(value & 0xFF), true
}

func caslProtocolModelFromDeviceType(deviceType string) caslProtocolModel {
	switch strings.TrimSpace(deviceType) {
	case "TYPE_DEVICE_Ajax_SIA", "TYPE_DEVICE_Bron_SIA":
		return caslProtocolModelSIA
	case "TYPE_DEVICE_Dunay_4_3", "TYPE_DEVICE_Dunay_4_3S", "TYPE_DEVICE_VBD4_ECOM", "TYPE_DEVICE_VBD_16":
		return caslProtocolModelVBD4
	case "TYPE_DEVICE_Dozor_4", "TYPE_DEVICE_Dozor_8", "TYPE_DEVICE_Dozor_8MG":
		return caslProtocolModelDozor
	case "TYPE_DEVICE_Dunay_16_32", "TYPE_DEVICE_Dunay_8_32", "TYPE_DEVICE_Dunay_PSPN_ECOM":
		return caslProtocolModelD128
	default:
		return caslProtocolModelRcom
	}
}

func decodedStatic(key string) (caslDecodedEventCode, bool) {
	if strings.TrimSpace(key) == "" {
		return caslDecodedEventCode{}, false
	}
	return caslDecodedEventCode{MessageKey: key}, true
}

func decodedWithOffset(key string, b2 byte, offset int) (caslDecodedEventCode, bool) {
	if strings.TrimSpace(key) == "" {
		return caslDecodedEventCode{}, false
	}
	return caslDecodedEventCode{
		MessageKey: key,
		Number:     int(b2) + offset,
		HasNumber:  true,
	}, true
}

func decodedWithSecondByte(key string, b2 byte) (caslDecodedEventCode, bool) {
	return decodedWithOffset(key, b2, 0)
}

func decodeCASLSystemCode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	switch b1 {
	case 0x00:
		switch b2 {
		case 0xB3:
			return decodedStatic("BAN_TIME")
		case 0xBD:
			return decodedStatic("REQUIRED_GROUP_ON")
		case 0x60:
			return decodedStatic("PPK_CONN_OK")
		case 0x66:
			return decodedStatic("SUSPICIOUS_ACTIVITY")
		case 0x67:
			return decodedStatic("SABOTAGE")
		}
	case 0x01:
		switch b2 {
		case 0x61:
			return decodedStatic("OO_NO_POLL")
		case 0x62:
			return decodedStatic("OO_NO_PING")
		}
	}
	return caslDecodedEventCode{}, false
}

func decodeCASLRcomSurgardCode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	if b1 == 0x3B {
		switch b2 {
		case 0x00:
			return decodedStatic("REP_FIRMW_4L")
		case 0x01:
			return decodedStatic("END_FIRMW_4L")
		case 0x02:
			return decodedStatic("REQ_REP_FIRMW_4L")
		case 0x03:
			return decodedStatic("REC_CONFIG_4L")
		case 0x04:
			return decodedStatic("END_CONFIG_4L")
		case 0x05:
			return decodedStatic("PPK_SIM_4L")
		case 0x06:
			return decodedStatic("PPK_IMEIL_4L")
		case 0x07:
			return decodedStatic("PPK_COORD_4L")
		case 0x08:
			return decodedStatic("PPK_CSQ_4L")
		case 0x09:
			return decodedStatic("CONTROL_4L")
		}
	}

	if b1 == 0x08 {
		switch b2 {
		case 0x27:
			return decodedWithOffset("ID_HOZ", b2, -0x0f)
		case 0x28:
			return decodedStatic("SET_INPUT_CONTROL")
		case 0x29:
			return decodedStatic("KEYPAD_PROGRAMMING")
		case 0x2A:
			return decodedStatic("PROGRAMMING_CP_USB")
		case 0x2B:
			return decodedStatic("PROGRAMMING_CP_INTERNET")
		case 0x2C:
			return decodedStatic("MANAGEMENT_FROM_DUNAY")
		case 0x2D:
			return decodedStatic("REMOTE_CONTROL")
		case 0x2E:
			return decodedStatic("KEYFOB_KEYBOARD")
		default:
			return decodedWithOffset("ID_HOZ", b2, -0x0f)
		}
	}

	switch b1 {
	case 0x00:
		switch b2 {
		case 0x02:
			return decodedStatic("CANNOT_AUTO_ARM")
		case 0x03:
			return decodedStatic("DEVICE_TEMPORARILY_DEACTIVATED")
		case 0x04:
			return decodedStatic("DEVICE_ACTIVE_AGAIN")
		case 0x05:
			return decodedStatic("TAMPER_ON")
		case 0x06:
			return decodedStatic("DEACTIVATED_AUTO_MAX_ALARMS")
		case 0x07:
			return decodedStatic("RESTORED_AFTER_AUTO_DEACTIVATION")
		case 0x08:
			return decodedStatic("SYSTEM_RESTORED_AFTER_ALARM_BY_USER")
		case 0x09:
			return decodedStatic("INTRUSION_VERIFIER")
		case 0x0A:
			return decodedStatic("PANIC_VERIFIER")
		case 0x2F:
			return decodedStatic("MALFUNCTION_DURING_ARMING_SYSTEM_INTEGRITY_CHECK")
		case 0x57:
			return decodedStatic("SERVER_CONNECTION_VIA_ETHERNET_LOST")
		case 0x58:
			return decodedStatic("SERVER_CONNECTION_VIA_ETHERNET_RESTORED")
		case 0x59:
			return decodedStatic("PHOTOVERIFICATION")
		case 0x61:
			return decodedStatic("PPK_NO_CONN")
		case 0x62:
			return decodedStatic("DIRECT_ERROR")
		case 0x63:
			return decodedStatic("PPK_BAD")
		case 0x64:
			return decodedStatic("ENABLED")
		case 0x65:
			return decodedStatic("DISABLED")
		case 0x68:
			return decodedStatic("NO_220")
		case 0x69:
			return decodedStatic("OK_220")
		case 0x6A:
			return decodedStatic("ACC_OK")
		case 0x6B:
			return decodedStatic("ACC_BAD")
		case 0x6C:
			return decodedStatic("DOOR_OP")
		case 0x6D:
			return decodedStatic("DOOR_CL")
		case 0x6E:
			return decodedStatic("SERVER_CONNECTION_VIA_CELLULAR_LOST")
		case 0x6F:
			return decodedStatic("SERVER_CONNECTION_VIA_CELLULAR_RESTORED")
		case 0x70:
			return decodedStatic("SERVER_CONNECTION_VIA_WI_FI_LOST")
		case 0x71:
			return decodedStatic("SERVER_CONNECTION_VIA_WI_FI_RESTORED")
		case 0x72:
			return decodedStatic("NOT_RESPONDING_DEVICE_IN_ROOM")
		case 0x73:
			return decodedStatic("PHOTO_ON_DEMAND_FEATURE_ENABLED_FOR_HUB_BY_USER")
		case 0x74:
			return decodedStatic("PHOTO_ON_DEMAND_FEATURE_DISABLED_FOR_HUB_BY_USER")
		case 0x75:
			return decodedStatic("PHOTO_BY_ALARM_SCENARIOS_FEATURE_ENABLED_FOR_HUB_BY_USER")
		case 0x76:
			return decodedStatic("PHOTO_BY_ALARM_SCENARIOS_FEATURE_DISABLED_FOR_HUB_BY_USER")
		case 0x77:
			return decodedStatic("MALFUNCTION_DETECTED_DEVICE_IN_ROOM_PULSE_EVENT")
		case 0x78:
			return decodedStatic("MALFUNCTION_RESOLVED_DEVICE_IN_ROOM")
		case 0x79:
			return decodedStatic("RING_DISCONNECTED")
		case 0x80:
			return decodedStatic("RING_CONNECTED")
		case 0xC7:
			return decodedStatic("ARMING_ATTEMPT_ON_HUB")
		case 0xC8:
			return decodedStatic("PANIC_VERIFIER_NEW")
		case 0xC9:
			return decodedStatic("INTRUSION_VERIFIER_NEW")
		case 0xCA:
			return decodedStatic("ALARM_2_MINS_AFTER_ARMING")
		case 0xCB:
			return decodedStatic("HUB_IN_BATTERY_SAVING_MODE")
		case 0xCC:
			return decodedStatic("HUB_OUT_OF_BATTERY_SAVING_MODE")
		case 0xCD:
			return decodedStatic("RECEIVED_PHOTO_BY_SCHEDULE")
		case 0xCE:
			return decodedStatic("COMPLETED_RECEAVING_PHOTO_BY_SCHEDULE")

		case 0xB9:
			return decodedStatic("FULL_REBOOT")
		}
	case 0x01:
		switch b2 {
		case 0x63:
			return decodedStatic("CHANGE_IP_OK")
		case 0x64:
			return decodedStatic("CHANGE_IP_FAIL")
		case 0x68:
			return decodedStatic("OO_NO_220")
		case 0x69:
			return decodedStatic("OO_OK_220")
		case 0x6A:
			return decodedStatic("OO_ACC_OK")
		case 0x6B:
			return decodedStatic("OO_ACC_BAD")
		case 0x6C:
			return decodedStatic("OO_DOOR_OP")
		case 0x6D:
			return decodedStatic("OO_DOOR_CL")
		}
	case 0x02:
		return decodedWithOffset("WL_ACC_OK", b2, 1)
	case 0x03:
		return decodedWithOffset("WL_ACC_BAD", b2, 1)
	case 0x04:
		return decodedWithOffset("WL_DOOR_CL", b2, 1)
	case 0x05:
		return decodedWithOffset("WL_DOOR_OP", b2, 1)
	case 0x06:
		return decodedWithOffset("WL_TROUBLE", b2, 1)
	case 0x07:
		return decodedWithOffset("WL_NORM", b2, 1)
	case 0x09:
		return decodedWithOffset("PRIMUS", b2, -0x0f)
	case 0x0A:
		return decodedWithOffset("ID_HOZ", b2, 0x10+1)
	case 0x0B:
		return decodedWithOffset("PRIMUS", b2, 0x10+1)
	case 0x0C:
		return decodedWithOffset("ID_HOZ", b2, 0x30+1)
	case 0x0D:
		return decodedWithOffset("PRIMUS", b2, 0x30+1)
	case 0x0E:
		return decodedWithOffset("ID_HOZ", b2, 0x50+1)
	case 0x0F:
		return decodedWithOffset("PRIMUS", b2, 0x50+1)
	case 0x30:
		return decodedWithOffset("AD_DOOR_OP", b2, -0x0f)
	case 0x31:
		return decodedWithOffset("OO_AD_DOOR_OP", b2, -0x0f)
	case 0x32:
		return decodedWithOffset("AD_DOOR_CL", b2, -0x0f)
	case 0x33:
		return decodedWithOffset("OO_AD_DOOR_CL", b2, -0x0f)
	case 0x34:
		return decodedWithOffset("AD_NO_CONN", b2, -0x0f)
	case 0x35:
		return decodedWithOffset("OO_AD_NO_CONN", b2, -0x0f)
	case 0x36:
		return decodedWithOffset("AD_CONN_OK", b2, -0x0f)
	case 0x37:
		return decodedWithOffset("OO_AD_CONN_OK", b2, -0x0f)
	case 0x38:
		return decodedWithOffset("AD_BAD_FOOD", b2, -0x0f)
	case 0x39:
		return decodedWithOffset("OO_ALM_AD_POWER", b2, -0x0f)
	case 0x3A:
		return decodedWithOffset("AD_FOOD_OK", b2, -0x0f)
	case 0x3B:
		return decodedWithOffset("OO_AD_POWER_OK", b2, -0x0f)
	case 0x3D:
		switch b2 {
		case 0x08:
			return decodedStatic("CONT_OUT_REL0_1")
		case 0x09:
			return decodedStatic("CONT_OUT_REL0_0")
		case 0x10:
			return decodedStatic("CONT_OUT_UK1_1")
		case 0x11:
			return decodedStatic("CONT_OUT_UK1_0")
		case 0x12:
			return decodedStatic("CONT_OUT_UK2_1")
		case 0x13:
			return decodedStatic("CONT_OUT_UK2_0")
		case 0x14:
			return decodedStatic("CONT_OUT_UK3_1")
		case 0x15:
			return decodedStatic("CONT_OUT_UK3_0")
		case 0x16:
			return decodedStatic("CONT_OUT_REL1_1")
		case 0x17:
			return decodedStatic("CONT_OUT_REL1_0")
		case 0x18:
			return decodedStatic("CONT_OUT_REL2_1")
		case 0x19:
			return decodedStatic("CONT_OUT_REL2_0")
		case 0x20:
			return decodedStatic("CONT_OUT_C1_1")
		case 0x21:
			return decodedStatic("CONT_OUT_C1_0")
		case 0x22:
			return decodedStatic("CONT_OUT_C2_1")
		case 0x23:
			return decodedStatic("CONT_OUT_C2_0")
		case 0x24:
			return decodedStatic("CONT_OUT_C3_1")
		case 0x25:
			return decodedStatic("CONT_OUT_C3_0")
		case 0x26:
			return decodedStatic("RADIO_SOCKET_1_1")
		case 0x27:
			return decodedStatic("RADIO_SOCKET_1_0")
		case 0x28:
			return decodedStatic("RADIO_SOCKET_2_1")
		case 0x29:
			return decodedStatic("RADIO_SOCKET_2_0")
		case 0x2A:
			return decodedStatic("RADIO_SOCKET_3_1")
		case 0x2B:
			return decodedStatic("RADIO_SOCKET_3_0")
		}
	case 0x3E:
		return decodedWithSecondByte("PPK_FW_VERSION", b2)
	case 0x3F:
		switch b2 {
		case 0x09, 0x8F:
			return decodedStatic("COERCION")
		case 0x10, 0x90:
			return decodedStatic("RESTART")
		case 0x11, 0x91:
			return decodedStatic("CHECK_CONN")
		case 0x12, 0x92:
			return decodedStatic("DECONCERV")
		case 0x13, 0x93:
			return decodedStatic("CONCERV")
		case 0x14, 0x94:
			return decodedStatic("EDIT_CONF")
		case 0x15, 0x95:
			return decodedStatic("ENABLED")
		case 0x16, 0x96:
			return decodedStatic("DISABLED")
		}
	case 0x40:
		return decodedWithOffset("GROUP_ON", b2, -0x0f)
	case 0x41:
		return decodedWithOffset("OO_GROUP_ON", b2, -0x0f)
	case 0x42:
		return decodedWithOffset("GROUP_ON", b2, 0x10+1)
	case 0x43:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x10+1)
	case 0x44:
		return decodedWithOffset("GROUP_ON", b2, 0x30+1)
	case 0x45:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x30+1)
	case 0x46:
		return decodedWithOffset("GROUP_ON", b2, 0x50+1)
	case 0x47:
		return decodedWithOffset("OO_GROUP_ON", b2, 0x50+1)
	case 0x48:
		return decodedWithOffset("GROUP_OFF", b2, -0x0f)
	case 0x49:
		return decodedWithOffset("OO_GROUP_OFF", b2, -0x0f)
	case 0x4A:
		return decodedWithOffset("GROUP_OFF", b2, 0x10+1)
	case 0x4B:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x10+1)
	case 0x4C:
		return decodedWithOffset("GROUP_OFF", b2, 0x30+1)
	case 0x4D:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x30+1)
	case 0x4E:
		return decodedWithOffset("GROUP_OFF", b2, 0x50+1)
	case 0x4F:
		return decodedWithOffset("OO_GROUP_OFF", b2, 0x50+1)
	case 0x50:
		return decodedWithOffset("LINE_BRK", b2, -0x0f)
	case 0x51:
		return decodedWithOffset("OO_LINE_BRK", b2, -0x0f)
	case 0x52:
		return decodedWithOffset("LINE_BRK", b2, 17)
	case 0x53:
		return decodedWithOffset("OO_LINE_BRK", b2, 0x10+1)
	case 0x54:
		return decodedWithOffset("LINE_BRK", b2, 0x30+1)
	case 0x55:
		return decodedWithOffset("OO_LINE_BRK", b2, 0x30+1)
	case 0x56:
		return decodedWithOffset("LINE_BRK", b2, 81)
	case 0x57:
		return decodedWithOffset("OO_LINE_BRK", b2, 81)
	case 0x58:
		return decodedWithOffset("LINE_NORM", b2, -0x0f)
	case 0x59:
		return decodedWithOffset("OO_LINE_NORM", b2, -0x0f)
	case 0x5A:
		return decodedWithOffset("LINE_NORM", b2, 17)
	case 0x5B:
		return decodedWithOffset("OO_LINE_NORM", b2, 17)
	case 0x5C:
		return decodedWithOffset("LINE_NORM", b2, 0x30+1)
	case 0x5D:
		return decodedWithOffset("OO_LINE_NORM", b2, 0x30+1)
	case 0x5E:
		return decodedWithOffset("LINE_NORM", b2, 81)
	case 0x5F:
		return decodedWithOffset("OO_LINE_NORM", b2, 81)
	case 0x60:
		return decodedStatic("PPK_CONN_OK")
	case 0x61:
		return decodedStatic("PPK_NO_CONN")
	case 0x63:
		return decodedStatic("PPK_BAD")
	case 0x64:
		return decodedStatic("ENABL_PPK_OK")
	case 0x65:
		return decodedStatic("DISABL_PPK_OK")
	case 0x68:
		return decodedStatic("NO_220")
	case 0x69:
		return decodedStatic("OK_220")
	case 0x6A:
		return decodedStatic("ACC_OK")
	case 0x6B:
		return decodedStatic("ACC_BAD")
	case 0x6C:
		return decodedStatic("DOOR_OP")
	case 0x6D:
		return decodedStatic("DOOR_CL")
	case 0x6E:
		return decodedStatic("SABOTAGE")
	case 0x6F:
		return decodedStatic("ENABLED_DISABLED_ERROR")
	case 0x70:
		return decodedWithOffset("LINE_KZ", b2, -0x0f)
	case 0x71:
		return decodedWithOffset("OO_LINE_KZ", b2, -0x0f)
	case 0x72:
		return decodedWithOffset("LINE_KZ", b2, 0x10+1)
	case 0x73:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x10+1)
	case 0x74:
		return decodedWithOffset("LINE_KZ", b2, 0x30+1)
	case 0x75:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x30+1)
	case 0x76:
		return decodedWithOffset("LINE_KZ", b2, 0x50+1)
	case 0x77:
		return decodedWithOffset("OO_LINE_KZ", b2, 0x50+1)
	case 0x78:
		return decodedWithOffset("LINE_BAD", b2, -0x0f)
	case 0x79:
		return decodedWithOffset("OO_LINE_BAD", b2, -0x0f)
	case 0x7A:
		return decodedWithOffset("LINE_BAD", b2, 0x10+1)
	case 0x7B:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x10+1)
	case 0x7C:
		return decodedWithOffset("LINE_BAD", b2, 0x30+1)
	case 0x7D:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x30+1)
	case 0x7E:
		return decodedWithOffset("LINE_BAD", b2, 0x50+1)
	case 0x7F:
		return decodedWithOffset("OO_LINE_BAD", b2, 0x50+1)
	case 0x90:
		return decodedWithSecondByte("HIGH_TEMP_DETECTED", b2)
	case 0x91:
		return decodedWithSecondByte("TEMP_IS_OK", b2)
	case 0x92:
		return decodedWithSecondByte("LOW_TEMP_DETECTED", b2)
	case 0x93:
		return decodedWithSecondByte("TEMP_IS_OK_AFTER_LOW", b2)
	case 0x94:
		return decodedWithSecondByte("VIBRATION_DETECTED", b2)
	case 0x95:
		return decodedWithSecondByte("ZONE_MALFUNCTION", b2)
	case 0x96:
		return decodedWithSecondByte("ZONE_OK", b2)
	case 0x97:
		return decodedWithSecondByte("BOLT_LOCK_UNLOCKED", b2)
	case 0x98:
		return decodedWithSecondByte("BOLT_LOCK_LOCKED", b2)
	case 0xA0:
		return decodedWithSecondByte("SMOKE", b2)
	case 0xA1:
		return decodedWithSecondByte("HEAT", b2)
	case 0xA2:
		return decodedWithSecondByte("WATER", b2)
	case 0xA3:
		return decodedWithSecondByte("CO_GAS", b2)
	case 0xA4:
		return decodedWithSecondByte("BRUTFORS_CANCELLED", b2)
	case 0xA5:
		return decodedWithSecondByte("JAMMING", b2)
	case 0xA6:
		return decodedWithSecondByte("SENSOR_NO_CONN", b2)
	case 0xA7:
		return decodedWithSecondByte("AKSEL", b2)
	case 0xA8:
		return decodedWithSecondByte("BTTR_FAIL", b2)
	case 0xA9:
		return decodedWithSecondByte("HRDW_FAIL", b2)
	case 0xAA:
		return decodedWithSecondByte("DUST", b2)
	case 0xAB:
		return decodedWithSecondByte("FIRE_ALARM_FINISH", b2)
	case 0xAC:
		return decodedWithSecondByte("TMP_OK", b2)
	case 0xAD:
		return decodedWithSecondByte("GAS_ALARM", b2)
	case 0xAE:
		return decodedWithSecondByte("GAS_ALARM_FINISH", b2)
	case 0xAF:
		return decodedWithSecondByte("WATER_LEAK_FINISH", b2)
	case 0xB0:
		return decodedWithSecondByte("CO_OKEY", b2)
	case 0xB1:
		return decodedWithSecondByte("NO_EXTERNAL_POWER", b2)
	case 0xB2:
		return decodedWithSecondByte("EXTERNAL_POWER_OK", b2)
	case 0xB3:
		return decodedWithSecondByte("AKSEL_BAD", b2)
	case 0xB4:
		return decodedWithSecondByte("AKSEL_OK", b2)
	case 0xB5:
		return decodedWithSecondByte("DEEP_RESTART", b2)
	case 0xB6:
		return decodedWithSecondByte("LOST_PHOTO_CONN", b2)
	case 0xB7:
		return decodedWithSecondByte("PHOTO_CONN_OK", b2)
	case 0xB8:
		return decodedWithSecondByte("GAS_CLEAN", b2)
	case 0xB9:
		return decodedWithSecondByte("SCENARIO_OFF", b2)
	case 0xBA:
		return decodedWithSecondByte("SCENARIO_ON", b2)
	case 0xBB:
		return decodedWithSecondByte("GROUP_NIGHT_OFF", b2)
	case 0xBC:
		return decodedWithSecondByte("GROUP_NIGHT_ON", b2)
	case 0xBD:
		return decodedWithSecondByte("SMOKE_CHAMBER_OK", b2)
	case 0xBE:
		return decodedWithSecondByte("FAULTY_DETECTOR", b2)
	case 0xBF:
		return decodedWithSecondByte("INTERFERENCE_DETECT_OK_NEW", b2)
	case 0xC0:
		return decodedWithSecondByte("TAMPER_ON_NEW", b2)
	case 0xC1:
		return decodedWithSecondByte("LID_NOTIFS_DISABLED", b2)
	case 0xC2:
		return decodedWithSecondByte("LID_NOTIFS_ON", b2)
	case 0xC3:
		return decodedWithSecondByte("KEYPAD_LOCKED", b2)
	case 0xC4:
		return decodedWithSecondByte("KEYPAD_UNLOCKED", b2)
	case 0xC5:
		return decodedWithSecondByte("ENCLOSURE_TAMPER_SWITCH_DISCONNECTED", b2)
	case 0xC6:
		return decodedWithSecondByte("ENCLOSURE_TAMPER_SWITCH_CONNECTED", b2)
	case 0xC7:
		return decodedWithSecondByte("BATTERY_DEVICE_OUT_OF_ORDER", b2)
	case 0xC8:
		return decodedWithSecondByte("BATTERY_DEVICE_IS_OK", b2)
	case 0xC9:
		return decodedWithSecondByte("POWER_UNIT_OUT_OF_ORDER", b2)
	case 0xCA:
		return decodedWithSecondByte("POWER_UNIT_OK", b2)
	case 0xCB:
		return decodedWithSecondByte("FIRE_ALARM_ACTIVATED_MANUAL", b2)
	case 0xCC:
		return decodedWithSecondByte("FIRE_ALARM_DEACTIVATED_MANUAL", b2)
	case 0xCD:
		return decodedWithSecondByte("AUTO_DEACTIVATION_ALARMS_OR_EXPIRATION", b2)
	case 0xCE:
		return decodedWithSecondByte("AUTO_DEACTIVATION_ALARMS_OR_EXPIRATION_RESTORED", b2)
	case 0xCF:
		return decodedWithSecondByte("DEVICE_NOT_CLOSED_DURING_ARMING_ATTEMPT", b2)
	case 0xD0:
		return decodedWithSecondByte("EMP_ON_TIME", b2)
	case 0xD1:
		return decodedWithSecondByte("PART_OFF", b2)
	case 0xD2:
		return decodedWithSecondByte("MASKING_LINE", b2)
	case 0xD3:
		return decodedWithSecondByte("SENSOR_CONN_OK", b2)
	case 0xD4:
		return decodedWithSecondByte("SENSOR_FOOD_OK", b2)
	case 0xD5:
		return decodedWithSecondByte("NORM_MASKING_LINE", b2)
	case 0xD6:
		return decodedWithSecondByte("MALFUNCTION_DETECTED_DEVICE_IN_ROOM_PULSE_EVENT_NEW", b2)
	case 0xD7:
		return decodedWithSecondByte("MALFUNCTION_RESOLVED_DEVICE_IN_ROOM_NEW", b2)
	case 0xD8:
		return decodedWithSecondByte("ACC_LACK_DEVICE", b2)
	case 0xD9:
		return decodedWithSecondByte("ACC_OK_DEVICE", b2)
	case 0xDA:
		return decodedWithSecondByte("MEDICAL_ALARM_NEW", b2)
	case 0xDB:
		return decodedWithSecondByte("MEDICAL_ALARM_FINISH_NEW", b2)
	case 0xDC:
		return decodedWithSecondByte("DEVICE_TEMPORARILY_DEACTIVATED_NEW", b2)
	case 0xDD:
		return decodedWithSecondByte("DEVICE_ACTIVE_AGAIN_NEW", b2)
	case 0xDE:
		return decodedWithSecondByte("NOT_RESPONDING_DEVICE_IN_ROOM_NEW", b2)
	case 0xDF:
		return decodedWithSecondByte("BRUTFORS_NEW", b2)
	case 0xE0:
		return decodedWithSecondByte("NORM_24", b2)
	case 0xE1:
		return decodedWithSecondByte("ALM_IO", b2)
	case 0xE2:
		return decodedWithSecondByte("NORM_IO", b2)
	case 0xE3:
		return decodedWithSecondByte("BAD_FIRE_PL", b2)
	case 0xE4:
		return decodedWithSecondByte("FIRE_PL_OK", b2)
	case 0xE5:
		return decodedWithSecondByte("GROUP_OFF_USER", b2)
	case 0xE6:
		return decodedWithSecondByte("GROUP_ON_USER", b2)
	case 0xE7:
		return decodedWithSecondByte("SECT_OFF", b2)
	case 0xE8:
		return decodedWithSecondByte("SECT_ON", b2)
	case 0xE9:
		return decodedWithSecondByte("OFF_WITH_PPL", b2)
	case 0xEA:
		return decodedWithSecondByte("ON_WITH_PPL", b2)
	case 0xEB:
		return decodedWithSecondByte("OFF_BFR_TIME", b2)
	case 0xEC:
		return decodedWithSecondByte("ON_BFR_TIME", b2)
	case 0xED:
		return decodedWithSecondByte("OFF_AFTR_TIME", b2)
	case 0xEE:
		return decodedWithSecondByte("ON_AFTR_TIME", b2)
	case 0xEF:
		return decodedWithSecondByte("EMP_OFF_TIME", b2)
	case 0xF0:
		return decodedWithSecondByte("STAYIN_HOME", b2)
	case 0xF1:
		return decodedWithSecondByte("OO_STAYIN_HOME", b2)
	case 0xF2:
		return decodedWithSecondByte("INGINEER_PL", b2)
	case 0xF3:
		return decodedWithSecondByte("ZONE_ALM", b2)
	case 0xF4:
		return decodedWithSecondByte("ALM_BTN_PRS", b2)
	case 0xF5:
		return decodedWithSecondByte("ALM_BTN_RLZ", b2)
	case 0xF6:
		return decodedWithSecondByte("ZONE_NORM", b2)
	case 0xF7:
		return decodedWithSecondByte("SENS_TAMP", b2)
	case 0xF8:
		return decodedWithSecondByte("SENS_TAMP_N", b2)
	case 0xF9:
		return decodedWithSecondByte("HUB_TAMP", b2)
	case 0xFA:
		return decodedWithSecondByte("HUB_TAMP_N", b2)
	case 0xFB:
		return decodedWithSecondByte("ALM_PERIM_ZONE", b2)
	case 0xFC:
		return decodedWithSecondByte("NORM_PERIM_ZONE", b2)
	case 0xFD:
		return decodedWithSecondByte("ALM_INNER_ZONE", b2)
	case 0xFE:
		return decodedWithSecondByte("NORM_INNER_ZONE", b2)
	case 0xFF:
		return decodedWithSecondByte("ALM_24_ZONE", b2)
	}

	return caslDecodedEventCode{}, false
}

func decodeCASLProtocolCode(code string, deviceType string) (caslDecodedEventCode, bool) {
	b1, b2, ok := parseCASLCodeBytes(code)
	if !ok {
		return caslDecodedEventCode{}, false
	}

	if decoded, ok := decodeCASLSystemCode(b1, b2); ok {
		return decoded, true
	}

	switch caslProtocolModelFromDeviceType(deviceType) {
	case caslProtocolModelSIA:
		return decodeCASLSIACode(b1, b2)
	case caslProtocolModelVBD4:
		return decodeCASLVBD4Code(b1, b2)
	case caslProtocolModelDozor:
		return decodeCASLDozorCode(b1, b2)
	case caslProtocolModelD128:
		return decodeCASLD128Code(b1, b2)
	default:
		return decodeCASLRcomSurgardCode(b1, b2)
	}
}

func classifyCASLEventTypeWithContext(code string, contactID string, sourceType string, details string) models.EventType {
	normalizedType := strings.TrimSpace(sourceType)
	if normalizedType != "" && !strings.EqualFold(normalizedType, "user_action") && !strings.EqualFold(normalizedType, "mob_user_action") {
		if mapped := mapCASLTapeEventType(normalizedType); mapped != models.EventFault || strings.EqualFold(normalizedType, "fault") {
			return mapped
		}
	}

	if byCode := classifyCASLEventType(code); byCode != models.EventFault {
		return byCode
	}

	if decoded, ok := decodeCASLProtocolCode(code, ""); ok {
		if byDecoded := classifyCASLEventType(decoded.MessageKey); byDecoded != models.EventFault {
			return byDecoded
		}
	}

	if byContact := classifyCASLEventType(contactID); byContact != models.EventFault {
		return byContact
	}

	if byDetails := classifyCASLEventType(details); byDetails != models.EventFault {
		return byDetails
	}

	return models.EventFault
}

func mapEventTypeToAlarmType(eventType models.EventType) (models.AlarmType, bool) {
	switch eventType {
	case models.EventFire:
		return models.AlarmFire, true
	case models.EventBurglary:
		return models.AlarmBurglary, true
	case models.EventPanic:
		return models.AlarmPanic, true
	case models.EventMedical:
		return models.AlarmMedical, true
	case models.EventGas:
		return models.AlarmGas, true
	case models.EventTamper:
		return models.AlarmTamper, true
	case models.EventPowerFail:
		return models.AlarmPowerFail, true
	case models.EventBatteryLow:
		return models.AlarmBatteryLow, true
	case models.EventOffline:
		return models.AlarmOffline, true
	case models.SystemEvent:
		return models.AlarmSystemEvent, true
	case models.EventFault:
		return models.AlarmFault, true
	case models.EventNotification:
		return models.AlarmNotification, true
	case models.EventOperator:
		return models.AlarmOperator, true
	case models.EventMobile:
		return models.AlarmMobile, true
	case models.EventEliminated:
		return models.AlarmEliminated, true
	case models.EventFireTrouble:
		return models.AlarmFireTrouble, true
	default:
		return "", false
	}
}

func mapCASLEventSC1(eventType models.EventType) int {
	switch eventType {
	case models.EventFire:
		return 1
	case models.EventBurglary:
		return 22
	case models.EventPanic:
		return 21
	case models.EventMedical:
		return 23
	case models.EventGas:
		return 24
	case models.EventTamper:
		return 25
	case models.EventRestore, models.EventPowerOK:
		return 5
	case models.EventArm:
		return 10
	case models.EventDisarm:
		return 14
	case models.EventPowerFail:
		return 26
	case models.EventBatteryLow:
		return 27
	case models.EventOnline:
		return 28
	case models.EventOffline:
		return 29
	case models.EventTest:
		return 16
	case models.SystemEvent:
		return 30
	case models.EventOperator:
		return 31
	case models.EventMobile:
		return 32
	case models.EventEliminated:
		return 34
	case models.EventFireTrouble:
		return 35
	default:
		return 2
	}
}
