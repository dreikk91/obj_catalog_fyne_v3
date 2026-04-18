package backend

import (
	"context"
	"errors"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

type frontendTestDataProvider struct {
	objectByID map[string]models.Object
	objects    []models.Object
	events     []models.Event
	alarms     []models.Alarm
}

func (p *frontendTestDataProvider) GetObjects() []models.Object {
	return append([]models.Object(nil), p.objects...)
}

func (p *frontendTestDataProvider) GetObjectByID(id string) *models.Object {
	if item, ok := p.objectByID[id]; ok {
		copy := item
		return &copy
	}
	return nil
}

func (p *frontendTestDataProvider) GetZones(string) []models.Zone {
	return nil
}

func (p *frontendTestDataProvider) GetEmployees(string) []models.Contact {
	return nil
}

func (p *frontendTestDataProvider) GetTestMessages(string) []models.TestMessage {
	return nil
}

func (p *frontendTestDataProvider) GetExternalData(string) (string, string, time.Time, time.Time) {
	return "", "", time.Time{}, time.Time{}
}

func (p *frontendTestDataProvider) GetEvents() []models.Event {
	return append([]models.Event(nil), p.events...)
}

func (p *frontendTestDataProvider) GetObjectEvents(string) []models.Event {
	return nil
}

func (p *frontendTestDataProvider) GetAlarms() []models.Alarm {
	return append([]models.Alarm(nil), p.alarms...)
}

func (p *frontendTestDataProvider) ProcessAlarm(string, string, string) error {
	return nil
}

type frontendTestAdminMutator struct {
	currentCard  contracts.AdminObjectCard
	createdCards []contracts.AdminObjectCard
	updatedCards []contracts.AdminObjectCard
	getErr       error
}

func (m *frontendTestAdminMutator) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	if m.getErr != nil {
		return contracts.AdminObjectCard{}, m.getErr
	}
	card := m.currentCard
	if card.ObjN == 0 {
		card.ObjN = objn
	}
	return card, nil
}

func (m *frontendTestAdminMutator) CreateObject(card contracts.AdminObjectCard) error {
	m.createdCards = append(m.createdCards, card)
	return nil
}

func (m *frontendTestAdminMutator) UpdateObject(card contracts.AdminObjectCard) error {
	m.updatedCards = append(m.updatedCards, card)
	return nil
}

type frontendTestCASLMutator struct {
	snapshot      contracts.CASLObjectEditorSnapshot
	createID      string
	createErr     error
	updateErr     error
	createCalls   []contracts.CASLGuardObjectCreate
	updateCalls   []contracts.CASLGuardObjectUpdate
	snapshotCalls []int64
}

func (m *frontendTestCASLMutator) GetCASLObjectEditorSnapshot(ctx context.Context, objectID int64) (contracts.CASLObjectEditorSnapshot, error) {
	_ = ctx
	m.snapshotCalls = append(m.snapshotCalls, objectID)
	return m.snapshot, nil
}

func (m *frontendTestCASLMutator) CreateCASLObject(ctx context.Context, create contracts.CASLGuardObjectCreate) (string, error) {
	_ = ctx
	m.createCalls = append(m.createCalls, create)
	if m.createErr != nil {
		return "", m.createErr
	}
	return m.createID, nil
}

func (m *frontendTestCASLMutator) UpdateCASLObject(ctx context.Context, update contracts.CASLGuardObjectUpdate) error {
	_ = ctx
	m.updateCalls = append(m.updateCalls, update)
	return m.updateErr
}

type frontendTestCapabilityProvider struct {
	capabilities []contracts.FrontendSourceCapability
}

func (p *frontendTestCapabilityProvider) FrontendSourceCapabilities() []contracts.FrontendSourceCapability {
	return append([]contracts.FrontendSourceCapability(nil), p.capabilities...)
}

func TestFrontendAdapterCreateLegacyObject(t *testing.T) {
	admin := &frontendTestAdminMutator{}
	adapter := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendAdminObjectMutator(admin),
	)

	result, err := adapter.CreateObject(context.Background(), contracts.FrontendObjectUpsertRequest{
		Source: contracts.FrontendSourceBridge,
		Core: contracts.FrontendObjectCoreFields{
			Name:     "Школа 12",
			Address:  "Львів",
			Contract: "DOG-1",
			Notes:    "нічний режим",
		},
		Legacy: &contracts.FrontendLegacyObjectPayload{
			ObjN:               1204,
			ObjTypeID:          7,
			ShortName:          "Школа 12",
			Phones:             "0501234567",
			TestControlEnabled: true,
			TestIntervalMin:    15,
		},
	})
	if err != nil {
		t.Fatalf("CreateObject() error = %v", err)
	}
	if result.ObjectID != 1204 {
		t.Fatalf("CreateObject() ObjectID = %d, want 1204", result.ObjectID)
	}
	if len(admin.createdCards) != 1 {
		t.Fatalf("CreateObject() created %d cards, want 1", len(admin.createdCards))
	}
	created := admin.createdCards[0]
	if created.ShortName != "Школа 12" {
		t.Fatalf("created.ShortName = %q, want %q", created.ShortName, "Школа 12")
	}
	if created.Address != "Львів" {
		t.Fatalf("created.Address = %q, want %q", created.Address, "Львів")
	}
	if created.Contract != "DOG-1" {
		t.Fatalf("created.Contract = %q, want %q", created.Contract, "DOG-1")
	}
	if created.TestIntervalMin != 15 {
		t.Fatalf("created.TestIntervalMin = %d, want 15", created.TestIntervalMin)
	}
}

func TestFrontendAdapterUpdateLegacyObjectMergesCurrentCard(t *testing.T) {
	admin := &frontendTestAdminMutator{
		currentCard: contracts.AdminObjectCard{
			ObjN:      1500,
			ShortName: "Старе ім'я",
			FullName:  "Старе повне ім'я",
			ObjTypeID: 2,
			ObjRegID:  3,
			Phones:    "0321234567",
			Contract:  "OLD",
			Address:   "Стара адреса",
			Location:  "Підвал",
		},
	}
	adapter := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendAdminObjectMutator(admin),
	)

	_, err := adapter.UpdateObject(context.Background(), contracts.FrontendObjectUpsertRequest{
		ObjectID: 1500,
		Core: contracts.FrontendObjectCoreFields{
			Address: "Нова адреса",
			Notes:   "оновлено",
		},
		Legacy: &contracts.FrontendLegacyObjectPayload{
			ShortName: "Нове ім'я",
		},
	})
	if err != nil {
		t.Fatalf("UpdateObject() error = %v", err)
	}
	if len(admin.updatedCards) != 1 {
		t.Fatalf("UpdateObject() updated %d cards, want 1", len(admin.updatedCards))
	}
	updated := admin.updatedCards[0]
	if updated.ObjN != 1500 {
		t.Fatalf("updated.ObjN = %d, want 1500", updated.ObjN)
	}
	if updated.ShortName != "Нове ім'я" {
		t.Fatalf("updated.ShortName = %q, want %q", updated.ShortName, "Нове ім'я")
	}
	if updated.FullName != "Старе повне ім'я" {
		t.Fatalf("updated.FullName = %q, want preserved value", updated.FullName)
	}
	if updated.Address != "Нова адреса" {
		t.Fatalf("updated.Address = %q, want %q", updated.Address, "Нова адреса")
	}
	if updated.Location != "Підвал" {
		t.Fatalf("updated.Location = %q, want preserved value", updated.Location)
	}
}

func TestFrontendAdapterCreateCASLObject(t *testing.T) {
	casl := &frontendTestCASLMutator{createID: "777"}
	adapter := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendCASLObjectMutator(casl),
	)

	result, err := adapter.CreateObject(context.Background(), contracts.FrontendObjectUpsertRequest{
		Source: contracts.FrontendSourceCASL,
		Core: contracts.FrontendObjectCoreFields{
			Name:        "CASL объект",
			Address:     "вул. Тестова, 1",
			Contract:    "CS-77",
			Description: "Основний корпус",
			Notes:       "черговий на місці",
			Latitude:    "49.84",
			Longitude:   "24.03",
		},
		CASL: &contracts.FrontendCASLObjectPayload{
			ManagerID:      "12",
			Status:         "active",
			ObjectType:     "fire",
			IDRequest:      "REQ-7",
			ReactingPultID: "3",
			StartDate:      1234567890,
			GeoZoneID:      44,
		},
	})
	if err != nil {
		t.Fatalf("CreateObject() error = %v", err)
	}
	if result.NativeID != "777" {
		t.Fatalf("CreateObject() NativeID = %q, want %q", result.NativeID, "777")
	}
	if len(casl.createCalls) != 1 {
		t.Fatalf("CreateObject() create calls = %d, want 1", len(casl.createCalls))
	}
	created := casl.createCalls[0]
	if created.Name != "CASL объект" {
		t.Fatalf("created.Name = %q, want %q", created.Name, "CASL объект")
	}
	if created.Lat != "49.84" || created.Long != "24.03" {
		t.Fatalf("created coordinates = %q/%q, want 49.84/24.03", created.Lat, created.Long)
	}
}

func TestFrontendAdapterUpdateCASLObjectUsesSnapshotFallbacks(t *testing.T) {
	objectID := ids.CASLObjectIDNamespaceStart + 42
	casl := &frontendTestCASLMutator{
		snapshot: contracts.CASLObjectEditorSnapshot{
			Object: contracts.CASLGuardObjectDetails{
				ObjID:          "42",
				Name:           "Поточний об'єкт",
				Address:        "Стара адреса",
				Lat:            "49.0",
				Long:           "24.0",
				Description:    "Опис",
				Contract:       "C-1",
				ManagerID:      "100",
				Note:           "Нотатка",
				StartDate:      1000,
				ObjectType:     "fire",
				IDRequest:      "REQ-1",
				ReactingPultID: "7",
				GeoZoneID:      9,
			},
		},
	}
	adapter := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendCASLObjectMutator(casl),
	)

	_, err := adapter.UpdateObject(context.Background(), contracts.FrontendObjectUpsertRequest{
		ObjectID: objectID,
		Core: contracts.FrontendObjectCoreFields{
			Address: "Нова адреса",
		},
		CASL: &contracts.FrontendCASLObjectPayload{
			Status: "blocked",
		},
	})
	if err != nil {
		t.Fatalf("UpdateObject() error = %v", err)
	}
	if len(casl.updateCalls) != 1 {
		t.Fatalf("UpdateObject() update calls = %d, want 1", len(casl.updateCalls))
	}
	updated := casl.updateCalls[0]
	if updated.ObjID != "42" {
		t.Fatalf("updated.ObjID = %q, want %q", updated.ObjID, "42")
	}
	if updated.Address != "Нова адреса" {
		t.Fatalf("updated.Address = %q, want %q", updated.Address, "Нова адреса")
	}
	if updated.Name != "Поточний об'єкт" {
		t.Fatalf("updated.Name = %q, want snapshot value", updated.Name)
	}
	if updated.Status != "blocked" {
		t.Fatalf("updated.Status = %q, want %q", updated.Status, "blocked")
	}
}

func TestFrontendAdapterCapabilitiesUsesProvider(t *testing.T) {
	adapter := NewFrontendAdapter(
		&frontendTestDataProvider{},
		WithFrontendSourceCapabilityProvider(&frontendTestCapabilityProvider{
			capabilities: []contracts.FrontendSourceCapability{
				{
					Source:            contracts.FrontendSourcePhoenix,
					DisplayName:       contracts.FrontendSourcePhoenix.DisplayName(),
					ReadObjects:       true,
					ReadObjectDetails: true,
					ReadEvents:        true,
					ReadAlarms:        true,
				},
			},
		}),
	)

	capabilities, err := adapter.Capabilities(context.Background())
	if err != nil {
		t.Fatalf("Capabilities() error = %v", err)
	}
	if len(capabilities.Sources) != 1 {
		t.Fatalf("Capabilities() sources = %d, want 1", len(capabilities.Sources))
	}
	if capabilities.Sources[0].Source != contracts.FrontendSourcePhoenix {
		t.Fatalf("Capabilities() source = %q, want %q", capabilities.Sources[0].Source, contracts.FrontendSourcePhoenix)
	}
}

func TestFrontendAdapterListObjectsNormalizesStateFields(t *testing.T) {
	adapter := NewFrontendAdapter(&frontendTestDataProvider{
		objects: []models.Object{
			{
				ID:                ids.CASLObjectIDNamespaceStart + 5,
				Name:              "CASL",
				Status:            models.StatusFault,
				GuardState:        0,
				IsConnState:       0,
				BlockedArmedOnOff: 1,
				HasAssignment:     false,
			},
		},
	})

	objects, err := adapter.ListObjects(context.Background())
	if err != nil {
		t.Fatalf("ListObjects() error = %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("len(ListObjects()) = %d, want 1", len(objects))
	}
	got := objects[0]
	if got.GuardStatus != contracts.FrontendGuardStatusDisarmed {
		t.Fatalf("GuardStatus = %q, want %q", got.GuardStatus, contracts.FrontendGuardStatusDisarmed)
	}
	if got.ConnectionStatus != contracts.FrontendConnectionStatusOffline {
		t.Fatalf("ConnectionStatus = %q, want %q", got.ConnectionStatus, contracts.FrontendConnectionStatusOffline)
	}
	if got.MonitoringStatus != contracts.FrontendMonitoringStatusBlocked {
		t.Fatalf("MonitoringStatus = %q, want %q", got.MonitoringStatus, contracts.FrontendMonitoringStatusBlocked)
	}
	if got.HasAssignment {
		t.Fatal("HasAssignment = true, want false")
	}
}

func TestFrontendAdapterListEventsAndAlarmsNormalizesVisualSeverity(t *testing.T) {
	adapter := NewFrontendAdapter(&frontendTestDataProvider{
		events: []models.Event{
			{ID: 1, ObjectID: 10, Type: models.EventOffline},
			{ID: 2, ObjectID: 10, Type: models.EventOperatorAction},
		},
		alarms: []models.Alarm{
			{ID: 3, ObjectID: 10, Type: models.AlarmFire},
			{ID: 4, ObjectID: 10, Type: models.AlarmNotification},
		},
	})

	events, err := adapter.ListEvents(context.Background())
	if err != nil {
		t.Fatalf("ListEvents() error = %v", err)
	}
	if events[0].VisualSeverity != contracts.FrontendVisualSeverityCritical {
		t.Fatalf("event severity = %q, want critical", events[0].VisualSeverity)
	}
	if events[1].VisualSeverity != contracts.FrontendVisualSeverityInfo {
		t.Fatalf("event severity = %q, want info", events[1].VisualSeverity)
	}

	alarms, err := adapter.ListAlarms(context.Background())
	if err != nil {
		t.Fatalf("ListAlarms() error = %v", err)
	}
	if alarms[0].VisualSeverity != contracts.FrontendVisualSeverityCritical {
		t.Fatalf("alarm severity = %q, want critical", alarms[0].VisualSeverity)
	}
	if alarms[1].VisualSeverity != contracts.FrontendVisualSeverityInfo {
		t.Fatalf("alarm severity = %q, want info", alarms[1].VisualSeverity)
	}
}

func TestFrontendAdapterErrorsWithoutPayload(t *testing.T) {
	adapter := NewFrontendAdapter(&frontendTestDataProvider{})

	_, err := adapter.CreateObject(context.Background(), contracts.FrontendObjectUpsertRequest{
		Source: contracts.FrontendSourceBridge,
	})
	if !errors.Is(err, contracts.ErrUnsupportedFrontendSource) && !errors.Is(err, contracts.ErrMissingLegacyObjectPayload) {
		t.Fatalf("CreateObject() error = %v, want missing payload or unsupported source", err)
	}
}
