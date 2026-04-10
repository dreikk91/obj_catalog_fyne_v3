package application

import (
	"context"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

type simInventoryStatisticProvider interface {
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
}

type simInventoryObjectProvider interface {
	GetObjects() []models.Object
	GetObjectByID(id string) *models.Object
}

type simInventoryAdminProvider interface {
	GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error)
	GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error)
}

type simInventoryVodafoneListProvider interface {
	ListVodafoneSIMInventory() (map[string]contracts.VodafoneSIMInventoryEntry, error)
}

type simInventoryKyivstarListProvider interface {
	ListKyivstarSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error)
}

type appSIMInventoryReportProvider struct {
	objects              simInventoryObjectProvider
	reporter             simInventoryStatisticProvider
	hasCASLReports       bool
	admin                simInventoryAdminProvider
	vodafoneInventoryAPI simInventoryVodafoneListProvider
	kyivstarInventoryAPI simInventoryKyivstarListProvider
}

func (p appSIMInventoryReportProvider) GetObjects() []models.Object {
	return p.objects.GetObjects()
}

func (p appSIMInventoryReportProvider) GetObjectByID(id string) *models.Object {
	return p.objects.GetObjectByID(id)
}

func (p appSIMInventoryReportProvider) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	if p.reporter == nil {
		return nil, nil
	}
	return p.reporter.GetStatisticReport(ctx, name, limit)
}

func (p appSIMInventoryReportProvider) GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error) {
	return p.admin.GetVodafoneSIMStatus(msisdn)
}

func (p appSIMInventoryReportProvider) GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error) {
	return p.admin.GetKyivstarSIMStatus(msisdn)
}

func (p appSIMInventoryReportProvider) SupportsCASLReports() bool {
	return p.hasCASLReports
}

func (p appSIMInventoryReportProvider) ListVodafoneSIMInventory() (map[string]contracts.VodafoneSIMInventoryEntry, error) {
	if p.vodafoneInventoryAPI == nil {
		return nil, nil
	}
	return p.vodafoneInventoryAPI.ListVodafoneSIMInventory()
}

func (p appSIMInventoryReportProvider) ListKyivstarSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error) {
	if p.kyivstarInventoryAPI == nil {
		return nil, nil
	}
	return p.kyivstarInventoryAPI.ListKyivstarSIMInventory(numbers)
}

func (a *Application) resolveSIMInventoryReportProvider() (dialogs.SIMInventoryReportProvider, bool) {
	provider := a.getDataProvider()
	if provider == nil {
		return nil, false
	}

	objects, ok := provider.(simInventoryObjectProvider)
	if !ok {
		return nil, false
	}
	reporter, _ := provider.(simInventoryStatisticProvider)
	admin, ok := resolveAdminCapability[simInventoryAdminProvider](a)
	if !ok {
		return nil, false
	}
	vodafoneInventoryAPI, _ := resolveAdminCapability[simInventoryVodafoneListProvider](a)
	kyivstarInventoryAPI, _ := resolveAdminCapability[simInventoryKyivstarListProvider](a)

	return appSIMInventoryReportProvider{
		objects:              objects,
		reporter:             reporter,
		hasCASLReports:       reporter != nil,
		admin:                admin,
		vodafoneInventoryAPI: vodafoneInventoryAPI,
		kyivstarInventoryAPI: kyivstarInventoryAPI,
	}, true
}

func (a *Application) openSIMInventoryReportDialog() {
	provider, ok := a.resolveSIMInventoryReportProvider()
	if !ok {
		dialogs.ShowInfoDialog(
			a.mainWindow,
			"Недоступно",
			"Звіт по SIM-картах недоступний. Потрібні джерела об'єктів та активний адмін-провайдер для Vodafone/Kyivstar.",
		)
		return
	}

	dialogs.ShowSIMInventoryReportDialog(a.mainWindow, provider)
}
