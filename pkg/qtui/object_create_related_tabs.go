//go:build qt

package qtui

import (
	"fmt"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectCreatePersonalsTab struct {
	parent *qt.QWidget
	table  *qt.QTableWidget
	items  []contracts.AdminObjectPersonal
}

func newObjectCreatePersonalsTab(parent *qt.QWidget) *objectCreatePersonalsTab {
	tab := &objectCreatePersonalsTab{
		parent: parent,
		table:  qt.NewQTableWidget3(0, 5),
	}
	tab.table.SetHorizontalHeaderLabels([]string{"№", "ПІБ", "Телефон", "Посада", "Примітка"})
	tab.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	tab.table.OnCellDoubleClicked(func(row int, _ int) { tab.edit(row) })
	tab.applyColumnWidths()
	return tab
}

func (tab *objectCreatePersonalsTab) widget() *qt.QWidget {
	addButton := qt.NewQPushButton3("Додати")
	editButton := qt.NewQPushButton3("Змінити")
	deleteButton := qt.NewQPushButton3("Видалити")
	addButton.OnClicked(tab.add)
	editButton.OnClicked(func() { tab.edit(tab.table.CurrentRow()) })
	deleteButton.OnClicked(tab.delete)

	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(deleteButton.QWidget)
	actions.AddStretch()

	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(tab.table.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (tab *objectCreatePersonalsTab) add() {
	item, accepted := showObjectPersonalEditDialog(tab.parent, contracts.AdminObjectPersonal{
		Number: int64(len(tab.items) + 1),
		IsRang: true,
	})
	if !accepted {
		return
	}
	tab.items = append(tab.items, item)
	tab.refresh()
}

func (tab *objectCreatePersonalsTab) edit(row int) {
	if row < 0 || row >= len(tab.items) {
		return
	}
	item, accepted := showObjectPersonalEditDialog(tab.parent, tab.items[row])
	if !accepted {
		return
	}
	tab.items[row] = item
	tab.refresh()
	tab.table.SetCurrentCell(row, 0)
}

func (tab *objectCreatePersonalsTab) delete() {
	row := tab.table.CurrentRow()
	if row < 0 || row >= len(tab.items) {
		return
	}
	tab.items = append(tab.items[:row], tab.items[row+1:]...)
	tab.refresh()
}

func (tab *objectCreatePersonalsTab) refresh() {
	tab.table.SetRowCount(len(tab.items))
	for row, item := range tab.items {
		tab.table.SetItem(row, 0, qt.NewQTableWidgetItem2(formatInt64NonZero(item.Number)))
		tab.table.SetItem(row, 1, qt.NewQTableWidgetItem2(objectPersonalFullName(item)))
		tab.table.SetItem(row, 2, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Phones)))
		tab.table.SetItem(row, 3, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Position)))
		tab.table.SetItem(row, 4, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Notes)))
	}
	tab.applyColumnWidths()
}

func (tab *objectCreatePersonalsTab) applyColumnWidths() {
	for column, width := range []int{60, 260, 180, 160, 220} {
		tab.table.SetColumnWidth(column, width)
	}
}

func (tab *objectCreatePersonalsTab) viewModelItems() []viewmodels.ObjectPersonal {
	return viewmodels.ObjectPersonalsFromContracts(tab.items)
}

func (tab *objectCreatePersonalsTab) hasChanges() bool {
	return len(tab.items) > 0
}

type objectCreateZonesTab struct {
	parent *qt.QWidget
	table  *qt.QTableWidget
	zones  []contracts.AdminObjectZone
}

func newObjectCreateZonesTab(parent *qt.QWidget) *objectCreateZonesTab {
	tab := &objectCreateZonesTab{
		parent: parent,
		table:  qt.NewQTableWidget3(0, 2),
	}
	tab.table.SetHorizontalHeaderLabels([]string{"Зона", "Опис"})
	tab.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	tab.table.OnCellDoubleClicked(func(row int, _ int) { tab.edit(row) })
	tab.applyColumnWidths()
	return tab
}

func (tab *objectCreateZonesTab) widget() *qt.QWidget {
	addButton := qt.NewQPushButton3("Додати")
	editButton := qt.NewQPushButton3("Змінити")
	deleteButton := qt.NewQPushButton3("Видалити")
	fillButton := qt.NewQPushButton3("Заповнити")
	clearButton := qt.NewQPushButton3("Очистити")
	addButton.OnClicked(tab.add)
	editButton.OnClicked(func() { tab.edit(tab.table.CurrentRow()) })
	deleteButton.OnClicked(tab.delete)
	fillButton.OnClicked(tab.fill)
	clearButton.OnClicked(func() {
		tab.zones = nil
		tab.refresh()
	})

	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(addButton.QWidget)
	actions.AddWidget(editButton.QWidget)
	actions.AddWidget(deleteButton.QWidget)
	actions.AddWidget(fillButton.QWidget)
	actions.AddWidget(clearButton.QWidget)
	actions.AddStretch()

	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddLayout(actions.QLayout)
	layout.AddWidget(tab.table.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (tab *objectCreateZonesTab) add() {
	zone, accepted := showObjectZoneEditDialog(tab.parent, contracts.AdminObjectZone{
		ZoneNumber: tab.nextNumber(),
		ZoneType:   1,
	})
	if !accepted {
		return
	}
	if tab.findNumber(zone.ZoneNumber, -1) >= 0 {
		qt.QMessageBox_Information(tab.parent, "Зони", fmt.Sprintf("Зона #%d вже існує.", zone.ZoneNumber))
		return
	}
	tab.zones = append(tab.zones, zone)
	tab.refresh()
}

func (tab *objectCreateZonesTab) edit(row int) {
	if row < 0 || row >= len(tab.zones) {
		return
	}
	zone, accepted := showObjectZoneEditDialog(tab.parent, tab.zones[row])
	if !accepted {
		return
	}
	if tab.findNumber(zone.ZoneNumber, row) >= 0 {
		qt.QMessageBox_Information(tab.parent, "Зони", fmt.Sprintf("Зона #%d вже існує.", zone.ZoneNumber))
		return
	}
	tab.zones[row] = zone
	tab.refresh()
	tab.table.SetCurrentCell(row, 0)
}

func (tab *objectCreateZonesTab) delete() {
	row := tab.table.CurrentRow()
	if row < 0 || row >= len(tab.zones) {
		return
	}
	tab.zones = append(tab.zones[:row], tab.zones[row+1:]...)
	tab.refresh()
}

func (tab *objectCreateZonesTab) fill() {
	count := qt.QInputDialog_GetInt(tab.parent, "Заповнити зони", "Кількість зон:")
	if count <= 0 {
		return
	}
	for number := 1; number <= count; number++ {
		if tab.findNumber(int64(number), -1) >= 0 {
			continue
		}
		tab.zones = append(tab.zones, contracts.AdminObjectZone{
			ZoneNumber:  int64(number),
			ZoneType:    1,
			Description: "Шлейф " + strconv.Itoa(number),
		})
	}
	tab.refresh()
}

func (tab *objectCreateZonesTab) nextNumber() int64 {
	var maximum int64
	for _, zone := range tab.zones {
		if zone.ZoneNumber > maximum {
			maximum = zone.ZoneNumber
		}
	}
	return maximum + 1
}

func (tab *objectCreateZonesTab) findNumber(number int64, exceptRow int) int {
	for row, zone := range tab.zones {
		if row != exceptRow && zone.ZoneNumber == number {
			return row
		}
	}
	return -1
}

func (tab *objectCreateZonesTab) refresh() {
	tab.table.SetRowCount(len(tab.zones))
	for row, zone := range tab.zones {
		tab.table.SetItem(row, 0, qt.NewQTableWidgetItem2(strconv.FormatInt(zone.ZoneNumber, 10)))
		tab.table.SetItem(row, 1, qt.NewQTableWidgetItem2(strings.TrimSpace(zone.Description)))
	}
	tab.applyColumnWidths()
}

func (tab *objectCreateZonesTab) applyColumnWidths() {
	tab.table.SetColumnWidth(0, 90)
	tab.table.SetColumnWidth(1, 520)
}

func (tab *objectCreateZonesTab) viewModelItems() []viewmodels.ObjectZone {
	return viewmodels.ObjectZonesFromContracts(tab.zones)
}

func (tab *objectCreateZonesTab) hasChanges() bool {
	return len(tab.zones) > 0
}

type objectCreateCoordinatesTab struct {
	address    *qt.QLineEdit
	latitude   *qt.QLineEdit
	longitude  *qt.QLineEdit
	getAddress func() string
}

func newObjectCreateCoordinatesTab(getAddress func() string) *objectCreateCoordinatesTab {
	tab := &objectCreateCoordinatesTab{
		address:    newLineEdit(""),
		latitude:   newLineEdit(""),
		longitude:  newLineEdit(""),
		getAddress: getAddress,
	}
	tab.address.SetPlaceholderText("Адреса об'єкта")
	tab.latitude.SetPlaceholderText("Широта (LATITUDE)")
	tab.longitude.SetPlaceholderText("Довгота (LONGITUDE)")
	return tab
}

func (tab *objectCreateCoordinatesTab) widget() *qt.QWidget {
	useAddressButton := qt.NewQPushButton3("Взяти адресу з об'єкта")
	useAddressButton.OnClicked(func() {
		if tab.getAddress != nil {
			tab.address.SetText(strings.TrimSpace(tab.getAddress()))
		}
	})
	clearButton := qt.NewQPushButton3("Очистити координати")
	clearButton.OnClicked(func() {
		tab.latitude.Clear()
		tab.longitude.Clear()
	})

	formWidget := qt.NewQWidget2()
	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Адреса", tab.address.QWidget)
	form.AddRow3("Широта", tab.latitude.QWidget)
	form.AddRow3("Довгота", tab.longitude.QWidget)
	formWidget.SetLayout(form.QLayout)

	actions := qt.NewQHBoxLayout2()
	actions.AddWidget(useAddressButton.QWidget)
	actions.AddStretch()
	actions.AddWidget(clearButton.QWidget)

	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.AddWidget(formWidget)
	layout.AddLayout(actions.QLayout)
	layout.AddStretch()
	content.SetLayout(layout.QLayout)
	return content
}

func (tab *objectCreateCoordinatesTab) coordinates() viewmodels.ObjectCoordinates {
	return viewmodels.ObjectCoordinates{
		Latitude:  strings.TrimSpace(tab.latitude.Text()),
		Longitude: strings.TrimSpace(tab.longitude.Text()),
	}
}

func (tab *objectCreateCoordinatesTab) hasChanges() bool {
	return strings.TrimSpace(tab.address.Text()) != "" ||
		strings.TrimSpace(tab.latitude.Text()) != "" ||
		strings.TrimSpace(tab.longitude.Text()) != ""
}

type qtObjectWizardPersistence struct {
	provider contracts.AdminObjectWizardProvider
}

func (p qtObjectWizardPersistence) CreateObject(card contracts.AdminObjectCard) error {
	return p.provider.CreateObject(card)
}

func (p qtObjectWizardPersistence) AddObjectPersonal(objn int64, item viewmodels.ObjectPersonal) error {
	return p.provider.AddObjectPersonal(objn, item.ToContracts())
}

func (p qtObjectWizardPersistence) AddObjectZone(objn int64, zone viewmodels.ObjectZone) error {
	return p.provider.AddObjectZone(objn, zone.ToContracts())
}

func (p qtObjectWizardPersistence) SaveObjectCoordinates(objn int64, coords viewmodels.ObjectCoordinates) error {
	return p.provider.SaveObjectCoordinates(objn, coords.ToContracts())
}
