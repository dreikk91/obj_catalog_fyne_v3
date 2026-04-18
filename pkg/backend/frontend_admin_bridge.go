package backend

import (
	"context"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type frontendAdminWizardBridge struct {
	frontend contracts.FrontendBackend
	base     contracts.AdminObjectWizardProvider
}

func NewFrontendAdminWizardBridge(frontend contracts.FrontendBackend, base contracts.AdminObjectWizardProvider) contracts.AdminObjectWizardProvider {
	if frontend == nil || base == nil {
		return base
	}
	return &frontendAdminWizardBridge{
		frontend: frontend,
		base:     base,
	}
}

func (b *frontendAdminWizardBridge) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return b.base.ListObjectTypes()
}

func (b *frontendAdminWizardBridge) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	return b.base.ListObjectDistricts()
}

func (b *frontendAdminWizardBridge) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	return b.base.ListPPKConstructor()
}

func (b *frontendAdminWizardBridge) ListSubServers() ([]contracts.AdminSubServer, error) {
	return b.base.ListSubServers()
}

func (b *frontendAdminWizardBridge) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	return b.base.FindObjectsBySIMPhone(phone, excludeObjN)
}

func (b *frontendAdminWizardBridge) CreateObject(card contracts.AdminObjectCard) error {
	_, err := b.frontend.CreateObject(context.Background(), frontendLegacyUpsertRequest(0, card))
	return err
}

func (b *frontendAdminWizardBridge) ListObjectPersonals(objn int64) ([]contracts.AdminObjectPersonal, error) {
	return b.base.ListObjectPersonals(objn)
}

func (b *frontendAdminWizardBridge) AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	return b.base.AddObjectPersonal(objn, item)
}

func (b *frontendAdminWizardBridge) UpdateObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	return b.base.UpdateObjectPersonal(objn, item)
}

func (b *frontendAdminWizardBridge) DeleteObjectPersonal(objn int64, personalID int64) error {
	return b.base.DeleteObjectPersonal(objn, personalID)
}

func (b *frontendAdminWizardBridge) FindPersonalByPhone(phone string) (*contracts.AdminObjectPersonal, error) {
	return b.base.FindPersonalByPhone(phone)
}

func (b *frontendAdminWizardBridge) AddObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	return b.base.AddObjectZone(objn, zone)
}

func (b *frontendAdminWizardBridge) SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error {
	return b.base.SaveObjectCoordinates(objn, coords)
}

type frontendAdminCardBridge struct {
	frontend contracts.FrontendBackend
	base     contracts.AdminObjectCardProvider
}

func NewFrontendAdminCardBridge(frontend contracts.FrontendBackend, base contracts.AdminObjectCardProvider) contracts.AdminObjectCardProvider {
	if frontend == nil || base == nil {
		return base
	}
	return &frontendAdminCardBridge{
		frontend: frontend,
		base:     base,
	}
}

func (b *frontendAdminCardBridge) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return b.base.ListObjectTypes()
}

func (b *frontendAdminCardBridge) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	return b.base.ListObjectDistricts()
}

func (b *frontendAdminCardBridge) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	return b.base.ListPPKConstructor()
}

func (b *frontendAdminCardBridge) ListSubServers() ([]contracts.AdminSubServer, error) {
	return b.base.ListSubServers()
}

func (b *frontendAdminCardBridge) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	return b.base.GetObjectCard(objn)
}

func (b *frontendAdminCardBridge) CreateObject(card contracts.AdminObjectCard) error {
	_, err := b.frontend.CreateObject(context.Background(), frontendLegacyUpsertRequest(0, card))
	return err
}

func (b *frontendAdminCardBridge) UpdateObject(card contracts.AdminObjectCard) error {
	_, err := b.frontend.UpdateObject(context.Background(), frontendLegacyUpsertRequest(card.ObjN, card))
	return err
}

func (b *frontendAdminCardBridge) DeleteObject(objn int64) error {
	return b.base.DeleteObject(objn)
}

func (b *frontendAdminCardBridge) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	return b.base.FindObjectsBySIMPhone(phone, excludeObjN)
}

func (b *frontendAdminCardBridge) GetVodafoneAuthState() (contracts.VodafoneAuthState, error) {
	return b.base.GetVodafoneAuthState()
}

func (b *frontendAdminCardBridge) RequestVodafoneLoginSMS(phone string) error {
	return b.base.RequestVodafoneLoginSMS(phone)
}

func (b *frontendAdminCardBridge) VerifyVodafoneLogin(phone string, code string) (contracts.VodafoneAuthState, error) {
	return b.base.VerifyVodafoneLogin(phone, code)
}

func (b *frontendAdminCardBridge) ClearVodafoneLogin() error {
	return b.base.ClearVodafoneLogin()
}

func (b *frontendAdminCardBridge) GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error) {
	return b.base.GetVodafoneSIMStatus(msisdn)
}

func (b *frontendAdminCardBridge) BlockVodafoneSIM(msisdn string) (contracts.VodafoneSIMBarringResult, error) {
	return b.base.BlockVodafoneSIM(msisdn)
}

func (b *frontendAdminCardBridge) UnblockVodafoneSIM(msisdn string) (contracts.VodafoneSIMBarringResult, error) {
	return b.base.UnblockVodafoneSIM(msisdn)
}

func (b *frontendAdminCardBridge) RebootVodafoneSIM(msisdn string) (contracts.VodafoneSIMRebootResult, error) {
	return b.base.RebootVodafoneSIM(msisdn)
}

func (b *frontendAdminCardBridge) UpdateVodafoneSIMMetadata(msisdn string, name string, comment string) error {
	return b.base.UpdateVodafoneSIMMetadata(msisdn, name, comment)
}

func (b *frontendAdminCardBridge) GetKyivstarAuthState() (contracts.KyivstarAuthState, error) {
	return b.base.GetKyivstarAuthState()
}

func (b *frontendAdminCardBridge) RefreshKyivstarToken() (contracts.KyivstarAuthState, error) {
	return b.base.RefreshKyivstarToken()
}

func (b *frontendAdminCardBridge) ClearKyivstarToken() error {
	return b.base.ClearKyivstarToken()
}

func (b *frontendAdminCardBridge) GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error) {
	return b.base.GetKyivstarSIMStatus(msisdn)
}

func (b *frontendAdminCardBridge) ListKyivstarSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error) {
	return b.base.ListKyivstarSIMInventory(numbers)
}

func (b *frontendAdminCardBridge) PauseKyivstarSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	return b.base.PauseKyivstarSIM(msisdn)
}

func (b *frontendAdminCardBridge) ActivateKyivstarSIM(msisdn string) (contracts.KyivstarSIMOperationResult, error) {
	return b.base.ActivateKyivstarSIM(msisdn)
}

func (b *frontendAdminCardBridge) PauseKyivstarSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	return b.base.PauseKyivstarSIMServices(msisdn, serviceIDs)
}

func (b *frontendAdminCardBridge) ActivateKyivstarSIMServices(msisdn string, serviceIDs []string) (contracts.KyivstarSIMOperationResult, error) {
	return b.base.ActivateKyivstarSIMServices(msisdn, serviceIDs)
}

func (b *frontendAdminCardBridge) RebootKyivstarSIM(msisdn string) (contracts.KyivstarSIMResetResult, error) {
	return b.base.RebootKyivstarSIM(msisdn)
}

func (b *frontendAdminCardBridge) UpdateKyivstarSIMMetadata(msisdn string, deviceName string, deviceID string) error {
	return b.base.UpdateKyivstarSIMMetadata(msisdn, deviceName, deviceID)
}

func (b *frontendAdminCardBridge) ListObjectPersonals(objn int64) ([]contracts.AdminObjectPersonal, error) {
	return b.base.ListObjectPersonals(objn)
}

func (b *frontendAdminCardBridge) AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	return b.base.AddObjectPersonal(objn, item)
}

func (b *frontendAdminCardBridge) UpdateObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	return b.base.UpdateObjectPersonal(objn, item)
}

func (b *frontendAdminCardBridge) DeleteObjectPersonal(objn int64, personalID int64) error {
	return b.base.DeleteObjectPersonal(objn, personalID)
}

func (b *frontendAdminCardBridge) FindPersonalByPhone(phone string) (*contracts.AdminObjectPersonal, error) {
	return b.base.FindPersonalByPhone(phone)
}

func (b *frontendAdminCardBridge) ListObjectZones(objn int64) ([]contracts.AdminObjectZone, error) {
	return b.base.ListObjectZones(objn)
}

func (b *frontendAdminCardBridge) AddObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	return b.base.AddObjectZone(objn, zone)
}

func (b *frontendAdminCardBridge) UpdateObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	return b.base.UpdateObjectZone(objn, zone)
}

func (b *frontendAdminCardBridge) DeleteObjectZone(objn int64, zoneID int64) error {
	return b.base.DeleteObjectZone(objn, zoneID)
}

func (b *frontendAdminCardBridge) FillObjectZones(objn int64, count int64) error {
	return b.base.FillObjectZones(objn, count)
}

func (b *frontendAdminCardBridge) ClearObjectZones(objn int64) error {
	return b.base.ClearObjectZones(objn)
}

func (b *frontendAdminCardBridge) GetObjectCoordinates(objn int64) (contracts.AdminObjectCoordinates, error) {
	return b.base.GetObjectCoordinates(objn)
}

func (b *frontendAdminCardBridge) SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error {
	return b.base.SaveObjectCoordinates(objn, coords)
}

func frontendLegacyUpsertRequest(objectID int64, card contracts.AdminObjectCard) contracts.FrontendObjectUpsertRequest {
	return contracts.FrontendObjectUpsertRequest{
		Source:   contracts.FrontendSourceBridge,
		ObjectID: int(firstPositiveInt64(objectID, card.ObjN)),
		Core: contracts.FrontendObjectCoreFields{
			Name:     firstNonEmpty(card.FullName, card.ShortName),
			Address:  card.Address,
			Contract: card.Contract,
			Notes:    card.Notes,
		},
		Legacy: &contracts.FrontendLegacyObjectPayload{
			ObjUIN:             card.ObjUIN,
			ObjN:               card.ObjN,
			GrpN:               card.GrpN,
			ObjTypeID:          card.ObjTypeID,
			ObjRegID:           card.ObjRegID,
			ChannelCode:        card.ChannelCode,
			PPKID:              card.PPKID,
			GSMHiddenN:         card.GSMHiddenN,
			TestIntervalMin:    card.TestIntervalMin,
			ShortName:          card.ShortName,
			FullName:           card.FullName,
			Phones:             card.Phones,
			StartDate:          card.StartDate,
			Location:           card.Location,
			GSMPhone1:          card.GSMPhone1,
			GSMPhone2:          card.GSMPhone2,
			SubServerA:         card.SubServerA,
			SubServerB:         card.SubServerB,
			TestControlEnabled: card.TestControlEnabled,
		},
	}
}

func firstPositiveInt64(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
