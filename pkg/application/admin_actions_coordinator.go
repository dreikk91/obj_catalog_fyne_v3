package application

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2/dialog"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

func (a *Application) withAdminProvider(onReady func(contracts.AdminProvider)) func() {
	return func() {
		adminProvider, ok := a.dataProvider.(contracts.AdminProvider)
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
