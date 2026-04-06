package application

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/eventbus"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
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
	if a.currentObject == nil || a.currentObject.ID <= 0 {
		dialogs.ShowInfoDialog(a.mainWindow, "Об'єкт не вибрано", "Виберіть CASL-об'єкт у списку.")
		return
	}
	if !viewmodels.IsCASLObjectID(a.currentObject.ID) {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-редактор працює лише для CASL-об'єктів.")
		return
	}

	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-редактор недоступний у поточній конфігурації.")
		return
	}

	dialogs.ShowCASLObjectEditorDialog(a.mainWindow, editor, int64(a.currentObject.ID), func() {
		a.publishDataRefresh(eventbus.DataRefreshEvent{
			RefreshObjects: true,
			RefreshAlarms:  true,
			RefreshEvents:  true,
		})
	})
}

func (a *Application) openCASLObjectCreator() {
	editor, ok := a.resolveCASLObjectEditorProvider()
	if !ok {
		dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "CASL-редактор недоступний у поточній конфігурації.")
		return
	}

	dialogs.ShowCASLObjectEditorDialog(a.mainWindow, editor, 0, func() {
		a.publishDataRefresh(eventbus.DataRefreshEvent{
			RefreshObjects: true,
			RefreshAlarms:  true,
			RefreshEvents:  true,
		})
	})
}
