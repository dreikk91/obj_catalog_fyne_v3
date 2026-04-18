package backend

import (
	"context"
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestAdminV1ObjectWizardProviderUsesFrontendMutationBoundary(t *testing.T) {
	adminMutator := &frontendTestAdminMutator{}
	frontend := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendAdminObjectMutator(adminMutator),
	)

	provider := NewAdminV1ObjectWizardProvider(
		NewFrontendAdminWizardBridge(frontend, frontendAdminWizardBaseStub{}),
	)

	err := provider.CreateObject(adminv1.ObjectCard{
		ObjN:      4100,
		ObjTypeID: 7,
		ShortName: "Ліцей 7",
		FullName:  "Ліцей №7",
		Address:   "Львів",
		Phones:    "0501234567",
	})
	if err != nil {
		t.Fatalf("CreateObject() error = %v", err)
	}
	if len(adminMutator.createdCards) != 1 {
		t.Fatalf("created cards = %d, want 1", len(adminMutator.createdCards))
	}
	created := adminMutator.createdCards[0]
	if created.ObjN != 4100 {
		t.Fatalf("created.ObjN = %d, want 4100", created.ObjN)
	}
	if created.ShortName != "Ліцей 7" {
		t.Fatalf("created.ShortName = %q, want %q", created.ShortName, "Ліцей 7")
	}
	if created.ObjTypeID != 7 {
		t.Fatalf("created.ObjTypeID = %d, want 7", created.ObjTypeID)
	}
}

func TestAdminV1ObjectCardProviderUsesFrontendMutationBoundary(t *testing.T) {
	adminMutator := &frontendTestAdminMutator{
		currentCard: contracts.AdminObjectCard{
			ObjN:      4200,
			ObjUIN:    91,
			ShortName: "Старий об'єкт",
			FullName:  "Старий об'єкт повністю",
			ObjTypeID: 3,
			Address:   "Стара адреса",
		},
	}
	frontend := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendAdminObjectMutator(adminMutator),
	)

	provider := NewAdminV1ObjectCardProvider(
		NewFrontendAdminCardBridge(frontend, frontendAdminCardBaseStub{}),
	)

	err := provider.UpdateObject(adminv1.ObjectCard{
		ObjN:      4200,
		ObjUIN:    91,
		ShortName: "Новий об'єкт",
		Address:   "Нова адреса",
	})
	if err != nil {
		t.Fatalf("UpdateObject() error = %v", err)
	}
	if len(adminMutator.updatedCards) != 1 {
		t.Fatalf("updated cards = %d, want 1", len(adminMutator.updatedCards))
	}
	updated := adminMutator.updatedCards[0]
	if updated.ObjN != 4200 {
		t.Fatalf("updated.ObjN = %d, want 4200", updated.ObjN)
	}
	if updated.ObjUIN != 91 {
		t.Fatalf("updated.ObjUIN = %d, want 91", updated.ObjUIN)
	}
	if updated.ShortName != "Новий об'єкт" {
		t.Fatalf("updated.ShortName = %q, want %q", updated.ShortName, "Новий об'єкт")
	}
	if updated.FullName != "Новий об'єкт" {
		t.Fatalf("updated.FullName = %q, want %q", updated.FullName, "Новий об'єкт")
	}
	if updated.Address != "Нова адреса" {
		t.Fatalf("updated.Address = %q, want %q", updated.Address, "Нова адреса")
	}
}

func TestAdminV1ObjectDeleteProviderDelegatesToBaseProvider(t *testing.T) {
	base := &adminV1ObjectStub{}
	provider := NewAdminV1ObjectDeleteProvider(base)

	if err := provider.DeleteObject(4300); err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}
	if base.deletedObjectID != 4300 {
		t.Fatalf("deletedObjectID = %d, want 4300", base.deletedObjectID)
	}
}

func TestFrontendAdminBridgesCanBeWrappedByAdminV1Providers(t *testing.T) {
	backendStub := &frontendAdminBridgeBackendStub{}

	wizardProvider := NewAdminV1ObjectWizardProvider(
		NewFrontendAdminWizardBridge(backendStub, frontendAdminWizardBaseStub{}),
	)
	if err := wizardProvider.CreateObject(adminv1.ObjectCard{ObjN: 4400, ShortName: "Школа", ObjTypeID: 4}); err != nil {
		t.Fatalf("wizard CreateObject() error = %v", err)
	}
	if backendStub.createInput.Source != contracts.FrontendSourceBridge || backendStub.createInput.Legacy == nil || backendStub.createInput.Legacy.ObjN != 4400 {
		t.Fatalf("wizard create input = %+v", backendStub.createInput)
	}

	cardProvider := NewAdminV1ObjectCardProvider(
		NewFrontendAdminCardBridge(backendStub, frontendAdminCardBaseStub{}),
	)
	if err := cardProvider.UpdateObject(adminv1.ObjectCard{ObjN: 4500, ObjUIN: 77, ShortName: "Гімназія"}); err != nil {
		t.Fatalf("card UpdateObject() error = %v", err)
	}
	if backendStub.updateInput.ObjectID != 4500 || backendStub.updateInput.Legacy == nil || backendStub.updateInput.Legacy.ObjUIN != 77 {
		t.Fatalf("card update input = %+v", backendStub.updateInput)
	}
}

var _ interface {
	CreateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error)
	UpdateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error)
} = (*frontendAdminBridgeBackendStub)(nil)
