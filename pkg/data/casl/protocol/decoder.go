package protocol

type DecodedEvent struct {
	MessageKey string
	Number     int
	HasNumber  bool
}

type Decoder interface {
	Decode(b1, b2 byte) (DecodedEvent, bool)
}

type BaseDecoder struct{}

func (d *BaseDecoder) Decode(b1, b2 byte) (DecodedEvent, bool) {
	switch b1 {
	case 0x00:
		switch b2 {
		case 0xB3:
			return DecodedEvent{MessageKey: "BAN_TIME"}, true
		case 0xBD:
			return DecodedEvent{MessageKey: "REQUIRED_GROUP_ON"}, true
		case 0x60:
			return DecodedEvent{MessageKey: "PPK_CONN_OK"}, true
		case 0x66:
			return DecodedEvent{MessageKey: "SUSPICIOUS_ACTIVITY"}, true
		case 0x67:
			return DecodedEvent{MessageKey: "SABOTAGE"}, true
		}
	case 0x01:
		switch b2 {
		case 0x61:
			return DecodedEvent{MessageKey: "OO_NO_POLL"}, true
		case 0x62:
			return DecodedEvent{MessageKey: "OO_NO_PING"}, true
		}
	}
	return DecodedEvent{}, false
}

type RcomDecoder struct {
	BaseDecoder
}

func (d *RcomDecoder) Decode(b1, b2 byte) (DecodedEvent, bool) {
	if res, ok := d.BaseDecoder.Decode(b1, b2); ok {
		return res, ok
	}

	if b1 == 0x3B {
		switch b2 {
		case 0x00: return DecodedEvent{MessageKey: "REP_FIRMW_4L"}, true
		case 0x01: return DecodedEvent{MessageKey: "END_FIRMW_4L"}, true
		case 0x02: return DecodedEvent{MessageKey: "REQ_REP_FIRMW_4L"}, true
		case 0x03: return DecodedEvent{MessageKey: "REC_CONFIG_4L"}, true
		case 0x04: return DecodedEvent{MessageKey: "END_CONFIG_4L"}, true
		case 0x05: return DecodedEvent{MessageKey: "PPK_SIM_4L"}, true
		case 0x06: return DecodedEvent{MessageKey: "PPK_IMEIL_4L"}, true
		case 0x07: return DecodedEvent{MessageKey: "PPK_COORD_4L"}, true
		case 0x08: return DecodedEvent{MessageKey: "PPK_CSQ_4L"}, true
		case 0x09: return DecodedEvent{MessageKey: "CONTROL_4L"}, true
		}
	}

	if b1 == 0x08 {
		return DecodedEvent{MessageKey: "ID_HOZ", Number: int(b2) - 0x0f, HasNumber: true}, true
	}

	switch b1 {
	case 0x00:
		switch b2 {
		case 0x02: return DecodedEvent{MessageKey: "CANNOT_AUTO_ARM"}, true
		case 0x03: return DecodedEvent{MessageKey: "DEVICE_TEMPORARILY_DEACTIVATED"}, true
		case 0x04: return DecodedEvent{MessageKey: "DEVICE_ACTIVE_AGAIN"}, true
		case 0x05: return DecodedEvent{MessageKey: "TAMPER_ON"}, true
		case 0x57: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_ETHERNET_LOST"}, true
		case 0x58: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_ETHERNET_RESTORED"}, true
		case 0x61: return DecodedEvent{MessageKey: "PPK_NO_CONN"}, true
		case 0x63: return DecodedEvent{MessageKey: "PPK_BAD"}, true
		case 0x64: return DecodedEvent{MessageKey: "ENABLED"}, true
		case 0x65: return DecodedEvent{MessageKey: "DISABLED"}, true
		case 0x68: return DecodedEvent{MessageKey: "NO_220"}, true
		case 0x69: return DecodedEvent{MessageKey: "OK_220"}, true
		case 0x6A: return DecodedEvent{MessageKey: "ACC_OK"}, true
		case 0x6B: return DecodedEvent{MessageKey: "ACC_BAD"}, true
		case 0x6C: return DecodedEvent{MessageKey: "DOOR_OP"}, true
		case 0x6D: return DecodedEvent{MessageKey: "DOOR_CL"}, true
		case 0x6E: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_CELLULAR_LOST"}, true
		case 0x6F: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_CELLULAR_RESTORED"}, true
		case 0x70: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_WI_FI_LOST"}, true
		case 0x71: return DecodedEvent{MessageKey: "SERVER_CONNECTION_VIA_WI_FI_RESTORED"}, true
		case 0x79: return DecodedEvent{MessageKey: "RING_DISCONNECTED"}, true
		case 0x80: return DecodedEvent{MessageKey: "RING_CONNECTED"}, true
		case 0xB9: return DecodedEvent{MessageKey: "FULL_REBOOT"}, true
		}
	case 0x01:
		switch b2 {
		case 0x63: return DecodedEvent{MessageKey: "CHANGE_IP_OK"}, true
		case 0x64: return DecodedEvent{MessageKey: "CHANGE_IP_FAIL"}, true
		case 0x68: return DecodedEvent{MessageKey: "OO_NO_220"}, true
		case 0x69: return DecodedEvent{MessageKey: "OO_OK_220"}, true
		case 0x6A: return DecodedEvent{MessageKey: "OO_ACC_OK"}, true
		case 0x6B: return DecodedEvent{MessageKey: "OO_ACC_BAD"}, true
		case 0x6C: return DecodedEvent{MessageKey: "OO_DOOR_OP"}, true
		case 0x6D: return DecodedEvent{MessageKey: "OO_DOOR_CL"}, true
		}
	case 0x02: return DecodedEvent{MessageKey: "WL_ACC_OK", Number: int(b2) + 1, HasNumber: true}, true
	case 0x03: return DecodedEvent{MessageKey: "WL_ACC_BAD", Number: int(b2) + 1, HasNumber: true}, true
	case 0x04: return DecodedEvent{MessageKey: "WL_DOOR_CL", Number: int(b2) + 1, HasNumber: true}, true
	case 0x05: return DecodedEvent{MessageKey: "WL_DOOR_OP", Number: int(b2) + 1, HasNumber: true}, true
	case 0x06: return DecodedEvent{MessageKey: "WL_TROUBLE", Number: int(b2) + 1, HasNumber: true}, true
	case 0x07: return DecodedEvent{MessageKey: "WL_NORM", Number: int(b2) + 1, HasNumber: true}, true
	case 0x09: return DecodedEvent{MessageKey: "PRIMUS", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x0A: return DecodedEvent{MessageKey: "ID_HOZ", Number: int(b2) + 0x10 + 1, HasNumber: true}, true
	case 0x0B: return DecodedEvent{MessageKey: "PRIMUS", Number: int(b2) + 0x10 + 1, HasNumber: true}, true
	case 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3A, 0x3B:
		msgKey := ""
		switch b1 {
		case 0x30: msgKey = "AD_DOOR_OP"
		case 0x31: msgKey = "OO_AD_DOOR_OP"
		case 0x32: msgKey = "AD_DOOR_CL"
		case 0x33: msgKey = "OO_AD_DOOR_CL"
		case 0x34: msgKey = "AD_NO_CONN"
		case 0x35: msgKey = "OO_AD_NO_CONN"
		case 0x36: msgKey = "AD_CONN_OK"
		case 0x37: msgKey = "OO_AD_CONN_OK"
		case 0x38: msgKey = "AD_BAD_FOOD"
		case 0x39: msgKey = "OO_ALM_AD_POWER"
		case 0x3A: msgKey = "AD_FOOD_OK"
		case 0x3B: msgKey = "OO_AD_POWER_OK"
		}
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x3E: return DecodedEvent{MessageKey: "PPK_FW_VERSION", Number: int(b2), HasNumber: true}, true
	case 0x3F:
		switch b2 {
		case 0x09, 0x8F: return DecodedEvent{MessageKey: "COERCION"}, true
		case 0x10, 0x90: return DecodedEvent{MessageKey: "RESTART"}, true
		case 0x11, 0x91: return DecodedEvent{MessageKey: "CHECK_CONN"}, true
		case 0x12, 0x92: return DecodedEvent{MessageKey: "DECONCERV"}, true
		case 0x13, 0x93: return DecodedEvent{MessageKey: "CONCERV"}, true
		case 0x14, 0x94: return DecodedEvent{MessageKey: "EDIT_CONF"}, true
		case 0x15, 0x95: return DecodedEvent{MessageKey: "ENABLED"}, true
		case 0x16, 0x96: return DecodedEvent{MessageKey: "DISABLED"}, true
		}
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		msgKey := "GROUP_ON"
		if b1%2 != 0 { msgKey = "OO_GROUP_ON" }
		offset := -0x0f
		if b1 >= 0x42 && b1 <= 0x43 { offset = 17 }
		if b1 >= 0x44 && b1 <= 0x45 { offset = 49 }
		if b1 >= 0x46 && b1 <= 0x47 { offset = 81 }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) + offset, HasNumber: true}, true
	case 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F:
		msgKey := "GROUP_OFF"
		if b1%2 != 0 { msgKey = "OO_GROUP_OFF" }
		offset := -0x0f
		if b1 >= 0x4A && b1 <= 0x4B { offset = 17 }
		if b1 >= 0x4C && b1 <= 0x4D { offset = 49 }
		if b1 >= 0x4E && b1 <= 0x4F { offset = 81 }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) + offset, HasNumber: true}, true
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		msgKey := "LINE_BRK"
		if b1%2 != 0 { msgKey = "OO_LINE_BRK" }
		offset := -0x0f
		if b1 >= 0x52 && b1 <= 0x53 { offset = 17 }
		if b1 >= 0x54 && b1 <= 0x55 { offset = 49 }
		if b1 >= 0x56 && b1 <= 0x57 { offset = 81 }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) + offset, HasNumber: true}, true
	case 0x58, 0x59, 0x5A, 0x5B, 0x5C, 0x5D, 0x5E, 0x5F:
		msgKey := "LINE_NORM"
		if b1%2 != 0 { msgKey = "OO_LINE_NORM" }
		offset := -0x0f
		if b1 >= 0x5A && b1 <= 0x5B { offset = 17 }
		if b1 >= 0x5C && b1 <= 0x5D { offset = 49 }
		if b1 >= 0x5E && b1 <= 0x5F { offset = 81 }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) + offset, HasNumber: true}, true
	case 0x70, 0x71, 0x72, 0x73, 0x74, 0x75, 0x76, 0x77:
		msgKey := "LINE_KZ"
		if b1%2 != 0 { msgKey = "OO_LINE_KZ" }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x78, 0x79, 0x7A, 0x7B, 0x7C, 0x7D, 0x7E, 0x7F:
		msgKey := "LINE_BAD"
		if b1%2 != 0 { msgKey = "OO_LINE_BAD" }
		return DecodedEvent{MessageKey: msgKey, Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x90: return DecodedEvent{MessageKey: "HIGH_TEMP_DETECTED", Number: int(b2), HasNumber: true}, true
	case 0x91: return DecodedEvent{MessageKey: "TEMP_IS_OK", Number: int(b2), HasNumber: true}, true
	case 0x94: return DecodedEvent{MessageKey: "VIBRATION_DETECTED", Number: int(b2), HasNumber: true}, true
	case 0xA0: return DecodedEvent{MessageKey: "SMOKE", Number: int(b2), HasNumber: true}, true
	case 0xA1: return DecodedEvent{MessageKey: "HEAT", Number: int(b2), HasNumber: true}, true
	case 0xA2: return DecodedEvent{MessageKey: "WATER", Number: int(b2), HasNumber: true}, true
	case 0xA3: return DecodedEvent{MessageKey: "CO_GAS", Number: int(b2), HasNumber: true}, true
	case 0xA5: return DecodedEvent{MessageKey: "JAMMING", Number: int(b2), HasNumber: true}, true
	case 0xA6: return DecodedEvent{MessageKey: "SENSOR_NO_CONN", Number: int(b2), HasNumber: true}, true
	case 0xA8: return DecodedEvent{MessageKey: "BTTR_FAIL", Number: int(b2), HasNumber: true}, true
	case 0xA9: return DecodedEvent{MessageKey: "HRDW_FAIL", Number: int(b2), HasNumber: true}, true
	case 0xE5: return DecodedEvent{MessageKey: "GROUP_OFF_USER", Number: int(b2), HasNumber: true}, true
	case 0xE6: return DecodedEvent{MessageKey: "GROUP_ON_USER", Number: int(b2), HasNumber: true}, true
	case 0xEF: return DecodedEvent{MessageKey: "EMP_OFF_TIME", Number: int(b2), HasNumber: true}, true
	case 0xF0: return DecodedEvent{MessageKey: "STAYIN_HOME", Number: int(b2), HasNumber: true}, true
	case 0xF3: return DecodedEvent{MessageKey: "ZONE_ALM", Number: int(b2), HasNumber: true}, true
	case 0xF4: return DecodedEvent{MessageKey: "ALM_BTN_PRS", Number: int(b2), HasNumber: true}, true
	case 0xF6: return DecodedEvent{MessageKey: "ZONE_NORM", Number: int(b2), HasNumber: true}, true
	case 0xF7: return DecodedEvent{MessageKey: "SENS_TAMP", Number: int(b2), HasNumber: true}, true
	case 0xF9: return DecodedEvent{MessageKey: "HUB_TAMP", Number: int(b2), HasNumber: true}, true
	case 0xFB: return DecodedEvent{MessageKey: "ALM_PERIM_ZONE", Number: int(b2), HasNumber: true}, true
	case 0xFC: return DecodedEvent{MessageKey: "NORM_PERIM_ZONE", Number: int(b2), HasNumber: true}, true
	case 0xFD: return DecodedEvent{MessageKey: "ALM_INNER_ZONE", Number: int(b2), HasNumber: true}, true
	case 0xFE: return DecodedEvent{MessageKey: "NORM_INNER_ZONE", Number: int(b2), HasNumber: true}, true
	case 0xFF: return DecodedEvent{MessageKey: "ALM_24_ZONE", Number: int(b2), HasNumber: true}, true
	}

	return DecodedEvent{}, false
}

func GetDecoder(deviceType string) Decoder {
	// Strategy selection can be expanded for SIA, VBD4, etc.
	return &RcomDecoder{}
}
