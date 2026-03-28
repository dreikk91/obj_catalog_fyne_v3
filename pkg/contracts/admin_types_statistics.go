package contracts

type AdminStatisticsConnectionMode int16

const (
	StatsConnectionAll AdminStatisticsConnectionMode = iota
	StatsConnectionOnline
	StatsConnectionOffline
)

type AdminStatisticsProtocolFilter string

const (
	StatsProtocolAll      AdminStatisticsProtocolFilter = ""
	StatsProtocolAutodial AdminStatisticsProtocolFilter = "autodial"
	StatsProtocolMost     AdminStatisticsProtocolFilter = "most"
	StatsProtocolNova     AdminStatisticsProtocolFilter = "nova"
)

type AdminStatisticsFilter struct {
	ConnectionMode AdminStatisticsConnectionMode
	ProtocolFilter AdminStatisticsProtocolFilter
	ChannelCode    *int64
	GuardState     *int64
	ObjTypeID      *int64
	RegionID       *int64
	BlockMode      *DisplayBlockMode
	Search         string
}

type AdminStatisticsRow struct {
	ObjUIN         int64
	ObjN           int64
	GrpN           int64
	ShortName      string
	FullName       string
	Address        string
	Phones         string
	Contract       string
	StartDate      string
	Location       string
	Notes          string
	ChannelCode    int64
	PPKID          int64
	PPKName        string
	GSMPhone1      string
	GSMPhone2      string
	GSMHiddenN     int64
	SubServerA     string
	SubServerB     string
	TestControl    int64
	TestTime       int64
	GuardState     int64
	IsConnState    int64
	AlarmState     int64
	TechAlarmState int64
	ObjTypeID      int64
	ObjTypeName    string
	RegionID       int64
	RegionName     string
	BlockMode      DisplayBlockMode
}
