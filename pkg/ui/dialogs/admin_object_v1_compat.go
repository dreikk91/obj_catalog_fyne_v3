package dialogs

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectV1ReferencesAdapter struct {
	base adminv1.ObjectReferenceProvider
}

func (a objectV1ReferencesAdapter) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	items, err := a.base.ListObjectTypes()
	if err != nil {
		return nil, err
	}
	return toContractsDictionaryItems(items), nil
}

func (a objectV1ReferencesAdapter) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	items, err := a.base.ListObjectDistricts()
	if err != nil {
		return nil, err
	}
	return toContractsDictionaryItems(items), nil
}

func (a objectV1ReferencesAdapter) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	items, err := a.base.ListPPKConstructor()
	if err != nil {
		return nil, err
	}
	return toContractsPPKConstructorItems(items), nil
}

func (a objectV1ReferencesAdapter) ListSubServers() ([]contracts.AdminSubServer, error) {
	items, err := a.base.ListSubServers()
	if err != nil {
		return nil, err
	}
	return toContractsSubServers(items), nil
}

type objectV1SIMLookupAdapter struct {
	base adminv1.ObjectSIMLookupProvider
}

func (a objectV1SIMLookupAdapter) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]viewmodels.SIMPhoneUsage, error) {
	items, err := a.base.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		return nil, err
	}
	return viewmodels.SIMPhoneUsagesFromContracts(toContractsSIMPhoneUsages(items)), nil
}

type objectV1CardPersistenceAdapter struct {
	base adminv1.ObjectCardProvider
}

func (a objectV1CardPersistenceAdapter) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	item, err := a.base.GetObjectCard(objn)
	if err != nil {
		return contracts.AdminObjectCard{}, err
	}
	return adminv1.ToContractsObjectCard(item), nil
}

func (a objectV1CardPersistenceAdapter) CreateObject(card contracts.AdminObjectCard) error {
	return a.base.CreateObject(adminv1.ToObjectCard(card))
}

func (a objectV1CardPersistenceAdapter) UpdateObject(card contracts.AdminObjectCard) error {
	return a.base.UpdateObject(adminv1.ToObjectCard(card))
}

type objectV1WizardPersistenceAdapter struct {
	base adminv1.ObjectWizardProvider
}

func (a objectV1WizardPersistenceAdapter) CreateObject(card contracts.AdminObjectCard) error {
	return a.base.CreateObject(adminv1.ToObjectCard(card))
}

func (a objectV1WizardPersistenceAdapter) AddObjectPersonal(objn int64, item viewmodels.ObjectPersonal) error {
	return a.base.AddObjectPersonal(objn, adminv1.ToObjectPersonal(item.ToContracts()))
}

func (a objectV1WizardPersistenceAdapter) AddObjectZone(objn int64, zone viewmodels.ObjectZone) error {
	return a.base.AddObjectZone(objn, adminv1.ToObjectZone(zone.ToContracts()))
}

func (a objectV1WizardPersistenceAdapter) SaveObjectCoordinates(objn int64, coords viewmodels.ObjectCoordinates) error {
	return a.base.SaveObjectCoordinates(objn, adminv1.ToObjectCoordinates(coords.ToContracts()))
}

type objectV1DistrictReferenceAdapter struct {
	base interface {
		ListObjectDistricts() ([]adminv1.DictionaryItem, error)
	}
}

func (a objectV1DistrictReferenceAdapter) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	items, err := a.base.ListObjectDistricts()
	if err != nil {
		return nil, err
	}
	return toContractsDictionaryItems(items), nil
}

func toContractsDictionaryItems(items []adminv1.DictionaryItem) []contracts.DictionaryItem {
	result := make([]contracts.DictionaryItem, 0, len(items))
	for _, item := range items {
		result = append(result, contracts.DictionaryItem{
			ID:    item.ID,
			Name:  item.Name,
			Code:  item.Code,
			Extra: item.Extra,
		})
	}
	return result
}

func toContractsPPKConstructorItems(items []adminv1.PPKConstructorItem) []contracts.PPKConstructorItem {
	result := make([]contracts.PPKConstructorItem, 0, len(items))
	for _, item := range items {
		result = append(result, contracts.PPKConstructorItem{
			ID:        item.ID,
			Name:      item.Name,
			Channel:   item.Channel,
			ZoneCount: item.ZoneCount,
		})
	}
	return result
}

func toContractsSubServers(items []adminv1.SubServer) []contracts.AdminSubServer {
	result := make([]contracts.AdminSubServer, 0, len(items))
	for _, item := range items {
		result = append(result, contracts.AdminSubServer{
			ID:    item.ID,
			Info:  item.Info,
			Bind:  item.Bind,
			Host:  item.Host,
			Type:  item.Type,
			Host2: item.Host2,
		})
	}
	return result
}

func toContractsSIMPhoneUsages(items []adminv1.SIMPhoneUsage) []contracts.AdminSIMPhoneUsage {
	result := make([]contracts.AdminSIMPhoneUsage, 0, len(items))
	for _, item := range items {
		result = append(result, contracts.AdminSIMPhoneUsage{
			ObjN:          item.ObjN,
			DisplayNumber: item.DisplayNumber,
			Name:          item.Name,
			Slot:          item.Slot,
			Source:        item.Source,
		})
	}
	return result
}

func toViewmodelObjectPersonals(items []adminv1.ObjectPersonal) []viewmodels.ObjectPersonal {
	result := make([]viewmodels.ObjectPersonal, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodels.ObjectPersonalFromContracts(adminv1.ToContractsObjectPersonal(item)))
	}
	return result
}

func toAdminObjectPersonals(items []viewmodels.ObjectPersonal) []adminv1.ObjectPersonal {
	result := make([]adminv1.ObjectPersonal, 0, len(items))
	for _, item := range items {
		result = append(result, adminv1.ToObjectPersonal(item.ToContracts()))
	}
	return result
}

func toViewmodelObjectZones(items []adminv1.ObjectZone) []viewmodels.ObjectZone {
	result := make([]viewmodels.ObjectZone, 0, len(items))
	for _, item := range items {
		result = append(result, viewmodels.ObjectZoneFromContracts(adminv1.ToContractsObjectZone(item)))
	}
	return result
}

func toAdminObjectZones(items []viewmodels.ObjectZone) []adminv1.ObjectZone {
	result := make([]adminv1.ObjectZone, 0, len(items))
	for _, item := range items {
		result = append(result, adminv1.ToObjectZone(item.ToContracts()))
	}
	return result
}

func toAdminObjectCoordinates(coords viewmodels.ObjectCoordinates) adminv1.ObjectCoordinates {
	return adminv1.ToObjectCoordinates(coords.ToContracts())
}

var (
	_ viewmodels.ObjectCardReferenceProvider = (*objectV1ReferencesAdapter)(nil)
	_ viewmodels.SIMPhoneUsageLookup         = (*objectV1SIMLookupAdapter)(nil)
	_ viewmodels.ObjectCardPersistence       = (*objectV1CardPersistenceAdapter)(nil)
	_ viewmodels.ObjectWizardPersistence     = (*objectV1WizardPersistenceAdapter)(nil)
)
