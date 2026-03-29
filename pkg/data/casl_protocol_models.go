package data

// decodeCASLSIACode декодує бінарні коди подій для приладів протоколу SIA
// (TYPE_DEVICE_Ajax_SIA, TYPE_DEVICE_Bron_SIA).
// Портовано з TypeScript siaTranslator + siaDictionary.
func decodeCASLSIACode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	switch b1 {
	case 0x00:
		switch b2 {
		case 0x01:
			return decodedStatic("FORCED_AUTO_CLOSING")
		case 0x02:
			return decodedStatic("ALARM_CANCEL")
		case 0x03:
			return decodedStatic("CLOSING_EXTEND")
		case 0x0B:
			return decodedStatic("RECENT_CLOSING")
		case 0x0C:
			return decodedStatic("ACCESS_CLOSED")
		case 0x0D:
			return decodedStatic("ACCESS_DENIED")
		case 0x0E:
			return decodedStatic("DOOR_FORCED")
		case 0x0F:
			return decodedStatic("ACCESS_GRANTED")
		case 0x10:
			return decodedStatic("ACCESS_LOCKOUT")
		case 0x11:
			return decodedStatic("ACCESS_OPEN")
		case 0x12:
			return decodedStatic("DOOR_RESTORAL")
		case 0x13:
			return decodedStatic("DOOR_STATION")
		case 0x14:
			return decodedStatic("ACCESS_TROUBLE")
		case 0x15:
			return decodedStatic("DEALER_ID")
		case 0x16:
			return decodedStatic("EXIT_ERROR")
		case 0x17:
			return decodedStatic("EXPANSION_REST")
		case 0x18:
			return decodedStatic("EXPANSION_TROUBLE")
		case 0x19:
			return decodedStatic("DATE_CHANGED")
		case 0x1A:
			return decodedStatic("HOLIDAY_CHANGED")
		case 0x1B:
			return decodedStatic("LOG_HOLD")
		case 0x1C:
			return decodedStatic("LOG_OVERFLOW")
		case 0x1D:
			return decodedStatic("SCHEDULE_CHANGE")
		case 0x1E:
			return decodedStatic("TIME_CHANGED")
		case 0x1F:
			return decodedStatic("USER_CODE_CHANGE")
		case 0x20:
			return decodedStatic("USER_CODE_DELETE")
		case 0x21:
			return decodedStatic("LOCAL_PROGRAM")
		case 0x22:
			return decodedStatic("LOCAL_PROGRAM_DENIED")
		case 0x23:
			return decodedStatic("LISTEN_ENDED")
		case 0x24:
			return decodedStatic("LISTEN_BEGIN")
		case 0x25:
			return decodedStatic("PHONE_LINE_REST")
		case 0x26:
			return decodedStatic("LOCAL_PROGRAM")
		case 0x27:
			return decodedStatic("PHONE_LINE_TROUBLE")
		case 0x28:
			return decodedStatic("LOCAL_PROG_FAIL")
		case 0x29:
			return decodedStatic("LOCAL_PROG_ENDED")
		case 0x2A:
			return decodedStatic("NO_ACTIVITY")
		case 0x2B:
			return decodedStatic("CANCEL_REPORT")
		case 0x2E:
			return decodedStatic("OPEN_REPORT")
		case 0x30:
			return decodedStatic("LATE_CLOSE")
		case 0x31:
			return decodedStatic("REMOTE_FAILED")
		case 0x32:
			return decodedStatic("RELAY_CLOSE")
		case 0x33:
			return decodedStatic("REMOTE_PROG_DENIED")
		case 0x34:
			return decodedStatic("REMOTE_RESET")
		case 0x35:
			return decodedStatic("RELAY_OPEN")
		case 0x36:
			return decodedStatic("AUTO_TEST")
		case 0x37:
			return decodedStatic("POWER_UP")
		case 0x38:
			return decodedStatic("DATA_LOST")
		case 0x39:
			return decodedStatic("REMOTE_PROG_FAIL")
		case 0x3A:
			return decodedStatic("MANUAL_TEST")
		case 0x3B:
			return decodedStatic("TAMPER_OFF")
		case 0x3C:
			return decodedStatic("TEST_END")
		case 0x3D:
			return decodedStatic("TEST_START")
		case 0x3E:
			return decodedStatic("UNDEFINED")
		case 0x3F:
			return decodedStatic("PRIN_PAPER_IN")
		case 0x40:
			return decodedStatic("PRIN_PAPER_OUT")
		case 0x41:
			return decodedStatic("PRIN_RESTORE")
		case 0x42:
			return decodedStatic("PRIN_TROUBLE")
		case 0x43:
			return decodedStatic("PRIN_TEST")
		case 0x44:
			return decodedStatic("PRIN_ONLINE")
		case 0x45:
			return decodedStatic("PRIN_OFFLINE")
		case 0x46:
			return decodedStatic("EXTRA_POINT")
		case 0x47:
			return decodedStatic("EXTRA_RF_POINT")
		case 0x48:
			return decodedStatic("SYS_BATTERY_CONN_MISS")
		case 0x49:
			return decodedStatic("BUSY_SECONDS")
		case 0x4A:
			return decodedStatic("RX_LINE_CARD_TROUBLE")
		case 0x4B:
			return decodedStatic("RX_LINE_CARD_RESTORAL")
		case 0x4C:
			return decodedStatic("PARAMETER_FAIL")
		case 0x4D:
			return decodedStatic("SYS_BATTERY_MISS")
		case 0x4E:
			return decodedStatic("INVALID_REPORT")
		case 0x4F:
			return decodedStatic("UNKNOWN_MESSAGE")
		case 0x50:
			return decodedStatic("POWER_SUPPLY_TROUBLE")
		case 0x51:
			return decodedStatic("POWER_SUPPLY_RESTORED")
		case 0x52:
			return decodedStatic("WATCHDOG_RESET")
		case 0x53:
			return decodedStatic("SERVICE_REQUIRED")
		case 0x54:
			return decodedStatic("STATUS_REPORT")
		case 0x55:
			return decodedStatic("SERVICE_COMPLETED")
		case 0x61:
			return decodedStatic("PPK_NO_CONN")
		}
	case 0x02:
		return decodedWithSecondByte("AC_TROUBLE", b2)
	case 0x03:
		return decodedWithSecondByte("BURGLARY_ALARM", b2)
	case 0x04:
		return decodedWithSecondByte("ALARM_ELIMINATED", b2)
	case 0x05:
		return decodedWithSecondByte("MALFUN_ELIMINATED", b2)
	case 0x06:
		return decodedWithSecondByte("BURGLARY_RESTORAL", b2)
	case 0x07:
		return decodedWithSecondByte("BURGLARY_SUPERVISORY", b2)
	case 0x08:
		return decodedWithSecondByte("BURGLARY_TROUBLE", b2)
	case 0x09:
		return decodedWithSecondByte("BURGLARY_VERIFIED", b2)
	case 0x0A:
		return decodedWithSecondByte("BURGLARY_TEST", b2)
	case 0x0B:
		return decodedWithSecondByte("AUTO_CLOSING", b2)
	case 0x0C:
		return decodedWithSecondByte("AUTO_CLOSING_GROUP", b2)
	case 0x0D:
		return decodedWithSecondByte("AUTO_CLOSING_ERROR", b2)
	case 0x0E:
		return decodedWithSecondByte("AUTO_CLOSING_GROUP_ERROR", b2)
	case 0x0F:
		return decodedWithSecondByte("CLOSING_SWITCH", b2)
	case 0x10:
		return decodedWithSecondByte("LATE_OPEN", b2)
	case 0x11:
		return decodedWithSecondByte("FORCE_ARMED", b2)
	case 0x12:
		return decodedWithSecondByte("POINT_CLOSING", b2)
	case 0x13:
		return decodedWithSecondByte("EXIT_ALARM", b2)
	case 0x14:
		return decodedWithSecondByte("FIRE_ALARM", b2)
	case 0x15:
		return decodedWithSecondByte("FIRE_BYPASS", b2)
	case 0x16:
		return decodedWithSecondByte("FIRE_ALARM_RESTORE", b2)
	case 0x17:
		return decodedWithSecondByte("FIRE_TEST_BEGIN", b2)
	case 0x18:
		return decodedWithSecondByte("FIRE_TROUBLE_RESTORE", b2)
	case 0x19:
		return decodedWithSecondByte("FIRE_TEST_END", b2)
	case 0x1A:
		return decodedWithSecondByte("FIRE_RESTORAL", b2)
	case 0x1B:
		return decodedWithSecondByte("FIRE_SUPERVISORY", b2)
	case 0x1C:
		return decodedWithSecondByte("FIRE_TROUBLE", b2)
	case 0x1D:
		return decodedWithSecondByte("FIRE_UNBYPASS", b2)
	case 0x1E:
		return decodedWithSecondByte("FIRE_TEST", b2)
	case 0x1F:
		return decodedWithSecondByte("MISSING_FIRE_TROUBLE", b2)
	case 0x20:
		return decodedWithSecondByte("GAS_ALARM", b2)
	case 0x21:
		return decodedWithSecondByte("GAS_BYPASS", b2)
	case 0x22:
		return decodedWithSecondByte("GAS_A_RESTORE", b2)
	case 0x23:
		return decodedWithSecondByte("GAS_T_RESTORE", b2)
	case 0x24:
		return decodedWithSecondByte("GAS_RESTORAL", b2)
	case 0x25:
		return decodedWithSecondByte("GAS_SUPERVISORY", b2)
	case 0x26:
		return decodedWithSecondByte("GAS_TROUBLE", b2)
	case 0x27:
		return decodedWithSecondByte("GAS_UNBYPASS", b2)
	case 0x28:
		return decodedWithSecondByte("GAS_TEST_GAS", b2)
	case 0x29:
		return decodedWithSecondByte("HOLD_ALARM", b2)
	case 0x2A:
		return decodedWithSecondByte("HOLD_BYPASS", b2)
	case 0x2B:
		return decodedWithSecondByte("HOLD_A_REST", b2)
	case 0x2C:
		return decodedWithSecondByte("HOLD_T_REST", b2)
	case 0x2D:
		return decodedWithSecondByte("HOLD_RESTORAL", b2)
	case 0x2E:
		return decodedWithSecondByte("HOLD_SUPERVISION", b2)
	case 0x2F:
		return decodedWithSecondByte("HOLD_TROUBLE", b2)
	case 0x31:
		return decodedWithSecondByte("USER_CODE_TAMPER", b2)
	case 0x32:
		return decodedWithSecondByte("USER_CODE_CANCELED", b2)
	case 0x33:
		return decodedWithSecondByte("SCHEDULE_EXECUTE", b2)
	case 0x34:
		return decodedWithSecondByte("HEAT_ALARM", b2)
	case 0x35:
		return decodedWithSecondByte("HEAT_BYPASS", b2)
	case 0x36:
		return decodedWithSecondByte("HEAT_ALARM_RESTORE", b2)
	case 0x37:
		return decodedWithSecondByte("HEAT_TROUBLE_RESTORE", b2)
	case 0x38:
		return decodedWithSecondByte("HEAT_RESTORAL", b2)
	case 0x39:
		return decodedWithSecondByte("HEAT_SUPERVISORY", b2)
	case 0x3A:
		return decodedWithSecondByte("HEAT_TROUBLE", b2)
	case 0x3B:
		return decodedWithSecondByte("HEAT_UNBYPASS", b2)
	case 0x3C:
		return decodedWithSecondByte("MED_INPULSE_ON", b2)
	case 0x3D:
		return decodedWithSecondByte("MED_BYPASS", b2)
	case 0x3E:
		return decodedWithSecondByte("MED_ALARM_RESTORE", b2)
	case 0x3F:
		return decodedWithSecondByte("MED_T_REST", b2)
	case 0x40:
		return decodedWithSecondByte("MED_RESTORAL", b2)
	case 0x41:
		return decodedWithSecondByte("MED_SUPERVISION", b2)
	case 0x42:
		return decodedWithSecondByte("MED_TROUBLE", b2)
	case 0x43:
		return decodedWithSecondByte("MED_UNBYPASS", b2)
	case 0x44:
		return decodedWithSecondByte("FORCED_PERIMETER", b2)
	case 0x45:
		return decodedWithSecondByte("PERIMETR_ARM_AUTO", b2)
	case 0x46:
		return decodedWithSecondByte("PERIMETR_DISARM_AREA", b2)
	case 0x47:
		return decodedWithSecondByte("ERROR_PERIMETR", b2)
	case 0x48:
		return decodedWithSecondByte("FORCED_PERIMETER_ARM", b2)
	case 0x49:
		return decodedWithSecondByte("PERIMETR_ARM", b2)
	case 0x4A:
		return decodedWithSecondByte("PERIMETR_DISARM_AUTO", b2)
	case 0x4B:
		return decodedWithSecondByte("PERIMETR_DISARM", b2)
	case 0x4C:
		return decodedWithSecondByte("AUTO_OPEN", b2)
	case 0x4D:
		return decodedWithSecondByte("AUTO_OPEN_GROUP", b2)
	case 0x4E:
		return decodedWithSecondByte("OPEN_AREA", b2)
	case 0x4F:
		return decodedWithSecondByte("FAIL_OPEN", b2)
	case 0x50:
		return decodedWithSecondByte("OPEN_KEYSWITCH", b2)
	case 0x51:
		return decodedWithSecondByte("POINT_OPEN", b2)
	case 0x52:
		return decodedWithSecondByte("PANIC_ALARM", b2)
	case 0x53:
		return decodedWithSecondByte("PANIC_BYPASS", b2)
	case 0x54:
		return decodedWithSecondByte("PHOTO_FAIL", b2)
	case 0x55:
		return decodedWithSecondByte("PANIC_ALARM_RESTORE", b2)
	case 0x56:
		return decodedWithSecondByte("PANIC_TROUBLE_RESTORE", b2)
	case 0x57:
		return decodedWithSecondByte("PHOTO_RESTORE", b2)
	case 0x58:
		return decodedWithSecondByte("PANIC_RESTORAL", b2)
	case 0x59:
		return decodedWithSecondByte("PANIC_SUPERVISORY", b2)
	case 0x5A:
		return decodedWithSecondByte("PANIC_TROUBLE", b2)
	case 0x5B:
		return decodedWithSecondByte("PANIC_UNBYPASS", b2)
	case 0x5C:
		return decodedWithSecondByte("EMERGENCY_ALARM", b2)
	case 0x5D:
		return decodedWithSecondByte("DEVICE_OFF", b2)
	case 0x5E:
		return decodedWithSecondByte("EMERGENCY_A_REST", b2)
	case 0x5F:
		return decodedWithSecondByte("EMERGENCY_T_REST", b2)
	case 0x60:
		return decodedWithSecondByte("EMERGENCY_RESTORAL", b2)
	case 0x61:
		return decodedWithSecondByte("EMERGENCY_SUPERVISION", b2)
	case 0x62:
		return decodedWithSecondByte("EMERGENCY_TROUBLE", b2)
	case 0x63:
		return decodedWithSecondByte("DEVICE_ON", b2)
	case 0x64:
		return decodedWithSecondByte("REMOTE_PROG_ON", b2)
	case 0x65:
		return decodedWithSecondByte("REMOTE_PROG_SUCCESS", b2)
	case 0x66:
		return decodedWithSecondByte("SPRIN_ALARM", b2)
	case 0x67:
		return decodedWithSecondByte("SPRIN_BYPASS", b2)
	case 0x68:
		return decodedWithSecondByte("SENSOR_MOVED_RESTORE", b2)
	case 0x69:
		return decodedWithSecondByte("SPRIN_ALARM_RES", b2)
	case 0x6A:
		return decodedWithSecondByte("SPRIN_TROUBLE_RES", b2)
	case 0x6B:
		return decodedWithSecondByte("SENSOR_MOVED", b2)
	case 0x6C:
		return decodedWithSecondByte("SPRIN_RESTORE", b2)
	case 0x6D:
		return decodedWithSecondByte("SPRIN_SUPERVISION", b2)
	case 0x6E:
		return decodedWithSecondByte("SPRIN_TROUBLE", b2)
	case 0x6F:
		return decodedWithSecondByte("SPRIN_UNBYPASS", b2)
	case 0x70:
		return decodedWithSecondByte("TAMPER_ALARM", b2)
	case 0x71:
		return decodedWithSecondByte("TAMPER_RESTORAL", b2)
	case 0x72:
		return decodedWithSecondByte("TEST_REPORT", b2)
	case 0x73:
		return decodedWithSecondByte("UNTYPED_ZONE_ALARM", b2)
	case 0x74:
		return decodedWithSecondByte("UNTYPED_ZONE_BYPASS", b2)
	case 0x75:
		return decodedWithSecondByte("UNTYPED_ALARM_RESTORAL", b2)
	case 0x76:
		return decodedWithSecondByte("UNTYPED_TROUBLE_RESTORAL", b2)
	case 0x77:
		return decodedWithSecondByte("UNTYPED_ZONE_RESTORAL", b2)
	case 0x78:
		return decodedWithSecondByte("UNTYPED_ZONE_SUP", b2)
	case 0x79:
		return decodedWithSecondByte("UNTYPED_ZONE_TROUBLE", b2)
	case 0x7A:
		return decodedWithSecondByte("UNTYPED_ZONE_UNBYPASS", b2)
	case 0x7B:
		return decodedWithSecondByte("UNTYPED_MISS_TROUBLE", b2)
	case 0x7C:
		return decodedWithSecondByte("UNTYPED_MISS_ALARM", b2)
	case 0x7D:
		return decodedWithSecondByte("WATER_ALARM", b2)
	case 0x7E:
		return decodedWithSecondByte("WATER_BYPASS", b2)
	case 0x7F:
		return decodedWithSecondByte("WATER_ALARM_RES", b2)
	case 0x80:
		return decodedWithSecondByte("WATER_TROUBLE_RES", b2)
	case 0x81:
		return decodedWithSecondByte("WATER_RES", b2)
	case 0x82:
		return decodedWithSecondByte("WATER_SUPERVISORY", b2)
	case 0x83:
		return decodedWithSecondByte("WATER_TROUBLE", b2)
	case 0x84:
		return decodedWithSecondByte("WATER_UNBYPASS", b2)
	case 0x85:
		return decodedWithSecondByte("CONNECTION_RESTORED", b2)
	case 0x86:
		return decodedWithSecondByte("RF_RESTORAL", b2)
	case 0x87:
		return decodedWithSecondByte("FACTORY_RESET", b2)
	case 0x88:
		return decodedWithSecondByte("CONNECTION_LOST", b2)
	case 0x89:
		return decodedWithSecondByte("RF_INTERFERENCE", b2)
	case 0x8A:
		return decodedWithSecondByte("TX_BATTERY_RESTORAL", b2)
	case 0x8B:
		return decodedWithSecondByte("TX_BATTERY_TROUBLE", b2)
	case 0x8C:
		return decodedWithSecondByte("FORCED_POINT", b2)
	case 0x8D:
		return decodedWithSecondByte("COMMUNICATION_FAIL", b2)
	case 0x8E:
		return decodedWithSecondByte("PARAMETER_CHANGED", b2)
	case 0x8F:
		return decodedWithSecondByte("COMMUNICATION_RESTORAL", b2)
	case 0x90:
		return decodedWithSecondByte("SYS_BATTERY_RESTORAL", b2)
	case 0x91:
		return decodedWithSecondByte("COMMUNICATION_TROUBLE", b2)
	case 0x92:
		return decodedWithSecondByte("SYS_BATTERY_TROUBLE", b2)
	case 0x93:
		return decodedWithSecondByte("FREEZE_ALARM", b2)
	case 0x94:
		return decodedWithSecondByte("FREEZE_BYPASS", b2)
	case 0x95:
		return decodedWithSecondByte("FREEZE_A_RESTORE", b2)
	case 0x96:
		return decodedWithSecondByte("FREEZE_T_RESTORE", b2)
	case 0x97:
		return decodedWithSecondByte("FREEZE_RESTORAL", b2)
	case 0x98:
		return decodedWithSecondByte("FREEZE_SUPERVISORY", b2)
	case 0x99:
		return decodedWithSecondByte("FREEZE_TROUBLE", b2)
	case 0x9A:
		return decodedWithSecondByte("FREEZE_UNBYPASS", b2)
	case 0x9B:
		return decodedWithSecondByte("HUB_ON", b2)
	case 0x9C:
		return decodedWithSecondByte("HUB_OFF", b2)
	case 0x9D:
		return decodedStatic("UNKNOWN")
	case 0x9E:
		return decodedWithSecondByte("AC_RESTORAL", b2)
	case 0x9F:
		return decodedWithSecondByte("ID_HOZ", b2)
	case 0xA0:
		return decodedWithSecondByte("FORCED_CLOSING", b2)
	case 0xA1:
		return decodedWithSecondByte("CLOSE_AREA", b2)
	case 0xA2:
		return decodedWithSecondByte("FAIL_CLOSE_WINDOW", b2)
	case 0xA3:
		return decodedWithSecondByte("LATE_CLOSE_WINDOW", b2)
	case 0xA4:
		return decodedWithSecondByte("EARLY_CLOSE", b2)
	case 0xA5:
		return decodedWithSecondByte("CLOSING_REPORT", b2)
	case 0xA6:
		return decodedWithSecondByte("AUTOMATIC_CLOSING", b2)
	case 0xA7:
		return decodedWithSecondByte("LATE_OPEN", b2)
	case 0xA8:
		return decodedWithSecondByte("EARLY_OPEN", b2)
	case 0xA9:
		return decodedWithSecondByte("HOLD_UNBYPASS", b2)
	case 0xAA:
		return decodedWithSecondByte("DISARM_ALARM", b2)
	}
	return caslDecodedEventCode{}, false
}

// decodeCASLVBD4Code декодує бінарні коди подій для приладів протоколу VBD4
// (TYPE_DEVICE_Dunay_4_3, TYPE_DEVICE_Dunay_4_3S, TYPE_DEVICE_VBD4_ECOM, TYPE_DEVICE_VBD_16).
// Портовано з TypeScript vbd4Translator.
func decodeCASLVBD4Code(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	if b1 == 0x01 {
		// OO_LINE_BAD: 0x40..0x47
		if b2 >= 0x40 && b2 <= 0x47 {
			return decodedWithOffset("OO_LINE_BAD", b2, -0x3F)
		}
		// OO_LINE_BRK: 0x48..0x4F
		if b2 >= 0x48 && b2 <= 0x4F {
			return decodedWithOffset("OO_LINE_BRK", b2, -0x47)
		}
		// OO_LINE_KZ: 0x50..0x57
		if b2 >= 0x50 && b2 <= 0x57 {
			return decodedWithOffset("OO_LINE_KZ", b2, -0x4F)
		}
		// OO_LINE_NORM: 0x58..0x5F
		if b2 >= 0x58 && b2 <= 0x5F {
			return decodedWithOffset("OO_LINE_NORM", b2, -0x57)
		}
		switch b2 {
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
		// OO_GROUP_OFF: 0x70..0x73
		if b2 >= 0x70 && b2 <= 0x73 {
			return decodedWithOffset("OO_GROUP_OFF", b2, -0x6F)
		}
		// OO_GROUP_ON: 0x78..0x7B
		if b2 >= 0x78 && b2 <= 0x7B {
			return decodedWithOffset("OO_GROUP_ON", b2, -0x77)
		}
		return decodedStatic("UNKNOWN")
	}
	if b1 == 0x02 {
		// ID_HOZ: 0x28..0x3F
		if b2 >= 0x28 && b2 <= 0x3F {
			return decodedWithOffset("ID_HOZ", b2, -0x27)
		}
		return decodedStatic("UNKNOWN")
	}

	// LINE_BAD: b1 0x40..0x47
	if b1 >= 0x40 && b1 <= 0x47 {
		return decodedWithOffset("LINE_BAD", b1, -0x3F)
	}
	// LINE_BRK: b1 0x48..0x4F
	if b1 >= 0x48 && b1 <= 0x4F {
		return decodedWithOffset("LINE_BRK", b1, -0x47)
	}
	// LINE_KZ: b1 0x50..0x57
	if b1 >= 0x50 && b1 <= 0x57 {
		return decodedWithOffset("LINE_KZ", b1, -0x4F)
	}
	// LINE_NORM: b1 0x58..0x5F
	if b1 >= 0x58 && b1 <= 0x5F {
		return decodedWithOffset("LINE_NORM", b1, -0x57)
	}

	switch b1 {
	case 0x60:
		return decodedStatic("PPK_CONN_OK")
	case 0x61:
		return decodedStatic("PPK_NO_CONN")
	case 0x64:
		return decodedStatic("ENABLED")
	case 0x65:
		return decodedStatic("DISABLED")
	case 0x6F:
		return decodedStatic("ENABLED_DISABLED_ERROR")
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
	}

	// GROUP_OFF: b1 0x70..0x73
	if b1 >= 0x70 && b1 <= 0x73 {
		return decodedWithOffset("GROUP_OFF", b1, -0x6F)
	}
	// PRIMUS: b1 0x74..0x77
	if b1 >= 0x74 && b1 <= 0x77 {
		return decodedWithOffset("PRIMUS", b1, -0x73)
	}
	// GROUP_ON: b1 0x78..0x7B
	if b1 >= 0x78 && b1 <= 0x7B {
		return decodedWithOffset("GROUP_ON", b1, -0x77)
	}
	// COERCION: b1 0x7C..0x7F
	if b1 >= 0x7C && b1 <= 0x7F {
		return decodedWithOffset("COERCION", b1, -0x7B)
	}
	if b1 == 0xF0 {
		return decodedStatic("CHECK_CONN")
	}
	return decodedStatic("UNKNOWN")
}

// decodeCASLDozorCode декодує бінарні коди подій для приладів Дозор
// (TYPE_DEVICE_Dozor_4, TYPE_DEVICE_Dozor_8, TYPE_DEVICE_Dozor_8MG).
// Портовано з TypeScript dozorTranslator.
func decodeCASLDozorCode(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	if b1 == 0x01 {
		switch b2 {
		case 0x63:
			return decodedStatic("OO_PPK_BAD")
		case 0x64:
			return decodedStatic("ENABLED")
		case 0x65:
			return decodedStatic("DISABLED")
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
		case 0x6F:
			return decodedStatic("OO_ENABLED_DISABLED_ERROR")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	}

	switch b1 {
	case 0x08:
		return decodedWithOffset("FIRE_PL_CONT_POWER_OK", b2, -0x10+1)
	case 0x09:
		return decodedWithOffset("OO_FIRE_PL_CONT_POWER_OK", b2, -0x10+1)
	case 0x0A:
		return decodedWithOffset("FIRE_PL_CONT_POWER_BAD", b2, -0x10+1)
	case 0x0B:
		return decodedWithOffset("OO_FIRE_PL_CONT_POWER_BAD", b2, -0x10+1)
	case 0x0C:
		return decodedWithOffset("FIRE_PL_CONT_POWER_OFF", b2, -0x10+1)
	case 0x0D:
		return decodedWithOffset("OO_FIRE_PL_CONT_POWER_OFF", b2, -0x10+1)
	case 0x0E:
		if b2 == 0x00 || b2 == 0x10 {
			return decodedStatic("OK_220")
		}
		if b2 == 0x01 || b2 == 0x11 {
			return decodedStatic("OO_OK_220")
		}
		return decodedWithSecondByte("UNKNOWN", b2)
	case 0x0F:
		if b2 == 0x00 || b2 == 0x10 {
			return decodedStatic("POWER_SUP_BAD")
		}
		if b2 == 0x01 || b2 == 0x11 {
			return decodedStatic("OO_POWER_SUP_BAD")
		}
		return decodedWithSecondByte("UNKNOWN", b2)
	case 0x30:
		switch b2 {
		case 0x10:
			return decodedStatic("NO_220")
		case 0x11:
			return decodedStatic("REL_MOD1_POWER_BAD")
		case 0x12:
			return decodedStatic("REL_MOD2_POWER_BAD")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	case 0x32:
		switch b2 {
		case 0x10:
			return decodedStatic("OK_220")
		case 0x11:
			return decodedStatic("REL_MOD1_POWER_OK")
		case 0x12:
			return decodedStatic("REL_MOD2_POWER_OK")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	case 0x33:
		switch b2 {
		case 0x10:
			return decodedStatic("OO_OK_220")
		case 0x11:
			return decodedStatic("OO_REL_MOD1_POWER_OK")
		case 0x12:
			return decodedStatic("OO_REL_MOD2_POWER_OK")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	case 0x34:
		return decodedWithOffset("REL_MOD_CONN_NO", b2, -0x10+1)
	case 0x35:
		return decodedWithOffset("OO_REL_MOD_CONN_NO", b2, -0x10+1)
	case 0x36:
		return decodedWithOffset("REL_MOD_CONN_OK", b2, -0x10+1)
	case 0x37:
		return decodedWithOffset("OO_REL_MOD_CONN_OK", b2, -0x10+1)
	case 0x38:
		return decodedWithOffset("RESERVE", b2, -0x10+1)
	case 0x39:
		return decodedWithOffset("RESERVE", b2, -0x10+1)
	case 0x3A:
		return decodedWithOffset("ID_HOZ", b2, -0x10+1)
	case 0x3B:
		return decodedWithOffset("ID_HOZ", b2, -0x10+1+32)
	case 0x3C:
		return decodedWithOffset("RESERVE", b2, -0x10+1)
	case 0x3D:
		return decodedWithOffset("MICPRG_V_CODE_DOZOR", b2, -0x10+1)
	case 0x3E:
		return decodedWithOffset("RESERVE", b2, -0x10+1)
	case 0x3F:
		switch b2 {
		case 0x10:
			return decodedStatic("RESTART")
		case 0x11:
			return decodedStatic("OO_CHECK_CONN")
		case 0x12:
			return decodedStatic("RESERVE")
		case 0x13:
			return decodedStatic("SYS_ERR")
		case 0x14:
			return decodedStatic("PPK_EDIT_CONF_ADMIN")
		case 0x15:
			return decodedStatic("ENABLED")
		case 0x16:
			return decodedStatic("DISABLED")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	case 0x50:
		return decodedWithOffset("ZONE_FIRE", b2, -0x10+1)
	case 0x51:
		return decodedWithOffset("OO_ZONE_FIRE", b2, -0x10+1)
	case 0x52:
		return decodedWithOffset("ZONE_OFF", b2, -0x10+1)
	case 0x53:
		return decodedWithOffset("OO_ZONE_OFF", b2, -0x10+1)
	case 0x54:
		return decodedWithOffset("ZONE_BAD", b2, -0x10+1)
	case 0x55:
		return decodedWithOffset("OO_ZONE_BAD", b2, -0x10+1)
	case 0x56:
		return decodedWithOffset("ZONE_NORM", b2, -0x10+1)
	case 0x57:
		return decodedWithOffset("OO_ZONE_NORM", b2, -0x10+1)
	case 0x58:
		return decodedWithOffset("ZONE_FIRE_THR_OFF", b2, -0x10+1)
	case 0x5A:
		return decodedWithOffset("BREAK_UZ", b2, -0x10+1)
	case 0x5B:
		return decodedWithOffset("OO_BREAK_UZ", b2, -0x10+1)
	case 0x5C:
		return decodedWithOffset("FIRE_ON_PL", b2, -0x10+1)
	case 0x5D:
		return decodedWithOffset("OO_FIRE_ON_PL", b2, -0x10+1)
	case 0x5E:
		return decodedWithOffset("PL_OFF", b2, -0x10+1)
	case 0x5F:
		return decodedWithOffset("OO_PL_OFF", b2, -0x10+1)
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
		return decodedWithOffset("LINE_NORM", b2, -0x10+1)
	case 0x71:
		return decodedWithOffset("OO_LINE_NORM", b2, -0x10+1)
	case 0x72:
		return decodedWithOffset("LINE_BRK", b2, -0x10+1)
	case 0x73:
		return decodedWithOffset("OO_LINE_BRK", b2, -0x10+1)
	case 0x74:
		return decodedWithOffset("LINE_KZ", b2, -0x10+1)
	case 0x75:
		return decodedWithOffset("OO_LINE_KZ", b2, -0x10+1)
	case 0x76:
		if b2 < 0x04 {
			return decodedWithOffset("UZ_OUTPUT_OFF", b2, -0x10+1)
		}
		return decodedWithOffset("REL_OFF", b2, -0x10+1-4)
	case 0x77:
		if b2 < 0x04 {
			return decodedWithOffset("OO_UZ_OUTPUT_OFF", b2, -0x10+1)
		}
		return decodedWithOffset("OO_REL_OFF", b2, -0x10+1-4)
	case 0x78:
		if b2 < 0x04 {
			return decodedWithOffset("UZ_OUTPUT_ON", b2, -0x10+1)
		}
		return decodedWithOffset("REL_ON", b2, -0x10+1-4)
	case 0x79:
		if b2 < 0x04 {
			return decodedWithOffset("OO_UZ_OUTPUT_ON", b2, -0x10+1)
		}
		return decodedWithOffset("OO_REL_ON", b2, -0x10+1-4)
	case 0x7A:
		if b2 < 0x04 {
			return decodedWithOffset("UZ_OUTPUT_OFF", b2, -0x10+1)
		}
		return decodedWithOffset("REL_OFF", b2, -0x10+1-4)
	case 0x7B:
		if b2 < 0x04 {
			return decodedWithOffset("OO_UZ_OUTPUT_OFF", b2, -0x10+1)
		}
		return decodedWithOffset("OO_REL_OFF", b2, -0x10+1-4)
	case 0x7C:
		return decodedWithOffset("UZ_OUTPUT_KZ", b2, -0x10+1)
	case 0x7D:
		return decodedWithOffset("OO_UZ_OUTPUT_KZ", b2, -0x10+1)
	case 0x7E:
		return decodedWithOffset("UZ_OUTPUT_OK", b2, -0x10+1)
	case 0x7F:
		return decodedWithOffset("OO_UZ_OUTPUT_OK", b2, -0x10+1)
	}

	return decodedWithOffset("UNKNOWN", b1, 0)
}

// decodeCASLD128StdSchema реалізує логіку stdSchema з TypeScript:
// обчислює номер зони/групи з урахуванням зсуву залежно від того,
// в якому діапазоні знаходиться b1 (g1/g2) та повертає відповідний message key.
func decodeCASLD128StdSchema(g1 [4]byte, g2 [4]byte, b1 byte, b2 byte, msgPrimary string, msgSecondary string) (caslDecodedEventCode, bool) {
	num := int(b2) + 1
	switch {
	case b1 == g1[0] || b1 == g2[0]:
		num -= 0x10
	case b1 == g1[1] || b1 == g2[1]:
		num += 0x10
	case b1 == g1[2] || b1 == g2[2]:
		num += 0x30
	case b1 == g1[3] || b1 == g2[3]:
		num += 0x50
	}

	for _, g := range g1 {
		if b1 == g {
			return caslDecodedEventCode{MessageKey: msgPrimary, Number: num, HasNumber: true}, true
		}
	}
	return caslDecodedEventCode{MessageKey: msgSecondary, Number: num, HasNumber: true}, true
}

// decodeCASLD128Code декодує бінарні коди подій для приладів Дунай-128
// (TYPE_DEVICE_Dunay_16_32, TYPE_DEVICE_Dunay_8_32, TYPE_DEVICE_Dunay_PSPN_ECOM).
// Портовано з TypeScript d128Translator.
func decodeCASLD128Code(b1 byte, b2 byte) (caslDecodedEventCode, bool) {
	if b1 == 0x01 {
		switch b2 {
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
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	}

	// ID_HOZ / ID_HOZP: b1 0x08..0x0F
	if b1 >= 0x08 && b1 <= 0x0F {
		return decodeCASLD128StdSchema(
			[4]byte{0x08, 0x0A, 0x0C, 0x0E},
			[4]byte{0x09, 0x0B, 0x0D, 0x0F},
			b1, b2, "ID_HOZ", "ID_HOZP",
		)
	}

	switch b1 {
	case 0x30:
		return decodedWithOffset("BRK_AD", b2, -0x10+1)
	case 0x31:
		return decodedWithOffset("OO_BRK_AD", b2, -0x10+1)
	case 0x32:
		return decodedWithOffset("AD_DOOR_CL", b2, -0x10+1)
	case 0x33:
		return decodedWithOffset("OO_AD_DOOR_CL", b2, -0x10+1)
	case 0x34:
		return decodedWithOffset("AD_NO_CONN", b2, -0x10+1)
	case 0x35:
		return decodedWithOffset("OO_AD_NO_CONN", b2, -0x10+1)
	case 0x36:
		return decodedWithOffset("AD_CONN_OK", b2, -0x10+1)
	case 0x37:
		return decodedWithOffset("OO_AD_CONN_OK", b2, -0x10+1)
	case 0x38:
		return decodedWithOffset("AD_BAD_FOOD", b2, -0x10+1)
	case 0x39:
		return decodedWithOffset("OO_ALM_AD_POWER", b2, -0x10+1)
	case 0x3A:
		return decodedWithOffset("AD_FOOD_OK", b2, -0x10+1)
	case 0x3B:
		return decodedWithOffset("OO_AD_POWER_OK", b2, -0x10+1)
	case 0x3C:
		return decodedWithOffset("SABOTAGE_AD", b2, -0x10+1)
	case 0x3D:
		return decodedWithOffset("MICPRG_V_CODE", b2, -0x10)
	case 0x3E:
		return decodedWithOffset("AD_COM_ERR_DATA_NET485", b2, -0x10+1)
	case 0x3F:
		switch b2 {
		case 0x10:
			return decodedStatic("RESTART")
		case 0x11:
			return decodedStatic("OO_CHECK_CONN")
		case 0x12:
			return decodedStatic("DECONCERV")
		case 0x13:
			return decodedStatic("CONCERV")
		case 0x14:
			return decodedStatic("EDIT_CONF")
		case 0x15:
			return decodedStatic("ENABLED")
		case 0x16:
			return decodedStatic("DISABLED")
		default:
			return decodedWithSecondByte("UNKNOWN", b2)
		}
	}

	// GROUP_ON / OO_GROUP_ON: b1 0x40..0x47
	if b1 >= 0x40 && b1 <= 0x47 {
		return decodeCASLD128StdSchema(
			[4]byte{0x40, 0x42, 0x44, 0x46},
			[4]byte{0x41, 0x43, 0x45, 0x47},
			b1, b2, "GROUP_ON", "OO_GROUP_ON",
		)
	}
	// GROUP_OFF / OO_GROUP_OFF: b1 0x48..0x4F
	if b1 >= 0x48 && b1 <= 0x4F {
		return decodeCASLD128StdSchema(
			[4]byte{0x48, 0x4A, 0x4C, 0x4E},
			[4]byte{0x49, 0x4B, 0x4D, 0x4F},
			b1, b2, "GROUP_OFF", "OO_GROUP_OFF",
		)
	}
	// LINE_BRK / OO_LINE_BRK: b1 0x50..0x57
	if b1 >= 0x50 && b1 <= 0x57 {
		return decodeCASLD128StdSchema(
			[4]byte{0x50, 0x52, 0x54, 0x56},
			[4]byte{0x51, 0x53, 0x55, 0x57},
			b1, b2, "LINE_BRK", "OO_LINE_BRK",
		)
	}
	// LINE_NORM / OO_LINE_NORM: b1 0x58..0x5F
	if b1 >= 0x58 && b1 <= 0x5F {
		return decodeCASLD128StdSchema(
			[4]byte{0x58, 0x5A, 0x5C, 0x5E},
			[4]byte{0x59, 0x5B, 0x5D, 0x5F},
			b1, b2, "LINE_NORM", "OO_LINE_NORM",
		)
	}

	switch b1 {
	case 0x60:
		return decodedStatic("PPK_CONN_OK")
	case 0x61:
		return decodedStatic("PPK_NO_CONN")
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
	}

	// LINE_KZ / OO_LINE_KZ: b1 0x70..0x77
	if b1 >= 0x70 && b1 <= 0x77 {
		return decodeCASLD128StdSchema(
			[4]byte{0x70, 0x72, 0x74, 0x76},
			[4]byte{0x71, 0x73, 0x75, 0x77},
			b1, b2, "LINE_KZ", "OO_LINE_KZ",
		)
	}
	// LINE_BAD / OO_LINE_BAD: b1 0x78..0x7F
	if b1 >= 0x78 && b1 <= 0x7F {
		return decodeCASLD128StdSchema(
			[4]byte{0x78, 0x7A, 0x7C, 0x7E},
			[4]byte{0x79, 0x7B, 0x7D, 0x7F},
			b1, b2, "LINE_BAD", "OO_LINE_BAD",
		)
	}

	return decodedStatic("UNKNOWN")
}
