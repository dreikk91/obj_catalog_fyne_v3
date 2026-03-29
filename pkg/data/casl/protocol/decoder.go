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

	// Porting the massive Rcom switch from original code
	switch b1 {
	case 0x40:
		return DecodedEvent{MessageKey: "GROUP_ON", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x48:
		return DecodedEvent{MessageKey: "GROUP_OFF", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x50:
		return DecodedEvent{MessageKey: "LINE_BRK", Number: int(b2) - 0x0f, HasNumber: true}, true
	case 0x58:
		return DecodedEvent{MessageKey: "LINE_NORM", Number: int(b2) - 0x0f, HasNumber: true}, true
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
	case 0xF3:
		return DecodedEvent{MessageKey: "ZONE_ALM", Number: int(b2), HasNumber: true}, true
	case 0xF6:
		return DecodedEvent{MessageKey: "ZONE_NORM", Number: int(b2), HasNumber: true}, true
	}
	return DecodedEvent{}, false
}

func GetDecoder(deviceType string) Decoder {
	// Strategy selection based on device type
	return &RcomDecoder{}
}
