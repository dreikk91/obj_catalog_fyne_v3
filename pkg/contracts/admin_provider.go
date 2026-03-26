package contracts

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
