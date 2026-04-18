package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1StatisticsBase interface {
	CollectObjectStatistics(filter contracts.AdminStatisticsFilter, limit int) ([]contracts.AdminStatisticsRow, error)
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	ListObjectDistricts() ([]contracts.DictionaryItem, error)
}

type adminV1DisplayBlockingBase interface {
	ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode contracts.DisplayBlockMode) error
}

type adminV1StatisticsAdapter struct {
	base adminV1StatisticsBase
}

type adminV1DisplayBlockingAdapter struct {
	base adminV1DisplayBlockingBase
}

func NewAdminV1StatisticsProvider(base adminV1StatisticsBase) adminv1.StatisticsProvider {
	if base == nil {
		return nil
	}
	return &adminV1StatisticsAdapter{base: base}
}

func NewAdminV1DisplayBlockingProvider(base adminV1DisplayBlockingBase) adminv1.DisplayBlockingProvider {
	if base == nil {
		return nil
	}
	return &adminV1DisplayBlockingAdapter{base: base}
}

func (a *adminV1StatisticsAdapter) CollectObjectStatistics(filter adminv1.StatisticsFilter, limit int) ([]adminv1.StatisticsRow, error) {
	rows, err := a.base.CollectObjectStatistics(adminv1.ToStatisticsFilter(filter), limit)
	if err != nil {
		return nil, err
	}
	return adminv1.ToStatisticsRows(rows), nil
}

func (a *adminV1StatisticsAdapter) ListObjectTypes() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectTypes()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1StatisticsAdapter) ListObjectDistricts() ([]adminv1.DictionaryItem, error) {
	items, err := a.base.ListObjectDistricts()
	if err != nil {
		return nil, err
	}
	return adminv1.ToDictionaryItems(items), nil
}

func (a *adminV1DisplayBlockingAdapter) ListDisplayBlockObjects(filter string) ([]adminv1.DisplayBlockObject, error) {
	items, err := a.base.ListDisplayBlockObjects(filter)
	if err != nil {
		return nil, err
	}
	return adminv1.ToDisplayBlockObjects(items), nil
}

func (a *adminV1DisplayBlockingAdapter) SetDisplayBlockMode(objn int64, mode adminv1.DisplayBlockMode) error {
	return a.base.SetDisplayBlockMode(objn, adminv1.ToContractsDisplayBlockMode(mode))
}
