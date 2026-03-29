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

	// Porting the exhaustive Rcom switch from original code
	switch b1 {
	case 0x08:
		switch b2 {
		case 0x27:
			return DecodedEvent{MessageKey: "ID_HOZ", Number: int(b2) - 0x0f, HasNumber: true}, true
		default:
			return DecodedEvent{MessageKey: "ID_HOZ", Number: int(b2) - 0x0f, HasNumber: true}, true
		}
	case 0x3F:
		switch b2 {
		case 0x09, 0x8F:
			return DecodedEvent{MessageKey: "COERCION"}, true
		case 0x11, 0x91:
			return DecodedEvent{MessageKey: "CHECK_CONN"}, true
		}
	case 0x40, 0x42, 0x44, 0x46:
		offset := -0x0f
		if b1 == 0x42 { offset = 17 }
		if b1 == 0x44 { offset = 49 }
		if b1 == 0x46 { offset = 81 }
		return DecodedEvent{MessageKey: "GROUP_ON", Number: int(b2) + offset, HasNumber: true}, true
	case 0x48, 0x4A, 0x4C, 0x4E:
		offset := -0x0f
		if b1 == 0x4A { offset = 17 }
		if b1 == 0x4C { offset = 49 }
		if b1 == 0x4E { offset = 81 }
		return DecodedEvent{MessageKey: "GROUP_OFF", Number: int(b2) + offset, HasNumber: true}, true
	case 0x50, 0x52, 0x54, 0x56:
		offset := -0x0f
		if b1 == 0x52 { offset = 17 }
		if b1 == 0x54 { offset = 49 }
		if b1 == 0x56 { offset = 81 }
		return DecodedEvent{MessageKey: "LINE_BRK", Number: int(b2) + offset, HasNumber: true}, true
	case 0x58, 0x5A, 0x5C, 0x5E:
		offset := -0x0f
		if b1 == 0x5A { offset = 17 }
		if b1 == 0x5C { offset = 49 }
		if b1 == 0x5E { offset = 81 }
		return DecodedEvent{MessageKey: "LINE_NORM", Number: int(b2) + offset, HasNumber: true}, true
	case 0x60:
		return DecodedEvent{MessageKey: "PPK_CONN_OK"}, true
	case 0x61:
		return DecodedEvent{MessageKey: "PPK_NO_CONN"}, true
	case 0x68:
		return DecodedEvent{MessageKey: "NO_220"}, true
	case 0x69:
		return DecodedEvent{MessageKey: "OK_220"}, true
	case 0x6A:
		return DecodedEvent{MessageKey: "ACC_OK"}, true
	case 0x6B:
		return DecodedEvent{MessageKey: "ACC_BAD"}, true
	case 0x6C:
		return DecodedEvent{MessageKey: "DOOR_OP"}, true
	case 0x6D:
		return DecodedEvent{MessageKey: "DOOR_CL"}, true
	case 0x70, 0x72, 0x74, 0x76:
		return DecodedEvent{MessageKey: "LINE_KZ", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x78, 0x7A, 0x7C, 0x7E:
		return DecodedEvent{MessageKey: "LINE_BAD", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0xA0:
		return DecodedEvent{MessageKey: "SMOKE", Number: int(b2), HasNumber: true}, true
	case 0xA1:
		return DecodedEvent{MessageKey: "HEAT", Number: int(b2), HasNumber: true}, true
	case 0xF3:
		return DecodedEvent{MessageKey: "ZONE_ALM", Number: int(b2), HasNumber: true}, true
	case 0xF4:
		return DecodedEvent{MessageKey: "ALM_BTN_PRS", Number: int(b2), HasNumber: true}, true
	case 0xF6:
		return DecodedEvent{MessageKey: "ZONE_NORM", Number: int(b2), HasNumber: true}, true
	case 0xFD:
		return DecodedEvent{MessageKey: "ALM_INNER_ZONE", Number: int(b2), HasNumber: true}, true
	case 0xFE:
		return DecodedEvent{MessageKey: "NORM_INNER_ZONE", Number: int(b2), HasNumber: true}, true
	}
	return DecodedEvent{}, false
}

func GetDecoder(deviceType string) Decoder {
	return &RcomDecoder{}
}
