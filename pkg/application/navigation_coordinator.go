package application

import (
	"strconv"

	"obj_catalog_fyne_v3/pkg/models"
)

// resolveObjectByID повертає об'єкт за ID через поточний провайдер даних.
func (a *Application) resolveObjectByID(objectID int64) *models.Object {
	if a == nil || a.dataProvider == nil {
		return nil
	}
	return a.dataProvider.GetObjectByID(strconv.FormatInt(objectID, 10))
}

// applyObjectContext синхронізує стан заголовка, правої панелі й контекстного журналу.
func (a *Application) applyObjectContext(obj *models.Object, showDetailsTab bool) {
	if a == nil || obj == nil {
		return
	}

	a.currentObject = obj
	a.updateWindowTitle()

	if a.workArea != nil {
		a.workArea.SetObject(*obj)
	}
	if a.eventLog != nil {
		a.eventLog.SetCurrentObject(obj)
	}
	if showDetailsTab && a.rightTabs != nil {
		a.rightTabs.SelectIndex(0)
	}
}

// applyObjectContextByID знаходить об'єкт та застосовує його контекст до UI.
func (a *Application) applyObjectContextByID(objectID int64, showDetailsTab bool) {
	obj := a.resolveObjectByID(objectID)
	if obj == nil {
		return
	}
	a.applyObjectContext(obj, showDetailsTab)
}

func (a *Application) clearObjectContext() {
	if a == nil {
		return
	}
	a.currentObject = nil
	a.updateWindowTitle()
	if a.eventLog != nil {
		a.eventLog.SetCurrentObject(nil)
	}
}

func (a *Application) selectDetailsTab() {
	if a == nil || a.rightTabs == nil {
		return
	}
	a.rightTabs.SelectIndex(0)
}
