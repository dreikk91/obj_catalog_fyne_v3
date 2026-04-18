package backend

import (
	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1FireMonitoringBase interface {
	GetFireMonitoringSettings() (contracts.FireMonitoringSettings, error)
	SaveFireMonitoringSettings(settings contracts.FireMonitoringSettings) error
}

type adminV1FireMonitoringAdapter struct {
	base adminV1FireMonitoringBase
}

func NewAdminV1FireMonitoringProvider(base adminV1FireMonitoringBase) adminv1.FireMonitoringProvider {
	if base == nil {
		return nil
	}
	return &adminV1FireMonitoringAdapter{base: base}
}

func (a *adminV1FireMonitoringAdapter) GetFireMonitoringSettings() (adminv1.FireMonitoringSettings, error) {
	settings, err := a.base.GetFireMonitoringSettings()
	if err != nil {
		return adminv1.FireMonitoringSettings{}, err
	}
	return adminv1.ToFireMonitoringSettings(settings), nil
}

func (a *adminV1FireMonitoringAdapter) SaveFireMonitoringSettings(settings adminv1.FireMonitoringSettings) error {
	return a.base.SaveFireMonitoringSettings(adminv1.ToContractsFireMonitoringSettings(settings))
}
