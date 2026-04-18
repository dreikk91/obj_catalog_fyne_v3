package v1

import frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"

type DisplayBlockMode string

const (
	DisplayBlockModeNone         DisplayBlockMode = "none"
	DisplayBlockModeTemporaryOff DisplayBlockMode = "temporary_off"
	DisplayBlockModeDebug        DisplayBlockMode = "debug"
)

type StatisticsConnectionMode string

const (
	StatisticsConnectionModeAll     StatisticsConnectionMode = "all"
	StatisticsConnectionModeOnline  StatisticsConnectionMode = "online"
	StatisticsConnectionModeOffline StatisticsConnectionMode = "offline"
)

type StatisticsProtocolFilter string

const (
	StatisticsProtocolAll      StatisticsProtocolFilter = ""
	StatisticsProtocolAutodial StatisticsProtocolFilter = "autodial"
	StatisticsProtocolMost     StatisticsProtocolFilter = "most"
	StatisticsProtocolNova     StatisticsProtocolFilter = "nova"
)

type DictionaryItem struct {
	ID    int64
	Name  string
	Code  *int64
	Extra string
}

type StatisticsFilter struct {
	ConnectionMode StatisticsConnectionMode
	ProtocolFilter StatisticsProtocolFilter
	ChannelCode    *int64
	GuardState     *int64
	ObjTypeID      *int64
	RegionID       *int64
	BlockMode      *DisplayBlockMode
	Search         string
}

type StatisticsRow struct {
	ObjUIN           int64
	ObjN             int64
	GrpN             int64
	ShortName        string
	FullName         string
	Address          string
	Phones           string
	Contract         string
	StartDate        string
	Location         string
	Notes            string
	ChannelCode      int64
	PPKID            int64
	PPKName          string
	GSMPhone1        string
	GSMPhone2        string
	GSMHiddenN       int64
	SubServerA       string
	SubServerB       string
	TestControl      int64
	TestTime         int64
	GuardState       int64
	IsConnState      int64
	AlarmState       int64
	TechAlarmState   int64
	ObjTypeID        int64
	ObjTypeName      string
	RegionID         int64
	RegionName       string
	BlockMode        DisplayBlockMode
	GuardStatus      frontendv1.GuardStatus
	ConnectionStatus frontendv1.ConnectionStatus
	MonitoringStatus frontendv1.MonitoringStatus
	VisualSeverity   frontendv1.VisualSeverity
}

type DisplayBlockObject struct {
	ObjN             int64
	Name             string
	BlockMode        DisplayBlockMode
	AlarmState       int64
	GuardState       int64
	TechAlarmState   int64
	IsConnState      int64
	GuardStatus      frontendv1.GuardStatus
	ConnectionStatus frontendv1.ConnectionStatus
	MonitoringStatus frontendv1.MonitoringStatus
	VisualSeverity   frontendv1.VisualSeverity
}

type Message struct {
	UIN          int64
	ProtocolID   *int64
	MessageID    *int64
	MessageHex   string
	Text         string
	SC1          *int64
	ForAdminOnly bool
}

type Message220VMode string

const (
	Message220VModeNone    Message220VMode = "none"
	Message220VModeAlarm   Message220VMode = "alarm"
	Message220VModeRestore Message220VMode = "restore"
)

type Message220VBuckets struct {
	Free    []Message
	Alarm   []Message
	Restore []Message
}
