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

func (a *Application) withAdminProvider(onReady func(contracts.AdminProvider)) func() {
	return func() {
		provider := a.getDataProvider()
		if a.currentObject != nil {
			if scopedProvider, ok := provider.(adminObjectScope); ok && !scopedProvider.CanUseAdminForObjectID(a.currentObject.ID) {
				source := normalizeSourceLabel(scopedProvider.SourceNameForObjectID(a.currentObject.ID))
				dialogs.ShowInfoDialog(
					a.mainWindow,
					"Недоступно для цього джерела",
					fmt.Sprintf("Адмін-операції недоступні для джерела \"%s\". Використовуйте окреме меню CASL.", source),
				)
				return
			}
		}
		adminProvider, ok := backend.AsAdminProvider(provider)
		if !ok {
			dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "Поточний провайдер даних не підтримує адмінські функції.")
			return
		}
		access, err := adminProvider.GetAdminAccessStatus()
		if err != nil {
			dialogs.ShowErrorDialog(a.mainWindow, "Помилка перевірки прав доступу", err)
			return
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
			return
		}
		onReady(adminProvider)
	}
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

func (a *Application) openNewObjectDialog(admin contracts.AdminProvider) {
	dialogs.ShowNewObjectDialog(a.mainWindow, admin, func(objn int64) {
		a.publishObjectSaved(objn)
	})
}

func (a *Application) openEditCurrentObjectDialog(admin contracts.AdminProvider) {
	objectID, _, ok := a.ensureCurrentObjectSelected()
	if !ok {
		return
	}
	dialogs.ShowEditObjectDialog(a.mainWindow, admin, objectID, func(objn int64) {
		a.publishObjectSaved(objn)
	})
}

func (a *Application) confirmDeleteCurrentObject(admin contracts.AdminProvider) {
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
