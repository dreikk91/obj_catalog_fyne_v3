package backend

import (
	"context"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type frontendAdminBridgeBackendStub struct {
	createInput contracts.FrontendObjectUpsertRequest
	updateInput contracts.FrontendObjectUpsertRequest
}

func (s *frontendAdminBridgeBackendStub) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	return contracts.FrontendCapabilities{}, nil
}

func (s *frontendAdminBridgeBackendStub) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	return nil, nil
}

func (s *frontendAdminBridgeBackendStub) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	return nil, nil
}

func (s *frontendAdminBridgeBackendStub) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	return nil, nil
}

func (s *frontendAdminBridgeBackendStub) GetObjectDetails(context.Context, int) (contracts.FrontendObjectDetails, error) {
	return contracts.FrontendObjectDetails{}, nil
}

func (s *frontendAdminBridgeBackendStub) CreateObject(_ context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	s.createInput = request
	return contracts.FrontendObjectMutationResult{}, nil
}

func (s *frontendAdminBridgeBackendStub) UpdateObject(_ context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	s.updateInput = request
	return contracts.FrontendObjectMutationResult{}, nil
}

type frontendAdminWizardBaseStub struct{}

func (frontendAdminWizardBaseStub) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) ListSubServers() ([]contracts.AdminSubServer, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) FindObjectsBySIMPhone(string, *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) CreateObject(contracts.AdminObjectCard) error {
	return nil
}

func (frontendAdminWizardBaseStub) ListObjectPersonals(int64) ([]contracts.AdminObjectPersonal, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) AddObjectPersonal(int64, contracts.AdminObjectPersonal) error {
	return nil
}

func (frontendAdminWizardBaseStub) UpdateObjectPersonal(int64, contracts.AdminObjectPersonal) error {
	return nil
}

func (frontendAdminWizardBaseStub) DeleteObjectPersonal(int64, int64) error {
	return nil
}

func (frontendAdminWizardBaseStub) FindPersonalByPhone(string) (*contracts.AdminObjectPersonal, error) {
	return nil, nil
}

func (frontendAdminWizardBaseStub) AddObjectZone(int64, contracts.AdminObjectZone) error {
	return nil
}

func (frontendAdminWizardBaseStub) SaveObjectCoordinates(int64, contracts.AdminObjectCoordinates) error {
	return nil
}

type frontendAdminCardBaseStub struct {
	frontendAdminWizardBaseStub
}

func (frontendAdminCardBaseStub) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	return contracts.AdminObjectCard{ObjN: objn, ObjUIN: 77}, nil
}

func (frontendAdminCardBaseStub) UpdateObject(contracts.AdminObjectCard) error {
	return nil
}

func (frontendAdminCardBaseStub) DeleteObject(int64) error {
	return nil
}

func (frontendAdminCardBaseStub) GetVodafoneAuthState() (contracts.VodafoneAuthState, error) {
	return contracts.VodafoneAuthState{}, nil
}

func (frontendAdminCardBaseStub) RequestVodafoneLoginSMS(string) error {
	return nil
}

func (frontendAdminCardBaseStub) VerifyVodafoneLogin(string, string) (contracts.VodafoneAuthState, error) {
	return contracts.VodafoneAuthState{}, nil
}

func (frontendAdminCardBaseStub) ClearVodafoneLogin() error {
	return nil
}

func (frontendAdminCardBaseStub) GetVodafoneSIMStatus(string) (contracts.VodafoneSIMStatus, error) {
	return contracts.VodafoneSIMStatus{}, nil
}

func (frontendAdminCardBaseStub) BlockVodafoneSIM(string) (contracts.VodafoneSIMBarringResult, error) {
	return contracts.VodafoneSIMBarringResult{}, nil
}

func (frontendAdminCardBaseStub) UnblockVodafoneSIM(string) (contracts.VodafoneSIMBarringResult, error) {
	return contracts.VodafoneSIMBarringResult{}, nil
}

func (frontendAdminCardBaseStub) RebootVodafoneSIM(string) (contracts.VodafoneSIMRebootResult, error) {
	return contracts.VodafoneSIMRebootResult{}, nil
}

func (frontendAdminCardBaseStub) UpdateVodafoneSIMMetadata(string, string, string) error {
	return nil
}

func (frontendAdminCardBaseStub) GetKyivstarAuthState() (contracts.KyivstarAuthState, error) {
	return contracts.KyivstarAuthState{}, nil
}

func (frontendAdminCardBaseStub) RefreshKyivstarToken() (contracts.KyivstarAuthState, error) {
	return contracts.KyivstarAuthState{}, nil
}

func (frontendAdminCardBaseStub) ClearKyivstarToken() error {
	return nil
}

func (frontendAdminCardBaseStub) GetKyivstarSIMStatus(string) (contracts.KyivstarSIMStatus, error) {
	return contracts.KyivstarSIMStatus{}, nil
}

func (frontendAdminCardBaseStub) ListKyivstarSIMInventory([]string) (map[string]contracts.KyivstarSIMInventoryEntry, error) {
	return nil, nil
}

func (frontendAdminCardBaseStub) PauseKyivstarSIM(string) (contracts.KyivstarSIMOperationResult, error) {
	return contracts.KyivstarSIMOperationResult{}, nil
}

func (frontendAdminCardBaseStub) ActivateKyivstarSIM(string) (contracts.KyivstarSIMOperationResult, error) {
	return contracts.KyivstarSIMOperationResult{}, nil
}

func (frontendAdminCardBaseStub) PauseKyivstarSIMServices(string, []string) (contracts.KyivstarSIMOperationResult, error) {
	return contracts.KyivstarSIMOperationResult{}, nil
}

func (frontendAdminCardBaseStub) ActivateKyivstarSIMServices(string, []string) (contracts.KyivstarSIMOperationResult, error) {
	return contracts.KyivstarSIMOperationResult{}, nil
}

func (frontendAdminCardBaseStub) RebootKyivstarSIM(string) (contracts.KyivstarSIMResetResult, error) {
	return contracts.KyivstarSIMResetResult{}, nil
}

func (frontendAdminCardBaseStub) UpdateKyivstarSIMMetadata(string, string, string) error {
	return nil
}

func (frontendAdminCardBaseStub) ListObjectZones(int64) ([]contracts.AdminObjectZone, error) {
	return nil, nil
}

func (frontendAdminCardBaseStub) UpdateObjectZone(int64, contracts.AdminObjectZone) error {
	return nil
}

func (frontendAdminCardBaseStub) DeleteObjectZone(int64, int64) error {
	return nil
}

func (frontendAdminCardBaseStub) FillObjectZones(int64, int64) error {
	return nil
}

func (frontendAdminCardBaseStub) ClearObjectZones(int64) error {
	return nil
}

func (frontendAdminCardBaseStub) GetObjectCoordinates(int64) (contracts.AdminObjectCoordinates, error) {
	return contracts.AdminObjectCoordinates{}, nil
}

func TestFrontendAdminWizardBridgeCreateObjectUsesFrontendBackend(t *testing.T) {
	backendStub := &frontendAdminBridgeBackendStub{}
	bridge := NewFrontendAdminWizardBridge(backendStub, frontendAdminWizardBaseStub{})

	err := bridge.CreateObject(contracts.AdminObjectCard{
		ObjN:      1200,
		ObjTypeID: 7,
		ShortName: "Школа",
		FullName:  "Школа №7",
		Address:   "Львів",
	})
	if err != nil {
		t.Fatalf("CreateObject() error = %v", err)
	}
	if backendStub.createInput.Source != contracts.FrontendSourceBridge {
		t.Fatalf("create source = %q, want %q", backendStub.createInput.Source, contracts.FrontendSourceBridge)
	}
	if backendStub.createInput.Legacy == nil || backendStub.createInput.Legacy.ObjN != 1200 {
		t.Fatalf("create legacy objn = %+v, want 1200", backendStub.createInput.Legacy)
	}
}

func TestFrontendAdminCardBridgeUpdateObjectUsesFrontendBackend(t *testing.T) {
	backendStub := &frontendAdminBridgeBackendStub{}
	bridge := NewFrontendAdminCardBridge(backendStub, frontendAdminCardBaseStub{})

	err := bridge.UpdateObject(contracts.AdminObjectCard{
		ObjN:      1300,
		ObjUIN:    55,
		ObjTypeID: 8,
		ShortName: "Ліцей",
		Address:   "Стрий",
	})
	if err != nil {
		t.Fatalf("UpdateObject() error = %v", err)
	}
	if backendStub.updateInput.ObjectID != 1300 {
		t.Fatalf("update object id = %d, want 1300", backendStub.updateInput.ObjectID)
	}
	if backendStub.updateInput.Legacy == nil || backendStub.updateInput.Legacy.ObjUIN != 55 {
		t.Fatalf("update legacy = %+v, want ObjUIN 55", backendStub.updateInput.Legacy)
	}
}
