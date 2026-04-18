package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1ObjectWizardBase interface {
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	ListObjectDistricts() ([]contracts.DictionaryItem, error)
	ListPPKConstructor() ([]contracts.PPKConstructorItem, error)
	ListSubServers() ([]contracts.AdminSubServer, error)
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error)
	CreateObject(card contracts.AdminObjectCard) error
	ListObjectPersonals(objn int64) ([]contracts.AdminObjectPersonal, error)
	AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error
	UpdateObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error
	DeleteObjectPersonal(objn int64, personalID int64) error
	FindPersonalByPhone(phone string) (*contracts.AdminObjectPersonal, error)
	AddObjectZone(objn int64, zone contracts.AdminObjectZone) error
	SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error
}

type adminV1ObjectCardBase interface {
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	ListObjectDistricts() ([]contracts.DictionaryItem, error)
	ListPPKConstructor() ([]contracts.PPKConstructorItem, error)
	ListSubServers() ([]contracts.AdminSubServer, error)
	GetObjectCard(objn int64) (contracts.AdminObjectCard, error)
	CreateObject(card contracts.AdminObjectCard) error
	UpdateObject(card contracts.AdminObjectCard) error
	DeleteObject(objn int64) error
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error)
	ListObjectPersonals(objn int64) ([]contracts.AdminObjectPersonal, error)
	AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error
	UpdateObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error
	DeleteObjectPersonal(objn int64, personalID int64) error
	FindPersonalByPhone(phone string) (*contracts.AdminObjectPersonal, error)
	ListObjectZones(objn int64) ([]contracts.AdminObjectZone, error)
	AddObjectZone(objn int64, zone contracts.AdminObjectZone) error
	UpdateObjectZone(objn int64, zone contracts.AdminObjectZone) error
	DeleteObjectZone(objn int64, zoneID int64) error
	FillObjectZones(objn int64, count int64) error
	ClearObjectZones(objn int64) error
	GetObjectCoordinates(objn int64) (contracts.AdminObjectCoordinates, error)
	SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error
}

type adminV1ObjectWizardAdapter struct {
	base adminV1ObjectWizardBase
}

type adminV1ObjectCardAdapter struct {
	base adminV1ObjectCardBase
}

type adminV1ObjectDeleteAdapter struct {
	base interface {
		DeleteObject(objn int64) error
	}
}

func NewAdminV1ObjectWizardProvider(base adminV1ObjectWizardBase) adminv1.ObjectWizardProvider {
	if base == nil {
		return nil
	}
	return &adminV1ObjectWizardAdapter{base: base}
}

func NewAdminV1ObjectCardProvider(base adminV1ObjectCardBase) adminv1.ObjectCardProvider {
	if base == nil {
		return nil
	}
	return &adminV1ObjectCardAdapter{base: base}
}

func NewAdminV1ObjectDeleteProvider(base interface {
	DeleteObject(objn int64) error
}) adminv1.ObjectDeleteProvider {
	if base == nil {
		return nil
	}
	return &adminV1ObjectDeleteAdapter{base: base}
}

func (a *adminV1ObjectWizardAdapter) ListObjectTypes() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectTypes()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1ObjectWizardAdapter) ListObjectDistricts() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectDistricts()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1ObjectWizardAdapter) ListPPKConstructor() ([]adminv1.PPKConstructorItem, error) {
	items, err := a.base.ListPPKConstructor()
	if err != nil {
		return nil, err
	}
	return adminv1.ToPPKConstructorItems(items), nil
}

func (a *adminV1ObjectWizardAdapter) ListSubServers() ([]adminv1.SubServer, error) {
	items, err := a.base.ListSubServers()
	if err != nil {
		return nil, err
	}
	return adminv1.ToSubServers(items), nil
}

func (a *adminV1ObjectWizardAdapter) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]adminv1.SIMPhoneUsage, error) {
	items, err := a.base.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		return nil, err
	}
	return adminv1.ToSIMPhoneUsages(items), nil
}

func (a *adminV1ObjectWizardAdapter) CreateObject(card adminv1.ObjectCard) error {
	return a.base.CreateObject(adminv1.ToContractsObjectCard(card))
}

func (a *adminV1ObjectWizardAdapter) ListObjectPersonals(objn int64) ([]adminv1.ObjectPersonal, error) {
	items, err := a.base.ListObjectPersonals(objn)
	if err != nil {
		return nil, err
	}
	return adminv1.ToObjectPersonals(items), nil
}

func (a *adminV1ObjectWizardAdapter) AddObjectPersonal(objn int64, item adminv1.ObjectPersonal) error {
	return a.base.AddObjectPersonal(objn, adminv1.ToContractsObjectPersonal(item))
}

func (a *adminV1ObjectWizardAdapter) UpdateObjectPersonal(objn int64, item adminv1.ObjectPersonal) error {
	return a.base.UpdateObjectPersonal(objn, adminv1.ToContractsObjectPersonal(item))
}

func (a *adminV1ObjectWizardAdapter) DeleteObjectPersonal(objn int64, personalID int64) error {
	return a.base.DeleteObjectPersonal(objn, personalID)
}

func (a *adminV1ObjectWizardAdapter) FindPersonalByPhone(phone string) (*adminv1.ObjectPersonal, error) {
	item, err := a.base.FindPersonalByPhone(phone)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	value := adminv1.ToObjectPersonal(*item)
	return &value, nil
}

func (a *adminV1ObjectWizardAdapter) AddObjectZone(objn int64, zone adminv1.ObjectZone) error {
	return a.base.AddObjectZone(objn, adminv1.ToContractsObjectZone(zone))
}

func (a *adminV1ObjectWizardAdapter) SaveObjectCoordinates(objn int64, coords adminv1.ObjectCoordinates) error {
	return a.base.SaveObjectCoordinates(objn, adminv1.ToContractsObjectCoordinates(coords))
}

func (a *adminV1ObjectCardAdapter) ListObjectTypes() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectTypes()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1ObjectCardAdapter) ListObjectDistricts() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectDistricts()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1ObjectCardAdapter) ListPPKConstructor() ([]adminv1.PPKConstructorItem, error) {
	items, err := a.base.ListPPKConstructor()
	if err != nil {
		return nil, err
	}
	return adminv1.ToPPKConstructorItems(items), nil
}

func (a *adminV1ObjectCardAdapter) ListSubServers() ([]adminv1.SubServer, error) {
	items, err := a.base.ListSubServers()
	if err != nil {
		return nil, err
	}
	return adminv1.ToSubServers(items), nil
}

func (a *adminV1ObjectCardAdapter) GetObjectCard(objn int64) (adminv1.ObjectCard, error) {
	item, err := a.base.GetObjectCard(objn)
	if err != nil {
		return adminv1.ObjectCard{}, err
	}
	return adminv1.ToObjectCard(item), nil
}

func (a *adminV1ObjectCardAdapter) CreateObject(card adminv1.ObjectCard) error {
	return a.base.CreateObject(adminv1.ToContractsObjectCard(card))
}

func (a *adminV1ObjectCardAdapter) UpdateObject(card adminv1.ObjectCard) error {
	return a.base.UpdateObject(adminv1.ToContractsObjectCard(card))
}

func (a *adminV1ObjectCardAdapter) DeleteObject(objn int64) error {
	return a.base.DeleteObject(objn)
}

func (a *adminV1ObjectDeleteAdapter) DeleteObject(objn int64) error {
	return a.base.DeleteObject(objn)
}

func (a *adminV1ObjectCardAdapter) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]adminv1.SIMPhoneUsage, error) {
	items, err := a.base.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		return nil, err
	}
	return adminv1.ToSIMPhoneUsages(items), nil
}

func (a *adminV1ObjectCardAdapter) ListObjectPersonals(objn int64) ([]adminv1.ObjectPersonal, error) {
	items, err := a.base.ListObjectPersonals(objn)
	if err != nil {
		return nil, err
	}
	return adminv1.ToObjectPersonals(items), nil
}

func (a *adminV1ObjectCardAdapter) AddObjectPersonal(objn int64, item adminv1.ObjectPersonal) error {
	return a.base.AddObjectPersonal(objn, adminv1.ToContractsObjectPersonal(item))
}

func (a *adminV1ObjectCardAdapter) UpdateObjectPersonal(objn int64, item adminv1.ObjectPersonal) error {
	return a.base.UpdateObjectPersonal(objn, adminv1.ToContractsObjectPersonal(item))
}

func (a *adminV1ObjectCardAdapter) DeleteObjectPersonal(objn int64, personalID int64) error {
	return a.base.DeleteObjectPersonal(objn, personalID)
}

func (a *adminV1ObjectCardAdapter) FindPersonalByPhone(phone string) (*adminv1.ObjectPersonal, error) {
	item, err := a.base.FindPersonalByPhone(phone)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	value := adminv1.ToObjectPersonal(*item)
	return &value, nil
}

func (a *adminV1ObjectCardAdapter) ListObjectZones(objn int64) ([]adminv1.ObjectZone, error) {
	items, err := a.base.ListObjectZones(objn)
	if err != nil {
		return nil, err
	}
	return adminv1.ToObjectZones(items), nil
}

func (a *adminV1ObjectCardAdapter) AddObjectZone(objn int64, zone adminv1.ObjectZone) error {
	return a.base.AddObjectZone(objn, adminv1.ToContractsObjectZone(zone))
}

func (a *adminV1ObjectCardAdapter) UpdateObjectZone(objn int64, zone adminv1.ObjectZone) error {
	return a.base.UpdateObjectZone(objn, adminv1.ToContractsObjectZone(zone))
}

func (a *adminV1ObjectCardAdapter) DeleteObjectZone(objn int64, zoneID int64) error {
	return a.base.DeleteObjectZone(objn, zoneID)
}

func (a *adminV1ObjectCardAdapter) FillObjectZones(objn int64, count int64) error {
	return a.base.FillObjectZones(objn, count)
}

func (a *adminV1ObjectCardAdapter) ClearObjectZones(objn int64) error {
	return a.base.ClearObjectZones(objn)
}

func (a *adminV1ObjectCardAdapter) GetObjectCoordinates(objn int64) (adminv1.ObjectCoordinates, error) {
	item, err := a.base.GetObjectCoordinates(objn)
	if err != nil {
		return adminv1.ObjectCoordinates{}, err
	}
	return adminv1.ToObjectCoordinates(item), nil
}

func (a *adminV1ObjectCardAdapter) SaveObjectCoordinates(objn int64, coords adminv1.ObjectCoordinates) error {
	return a.base.SaveObjectCoordinates(objn, adminv1.ToContractsObjectCoordinates(coords))
}
