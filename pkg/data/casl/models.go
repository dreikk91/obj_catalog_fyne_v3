package casl

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// SessionInfo keeps the current auth/session details.
type SessionInfo struct {
	Token  string
	WSURL  string
	UserID string
	PultID int64
}

type Int64 int64

func (v Int64) Int64() int64 { return int64(v) }

func (v *Int64) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = 0
		return nil
	}

	if strings.HasPrefix(raw, "\"") {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return nil
		}
		value = strings.TrimSpace(value)
		if value == "" {
			*v = 0
			return nil
		}
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			*v = Int64(i)
			return nil
		}
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			*v = Int64(int64(f))
			return nil
		}
		*v = 0
		return nil
	}

	if i, err := strconv.ParseInt(raw, 10, 64); err == nil {
		*v = Int64(i)
		return nil
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		*v = Int64(int64(f))
		return nil
	}

	*v = 0
	return nil
}

type Text string

func (v Text) String() string { return strings.TrimSpace(string(v)) }

func (v *Text) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*v = ""
		return nil
	}

	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*v = Text(text)
		return nil
	}

	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		*v = Text(number.String())
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

	*v = Text(strings.Trim(raw, "\""))
	return nil
}

type Pult struct {
	PultID   string   `json:"pult_id"`
	Name     string   `json:"name"`
	Nickname string   `json:"nickname"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Zoom     int      `json:"zoom"`
	Users    []string `json:"users"`
}

type Room struct {
	RoomID      string `json:"room_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	RTSP        string `json:"rtsp"`
	Users       []User `json:"users"`
}

type GrdObject struct {
	ObjID          string   `json:"obj_id"`
	Name           string   `json:"name"`
	Address        string   `json:"address"`
	Lat            string   `json:"lat"`
	Long           string   `json:"long"`
	Description    string   `json:"description"`
	ReactingPultID string   `json:"reacting_pult_id"`
	Contract       string   `json:"contract"`
	Note           string   `json:"note"`
	StartDate      Int64    `json:"start_date"`
	Status         string   `json:"status"`
	ObjectType     string   `json:"object_type"`
	DeviceNumber   Int64    `json:"device_number"`
	DeviceBlocked  bool     `json:"device_blocked"`
	DeviceID       Int64    `json:"device_id"`
	BlockMessage   Text     `json:"block_message"`
	TimeUnblock    Text     `json:"time_unblock"`
	ManagerID      string   `json:"manager_id"`
	Manager        User     `json:"manager"`
	InCharge       []string `json:"in_charge"`
	Rooms          []Room   `json:"rooms"`
}

type Device struct {
	DeviceID Text         `json:"device_id"`
	ObjID    Text         `json:"obj_id"`
	Number   Int64        `json:"number"`
	Name     Text         `json:"name"`
	Type     Text         `json:"type"`
	SIM1     Text         `json:"sim1"`
	SIM2     Text         `json:"sim2"`
	Lines    []DeviceLine `json:"lines"`
}

type ConnectionRecord struct {
	GuardedObject GrdObject
	Device        Device
}

func (r *ConnectionRecord) UnmarshalJSON(data []byte) error {
	type rawConnection struct {
		GuardedObject    json.RawMessage `json:"guardedObject"`
		GuardedObjectAlt json.RawMessage `json:"guarded_object"`
		Device           json.RawMessage `json:"device"`
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
	_ = json.Unmarshal(guardedObjectRaw, &r.GuardedObject)

	if len(raw.Device) > 0 {
		_ = json.Unmarshal(raw.Device, &r.Device)
	}

	return nil
}

type DeviceLine struct {
	ID     Int64 `json:"id"`
	Name   Text  `json:"name"`
	Number Int64 `json:"number"`
	Type   Text  `json:"type"`
}

func (d *Device) UnmarshalJSON(data []byte) error {
	type rawDevice struct {
		DeviceID Text            `json:"device_id"`
		ObjID    Text            `json:"obj_id"`
		Number   Int64           `json:"number"`
		Name     Text            `json:"name"`
		Type     Text            `json:"type"`
		SIM1     Text            `json:"sim1"`
		SIM2     Text            `json:"sim2"`
		Lines    json.RawMessage `json:"lines"`
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
	d.SIM1 = raw.SIM1
	d.SIM2 = raw.SIM2
	d.Lines = decodeDeviceLines(raw.Lines)
	return nil
}

func decodeDeviceLines(raw json.RawMessage) []DeviceLine {
	body := bytes.TrimSpace(raw)
	if len(body) == 0 || bytes.Equal(body, []byte("null")) {
		return nil
	}

	if body[0] == '[' {
		var lines []DeviceLine
		if err := json.Unmarshal(body, &lines); err == nil {
			return lines
		}
		return nil
	}

	if body[0] != '{' {
		return nil
	}

	var source map[string]json.RawMessage
	if err := json.Unmarshal(body, &source); err != nil {
		return nil
	}
	if len(source) == 0 {
		return nil
	}

	lines := make([]DeviceLine, 0, len(source))
	for key, val := range source {
		var line DeviceLine
		if err := json.Unmarshal(val, &line); err == nil {
			if line.ID.Int64() == 0 {
				if parsed, err := strconv.ParseInt(key, 10, 64); err == nil && parsed > 0 {
					line.ID = Int64(parsed)
				}
			}
			if line.Number.Int64() == 0 {
				line.Number = line.ID
			}
			lines = append(lines, line)
		}
	}
	return lines
}

type PhoneNumber struct {
	Active bool   `json:"active"`
	Number string `json:"number"`
}

type User struct {
	UserID       string        `json:"user_id"`
	Email        string        `json:"email"`
	LastName     string        `json:"last_name"`
	FirstName    string        `json:"first_name"`
	MiddleName   string        `json:"middle_name"`
	Role         string        `json:"role"`
	Tag          Text          `json:"tag"`
	PhoneNumbers []PhoneNumber `json:"phone_numbers"`
}

func (u User) FullName() string {
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

func (u User) PrimaryPhone() string {
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

type ObjectEvent struct {
	PPKNum    Int64  `json:"ppk_num"`
	DeviceID  Text   `json:"device_id"`
	ObjID     Text   `json:"obj_id"`
	ObjName   Text   `json:"obj_name"`
	ObjAddr   Text   `json:"obj_address"`
	Action    Text   `json:"action"`
	AlarmType Text   `json:"alarm_type"`
	MgrID     Text   `json:"mgr_id"`
	UserID    Text   `json:"user_id"`
	UserFIO   Text   `json:"user_fio"`
	Time      Int64  `json:"time"`
	Code      Text   `json:"code"`
	Type      string `json:"type"`
	Number    Int64  `json:"number"`
	ContactID Text   `json:"contact_id"`
	HozUserID Text   `json:"hoz_user_id"`
}

type DeviceState struct {
	Power        Int64 `json:"power"`
	Accum        Int64 `json:"accum"`
	Door         Int64 `json:"door"`
	Online       Int64 `json:"online"`
	LastPingDate Int64 `json:"lastPingDate"`
	Lines        any   `json:"lines"`
	Groups       any   `json:"groups"`
	Adapters     any   `json:"adapters"`
}

type StatsAlarmsData struct {
	DeviceID            string `json:"device_id"`
	ObjectID            string `json:"obj_id"`
	ResponseFrequencies Int64  `json:"responseFrequencies"`
	CommunicQuality     Int64  `json:"communicQuality"`
	PowerFailure        Int64  `json:"powerFailure"`
	Criminogenicity     Int64  `json:"criminogenicity"`
	CustomWins          Int64  `json:"customWins"`
}

type LoginResponse struct {
	Status string `json:"status"`
	UserID string `json:"user_id"`
	FIO    string `json:"fio"`
	Token  string `json:"token"`
	WSURL  string `json:"ws_url"`
	Error  string `json:"error"`
}

type ReadPultResponse struct {
	Status string `json:"status"`
	Data   []Pult `json:"data"`
	Error  string `json:"error"`
}

type ReadGrdObjectResponse struct {
	Status string      `json:"status"`
	Data   []GrdObject `json:"data"`
	Error  string      `json:"error"`
}

type ReadUserResponse struct {
	Status string `json:"status"`
	Data   []User `json:"data"`
	Error  string `json:"error"`
}

type ReadDeviceResponse struct {
	Status string   `json:"status"`
	Data   []Device `json:"data"`
	Error  string   `json:"error"`
}

type ReadEventsByIDResponse struct {
	Status string        `json:"status"`
	Data   []ObjectEvent `json:"data"`
	Events []ObjectEvent `json:"events"`
	Error  string        `json:"error"`
}

type ReadDeviceStateResponse struct {
	Status string      `json:"status"`
	State  DeviceState `json:"state"`
	Error  string      `json:"error"`
}

type GetStatisticResponse struct {
	Status string          `json:"status"`
	Data   StatsAlarmsData `json:"data"`
	Error  string          `json:"error"`
}

type BasketResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
	Error  string `json:"error"`
}

type StatusOnlyResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// Request types from api_support

type CaptchaConfig struct {
	Status               string `json:"status"`
	CaptchaShow          bool   `json:"captchaShow"`
	GoogleCaptchaSiteKey string `json:"GoogleCaptchaSiteKey"`
	Error                string `json:"error"`
}

type AlarmEventDefinition struct {
	Code           string
	IsAlarmInStart int
	IsAlarm        int
}

type ReadEventsByIDRequest struct {
	IsFullEventsInfo bool
	TimeStart        int64
	TimeEnd          int64
	TimeRequest      int64
	ObjIDs           []string
	DeviceIDs        []string
	DeviceNumbers    []int64
}

type GetStatisticRequest struct {
	Name      string
	DeviceID  string
	ObjectID  string
	StartTime int64
	EndTime   int64
	Limit     int
}

type ReadEventsRequest struct {
	TimeStart   int64
	TimeEnd     int64
	TimeRequest int64
}
