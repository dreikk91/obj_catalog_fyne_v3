package data

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/models"
)

type caslProtocolModel int

type caslDecodedEventCode struct {
	MessageKey string
	Number     int
	HasNumber  bool
}

type caslEventContext struct {
	ObjectID         int
	ObjectNum        string
	ObjectName       string
	DeviceType       string
	Translator       map[string]string
	TranslatorAlarms map[string]bool
	LineNames        map[int]string
}

// caslTapeMessage is a native CASL tape ppk_msg row (general_tape_item/general_tape_objects).
type caslTapeMessage struct {
	Time         int64
	Code         string
	DictName     string
	ContactID    string
	Number       int
	EventType    string
	Subtype      string
	Details      string
	MessageKey   string
	Type         models.EventType
	IsAlarm      bool
	HasAlarmFlag bool
}

// caslTapeItem is a native CASL tape object model.
type caslTapeItem struct {
	ID              int
	Time            int64
	ObjectID        int
	ObjectNum       string
	ObjectName      string
	ObjID           string
	DeviceID        string
	DeviceType      string
	ObjAddr         string
	ZoneNumber      int
	Code            string
	ContactID       string
	EventType       string
	Subtype         string
	AlarmType       string
	PultID          string
	UserID          string
	LastAct         string
	Msg             string
	ReasonAlarm     string
	Translator      map[string]string
	TranslatorFlags map[string]bool
	PPKMsgs         []caslTapeMessage
}

type caslObjectStatusState struct {
	Status         models.ObjectStatus
	StatusText     string
	AlarmState     int64
	GuardState     int64
	TechAlarmState int64
	IsConnState    int64
	IsUnderGuard   bool
}

type caslGroupCandidate struct {
	key   string
	value any
}

type caslInt64 int64

func (v caslInt64) Int64() int64 { return int64(v) }

func (v *caslInt64) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}

	if strings.HasPrefix(raw, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("casl int64: decode quoted value: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "null") {
			*v = 0
			return nil
		}
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			*v = caslInt64(i)
			return nil
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			*v = caslInt64(int64(f))
			return nil
		}
		return fmt.Errorf("casl int64: invalid numeric value %q", value)
	}

	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		*v = caslInt64(i)
		return nil
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		*v = caslInt64(int64(f))
		return nil
	}

	return fmt.Errorf("casl int64: invalid numeric token %s", raw)
}

type caslText string

func (v caslText) String() string { return strings.TrimSpace(string(v)) }

func (v *caslText) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = caslText(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		*v = caslText(number.String())
		return nil
	}

	var boolean bool
	if err := json.Unmarshal(data, &boolean); err == nil {
		if boolean {
			*v = "true"
		} else {
			*v = "false"
		}
		return nil
	}

	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		return fmt.Errorf("casl text: expected scalar, got %s", raw)
	}

	return fmt.Errorf("casl text: invalid scalar token %s", raw)
}

type caslNullableFloat64 struct {
	value float64
	valid bool
}

func (v caslNullableFloat64) Float64Ptr() *float64 {
	if !v.valid {
		return nil
	}
	value := v.value
	return &value
}

func (v *caslNullableFloat64) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = caslNullableFloat64{}
		return nil
	}

	if strings.HasPrefix(raw, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return fmt.Errorf("casl float64: decode quoted value: %w", err)
		}
		value = strings.TrimSpace(value)
		if value == "" || strings.EqualFold(value, "null") {
			*v = caslNullableFloat64{}
			return nil
		}
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			*v = caslNullableFloat64{value: parsed, valid: true}
			return nil
		}
		return fmt.Errorf("casl float64: invalid numeric value %q", value)
	}

	if parsed, err := strconv.ParseFloat(raw, 64); err == nil {
		*v = caslNullableFloat64{value: parsed, valid: true}
		return nil
	}

	return fmt.Errorf("casl float64: invalid numeric token %s", raw)
}

type caslPult struct {
	PultID   string   `json:"pult_id"`
	Name     string   `json:"name"`
	Nickname string   `json:"nickname"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Zoom     int      `json:"zoom"`
	Users    []string `json:"users"`
}

type caslRoom struct {
	RoomID      string     `json:"room_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RTSP        string     `json:"rtsp"`
	Users       []caslUser `json:"users"`
}

type caslGrdObject struct {
	ObjID          string     `json:"obj_id"`
	Name           string     `json:"name"`
	Address        string     `json:"address"`
	Lat            string     `json:"lat"`
	Long           string     `json:"long"`
	Description    string     `json:"description"`
	ReactingPultID string     `json:"reacting_pult_id"`
	Contract       string     `json:"contract"`
	Note           string     `json:"note"`
	StartDate      caslInt64  `json:"start_date"`
	Status         string     `json:"status"`
	ObjectType     string     `json:"object_type"`
	DeviceNumber   caslInt64  `json:"device_number"`
	DeviceBlocked  bool       `json:"device_blocked"`
	DeviceID       caslInt64  `json:"device_id"`
	BlockMessage   caslText   `json:"block_message"`
	TimeUnblock    caslText   `json:"time_unblock"`
	ManagerID      string     `json:"manager_id"`
	Manager        caslUser   `json:"manager"`
	InCharge       []string   `json:"in_charge"`
	Rooms          []caslRoom `json:"rooms"`
}

type caslDevice struct {
	DeviceID     caslText         `json:"device_id"`
	ObjID        caslText         `json:"obj_id"`
	Number       caslInt64        `json:"number"`
	Name         caslText         `json:"name"`
	Type         caslText         `json:"type"`
	Timeout      caslInt64        `json:"timeout"`
	LastPingDate caslInt64        `json:"lastPingDate"`
	Blocked      bool             `json:"blocked"`
	SIM1         caslText         `json:"sim1"`
	SIM2         caslText         `json:"sim2"`
	Lines        []caslDeviceLine `json:"lines"`
}

type caslConnectionRecord struct {
	GuardedObject caslGrdObject
	Device        caslDevice
}

func (r *caslConnectionRecord) UnmarshalJSON(data []byte) error {
	type rawConnection struct {
		GuardedObject    json.RawMessage `json:"guardedObject"`
		GuardedObjectAlt json.RawMessage `json:"guarded_object"`
		Device           json.RawMessage `json:"device"`
		Devices          json.RawMessage `json:"devices"`
	}

	var raw rawConnection
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	guardedObjectRaw := raw.GuardedObject
	if len(guardedObjectRaw) == 0 {
		guardedObjectRaw = raw.GuardedObjectAlt
	}
	if len(guardedObjectRaw) == 0 {
		guardedObjectRaw = data
	}
	if err := json.Unmarshal(guardedObjectRaw, &r.GuardedObject); err != nil {
		return fmt.Errorf("casl connection guarded object: %w", err)
	}

	if len(raw.Device) > 0 {
		if err := json.Unmarshal(raw.Device, &r.Device); err != nil {
			return fmt.Errorf("casl connection device: %w", err)
		}
	}
	if deviceMap, ok := findCASLDeviceMapInAny(raw.Devices, "", strings.TrimSpace(r.GuardedObject.ObjID), r.GuardedObject.DeviceNumber.Int64()); ok {
		var overlay caslDevice
		encoded, err := json.Marshal(deviceMap)
		if err != nil {
			return fmt.Errorf("casl connection devices overlay: %w", err)
		}
		if err := json.Unmarshal(encoded, &overlay); err != nil {
			return fmt.Errorf("casl connection devices overlay: %w", err)
		}
		overlayCASLCoreDevice(&r.Device, overlay)
	}

	normalizeCASLObjectRecord(&r.GuardedObject, r.Device)
	if strings.TrimSpace(r.Device.ObjID.String()) == "" && strings.TrimSpace(r.GuardedObject.ObjID) != "" {
		r.Device.ObjID = caslText(r.GuardedObject.ObjID)
	}
	return nil
}

func (r caslConnectionRecord) hasPayload() bool {
	if strings.TrimSpace(r.GuardedObject.ObjID) != "" || strings.TrimSpace(r.GuardedObject.Name) != "" {
		return true
	}
	if strings.TrimSpace(r.Device.DeviceID.String()) != "" || r.Device.Number.Int64() > 0 {
		return true
	}
	return false
}

func (d *caslDevice) UnmarshalJSON(data []byte) error {
	type rawDevice struct {
		DeviceID     caslText        `json:"device_id"`
		ObjID        caslText        `json:"obj_id"`
		Number       caslInt64       `json:"number"`
		Name         caslText        `json:"name"`
		Type         caslText        `json:"type"`
		Timeout      caslInt64       `json:"timeout"`
		LastPingDate caslInt64       `json:"lastPingDate"`
		Blocked      bool            `json:"blocked"`
		SIM1         caslText        `json:"sim1"`
		SIM2         caslText        `json:"sim2"`
		Lines        json.RawMessage `json:"lines"`
	}

	var raw rawDevice
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.DeviceID = raw.DeviceID
	d.ObjID = raw.ObjID
	d.Number = raw.Number
	d.Name = raw.Name
	d.Type = raw.Type
	d.Timeout = raw.Timeout
	d.LastPingDate = raw.LastPingDate
	d.Blocked = raw.Blocked
	d.SIM1 = raw.SIM1
	d.SIM2 = raw.SIM2
	d.Lines = decodeCASLDeviceLines(raw.Lines)
	return nil
}

func overlayCASLCoreDevice(base *caslDevice, overlay caslDevice) {
	if base == nil {
		return
	}
	if strings.TrimSpace(base.DeviceID.String()) == "" {
		base.DeviceID = overlay.DeviceID
	}
	if strings.TrimSpace(base.ObjID.String()) == "" {
		base.ObjID = overlay.ObjID
	}
	if base.Number.Int64() <= 0 {
		base.Number = overlay.Number
	}
	if strings.TrimSpace(base.Name.String()) == "" {
		base.Name = overlay.Name
	}
	if strings.TrimSpace(base.Type.String()) == "" {
		base.Type = overlay.Type
	}
	if base.Timeout.Int64() <= 0 {
		base.Timeout = overlay.Timeout
	}
	if base.LastPingDate.Int64() <= 0 {
		base.LastPingDate = overlay.LastPingDate
	}
	if !base.Blocked {
		base.Blocked = overlay.Blocked
	}
	if strings.TrimSpace(base.SIM1.String()) == "" {
		base.SIM1 = overlay.SIM1
	}
	if strings.TrimSpace(base.SIM2.String()) == "" {
		base.SIM2 = overlay.SIM2
	}
	if len(base.Lines) == 0 {
		base.Lines = overlay.Lines
	}
}

type caslDeviceLine struct {
	ID            caslInt64 `json:"id"`
	Name          caslText  `json:"name"`
	Number        caslInt64 `json:"number"`
	Type          caslText  `json:"type"`
	GroupID       caslText  `json:"group_id"`
	Group         caslText  `json:"group"`
	GroupNumber   caslInt64 `json:"group_number"`
	RoomID        caslText  `json:"room_id"`
	AdapterType   caslText  `json:"adapter_type"`
	AdapterNumber caslInt64 `json:"adapter_number"`
	Description   caslText  `json:"description"`
	LineType      caslText  `json:"line_type"`
	IsBlocked     bool      `json:"isBlocked"`
}

func (l *caslDeviceLine) UnmarshalJSON(data []byte) error {
	type rawLine struct {
		ID            caslInt64 `json:"id"`
		LineID        caslInt64 `json:"line_id"`
		Name          caslText  `json:"name"`
		Description   caslText  `json:"description"`
		Number        caslInt64 `json:"number"`
		LineNumber    caslInt64 `json:"line_number"`
		Type          caslText  `json:"type"`
		LineType      caslText  `json:"line_type"`
		GroupID       caslText  `json:"group_id"`
		Group         caslText  `json:"group"`
		GroupNumber   caslInt64 `json:"group_number"`
		RoomID        caslText  `json:"room_id"`
		AdapterType   caslText  `json:"adapter_type"`
		AdapterNumber caslInt64 `json:"adapter_number"`
		IsBlocked     bool      `json:"isBlocked"`
	}

	var raw rawLine
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	l.ID = raw.ID
	if l.ID.Int64() <= 0 {
		l.ID = raw.LineID
	}
	l.Name = raw.Name
	if l.Name.String() == "" {
		l.Name = raw.Description
	}
	l.Number = raw.Number
	if l.Number.Int64() <= 0 {
		l.Number = raw.LineNumber
	}
	l.Type = raw.Type
	if l.Type.String() == "" {
		l.Type = raw.LineType
	}
	l.GroupID = raw.GroupID
	l.Group = raw.Group
	l.GroupNumber = raw.GroupNumber
	l.RoomID = raw.RoomID
	l.AdapterType = raw.AdapterType
	l.AdapterNumber = raw.AdapterNumber
	l.Description = raw.Description
	l.LineType = raw.LineType
	l.IsBlocked = raw.IsBlocked
	return nil
}

type caslUser struct {
	UserID       string            `json:"user_id"`
	Email        string            `json:"email"`
	LastName     string            `json:"last_name"`
	FirstName    string            `json:"first_name"`
	MiddleName   string            `json:"middle_name"`
	Role         string            `json:"role"`
	Tag          caslText          `json:"tag"`
	PhoneNumbers []caslPhoneNumber `json:"phone_numbers"`
}

func (u caslUser) FullName() string {
	parts := []string{strings.TrimSpace(u.LastName), strings.TrimSpace(u.FirstName), strings.TrimSpace(u.MiddleName)}
	filtered := make([]string, 0, 3)
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return "Користувач #" + strings.TrimSpace(u.UserID)
	}
	return strings.Join(filtered, " ")
}

func (u caslUser) PrimaryPhone() string {
	for _, phone := range u.PhoneNumbers {
		if phone.Active && strings.TrimSpace(phone.Number) != "" {
			return strings.TrimSpace(phone.Number)
		}
	}
	for _, phone := range u.PhoneNumbers {
		if strings.TrimSpace(phone.Number) != "" {
			return strings.TrimSpace(phone.Number)
		}
	}
	if strings.TrimSpace(u.Tag.String()) != "" {
		return strings.TrimSpace(u.Tag.String())
	}
	return ""
}

type caslDeviceState struct {
	Power        caslInt64 `json:"power"`
	Accum        caslInt64 `json:"accum"`
	Door         caslInt64 `json:"door"`
	Online       caslInt64 `json:"online"`
	LastPingDate caslInt64 `json:"lastPingDate"`
	Lines        any       `json:"lines"`
	Groups       any       `json:"groups"`
	Adapters     any       `json:"adapters"`
}

type caslStatsAlarmsData struct {
	DeviceID            string    `json:"device_id"`
	ObjectID            string    `json:"obj_id"`
	ResponseFrequencies caslInt64 `json:"responseFrequencies"`
	CommunicQuality     caslInt64 `json:"communicQuality"`
	PowerFailure        caslInt64 `json:"powerFailure"`
	Criminogenicity     caslInt64 `json:"criminogenicity"`
	CustomWins          caslInt64 `json:"customWins"`
}

type caslStatusOnlyResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}
