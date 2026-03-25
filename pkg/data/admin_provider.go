package data

// DisplayBlockMode визначає режим блокування відображення інформації по об'єкту.
type DisplayBlockMode int16

const (
	DisplayBlockNone DisplayBlockMode = iota
	DisplayBlockTemporaryOff
	DisplayBlockDebug
)

// FireMonitoringServer - опис сервера пожежного моніторингу.
type FireMonitoringServer struct {
	Host    string
	Port    int64
	Info    string
	Enabled bool
}

// FireMonitoringSettings - налаштування розділу "Пожежний моніторинг".
type FireMonitoringSettings struct {
	Enabled       bool
	ObjectID      string
	AckWaitSec    int64
	UseStdDateFmt bool
	Servers       []FireMonitoringServer
}

// PPKConstructorItem - запис довідника "Конструктор ППК".
type PPKConstructorItem struct {
	ID        int64
	Name      string
	Channel   int64
	ZoneCount int64
}

// AdminObjectCard - дані картки об'єкта (вкладка "Об'єкт").
type AdminObjectCard struct {
	ObjUIN int64
	ObjN   int64
	GrpN   int64

	ShortName string
	FullName  string
	ObjTypeID int64
	ObjRegID  int64

	Address   string
	Phones    string
	Contract  string
	StartDate string
	Location  string
	Notes     string

	ChannelCode int64
	PPKID       int64 // ID з довідника PPK (без +100)
	GSMPhone1   string
	GSMPhone2   string
	GSMHiddenN  int64
	SubServerA  string
	SubServerB  string

	TestControlEnabled bool
	TestIntervalMin    int64
}

// AdminSubServer - запис підсервера (SBS).
type AdminSubServer struct {
	ID    int64
	Info  string
	Bind  string
	Host  string
	Type  int64
	Host2 string
}

// AdminSubServerObject - об'єкт з прив'язкою до підсервера (SBSA/SBSB).
type AdminSubServerObject struct {
	ObjN       int64
	Name       string
	Address    string
	SubServerA string
	SubServerB string
}

// AdminObjectPersonal - контакт/відповідальна особа по об'єкту (вкладка "В/О").
type AdminObjectPersonal struct {
	ID          int64
	SourceObjN  int64
	Number      int64
	Surname     string
	Name        string
	SecName     string
	Address     string
	Phones      string
	Position    string
	Notes       string
	IsTRKTester bool
	Access1     int64
	IsRang      bool
	ViberID     string
	TelegramID  string
	CreatedAt   string
}

// AdminObjectZone - запис шлейфу/зони об'єкта (вкладка "Зони").
type AdminObjectZone struct {
	ID            int64
	ZoneNumber    int64
	ZoneType      int64
	Description   string
	EntryDelaySec int64
}

// AdminObjectCoordinates - координати об'єкта (вкладка "Додатково").
type AdminObjectCoordinates struct {
	Latitude  string
	Longitude string
}

// AdminSIMPhoneUsage - використання SIM-номера в картках об'єктів.
type AdminSIMPhoneUsage struct {
	ObjN int64
	Name string
	Slot string
}

// DictionaryItem - універсальний елемент довідника (назва + опційний код).
type DictionaryItem struct {
	ID    int64
	Name  string
	Code  *int64
	Extra string
}

// AdminMessage - запис з довідника повідомлень MESSLIST.
type AdminMessage struct {
	UIN          int64
	ProtocolID   *int64
	MessageID    *int64
	MessageHex   string
	Text         string
	SC1          *int64
	ForAdminOnly bool
}

// DisplayBlockObject - об'єкт зі станом блокування відображення.
type DisplayBlockObject struct {
	ObjN           int64
	Name           string
	BlockMode      DisplayBlockMode
	AlarmState     int64
	GuardState     int64
	TechAlarmState int64
	IsConnState    int64
}

// AdminAccessStatus - стан доступу поточного користувача до адмін-функцій.
type AdminAccessStatus struct {
	CurrentUser      string
	MatchedPersonal  string
	HasFullAccess    bool
	AdminUsersCount  int64
	MatchDescription string
}

// AdminDataCheckIssue - проблема, виявлена read-only перевіркою цілісності БД.
type AdminDataCheckIssue struct {
	Severity string
	Code     string
	ObjN     int64
	Details  string
}

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

// AdminProvider визначає доступний у UI адмінський функціонал.
type AdminProvider interface {
	ListObjectTypes() ([]DictionaryItem, error)
	AddObjectType(name string) error
	UpdateObjectType(id int64, name string) error
	DeleteObjectType(id int64) error

	ListRegions() ([]DictionaryItem, error)
	AddRegion(name string, regionCode *int64) error
	UpdateRegion(id int64, name string, regionCode *int64) error
	DeleteRegion(id int64) error
	ListObjectDistricts() ([]DictionaryItem, error)

	ListAlarmReasons() ([]DictionaryItem, error)
	AddAlarmReason(name string) error
	UpdateAlarmReason(id int64, name string) error
	DeleteAlarmReason(id int64) error
	MoveAlarmReason(id int64, direction int) error

	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]AdminMessage, error)
	SetMessageAdminOnly(uin int64, adminOnly bool) error
	SetMessageCategory(uin int64, sc1 *int64) error

	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode DisplayBlockMode) error

	GetFireMonitoringSettings() (FireMonitoringSettings, error)
	SaveFireMonitoringSettings(settings FireMonitoringSettings) error

	ListPPKConstructor() ([]PPKConstructorItem, error)
	AddPPKConstructor(name string, channel int64, zoneCount int64) error
	UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error
	DeletePPKConstructor(id int64) error
	ListSubServers() ([]AdminSubServer, error)
	ListSubServerObjects(filter string) ([]AdminSubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error

	GetObjectCard(objn int64) (AdminObjectCard, error)
	CreateObject(card AdminObjectCard) error
	UpdateObject(card AdminObjectCard) error
	DeleteObject(objn int64) error

	ListObjectPersonals(objn int64) ([]AdminObjectPersonal, error)
	AddObjectPersonal(objn int64, item AdminObjectPersonal) error
	UpdateObjectPersonal(objn int64, item AdminObjectPersonal) error
	DeleteObjectPersonal(objn int64, personalID int64) error
	FindPersonalByPhone(phone string) (*AdminObjectPersonal, error)

	ListObjectZones(objn int64) ([]AdminObjectZone, error)
	AddObjectZone(objn int64, zone AdminObjectZone) error
	UpdateObjectZone(objn int64, zone AdminObjectZone) error
	DeleteObjectZone(objn int64, zoneID int64) error
	FillObjectZones(objn int64, count int64) error
	ClearObjectZones(objn int64) error

	GetObjectCoordinates(objn int64) (AdminObjectCoordinates, error)
	SaveObjectCoordinates(objn int64, coords AdminObjectCoordinates) error
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]AdminSIMPhoneUsage, error)
	GetAdminAccessStatus() (AdminAccessStatus, error)
	RunDataIntegrityChecks(limit int) ([]AdminDataCheckIssue, error)
	CollectObjectStatistics(filter AdminStatisticsFilter, limit int) ([]AdminStatisticsRow, error)

	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}
