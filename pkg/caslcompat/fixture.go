package caslcompat

type Fixture struct {
	User                FixtureUser
	Users               []FixtureUser
	Managers            []map[string]any
	Pults               []map[string]any
	Objects             []FixtureObject
	Devices             []FixtureDevice
	Connections         []FixtureConnection
	Rooms               []FixtureRoom
	Dictionary          map[string]any
	MessageTranslators  map[string][]FixtureTranslatorRow
	GeneralTape         []FixtureTapeRow
	GeneralTapeItems    map[string][]FixtureEvent
	DisconnectedDevices []FixtureDisconnectedDevice
	Statistics          map[string]any
	AlarmEvents         []map[string]any
}

type FixtureUser struct {
	UserID       string           `json:"user_id"`
	Email        string           `json:"email"`
	Role         string           `json:"role"`
	FirstName    string           `json:"first_name"`
	LastName     string           `json:"last_name"`
	MiddleName   string           `json:"middle_name"`
	PultID       int              `json:"pult_id"`
	Images       []any            `json:"images"`
	PhoneNumbers []map[string]any `json:"phone_numbers"`
	Tag          string           `json:"tag"`
	DeviceIDs    []any            `json:"device_ids"`
	OneboxID     string           `json:"onebox_id"`
	UserNotif    map[string]any   `json:"user_notif"`
	BasketID     int              `json:"basket_id"`
	WSURL        string           `json:"ws_url,omitempty"`
}

type FixtureObject struct {
	ObjID          int    `json:"obj_id"`
	DisplayNumber  string `json:"display_number,omitempty"`
	Name           string `json:"name"`
	Address        string `json:"address"`
	Lat            string `json:"lat"`
	Long           string `json:"long"`
	Description    string `json:"description"`
	Contract       string `json:"contract"`
	Status         string `json:"status"`
	ObjectType     string `json:"object_type"`
	ReactingPultID int    `json:"reacting_pult_id"`
}

type FixtureDevice struct {
	DeviceID          int                    `json:"device_id"`
	ObjID             int                    `json:"obj_id"`
	Number            int                    `json:"number"`
	Name              string                 `json:"name"`
	Type              string                 `json:"type"`
	DeviceType        string                 `json:"device_type"`
	SignalLevel       int                    `json:"signal_level,omitempty"`
	Timeout           int                    `json:"timeout"`
	Sim1              string                 `json:"sim1"`
	Sim2              string                 `json:"sim2"`
	TechnicianID      string                 `json:"technician_id"`
	Units             string                 `json:"units"`
	Requisites        string                 `json:"requisites"`
	ChangeDate        string                 `json:"change_date"`
	ReglamentDate     string                 `json:"reglament_date"`
	MoreAlarmTime     []map[string]any       `json:"moreAlarmTime"`
	IgnoringAlarmTime []map[string]any       `json:"ignoringAlarmTime"`
	LicenceKey        string                 `json:"licence_key"`
	PasswRemote       string                 `json:"passw_remote"`
	Enabled           int64                  `json:"enabled"`
	Offline           int64                  `json:"offline"`
	Disconnected      bool                   `json:"disconnected"`
	Blocked           bool                   `json:"blocked"`
	Lines             map[string]FixtureLine `json:"lines"`
}

type FixtureLine struct {
	LineID        int    `json:"line_id"`
	LineNumber    int    `json:"line_number"`
	AdapterType   string `json:"adapter_type"`
	LineType      string `json:"line_type"`
	Description   string `json:"description"`
	GroupNumber   int    `json:"group_number"`
	RoomID        string `json:"room_id,omitempty"`
	AdapterNumber int    `json:"adapter_number"`
	IsBroken      int    `json:"is_broken"`
	IsBlocked     bool   `json:"isBlocked"`
}

type FixtureConnection struct {
	GuardedObject FixtureObject `json:"guardedObject"`
	Device        FixtureDevice `json:"device"`
}

type FixtureRoom struct {
	RoomID      string                     `json:"room_id"`
	ObjID       string                     `json:"obj_id,omitempty"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	RTSP        string                     `json:"rtsp"`
	Images      []any                      `json:"images"`
	Lines       map[string]FixtureRoomLine `json:"lines,omitempty"`
	Users       []FixtureRoomUser          `json:"users,omitempty"`
}

type FixtureRoomLine struct {
	LineID        int    `json:"line_id,omitempty"`
	LineNumber    int    `json:"line_number,omitempty"`
	AdapterType   string `json:"adapter_type"`
	LineType      string `json:"line_type,omitempty"`
	Description   string `json:"description,omitempty"`
	GroupNumber   int    `json:"group_number"`
	RoomID        string `json:"room_id,omitempty"`
	DeviceID      int    `json:"device_id,omitempty"`
	DeviceNumber  int    `json:"device_number,omitempty"`
	AdapterNumber int    `json:"adapter_number"`
	IsBroken      int    `json:"is_broken,omitempty"`
	IsBlocked     bool   `json:"isBlocked"`
}

type FixtureRoomUser struct {
	UserID   string `json:"user_id"`
	Priority int    `json:"priority"`
	HozNum   string `json:"hoz_num"`
}

type FixtureTranslatorRow struct {
	TypeProtocol   string `json:"type_protocol"`
	Code           int    `json:"code"`
	TypeEvent      string `json:"type_event"`
	AdditionalType int    `json:"additional_type"`
	EventByUser    string `json:"event_by_user"`
	IsAlarm        int    `json:"is_alarm"`
}

type FixtureTapeRow struct {
	Time        int64  `json:"time"`
	UserID      string `json:"user_id"`
	ObjID       int    `json:"obj_id"`
	DeviceID    int    `json:"device_id"`
	AlarmType   string `json:"alarm_type"`
	MgrID       any    `json:"mgr_id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	PultID      string `json:"pult_id"`
	Description string `json:"description"`
	ReasonAlarm string `json:"reasonAlarm"`
	LastAct     string `json:"last_act"`
}

type FixtureEvent struct {
	ObjID          int    `json:"obj_id,omitempty"`
	DeviceID       int    `json:"device_id,omitempty"`
	PPKNum         int    `json:"ppk_num,omitempty"`
	Time           int64  `json:"time"`
	Code           int    `json:"code,omitempty"`
	Type           string `json:"type,omitempty"`
	TypeEvent      string `json:"type_event,omitempty"`
	AdditionalType int    `json:"additional_type,omitempty"`
	Msg            string `json:"msg,omitempty"`
	DictName       string `json:"dict_name,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	MgrID          string `json:"mgr_id,omitempty"`
	Cause          string `json:"cause,omitempty"`
	Note           string `json:"note,omitempty"`
	Line           int    `json:"line,omitempty"`
	LineNumber     int    `json:"line_number,omitempty"`
	Number         any    `json:"number"`
	HozUserID      any    `json:"hoz_user_id"`
	ContactID      any    `json:"contact_id"`
}

type FixtureDisconnectedDevice struct {
	ObjID        int   `json:"obj_id"`
	DeviceID     int   `json:"device_id"`
	Number       int   `json:"number"`
	Offline      int64 `json:"offline"`
	Disconnected bool  `json:"disconnected"`
}

func DefaultFixture() Fixture {
	return buildFixtureFromUnified(defaultUnifiedFixture())
}
