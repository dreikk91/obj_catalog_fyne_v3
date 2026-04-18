package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1ObjectStub struct {
	card       contracts.AdminObjectCard
	personals  []contracts.AdminObjectPersonal
	zones      []contracts.AdminObjectZone
	coords     contracts.AdminObjectCoordinates
	subServers []contracts.AdminSubServer
	ppkItems   []contracts.PPKConstructorItem
	types      []contracts.DictionaryItem
	districts  []contracts.DictionaryItem
	simUsages  []contracts.AdminSIMPhoneUsage

	lastCreatedCard   contracts.AdminObjectCard
	lastUpdatedCard   contracts.AdminObjectCard
	lastAddedPerson   contracts.AdminObjectPersonal
	lastUpdatedPerson contracts.AdminObjectPersonal
	lastAddedZone     contracts.AdminObjectZone
	lastUpdatedZone   contracts.AdminObjectZone
	lastSavedCoords   contracts.AdminObjectCoordinates
	deletedPersonalID int64
	deletedZoneID     int64
	deletedObjectID   int64
	fillCount         int64
	clearedZones      bool
}

func (s *adminV1ObjectStub) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return s.types, nil
}
func (s *adminV1ObjectStub) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	return s.districts, nil
}
func (s *adminV1ObjectStub) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	return s.ppkItems, nil
}
func (s *adminV1ObjectStub) ListSubServers() ([]contracts.AdminSubServer, error) {
	return s.subServers, nil
}
func (s *adminV1ObjectStub) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	return s.card, nil
}
func (s *adminV1ObjectStub) CreateObject(card contracts.AdminObjectCard) error {
	s.lastCreatedCard = card
	return nil
}
func (s *adminV1ObjectStub) UpdateObject(card contracts.AdminObjectCard) error {
	s.lastUpdatedCard = card
	return nil
}
func (s *adminV1ObjectStub) DeleteObject(objn int64) error {
	s.deletedObjectID = objn
	return nil
}
func (s *adminV1ObjectStub) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]contracts.AdminSIMPhoneUsage, error) {
	return s.simUsages, nil
}
func (s *adminV1ObjectStub) ListObjectPersonals(objn int64) ([]contracts.AdminObjectPersonal, error) {
	return s.personals, nil
}
func (s *adminV1ObjectStub) AddObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	s.lastAddedPerson = item
	return nil
}
func (s *adminV1ObjectStub) UpdateObjectPersonal(objn int64, item contracts.AdminObjectPersonal) error {
	s.lastUpdatedPerson = item
	return nil
}
func (s *adminV1ObjectStub) DeleteObjectPersonal(objn int64, personalID int64) error {
	s.deletedPersonalID = personalID
	return nil
}
func (s *adminV1ObjectStub) FindPersonalByPhone(phone string) (*contracts.AdminObjectPersonal, error) {
	if len(s.personals) == 0 {
		return nil, nil
	}
	item := s.personals[0]
	return &item, nil
}
func (s *adminV1ObjectStub) ListObjectZones(objn int64) ([]contracts.AdminObjectZone, error) {
	return s.zones, nil
}
func (s *adminV1ObjectStub) AddObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	s.lastAddedZone = zone
	return nil
}
func (s *adminV1ObjectStub) UpdateObjectZone(objn int64, zone contracts.AdminObjectZone) error {
	s.lastUpdatedZone = zone
	return nil
}
func (s *adminV1ObjectStub) DeleteObjectZone(objn int64, zoneID int64) error {
	s.deletedZoneID = zoneID
	return nil
}
func (s *adminV1ObjectStub) FillObjectZones(objn int64, count int64) error {
	s.fillCount = count
	return nil
}
func (s *adminV1ObjectStub) ClearObjectZones(objn int64) error {
	s.clearedZones = true
	return nil
}
func (s *adminV1ObjectStub) GetObjectCoordinates(objn int64) (contracts.AdminObjectCoordinates, error) {
	return s.coords, nil
}
func (s *adminV1ObjectStub) SaveObjectCoordinates(objn int64, coords contracts.AdminObjectCoordinates) error {
	s.lastSavedCoords = coords
	return nil
}

func TestAdminV1ObjectWizardProvider(t *testing.T) {
	base := &adminV1ObjectStub{
		types:      []contracts.DictionaryItem{{ID: 1, Name: "School"}},
		districts:  []contracts.DictionaryItem{{ID: 2, Name: "Lviv"}},
		ppkItems:   []contracts.PPKConstructorItem{{ID: 3, Name: "PPK", ZoneCount: 16}},
		subServers: []contracts.AdminSubServer{{ID: 4, Bind: "srv-a"}},
		simUsages:  []contracts.AdminSIMPhoneUsage{{ObjN: 1001, Slot: "SIM1"}},
		personals:  []contracts.AdminObjectPersonal{{ID: 5, Name: "Ivan"}},
	}
	provider := NewAdminV1ObjectWizardProvider(base)

	typeItems, err := provider.ListObjectTypes()
	if err != nil || len(typeItems) != 1 || typeItems[0].Name != "School" {
		t.Fatalf("ListObjectTypes = %+v, err=%v", typeItems, err)
	}
	if usages, err := provider.FindObjectsBySIMPhone("38067", nil); err != nil || len(usages) != 1 || usages[0].Slot != "SIM1" {
		t.Fatalf("FindObjectsBySIMPhone = %+v, err=%v", usages, err)
	}
	if err := provider.CreateObject(adminv1.ObjectCard{ObjN: 1001, ShortName: "School"}); err != nil {
		t.Fatalf("CreateObject err=%v", err)
	}
	if base.lastCreatedCard.ObjN != 1001 || base.lastCreatedCard.ShortName != "School" {
		t.Fatalf("lastCreatedCard = %+v", base.lastCreatedCard)
	}
	if found, err := provider.FindPersonalByPhone("38067"); err != nil || found == nil || found.Name != "Ivan" {
		t.Fatalf("FindPersonalByPhone = %+v, err=%v", found, err)
	}
}

func TestAdminV1ObjectCardProvider(t *testing.T) {
	base := &adminV1ObjectStub{
		card:      contracts.AdminObjectCard{ObjN: 1002, ShortName: "Warehouse", ObjTypeID: 7},
		personals: []contracts.AdminObjectPersonal{{ID: 9, Surname: "Petrenko"}},
		zones:     []contracts.AdminObjectZone{{ID: 8, ZoneNumber: 4, Description: "Hall"}},
		coords:    contracts.AdminObjectCoordinates{Latitude: "49.1", Longitude: "24.1"},
	}
	provider := NewAdminV1ObjectCardProvider(base)

	card, err := provider.GetObjectCard(1002)
	if err != nil || card.ShortName != "Warehouse" || card.ObjTypeID != 7 {
		t.Fatalf("GetObjectCard = %+v, err=%v", card, err)
	}
	if err := provider.UpdateObject(adminv1.ObjectCard{ObjN: 1002, ShortName: "Warehouse 2"}); err != nil {
		t.Fatalf("UpdateObject err=%v", err)
	}
	if base.lastUpdatedCard.ShortName != "Warehouse 2" {
		t.Fatalf("lastUpdatedCard = %+v", base.lastUpdatedCard)
	}
	if zones, err := provider.ListObjectZones(1002); err != nil || len(zones) != 1 || zones[0].Description != "Hall" {
		t.Fatalf("ListObjectZones = %+v, err=%v", zones, err)
	}
	if err := provider.UpdateObjectZone(1002, adminv1.ObjectZone{ZoneNumber: 5, Description: "Office"}); err != nil {
		t.Fatalf("UpdateObjectZone err=%v", err)
	}
	if base.lastUpdatedZone.ZoneNumber != 5 || base.lastUpdatedZone.Description != "Office" {
		t.Fatalf("lastUpdatedZone = %+v", base.lastUpdatedZone)
	}
	if err := provider.SaveObjectCoordinates(1002, adminv1.ObjectCoordinates{Latitude: "50.0", Longitude: "25.0"}); err != nil {
		t.Fatalf("SaveObjectCoordinates err=%v", err)
	}
	if base.lastSavedCoords.Latitude != "50.0" || base.lastSavedCoords.Longitude != "25.0" {
		t.Fatalf("lastSavedCoords = %+v", base.lastSavedCoords)
	}
}

func TestAdminV1ObjectDeleteProvider(t *testing.T) {
	base := &adminV1ObjectStub{}
	provider := NewAdminV1ObjectDeleteProvider(base)

	if err := provider.DeleteObject(1003); err != nil {
		t.Fatalf("DeleteObject err=%v", err)
	}
	if base.deletedObjectID != 1003 {
		t.Fatalf("deletedObjectID = %d, want 1003", base.deletedObjectID)
	}
}

var (
	_ adminv1.ObjectWizardProvider = (*adminV1ObjectWizardAdapter)(nil)
	_ adminv1.ObjectCardProvider   = (*adminV1ObjectCardAdapter)(nil)
	_ adminv1.ObjectDeleteProvider = (*adminV1ObjectDeleteAdapter)(nil)
)
