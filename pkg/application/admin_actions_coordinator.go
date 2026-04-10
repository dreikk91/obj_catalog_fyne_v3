package application

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/dialog"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

type adminObjectScope interface {
	CanUseAdminForObjectID(objectID int) bool
	SourceNameForObjectID(objectID int) string
}

type adminObjectDeleteProvider interface {
	DeleteObject(objn int64) error
}

type adminDisplayBlockingProvider interface {
	ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error)
	SetDisplayBlockMode(objn int64, mode contracts.DisplayBlockMode) error
}

type adminEventEmulationProvider interface {
	ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error)
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
	ListMessageProtocols() ([]int64, error)
	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}

type adminEventOverrideProvider interface {
	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
	SetMessageAdminOnly(uin int64, adminOnly bool) error
	SetMessageCategory(uin int64, sc1 *int64) error
	List220VMessageBuckets(protocolIDs []int64, filter string) (contracts.Admin220VMessageBuckets, error)
	SetMessage220VMode(uin int64, mode contracts.Admin220VMode) error
}

type adminMessagesProvider interface {
	ListMessageProtocols() ([]int64, error)
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
	SetMessageAdminOnly(uin int64, adminOnly bool) error
}

type adminSystemControlProvider interface {
	GetAdminAccessStatus() (contracts.AdminAccessStatus, error)
	RunDataIntegrityChecks(limit int) ([]contracts.AdminDataCheckIssue, error)
}

type adminFireMonitoringProvider interface {
	GetFireMonitoringSettings() (contracts.FireMonitoringSettings, error)
	SaveFireMonitoringSettings(settings contracts.FireMonitoringSettings) error
}

type adminSubServerObjectsProvider interface {
	ListSubServers() ([]contracts.AdminSubServer, error)
	ListSubServerObjects(filter string) ([]contracts.AdminSubServerObject, error)
	SetObjectSubServer(objn int64, channel int, bind string) error
	ClearObjectSubServer(objn int64, channel int) error
}

type adminStatisticsProvider interface {
	CollectObjectStatistics(filter contracts.AdminStatisticsFilter, limit int) ([]contracts.AdminStatisticsRow, error)
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	ListObjectDistricts() ([]contracts.DictionaryItem, error)
}

type adminPPKConstructorProvider interface {
	AddPPKConstructor(name string, channel int64, zoneCount int64) error
	UpdatePPKConstructor(id int64, name string, channel int64, zoneCount int64) error
	DeletePPKConstructor(id int64) error
	ListPPKConstructor() ([]contracts.PPKConstructorItem, error)
}

type adminObjectTypesProvider interface {
	ListObjectTypes() ([]contracts.DictionaryItem, error)
	AddObjectType(name string) error
	UpdateObjectType(id int64, name string) error
	DeleteObjectType(id int64) error
}

type adminRegionsProvider interface {
	ListRegions() ([]contracts.DictionaryItem, error)
	AddRegion(name string, regionCode *int64) error
	UpdateRegion(id int64, name string, regionCode *int64) error
	DeleteRegion(id int64) error
}

type adminAlarmReasonsProvider interface {
	ListAlarmReasons() ([]contracts.DictionaryItem, error)
	AddAlarmReason(name string) error
	UpdateAlarmReason(id int64, name string) error
	DeleteAlarmReason(id int64) error
	MoveAlarmReason(id int64, direction int) error
}

func normalizeSourceLabel(value string) string {
	source := strings.TrimSpace(strings.ToLower(value))
	switch source {
	case "casl":
		return "CASL Cloud"
	case "primary":
		return "БД/МІСТ"
	default:
		if strings.TrimSpace(value) == "" {
			return "невідоме джерело"
		}
		return value
	}
}

func resolveAdminCapability[T any](a *Application) (T, bool) {
	var zero T
	adminProvider := a.resolveAdminProvider()
	if adminProvider == nil {
		return zero, false
	}
	capability, ok := any(adminProvider).(T)
	if !ok {
		return zero, false
	}
	return capability, true
}

func withAdminCapability[T any](a *Application, onReady func(T)) func() {
	return func() {
		adminProvider, ok := a.ensureAdminProviderAccess()
		if !ok {
			return
		}

		capability, ok := any(adminProvider).(T)
		if !ok {
			dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "Поточний провайдер даних не підтримує потрібну адмін-функцію.")
			return
		}

		onReady(capability)
	}
}

func (a *Application) ensureAdminProviderAccess() (contracts.AdminProvider, bool) {
	provider := a.getDataProvider()
	if a.currentObject != nil {
		if scopedProvider, ok := provider.(adminObjectScope); ok && !scopedProvider.CanUseAdminForObjectID(a.currentObject.ID) {
			source := normalizeSourceLabel(scopedProvider.SourceNameForObjectID(a.currentObject.ID))
			dialogs.ShowInfoDialog(
				a.mainWindow,
				"Недоступно для цього джерела",
				fmt.Sprintf("Адмін-операції недоступні для джерела \"%s\". Використовуйте окреме меню CASL.", source),
			)
			return nil, false
		}
	}

	adminProvider, ok := backend.AsAdminProvider(provider)
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "Поточний провайдер даних не підтримує адмінські функції.")
		return nil, false
	}

	access, err := adminProvider.GetAdminAccessStatus()
	if err != nil {
		dialogs.ShowErrorDialog(a.mainWindow, "Помилка перевірки прав доступу", err)
		return nil, false
	}
	if !access.HasFullAccess {
		userLabel := strings.TrimSpace(access.CurrentUser)
		if userLabel == "" {
			userLabel = "невизначений користувач"
		}
		msg := fmt.Sprintf(
			"Користувач \"%s\" не має повного доступу до адмін-функцій.\n\nПотрібно, щоб у таблиці PERSONAL був запис користувача з ACCESS1=1.\nАдмін-записів у PERSONAL: %d.",
			userLabel,
			access.AdminUsersCount,
		)
		dialogs.ShowInfoDialog(a.mainWindow, "Доступ обмежено", msg)
		return nil, false
	}

	return adminProvider, true
}

func (a *Application) ensureCurrentObjectSelected() (id int64, name string, ok bool) {
	if a.currentObject == nil || a.currentObject.ID <= 0 {
		dialogs.ShowInfoDialog(a.mainWindow, "Об'єкт не вибрано", "Виберіть об'єкт у сітці, а потім спробуйте знову.")
		return 0, "", false
	}
	if scopedProvider, ok := a.getDataProvider().(adminObjectScope); ok {
		if !scopedProvider.CanUseAdminForObjectID(a.currentObject.ID) {
			source := normalizeSourceLabel(scopedProvider.SourceNameForObjectID(a.currentObject.ID))
			dialogs.ShowInfoDialog(
				a.mainWindow,
				"Недоступно для цього джерела",
				fmt.Sprintf("Для об'єкта з джерела \"%s\" адмін-операції недоступні.", source),
			)
			return 0, "", false
		}
	}
	return int64(a.currentObject.ID), a.currentObject.Name, true
}

func (a *Application) openNewObjectDialog(admin contracts.AdminObjectWizardProvider) {
	dialogs.ShowNewObjectDialog(a.mainWindow, admin, func(objn int64) {
		a.publishObjectSaved(objn)
	})
}

func (a *Application) openEditCurrentObjectDialog(admin contracts.AdminObjectCardProvider) {
	objectID, _, ok := a.ensureCurrentObjectSelected()
	if !ok {
		return
	}
	dialogs.ShowEditObjectDialog(a.mainWindow, admin, objectID, func(objn int64) {
		a.publishObjectSaved(objn)
	})
}

func (a *Application) confirmDeleteCurrentObject(admin adminObjectDeleteProvider) {
	objectID, objectName, ok := a.ensureCurrentObjectSelected()
	if !ok {
		return
	}

	dialog.ShowConfirm(
		"Підтвердження видалення",
		fmt.Sprintf("Видалити об'єкт №%d \"%s\"?", objectID, objectName),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := admin.DeleteObject(objectID); err != nil {
				dialogs.ShowErrorDialog(a.mainWindow, "Помилка видалення об'єкта", err)
				return
			}
			a.publishObjectDeleted(objectID)
			dialogs.ShowInfoDialog(a.mainWindow, "Готово", "Об'єкт видалено")
		},
		a.mainWindow,
	)
}
