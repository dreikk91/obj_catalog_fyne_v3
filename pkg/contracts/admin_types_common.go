package contracts

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

// Admin220VMode - режим використання повідомлення для контролю 220В.
type Admin220VMode int16

const (
	Admin220VNone    Admin220VMode = 0
	Admin220VAlarm   Admin220VMode = 1
	Admin220VRestore Admin220VMode = 2
)

// Admin220VMessageBuckets - групування повідомлень для конструктора 220В.
type Admin220VMessageBuckets struct {
	Free    []AdminMessage
	Alarm   []AdminMessage
	Restore []AdminMessage
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
