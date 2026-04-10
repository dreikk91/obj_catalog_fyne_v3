package application

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func (a *Application) registerShortcuts(themeBtn *widget.Button) {
	if a == nil || a.mainWindow == nil {
		return
	}
	canvas := a.mainWindow.Canvas()

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if themeBtn != nil && themeBtn.OnTapped != nil {
			themeBtn.OnTapped()
		}
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if a.objectList != nil && a.objectList.SearchEntry != nil {
			a.mainWindow.Canvas().Focus(a.objectList.SearchEntry)
		}
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyN, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		withAdminCapability(a, func(admin contracts.AdminObjectWizardProvider) {
			a.openNewObjectDialog(admin)
		})()
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		withAdminCapability(a, func(admin contracts.AdminObjectCardProvider) {
			a.openEditCurrentObjectDialog(admin)
		})()
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyX, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		withAdminCapability(a, func(admin adminObjectDeleteProvider) {
			a.confirmDeleteCurrentObject(admin)
		})()
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key1, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if a.rightTabs != nil {
			a.rightTabs.SelectIndex(0)
		}
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key2, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if a.rightTabs != nil {
			a.rightTabs.SelectIndex(1)
		}
	})

	canvas.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key3, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if a.rightTabs != nil {
			a.rightTabs.SelectIndex(2)
		}
	})
}
