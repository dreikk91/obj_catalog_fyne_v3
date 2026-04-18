package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1DictionaryPPKStub struct {
	objectTypes []contracts.DictionaryItem
	regions     []contracts.DictionaryItem
	reasons     []contracts.DictionaryItem
	ppkItems    []contracts.PPKConstructorItem

	objectTypeName string
	objectTypeID   int64
	regionName     string
	regionID       int64
	regionCode     *int64
	reasonName     string
	reasonID       int64
	moveID         int64
	moveDirection  int
	ppkID          int64
	ppkName        string
	ppkChannel     int64
	ppkZoneCount   int64
}

func (s *adminV1DictionaryPPKStub) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return s.objectTypes, nil
}
func (s *adminV1DictionaryPPKStub) AddObjectType(name string) error {
	s.objectTypeName = name
	return nil
}
func (s *adminV1DictionaryPPKStub) UpdateObjectType(id int64, name string) error {
	s.objectTypeID, s.objectTypeName = id, name
	return nil
}
func (s *adminV1DictionaryPPKStub) DeleteObjectType(id int64) error {
	s.objectTypeID = id
	return nil
}
func (s *adminV1DictionaryPPKStub) ListRegions() ([]contracts.DictionaryItem, error) {
	return s.regions, nil
}
func (s *adminV1DictionaryPPKStub) AddRegion(name string, regionCode *int64) error {
	s.regionName, s.regionCode = name, regionCode
	return nil
}
func (s *adminV1DictionaryPPKStub) UpdateRegion(id int64, name string, regionCode *int64) error {
	s.regionID, s.regionName, s.regionCode = id, name, regionCode
	return nil
}
func (s *adminV1DictionaryPPKStub) DeleteRegion(id int64) error { s.regionID = id; return nil }
func (s *adminV1DictionaryPPKStub) ListAlarmReasons() ([]contracts.DictionaryItem, error) {
	return s.reasons, nil
}
func (s *adminV1DictionaryPPKStub) AddAlarmReason(name string) error { s.reasonName = name; return nil }
func (s *adminV1DictionaryPPKStub) UpdateAlarmReason(id int64, name string) error {
	s.reasonID, s.reasonName = id, name
	return nil
}
func (s *adminV1DictionaryPPKStub) DeleteAlarmReason(id int64) error { s.reasonID = id; return nil }
func (s *adminV1DictionaryPPKStub) MoveAlarmReason(id int64, direction int) error {
	s.moveID, s.moveDirection = id, direction
	return nil
}
func (s *adminV1DictionaryPPKStub) AddPPKConstructor(name string, channel int64, zoneCount int64) error {
	s.ppkName, s.ppkChannel, s.ppkZoneCount = name, channel, zoneCount
	return nil
}
func (s *adminV1DictionaryPPKStub) UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error {
	s.ppkID, s.ppkName, s.ppkChannel, s.ppkZoneCount = id, name, channel, zoneCount
	return nil
}
func (s *adminV1DictionaryPPKStub) DeletePPKConstructor(id int64) error { s.ppkID = id; return nil }
func (s *adminV1DictionaryPPKStub) ListPPKConstructor() ([]contracts.PPKConstructorItem, error) {
	return s.ppkItems, nil
}

func TestAdminV1DictionaryProviders(t *testing.T) {
	code := int64(42)
	base := &adminV1DictionaryPPKStub{
		objectTypes: []contracts.DictionaryItem{{ID: 1, Name: "Type"}},
		regions:     []contracts.DictionaryItem{{ID: 2, Name: "Region", Code: &code}},
		reasons:     []contracts.DictionaryItem{{ID: 3, Name: "Reason"}},
	}

	objectTypes := NewAdminV1ObjectTypesDictionaryProvider(base)
	items, err := objectTypes.ListObjectTypes()
	if err != nil || len(items) != 1 || items[0].Name != "Type" {
		t.Fatalf("object types = %+v, err=%v", items, err)
	}
	if err := objectTypes.AddObjectType("New"); err != nil || base.objectTypeName != "New" {
		t.Fatalf("AddObjectType err=%v name=%q", err, base.objectTypeName)
	}

	regions := NewAdminV1RegionsDictionaryProvider(base)
	rItems, err := regions.ListRegions()
	if err != nil || len(rItems) != 1 || rItems[0].Code == nil || *rItems[0].Code != 42 {
		t.Fatalf("regions = %+v, err=%v", rItems, err)
	}

	reasons := NewAdminV1AlarmReasonsDictionaryProvider(base)
	if err := reasons.MoveAlarmReason(9, -1); err != nil || base.moveID != 9 || base.moveDirection != -1 {
		t.Fatalf("MoveAlarmReason err=%v id=%d direction=%d", err, base.moveID, base.moveDirection)
	}
}

func TestAdminV1PPKConstructorProvider(t *testing.T) {
	base := &adminV1DictionaryPPKStub{
		ppkItems: []contracts.PPKConstructorItem{{ID: 5, Name: "PPK", Channel: 7, ZoneCount: 16}},
	}
	provider := NewAdminV1PPKConstructorProvider(base)

	items, err := provider.ListPPKConstructor()
	if err != nil || len(items) != 1 || items[0].Channel != 7 {
		t.Fatalf("ppk items = %+v, err=%v", items, err)
	}
	if err := provider.UpdatePPKConstructor(5, "New", 9, 24); err != nil {
		t.Fatalf("UpdatePPKConstructor err=%v", err)
	}
	if base.ppkID != 5 || base.ppkName != "New" || base.ppkChannel != 9 || base.ppkZoneCount != 24 {
		t.Fatalf("ppk update captured wrong values: %+v", base)
	}
}

var (
	_ adminv1.ObjectTypesDictionaryProvider  = (*adminV1ObjectTypesDictionaryAdapter)(nil)
	_ adminv1.RegionsDictionaryProvider      = (*adminV1RegionsDictionaryAdapter)(nil)
	_ adminv1.AlarmReasonsDictionaryProvider = (*adminV1AlarmReasonsDictionaryAdapter)(nil)
	_ adminv1.PPKConstructorProvider         = (*adminV1PPKConstructorAdapter)(nil)
)
