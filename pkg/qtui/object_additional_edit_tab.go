//go:build qt

package qtui

import (
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectAdditionalEditTab struct {
	parent      *qt.QWidget
	provider    contracts.AdminObjectCoordinatesService
	objn        int64
	statusLabel *qt.QLabel
	getAddress  func() string
	vm          *viewmodels.ObjectAdditionalTabViewModel
	loaded      bool
	loading     bool

	address   *qt.QLineEdit
	latitude  *qt.QLineEdit
	longitude *qt.QLineEdit
}

func newObjectAdditionalEditTab(
	parent *qt.QWidget,
	provider contracts.AdminObjectCoordinatesService,
	objn int64,
	statusLabel *qt.QLabel,
	getAddress func() string,
) (*qt.QWidget, func()) {
	tab := &objectAdditionalEditTab{
		parent:      parent,
		provider:    provider,
		objn:        objn,
		statusLabel: statusLabel,
		getAddress:  getAddress,
		vm:          viewmodels.NewObjectAdditionalTabViewModel(),
		address:     newLineEdit(""),
		latitude:    newLineEdit(""),
		longitude:   newLineEdit(""),
	}
	tab.address.SetPlaceholderText("Адреса об'єкта")
	tab.latitude.SetPlaceholderText("Широта (LATITUDE)")
	tab.longitude.SetPlaceholderText("Довгота (LONGITUDE)")

	return tab.buildContent(), tab.ensureLoaded
}

func (tab *objectAdditionalEditTab) buildContent() *qt.QWidget {
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)

	formWidget := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Адреса", tab.address.QWidget)
	form.AddRow3("Широта", tab.latitude.QWidget)
	form.AddRow3("Довгота", tab.longitude.QWidget)
	formWidget.SetLayout(form.QLayout)
	layout.AddWidget(formWidget)

	useAddressButton := qt.NewQPushButton3("Взяти адресу з об'єкта")
	useAddressButton.OnClicked(func() {
		if address, ok := tab.vm.AddressFromObjectTab(tab.getAddress); ok {
			tab.address.SetText(address)
			tab.setStatus("Адресу синхронізовано")
		}
	})
	saveButton := qt.NewQPushButton3("Зберегти координати")
	saveButton.OnClicked(tab.save)
	clearButton := qt.NewQPushButton3("Очистити")
	clearButton.OnClicked(func() {
		tab.latitude.Clear()
		tab.longitude.Clear()
		tab.save()
	})
	refreshButton := qt.NewQPushButton3("Оновити")
	refreshButton.OnClicked(tab.reload)

	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(useAddressButton.QWidget)
	actions.AddStretch()
	actions.AddWidget(refreshButton.QWidget)
	actions.AddWidget(clearButton.QWidget)
	actions.AddWidget(saveButton.QWidget)
	layout.AddLayout(actions.QLayout)
	layout.AddStretch()
	content.SetLayout(layout.QLayout)
	return content
}

func (tab *objectAdditionalEditTab) reload() {
	if tab.provider == nil || tab.loading {
		return
	}
	tab.loading = true
	tab.setStatus("Завантаження координат...")
	go func() {
		coords, err := tab.provider.GetObjectCoordinates(tab.objn)
		RunOnMainThread(func() {
			tab.loading = false
			if err != nil {
				tab.setStatus("Не вдалося завантажити координати")
				qt.QMessageBox_Critical(tab.parent, "Координати об'єкта", err.Error())
				return
			}
			tab.loaded = true
			if address, ok := tab.vm.AddressFromObjectTab(tab.getAddress); ok {
				tab.address.SetText(address)
			}
			tab.latitude.SetText(strings.TrimSpace(coords.Latitude))
			tab.longitude.SetText(strings.TrimSpace(coords.Longitude))
			tab.setStatus("Координати завантажено")
		})
	}()
}

func (tab *objectAdditionalEditTab) ensureLoaded() {
	if !tab.loaded {
		tab.reload()
	}
}

func (tab *objectAdditionalEditTab) save() {
	if tab.provider == nil {
		return
	}
	coords := tab.vm.BuildCoordinates(tab.latitude.Text(), tab.longitude.Text())
	err := tab.provider.SaveObjectCoordinates(tab.objn, contracts.AdminObjectCoordinates{
		Latitude:  coords.Latitude,
		Longitude: coords.Longitude,
	})
	if err != nil {
		tab.setStatus("Не вдалося зберегти координати")
		qt.QMessageBox_Critical(tab.parent, "Координати об'єкта", err.Error())
		return
	}
	tab.setStatus("Координати збережено")
}

func (tab *objectAdditionalEditTab) setStatus(text string) {
	if tab.statusLabel != nil {
		tab.statusLabel.SetText(text)
	}
}
