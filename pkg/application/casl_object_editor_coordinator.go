package application

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/eventbus"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
)

func (a *Application) resolveCASLObjectEditorProvider() (contracts.CASLObjectEditorProvider, bool) {
	provider := a.getDataProvider()
	if provider == nil {
		return nil, false
	}
	editor, ok := provider.(contracts.CASLObjectEditorProvider)
	if !ok {
		return nil, false
	}
	return editor, true
}

func (a *Application) openCASLObjectEditor() {
	objectID, ok := a.currentCASLObjectID()
	if !ok {
		return
	}

	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-редактор недоступний у поточній конфігурації.")
		return
	}

	dialogs.ShowCASLObjectEditorDialog(a.mainWindow, editor, objectID, a.refreshCASLObjectData)
}

func (a *Application) openCASLObjectCreator() {
	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-редактор недоступний у поточній конфігурації.")
		return
	}

	dialogs.ShowCASLObjectEditorDialog(a.mainWindow, editor, 0, func() {
		a.refreshCASLObjectData()
	})
}

func (a *Application) openCASLObjectDeleteDialog() {
	objectID, ok := a.currentCASLObjectID()
	if !ok {
		return
	}
	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-меню недоступне у поточній конфігурації.")
		return
	}
	dialogs.ShowCASLObjectDeleteDialog(a.mainWindow, editor, objectID, a.refreshCASLObjectData)
}

func (a *Application) openCASLObjectBasketDialog() {
	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-меню недоступне у поточній конфігурації.")
		return
	}
	dialogs.ShowCASLObjectBasketDialog(a.mainWindow, editor)
}

func (a *Application) openCASLObjectBlockDialog() {
	objectID, ok := a.currentCASLObjectID()
	if !ok {
		return
	}
	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-меню недоступне у поточній конфігурації.")
		return
	}
	dialogs.ShowCASLObjectBlockDialog(a.mainWindow, editor, objectID, a.refreshCASLObjectData)
}

func (a *Application) currentCASLObjectID() (int64, bool) {
	if a.currentObject == nil || a.currentObject.ID <= 0 {
		dialogs.ShowInfoDialog(a.mainWindow, "Об'єкт не вибрано", "Виберіть CASL-об'єкт у списку.")
		return 0, false
	}
	if !ids.IsCASLObjectID(a.currentObject.ID) {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-меню працює лише для CASL-об'єктів.")
		return 0, false
	}
	return int64(a.currentObject.ID), true
}

func (a *Application) refreshCASLObjectData() {
	a.publishDataRefresh(eventbus.DataRefreshEvent{
		RefreshObjects: true,
		RefreshAlarms:  true,
		RefreshEvents:  true,
	})
}
