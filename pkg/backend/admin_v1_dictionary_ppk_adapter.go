package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1ObjectTypesDictionaryBase interface {
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	AddObjectType(name string) error
	UpdateObjectType(id int64, name string) error
	DeleteObjectType(id int64) error
}

type adminV1RegionsDictionaryBase interface {
	ListRegions() ([]contracts.DictionaryItem, error)
	AddRegion(name string, regionCode *int64) error
	UpdateRegion(id int64, name string, regionCode *int64) error
	DeleteRegion(id int64) error
}

type adminV1AlarmReasonsDictionaryBase interface {
	ListAlarmReasons() ([]contracts.DictionaryItem, error)
	AddAlarmReason(name string) error
	UpdateAlarmReason(id int64, name string) error
	DeleteAlarmReason(id int64) error
	MoveAlarmReason(id int64, direction int) error
}

type adminV1PPKConstructorBase interface {
	AddPPKConstructor(name string, channel int64, zoneCount int64) error
	UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error
	DeletePPKConstructor(id int64) error
	ListPPKConstructor() ([]contracts.PPKConstructorItem, error)
}

type adminV1ObjectTypesDictionaryAdapter struct {
	base adminV1ObjectTypesDictionaryBase
}
type adminV1RegionsDictionaryAdapter struct{ base adminV1RegionsDictionaryBase }
type adminV1AlarmReasonsDictionaryAdapter struct {
	base adminV1AlarmReasonsDictionaryBase
}
type adminV1PPKConstructorAdapter struct{ base adminV1PPKConstructorBase }

func NewAdminV1ObjectTypesDictionaryProvider(base adminV1ObjectTypesDictionaryBase) adminv1.ObjectTypesDictionaryProvider {
	if base == nil {
		return nil
	}
	return &adminV1ObjectTypesDictionaryAdapter{base: base}
}

func NewAdminV1RegionsDictionaryProvider(base adminV1RegionsDictionaryBase) adminv1.RegionsDictionaryProvider {
	if base == nil {
		return nil
	}
	return &adminV1RegionsDictionaryAdapter{base: base}
}

func NewAdminV1AlarmReasonsDictionaryProvider(base adminV1AlarmReasonsDictionaryBase) adminv1.AlarmReasonsDictionaryProvider {
	if base == nil {
		return nil
	}
	return &adminV1AlarmReasonsDictionaryAdapter{base: base}
}

func NewAdminV1PPKConstructorProvider(base adminV1PPKConstructorBase) adminv1.PPKConstructorProvider {
	if base == nil {
		return nil
	}
	return &adminV1PPKConstructorAdapter{base: base}
}

func (a *adminV1ObjectTypesDictionaryAdapter) ListObjectTypes() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectTypes()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1ObjectTypesDictionaryAdapter) AddObjectType(name string) error {
	return a.base.AddObjectType(name)
}

func (a *adminV1ObjectTypesDictionaryAdapter) UpdateObjectType(id int64, name string) error {
	return a.base.UpdateObjectType(id, name)
}

func (a *adminV1ObjectTypesDictionaryAdapter) DeleteObjectType(id int64) error {
	return a.base.DeleteObjectType(id)
}

func (a *adminV1RegionsDictionaryAdapter) ListRegions() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListRegions()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1RegionsDictionaryAdapter) AddRegion(name string, regionCode *int64) error {
	return a.base.AddRegion(name, regionCode)
}

func (a *adminV1RegionsDictionaryAdapter) UpdateRegion(id int64, name string, regionCode *int64) error {
	return a.base.UpdateRegion(id, name, regionCode)
}

func (a *adminV1RegionsDictionaryAdapter) DeleteRegion(id int64) error {
	return a.base.DeleteRegion(id)
}

func (a *adminV1AlarmReasonsDictionaryAdapter) ListAlarmReasons() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListAlarmReasons()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1AlarmReasonsDictionaryAdapter) AddAlarmReason(name string) error {
	return a.base.AddAlarmReason(name)
}

func (a *adminV1AlarmReasonsDictionaryAdapter) UpdateAlarmReason(id int64, name string) error {
	return a.base.UpdateAlarmReason(id, name)
}

func (a *adminV1AlarmReasonsDictionaryAdapter) DeleteAlarmReason(id int64) error {
	return a.base.DeleteAlarmReason(id)
}

func (a *adminV1AlarmReasonsDictionaryAdapter) MoveAlarmReason(id int64, direction int) error {
	return a.base.MoveAlarmReason(id, direction)
}

func (a *adminV1PPKConstructorAdapter) AddPPKConstructor(name string, channel int64, zoneCount int64) error {
	return a.base.AddPPKConstructor(name, channel, zoneCount)
}

func (a *adminV1PPKConstructorAdapter) UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error {
	return a.base.UpdatePPKConstructor(id, name, channel, zoneCount)
}

func (a *adminV1PPKConstructorAdapter) DeletePPKConstructor(id int64) error {
	return a.base.DeletePPKConstructor(id)
}

func (a *adminV1PPKConstructorAdapter) ListPPKConstructor() ([]adminv1.PPKConstructorItem, error) {
	items, err := a.base.ListPPKConstructor()
	if err != nil {
		return nil, err
	}
	return adminv1.ToPPKConstructorItems(items), nil
}
