package dialogs

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const (
	mapCenterModePrefKey    = "admin.map.center.mode"
	mapCenterCustomLatKey   = "admin.map.center.custom.lat"
	mapCenterCustomLonKey   = "admin.map.center.custom.lon"
	mapCenterLastLatPrefKey = "admin.map.center.last.lat"
	mapCenterLastLonPrefKey = "admin.map.center.last.lon"

	mapCenterModeLviv   = "lviv"
	mapCenterModeKyiv   = "kyiv"
	mapCenterModeCustom = "custom"
	mapCenterModeLast   = "last"

	mapDefaultLvivLat = 49.8397
	mapDefaultLvivLon = 24.0297
	mapDefaultKyivLat = 50.4501
	mapDefaultKyivLon = 30.5234
	mapDefaultZoom    = 12
)

func buildObjectPersonalTab(parent fyne.Window, provider contracts.AdminObjectPersonalTabProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	state := newObjectPersonalTabState(parent, provider, objn, statusLabel)
	content := state.buildContent()
	state.reload()
	return content
}

type objectPersonalTabState struct {
	parent      fyne.Window
	provider    contracts.AdminObjectPersonalTabProvider
	objn        int64
	statusLabel *widget.Label
	vm          *viewmodels.ObjectPersonalsTabViewModel
	table       *widget.Table
}

func newObjectPersonalTabState(
	parent fyne.Window,
	provider contracts.AdminObjectPersonalTabProvider,
	objn int64,
	statusLabel *widget.Label,
) *objectPersonalTabState {
	state := &objectPersonalTabState{
		parent:      parent,
		provider:    provider,
		objn:        objn,
		statusLabel: statusLabel,
		vm:          viewmodels.NewObjectPersonalsTabViewModel(),
	}
	state.table = state.buildTable()
	return state
}

func (s *objectPersonalTabState) buildContent() fyne.CanvasObject {
	tableScroll := container.NewScroll(s.table)
	tableScroll.SetMinSize(fyne.NewSize(420, 260))

	return container.NewBorder(
		container.NewVBox(
			container.NewHBox(
				widget.NewButton("Додати", s.showAddDialog),
				widget.NewButton("Змінити", s.showEditDialog),
				widget.NewButton("Видалити", s.deleteSelected),
				layout.NewSpacer(),
				widget.NewButton("Оновити", s.reload),
			),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		tableScroll,
	)
}

func (s *objectPersonalTabState) buildTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return s.vm.Count() + 1, 6 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			s.updateTableCell(id, obj.(*widget.Label))
		},
	)
	const (
		personalColWNum   = float32(60)
		personalColWName  = float32(280)
		personalColWPhone = float32(200)
		personalColWPos   = float32(180)
		personalColWRole  = float32(110)
		personalColWNote  = float32(220)
	)
	table.SetColumnWidth(0, personalColWNum)
	table.SetColumnWidth(1, personalColWName)
	table.SetColumnWidth(2, personalColWPhone)
	table.SetColumnWidth(3, personalColWPos)
	table.SetColumnWidth(4, personalColWRole)
	table.SetColumnWidth(5, personalColWNote)
	table.OnSelected = s.handleTableSelection

	return table
}

func (s *objectPersonalTabState) updateTableCell(id widget.TableCellID, label *widget.Label) {
	if id.Row == 0 {
		switch id.Col {
		case 0:
			label.SetText("№")
		case 1:
			label.SetText("ПІБ")
		case 2:
			label.SetText("Телефон")
		case 3:
			label.SetText("Посада")
		case 4:
			label.SetText("Доступ")
		default:
			label.SetText("Примітка")
		}
		return
	}

	itemIndex := id.Row - 1
	item, ok := s.vm.ItemAt(itemIndex)
	if !ok {
		label.SetText("")
		return
	}

	switch id.Col {
	case 0:
		label.SetText(strconv.FormatInt(item.Number, 10))
	case 1:
		label.SetText(s.vm.FullName(item))
	case 2:
		label.SetText(strings.TrimSpace(item.Phones))
	case 3:
		label.SetText(strings.TrimSpace(item.Position))
	case 4:
		if item.Access1 > 0 {
			label.SetText("Адмін")
		} else {
			label.SetText("Оператор")
		}
	case 5:
		label.SetText(strings.TrimSpace(item.Notes))
	}
}

func (s *objectPersonalTabState) handleTableSelection(id widget.TableCellID) {
	s.vm.SelectByTableRow(id.Row)
}

func (s *objectPersonalTabState) reload() {
	loaded, err := s.provider.ListObjectPersonals(s.objn)
	if err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося завантажити В/О")
		return
	}

	s.vm.SetItems(loaded)
	s.table.UnselectAll()
	s.table.Refresh()
	s.statusLabel.SetText(s.vm.CountStatusText())
}

func (s *objectPersonalTabState) showAddDialog() {
	showObjectPersonalEditor(
		s.parent,
		s.provider,
		"Додати В/О",
		contracts.AdminObjectPersonal{},
		func(item contracts.AdminObjectPersonal) error {
			return s.provider.AddObjectPersonal(s.objn, item)
		},
		s.statusLabel,
		func() {
			s.reload()
			s.statusLabel.SetText("В/О додано")
		},
	)
}

func (s *objectPersonalTabState) showEditDialog() {
	initial, ok := s.vm.SelectedItem()
	if !ok {
		s.statusLabel.SetText("Виберіть В/О у таблиці")
		return
	}

	showObjectPersonalEditor(
		s.parent,
		s.provider,
		"Редагування В/О",
		initial,
		func(item contracts.AdminObjectPersonal) error {
			return s.provider.UpdateObjectPersonal(s.objn, s.vm.PrepareUpdatedItem(initial, item))
		},
		s.statusLabel,
		func() {
			s.reload()
			s.statusLabel.SetText("В/О оновлено")
		},
	)
}

func (s *objectPersonalTabState) deleteSelected() {
	target, ok := s.vm.SelectedItem()
	if !ok {
		s.statusLabel.SetText("Виберіть В/О у таблиці")
		return
	}

	dialog.ShowConfirm(
		"Підтвердження",
		fmt.Sprintf("Видалити запис \"%s\"?", s.vm.FullName(target)),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := s.provider.DeleteObjectPersonal(s.objn, target.ID); err != nil {
				dialog.ShowError(err, s.parent)
				s.statusLabel.SetText("Не вдалося видалити В/О")
				return
			}
			s.reload()
			s.statusLabel.SetText("В/О видалено")
		},
		s.parent,
	)
}

func buildObjectZonesTab(parent fyne.Window, provider contracts.AdminObjectZonesTabProvider, objn int64, statusLabel *widget.Label) fyne.CanvasObject {
	state := newObjectZonesTabState(parent, provider, objn, statusLabel)
	content := state.buildContent()
	state.reload()
	return content
}

type objectZonesTabState struct {
	parent            fyne.Window
	provider          contracts.AdminObjectZonesTabProvider
	objn              int64
	statusLabel       *widget.Label
	vm                *viewmodels.ObjectZonesTabViewModel
	quickNameEntry    *widget.Entry
	selectedZoneLabel *widget.Label
	table             *widget.Table
}

func newObjectZonesTabState(
	parent fyne.Window,
	provider contracts.AdminObjectZonesTabProvider,
	objn int64,
	statusLabel *widget.Label,
) *objectZonesTabState {
	state := &objectZonesTabState{
		parent:            parent,
		provider:          provider,
		objn:              objn,
		statusLabel:       statusLabel,
		vm:                viewmodels.NewObjectZonesTabViewModel(),
		quickNameEntry:    widget.NewEntry(),
		selectedZoneLabel: widget.NewLabel("Зона: —"),
	}
	state.quickNameEntry.SetPlaceHolder("Назва зони (Enter -> наступна зона)")
	state.quickNameEntry.OnSubmitted = func(string) {
		state.moveToNextZone()
	}
	state.table = state.buildTable()

	return state
}

func (s *objectZonesTabState) buildContent() fyne.CanvasObject {
	tableScroll := container.NewScroll(s.table)
	tableScroll.SetMinSize(fyne.NewSize(420, 260))

	return container.NewBorder(
		container.NewVBox(
			container.NewHBox(
				widget.NewButton("Додати", s.addZone),
				widget.NewButton("Змінити", s.editZone),
				widget.NewButton("Видалити", s.deleteZone),
				widget.NewButton("Заповнити", s.fillZones),
				widget.NewButton("Очистити", s.clearZones),
				layout.NewSpacer(),
				widget.NewButton("Оновити", s.reload),
			),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel("Швидке введення:"), layout.NewSpacer(), s.selectedZoneLabel),
				widget.NewButton("Enter -> Наступна", s.moveToNextZone),
				s.quickNameEntry,
			),
		),
		nil,
		nil,
		tableScroll,
	)
}

func (s *objectZonesTabState) buildTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return s.vm.Count() + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			s.updateTableCell(id, obj.(*widget.Label))
		},
	)
	table.StickyRowCount = 1
	table.StickyColumnCount = 1
	s.applyTableLayout(table)
	table.OnSelected = s.handleTableSelection

	return table
}

func (s *objectZonesTabState) updateTableCell(id widget.TableCellID, label *widget.Label) {
	if id.Row == 0 {
		switch id.Col {
		case 0:
			label.SetText("ZONEN")
		case 1:
			label.SetText("Тип")
		default:
			label.SetText("Опис")
		}
		return
	}

	itemIndex := id.Row - 1
	item, ok := s.vm.ItemAt(itemIndex)
	if !ok {
		label.SetText("")
		return
	}

	switch id.Col {
	case 0:
		label.SetText(strconv.FormatInt(s.vm.EffectiveZoneNumberAt(itemIndex), 10))
	case 1:
		label.SetText("пож.")
	default:
		label.SetText(strings.TrimSpace(item.Description))
	}
}

func (s *objectZonesTabState) applyTableLayout(table *widget.Table) {
	const (
		zoneColWNum  = float32(120)
		zoneColWType = float32(120)
		zoneColWDesc = float32(520)
	)

	table.SetColumnWidth(0, zoneColWNum)
	table.SetColumnWidth(1, zoneColWType)
	table.SetColumnWidth(2, zoneColWDesc)
}

func (s *objectZonesTabState) updateSelectedZoneLabel() {
	s.selectedZoneLabel.SetText(s.vm.SelectedZoneLabel())
}

func (s *objectZonesTabState) ensureZoneExists(zoneNumber int64, defaultDescription string) error {
	if s.vm.FindRowByZoneNumber(zoneNumber) >= 0 {
		return nil
	}

	zone, err := s.vm.BuildZoneForCreate(zoneNumber, defaultDescription)
	if err != nil {
		return err
	}
	return s.provider.AddObjectZone(s.objn, zone)
}

func (s *objectZonesTabState) selectByZoneNumber(zoneNumber int64, focusQuickName bool) {
	if !s.vm.SelectZoneByNumber(zoneNumber) {
		s.table.UnselectAll()
		s.quickNameEntry.SetText("")
		s.updateSelectedZoneLabel()
		return
	}

	targetRow := s.vm.SelectedRow()
	s.table.Select(widget.TableCellID{Row: targetRow + 1, Col: 0})
	s.quickNameEntry.SetText(s.vm.SelectedZoneDescription())
	s.updateSelectedZoneLabel()
	if focusQuickName {
		focusIfOnCanvas(s.parent, s.quickNameEntry)
	}
}

func (s *objectZonesTabState) reloadAndSelect(targetZoneNumber int64, focusQuickName bool) {
	loaded, err := s.provider.ListObjectZones(s.objn)
	if err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося завантажити зони")
		return
	}

	s.vm.SetItems(loaded)
	s.table.Refresh()
	s.applyTableLayout(s.table)
	s.statusLabel.SetText(s.vm.CountStatusText())
	s.selectByZoneNumber(targetZoneNumber, focusQuickName)
}

func (s *objectZonesTabState) reload() {
	s.reloadAndSelect(0, false)
}

func (s *objectZonesTabState) handleTableSelection(id widget.TableCellID) {
	if !s.vm.SelectByTableRow(id.Row) {
		s.quickNameEntry.SetText("")
		s.updateSelectedZoneLabel()
		return
	}

	s.quickNameEntry.SetText(s.vm.SelectedZoneDescription())
	s.updateSelectedZoneLabel()
	focusIfOnCanvas(s.parent, s.quickNameEntry)
}

func (s *objectZonesTabState) moveToNextZone() {
	if _, ok := s.vm.SelectedItem(); !ok {
		if s.vm.Count() == 0 {
			if err := s.ensureZoneExists(1, strings.TrimSpace(s.quickNameEntry.Text)); err != nil {
				dialog.ShowError(err, s.parent)
				s.statusLabel.SetText("Не вдалося додати першу зону")
				return
			}
			s.reloadAndSelect(1, true)
			s.statusLabel.SetText("Додано зону #1")
			return
		}
		s.selectByZoneNumber(0, true)
	}

	current, currentZoneNumber, ok := s.vm.PrepareSelectedZoneForSave(s.quickNameEntry.Text)
	if !ok {
		s.statusLabel.SetText("Виберіть зону у таблиці")
		return
	}
	if err := s.provider.UpdateObjectZone(s.objn, current); err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося зберегти назву зони")
		return
	}

	nextZoneNumber := currentZoneNumber + 1
	if err := s.ensureZoneExists(nextZoneNumber, ""); err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося додати наступну зону")
		return
	}

	s.reloadAndSelect(nextZoneNumber, true)
	s.statusLabel.SetText(fmt.Sprintf("Збережено зону #%d, перехід на #%d", currentZoneNumber, nextZoneNumber))
}

func (s *objectZonesTabState) addZone() {
	nextZoneNumber := s.vm.NextZoneNumberForAdd()
	if err := s.ensureZoneExists(nextZoneNumber, ""); err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося додати зону")
		return
	}
	s.reloadAndSelect(nextZoneNumber, true)
	s.statusLabel.SetText(fmt.Sprintf("Готово до введення зони #%d", nextZoneNumber))
}

func (s *objectZonesTabState) editZone() {
	if s.vm.Count() == 0 {
		if err := s.ensureZoneExists(1, ""); err != nil {
			dialog.ShowError(err, s.parent)
			s.statusLabel.SetText("Не вдалося створити першу зону")
			return
		}
		s.reloadAndSelect(1, true)
		s.statusLabel.SetText("Створено зону #1, можна вводити назву")
		return
	}
	if _, ok := s.vm.SelectedItem(); !ok {
		s.selectByZoneNumber(0, true)
		s.statusLabel.SetText("Виберіть зону і вводьте назву")
		return
	}
	zoneNumber, ok := s.vm.SelectedZoneNumber()
	if !ok {
		s.statusLabel.SetText("Виберіть зону і вводьте назву")
		return
	}

	s.updateSelectedZoneLabel()
	focusIfOnCanvas(s.parent, s.quickNameEntry)
	s.statusLabel.SetText(fmt.Sprintf("Редагування зони #%d: введіть назву і натисніть Enter", zoneNumber))
}

func (s *objectZonesTabState) deleteZone() {
	target, ok := s.vm.SelectedItem()
	if !ok {
		s.statusLabel.SetText("Виберіть зону у таблиці")
		return
	}

	targetZoneNumber, ok := s.vm.SelectedZoneNumber()
	if !ok {
		targetZoneNumber = target.ZoneNumber
	}
	dialog.ShowConfirm(
		"Підтвердження",
		fmt.Sprintf("Видалити зону #%d?", targetZoneNumber),
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := s.provider.DeleteObjectZone(s.objn, target.ID); err != nil {
				dialog.ShowError(err, s.parent)
				s.statusLabel.SetText("Не вдалося видалити зону")
				return
			}
			s.reload()
			s.statusLabel.SetText("Зону видалено")
		},
		s.parent,
	)
}

func (s *objectZonesTabState) fillZones() {
	defaultCount := suggestZoneFillCount(s.provider, s.objn, s.vm.Items())
	showZoneFillDialog(s.parent, defaultCount, func(count int64) {
		if err := s.provider.FillObjectZones(s.objn, count); err != nil {
			dialog.ShowError(err, s.parent)
			s.statusLabel.SetText("Не вдалося заповнити зони")
			return
		}
		s.reload()
		s.statusLabel.SetText("Зони заповнено")
	}, s.statusLabel)
}

func (s *objectZonesTabState) clearZones() {
	dialog.ShowConfirm(
		"Підтвердження",
		"Видалити всі зони об'єкта?",
		func(confirmed bool) {
			if !confirmed {
				return
			}
			if err := s.provider.ClearObjectZones(s.objn); err != nil {
				dialog.ShowError(err, s.parent)
				s.statusLabel.SetText("Не вдалося очистити зони")
				return
			}
			s.reload()
			s.statusLabel.SetText("Зони очищено")
		},
		s.parent,
	)
}

func buildObjectAdditionalTab(
	parent fyne.Window,
	provider contracts.AdminObjectAdditionalTabProvider,
	objn int64,
	statusLabel *widget.Label,
	getAddressFromObjectTab func() string,
	setRegionInObjectTab func(regionID int64) bool,
) fyne.CanvasObject {
	state := newObjectAdditionalTabState(
		parent,
		provider,
		objn,
		statusLabel,
		getAddressFromObjectTab,
		setRegionInObjectTab,
	)
	content := state.buildContent()
	state.reload()
	return content
}

type objectAdditionalTabState struct {
	parent               fyne.Window
	provider             contracts.AdminObjectAdditionalTabProvider
	objn                 int64
	statusLabel          *widget.Label
	getAddressFromObject func() string
	setRegionInObjectTab func(regionID int64) bool
	vm                   *viewmodels.ObjectAdditionalTabViewModel
	addressEntry         *widget.Entry
	latitudeEntry        *widget.Entry
	longitudeEntry       *widget.Entry
}

func newObjectAdditionalTabState(
	parent fyne.Window,
	provider contracts.AdminObjectAdditionalTabProvider,
	objn int64,
	statusLabel *widget.Label,
	getAddressFromObjectTab func() string,
	setRegionInObjectTab func(regionID int64) bool,
) *objectAdditionalTabState {
	state := &objectAdditionalTabState{
		parent:               parent,
		provider:             provider,
		objn:                 objn,
		statusLabel:          statusLabel,
		getAddressFromObject: getAddressFromObjectTab,
		setRegionInObjectTab: setRegionInObjectTab,
		vm:                   viewmodels.NewObjectAdditionalTabViewModel(),
		addressEntry:         widget.NewEntry(),
		latitudeEntry:        widget.NewEntry(),
		longitudeEntry:       widget.NewEntry(),
	}
	state.addressEntry.SetPlaceHolder("Адреса для геопошуку")
	state.latitudeEntry.SetPlaceHolder("Широта (LATITUDE)")
	state.longitudeEntry.SetPlaceHolder("Довгота (LONGITUDE)")
	return state
}

func (s *objectAdditionalTabState) buildContent() fyne.CanvasObject {
	form := widget.NewForm(
		widget.NewFormItem("Адреса:", s.addressEntry),
		widget.NewFormItem("Широта:", s.latitudeEntry),
		widget.NewFormItem("Довгота:", s.longitudeEntry),
	)

	return container.NewBorder(
		container.NewVBox(
			container.NewHBox(
				widget.NewButton("Зберегти координати", s.save),
				widget.NewButton("Очистити", s.clearAndSave),
				widget.NewButton("Вибрати на карті", s.pickCoordinatesOnMap),
				widget.NewButton("Координати з адреси", s.findCoordinatesByAddress),
				widget.NewButton("Район з адреси", s.fillDistrictFromAddress),
			),
			container.NewHBox(
				widget.NewButton("Взяти адресу з Об'єкта", s.useObjectAddress),
				layout.NewSpacer(),
				widget.NewButton("Оновити", s.reload),
			),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		container.NewPadded(form),
	)
}

func (s *objectAdditionalTabState) syncAddressFromObjectTab() {
	address, ok := s.vm.AddressFromObjectTab(s.getAddressFromObject)
	if !ok {
		return
	}
	s.addressEntry.SetText(address)
}

func (s *objectAdditionalTabState) geoByAddress(addressRaw string) (string, string, []string, error) {
	address, err := s.vm.RequireLookupAddress(addressRaw)
	if err != nil {
		return "", "", nil, err
	}
	lat, lon, districtHints, err := geocodeAddress(address)
	if err != nil {
		return "", "", nil, err
	}
	s.vm.RememberGeocode(address, districtHints)
	return lat, lon, districtHints, nil
}

func (s *objectAdditionalTabState) reload() {
	coords, err := s.provider.GetObjectCoordinates(s.objn)
	if err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося завантажити координати")
		return
	}
	s.syncAddressFromObjectTab()
	s.latitudeEntry.SetText(strings.TrimSpace(coords.Latitude))
	s.longitudeEntry.SetText(strings.TrimSpace(coords.Longitude))
	s.statusLabel.SetText("Координати завантажено")
}

func (s *objectAdditionalTabState) save() {
	coords := s.vm.BuildCoordinates(s.latitudeEntry.Text, s.longitudeEntry.Text)
	if err := s.provider.SaveObjectCoordinates(s.objn, coords); err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося зберегти координати")
		return
	}
	s.statusLabel.SetText("Координати збережено")
}

func (s *objectAdditionalTabState) clearAndSave() {
	s.latitudeEntry.SetText("")
	s.longitudeEntry.SetText("")
	s.save()
}

func (s *objectAdditionalTabState) pickCoordinatesOnMap() {
	showCoordinatesMapPicker(
		s.parent,
		strings.TrimSpace(s.latitudeEntry.Text),
		strings.TrimSpace(s.longitudeEntry.Text),
		func(lat, lon string) {
			s.latitudeEntry.SetText(lat)
			s.longitudeEntry.SetText(lon)
			s.statusLabel.SetText("Координати вибрано на карті")
		},
	)
}

func (s *objectAdditionalTabState) findCoordinatesByAddress() {
	lat, lon, districtHints, err := s.geoByAddress(s.addressEntry.Text)
	if err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося знайти координати за адресою")
		return
	}
	s.latitudeEntry.SetText(lat)
	s.longitudeEntry.SetText(lon)
	if len(districtHints) > 0 {
		s.statusLabel.SetText(fmt.Sprintf("Знайдено координати за адресою. Можна також заповнити район (%s)", districtHints[0]))
		return
	}
	s.statusLabel.SetText("Знайдено координати за адресою")
}

func (s *objectAdditionalTabState) fillDistrictFromAddress() {
	address, err := s.vm.RequireLookupAddress(s.addressEntry.Text)
	if err != nil {
		s.statusLabel.SetText(err.Error())
		return
	}

	hints, ok := s.vm.CachedDistrictHintsForAddress(address)
	if !ok {
		_, _, resolvedHints, resolveErr := s.geoByAddress(address)
		if resolveErr != nil {
			dialog.ShowError(resolveErr, s.parent)
			s.statusLabel.SetText("Не вдалося визначити район за адресою")
			return
		}
		hints = resolvedHints
	}

	regionID, regionName, err := resolveRegionByAddressHints(s.provider, hints)
	if err != nil {
		dialog.ShowError(err, s.parent)
		s.statusLabel.SetText("Не вдалося підібрати район за адресою")
		return
	}
	if s.setRegionInObjectTab != nil && s.setRegionInObjectTab(regionID) {
		s.statusLabel.SetText(fmt.Sprintf("Район встановлено: %s (натисніть \"Зберегти\" у картці об'єкта)", regionName))
		return
	}
	s.statusLabel.SetText(fmt.Sprintf("Знайдено район: %s, але не вдалося застосувати у вкладці \"Об'єкт\"", regionName))
}

func (s *objectAdditionalTabState) useObjectAddress() {
	s.syncAddressFromObjectTab()
	s.statusLabel.SetText("Адресу синхронізовано зі вкладки \"Об'єкт\"")
}

func geocodeAddress(address string) (string, string, []string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", "", nil, fmt.Errorf("адреса порожня")
	}

	target := buildGeocodeTarget(address)
	queries := buildGeocodeQueries(target.Cleaned)
	bestScore := math.Inf(-1)
	var best geocodeCandidate
	bestFound := false

	for _, query := range queries {
		rows, err := geocodeCandidatesForQuery(query)
		if err != nil {
			return "", "", nil, err
		}
		for _, row := range rows {
			score := scoreGeocodeCandidate(target, row)
			if !bestFound || score > bestScore {
				bestScore = score
				best = row
				bestFound = true
			}
		}
		// Достатньо точний збіг (місто+вулиця+будинок) - далі мережеві запити не потрібні.
		if bestFound && bestScore >= 90 {
			break
		}
	}

	if bestFound {
		lat := strings.TrimSpace(best.Lat)
		lon := strings.TrimSpace(best.Lon)
		if lat == "" || lon == "" {
			return "", "", nil, fmt.Errorf("геосервіс не повернув координати")
		}
		return lat, lon, collectDistrictHints(best.Address, best.DisplayName), nil
	}

	return "", "", nil, fmt.Errorf("адресу не знайдено")
}

// GeocodeAddressExact повертає координати з максимально точним підбором.
// Використовується також утилітою масової перевірки адрес.
func GeocodeAddressExact(address string) (lat string, lon string, cleaned string, err error) {
	target := buildGeocodeTarget(address)
	lat, lon, _, err = geocodeAddress(address)
	return lat, lon, target.Cleaned, err
}

// GeocodePreviewQueries повертає всі запити, які будуть використані для геопошуку.
func GeocodePreviewQueries(address string) []string {
	target := buildGeocodeTarget(address)
	return buildGeocodeQueries(target.Cleaned)
}

type geocodeTarget struct {
	Cleaned string
	City    string
	Street  string
	House   string
}

type geocodeCandidate struct {
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
	Importance  float64           `json:"importance"`
	Class       string            `json:"class"`
	Type        string            `json:"type"`
}

var (
	geocodeRequestMu    sync.Mutex
	geocodeLastRequest  time.Time
	geocodeMinInterval  = 1100 * time.Millisecond
	geocodeHTTPClient   = &http.Client{Timeout: 14 * time.Second}
	geocodeMaxRetry429  = 3
	geocodeRetryBackoff = 2 * time.Second
)

func buildGeocodeTarget(address string) geocodeTarget {
	cleaned := normalizeAddressForGeocode(address)
	city, street, house, _ := parseAddressComponents(cleaned)
	if city == "" {
		if cityOnly, ok := parseCityOnly(cleaned); ok {
			city = cityOnly
		}
	}
	return geocodeTarget{
		Cleaned: cleaned,
		City:    city,
		Street:  street,
		House:   house,
	}
}

func geocodeCandidatesForQuery(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("format", "jsonv2")
	params.Set("limit", "8")
	params.Set("addressdetails", "1")
	params.Set("accept-language", "uk")
	params.Set("countrycodes", "ua")
	params.Set("dedupe", "0")

	searchURL := "https://nominatim.openstreetmap.org/search?" + params.Encode()

	var last429Details string
	for attempt := 0; attempt <= geocodeMaxRetry429; attempt++ {
		waitForGeocodeRequestSlot()

		req, err := http.NewRequest(http.MethodGet, searchURL, nil)
		if err != nil {
			return nil, fmt.Errorf("не вдалося сформувати запит геопошуку: %w", err)
		}
		req.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")

		resp, err := geocodeHTTPClient.Do(req)
		if err != nil {
			if attempt < geocodeMaxRetry429 {
				time.Sleep(time.Duration(attempt+1) * geocodeRetryBackoff)
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
			if phErr != nil {
				return nil, fmt.Errorf("помилка запиту геопошуку: %v; fallback photon помилка: %v", err, phErr)
			}
			return nil, fmt.Errorf("помилка запиту геопошуку: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
			last429Details = strings.TrimSpace(string(body))
			if attempt < geocodeMaxRetry429 {
				time.Sleep(time.Duration(attempt+1) * geocodeRetryBackoff)
				continue
			}
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
			if phErr != nil {
				return nil, fmt.Errorf("геосервіс повернув 429 (%s), fallback photon помилка: %v", last429Details, phErr)
			}
			return nil, fmt.Errorf("геосервіс повернув 429: %s", last429Details)
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			_ = resp.Body.Close()
			return nil, fmt.Errorf("геосервіс повернув %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var rows []geocodeCandidate
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&rows)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("не вдалося обробити відповідь геосервісу: %w", decodeErr)
		}
		if len(rows) == 0 {
			phRows, phErr := geocodeCandidatesPhoton(query)
			if phErr == nil && len(phRows) > 0 {
				return phRows, nil
			}
		}
		return rows, nil
	}

	phRows, phErr := geocodeCandidatesPhoton(query)
	if phErr == nil && len(phRows) > 0 {
		return phRows, nil
	}
	if phErr != nil {
		return nil, fmt.Errorf("геосервіс недоступний, fallback photon помилка: %v", phErr)
	}
	return nil, fmt.Errorf("геосервіс недоступний")
}

func waitForGeocodeRequestSlot() {
	geocodeRequestMu.Lock()
	defer geocodeRequestMu.Unlock()

	if !geocodeLastRequest.IsZero() {
		wait := geocodeMinInterval - time.Since(geocodeLastRequest)
		if wait > 0 {
			time.Sleep(wait)
		}
	}
	geocodeLastRequest = time.Now()
}

func geocodeCandidatesPhoton(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	params := url.Values{}
	params.Set("q", query)
	params.Set("lang", "uk")
	params.Set("limit", "8")
	photonURL := "https://photon.komoot.io/api/?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, photonURL, nil)
	if err != nil {
		return nil, fmt.Errorf("не вдалося сформувати запит photon: %w", err)
	}
	req.Header.Set("User-Agent", "obj_catalog_fyne_v3/1.0")

	resp, err := geocodeHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("помилка запиту photon: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("photon повернув %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		Features []struct {
			Geometry struct {
				Coordinates []float64 `json:"coordinates"`
			} `json:"geometry"`
			Properties struct {
				Name        string `json:"name"`
				Street      string `json:"street"`
				HouseNumber string `json:"housenumber"`
				City        string `json:"city"`
				District    string `json:"district"`
				State       string `json:"state"`
				Country     string `json:"country"`
				CountryCode string `json:"countrycode"`
				OSMKey      string `json:"osm_key"`
				OSMValue    string `json:"osm_value"`
			} `json:"properties"`
		} `json:"features"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("не вдалося обробити відповідь photon: %w", err)
	}

	rows := make([]geocodeCandidate, 0, len(payload.Features))
	for _, f := range payload.Features {
		if len(f.Geometry.Coordinates) < 2 {
			continue
		}
		lon := strconv.FormatFloat(f.Geometry.Coordinates[0], 'f', 7, 64)
		lat := strconv.FormatFloat(f.Geometry.Coordinates[1], 'f', 7, 64)
		address := map[string]string{
			"road":         strings.TrimSpace(f.Properties.Street),
			"house_number": strings.TrimSpace(f.Properties.HouseNumber),
			"city":         strings.TrimSpace(f.Properties.City),
			"district":     strings.TrimSpace(f.Properties.District),
			"state":        strings.TrimSpace(f.Properties.State),
			"country":      strings.TrimSpace(f.Properties.Country),
			"country_code": strings.TrimSpace(f.Properties.CountryCode),
		}

		displayParts := []string{
			strings.TrimSpace(f.Properties.Name),
			strings.TrimSpace(f.Properties.Street),
			strings.TrimSpace(f.Properties.HouseNumber),
			strings.TrimSpace(f.Properties.City),
			strings.TrimSpace(f.Properties.State),
		}
		displayFiltered := make([]string, 0, len(displayParts))
		for _, p := range displayParts {
			if p != "" {
				displayFiltered = append(displayFiltered, p)
			}
		}

		rows = append(rows, geocodeCandidate{
			Lat:         lat,
			Lon:         lon,
			DisplayName: strings.Join(displayFiltered, ", "),
			Address:     address,
			Importance:  0,
			Class:       strings.TrimSpace(f.Properties.OSMKey),
			Type:        strings.TrimSpace(f.Properties.OSMValue),
		})
	}

	return rows, nil
}

func geocodeAutocompleteCandidates(query string) ([]geocodeCandidate, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	rows, err := geocodeCandidatesPhoton(query)
	if err == nil && len(rows) > 0 {
		return rows, nil
	}
	if err == nil {
		return geocodeCandidatesForQuery(query)
	}

	fallbackRows, fallbackErr := geocodeCandidatesForQuery(query)
	if fallbackErr != nil {
		return nil, err
	}
	return fallbackRows, nil
}

func geocodeSuggestionOptions(rows []geocodeCandidate) ([]string, map[string]geocodeCandidate) {
	options := make([]string, 0, len(rows))
	items := make(map[string]geocodeCandidate, len(rows))
	seen := make(map[string]int, len(rows))
	for _, row := range rows {
		label := strings.TrimSpace(row.DisplayName)
		if label == "" {
			label = strings.TrimSpace(strings.Join([]string{
				firstAddressValue(row.Address, "road", "street", "pedestrian", "residential"),
				firstAddressValue(row.Address, "house_number"),
				firstAddressValue(row.Address, "city", "town", "village"),
			}, ", "))
		}
		label = strings.Trim(label, " ,")
		if label == "" {
			continue
		}
		if count := seen[label]; count > 0 {
			label = fmt.Sprintf("%s [%s, %s]", label, strings.TrimSpace(row.Lat), strings.TrimSpace(row.Lon))
		}
		seen[label]++
		options = append(options, label)
		items[label] = row
	}
	return options, items
}

func scoreGeocodeCandidate(target geocodeTarget, row geocodeCandidate) float64 {
	score := row.Importance * 10

	candidateCity := firstAddressValue(row.Address, "city", "town", "village", "hamlet", "municipality")
	candidateStreet := firstAddressValue(row.Address, "road", "pedestrian", "residential", "street")
	candidateHouse := firstAddressValue(row.Address, "house_number")

	if target.City != "" {
		score += similarityScore(target.City, candidateCity, 38, 18, -7)
	}
	if target.Street != "" {
		// Для вулиці перевіряємо також display_name, бо інколи road порожній.
		streetScore := similarityScore(target.Street, candidateStreet, 35, 16, -6)
		if streetScore < 0 {
			streetScore = similarityScore(target.Street, row.DisplayName, 22, 10, -3)
		}
		score += streetScore
	}
	if target.House != "" {
		score += houseMatchScore(target.House, candidateHouse, row.DisplayName)
	}

	if strings.EqualFold(strings.TrimSpace(row.Address["country_code"]), "ua") {
		score += 2
	}

	poiType := strings.ToLower(strings.TrimSpace(row.Type))
	poiClass := strings.ToLower(strings.TrimSpace(row.Class))
	if poiType == "house" || poiType == "building" || poiClass == "building" {
		score += 6
	}
	if poiClass == "boundary" {
		score -= 12
	}

	return score
}

func similarityScore(target string, candidate string, exact float64, partial float64, mismatch float64) float64 {
	t := normalizeGeoToken(target)
	c := normalizeGeoToken(candidate)
	if t == "" || c == "" {
		return 0
	}
	if t == c {
		return exact
	}
	if strings.Contains(c, t) || strings.Contains(t, c) {
		return partial
	}
	return mismatch
}

func houseMatchScore(targetHouse string, candidateHouse string, displayName string) float64 {
	t := normalizeHouseToken(targetHouse)
	if t == "" {
		return 0
	}
	c := normalizeHouseToken(candidateHouse)
	if c != "" {
		if c == t {
			return 36
		}
		if strings.HasPrefix(c, t) || strings.HasPrefix(t, c) {
			return 18
		}
		return -10
	}

	if strings.Contains(normalizeGeoToken(displayName), normalizeGeoToken(targetHouse)) {
		return 10
	}
	return -4
}

func normalizeGeoToken(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	v = strings.NewReplacer("’", "'", "`", "'", "ʼ", "'", "ё", "е", "ї", "і").Replace(v)

	var b strings.Builder
	b.Grow(len(v))
	for _, r := range v {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func normalizeHouseToken(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	token := strings.ToLower(strings.TrimSpace(re.FindString(v)))
	token = strings.ReplaceAll(token, " ", "")
	token = strings.ReplaceAll(token, "/", "")
	token = strings.ReplaceAll(token, "-", "")
	return token
}

func firstAddressValue(address map[string]string, keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(address[key]); v != "" {
			return v
		}
	}
	return ""
}

func buildGeocodeQueries(address string) []string {
	queries := make([]string, 0, 16)
	addQuery := func(v string) {
		v = normalizeAddressSpaces(v)
		if v == "" {
			return
		}
		for _, existing := range queries {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		queries = append(queries, v)
	}

	raw := normalizeAddressSpaces(address)
	cleaned := normalizeAddressForGeocode(raw)
	addQuery(raw)
	addQuery(cleaned)

	expanded := expandAddressAbbreviations(raw)
	expandedClean := expandAddressAbbreviations(cleaned)
	addQuery(expanded)
	addQuery(expandedClean)
	addQuery(ensureCountrySuffix(raw))
	addQuery(ensureCountrySuffix(cleaned))
	addQuery(ensureCountrySuffix(expanded))
	addQuery(ensureCountrySuffix(expandedClean))

	city, street, house, ok := parseAddressComponents(cleaned)
	if ok {
		if house != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", street, house, city))
			addQuery(fmt.Sprintf("%s, вулиця %s, %s, Україна", city, street, house))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, %s, Україна", street, city))
		addQuery(fmt.Sprintf("%s, вулиця %s, Україна", city, street))
	}

	if cityOnly, ok := parseCityOnly(cleaned); ok {
		addQuery(fmt.Sprintf("%s, Україна", cityOnly))
	} else if streetOnly, houseOnly, ok := parseStreetAndHouseOnly(cleaned); ok {
		const defaultCity = "Львів"
		if houseOnly != "" {
			addQuery(fmt.Sprintf("вулиця %s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
			addQuery(fmt.Sprintf("%s %s, %s, Україна", streetOnly, houseOnly, defaultCity))
		}
		addQuery(fmt.Sprintf("вулиця %s, %s, Україна", streetOnly, defaultCity))
		addQuery(fmt.Sprintf("%s, %s, Україна", streetOnly, defaultCity))
	}

	return queries
}

func parseAddressComponents(address string) (string, string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	clean := make([]string, 0, len(parts))
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p != "" {
			clean = append(clean, p)
		}
	}
	if len(clean) == 0 {
		return "", "", "", false
	}

	city := ""
	street := ""
	house := ""

	for _, p := range clean {
		if city == "" || street == "" {
			combinedCity, combinedStreet, combinedHouse, ok := splitCombinedLocalityStreetPart(p)
			if ok {
				if city == "" {
					city = combinedCity
				}
				if street == "" {
					street = combinedStreet
				}
				if house == "" && combinedHouse != "" {
					house = combinedHouse
				}
			}
		}

		if city == "" {
			if v, ok := extractCity(p); ok {
				city = v
				continue
			}
		}
		if street == "" {
			if v, ok := extractStreet(p); ok {
				street = v
			}
		}
		if house == "" {
			house = extractHouseNumber(p)
		}
	}

	if city == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if _, ok := extractStreet(p); ok {
				continue
			}
			if !isAdministrativePart(p) {
				city = normalizeAddressSpaces(p)
				break
			}
		}
	}
	if street == "" {
		for _, p := range clean {
			if extractHouseNumber(p) != "" {
				continue
			}
			if isAdministrativePart(p) {
				continue
			}
			street = normalizeStreetName(p)
			if street != "" {
				break
			}
		}
	}
	if house == "" {
		if h := extractHouseNumber(raw); h != "" {
			house = h
		}
	}

	if city == "" || street == "" {
		return "", "", "", false
	}
	return city, street, house, true
}

func parseCityOnly(address string) (string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		p = normalizeAddressSpaces(p)
		if p == "" {
			continue
		}
		if city, ok := extractCity(p); ok {
			return city, true
		}
	}
	return "", false
}

func parseStreetAndHouseOnly(address string) (string, string, bool) {
	raw := expandAddressAbbreviations(normalizeAddressSpaces(address))
	house := extractHouseNumber(raw)
	if house == "" {
		return "", "", false
	}
	street := normalizeStreetName(raw)
	if street == "" || isAdministrativePart(street) {
		return "", "", false
	}
	return street, house, true
}

func normalizeAddressSpaces(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	return strings.Join(strings.Fields(v), " ")
}

func normalizeAddressForGeocode(v string) string {
	v = strings.TrimSpace(v)
	v = strings.Trim(v, "\"'`")
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}

	v = strings.NewReplacer(
		"Львіська", "Львівська",
		"Червоноі", "Червоної",
		"Мечнікова", "Мечникова",
		"буд.", " ",
		"буд ", " ",
	).Replace(v)

	// Склеюємо випадки типу "Незалежност, і".
	letterComma := regexp.MustCompile(`([\p{L}]{2,})\s*,\s*([\p{L}])\b`)
	for i := 0; i < 3; i++ {
		nv := letterComma.ReplaceAllString(v, "$1$2")
		if nv == v {
			break
		}
		v = nv
	}

	// Вирізаємо дужки та службові примітки.
	v = regexp.MustCompile(`\([^)]*\)`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\[[^\]]*\]`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\{[^}]*\}`).ReplaceAllString(v, " ")

	// Прибираємо телефони та поштові індекси.
	v = regexp.MustCompile(`\b\d{2,4}[- ]\d{2}[- ]\d{2}(?:[- ]\d{2})?\b`).ReplaceAllString(v, " ")
	v = regexp.MustCompile(`\b\d{5}\b`).ReplaceAllString(v, " ")

	// Зайві службові хвости.
	if idx := indexOfAddressNoise(v); idx > 0 {
		v = v[:idx]
	}

	if idx := strings.Index(v, "+"); idx > 0 {
		v = v[:idx]
	}

	// Нормалізуємо розділювачі.
	v = strings.ReplaceAll(v, ";", ", ")
	v = strings.ReplaceAll(v, "|", ", ")
	v = regexp.MustCompile(`\s*,\s*`).ReplaceAllString(v, ", ")
	v = regexp.MustCompile(`\s*\.\s*`).ReplaceAllString(v, ". ")
	v = strings.Trim(v, " ,.-")
	return normalizeAddressSpaces(v)
}

func indexOfAddressNoise(v string) int {
	lower := strings.ToLower(v)
	keywords := []string{
		" ю/а ",
		" фактична",
		" завгосп",
		" централь",
		" охор",
		" режим роботи",
		" пожеж",
		" вхід ",
		" у дворі",
		" напроти ",
		" біля ",
		" на територ",
		" терітор",
		" корпус",
	}
	best := -1
	for _, kw := range keywords {
		idx := strings.Index(lower, kw)
		if idx >= 0 && (best < 0 || idx < best) {
			best = idx
		}
	}
	return best
}

func expandAddressAbbreviations(v string) string {
	v = normalizeAddressSpaces(v)
	abbrRules := []struct {
		pattern string
		repl    string
	}{
		{pattern: `(?i)(^|[\s,])м\.\s*`, repl: "${1}місто "},
		{pattern: `(?i)(^|[\s,])с\.\s*`, repl: "${1}село "},
		{pattern: `(?i)(^|[\s,])в\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])вуп\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])смт\.\s*`, repl: "${1}смт "},
		{pattern: `(?i)(^|[\s,])обл\.\s*`, repl: "${1}область "},
		{pattern: `(?i)(^|[\s,])вул\.\s*`, repl: "${1}вулиця "},
		{pattern: `(?i)(^|[\s,])пр\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])просп\.\s*`, repl: "${1}проспект "},
		{pattern: `(?i)(^|[\s,])пл\.\s*`, repl: "${1}площа "},
		{pattern: `(?i)(^|[\s,])бул\.\s*`, repl: "${1}бульвар "},
		{pattern: `(?i)(^|[\s,])пров\.\s*`, repl: "${1}провулок "},
	}
	for _, rule := range abbrRules {
		re := regexp.MustCompile(rule.pattern)
		v = re.ReplaceAllString(v, rule.repl)
	}

	repl := strings.NewReplacer(
		"м.", "місто ",
		"м ", "місто ",
		"с.", "село ",
		"с ", "село ",
		"в.", "вулиця ",
		"обл.", "область ",
		"обл ", "область ",
		"смт.", "смт ",
		"вул.", "вулиця ",
		"вул ", "вулиця ",
		"пр-т.", "проспект ",
		"пр-т", "проспект ",
		"просп.", "проспект ",
		"пл.", "площа ",
		"бул.", "бульвар ",
		"пров.", "провулок ",
	)
	v = repl.Replace(v)
	return normalizeAddressSpaces(v)
}

func ensureCountrySuffix(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	lower := strings.ToLower(v)
	if strings.Contains(lower, "україн") || strings.Contains(lower, "ukraine") {
		return v
	}
	return v + ", Україна"
}

func extractCity(v string) (string, bool) {
	v = strings.NewReplacer(
		"смт.", "смт ",
		"м.", "місто ",
		"місто.", "місто ",
		"село.", "село ",
		"с.", "село ",
	).Replace(v)
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	switch {
	case strings.HasPrefix(lower, "місто "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("місто "):])), true
	case strings.HasPrefix(lower, "смт "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("смт "):])), true
	case strings.HasPrefix(lower, "селище "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("селище "):])), true
	case strings.HasPrefix(lower, "село "):
		return normalizeAddressSpaces(strings.TrimSpace(v[len("село "):])), true
	}

	// Підтримка рядків типу:
	// "Львівська область місто Львів" / "Львівська область село Зимна Вода".
	for _, marker := range []string{" місто ", " смт ", " селище ", " село "} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			candidate := normalizeAddressSpaces(strings.TrimSpace(v[idx+len(marker):]))
			if candidate != "" {
				return candidate, true
			}
			before := normalizeAddressSpaces(strings.TrimSpace(v[:idx]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}

	// Підтримка "Яворів м." / "Яворів місто".
	for _, marker := range []string{" місто", " м"} {
		if strings.HasSuffix(lower, marker) {
			before := normalizeAddressSpaces(strings.TrimSpace(v[:len(v)-len(marker)]))
			if before != "" && !isAdministrativePart(before) {
				return before, true
			}
		}
	}
	return "", false
}

func extractStreet(v string) (string, bool) {
	v = normalizeAddressSpaces(v)
	lower := strings.ToLower(v)
	prefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}
	for _, pref := range prefixes {
		if strings.HasPrefix(lower, pref) {
			street := normalizeStreetName(strings.TrimSpace(v[len(pref):]))
			if street != "" {
				return street, true
			}
		}
	}

	// Підтримка "Маковея вулиця" / "Червоної Калини проспект".
	for _, pref := range prefixes {
		suffix := strings.TrimSpace(pref)
		if strings.HasSuffix(lower, " "+suffix) {
			street := normalizeStreetName(strings.TrimSpace(v[:len(v)-len(suffix)]))
			if street != "" {
				return street, true
			}
		}
	}
	return "", false
}

func splitCombinedLocalityStreetPart(part string) (string, string, string, bool) {
	part = normalizeAddressSpaces(part)
	if part == "" {
		return "", "", "", false
	}

	lower := strings.ToLower(part)
	streetPrefixes := []string{
		"вулиця ",
		"проспект ",
		"бульвар ",
		"площа ",
		"провулок ",
		"шосе ",
		"узвіз ",
	}

	streetIdx := -1
	for _, pref := range streetPrefixes {
		if strings.HasPrefix(lower, pref) {
			streetIdx = 0
			break
		}
		if idx := strings.Index(lower, " "+pref); idx >= 0 {
			idx++
			if streetIdx < 0 || idx < streetIdx {
				streetIdx = idx
			}
		}
	}
	if streetIdx <= 0 {
		return "", "", "", false
	}

	localityPart := normalizeAddressSpaces(part[:streetIdx])
	streetPart := normalizeAddressSpaces(part[streetIdx:])
	if localityPart == "" || streetPart == "" {
		return "", "", "", false
	}

	city, ok := extractCity(localityPart)
	if !ok {
		return "", "", "", false
	}

	street, ok := extractStreet(streetPart)
	if !ok {
		return "", "", "", false
	}
	house := extractHouseNumber(streetPart)
	return city, street, house, true
}

func normalizeStreetName(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	// Якщо номер будинку написали разом зі вулицею, відкидаємо номер.
	house := extractHouseNumber(v)
	if house != "" {
		v = strings.TrimSpace(strings.Replace(v, house, "", 1))
	}
	return normalizeAddressSpaces(v)
}

func extractHouseNumber(v string) string {
	v = normalizeAddressSpaces(v)
	if v == "" {
		return ""
	}
	re := regexp.MustCompile(`(?i)\d+[0-9\p{L}/-]*`)
	return normalizeAddressSpaces(re.FindString(v))
}

func isAdministrativePart(v string) bool {
	l := strings.ToLower(normalizeAddressSpaces(v))
	if l == "" {
		return true
	}
	adminWords := []string{
		"район",
		"область",
		"громада",
		"україна",
		"украина",
		"ukraine",
	}
	for _, w := range adminWords {
		if strings.Contains(l, w) {
			return true
		}
	}
	return false
}

func collectDistrictHints(address map[string]string, displayName string) []string {
	hints := make([]string, 0, 8)
	addHint := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		for _, existing := range hints {
			if strings.EqualFold(existing, v) {
				return
			}
		}
		hints = append(hints, v)
	}

	keys := []string{
		"city_district",
		"district",
		"suburb",
		"borough",
		"county",
		"state_district",
		"municipality",
	}
	for _, key := range keys {
		addHint(address[key])
	}

	parts := strings.Split(displayName, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.Contains(strings.ToLower(part), "район") {
			addHint(part)
		}
	}

	return hints
}

func resolveRegionByAddressHints(provider contracts.DistrictReferenceService, hints []string) (int64, string, error) {
	if len(hints) == 0 {
		return 0, "", fmt.Errorf("геосервіс не повернув район")
	}
	regions, err := provider.ListObjectDistricts()
	if err != nil {
		return 0, "", fmt.Errorf("не вдалося завантажити райони: %w", err)
	}
	if len(regions) == 0 {
		return 0, "", fmt.Errorf("довідник районів порожній")
	}

	type regionCandidate struct {
		ID   int64
		Name string
		Norm string
	}
	candidates := make([]regionCandidate, 0, len(regions))
	for _, region := range regions {
		name := strings.TrimSpace(region.Name)
		if name == "" || region.ID <= 0 {
			continue
		}
		candidates = append(candidates, regionCandidate{
			ID:   region.ID,
			Name: name,
			Norm: normalizeDistrictName(name),
		})
	}
	if len(candidates) == 0 {
		return 0, "", fmt.Errorf("не знайдено валідних районів у довіднику")
	}

	hintNorms := make([]string, 0, len(hints))
	for _, hint := range hints {
		if norm := normalizeDistrictName(hint); norm != "" {
			hintNorms = append(hintNorms, norm)
		}
	}
	if len(hintNorms) == 0 {
		return 0, "", fmt.Errorf("не вдалося витягнути назву району з адреси")
	}

	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm != "" && c.Norm == hintNorm {
				return c.ID, c.Name, nil
			}
		}
	}
	for _, hintNorm := range hintNorms {
		for _, c := range candidates {
			if c.Norm == "" {
				continue
			}
			if strings.Contains(hintNorm, c.Norm) || strings.Contains(c.Norm, hintNorm) {
				return c.ID, c.Name, nil
			}
		}
	}

	return 0, "", fmt.Errorf("район не зіставлено з довідником: %s", strings.Join(hints, ", "))
}

func normalizeDistrictName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"’", "'",
		"`", "'",
		"ʼ", "'",
		".", " ",
		",", " ",
		"(", " ",
		")", " ",
		"-", " ",
		"/", " ",
	)
	s = replacer.Replace(s)
	s = strings.ReplaceAll(s, "р-н", "район")

	tokens := strings.Fields(s)
	filtered := make([]string, 0, len(tokens))
	stopWords := map[string]struct{}{
		"район":   {},
		"місто":   {},
		"м":       {},
		"область": {},
		"обл":     {},
		"city":    {},
	}
	for _, t := range tokens {
		if _, skip := stopWords[t]; skip {
			continue
		}
		filtered = append(filtered, t)
	}
	return strings.Join(filtered, " ")
}

func showObjectPersonalEditor(
	parent fyne.Window,
	provider contracts.AdminObjectPersonalService,
	title string,
	initial contracts.AdminObjectPersonal,
	onSave func(item contracts.AdminObjectPersonal) error,
	statusLabel *widget.Label,
	onDone func(),
) {
	numberEntry := widget.NewEntry()
	if initial.Number > 0 {
		numberEntry.SetText(strconv.FormatInt(initial.Number, 10))
	}
	numberEntry.SetPlaceHolder("1..999")
	surnameEntry := widget.NewEntry()
	surnameEntry.SetText(initial.Surname)
	nameEntry := widget.NewEntry()
	nameEntry.SetText(initial.Name)
	secNameEntry := widget.NewEntry()
	secNameEntry.SetText(initial.SecName)
	addressEntry := widget.NewEntry()
	addressEntry.SetText(initial.Address)
	phonesEntry := widget.NewEntry()
	phonesEntry.SetText(initial.Phones)
	phoneLookupLabel := widget.NewLabel("")
	phoneLookupLabel.Wrapping = fyne.TextWrapWord
	positionEntry := widget.NewEntry()
	positionEntry.SetText(initial.Position)
	notesEntry := widget.NewEntry()
	notesEntry.SetText(initial.Notes)
	isRangCheck := widget.NewCheck("ISRANG (старший/ранг)", nil)
	isRangCheck.SetChecked(initial.IsRang)
	if initial.ID == 0 {
		isRangCheck.SetChecked(true)
	}
	accessCheck := widget.NewCheck("Повний доступ до адмін-функцій (ACCESS1=1)", nil)
	accessCheck.SetChecked(initial.Access1 > 0)
	viberIDEntry := widget.NewEntry()
	viberIDEntry.SetText(initial.ViberID)
	viberIDEntry.SetPlaceHolder("Viber ID (необов'язково)")
	telegramIDEntry := widget.NewEntry()
	telegramIDEntry.SetText(initial.TelegramID)
	telegramIDEntry.SetPlaceHolder("Telegram ID (необов'язково)")
	createdAtLabel := widget.NewLabel(initial.CreatedAt)
	if strings.TrimSpace(initial.CreatedAt) == "" {
		createdAtLabel.SetText("буде встановлено автоматично")
	}
	trkCheck := widget.NewCheck("Перевіряючий ТРК", nil)
	trkCheck.SetChecked(initial.IsTRKTester)

	digitsCount := func(s string) int {
		cnt := 0
		for _, r := range s {
			if r >= '0' && r <= '9' {
				cnt++
			}
		}
		return cnt
	}

	applyPersonalLookup := func(found *contracts.AdminObjectPersonal) {
		if found == nil {
			return
		}
		if strings.TrimSpace(numberEntry.Text) == "" && found.Number > 0 {
			numberEntry.SetText(strconv.FormatInt(found.Number, 10))
		}
		surnameEntry.SetText(strings.TrimSpace(found.Surname))
		nameEntry.SetText(strings.TrimSpace(found.Name))
		secNameEntry.SetText(strings.TrimSpace(found.SecName))
		addressEntry.SetText(strings.TrimSpace(found.Address))
		positionEntry.SetText(strings.TrimSpace(found.Position))
		notesEntry.SetText(strings.TrimSpace(found.Notes))
		isRangCheck.SetChecked(found.IsRang)
		accessCheck.SetChecked(found.Access1 > 0)
		viberIDEntry.SetText(strings.TrimSpace(found.ViberID))
		telegramIDEntry.SetText(strings.TrimSpace(found.TelegramID))
		trkCheck.SetChecked(found.IsTRKTester)
		if strings.TrimSpace(createdAtLabel.Text) == "" || createdAtLabel.Text == "буде встановлено автоматично" {
			if strings.TrimSpace(found.CreatedAt) != "" {
				createdAtLabel.SetText(found.CreatedAt)
			}
		}

		source := "Знайдено контакт у базі, дані підтягнуто автоматично"
		if found.SourceObjN > 0 {
			source = fmt.Sprintf("Знайдено контакт у базі (об'єкт #%d), дані підтягнуто автоматично", found.SourceObjN)
		}
		phoneLookupLabel.SetText(source)
	}

	lastPhoneLookupRaw := ""
	tryLookupByPhone := func(raw string) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			lastPhoneLookupRaw = ""
			phoneLookupLabel.SetText("")
			return
		}
		if digitsCount(raw) < 10 {
			phoneLookupLabel.SetText("")
			return
		}
		if raw == lastPhoneLookupRaw {
			return
		}
		lastPhoneLookupRaw = raw

		found, err := provider.FindPersonalByPhone(raw)
		if err != nil {
			phoneLookupLabel.SetText("Не вдалося перевірити телефон у базі")
			return
		}
		if found == nil {
			phoneLookupLabel.SetText("")
			return
		}
		applyPersonalLookup(found)
	}
	phonesEntry.OnChanged = func(text string) {
		tryLookupByPhone(text)
	}
	phonesEntry.OnSubmitted = func(text string) {
		tryLookupByPhone(text)
	}

	form := widget.NewForm(
		widget.NewFormItem("№:", numberEntry),
		widget.NewFormItem("Створено:", createdAtLabel),
		widget.NewFormItem("Прізвище:", surnameEntry),
		widget.NewFormItem("Ім'я:", nameEntry),
		widget.NewFormItem("По батькові:", secNameEntry),
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Телефон:", container.NewVBox(phonesEntry, phoneLookupLabel)),
		widget.NewFormItem("Посада:", positionEntry),
		widget.NewFormItem("Примітка:", notesEntry),
		widget.NewFormItem("", isRangCheck),
		widget.NewFormItem("", accessCheck),
		widget.NewFormItem("Viber ID:", viberIDEntry),
		widget.NewFormItem("Telegram ID:", telegramIDEntry),
		widget.NewFormItem("", trkCheck),
	)

	dlg := dialog.NewCustomConfirm(title, "Зберегти", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}

		numRaw := strings.TrimSpace(numberEntry.Text)
		number := int64(0)
		if numRaw != "" {
			n, err := strconv.ParseInt(numRaw, 10, 64)
			if err != nil {
				statusLabel.SetText("Некоректний номер В/О")
				return
			}
			number = n
		}

		item := contracts.AdminObjectPersonal{
			Number:      number,
			Surname:     strings.TrimSpace(surnameEntry.Text),
			Name:        strings.TrimSpace(nameEntry.Text),
			SecName:     strings.TrimSpace(secNameEntry.Text),
			Address:     strings.TrimSpace(addressEntry.Text),
			Phones:      strings.TrimSpace(phonesEntry.Text),
			Position:    strings.TrimSpace(positionEntry.Text),
			Notes:       strings.TrimSpace(notesEntry.Text),
			IsRang:      isRangCheck.Checked,
			Access1:     boolToInt64(accessCheck.Checked),
			ViberID:     strings.TrimSpace(viberIDEntry.Text),
			TelegramID:  strings.TrimSpace(telegramIDEntry.Text),
			CreatedAt:   strings.TrimSpace(createdAtLabel.Text),
			IsTRKTester: trkCheck.Checked,
		}
		if err := onSave(item); err != nil {
			dialog.ShowError(err, parent)
			statusLabel.SetText("Не вдалося зберегти В/О")
			return
		}
		onDone()
	}, parent)
	dlg.Show()
}

func showZoneFillDialog(parent fyne.Window, defaultCount int64, onApply func(count int64), statusLabel *widget.Label) {
	entry := widget.NewEntry()
	if defaultCount <= 0 {
		defaultCount = 24
	}
	entry.SetText(strconv.FormatInt(defaultCount, 10))
	entry.SetPlaceHolder("Кількість зон")

	form := widget.NewForm(
		widget.NewFormItem("Кількість зон:", entry),
	)

	dlg := dialog.NewCustomConfirm("Заповнення зон", "Застосувати", "Відміна", form, func(ok bool) {
		if !ok {
			return
		}
		count, err := strconv.ParseInt(strings.TrimSpace(entry.Text), 10, 64)
		if err != nil {
			statusLabel.SetText("Некоректна кількість зон")
			return
		}
		onApply(count)
	}, parent)
	dlg.Show()
}

func suggestZoneFillCount(provider contracts.AdminObjectZonesTabProvider, objn int64, current []contracts.AdminObjectZone) int64 {
	maxZone := int64(0)
	for _, z := range current {
		if z.ZoneNumber > maxZone {
			maxZone = z.ZoneNumber
		}
	}

	card, err := provider.GetObjectCard(objn)
	if err == nil && card.PPKID > 0 {
		ppkItems, ppkErr := provider.ListPPKConstructor()
		if ppkErr == nil {
			for _, it := range ppkItems {
				if it.ID == card.PPKID && it.ZoneCount > 0 {
					return it.ZoneCount
				}
			}
		}
	}

	if maxZone > 0 {
		return maxZone
	}
	return 24
}

func focusIfOnCanvas(parent fyne.Window, target fyne.Focusable) {
	if parent == nil || target == nil {
		return
	}
	canvas := parent.Canvas()
	if canvas == nil {
		return
	}
	root := canvas.Content()
	if root == nil {
		return
	}
	targetObj, ok := target.(fyne.CanvasObject)
	if !ok {
		return
	}
	if !containsCanvasObject(root, targetObj) {
		return
	}
	canvas.Focus(target)
}

func containsCanvasObject(root fyne.CanvasObject, target fyne.CanvasObject) bool {
	if root == nil || target == nil {
		return false
	}
	if root == target {
		return true
	}
	cont, ok := root.(*fyne.Container)
	if !ok {
		return false
	}
	for _, child := range cont.Objects {
		if containsCanvasObject(child, target) {
			return true
		}
	}
	return false
}

type mapInteractionSurface struct {
	widget.BaseWidget

	onTapped          func(*fyne.PointEvent)
	onTappedSecondary func(*fyne.PointEvent)
	onDragged         func(*fyne.DragEvent)
	onDragEnd         func()
	onScrolled        func(*fyne.ScrollEvent)
}

func newMapInteractionSurface() *mapInteractionSurface {
	surface := &mapInteractionSurface{}
	surface.ExtendBaseWidget(surface)
	return surface
}

func (s *mapInteractionSurface) Tapped(ev *fyne.PointEvent) {
	if s.onTapped != nil {
		s.onTapped(ev)
	}
}

func (s *mapInteractionSurface) TappedSecondary(ev *fyne.PointEvent) {
	if s.onTappedSecondary != nil {
		s.onTappedSecondary(ev)
	}
}

func (s *mapInteractionSurface) Dragged(ev *fyne.DragEvent) {
	if s.onDragged != nil {
		s.onDragged(ev)
	}
}

func (s *mapInteractionSurface) DragEnd() {
	if s.onDragEnd != nil {
		s.onDragEnd()
	}
}

func (s *mapInteractionSurface) Scrolled(ev *fyne.ScrollEvent) {
	if s.onScrolled != nil {
		s.onScrolled(ev)
	}
}

func (s *mapInteractionSurface) CreateRenderer() fyne.WidgetRenderer {
	// Мінімальна прозорість, щоб поверхня гарантовано брала pointer-події.
	hitBox := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 1})
	return widget.NewSimpleRenderer(hitBox)
}

type coordinatesMapPickerOptions struct {
	Title           string
	InitialAddress  string
	ForceLvivCenter bool
}

func showCoordinatesMapPicker(parent fyne.Window, initialLatRaw string, initialLonRaw string, onPick func(lat, lon string)) {
	showCoordinatesMapPickerWithOptions(parent, initialLatRaw, initialLonRaw, coordinatesMapPickerOptions{}, onPick)
}

type coordinatesMapPickerState struct {
	opts   coordinatesMapPickerOptions
	onPick func(lat, lon string)

	win             fyne.Window
	mapView         *xwidget.Map
	previousMarker  *canvas.Circle
	selectedMarker  *canvas.Circle
	selectedHalo    *canvas.Circle
	interaction     *mapInteractionSurface
	mapStack        fyne.CanvasObject
	searchEntry     *widget.SelectEntry
	searchStatus    *widget.Label
	centerLabel     *widget.Label
	selectedLabel   *widget.Label
	selectionLat    float64
	selectionLon    float64
	objectMarkerLat float64
	objectMarkerLon float64
	hasObjectMarker bool

	suggestionOptions map[string]geocodeCandidate
	suggestionMu      sync.Mutex
	suggestionReqID   int
	lastMarkerUpdate  time.Time
	lastCenterUpdate  time.Time
}

func showCoordinatesMapPickerWithOptions(parent fyne.Window, initialLatRaw string, initialLonRaw string, opts coordinatesMapPickerOptions, onPick func(lat, lon string)) {
	state := newCoordinatesMapPickerState(initialLatRaw, initialLonRaw, opts, onPick)
	state.win.SetContent(state.buildContent())
	state.bindSearchHandlers()
	state.bindInteractionHandlers()
	state.forceMapOverlayRefresh()
	state.win.Show()
}

func newCoordinatesMapPickerState(
	initialLatRaw string,
	initialLonRaw string,
	opts coordinatesMapPickerOptions,
	onPick func(lat, lon string),
) *coordinatesMapPickerState {
	centerLat, centerLon, zoom, hasObjectMarker := resolveInitialMapCenterWithOptions(initialLatRaw, initialLonRaw, opts.ForceLvivCenter)

	mapView := xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithZoomButtons(false),
		xwidget.WithScrollButtons(false),
		xwidget.AtZoomLevel(zoom),
		xwidget.AtLatLon(centerLat, centerLon),
	)

	state := &coordinatesMapPickerState{
		opts:              opts,
		onPick:            onPick,
		mapView:           mapView,
		previousMarker:    newMapPickerMarker(color.NRGBA{R: 255, G: 40, B: 40, A: 210}, 12),
		selectedMarker:    newMapPickerMarker(color.NRGBA{R: 25, G: 122, B: 255, A: 210}, 16),
		selectedHalo:      newMapPickerHalo(),
		interaction:       newMapInteractionSurface(),
		centerLabel:       widget.NewLabel("Центр: —"),
		selectedLabel:     widget.NewLabel(""),
		searchEntry:       widget.NewSelectEntry(nil),
		searchStatus:      widget.NewLabel(""),
		suggestionOptions: map[string]geocodeCandidate{},
		objectMarkerLat:   centerLat,
		objectMarkerLon:   centerLon,
		hasObjectMarker:   hasObjectMarker,
	}

	state.selectedLabel.TextStyle = fyne.TextStyle{Bold: true}
	state.searchEntry.SetPlaceHolder("Пошук адреси")
	state.searchEntry.SetText(strings.TrimSpace(opts.InitialAddress))

	state.selectionLat = centerLat
	state.selectionLon = centerLon
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		state.selectionLat = lat
		state.selectionLon = lon
	}

	state.mapStack = container.NewStack(
		state.mapView,
		container.NewWithoutLayout(state.previousMarker, state.selectedHalo, state.selectedMarker),
		state.interaction,
	)

	title := strings.TrimSpace(opts.Title)
	if title == "" {
		title = "Вибір координат на карті"
	}
	state.win = fyne.CurrentApp().NewWindow(title)
	state.win.Resize(fyne.NewSize(980, 680))
	state.updateSelectedLabel()

	return state
}

func newMapPickerMarker(fill color.NRGBA, size float32) *canvas.Circle {
	marker := canvas.NewCircle(fill)
	marker.StrokeColor = color.NRGBA{R: 255, G: 255, B: 255, A: 230}
	marker.StrokeWidth = 2
	marker.Resize(fyne.NewSize(size, size))
	marker.Hide()
	return marker
}

func newMapPickerHalo() *canvas.Circle {
	halo := canvas.NewCircle(color.NRGBA{R: 25, G: 122, B: 255, A: 70})
	halo.Resize(fyne.NewSize(28, 28))
	halo.Hide()
	return halo
}

func (s *coordinatesMapPickerState) buildContent() fyne.CanvasObject {
	centerLvivBtn := widget.NewButton("Львів", func() {
		s.mapView.PanToLatLon(mapDefaultLvivLat, mapDefaultLvivLon)
		s.forceMapOverlayRefresh()
	})
	useSelectionBtn := widget.NewButton("Підтвердити вибір", s.confirmSelection)
	setFromCenterBtn := widget.NewButton("Точка = центр", s.setSelectionFromCenter)
	centerOnSelectionBtn := widget.NewButton("До вибраної точки", func() {
		s.mapView.PanToLatLon(s.selectionLat, s.selectionLon)
		s.forceMapOverlayRefresh()
	})
	zoomInBtn := widget.NewButton("＋", func() {
		s.mapView.ZoomIn()
		s.forceMapOverlayRefresh()
	})
	zoomOutBtn := widget.NewButton("－", func() {
		s.mapView.ZoomOut()
		s.forceMapOverlayRefresh()
	})
	refreshBtn := widget.NewButton("Оновити", s.forceMapOverlayRefresh)
	mapSettingsBtn := widget.NewButton("Налаштування карти", s.openMapSettings)
	cancelBtn := widget.NewButton("Скасувати", func() { s.win.Close() })

	return container.NewBorder(
		container.NewVBox(
			widget.NewLabel("ЛКМ: вибір точки | ПКМ: вибір + центрування | Колесо: зум | Перетягування: панорама."),
			widget.NewLabel("Червоний маркер: поточна точка об'єкта. Синій маркер: точка, яку ви обрали."),
			container.NewBorder(
				nil,
				nil,
				nil,
				container.NewHBox(widget.NewButton("Знайти адресу", s.runAddressSearch), centerLvivBtn),
				s.searchEntry,
			),
			s.searchStatus,
			widget.NewSeparator(),
		),
		container.NewVBox(
			container.NewHBox(s.centerLabel, layout.NewSpacer(), s.selectedLabel),
			container.NewHBox(
				widget.NewLabel("Зум:"),
				zoomOutBtn,
				zoomInBtn,
				layout.NewSpacer(),
				mapSettingsBtn,
				refreshBtn,
				setFromCenterBtn,
				centerOnSelectionBtn,
				useSelectionBtn,
				cancelBtn,
			),
		),
		nil,
		nil,
		s.mapStack,
	)
}

func (s *coordinatesMapPickerState) bindSearchHandlers() {
	s.searchEntry.OnSubmitted = func(string) {
		s.runAddressSearch()
	}
	s.searchEntry.OnChanged = s.handleSearchChange
}

func (s *coordinatesMapPickerState) bindInteractionHandlers() {
	s.interaction.onTapped = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(s.mapView, ev.Position.X, ev.Position.Y)
		if err == nil {
			s.setSelectionAt(lat, lon)
		}
	}
	s.interaction.onTappedSecondary = func(ev *fyne.PointEvent) {
		lat, lon, err := mapCanvasPointToLatLon(s.mapView, ev.Position.X, ev.Position.Y)
		if err != nil {
			return
		}
		s.setSelectionAt(lat, lon)
		s.mapView.PanToLatLon(lat, lon)
		s.forceMapOverlayRefresh()
	}
	s.interaction.onDragged = func(ev *fyne.DragEvent) {
		s.mapView.Dragged(ev)
		s.updateMapOverlayDuringDrag()
	}
	s.interaction.onDragEnd = func() {
		s.mapView.DragEnd()
		s.forceMapOverlayRefresh()
	}
	s.interaction.onScrolled = func(ev *fyne.ScrollEvent) {
		s.handleScroll(ev)
	}
}

func (s *coordinatesMapPickerState) updateCenterLabel() {
	lat, lon, err := mapCenterLatLon(s.mapView)
	if err != nil {
		s.centerLabel.SetText("Центр: невизначено")
		return
	}

	zoomText := "?"
	if state, stateErr := readMapInternalState(s.mapView); stateErr == nil {
		zoomText = strconv.Itoa(state.zoom)
	}
	s.centerLabel.SetText(fmt.Sprintf("Центр: %s, %s | Z=%s", formatCoordinate(lat), formatCoordinate(lon), zoomText))
}

func (s *coordinatesMapPickerState) updateSelectedLabel() {
	s.selectedLabel.SetText(fmt.Sprintf("Вибрана точка: %s, %s", formatCoordinate(s.selectionLat), formatCoordinate(s.selectionLon)))
}

func (s *coordinatesMapPickerState) updateMarkers() {
	s.updateObjectMarker()
	s.updateSelectionMarker()
}

func (s *coordinatesMapPickerState) updateObjectMarker() {
	if !s.hasObjectMarker {
		s.previousMarker.Hide()
		return
	}

	x, y, ok := mapLatLonToCanvasPoint(s.mapView, s.objectMarkerLat, s.objectMarkerLon)
	if !ok || !pointWithinMapBounds(s.mapView, x, y, 20) {
		s.previousMarker.Hide()
		return
	}

	size := s.previousMarker.Size()
	s.previousMarker.Move(fyne.NewPos(x-size.Width/2, y-size.Height/2))
	s.previousMarker.Show()
	s.previousMarker.Refresh()
}

func (s *coordinatesMapPickerState) updateSelectionMarker() {
	x, y, ok := mapLatLonToCanvasPoint(s.mapView, s.selectionLat, s.selectionLon)
	if !ok || !pointWithinMapBounds(s.mapView, x, y, 30) {
		s.selectedMarker.Hide()
		s.selectedHalo.Hide()
		return
	}

	haloSize := s.selectedHalo.Size()
	markerSize := s.selectedMarker.Size()
	s.selectedHalo.Move(fyne.NewPos(x-haloSize.Width/2, y-haloSize.Height/2))
	s.selectedMarker.Move(fyne.NewPos(x-markerSize.Width/2, y-markerSize.Height/2))
	s.selectedHalo.Show()
	s.selectedMarker.Show()
	s.selectedHalo.Refresh()
	s.selectedMarker.Refresh()
}

func pointWithinMapBounds(mapView *xwidget.Map, x float32, y float32, padding float32) bool {
	size := mapView.Size()
	return x >= -padding && y >= -padding && x <= size.Width+padding && y <= size.Height+padding
}

func (s *coordinatesMapPickerState) setSelectionAt(lat, lon float64) {
	s.selectionLat = lat
	s.selectionLon = lon
	s.updateSelectedLabel()
	s.updateSelectionMarker()
}

func (s *coordinatesMapPickerState) forceMapOverlayRefresh() {
	s.updateCenterLabel()
	s.updateMarkers()
	now := time.Now()
	s.lastMarkerUpdate = now
	s.lastCenterUpdate = now
}

func (s *coordinatesMapPickerState) updateMapOverlayDuringDrag() {
	now := time.Now()
	if now.Sub(s.lastMarkerUpdate) >= 80*time.Millisecond {
		s.updateMarkers()
		s.lastMarkerUpdate = now
	}
	if now.Sub(s.lastCenterUpdate) >= 220*time.Millisecond {
		s.updateCenterLabel()
		s.lastCenterUpdate = now
	}
}

func (s *coordinatesMapPickerState) setSuggestionState(options []string, items map[string]geocodeCandidate) {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	s.suggestionOptions = items
	s.searchEntry.SetOptions(options)
}

func (s *coordinatesMapPickerState) nextSuggestionRequestID() int {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	s.suggestionReqID++
	return s.suggestionReqID
}

func (s *coordinatesMapPickerState) isCurrentSuggestionRequest(id int) bool {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	return id == s.suggestionReqID
}

func (s *coordinatesMapPickerState) suggestionForValue(value string) (geocodeCandidate, bool) {
	s.suggestionMu.Lock()
	defer s.suggestionMu.Unlock()
	candidate, ok := s.suggestionOptions[value]
	return candidate, ok
}

func (s *coordinatesMapPickerState) runAddressSearch() {
	address := strings.TrimSpace(s.searchEntry.Text)
	if address == "" {
		s.searchStatus.SetText("Вкажіть адресу для пошуку")
		return
	}

	s.searchStatus.SetText("Пошук адреси...")
	go func() {
		latRaw, lonRaw, _, err := geocodeAddress(address)
		fyne.Do(func() {
			if err != nil {
				s.searchStatus.SetText("Адресу не знайдено")
				dialog.ShowError(err, s.win)
				return
			}

			lat, latErr := parseCoordinate(latRaw)
			lon, lonErr := parseCoordinate(lonRaw)
			if latErr != nil || lonErr != nil {
				s.searchStatus.SetText("Сервіс повернув некоректні координати")
				dialog.ShowError(fmt.Errorf("не вдалося розпізнати координати адреси"), s.win)
				return
			}

			s.setSelectionAt(lat, lon)
			s.mapView.PanToLatLon(lat, lon)
			s.forceMapOverlayRefresh()
			s.searchStatus.SetText(fmt.Sprintf("Знайдено: %s, %s", formatCoordinate(lat), formatCoordinate(lon)))
		})
	}()
}

func (s *coordinatesMapPickerState) applySuggestion(candidate geocodeCandidate) {
	lat, latErr := parseCoordinate(candidate.Lat)
	lon, lonErr := parseCoordinate(candidate.Lon)
	if latErr != nil || lonErr != nil {
		s.searchStatus.SetText("Підказка містить некоректні координати")
		return
	}

	s.setSelectionAt(lat, lon)
	s.mapView.PanToLatLon(lat, lon)
	s.forceMapOverlayRefresh()
	s.searchStatus.SetText(fmt.Sprintf("Підказка: %s", firstNonEmpty(candidate.DisplayName, s.searchEntry.Text)))
}

func (s *coordinatesMapPickerState) handleSearchChange(value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		s.setSuggestionState(nil, map[string]geocodeCandidate{})
		s.searchStatus.SetText("")
		return
	}

	if candidate, ok := s.suggestionForValue(value); ok {
		s.applySuggestion(candidate)
		return
	}

	if len([]rune(value)) < 3 {
		s.setSuggestionState(nil, map[string]geocodeCandidate{})
		s.searchStatus.SetText("Введіть щонайменше 3 символи для підказок")
		return
	}

	reqID := s.nextSuggestionRequestID()
	s.searchStatus.SetText("Пошук підказок...")
	go func(query string, expectedReqID int) {
		time.Sleep(350 * time.Millisecond)
		if !s.isCurrentSuggestionRequest(expectedReqID) {
			return
		}

		rows, err := geocodeAutocompleteCandidates(query)
		fyne.Do(func() {
			if !s.isCurrentSuggestionRequest(expectedReqID) {
				return
			}
			if err != nil {
				s.setSuggestionState(nil, map[string]geocodeCandidate{})
				s.searchStatus.SetText("Не вдалося завантажити підказки")
				return
			}

			options, items := geocodeSuggestionOptions(rows)
			s.setSuggestionState(options, items)
			if len(options) == 0 {
				s.searchStatus.SetText("Підказки не знайдено")
				return
			}
			s.searchStatus.SetText(fmt.Sprintf("Знайдено підказок: %d", len(options)))
		})
	}(value, reqID)
}

func (s *coordinatesMapPickerState) handleScroll(ev *fyne.ScrollEvent) {
	delta := ev.Scrolled.DY
	if math.Abs(float64(ev.Scrolled.DX)) > math.Abs(float64(delta)) {
		delta = ev.Scrolled.DX
	}

	steps := mapScrollStepCount(delta)
	if steps == 0 {
		return
	}

	centerLat, centerLon, centerErr := mapCenterLatLon(s.mapView)
	if delta > 0 {
		for range steps {
			s.mapView.ZoomIn()
		}
	} else {
		for range steps {
			s.mapView.ZoomOut()
		}
	}
	if centerErr == nil {
		s.mapView.PanToLatLon(centerLat, centerLon)
	}
	s.forceMapOverlayRefresh()
}

func (s *coordinatesMapPickerState) confirmSelection() {
	centerLat, centerLon, err := mapCenterLatLon(s.mapView)
	if err == nil {
		saveLastMapCenter(centerLat, centerLon)
	}
	if s.onPick != nil {
		s.onPick(formatCoordinate(s.selectionLat), formatCoordinate(s.selectionLon))
	}
	s.win.Close()
}

func (s *coordinatesMapPickerState) setSelectionFromCenter() {
	lat, lon, err := mapCenterLatLon(s.mapView)
	if err != nil {
		dialog.ShowError(err, s.win)
		return
	}
	s.setSelectionAt(lat, lon)
}

func (s *coordinatesMapPickerState) openMapSettings() {
	showMapCenterSettingsDialog(s.win, func(lat, lon float64, zoom int) {
		s.mapView.Zoom(zoom)
		s.mapView.PanToLatLon(lat, lon)
		s.forceMapOverlayRefresh()
	})
}

func parseLatLon(latRaw string, lonRaw string) (float64, float64, bool) {
	lat, err := parseCoordinate(latRaw)
	if err != nil {
		return 0, 0, false
	}
	lon, err := parseCoordinate(lonRaw)
	if err != nil {
		return 0, 0, false
	}
	if lat < -85 || lat > 85 {
		return 0, 0, false
	}
	if lon < -180 || lon > 180 {
		return 0, 0, false
	}
	return lat, lon, true
}

func parseCoordinate(raw string) (float64, error) {
	clean := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if clean == "" {
		return 0, fmt.Errorf("empty coordinate")
	}
	return strconv.ParseFloat(clean, 64)
}

func formatCoordinate(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func mapCenterLatLon(m *xwidget.Map) (float64, float64, error) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, err
	}
	xTile := state.mx + (state.centerX-state.midTileX-state.offsetX*state.scale)/state.tilePx
	yTile := state.my + (state.centerY-state.midTileY-state.offsetY*state.scale)/state.tilePx
	return tileXYToLatLon(xTile, yTile, state.n)
}

func mapLatLonToCanvasPoint(m *xwidget.Map, lat float64, lon float64) (float32, float32, bool) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, false
	}
	xTile, yTile := latLonToTileXY(lat, lon, state.n)
	px := state.midTileX + (xTile-state.mx)*state.tilePx + state.offsetX*state.scale
	py := state.midTileY + (yTile-state.my)*state.tilePx + state.offsetY*state.scale
	if math.IsNaN(px) || math.IsNaN(py) || math.IsInf(px, 0) || math.IsInf(py, 0) {
		return 0, 0, false
	}
	return float32(px / state.scale), float32(py / state.scale), true
}

func mapCanvasPointToLatLon(m *xwidget.Map, x float32, y float32) (float64, float64, error) {
	state, err := readMapInternalState(m)
	if err != nil {
		return 0, 0, err
	}
	px := float64(x) * state.scale
	py := float64(y) * state.scale
	xTile := state.mx + (px-state.midTileX-state.offsetX*state.scale)/state.tilePx
	yTile := state.my + (py-state.midTileY-state.offsetY*state.scale)/state.tilePx
	return tileXYToLatLon(xTile, yTile, state.n)
}

func mapScrollStepCount(deltaY float32) int {
	abs := math.Abs(float64(deltaY))
	if abs < 0.05 {
		return 0
	}
	// Робимо зум плавним: один рівень за одну подію прокрутки.
	return 1
}

type mapInternalState struct {
	mx, my             float64
	zoom               int
	n                  float64
	offsetX, offsetY   float64
	scale              float64
	centerX, centerY   float64
	midTileX, midTileY float64
	tilePx             float64
}

func readMapInternalState(m *xwidget.Map) (mapInternalState, error) {
	if m == nil {
		return mapInternalState{}, fmt.Errorf("map is nil")
	}

	mv := reflect.ValueOf(m)
	if mv.Kind() != reflect.Pointer || mv.IsNil() {
		return mapInternalState{}, fmt.Errorf("invalid map value")
	}
	me := mv.Elem()

	getIntField := func(name string) (int, error) {
		f := me.FieldByName(name)
		if !f.IsValid() || f.Kind() != reflect.Int {
			return 0, fmt.Errorf("map field %s is unavailable", name)
		}
		return int(f.Int()), nil
	}
	getFloatField := func(name string) (float64, error) {
		f := me.FieldByName(name)
		if !f.IsValid() {
			return 0, fmt.Errorf("map field %s is unavailable", name)
		}
		switch f.Kind() {
		case reflect.Float32, reflect.Float64:
			return f.Float(), nil
		default:
			return 0, fmt.Errorf("map field %s has unsupported type", name)
		}
	}

	x, err := getIntField("x")
	if err != nil {
		return mapInternalState{}, err
	}
	y, err := getIntField("y")
	if err != nil {
		return mapInternalState{}, err
	}
	zoom, err := getIntField("zoom")
	if err != nil {
		return mapInternalState{}, err
	}
	offsetX, err := getFloatField("offsetX")
	if err != nil {
		return mapInternalState{}, err
	}
	offsetY, err := getFloatField("offsetY")
	if err != nil {
		return mapInternalState{}, err
	}

	if zoom < 0 || zoom > 19 {
		return mapInternalState{}, fmt.Errorf("invalid zoom level")
	}
	count := 1 << zoom
	n := float64(count)
	half := int(float32(count)/2 - 0.5)
	mx := x + half
	my := y + half

	scale := float64(1)
	if c := fyne.CurrentApp().Driver().CanvasForObject(m); c != nil {
		scale = float64(c.Scale())
		if scale <= 0 {
			scale = 1
		}
	}

	size := m.Size()
	wPx := int(math.Round(float64(size.Width) * scale))
	hPx := int(math.Round(float64(size.Height) * scale))
	if wPx <= 0 || hPx <= 0 {
		return mapInternalState{}, fmt.Errorf("map is not sized yet")
	}

	tilePx := int(math.Round(256 * scale))
	if tilePx <= 0 {
		return mapInternalState{}, fmt.Errorf("invalid tile size")
	}

	midTileX := (wPx - tilePx*2) / 2
	midTileY := (hPx - tilePx*2) / 2
	if zoom == 0 {
		midTileX += tilePx / 2
		midTileY += tilePx / 2
	}

	return mapInternalState{
		mx:       float64(mx),
		my:       float64(my),
		zoom:     zoom,
		n:        n,
		offsetX:  offsetX,
		offsetY:  offsetY,
		scale:    scale,
		centerX:  float64(wPx) / 2,
		centerY:  float64(hPx) / 2,
		midTileX: float64(midTileX),
		midTileY: float64(midTileY),
		tilePx:   float64(tilePx),
	}, nil
}

func latLonToTileXY(lat float64, lon float64, n float64) (float64, float64) {
	xTile := (lon + 180.0) / 360.0 * n
	latRad := lat * math.Pi / 180.0
	yTile := (1.0 - math.Log(math.Tan(latRad)+1.0/math.Cos(latRad))/math.Pi) / 2.0 * n
	return xTile, yTile
}

func tileXYToLatLon(xTile float64, yTile float64, n float64) (float64, float64, error) {
	lon := xTile/n*360.0 - 180.0
	latRad := math.Atan(math.Sinh(math.Pi * (1 - 2*yTile/n)))
	lat := latRad * 180.0 / math.Pi
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return 0, 0, fmt.Errorf("failed to resolve coordinates")
	}
	return lat, lon, nil
}

func resolveInitialMapCenter(initialLatRaw string, initialLonRaw string) (float64, float64, int, bool) {
	return resolveInitialMapCenterWithOptions(initialLatRaw, initialLonRaw, false)
}

func resolveInitialMapCenterWithOptions(initialLatRaw string, initialLonRaw string, forceLvivCenter bool) (float64, float64, int, bool) {
	if lat, lon, ok := parseLatLon(initialLatRaw, initialLonRaw); ok {
		return lat, lon, mapDefaultZoom, true
	}
	if forceLvivCenter {
		return mapDefaultLvivLat, mapDefaultLvivLon, mapDefaultZoom, false
	}

	mode := mapCenterModeLviv
	prefs := fyne.CurrentApp().Preferences()
	if prefs != nil {
		if m := strings.TrimSpace(prefs.String(mapCenterModePrefKey)); m != "" {
			mode = m
		}
	}

	switch mode {
	case mapCenterModeKyiv:
		return mapDefaultKyivLat, mapDefaultKyivLon, mapDefaultZoom, false
	case mapCenterModeCustom:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterCustomLatKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterCustomLonKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapDefaultZoom, false
			}
		}
	case mapCenterModeLast:
		if prefs != nil {
			lat, latErr := parseCoordinate(prefs.String(mapCenterLastLatPrefKey))
			lon, lonErr := parseCoordinate(prefs.String(mapCenterLastLonPrefKey))
			if latErr == nil && lonErr == nil && lat >= -85 && lat <= 85 && lon >= -180 && lon <= 180 {
				return lat, lon, mapDefaultZoom, false
			}
		}
	}

	return mapDefaultLvivLat, mapDefaultLvivLon, mapDefaultZoom, false
}

func saveLastMapCenter(lat float64, lon float64) {
	prefs := fyne.CurrentApp().Preferences()
	if prefs == nil {
		return
	}
	prefs.SetString(mapCenterLastLatPrefKey, formatCoordinate(lat))
	prefs.SetString(mapCenterLastLonPrefKey, formatCoordinate(lon))
}

func showMapCenterSettingsDialog(parent fyne.Window, onApply func(lat, lon float64, zoom int)) {
	prefs := fyne.CurrentApp().Preferences()
	mode := mapCenterModeLviv
	customLat := "49.8397"
	customLon := "24.0297"
	if prefs != nil {
		if m := strings.TrimSpace(prefs.String(mapCenterModePrefKey)); m != "" {
			mode = m
		}
		if v := strings.TrimSpace(prefs.String(mapCenterCustomLatKey)); v != "" {
			customLat = v
		}
		if v := strings.TrimSpace(prefs.String(mapCenterCustomLonKey)); v != "" {
			customLon = v
		}
	}

	modeSelect := widget.NewSelect([]string{
		"Львів",
		"Київ",
		"Власні координати",
		"Остання вибрана точка",
	}, nil)
	switch mode {
	case mapCenterModeKyiv:
		modeSelect.SetSelected("Київ")
	case mapCenterModeCustom:
		modeSelect.SetSelected("Власні координати")
	case mapCenterModeLast:
		modeSelect.SetSelected("Остання вибрана точка")
	default:
		modeSelect.SetSelected("Львів")
	}

	customLatEntry := widget.NewEntry()
	customLonEntry := widget.NewEntry()
	customLatEntry.SetText(customLat)
	customLonEntry.SetText(customLon)
	customLatEntry.SetPlaceHolder("49.8397")
	customLonEntry.SetPlaceHolder("24.0297")

	updateCustomState := func() {
		enabled := modeSelect.Selected == "Власні координати"
		if enabled {
			customLatEntry.Enable()
			customLonEntry.Enable()
			return
		}
		customLatEntry.Disable()
		customLonEntry.Disable()
	}
	modeSelect.OnChanged = func(string) { updateCustomState() }
	updateCustomState()

	form := widget.NewForm(
		widget.NewFormItem("Центр мапи при відкритті:", modeSelect),
		widget.NewFormItem("Широта (власна):", customLatEntry),
		widget.NewFormItem("Довгота (власна):", customLonEntry),
	)

	dialog.ShowCustomConfirm(
		"Налаштування карти",
		"Зберегти",
		"Скасувати",
		container.NewPadded(form),
		func(ok bool) {
			if !ok {
				return
			}

			selectedMode := mapCenterModeLviv
			switch modeSelect.Selected {
			case "Київ":
				selectedMode = mapCenterModeKyiv
			case "Власні координати":
				selectedMode = mapCenterModeCustom
			case "Остання вибрана точка":
				selectedMode = mapCenterModeLast
			}

			customLatVal := strings.TrimSpace(customLatEntry.Text)
			customLonVal := strings.TrimSpace(customLonEntry.Text)
			if selectedMode == mapCenterModeCustom {
				lat, lon, ok := parseLatLon(customLatVal, customLonVal)
				if !ok {
					dialog.ShowError(fmt.Errorf("некоректні власні координати"), parent)
					return
				}
				customLatVal = formatCoordinate(lat)
				customLonVal = formatCoordinate(lon)
			}

			if prefs != nil {
				prefs.SetString(mapCenterModePrefKey, selectedMode)
				prefs.SetString(mapCenterCustomLatKey, customLatVal)
				prefs.SetString(mapCenterCustomLonKey, customLonVal)
			}

			if onApply != nil {
				lat, lon, zoom, _ := resolveInitialMapCenter("", "")
				onApply(lat, lon, zoom)
			}
		},
		parent,
	)
}

func boolToInt64(v bool) int64 {
	if v {
		return 1
	}
	return 0
}
