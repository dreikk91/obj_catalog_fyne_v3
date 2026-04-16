package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
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
