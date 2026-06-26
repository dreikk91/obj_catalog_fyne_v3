//go:build qt

package qtui

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type WorkAreaPanel struct {
	*qt.QWidget
	tabs                     *qt.QTabWidget
	headerName               *qt.QLabel
	headerAddress            *qt.QLabel
	cardFields               map[string]*qt.QLineEdit
	cardNotes                *qt.QTextEdit
	deviceVM                 *viewmodels.WorkAreaDeviceViewModel
	zonesModel               *qt.QStandardItemModel
	zonesFlatModel           *qt.QStandardItemModel
	contactsModel            *qt.QStandardItemModel
	eventsModel              *qt.QStandardItemModel
	zonesStack               *qt.QStackedWidget
	zonesTree                *qt.QTreeView
	zonesTable               *qt.QTableView
	contactsTable            *qt.QTableView
	eventsTable              *qt.QTableView
	columnAutoSized          map[string]bool
	columnManualSized        map[string]bool
	OnEditObjectRequested    func()
	OnSIMManagementRequested func()
	OnDialPhoneRequested     func(phone string)

	// Export fields
	currentObject         *models.Object
	zones                 []models.Zone
	contacts              []models.Contact
	events                []models.Event
	eventsLoadedObjectID  int
	eventsLoadingObjectID int
	eventsRowsSignature   string
	eventsRowsReady       bool
	eventsCacheMu         sync.Mutex
	eventsCache           map[int]objectEventsCacheEntry
	eventsCacheOrder      []int
	dataProvider          contracts.DataProvider
	prefs                 config.Preferences
	viewModel             *viewmodels.WorkAreaViewModel
	exportVM              *viewmodels.WorkAreaExportViewModel
	exportPDFBtn          *qt.QPushButton
	exportXLSXBtn         *qt.QPushButton
	copyExcelBtn          *qt.QPushButton
	addToDeletedBtn       *qt.QPushButton
	OnRunOnMainThread     func(f func())
}

const (
	objectEventsCacheLimit = 8
	objectEventsCacheTTL   = 2 * time.Minute
)

type objectEventsCacheEntry struct {
	events   []models.Event
	loadedAt time.Time
}

func NewWorkAreaPanel(prefs config.Preferences) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		QWidget:     qt.NewQWidget2(),
		deviceVM:    viewmodels.NewWorkAreaDeviceViewModel(),
		viewModel:   viewmodels.NewWorkAreaViewModel(),
		exportVM:    viewmodels.NewWorkAreaExportViewModel(),
		prefs:       prefs,
		eventsCache: map[int]objectEventsCacheEntry{},
	}
	layout := qt.NewQVBoxLayout(panel.QWidget)
	panel.headerName = qt.NewQLabel3("Оберіть об'єкт зі списку")
	panel.headerName.SetStyleSheet("font-weight: 600; font-size: 12pt;")
	panel.headerAddress = qt.NewQLabel3("")
	panel.headerAddress.SetWordWrap(true)
	actionsLayout := qt.NewQHBoxLayout2()
	editButton := qt.NewQPushButton3("Редагувати")
	editButton.OnClicked(func() {
		if panel.OnEditObjectRequested != nil {
			panel.OnEditObjectRequested()
		}
	})
	simButton := qt.NewQPushButton3("SIM")
	simButton.OnClicked(func() {
		if panel.OnSIMManagementRequested != nil {
			panel.OnSIMManagementRequested()
		}
	})
	actionsLayout.AddWidget(editButton.QWidget)
	actionsLayout.AddWidget(simButton.QWidget)
	actionsLayout.AddStretch()
	panel.tabs = qt.NewQTabWidget2()
	panel.tabs.AddTab(panel.buildObjectCardTab(), "Картка")
	panel.zonesModel = qt.NewQStandardItemModel2(0, 6)
	panel.zonesTree = newTree(panel.zonesModel, zoneTreeHeaders())
	panel.installTreeColumnContextMenu("object_zones", panel.zonesTree)
	panel.zonesFlatModel = qt.NewQStandardItemModel2(0, 4)
	panel.zonesTable = newTable(panel.zonesFlatModel, zoneTableHeaders())
	panel.installTableColumnContextMenu("object_zones_flat", panel.zonesTable)
	panel.zonesStack = qt.NewQStackedWidget2()
	panel.zonesStack.AddWidget(panel.zonesTable.QWidget)
	panel.zonesStack.AddWidget(panel.zonesTree.QWidget)
	panel.tabs.AddTab(panel.zonesStack.QWidget, "Зони")
	panel.contactsModel = qt.NewQStandardItemModel2(0, 4)
	panel.contactsTable = newTable(panel.contactsModel, []string{"Ім'я", "Посада", "Телефон", "Група"})
	panel.contactsTable.SetContextMenuPolicy(qt.CustomContextMenu)
	panel.contactsTable.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		panel.showContactContextMenu(pos)
	})
	panel.tabs.AddTab(panel.contactsTable.QWidget, "Контакти")
	panel.eventsModel = qt.NewQStandardItemModel2(0, 3)
	panel.eventsTable = newTable(panel.eventsModel, []string{"Час", "Подія", "Опис"})
	panel.installTableColumnContextMenu("object_events", panel.eventsTable)
	panel.tabs.AddTab(panel.eventsTable.QWidget, "Журнал")
	panel.tabs.AddTab(panel.buildExportTab(), "Експорт")
	panel.tabs.OnCurrentChanged(func(index int) {
		if panel.tabs.TabText(index) == "Журнал" {
			panel.loadEventsForCurrentObject()
		}
	})
	layout.AddWidget(panel.headerName.QWidget)
	layout.AddWidget(panel.headerAddress.QWidget)
	layout.AddLayout(actionsLayout.QLayout)
	layout.AddWidget(panel.tabs.QWidget)
	panel.SetLayout(layout.QLayout)
	return panel
}

func (panel *WorkAreaPanel) showContactContextMenu(pos *qt.QPoint) {
	if panel == nil || panel.contactsTable == nil || panel.contactsModel == nil || pos == nil {
		return
	}
	menu := qt.NewQMenu(panel.contactsTable.QWidget)
	index := panel.contactsTable.IndexAt(pos)
	if index.IsValid() {
		phone := panel.contactPhoneAtIndex(index)
		if phone != "" {
			panel.contactsTable.SelectRow(index.Row())
			dialAction := menu.AddActionWithText("Подзвонити " + phone)
			dialAction.OnTriggered(func() {
				if panel.OnDialPhoneRequested != nil {
					panel.OnDialPhoneRequested(phone)
				}
			})
			copyAction := menu.AddActionWithText("Копіювати телефон")
			copyAction.OnTriggered(func() {
				clipboard := qt.QGuiApplication_Clipboard()
				if clipboard != nil {
					clipboard.SetText(phone)
				}
			})
			menu.AddSeparator()
		}
		addTableCopyActions(menu, panel.contactsTable, index)
		menu.AddSeparator()
	}
	panel.addTableColumnMenuActions(menu, "object_contacts", panel.contactsTable)
	menu.ExecWithPos(panel.contactsTable.MapToGlobalWithQPoint(pos))
}

func (panel *WorkAreaPanel) contactPhoneAtIndex(index *qt.QModelIndex) string {
	if panel == nil || panel.contactsModel == nil || index == nil || !index.IsValid() {
		return ""
	}
	phoneIndex := index.SiblingAtColumn(2)
	if phoneIndex == nil || !phoneIndex.IsValid() {
		return ""
	}
	return strings.TrimSpace(panel.contactsModel.Data(phoneIndex, int(qt.DisplayRole)).ToString())
}

func (panel *WorkAreaPanel) installTableColumnContextMenu(key string, table *qt.QTableView) {
	if panel == nil || table == nil {
		return
	}
	table.SetContextMenuPolicy(qt.CustomContextMenu)
	table.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		menu := qt.NewQMenu(table.QWidget)
		index := table.IndexAt(pos)
		if index != nil && index.IsValid() {
			addTableCopyActions(menu, table, index)
			menu.AddSeparator()
		}
		panel.addTableColumnMenuActions(menu, key, table)
		menu.ExecWithPos(table.MapToGlobalWithQPoint(pos))
	})
}

func (panel *WorkAreaPanel) installTreeColumnContextMenu(key string, tree *qt.QTreeView) {
	if panel == nil || tree == nil {
		return
	}
	tree.SetContextMenuPolicy(qt.CustomContextMenu)
	tree.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		menu := qt.NewQMenu(tree.QWidget)
		panel.addTreeColumnMenuActions(menu, key, tree)
		menu.ExecWithPos(tree.MapToGlobalWithQPoint(pos))
	})
}

func (panel *WorkAreaPanel) addTableColumnMenuActions(menu *qt.QMenu, key string, table *qt.QTableView) {
	if panel == nil || menu == nil || table == nil {
		return
	}
	autofit := menu.AddActionWithText("Підігнати колонки")
	autofit.OnTriggered(func() {
		resizeTableToContentsWithMinimums(key, table)
		panel.markColumnsSized(key)
		panel.saveTableColumnPrefs(key, table)
	})
	reset := menu.AddActionWithText("Скинути ширини колонок")
	reset.OnTriggered(func() {
		panel.clearColumnsSized(key)
		panel.clearColumnPrefs(key)
		resizeTableToContentsWithMinimums(key, table)
		panel.markColumnsAutoSized(key)
	})
}

func (panel *WorkAreaPanel) addTreeColumnMenuActions(menu *qt.QMenu, key string, tree *qt.QTreeView) {
	if panel == nil || menu == nil || tree == nil {
		return
	}
	autofit := menu.AddActionWithText("Підігнати колонки")
	autofit.OnTriggered(func() {
		resizeTreeToContentsWithMinimums(key, tree)
		panel.markColumnsSized(key)
		panel.saveTreeColumnPrefs(key, tree)
	})
	reset := menu.AddActionWithText("Скинути ширини колонок")
	reset.OnTriggered(func() {
		panel.clearColumnsSized(key)
		panel.clearColumnPrefs(key)
		resizeTreeToContentsWithMinimums(key, tree)
		panel.markColumnsAutoSized(key)
	})
}

func (panel *WorkAreaPanel) saveTableColumnPrefs(key string, table *qt.QTableView) {
	prefs := panel.uiPreferences()
	if prefs == nil || table == nil {
		return
	}
	prefs.SetString(prefQtTablePrefix+key+".widths", encodeSizes(normalizedColumnWidths(key, captureTableColumnWidths(table))))
}

func (panel *WorkAreaPanel) saveTreeColumnPrefs(key string, tree *qt.QTreeView) {
	prefs := panel.uiPreferences()
	if prefs == nil || tree == nil {
		return
	}
	prefs.SetString(prefQtTablePrefix+key+".widths", encodeSizes(normalizedColumnWidths(key, captureTreeColumnWidths(tree))))
}

func (panel *WorkAreaPanel) clearColumnPrefs(key string) {
	prefs := panel.uiPreferences()
	if prefs == nil {
		return
	}
	prefs.SetString(prefQtTablePrefix+key+".widths", "")
}

func (panel *WorkAreaPanel) SetObject(object models.Object, zones []models.Zone, contacts []models.Contact, events []models.Event) {
	if panel == nil {
		return
	}
	previousObjectID := 0
	if panel.currentObject != nil {
		previousObjectID = panel.currentObject.ID
	}
	keepLoadedEvents := previousObjectID == object.ID && panel.eventsLoadedObjectID == object.ID && len(events) == 0
	previousEvents := panel.events
	previousEventsSignature := panel.eventsRowsSignature
	previousEventsRowsReady := panel.eventsRowsReady

	panel.currentObject = &object
	panel.zones = zones
	panel.contacts = contacts
	if keepLoadedEvents {
		panel.events = previousEvents
	} else {
		panel.events = events
		panel.eventsLoadedObjectID = 0
		panel.eventsRowsSignature = ""
		panel.eventsRowsReady = false
	}
	panel.eventsLoadingObjectID = 0
	if keepLoadedEvents {
		panel.eventsRowsSignature = previousEventsSignature
		panel.eventsRowsReady = previousEventsRowsReady
	}

	panel.headerName.SetText(strings.TrimSpace(object.Name) + " (№" + viewmodels.ObjectDisplayNumber(object) + ")")
	panel.headerAddress.SetText(workAreaHeaderAddress(object))

	presentation := panel.deviceVM.BuildObjectPresentation(object)
	panel.setObjectCard(object, presentation)

	if panel.dataProvider != nil {
		go func(id int) {
			externalData := panel.viewModel.LoadExternalData(panel.dataProvider, id)
			updateUI := func() {
				if panel.currentObject == nil || panel.currentObject.ID != id {
					return
				}
				panel.setCardValue("Тест-сигнал", externalData.TestMessage)
				panel.setCardValue("Рівень сигналу", externalData.Signal)

				lastTestStr := "—"
				if !externalData.LastTest.IsZero() {
					lastTestStr = externalData.LastTest.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Останній тест", lastTestStr)

				lastMsgStr := "—"
				if !externalData.LastMessage.IsZero() {
					lastMsgStr = externalData.LastMessage.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Ост. повідомлення", lastMsgStr)
			}

			if panel.OnRunOnMainThread != nil {
				panel.OnRunOnMainThread(updateUI)
			} else {
				updateUI()
			}
		}(object.ID)
	}

	zonesTreeWidths := panel.captureTreeWidthsIfSized("object_zones", panel.zonesTree)
	zonesTableWidths := panel.captureTableWidthsIfSized("object_zones_flat", panel.zonesTable)
	contactsWidths := panel.captureTableWidthsIfSized("object_contacts", panel.contactsTable)
	setZoneRows(panel.zonesModel, zones)
	panel.restoreTreeWidthsIfCaptured("object_zones", panel.zonesTree, zonesTreeWidths)
	setZoneTableRows(panel.zonesFlatModel, zones)
	panel.restoreTableWidthsIfCaptured("object_zones_flat", panel.zonesTable, zonesTableWidths)
	panel.updateZonesView(zones)
	setContactRows(panel.contactsModel, contacts)
	panel.restoreTableWidthsIfCaptured("object_contacts", panel.contactsTable, contactsWidths)
	if keepLoadedEvents {
		panel.eventsLoadedObjectID = object.ID
	} else if len(events) > 0 {
		panel.events = events
		panel.eventsLoadedObjectID = object.ID
		panel.setEventRowsIfChanged(events)
	} else {
		panel.events = nil
		if panel.tabs.TabText(panel.tabs.CurrentIndex()) == "Журнал" {
			panel.loadEventsForCurrentObject()
		} else {
			eventsWidths := panel.captureTableWidthsIfSized("object_events", panel.eventsTable)
			panel.eventsModel.Clear()
			panel.eventsModel.SetHorizontalHeaderLabels([]string{"Час", "Подія", "Опис"})
			addReadOnlyRow(panel.eventsModel, []string{"", "Оберіть вкладку для завантаження подій", ""})
			panel.restoreTableWidthsIfCaptured("object_events", panel.eventsTable, eventsWidths)
			panel.eventsRowsReady = false
		}
	}

	if panel.exportPDFBtn != nil {
		panel.exportPDFBtn.SetEnabled(true)
	}
	if panel.exportXLSXBtn != nil {
		panel.exportXLSXBtn.SetEnabled(true)
	}
	if panel.copyExcelBtn != nil {
		panel.copyExcelBtn.SetEnabled(true)
	}
	if panel.addToDeletedBtn != nil {
		panel.addToDeletedBtn.SetEnabled(true)
	}

	panel.resizeZonesViewIfNeeded(len(zones) > 0)
	panel.resizeTableToContentsOnce("object_contacts", panel.contactsTable, len(contacts) > 0)
	panel.resizeTableToContentsOnce("object_events", panel.eventsTable, len(events) > 0)
}

func (panel *WorkAreaPanel) updateZonesView(zones []models.Zone) {
	if panel == nil || panel.zonesStack == nil {
		return
	}
	if len(groupZones(zones)) <= 1 {
		panel.zonesStack.SetCurrentWidget(panel.zonesTable.QWidget)
	} else {
		panel.zonesStack.SetCurrentWidget(panel.zonesTree.QWidget)
	}
	panel.resizeZonesViewIfNeeded(len(zones) > 0)
}

func (panel *WorkAreaPanel) resizeZonesViewIfNeeded(hasRows bool) {
	if panel == nil || panel.zonesStack == nil {
		return
	}
	if panel.zonesStack.CurrentWidget() == panel.zonesTable.QWidget {
		panel.resizeTableToContentsOnce("object_zones_flat", panel.zonesTable, hasRows)
		return
	}
	panel.resizeTreeToContentsOnce("object_zones", panel.zonesTree, hasRows)
}

func (panel *WorkAreaPanel) resizeTableToContentsOnce(key string, table *qt.QTableView, hasRows bool) {
	if panel == nil || table == nil || !hasRows || panel.columnsSized(key) {
		return
	}
	resizeTableToContentsWithMinimums(key, table)
	panel.markColumnsAutoSized(key)
}

func (panel *WorkAreaPanel) resizeTreeToContentsOnce(key string, tree *qt.QTreeView, hasRows bool) {
	if panel == nil || tree == nil || !hasRows || panel.columnsSized(key) {
		return
	}
	resizeTreeToContentsWithMinimums(key, tree)
	panel.markColumnsAutoSized(key)
}

func (panel *WorkAreaPanel) captureTableWidthsIfSized(key string, table *qt.QTableView) []int {
	if panel == nil || !panel.columnsSized(key) {
		return nil
	}
	return captureTableColumnWidths(table)
}

func (panel *WorkAreaPanel) restoreTableWidthsIfCaptured(key string, table *qt.QTableView, widths []int) bool {
	if restoreTableColumnWidthsSnapshot(key, table, widths) {
		return true
	}
	return false
}

func (panel *WorkAreaPanel) captureTreeWidthsIfSized(key string, tree *qt.QTreeView) []int {
	if panel == nil || !panel.columnsSized(key) {
		return nil
	}
	return captureTreeColumnWidths(tree)
}

func (panel *WorkAreaPanel) restoreTreeWidthsIfCaptured(key string, tree *qt.QTreeView, widths []int) bool {
	if restoreTreeColumnWidthsSnapshot(key, tree, widths) {
		return true
	}
	return false
}

func (panel *WorkAreaPanel) columnsSized(key string) bool {
	if panel == nil || key == "" {
		return true
	}
	return panel.columnManualSized[key]
}

func (panel *WorkAreaPanel) markColumnsAutoSized(key string) {
	if panel == nil || key == "" {
		return
	}
	if panel.columnAutoSized == nil {
		panel.columnAutoSized = map[string]bool{}
	}
	panel.columnAutoSized[key] = true
}

func (panel *WorkAreaPanel) markColumnsSized(key string) {
	if panel == nil || key == "" {
		return
	}
	if panel.columnManualSized == nil {
		panel.columnManualSized = map[string]bool{}
	}
	panel.columnManualSized[key] = true
}

func (panel *WorkAreaPanel) clearColumnsSized(key string) {
	if panel == nil || key == "" {
		return
	}
	if panel.columnManualSized != nil {
		delete(panel.columnManualSized, key)
	}
	if panel.columnAutoSized != nil {
		delete(panel.columnAutoSized, key)
	}
}

func (panel *WorkAreaPanel) SetLoading(object models.Object) {
	if panel == nil {
		return
	}
	panel.currentObject = &object
	panel.zones = nil
	panel.contacts = nil
	panel.events = nil
	panel.eventsLoadedObjectID = 0
	panel.eventsLoadingObjectID = 0
	panel.eventsRowsSignature = ""
	panel.eventsRowsReady = false

	panel.clearObjectDetails()

	if panel.exportPDFBtn != nil {
		panel.exportPDFBtn.SetEnabled(false)
	}
	if panel.exportXLSXBtn != nil {
		panel.exportXLSXBtn.SetEnabled(false)
	}
	if panel.copyExcelBtn != nil {
		panel.copyExcelBtn.SetEnabled(false)
	}
	if panel.addToDeletedBtn != nil {
		panel.addToDeletedBtn.SetEnabled(false)
	}

	panel.headerName.SetText(strings.TrimSpace(object.Name) + " (№" + viewmodels.ObjectDisplayNumber(object) + ")")
	panel.headerAddress.SetText(workAreaHeaderAddress(object))
	panel.setObjectCard(object, panel.deviceVM.BuildObjectPresentation(object))
}

func (panel *WorkAreaPanel) buildObjectCardTab() *qt.QWidget {
	panel.cardFields = make(map[string]*qt.QLineEdit)
	panel.cardNotes = qt.NewQTextEdit2()
	panel.cardNotes.SetReadOnly(true)
	panel.cardNotes.SetMinimumHeight(72)
	panel.cardNotes.SetMaximumHeight(120)

	content := qt.NewQWidget2()
	grid := qt.NewQGridLayout(content)
	grid.SetHorizontalSpacing(12)
	grid.SetVerticalSpacing(6)
	grid.SetColumnStretch(1, 2)
	grid.SetColumnStretch(3, 2)

	row := 0
	row = panel.addCardSection(grid, row, "Основне")
	row = panel.addCardFields(grid, row, []string{"Номер", "Договір", "Телефон"})
	row = panel.addCardWideField(grid, row, "Назва")
	row = panel.addCardWideField(grid, row, "Адреса")
	row = panel.addCardWideField(grid, row, "Координати")
	row = panel.addCardFields(grid, row, []string{"Район", "Геокодування", "Словник об'єкта"})

	row = panel.addCardSection(grid, row, "Обладнання і зв'язок")
	row = panel.addCardFields(grid, row, []string{"Прилад", "Шифр приладу", "Контроль тестів"})
	row = panel.addCardWideField(grid, row, "Групи")
	row = panel.addCardFields(grid, row, []string{"Взяття/Зняття", "SIM 1", "SIM 2"})
	row = panel.addCardWideField(grid, row, "SIM-карта")
	row = panel.addCardWideField(grid, row, "Живлення")

	row = panel.addCardSection(grid, row, "Поточний оперативний стан")
	row = panel.addCardFields(grid, row, []string{"Охорона", "Зв'язок", "Ост. повідомлення"})
	row = panel.addCardFields(grid, row, []string{"АКБ", "Канал", "Тест-сигнал"})
	row = panel.addCardFields(grid, row, []string{"Рівень сигналу", "Останній тест", "Напрямок"})

	row = panel.addCardSection(grid, row, "Додатково")
	grid.AddWidget3(qt.NewQLabel3("Примітки").QWidget, row, 0, 1, 1)
	grid.AddWidget3(panel.cardNotes.QWidget, row, 1, 1, 5)
	row++

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func (panel *WorkAreaPanel) addCardSection(grid *qt.QGridLayout, row int, title string) int {
	label := qt.NewQLabel3(title)
	label.SetStyleSheet("font-weight: bold; color: #1a73e8; padding-top: 10px;")
	grid.AddWidget3(label.QWidget, row, 0, 1, 6)
	return row + 1
}

func (panel *WorkAreaPanel) addCardWideField(grid *qt.QGridLayout, row int, labelText string) int {
	grid.AddWidget3(qt.NewQLabel3(labelText).QWidget, row, 0, 1, 1)
	field := qt.NewQLineEdit2()
	field.SetReadOnly(true)
	field.SetMinimumWidth(420)
	grid.AddWidget3(field.QWidget, row, 1, 1, 5)
	panel.cardFields[labelText] = field
	return row + 1
}

func (panel *WorkAreaPanel) addCardFields(grid *qt.QGridLayout, row int, labels []string) int {
	col := 0
	for _, label := range labels {
		grid.AddWidget3(qt.NewQLabel3(label).QWidget, row, col, 1, 1)
		field := qt.NewQLineEdit2()
		field.SetReadOnly(true)
		field.SetMinimumWidth(150)
		grid.AddWidget3(field.QWidget, row, col+1, 1, 1)
		panel.cardFields[label] = field
		col += 2
	}
	return row + 1
}

func (panel *WorkAreaPanel) setObjectCard(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) {
	panel.setCardValue("Номер", viewmodels.ObjectDisplayNumber(object))
	panel.setCardValue("Договір", object.ContractNum)
	panel.setCardValue("Телефон", presentation.PhoneCopyText)

	panel.setCardValue("Назва", strings.TrimSpace(object.Name))
	panel.setCardValue("Район", "")
	panel.setCardValue("Адреса", strings.TrimSpace(object.Address))

	panel.setCardValue("Координати", object.Location1)
	panel.setCardValue("Геокодування", "")
	panel.setCardValue("Словник об'єкта", "")

	panel.setCardValue("Прилад", trimPresentationPrefix(presentation.DeviceTypeText))
	panel.setCardValue("Шифр приладу", trimPresentationPrefix(presentation.PanelMarkText))
	panel.setCardValue("Контроль тестів", trimPresentationPrefix(presentation.TestControlText))

	panel.setCardValue("Групи", trimPresentationPrefix(presentation.GroupsText))
	panel.setCardValue("Взяття/Зняття", trimPresentationPrefix(presentation.GuardText))
	panel.setCardValue("SIM-карта", trimPresentationPrefix(presentation.SIMText))

	panel.setCardValue("SIM 1", trimPresentationPrefix(presentation.SIM1Text))
	panel.setCardValue("SIM 2", trimPresentationPrefix(presentation.SIM2Text))
	panel.setCardValue("Живлення", trimPresentationPrefix(presentation.PowerText))

	panel.setCardValue("Охорона", objectCardGuardText(object, presentation))
	panel.setCardValue("Зв'язок", objectCardConnectionText(object, presentation))

	lastMsgStr := "—"
	if !object.LastMessageTime.IsZero() {
		lastMsgStr = object.LastMessageTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Ост. повідомлення", lastMsgStr)

	panel.setCardValue("АКБ", trimPresentationPrefix(presentation.AkbText))
	panel.setCardValue("Канал", trimPresentationPrefix(presentation.ChannelText))
	panel.setCardValue("Тест-сигнал", "Завантаження...")
	panel.setCardValue("Рівень сигналу", object.SignalStrength)

	lastTestStr := "—"
	if !object.LastTestTime.IsZero() {
		lastTestStr = object.LastTestTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Останній тест", lastTestStr)
	panel.setCardValue("Напрямок", "")

	panel.cardNotes.SetPlainText(emptyDash(object.Notes1))
}

func (panel *WorkAreaPanel) clearObjectDetails() {
	if panel == nil {
		return
	}
	for _, field := range panel.cardFields {
		if field == nil {
			continue
		}
		field.SetText("-")
		field.SetToolTip("-")
	}
	if panel.cardNotes != nil {
		panel.cardNotes.SetPlainText("-")
	}
	if panel.zonesModel != nil {
		widths := panel.captureTreeWidthsIfSized("object_zones", panel.zonesTree)
		setZoneRows(panel.zonesModel, nil)
		panel.restoreTreeWidthsIfCaptured("object_zones", panel.zonesTree, widths)
	}
	if panel.zonesFlatModel != nil {
		widths := panel.captureTableWidthsIfSized("object_zones_flat", panel.zonesTable)
		setZoneTableRows(panel.zonesFlatModel, nil)
		panel.restoreTableWidthsIfCaptured("object_zones_flat", panel.zonesTable, widths)
	}
	if panel.contactsModel != nil {
		widths := panel.captureTableWidthsIfSized("object_contacts", panel.contactsTable)
		setContactRows(panel.contactsModel, nil)
		panel.restoreTableWidthsIfCaptured("object_contacts", panel.contactsTable, widths)
	}
	if panel.eventsModel != nil {
		widths := panel.captureTableWidthsIfSized("object_events", panel.eventsTable)
		panel.eventsModel.Clear()
		panel.eventsModel.SetHorizontalHeaderLabels([]string{"Час", "Подія", "Опис"})
		addReadOnlyRow(panel.eventsModel, []string{"", "Оберіть вкладку для завантаження подій", ""})
		panel.restoreTableWidthsIfCaptured("object_events", panel.eventsTable, widths)
	}
}

func (panel *WorkAreaPanel) setCardValue(label string, value string) {
	if field, ok := panel.cardFields[label]; ok {
		text := emptyDash(value)
		field.SetText(text)
		field.SetToolTip(text)
	}
}

func trimPresentationPrefix(value string) string {
	if idx := strings.Index(value, ": "); idx >= 0 {
		return strings.TrimSpace(value[idx+2:])
	}
	return value
}

func objectCardConnectionText(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) string {
	val := trimPresentationPrefix(presentation.ConnectionText)
	if val == "невідомо" && object.Status == models.StatusNormal {
		return "в нормі"
	}
	return val
}

func objectCardGuardText(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) string {
	val := trimPresentationPrefix(presentation.GuardText)
	if val == "невідомо" && object.Status == models.StatusNormal {
		return "в нормі"
	}
	return val
}

func (panel *WorkAreaPanel) SelectTab(index int) {
	if panel == nil || panel.tabs == nil {
		return
	}
	if index >= 0 && index < panel.tabs.Count() {
		panel.tabs.SetCurrentIndex(index)
	}
}

func (panel *WorkAreaPanel) buildExportTab() *qt.QWidget {
	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)

	title := qt.NewQLabel3("Експорт даних об'єкта")
	title.SetStyleSheet("font-weight: bold; font-size: 11pt; margin-bottom: 8px;")
	layout.AddWidget(title.QWidget)

	info := qt.NewQLabel3("Виберіть потрібний формат для експорту поточного стану об'єкта, його зон, контактів та журналу подій.")
	info.SetWordWrap(true)
	layout.AddWidget(info.QWidget)

	layout.AddSpacing(12)

	panel.exportPDFBtn = qt.NewQPushButton3("Експортувати в PDF")
	panel.exportPDFBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.exportPDFBtn.OnClicked(func() {
		panel.exportObject("pdf")
	})

	panel.exportXLSXBtn = qt.NewQPushButton3("Експортувати в XLSX (Excel)")
	panel.exportXLSXBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.exportXLSXBtn.OnClicked(func() {
		panel.exportObject("xlsx")
	})

	panel.copyExcelBtn = qt.NewQPushButton3("Копіювати рядок для Excel (TSV)")
	panel.copyExcelBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.copyExcelBtn.OnClicked(func() {
		panel.copyRowForExcel()
	})

	panel.addToDeletedBtn = qt.NewQPushButton3("Додати в звіт видалених об'єктів")
	panel.addToDeletedBtn.SetStyleSheet("padding: 6px; font-weight: 500;")
	panel.addToDeletedBtn.OnClicked(func() {
		panel.addObjectToDeleted()
	})

	panel.exportPDFBtn.SetEnabled(false)
	panel.exportXLSXBtn.SetEnabled(false)
	panel.copyExcelBtn.SetEnabled(false)
	panel.addToDeletedBtn.SetEnabled(false)

	layout.AddWidget(panel.exportPDFBtn.QWidget)
	layout.AddWidget(panel.exportXLSXBtn.QWidget)
	layout.AddWidget(panel.copyExcelBtn.QWidget)
	layout.AddWidget(panel.addToDeletedBtn.QWidget)

	layout.AddStretch()
	return widget
}

func (panel *WorkAreaPanel) uiPreferences() config.Preferences {
	if panel != nil && panel.prefs != nil {
		return panel.prefs
	}
	return config.NewQtPreferences("MOST", "ObjCatalogQt")
}

func (panel *WorkAreaPanel) exportObject(format string) {
	if panel == nil || panel.currentObject == nil {
		return
	}

	obj := *panel.currentObject
	zones := slices.Clone(panel.zones)
	contacts := slices.Clone(panel.contacts)
	events := slices.Clone(panel.events)
	eventsLoaded := panel.eventsLoadedObjectID == obj.ID

	panel.exportPDFBtn.SetEnabled(false)
	panel.exportXLSXBtn.SetEnabled(false)

	go func() {
		if !eventsLoaded {
			events = panel.loadObjectEventsForExport(obj.ID)
		}
		externalData := panel.viewModel.LoadExternalData(panel.dataProvider, obj.ID)
		exportData := panel.exportVM.BuildObjectExportData(obj, zones, contacts, events, externalData)

		uiCfg := config.LoadUIConfig(panel.uiPreferences())
		exportDir := uiCfg.ExportDir

		var (
			filePath string
			err      error
		)

		switch strings.ToLower(format) {
		case "pdf":
			filePath, err = objexport.ExportObjectToPDF(exportData, exportDir)
		case "xlsx":
			filePath, err = objexport.ExportObjectToXLSX(exportData, exportDir)
		default:
			err = fmt.Errorf("unsupported export format: %s", format)
		}

		runCallback := func() {
			panel.exportPDFBtn.SetEnabled(true)
			panel.exportXLSXBtn.SetEnabled(true)

			if err != nil {
				qt.QMessageBox_Critical(panel.QWidget, "Помилка експорту", err.Error())
				return
			}
			qt.QMessageBox_Information(panel.QWidget, "Експорт виконано", "Файл створено:\n"+filePath)
		}

		if panel.OnRunOnMainThread != nil {
			panel.OnRunOnMainThread(runCallback)
		} else {
			runCallback()
		}
	}()
}

func (panel *WorkAreaPanel) copyRowForExcel() {
	if panel == nil || panel.currentObject == nil {
		return
	}

	row := panel.exportVM.BuildExcelRowTSV(*panel.currentObject, panel.contacts)

	clipboard := qt.QGuiApplication_Clipboard()
	if clipboard != nil {
		clipboard.SetText(row)
		qt.QMessageBox_Information(panel.QWidget, "Буфер обміну", "Рядок для Excel скопійовано в буфер обміну")
	} else {
		qt.QMessageBox_Critical(panel.QWidget, "Помилка", "Не вдалося отримати доступ до буфера обміну")
	}
}

func (panel *WorkAreaPanel) addObjectToDeleted() {
	if panel == nil || panel.currentObject == nil {
		return
	}

	excelProvider, ok := panel.dataProvider.(contracts.ExcelReportingProvider)
	if !ok {
		qt.QMessageBox_Information(panel.QWidget, "Помилка", "Поточний провайдер не підтримує експорт в Excel.")
		return
	}

	displayName := viewmodels.ObjectDisplayNumber(*panel.currentObject)
	filePath := `D:\goproject\obj_catalog_fyne_v3\Звіт прийнятих-знятих об’єктів (1).xlsx`

	reply := qt.QMessageBox_Question(panel.QWidget, "Підтвердження", fmt.Sprintf("Додати об'єкт №%s в звіт видалених?", displayName))
	if reply != qt.QMessageBox__Yes {
		return
	}

	panel.addToDeletedBtn.SetEnabled(false)

	go func() {
		obj := *panel.currentObject
		zones := slices.Clone(panel.zones)
		contacts := slices.Clone(panel.contacts)
		events := slices.Clone(panel.events)
		eventsLoaded := panel.eventsLoadedObjectID == obj.ID

		if !eventsLoaded {
			events = panel.loadObjectEventsForExport(obj.ID)
		}

		externalData := panel.viewModel.LoadExternalData(panel.dataProvider, obj.ID)
		exportData := panel.exportVM.BuildObjectExportData(obj, zones, contacts, events, externalData)

		uiCfg := config.LoadUIConfig(panel.uiPreferences())
		exportDir := uiCfg.ExportDir

		tempPDFPath, pdfErr := objexport.ExportObjectToPDF(exportData, exportDir)
		if pdfErr != nil {
			runCallback := func() {
				panel.addToDeletedBtn.SetEnabled(true)
				qt.QMessageBox_Critical(panel.QWidget, "Помилка генерації PDF", pdfErr.Error())
			}
			if panel.OnRunOnMainThread != nil {
				panel.OnRunOnMainThread(runCallback)
			} else {
				runCallback()
			}
			return
		}

		err := excelProvider.AppendObjectToDeletedReport(&obj, contacts, tempPDFPath, filePath)

		runCallback := func() {
			panel.addToDeletedBtn.SetEnabled(true)
			if err != nil {
				if gdriveErr, ok := err.(*objexport.GoogleDriveUploadError); ok {
					qt.QMessageBox_Warning(panel.QWidget, "Увага", fmt.Sprintf("Об'єкт додано в Excel, але не завантажено на Google Drive: %v", gdriveErr.Err))
				} else {
					qt.QMessageBox_Critical(panel.QWidget, "Помилка", err.Error())
				}
				return
			}
			qt.QMessageBox_Information(panel.QWidget, "Готово", "Об'єкт додано до знятих/видалених Excel та Google Drive")
		}

		if panel.OnRunOnMainThread != nil {
			panel.OnRunOnMainThread(runCallback)
		} else {
			runCallback()
		}
	}()
}

func (panel *WorkAreaPanel) SetDataProvider(provider contracts.DataProvider) {
	if panel == nil {
		return
	}
	panel.dataProvider = provider
}

func (panel *WorkAreaPanel) loadEventsForCurrentObject() {
	panel.loadEventsForCurrentObjectWithMode(false)
}

func (panel *WorkAreaPanel) RefreshEventsIfVisible() {
	if panel == nil || panel.tabs == nil || panel.tabs.TabText(panel.tabs.CurrentIndex()) != "Журнал" {
		return
	}
	panel.loadEventsForCurrentObjectWithMode(true)
}

func (panel *WorkAreaPanel) loadEventsForCurrentObjectWithMode(force bool) {
	if panel == nil || panel.currentObject == nil || panel.dataProvider == nil {
		return
	}

	objectID := panel.currentObject.ID
	if !force && (panel.eventsLoadedObjectID == objectID || panel.eventsLoadingObjectID == objectID) {
		return
	}
	if panel.eventsLoadingObjectID == objectID {
		return
	}
	if !force {
		if cached, ok := panel.cachedObjectEvents(objectID); ok {
			panel.events = cached
			panel.eventsLoadedObjectID = objectID
			panel.setEventRowsIfChanged(cached)
			return
		}
	}

	if !force || panel.eventsLoadedObjectID != objectID {
		eventsWidths := panel.captureTableWidthsIfSized("object_events", panel.eventsTable)
		panel.eventsModel.Clear()
		panel.eventsModel.SetHorizontalHeaderLabels([]string{"Час", "Подія", "Опис"})
		addReadOnlyRow(panel.eventsModel, []string{"--:--", "Завантаження подій...", ""})
		panel.restoreTableWidthsIfCaptured("object_events", panel.eventsTable, eventsWidths)
		panel.eventsRowsReady = false
	}

	eventLimit := config.LoadUIConfig(panel.uiPreferences()).ObjectLogLimit
	panel.eventsLoadingObjectID = objectID

	go func(id int) {
		events := panel.viewModel.LoadObjectEvents(panel.dataProvider, id, eventLimit)
		updateUI := func() {
			if panel.eventsLoadingObjectID == id {
				panel.eventsLoadingObjectID = 0
			}
			if panel.currentObject == nil || panel.currentObject.ID != id {
				return
			}
			panel.events = events
			panel.eventsLoadedObjectID = id
			panel.storeObjectEvents(id, events)
			panel.setEventRowsIfChanged(events)
		}
		if panel.OnRunOnMainThread != nil {
			panel.OnRunOnMainThread(updateUI)
		} else {
			updateUI()
		}
	}(objectID)
}

func (panel *WorkAreaPanel) loadObjectEventsForExport(objectID int) []models.Event {
	if panel == nil || panel.dataProvider == nil || panel.viewModel == nil {
		return nil
	}
	if events, ok := panel.cachedObjectEvents(objectID); ok {
		return events
	}
	eventLimit := config.LoadUIConfig(panel.uiPreferences()).ObjectLogLimit
	events := panel.viewModel.LoadObjectEvents(panel.dataProvider, objectID, eventLimit)
	panel.storeObjectEvents(objectID, events)
	return events
}

func (panel *WorkAreaPanel) cachedObjectEvents(objectID int) ([]models.Event, bool) {
	if panel == nil || panel.eventsCache == nil || objectID == 0 {
		return nil, false
	}
	panel.eventsCacheMu.Lock()
	defer panel.eventsCacheMu.Unlock()
	entry, ok := panel.eventsCache[objectID]
	if !ok || time.Since(entry.loadedAt) > objectEventsCacheTTL {
		delete(panel.eventsCache, objectID)
		return nil, false
	}
	return slices.Clone(entry.events), true
}

func (panel *WorkAreaPanel) storeObjectEvents(objectID int, events []models.Event) {
	if panel == nil || objectID == 0 {
		return
	}
	panel.eventsCacheMu.Lock()
	defer panel.eventsCacheMu.Unlock()
	if panel.eventsCache == nil {
		panel.eventsCache = map[int]objectEventsCacheEntry{}
	}
	if _, ok := panel.eventsCache[objectID]; !ok {
		panel.eventsCacheOrder = append(panel.eventsCacheOrder, objectID)
	}
	panel.eventsCache[objectID] = objectEventsCacheEntry{
		events:   slices.Clone(events),
		loadedAt: time.Now(),
	}
	for len(panel.eventsCacheOrder) > objectEventsCacheLimit {
		oldest := panel.eventsCacheOrder[0]
		panel.eventsCacheOrder = panel.eventsCacheOrder[1:]
		delete(panel.eventsCache, oldest)
	}
}

func (panel *WorkAreaPanel) setEventRowsIfChanged(events []models.Event) {
	if panel == nil || panel.eventsModel == nil {
		return
	}
	signature := objectEventRowsSignature(events)
	if panel.eventsRowsReady && panel.eventsRowsSignature == signature {
		return
	}
	panel.eventsRowsSignature = signature
	panel.eventsRowsReady = true
	columnWidths := panel.captureTableWidthsIfSized("object_events", panel.eventsTable)
	setEventRows(panel.eventsModel, events)
	if panel.restoreTableWidthsIfCaptured("object_events", panel.eventsTable, columnWidths) {
		return
	}
	panel.resizeTableToContentsOnce("object_events", panel.eventsTable, len(events) > 0)
}

func objectEventRowsSignature(events []models.Event) string {
	var b strings.Builder
	for _, event := range events {
		b.WriteString(eventRowSignature(event))
		b.WriteByte('|')
	}
	return b.String()
}
