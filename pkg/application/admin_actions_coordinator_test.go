package application

import (
	"testing"

	"fyne.io/fyne/v2"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/eventbus"
	"obj_catalog_fyne_v3/pkg/models"
)

type appAdminObjectStub struct {
	deletedObjectID int64
	deleteErr       error
}

func (s *appAdminObjectStub) ListObjectTypes() ([]adminv1.DictionaryItem, error)     { return nil, nil }
func (s *appAdminObjectStub) ListObjectDistricts() ([]adminv1.DictionaryItem, error) { return nil, nil }
func (s *appAdminObjectStub) ListPPKConstructor() ([]adminv1.PPKConstructorItem, error) {
	return nil, nil
}
func (s *appAdminObjectStub) ListSubServers() ([]adminv1.SubServer, error) { return nil, nil }
func (s *appAdminObjectStub) FindObjectsBySIMPhone(string, *int64) ([]adminv1.SIMPhoneUsage, error) {
	return nil, nil
}
func (s *appAdminObjectStub) CreateObject(adminv1.ObjectCard) error { return nil }
func (s *appAdminObjectStub) ListObjectPersonals(int64) ([]adminv1.ObjectPersonal, error) {
	return nil, nil
}
func (s *appAdminObjectStub) AddObjectPersonal(int64, adminv1.ObjectPersonal) error    { return nil }
func (s *appAdminObjectStub) UpdateObjectPersonal(int64, adminv1.ObjectPersonal) error { return nil }
func (s *appAdminObjectStub) DeleteObjectPersonal(int64, int64) error                  { return nil }
func (s *appAdminObjectStub) FindPersonalByPhone(string) (*adminv1.ObjectPersonal, error) {
	return nil, nil
}
func (s *appAdminObjectStub) AddObjectZone(int64, adminv1.ObjectZone) error { return nil }
func (s *appAdminObjectStub) SaveObjectCoordinates(int64, adminv1.ObjectCoordinates) error {
	return nil
}
func (s *appAdminObjectStub) GetObjectCard(int64) (adminv1.ObjectCard, error) {
	return adminv1.ObjectCard{}, nil
}
func (s *appAdminObjectStub) UpdateObject(adminv1.ObjectCard) error { return nil }
func (s *appAdminObjectStub) DeleteObject(objn int64) error {
	s.deletedObjectID = objn
	return s.deleteErr
}
func (s *appAdminObjectStub) ListObjectZones(int64) ([]adminv1.ObjectZone, error) { return nil, nil }
func (s *appAdminObjectStub) UpdateObjectZone(int64, adminv1.ObjectZone) error    { return nil }
func (s *appAdminObjectStub) DeleteObjectZone(int64, int64) error                 { return nil }
func (s *appAdminObjectStub) FillObjectZones(int64, int64) error                  { return nil }
func (s *appAdminObjectStub) ClearObjectZones(int64) error                        { return nil }
func (s *appAdminObjectStub) GetObjectCoordinates(int64) (adminv1.ObjectCoordinates, error) {
	return adminv1.ObjectCoordinates{}, nil
}

func TestOpenNewObjectDialogPublishesObjectSaved(t *testing.T) {
	prevShow := showNewObjectDialogFn
	t.Cleanup(func() {
		showNewObjectDialogFn = prevShow
	})

	app := &Application{eventBus: eventbus.NewBus()}
	provider := &appAdminObjectStub{}

	savedObjectID := int64(0)
	app.eventBus.Subscribe(eventbus.TopicObjectSaved, func(payload any) {
		event, ok := payload.(eventbus.ObjectSavedEvent)
		if !ok {
			t.Fatalf("unexpected payload type: %T", payload)
		}
		savedObjectID = event.ObjectID
	})

	showNewObjectDialogFn = func(_ fyne.Window, got adminv1.ObjectWizardProvider, onSaved func(objn int64)) {
		if got != provider {
			t.Fatalf("provider mismatch")
		}
		onSaved(5100)
	}

	app.openNewObjectDialog(provider)

	if savedObjectID != 5100 {
		t.Fatalf("savedObjectID = %d, want 5100", savedObjectID)
	}
}

func TestOpenEditCurrentObjectDialogPublishesObjectSaved(t *testing.T) {
	prevShow := showEditObjectDialogFn
	prevInfo := showInfoDialogFn
	t.Cleanup(func() {
		showEditObjectDialogFn = prevShow
		showInfoDialogFn = prevInfo
	})

	app := &Application{
		eventBus:      eventbus.NewBus(),
		currentObject: &models.Object{ID: 5200, Name: "School"},
	}
	provider := &appAdminObjectStub{}

	savedObjectID := int64(0)
	app.eventBus.Subscribe(eventbus.TopicObjectSaved, func(payload any) {
		event := payload.(eventbus.ObjectSavedEvent)
		savedObjectID = event.ObjectID
	})

	showInfoDialogFn = func(_ fyne.Window, _, _ string) {}
	showEditObjectDialogFn = func(_ fyne.Window, got adminv1.ObjectCardProvider, objn int64, onSaved func(objn int64)) {
		if got != provider {
			t.Fatalf("provider mismatch")
		}
		if objn != 5200 {
			t.Fatalf("objn = %d, want 5200", objn)
		}
		onSaved(5200)
	}

	app.openEditCurrentObjectDialog(provider)

	if savedObjectID != 5200 {
		t.Fatalf("savedObjectID = %d, want 5200", savedObjectID)
	}
}

func TestConfirmDeleteCurrentObjectPublishesObjectDeleted(t *testing.T) {
	prevConfirm := showConfirmDialogFn
	prevInfo := showInfoDialogFn
	prevError := showErrorDialogFn
	t.Cleanup(func() {
		showConfirmDialogFn = prevConfirm
		showInfoDialogFn = prevInfo
		showErrorDialogFn = prevError
	})

	app := &Application{
		eventBus:      eventbus.NewBus(),
		currentObject: &models.Object{ID: 5300, Name: "Warehouse"},
	}
	provider := &appAdminObjectStub{}

	deletedObjectID := int64(0)
	app.eventBus.Subscribe(eventbus.TopicObjectDeleted, func(payload any) {
		event := payload.(eventbus.ObjectDeletedEvent)
		deletedObjectID = event.ObjectID
	})

	showInfoDialogFn = func(_ fyne.Window, _, _ string) {}
	showErrorDialogFn = func(_ fyne.Window, _ string, _ error) {}
	showConfirmDialogFn = func(_ string, _ string, onConfirm func(bool), _ fyne.Window) {
		onConfirm(true)
	}

	app.confirmDeleteCurrentObject(provider)

	if provider.deletedObjectID != 5300 {
		t.Fatalf("deletedObjectID = %d, want 5300", provider.deletedObjectID)
	}
	if deletedObjectID != 5300 {
		t.Fatalf("event deletedObjectID = %d, want 5300", deletedObjectID)
	}
}
