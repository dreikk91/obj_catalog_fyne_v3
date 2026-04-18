package v1

type StatisticsProvider interface {
	CollectObjectStatistics(filter StatisticsFilter, limit int) ([]StatisticsRow, error)
	ListObjectTypes() ([]DictionaryItem, error)
	ListObjectDistricts() ([]DictionaryItem, error)
}

type DisplayBlockingProvider interface {
	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode DisplayBlockMode) error
}

type DisplayBlockObjectLookupProvider interface {
	ListDisplayBlockObjects(filter string) ([]DisplayBlockObject, error)
}

type MessageLookupProvider interface {
	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]Message, error)
}

type MessagesProvider interface {
	MessageLookupProvider
	SetMessageAdminOnly(uin int64, adminOnly bool) error
}

type Message220VProvider interface {
	List220VMessageBuckets(protocolIDs []int64, filter string) (Message220VBuckets, error)
	SetMessage220VMode(uin int64, mode Message220VMode) error
}

type EventOverrideProvider interface {
	MessagesProvider
	Message220VProvider
	SetMessageCategory(uin int64, sc1 *int64) error
}

type EventEmulationProvider interface {
	DisplayBlockObjectLookupProvider
	MessageLookupProvider
	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}

type SystemControlProvider interface {
	GetAdminAccessStatus() (AccessStatus, error)
	RunDataIntegrityChecks(limit int) ([]DataCheckIssue, error)
}

type SubServerObjectsProvider interface {
	ListSubServers() ([]SubServer, error)
	ListSubServerObjects(filter string) ([]SubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error
}

type ObjectTypesDictionaryProvider interface {
	ListObjectTypes() ([]DictionaryItem, error)
	AddObjectType(name string) error
	UpdateObjectType(id int64, name string) error
	DeleteObjectType(id int64) error
}

type RegionsDictionaryProvider interface {
	ListRegions() ([]DictionaryItem, error)
	AddRegion(name string, regionCode *int64) error
	UpdateRegion(id int64, name string, regionCode *int64) error
	DeleteRegion(id int64) error
}

type AlarmReasonsDictionaryProvider interface {
	ListAlarmReasons() ([]DictionaryItem, error)
	AddAlarmReason(name string) error
	UpdateAlarmReason(id int64, name string) error
	DeleteAlarmReason(id int64) error
	MoveAlarmReason(id int64, direction int) error
}

type PPKConstructorProvider interface {
	AddPPKConstructor(name string, channel int64, zoneCount int64) error
	UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error
	DeletePPKConstructor(id int64) error
	ListPPKConstructor() ([]PPKConstructorItem, error)
}

type FireMonitoringProvider interface {
	GetFireMonitoringSettings() (FireMonitoringSettings, error)
	SaveFireMonitoringSettings(settings FireMonitoringSettings) error
}

type ObjectReferenceProvider interface {
	ListObjectTypes() ([]DictionaryItem, error)
	ListObjectDistricts() ([]DictionaryItem, error)
	ListPPKConstructor() ([]PPKConstructorItem, error)
	ListSubServers() ([]SubServer, error)
}

type ObjectCardService interface {
	GetObjectCard(objn int64) (ObjectCard, error)
	CreateObject(card ObjectCard) error
	UpdateObject(card ObjectCard) error
	DeleteObject(objn int64) error
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]SIMPhoneUsage, error)
}

type ObjectCreateProvider interface {
	CreateObject(card ObjectCard) error
}

type ObjectDeleteProvider interface {
	DeleteObject(objn int64) error
}

type ObjectSIMLookupProvider interface {
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]SIMPhoneUsage, error)
}

type ObjectPersonalService interface {
	ListObjectPersonals(objn int64) ([]ObjectPersonal, error)
	AddObjectPersonal(objn int64, item ObjectPersonal) error
	UpdateObjectPersonal(objn int64, item ObjectPersonal) error
	DeleteObjectPersonal(objn int64, personalID int64) error
	FindPersonalByPhone(phone string) (*ObjectPersonal, error)
}

type ObjectZoneService interface {
	ListObjectZones(objn int64) ([]ObjectZone, error)
	AddObjectZone(objn int64, zone ObjectZone) error
	UpdateObjectZone(objn int64, zone ObjectZone) error
	DeleteObjectZone(objn int64, zoneID int64) error
	FillObjectZones(objn int64, count int64) error
	ClearObjectZones(objn int64) error
}

type ObjectCoordinatesService interface {
	GetObjectCoordinates(objn int64) (ObjectCoordinates, error)
	SaveObjectCoordinates(objn int64, coords ObjectCoordinates) error
}

type ObjectCoordinatesSaveProvider interface {
	SaveObjectCoordinates(objn int64, coords ObjectCoordinates) error
}

type ObjectZoneAddProvider interface {
	AddObjectZone(objn int64, zone ObjectZone) error
}

type ObjectWizardProvider interface {
	ObjectReferenceProvider
	ObjectSIMLookupProvider
	ObjectCreateProvider
	ObjectPersonalService
	ObjectZoneAddProvider
	ObjectCoordinatesSaveProvider
}

type ObjectPersonalTabProvider interface {
	ObjectPersonalService
}

type ObjectZonesTabProvider interface {
	ObjectZoneService
	GetObjectCard(objn int64) (ObjectCard, error)
	ListPPKConstructor() ([]PPKConstructorItem, error)
}

type ObjectAdditionalTabProvider interface {
	ObjectCoordinatesService
	ListObjectDistricts() ([]DictionaryItem, error)
}

type ObjectCardProvider interface {
	ObjectReferenceProvider
	ObjectCardService
	ObjectPersonalTabProvider
	ObjectZonesTabProvider
	ObjectAdditionalTabProvider
}
