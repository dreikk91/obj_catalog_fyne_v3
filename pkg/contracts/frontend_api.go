package contracts

import (
	"context"
	"errors"
	"obj_catalog_fyne_v3/pkg/ids"
	"time"
)

var (
	ErrFrontendBackendUnavailable = errors.New("frontend backend is unavailable")
	ErrUnsupportedFrontendSource  = errors.New("frontend source is not supported")
	ErrMissingLegacyObjectPayload = errors.New("legacy object payload is required")
	ErrMissingCASLObjectPayload   = errors.New("casl object payload is required")
)

type FrontendSource string

const (
	FrontendSourceUnknown FrontendSource = "unknown"
	FrontendSourceBridge  FrontendSource = "bridge"
	FrontendSourcePhoenix FrontendSource = "phoenix"
	FrontendSourceCASL    FrontendSource = "casl"
)

func (s FrontendSource) DisplayName() string {
	switch s {
	case FrontendSourceBridge:
		return "МІСТ/Firebird"
	case FrontendSourcePhoenix:
		return "Phoenix"
	case FrontendSourceCASL:
		return "CASL Cloud"
	default:
		return "Невідоме джерело"
	}
}

func DetectFrontendSourceByObjectID(objectID int) FrontendSource {
	switch {
	case ids.IsCASLObjectID(objectID):
		return FrontendSourceCASL
	case ids.IsPhoenixObjectID(objectID):
		return FrontendSourcePhoenix
	case objectID > 0:
		return FrontendSourceBridge
	default:
		return FrontendSourceUnknown
	}
}

type FrontendSourceCapability struct {
	Source            FrontendSource
	DisplayName       string
	ReadObjects       bool
	ReadObjectDetails bool
	ReadEvents        bool
	ReadAlarms        bool
	CreateObject      bool
	UpdateObject      bool
}

type FrontendConnectionStatus string

const (
	FrontendConnectionStatusUnknown FrontendConnectionStatus = "unknown"
	FrontendConnectionStatusOnline  FrontendConnectionStatus = "online"
	FrontendConnectionStatusOffline FrontendConnectionStatus = "offline"
)

type FrontendGuardStatus string

const (
	FrontendGuardStatusUnknown  FrontendGuardStatus = "unknown"
	FrontendGuardStatusGuarded  FrontendGuardStatus = "guarded"
	FrontendGuardStatusDisarmed FrontendGuardStatus = "disarmed"
)

type FrontendMonitoringStatus string

const (
	FrontendMonitoringStatusUnknown FrontendMonitoringStatus = "unknown"
	FrontendMonitoringStatusActive  FrontendMonitoringStatus = "active"
	FrontendMonitoringStatusBlocked FrontendMonitoringStatus = "blocked"
	FrontendMonitoringStatusDebug   FrontendMonitoringStatus = "debug"
)

type FrontendVisualSeverity string

const (
	FrontendVisualSeverityUnknown  FrontendVisualSeverity = "unknown"
	FrontendVisualSeverityNormal   FrontendVisualSeverity = "normal"
	FrontendVisualSeverityInfo     FrontendVisualSeverity = "info"
	FrontendVisualSeverityWarning  FrontendVisualSeverity = "warning"
	FrontendVisualSeverityCritical FrontendVisualSeverity = "critical"
)

type FrontendCapabilities struct {
	Sources []FrontendSourceCapability
}

type FrontendObjectSummary struct {
	ID               int
	Source           FrontendSource
	NativeID         string
	DisplayNumber    string
	Name             string
	Address          string
	ContractNumber   string
	Phone            string
	StatusCode       string
	StatusText       string
	DeviceType       string
	PanelMark        string
	SignalStrength   string
	SIM1             string
	SIM2             string
	LastTestTime     time.Time
	LastMessageTime  time.Time
	GuardStatus      FrontendGuardStatus
	ConnectionStatus FrontendConnectionStatus
	MonitoringStatus FrontendMonitoringStatus
	HasAssignment    bool
}

type FrontendZone struct {
	Number         int
	Name           string
	SensorType     string
	Status         string
	GroupID        string
	GroupNumber    int
	GroupName      string
	GroupStateText string
}

type FrontendContact struct {
	Name           string
	Position       string
	Phone          string
	Priority       int
	CodeWord       string
	GroupID        string
	GroupNumber    int
	GroupName      string
	GroupStateText string
}

type FrontendAlarmItem struct {
	ID             int
	Source         FrontendSource
	ObjectID       int
	ObjectNativeID string
	ObjectNumber   string
	ObjectName     string
	Address        string
	Time           time.Time
	Details        string
	TypeCode       string
	TypeText       string
	ZoneNumber     int
	ZoneName       string
	IsProcessed    bool
	ProcessedBy    string
	ProcessNote    string
	VisualSeverity FrontendVisualSeverity
}

type FrontendEventItem struct {
	ID             int
	Source         FrontendSource
	ObjectID       int
	ObjectNativeID string
	ObjectNumber   string
	ObjectName     string
	Time           time.Time
	TypeCode       string
	TypeText       string
	ZoneNumber     int
	Details        string
	UserName       string
	VisualSeverity FrontendVisualSeverity
}

type FrontendObjectDetails struct {
	Summary             FrontendObjectSummary
	GSMLevel            int
	PowerSource         string
	AutoTestHours       int
	SubServerA          string
	SubServerB          string
	ChannelCode         int
	AKBState            int64
	PowerFault          int64
	TestControl         bool
	TestIntervalMin     int64
	Phones              string
	Notes               string
	Location            string
	LaunchDate          string
	ExternalSignal      string
	ExternalTestMessage string
	ExternalLastTest    time.Time
	ExternalLastMessage time.Time
	Zones               []FrontendZone
	Contacts            []FrontendContact
	Events              []FrontendEventItem
}

type FrontendObjectCoreFields struct {
	Name        string
	Address     string
	Contract    string
	Description string
	Notes       string
	Latitude    string
	Longitude   string
}

type FrontendLegacyObjectPayload struct {
	ObjUIN             int64
	ObjN               int64
	GrpN               int64
	ObjTypeID          int64
	ObjRegID           int64
	ChannelCode        int64
	PPKID              int64
	GSMHiddenN         int64
	TestIntervalMin    int64
	ShortName          string
	FullName           string
	Phones             string
	StartDate          string
	Location           string
	GSMPhone1          string
	GSMPhone2          string
	SubServerA         string
	SubServerB         string
	TestControlEnabled bool
}

type FrontendCASLObjectPayload struct {
	ObjID          string
	ManagerID      string
	Status         string
	ObjectType     string
	IDRequest      string
	ReactingPultID string
	StartDate      int64
	GeoZoneID      int64
	BusinessCoeff  *float64
}

type FrontendObjectUpsertRequest struct {
	Source   FrontendSource
	ObjectID int
	Core     FrontendObjectCoreFields
	Legacy   *FrontendLegacyObjectPayload
	CASL     *FrontendCASLObjectPayload
}

type FrontendObjectMutationResult struct {
	Source   FrontendSource
	ObjectID int
	NativeID string
}

type FrontendBackend interface {
	Capabilities(ctx context.Context) (FrontendCapabilities, error)
	ListObjects(ctx context.Context) ([]FrontendObjectSummary, error)
	ListAlarms(ctx context.Context) ([]FrontendAlarmItem, error)
	ListEvents(ctx context.Context) ([]FrontendEventItem, error)
	GetObjectDetails(ctx context.Context, objectID int) (FrontendObjectDetails, error)
	CreateObject(ctx context.Context, request FrontendObjectUpsertRequest) (FrontendObjectMutationResult, error)
	UpdateObject(ctx context.Context, request FrontendObjectUpsertRequest) (FrontendObjectMutationResult, error)
}
