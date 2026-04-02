package contracts

// AdminObjectReferenceService надає довідники для картки об'єкта.
type AdminObjectReferenceService interface {
	ListObjectTypes() ([]DictionaryItem, error)
	ListObjectDistricts() ([]DictionaryItem, error)
	ListPPKConstructor() ([]PPKConstructorItem, error)
	ListSubServers() ([]AdminSubServer, error)
}

// AdminObjectCardService керує карткою об'єкта.
type AdminObjectCardService interface {
	GetObjectCard(objn int64) (AdminObjectCard, error)
	CreateObject(card AdminObjectCard) error
	UpdateObject(card AdminObjectCard) error
	DeleteObject(objn int64) error
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]AdminSIMPhoneUsage, error)
}

// AdminObjectCreateService створює об'єкт.
type AdminObjectCreateService interface {
	CreateObject(card AdminObjectCard) error
}

// AdminObjectSIMLookupService перевіряє використання SIM-номерів.
type AdminObjectSIMLookupService interface {
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]AdminSIMPhoneUsage, error)
}

// AdminObjectVodafoneService керує Vodafone IoT функціями для SIM-карток об'єкта.
type AdminObjectVodafoneService interface {
	GetVodafoneAuthState() (VodafoneAuthState, error)
	RequestVodafoneLoginSMS(phone string) error
	VerifyVodafoneLogin(phone string, code string) (VodafoneAuthState, error)
	ClearVodafoneLogin() error
	GetVodafoneSIMStatus(msisdn string) (VodafoneSIMStatus, error)
	RebootVodafoneSIM(msisdn string) (VodafoneSIMRebootResult, error)
	UpdateVodafoneSIMMetadata(msisdn string, name string, comment string) error
}

// AdminObjectPersonalService керує відповідальними особами об'єкта.
type AdminObjectPersonalService interface {
	ListObjectPersonals(objn int64) ([]AdminObjectPersonal, error)
	AddObjectPersonal(objn int64, item AdminObjectPersonal) error
	UpdateObjectPersonal(objn int64, item AdminObjectPersonal) error
	DeleteObjectPersonal(objn int64, personalID int64) error
	FindPersonalByPhone(phone string) (*AdminObjectPersonal, error)
}

// AdminObjectZoneService керує зонами об'єкта.
type AdminObjectZoneService interface {
	ListObjectZones(objn int64) ([]AdminObjectZone, error)
	AddObjectZone(objn int64, zone AdminObjectZone) error
	UpdateObjectZone(objn int64, zone AdminObjectZone) error
	DeleteObjectZone(objn int64, zoneID int64) error
	FillObjectZones(objn int64, count int64) error
	ClearObjectZones(objn int64) error
}

// AdminObjectCoordinatesService керує геокоординатами об'єкта.
type AdminObjectCoordinatesService interface {
	GetObjectCoordinates(objn int64) (AdminObjectCoordinates, error)
	SaveObjectCoordinates(objn int64, coords AdminObjectCoordinates) error
}

// AdminObjectCoordinatesSaveService зберігає геокоординати об'єкта.
type AdminObjectCoordinatesSaveService interface {
	SaveObjectCoordinates(objn int64, coords AdminObjectCoordinates) error
}

// AdminObjectZoneAddService додає зону об'єкта.
type AdminObjectZoneAddService interface {
	AddObjectZone(objn int64, zone AdminObjectZone) error
}

// AdminObjectWizardProvider - мінімальний контракт майстра створення об'єкта.
type AdminObjectWizardProvider interface {
	AdminObjectReferenceService
	AdminObjectSIMLookupService
	AdminObjectCreateService
	AdminObjectPersonalService
	AdminObjectZoneAddService
	AdminObjectCoordinatesSaveService
}

// AdminObjectCardLookupService надає читання картки об'єкта.
type AdminObjectCardLookupService interface {
	GetObjectCard(objn int64) (AdminObjectCard, error)
}

// PPKConstructorReferenceService надає довідник ППК.
type PPKConstructorReferenceService interface {
	ListPPKConstructor() ([]PPKConstructorItem, error)
}

// DistrictReferenceService надає довідник районів.
type DistrictReferenceService interface {
	ListObjectDistricts() ([]DictionaryItem, error)
}

// AdminObjectPersonalTabProvider - мінімальний контракт вкладки В/О.
type AdminObjectPersonalTabProvider interface {
	AdminObjectPersonalService
}

// AdminObjectZonesTabProvider - мінімальний контракт вкладки зон.
type AdminObjectZonesTabProvider interface {
	AdminObjectZoneService
	AdminObjectCardLookupService
	PPKConstructorReferenceService
}

// AdminObjectAdditionalTabProvider - мінімальний контракт вкладки "Додатково".
type AdminObjectAdditionalTabProvider interface {
	AdminObjectCoordinatesService
	DistrictReferenceService
}

// AdminObjectCardProvider - мінімальний контракт діалогу картки об'єкта.
type AdminObjectCardProvider interface {
	AdminObjectReferenceService
	AdminObjectCardService
	AdminObjectVodafoneService
	AdminObjectPersonalTabProvider
	AdminObjectZonesTabProvider
	AdminObjectAdditionalTabProvider
}

// AdminObjectDialogProvider - вузький контракт для UI картки/майстра об'єкта.
type AdminObjectDialogProvider interface {
	AdminObjectReferenceService
	AdminObjectCardService
	AdminObjectVodafoneService
	AdminObjectPersonalService
	AdminObjectZoneService
	AdminObjectCoordinatesService
}
