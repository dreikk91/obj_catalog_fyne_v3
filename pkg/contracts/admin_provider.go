package contracts

// AdminProvider визначає доступний у UI адмінський функціонал.
type AdminProvider interface {
	AdminObjectDialogProvider

	AddObjectType(name string) error
	UpdateObjectType(id int64, name string) error
	DeleteObjectType(id int64) error

	ListRegions() ([]DictionaryItem, error)
	AddRegion(name string, regionCode *int64) error
	UpdateRegion(id int64, name string, regionCode *int64) error
	DeleteRegion(id int64) error

	ListAlarmReasons() ([]DictionaryItem, error)
	AddAlarmReason(name string) error
	UpdateAlarmReason(id int64, name string) error
	DeleteAlarmReason(id int64) error
	MoveAlarmReason(id int64, direction int) error

	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]AdminMessage, error)
	SetMessageAdminOnly(uin int64, adminOnly bool) error
	SetMessageCategory(uin int64, sc1 *int64) error
	List220VMessageBuckets(protocolIDs []int64, filter string) (Admin220VMessageBuckets, error)
	SetMessage220VMode(uin int64, mode Admin220VMode) error

	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode DisplayBlockMode) error

	GetFireMonitoringSettings() (FireMonitoringSettings, error)
	SaveFireMonitoringSettings(settings FireMonitoringSettings) error

	AddPPKConstructor(name string, channel int64, zoneCount int64) error
	UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error
	DeletePPKConstructor(id int64) error
	ListSubServerObjects(filter string) ([]AdminSubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error

	GetAdminAccessStatus() (AdminAccessStatus, error)
	RunDataIntegrityChecks(limit int) ([]AdminDataCheckIssue, error)
	CollectObjectStatistics(filter AdminStatisticsFilter, limit int) ([]AdminStatisticsRow, error)

	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}
