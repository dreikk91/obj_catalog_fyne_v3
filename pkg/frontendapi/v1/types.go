package v1

import "time"

type Source string

const (
	SourceUnknown Source = "unknown"
	SourceBridge  Source = "bridge"
	SourcePhoenix Source = "phoenix"
	SourceCASL    Source = "casl"
)

type ConnectionStatus string

const (
	ConnectionStatusUnknown ConnectionStatus = "unknown"
	ConnectionStatusOnline  ConnectionStatus = "online"
	ConnectionStatusOffline ConnectionStatus = "offline"
)

type GuardStatus string

const (
	GuardStatusUnknown  GuardStatus = "unknown"
	GuardStatusGuarded  GuardStatus = "guarded"
	GuardStatusDisarmed GuardStatus = "disarmed"
)

type MonitoringStatus string

const (
	MonitoringStatusUnknown MonitoringStatus = "unknown"
	MonitoringStatusActive  MonitoringStatus = "active"
	MonitoringStatusBlocked MonitoringStatus = "blocked"
	MonitoringStatusDebug   MonitoringStatus = "debug"
)

type VisualSeverity string

const (
	VisualSeverityUnknown  VisualSeverity = "unknown"
	VisualSeverityNormal   VisualSeverity = "normal"
	VisualSeverityInfo     VisualSeverity = "info"
	VisualSeverityWarning  VisualSeverity = "warning"
	VisualSeverityCritical VisualSeverity = "critical"
)

type SourceCapability struct {
	Source            Source `json:"Source"`
	DisplayName       string `json:"DisplayName"`
	ReadObjects       bool   `json:"ReadObjects"`
	ReadObjectDetails bool   `json:"ReadObjectDetails"`
	ReadEvents        bool   `json:"ReadEvents"`
	ReadAlarms        bool   `json:"ReadAlarms"`
	CreateObject      bool   `json:"CreateObject"`
	UpdateObject      bool   `json:"UpdateObject"`
}

type Capabilities struct {
	Sources []SourceCapability `json:"Sources"`
}

type ObjectSummary struct {
	ID               int              `json:"ID"`
	Source           Source           `json:"Source"`
	NativeID         string           `json:"NativeID"`
	DisplayNumber    string           `json:"DisplayNumber"`
	Name             string           `json:"Name"`
	Address          string           `json:"Address"`
	ContractNumber   string           `json:"ContractNumber"`
	Phone            string           `json:"Phone"`
	StatusCode       string           `json:"StatusCode"`
	StatusText       string           `json:"StatusText"`
	DeviceType       string           `json:"DeviceType"`
	PanelMark        string           `json:"PanelMark"`
	SignalStrength   string           `json:"SignalStrength"`
	SIM1             string           `json:"SIM1"`
	SIM2             string           `json:"SIM2"`
	LastTestTime     time.Time        `json:"LastTestTime"`
	LastMessageTime  time.Time        `json:"LastMessageTime"`
	GuardStatus      GuardStatus      `json:"GuardStatus"`
	ConnectionStatus ConnectionStatus `json:"ConnectionStatus"`
	MonitoringStatus MonitoringStatus `json:"MonitoringStatus"`
	HasAssignment    bool             `json:"HasAssignment"`
}

type Zone struct {
	Number         int    `json:"Number"`
	Name           string `json:"Name"`
	SensorType     string `json:"SensorType"`
	Status         string `json:"Status"`
	GroupID        string `json:"GroupID"`
	GroupNumber    int    `json:"GroupNumber"`
	GroupName      string `json:"GroupName"`
	GroupStateText string `json:"GroupStateText"`
}

type Contact struct {
	Name           string `json:"Name"`
	Position       string `json:"Position"`
	Phone          string `json:"Phone"`
	Priority       int    `json:"Priority"`
	CodeWord       string `json:"CodeWord"`
	GroupID        string `json:"GroupID"`
	GroupNumber    int    `json:"GroupNumber"`
	GroupName      string `json:"GroupName"`
	GroupStateText string `json:"GroupStateText"`
}

type AlarmItem struct {
	ID             int            `json:"ID"`
	Source         Source         `json:"Source"`
	ObjectID       int            `json:"ObjectID"`
	ObjectNativeID string         `json:"ObjectNativeID"`
	ObjectNumber   string         `json:"ObjectNumber"`
	ObjectName     string         `json:"ObjectName"`
	Address        string         `json:"Address"`
	Time           time.Time      `json:"Time"`
	Details        string         `json:"Details"`
	TypeCode       string         `json:"TypeCode"`
	TypeText       string         `json:"TypeText"`
	ZoneNumber     int            `json:"ZoneNumber"`
	ZoneName       string         `json:"ZoneName"`
	IsProcessed    bool           `json:"IsProcessed"`
	ProcessedBy    string         `json:"ProcessedBy"`
	ProcessNote    string         `json:"ProcessNote"`
	VisualSeverity VisualSeverity `json:"VisualSeverity"`
}

type EventItem struct {
	ID             int            `json:"ID"`
	Source         Source         `json:"Source"`
	ObjectID       int            `json:"ObjectID"`
	ObjectNativeID string         `json:"ObjectNativeID"`
	ObjectNumber   string         `json:"ObjectNumber"`
	ObjectName     string         `json:"ObjectName"`
	Time           time.Time      `json:"Time"`
	TypeCode       string         `json:"TypeCode"`
	TypeText       string         `json:"TypeText"`
	ZoneNumber     int            `json:"ZoneNumber"`
	Details        string         `json:"Details"`
	UserName       string         `json:"UserName"`
	VisualSeverity VisualSeverity `json:"VisualSeverity"`
}

type ObjectDetails struct {
	Summary             ObjectSummary `json:"Summary"`
	GSMLevel            int           `json:"GSMLevel"`
	PowerSource         string        `json:"PowerSource"`
	AutoTestHours       int           `json:"AutoTestHours"`
	SubServerA          string        `json:"SubServerA"`
	SubServerB          string        `json:"SubServerB"`
	ChannelCode         int           `json:"ChannelCode"`
	AKBState            int64         `json:"AKBState"`
	PowerFault          int64         `json:"PowerFault"`
	TestControl         bool          `json:"TestControl"`
	TestIntervalMin     int64         `json:"TestIntervalMin"`
	Phones              string        `json:"Phones"`
	Notes               string        `json:"Notes"`
	Location            string        `json:"Location"`
	LaunchDate          string        `json:"LaunchDate"`
	ExternalSignal      string        `json:"ExternalSignal"`
	ExternalTestMessage string        `json:"ExternalTestMessage"`
	ExternalLastTest    time.Time     `json:"ExternalLastTest"`
	ExternalLastMessage time.Time     `json:"ExternalLastMessage"`
	Zones               []Zone        `json:"Zones"`
	Contacts            []Contact     `json:"Contacts"`
	Events              []EventItem   `json:"Events"`
}

type ObjectCoreFields struct {
	Name        string `json:"Name"`
	Address     string `json:"Address"`
	Contract    string `json:"Contract"`
	Description string `json:"Description"`
	Notes       string `json:"Notes"`
	Latitude    string `json:"Latitude"`
	Longitude   string `json:"Longitude"`
}

type LegacyObjectPayload struct {
	ObjUIN             int64  `json:"ObjUIN"`
	ObjN               int64  `json:"ObjN"`
	GrpN               int64  `json:"GrpN"`
	ObjTypeID          int64  `json:"ObjTypeID"`
	ObjRegID           int64  `json:"ObjRegID"`
	ChannelCode        int64  `json:"ChannelCode"`
	PPKID              int64  `json:"PPKID"`
	GSMHiddenN         int64  `json:"GSMHiddenN"`
	TestIntervalMin    int64  `json:"TestIntervalMin"`
	ShortName          string `json:"ShortName"`
	FullName           string `json:"FullName"`
	Phones             string `json:"Phones"`
	StartDate          string `json:"StartDate"`
	Location           string `json:"Location"`
	GSMPhone1          string `json:"GSMPhone1"`
	GSMPhone2          string `json:"GSMPhone2"`
	SubServerA         string `json:"SubServerA"`
	SubServerB         string `json:"SubServerB"`
	TestControlEnabled bool   `json:"TestControlEnabled"`
}

type CASLObjectPayload struct {
	ObjID          string   `json:"ObjID"`
	ManagerID      string   `json:"ManagerID"`
	Status         string   `json:"Status"`
	ObjectType     string   `json:"ObjectType"`
	IDRequest      string   `json:"IDRequest"`
	ReactingPultID string   `json:"ReactingPultID"`
	StartDate      int64    `json:"StartDate"`
	GeoZoneID      int64    `json:"GeoZoneID"`
	BusinessCoeff  *float64 `json:"BusinessCoeff"`
}

type ObjectUpsertRequest struct {
	Source   Source               `json:"Source"`
	ObjectID int                  `json:"ObjectID"`
	Core     ObjectCoreFields     `json:"Core"`
	Legacy   *LegacyObjectPayload `json:"Legacy"`
	CASL     *CASLObjectPayload   `json:"CASL"`
}

type ObjectMutationResult struct {
	Source   Source `json:"Source"`
	ObjectID int    `json:"ObjectID"`
	NativeID string `json:"NativeID"`
}

type ObjectListResponse struct {
	Items []ObjectSummary `json:"items"`
}

type AlarmListResponse struct {
	Items []AlarmItem `json:"items"`
}

type EventListResponse struct {
	Items []EventItem `json:"items"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
