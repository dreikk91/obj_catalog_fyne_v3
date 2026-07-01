//go:build qt

package qtui

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

// statusCard represents a single colored indicator card in the object card header.
type statusCard struct {
	frame *qt.QFrame
	title *qt.QLabel
	value *qt.QLabel
}

type WorkAreaPanel struct {
	*qt.QWidget
	tabs                     *qt.QTabWidget
	headerName               *qt.QLabel
	headerAddress            *qt.QLabel
	cardFields               map[string]*qt.QLineEdit
	cardNotes                *qt.QTextEdit
	statusCards              map[string]*statusCard
	deviceVM                 *viewmodels.WorkAreaDeviceViewModel
	overviewVM               *viewmodels.WorkAreaOverviewViewModel
	overviewFacts            map[string]*qt.QLabel
	overviewMetrics          map[string]*qt.QLabel
	overviewContactsModel    *qt.QStandardItemModel
	overviewContactsTable    *qt.QTableView
	overviewZonesLayout      *qt.QGridLayout
	mapCoordinates           *qt.QLabel
	mapButton                *qt.QPushButton
	testMessagesButton       *qt.QPushButton
	zonesModel               *qt.QStandardItemModel
	zonesFlatModel           *qt.QStandardItemModel
	contactsModel            *qt.QStandardItemModel
	eventsModel              *qt.QStandardItemModel
	eventsRange              *qt.QComboBox
	zonesStack               *qt.QStackedWidget
	zonesTree                *qt.QTreeView
	zonesTable               *qt.QTableView
	contactsTable            *qt.QTableView
	eventsTable              *qt.QTableView
	mediaList                *qt.QListWidget
	mediaPreview             *qt.QLabel
	mediaStatus              *qt.QLabel
	mediaOpenButton          *qt.QPushButton
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
	eventsLoadSeq         int
	eventsRowsSignature   string
	eventsRowsReady       bool
	eventsCacheMu         sync.Mutex
	eventsCache           map[int]objectEventsCacheEntry
	eventsCacheOrder      []int
	media                 []contracts.ObjectMedia
	mediaImageCache       map[string][]byte
	mediaLoadedObjectID   int
	mediaLoadingObjectID  int
	mediaLoadSeq          int
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
	prefQtObjectEventsDays = "qt.objectEvents.rangeDays"
)

type objectEventsCacheEntry struct {
	events   []models.Event
	loadedAt time.Time
}

func NewWorkAreaPanel(prefs config.Preferences) *WorkAreaPanel {
	panel := &WorkAreaPanel{
		QWidget:         qt.NewQWidget2(),
		deviceVM:        viewmodels.NewWorkAreaDeviceViewModel(),
		overviewVM:      viewmodels.NewWorkAreaOverviewViewModel(),
		viewModel:       viewmodels.NewWorkAreaViewModel(),
		exportVM:        viewmodels.NewWorkAreaExportViewModel(),
		prefs:           prefs,
		eventsCache:     map[int]objectEventsCacheEntry{},
		mediaImageCache: map[string][]byte{},
	}
	layout := qt.NewQVBoxLayout(panel.QWidget)
	panel.headerName = qt.NewQLabel3("Оберіть об'єкт зі списку")
	panel.headerName.SetStyleSheet("font-weight: 700; font-size: 15pt; color: #172B3A; padding: 2px 0;")
	panel.headerAddress = qt.NewQLabel3("")
	panel.headerAddress.SetWordWrap(true)
	panel.headerAddress.SetStyleSheet("color: " + qtMutedTextColor + "; font-size: 10pt; padding-bottom: 4px;")
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
	panel.tabs.AddTab(panel.buildOverviewTab(), "Огляд")
	panel.zonesModel = qt.NewQStandardItemModel2(0, 6)
	panel.zonesTree = newTree(panel.zonesModel, zoneTreeHeaders())
	panel.installTreeColumnContextMenu("object_zones", panel.zonesTree)
	panel.zonesFlatModel = qt.NewQStandardItemModel2(0, 4)
	panel.zonesTable = newTable(panel.zonesFlatModel, zoneTableHeaders())
	panel.installTableColumnContextMenu("object_zones_flat", panel.zonesTable)
	panel.zonesStack = qt.NewQStackedWidget2()
	panel.zonesStack.AddWidget(panel.zonesTable.QWidget)
	panel.zonesStack.AddWidget(panel.zonesTree.QWidget)
	panel.tabs.AddTab(panel.buildObjectCardTab(), "Деталі")
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
	panel.tabs.AddTab(panel.buildEventsTab(), "Журнал")
	panel.tabs.AddTab(panel.buildMediaTab(), "Медіа")

	panel.tabs.AddTab(panel.buildExportTab(), "Експорт")
	panel.tabs.OnCurrentChanged(func(index int) {
		switch panel.tabs.TabText(index) {
		case "Журнал":
			panel.loadEventsForCurrentObject()
		case "Медіа":
			panel.loadMediaForCurrentObject(false)
		}
	})
	layout.AddWidget(panel.headerName.QWidget)
	layout.AddWidget(panel.headerAddress.QWidget)
	layout.AddLayout(actionsLayout.QLayout)
	layout.AddWidget(panel.tabs.QWidget)
	panel.SetLayout(layout.QLayout)
	return panel
}

func (panel *WorkAreaPanel) buildOverviewTab() *qt.QWidget {
	panel.statusCards = make(map[string]*statusCard)
	panel.overviewFacts = make(map[string]*qt.QLabel)
	panel.overviewMetrics = make(map[string]*qt.QLabel)

	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	layout.SetContentsMargins(10, 10, 10, 10)
	layout.SetSpacing(9)

	statusRow := qt.NewQHBoxLayout2()
	statusRow.SetSpacing(8)
	panel.addStatusCard(statusRow, "guard", "ОХОРОНА", "—")
	panel.addStatusCard(statusRow, "connection", "ЗВ'ЯЗОК", "—")
	panel.addStatusCard(statusRow, "power", "ЖИВЛЕННЯ", "—")
	panel.addStatusCard(statusRow, "monitoring", "МОНІТОРИНГ", "—")
	layout.AddLayout(statusRow.QLayout)

	metricsRow := qt.NewQHBoxLayout2()
	metricsRow.SetSpacing(8)
	panel.addOverviewMetric(metricsRow, "groups", "ГРУПИ")
	panel.addOverviewMetric(metricsRow, "zones", "ЗОНИ")
	panel.addOverviewMetric(metricsRow, "contacts", "ВІДПОВІДАЛЬНІ")
	metricsRow.AddStretch()
	layout.AddLayout(metricsRow.QLayout)

	operationalGroup := qt.NewQGroupBox3("Оперативні дані")
	operationalGrid := qt.NewQGridLayout(operationalGroup.QWidget)
	operationalGrid.SetHorizontalSpacing(14)
	operationalGrid.SetVerticalSpacing(7)
	operationalGrid.SetColumnStretch(1, 1)
	operationalGrid.SetColumnStretch(3, 1)
	panel.addOverviewFact(operationalGrid, 0, 0, "Прилад", "device")
	panel.addOverviewFact(operationalGrid, 0, 2, "Канал", "channel")
	panel.addOverviewFact(operationalGrid, 1, 0, "Остання подія", "lastEvent")
	panel.addOverviewFact(operationalGrid, 1, 2, "Останній тест", "lastTest")
	panel.addOverviewFact(operationalGrid, 2, 0, "Рівень сигналу", "signal")
	panel.addOverviewFact(operationalGrid, 2, 2, "Контроль тесту", "testControl")
	panel.addOverviewWideFact(operationalGrid, 3, "Контакт / ГМР", "phone")
	panel.addOverviewWideFact(operationalGrid, 4, "Група реагування", "responseGroup")
	panel.addOverviewWideFact(operationalGrid, 5, "Розташування", "location")
	panel.addOverviewWideFact(operationalGrid, 6, "Додаткова інформація", "additionalInfo")
	layout.AddWidget(operationalGroup.QWidget)

	tables := qt.NewQSplitter3(qt.Horizontal)

	contactsGroup := qt.NewQGroupBox3("Відповідальні особи")
	contactsLayout := qt.NewQVBoxLayout(contactsGroup.QWidget)
	panel.overviewContactsModel = qt.NewQStandardItemModel2(0, 3)
	panel.overviewContactsTable = newTable(panel.overviewContactsModel, []string{"Особа", "Телефон", "Роль / група"})
	panel.overviewContactsTable.VerticalHeader().SetVisible(false)
	panel.overviewContactsTable.SetMaximumHeight(180)
	panel.overviewContactsTable.OnDoubleClicked(func(index *qt.QModelIndex) {
		if index == nil || !index.IsValid() || panel.OnDialPhoneRequested == nil {
			return
		}
		phone := strings.TrimSpace(panel.overviewContactsModel.Data(index.SiblingAtColumn(1), int(qt.DisplayRole)).ToString())
		if phone != "" && phone != "—" {
			panel.OnDialPhoneRequested(phone)
		}
	})
	contactsLayout.AddWidget(panel.overviewContactsTable.QWidget)
	tables.AddWidget(contactsGroup.QWidget)

	zonesGroup := qt.NewQGroupBox3("Стан зон")
	zonesLayout := qt.NewQVBoxLayout(zonesGroup.QWidget)
	zonesContent := qt.NewQWidget2()
	panel.overviewZonesLayout = qt.NewQGridLayout(zonesContent)
	panel.overviewZonesLayout.SetContentsMargins(4, 4, 4, 4)
	panel.overviewZonesLayout.SetHorizontalSpacing(4)
	panel.overviewZonesLayout.SetVerticalSpacing(4)
	zonesScroll := qt.NewQScrollArea2()
	zonesScroll.SetWidgetResizable(true)
	zonesScroll.SetMaximumHeight(180)
	zonesScroll.SetWidget(zonesContent)
	zonesLayout.AddWidget(zonesScroll.QWidget)
	tables.AddWidget(zonesGroup.QWidget)
	tables.SetSizes([]int{560, 420})

	layout.AddWidget(tables.QWidget)
	layout.AddStretch()

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func (panel *WorkAreaPanel) addOverviewMetric(layout *qt.QHBoxLayout, key string, title string) {
	frame := qt.NewQFrame2()
	frame.SetStyleSheet("QFrame { background: #F1F5F9; border: 1px solid #CBD5E1; border-radius: 4px; }")
	frameLayout := qt.NewQHBoxLayout(frame.QWidget)
	frameLayout.SetContentsMargins(10, 5, 10, 5)
	value := qt.NewQLabel3("0")
	value.SetStyleSheet("font-weight: 800; font-size: 15pt; color: #172B3A; border: 0; background: transparent;")
	label := qt.NewQLabel3(title)
	label.SetStyleSheet("font-weight: 700; font-size: 8pt; color: " + qtMutedTextColor + "; border: 0; background: transparent;")
	frameLayout.AddWidget(value.QWidget)
	frameLayout.AddWidget(label.QWidget)
	layout.AddWidget(frame.QWidget)
	panel.overviewMetrics[key] = value
}

func (panel *WorkAreaPanel) addOverviewFact(grid *qt.QGridLayout, row int, col int, title string, key string) {
	titleLabel := qt.NewQLabel3(title)
	titleLabel.SetStyleSheet("color: " + qtMutedTextColor + "; font-size: 9pt;")
	value := qt.NewQLabel3("—")
	value.SetWordWrap(true)
	value.SetTextInteractionFlags(qt.TextSelectableByMouse)
	value.SetStyleSheet("font-weight: 600; color: #172B3A;")
	grid.AddWidget3(titleLabel.QWidget, row, col, 1, 1)
	grid.AddWidget3(value.QWidget, row, col+1, 1, 1)
	panel.overviewFacts[key] = value
}

func (panel *WorkAreaPanel) addOverviewWideFact(grid *qt.QGridLayout, row int, title string, key string) {
	titleLabel := qt.NewQLabel3(title)
	titleLabel.SetStyleSheet("color: " + qtMutedTextColor + "; font-size: 9pt;")
	value := qt.NewQLabel3("—")
	value.SetWordWrap(true)
	value.SetTextInteractionFlags(qt.TextSelectableByMouse)
	value.SetStyleSheet("font-weight: 600; color: #172B3A;")
	grid.AddWidget3(titleLabel.QWidget, row, 0, 1, 1)
	grid.AddWidget3(value.QWidget, row, 1, 1, 3)
	panel.overviewFacts[key] = value
}

func (panel *WorkAreaPanel) buildMediaTab() *qt.QWidget {
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)

	panel.mediaStatus = qt.NewQLabel3("Оберіть об'єкт")
	panel.mediaStatus.SetStyleSheet("color: #555;")
	layout.AddWidget(panel.mediaStatus.QWidget)

	panel.mediaList = qt.NewQListWidget2()
	panel.mediaList.SetMinimumWidth(260)
	panel.mediaPreview = qt.NewQLabel3("Оберіть фото, схему або камеру")
	panel.mediaPreview.SetAlignment(qt.AlignCenter)
	panel.mediaPreview.SetWordWrap(true)
	panel.mediaPreview.SetMinimumSize2(420, 260)
	panel.mediaPreview.SetStyleSheet("background: #f6f6f6; border: 1px solid #d8d8d8;")

	splitter := qt.NewQSplitter3(qt.Horizontal)
	splitter.AddWidget(panel.mediaList.QWidget)
	splitter.AddWidget(panel.mediaPreview.QWidget)
	splitter.SetSizes([]int{280, 650})
	layout.AddWidget(splitter.QWidget)

	actions := qt.NewQHBoxLayout2()
	refreshButton := qt.NewQPushButton3("Оновити")
	refreshButton.OnClicked(func() { panel.loadMediaForCurrentObject(true) })
	panel.mediaOpenButton = qt.NewQPushButton3("Відкрити")
	panel.mediaOpenButton.SetEnabled(false)
	panel.mediaOpenButton.OnClicked(panel.openSelectedMedia)
	actions.AddWidget(refreshButton.QWidget)
	actions.AddWidget(panel.mediaOpenButton.QWidget)
	actions.AddStretch()
	layout.AddLayout(actions.QLayout)

	panel.mediaList.OnCurrentRowChanged(func(int) {
		panel.showSelectedMedia()
	})
	content.SetLayout(layout.QLayout)
	return content
}

func (panel *WorkAreaPanel) buildEventsTab() *qt.QWidget {
	content := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(content)
	toolbar := qt.NewQHBoxLayout2()
	toolbar.AddWidget(qt.NewQLabel3("Період").QWidget)
	panel.eventsRange = qt.NewQComboBox2()
	for _, option := range []struct {
		label string
		days  int
	}{
		{label: "Останні 24 години", days: 1},
		{label: "Останні 3 дні", days: 3},
		{label: "Останні 7 днів", days: 7},
		{label: "Останні 30 днів", days: 30},
		{label: "Останні 90 днів", days: 90},
	} {
		panel.eventsRange.AddItem3(option.label, qt.NewQVariant4(option.days))
	}
	selectedDays := 3
	if panel.prefs != nil {
		selectedDays = panel.prefs.IntWithFallback(prefQtObjectEventsDays, 3)
	}
	panel.setObjectEventRangeDays(selectedDays)
	panel.eventsRange.OnCurrentIndexChanged(func(_ int) {
		panel.objectEventRangeChanged()
	})
	toolbar.AddWidget(panel.eventsRange.QWidget)
	toolbar.AddStretch()
	layout.AddLayout(toolbar.QLayout)
	layout.AddWidget(panel.eventsTable.QWidget)
	content.SetLayout(layout.QLayout)
	return content
}

func (panel *WorkAreaPanel) setObjectEventRangeDays(days int) {
	if panel == nil || panel.eventsRange == nil {
		return
	}
	for index := 0; index < panel.eventsRange.Count(); index++ {
		if panel.eventsRange.ItemData(index).ToInt() == days {
			panel.eventsRange.SetCurrentIndex(index)
			return
		}
	}
	panel.eventsRange.SetCurrentIndex(1)
}

func (panel *WorkAreaPanel) objectEventRangeDays() int {
	if panel == nil || panel.eventsRange == nil || panel.eventsRange.CurrentIndex() < 0 {
		return 3
	}
	days := panel.eventsRange.ItemData(panel.eventsRange.CurrentIndex()).ToInt()
	if days <= 0 {
		return 3
	}
	return days
}

func (panel *WorkAreaPanel) objectEventRange(now time.Time) (time.Time, time.Time) {
	return now.Add(-time.Duration(panel.objectEventRangeDays()) * 24 * time.Hour), now
}

func (panel *WorkAreaPanel) objectEventRangeChanged() {
	if panel == nil {
		return
	}
	if panel.prefs != nil {
		panel.prefs.SetInt(prefQtObjectEventsDays, panel.objectEventRangeDays())
	}
	panel.eventsLoadSeq++
	panel.eventsLoadedObjectID = 0
	panel.eventsLoadingObjectID = 0
	panel.eventsRowsReady = false
	panel.eventsCacheMu.Lock()
	panel.eventsCache = map[int]objectEventsCacheEntry{}
	panel.eventsCacheOrder = nil
	panel.eventsCacheMu.Unlock()
	if panel.tabs != nil && panel.tabs.TabText(panel.tabs.CurrentIndex()) == "Журнал" {
		panel.loadEventsForCurrentObjectWithMode(true)
	}
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
	if panel.testMessagesButton != nil {
		isBridge := !ids.IsCASLObjectID(object.ID) && !ids.IsPhoenixObjectID(object.ID)
		panel.testMessagesButton.SetVisible(isBridge)
		panel.testMessagesButton.SetEnabled(isBridge && panel.dataProvider != nil)
	}
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
	if previousObjectID != object.ID {
		panel.clearMediaState()
	}
	if keepLoadedEvents {
		panel.eventsRowsSignature = previousEventsSignature
		panel.eventsRowsReady = previousEventsRowsReady
	}

	panel.headerName.SetText(strings.TrimSpace(object.Name) + " (№" + viewmodels.ObjectDisplayNumber(object) + ")")
	panel.headerAddress.SetText(workAreaHeaderAddress(object))

	presentation := panel.deviceVM.BuildObjectPresentation(object)
	panel.setObjectCard(object, presentation)
	panel.setOperationalOverview(object, zones, contacts, presentation)

	if panel.dataProvider != nil {
		go func(id int) {
			externalData := panel.viewModel.LoadExternalData(panel.dataProvider, id)
			updateUI := func() {
				if panel.currentObject == nil || panel.currentObject.ID != id {
					return
				}
				panel.setCardValue("Тест-повідомлення", externalData.TestMessage)
				panel.setCardValue("Якість зв'язку", externalData.Signal)
				panel.setOverviewFact("signal", externalData.Signal)

				lastTestStr := "—"
				if !externalData.LastTest.IsZero() {
					lastTestStr = externalData.LastTest.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Останній тест", lastTestStr)
				panel.setOverviewFact("lastTest", lastTestStr)

				lastMsgStr := "—"
				if !externalData.LastMessage.IsZero() {
					lastMsgStr = externalData.LastMessage.Format("02.01.2006 15:04:05")
				}
				panel.setCardValue("Остання подія", lastMsgStr)
				panel.setOverviewFact("lastEvent", lastMsgStr)
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
	if panel.tabs.TabText(panel.tabs.CurrentIndex()) == "Медіа" {
		panel.loadMediaForCurrentObject(false)
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
		// Replacing the model rows collapses the tree. Expand it for every
		// multi-group object, even when column widths were sized earlier.
		panel.zonesTree.ExpandAll()
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
	panel.clearMediaState()

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
	presentation := panel.deviceVM.BuildObjectPresentation(object)
	panel.setObjectCard(object, presentation)
	panel.setOperationalOverview(object, nil, nil, presentation)
}

func (panel *WorkAreaPanel) clearMediaState() {
	if panel == nil {
		return
	}
	panel.mediaLoadSeq++
	panel.media = nil
	panel.mediaImageCache = map[string][]byte{}
	panel.mediaLoadedObjectID = 0
	panel.mediaLoadingObjectID = 0
	if panel.mediaList != nil {
		panel.mediaList.Clear()
	}
	if panel.mediaPreview != nil {
		panel.mediaPreview.SetPixmap(qt.NewQPixmap())
		panel.mediaPreview.SetText("Оберіть фото, схему або камеру")
	}
	if panel.mediaStatus != nil {
		panel.mediaStatus.SetText("Медіа ще не завантажено")
	}
	if panel.mediaOpenButton != nil {
		panel.mediaOpenButton.SetEnabled(false)
	}
}

func (panel *WorkAreaPanel) loadMediaForCurrentObject(force bool) {
	if panel == nil || panel.currentObject == nil || panel.dataProvider == nil {
		return
	}
	provider, ok := panel.dataProvider.(contracts.ObjectMediaProvider)
	if !ok {
		panel.mediaStatus.SetText("Поточне джерело не підтримує медіа")
		return
	}
	objectID := panel.currentObject.ID
	if !force && (panel.mediaLoadedObjectID == objectID || panel.mediaLoadingObjectID == objectID) {
		return
	}
	panel.mediaLoadingObjectID = objectID
	panel.mediaLoadSeq++
	seq := panel.mediaLoadSeq
	panel.mediaStatus.SetText("Завантаження медіа...")
	panel.mediaList.Clear()
	panel.mediaPreview.SetPixmap(qt.NewQPixmap())
	panel.mediaPreview.SetText("Завантаження...")
	panel.mediaOpenButton.SetEnabled(false)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		media, err := provider.GetObjectMedia(ctx, objectID)
		panel.runOnMainThread(func() {
			if seq != panel.mediaLoadSeq || panel.currentObject == nil || panel.currentObject.ID != objectID {
				return
			}
			panel.mediaLoadingObjectID = 0
			if err != nil {
				panel.mediaStatus.SetText("Помилка завантаження: " + strings.TrimSpace(err.Error()))
				panel.mediaPreview.SetText("Не вдалося завантажити медіа")
				return
			}
			panel.media = append(panel.media[:0], media...)
			panel.mediaLoadedObjectID = objectID
			panel.fillMediaList()
		})
	}()
}

func (panel *WorkAreaPanel) fillMediaList() {
	panel.mediaList.Clear()
	for _, media := range panel.media {
		prefix := "Фото / схема"
		if media.Kind == contracts.ObjectMediaCamera {
			prefix = "Камера"
		}
		text := prefix + ": " + strings.TrimSpace(media.Title)
		if room := strings.TrimSpace(media.RoomName); room != "" {
			text += " | " + room
		}
		panel.mediaList.AddItem(text)
	}
	panel.mediaStatus.SetText(fmt.Sprintf("Медіа: %d", len(panel.media)))
	if len(panel.media) == 0 {
		panel.mediaPreview.SetText("Для цього об'єкта медіа не знайдено")
		return
	}
	panel.mediaList.SetCurrentRow(0)
}

func (panel *WorkAreaPanel) selectedMedia() (contracts.ObjectMedia, bool) {
	if panel == nil || panel.mediaList == nil {
		return contracts.ObjectMedia{}, false
	}
	row := panel.mediaList.CurrentRow()
	if row < 0 || row >= len(panel.media) {
		return contracts.ObjectMedia{}, false
	}
	return panel.media[row], true
}

func (panel *WorkAreaPanel) showSelectedMedia() {
	media, ok := panel.selectedMedia()
	if !ok {
		panel.mediaOpenButton.SetEnabled(false)
		return
	}
	panel.mediaOpenButton.SetEnabled(true)
	if media.Kind == contracts.ObjectMediaCamera {
		panel.mediaPreview.SetPixmap(qt.NewQPixmap())
		panel.mediaPreview.SetText(strings.TrimSpace(media.URL))
		return
	}
	provider, ok := panel.dataProvider.(contracts.ObjectMediaProvider)
	if !ok {
		return
	}
	panel.mediaLoadSeq++
	seq := panel.mediaLoadSeq
	objectID := panel.currentObject.ID
	panel.mediaPreview.SetPixmap(qt.NewQPixmap())
	panel.mediaPreview.SetText("Завантаження зображення...")
	if body := panel.mediaImageCache[media.ID]; len(body) > 0 {
		panel.showMediaPreviewBody(body)
		return
	}
	go func(selected contracts.ObjectMedia) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		body, err := provider.FetchObjectMedia(ctx, selected)
		panel.runOnMainThread(func() {
			if seq != panel.mediaLoadSeq || panel.currentObject == nil || panel.currentObject.ID != objectID {
				return
			}
			if err != nil {
				panel.mediaPreview.SetText("Не вдалося завантажити: " + strings.TrimSpace(err.Error()))
				return
			}
			if len(body) == 0 {
				panel.mediaPreview.SetText("Порожнє зображення")
				return
			}
			panel.mediaImageCache[selected.ID] = append([]byte(nil), body...)
			panel.showMediaPreviewBody(body)
		})
	}(media)
}

func (panel *WorkAreaPanel) showMediaPreviewBody(body []byte) {
	if len(body) == 0 {
		panel.mediaPreview.SetText("Порожнє зображення")
		return
	}
	pixmap := qt.NewQPixmap()
	if !pixmap.LoadFromData(&body[0], uint(len(body))) {
		panel.mediaPreview.SetText("Невідомий формат зображення")
		return
	}
	panel.mediaPreview.SetText("")
	panel.mediaPreview.SetPixmap(pixmap.Scaled(760, 500))
}

func (panel *WorkAreaPanel) openSelectedMedia() {
	media, ok := panel.selectedMedia()
	if !ok {
		return
	}
	if media.Kind == contracts.ObjectMediaCamera {
		if target := strings.TrimSpace(media.URL); target != "" {
			qt.QDesktopServices_OpenUrl(qt.NewQUrl3(target))
		}
		return
	}
	provider, ok := panel.dataProvider.(contracts.ObjectMediaProvider)
	if !ok {
		return
	}
	if body := panel.mediaImageCache[media.ID]; len(body) > 0 {
		showCASLImagePreview(panel.QWidget, body)
		return
	}
	objectID := panel.currentObject.ID
	go func(selected contracts.ObjectMedia) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		body, err := provider.FetchObjectMedia(ctx, selected)
		panel.runOnMainThread(func() {
			if panel.currentObject == nil || panel.currentObject.ID != objectID {
				return
			}
			if err != nil {
				qt.QMessageBox_Critical(panel.QWidget, "Медіа", err.Error())
				return
			}
			panel.mediaImageCache[selected.ID] = append([]byte(nil), body...)
			showCASLImagePreview(panel.QWidget, body)
		})
	}(media)
}

func (panel *WorkAreaPanel) runOnMainThread(f func()) {
	if panel.OnRunOnMainThread != nil {
		panel.OnRunOnMainThread(f)
		return
	}
	f()
}

func (panel *WorkAreaPanel) buildObjectCardTab() *qt.QWidget {
	panel.cardFields = make(map[string]*qt.QLineEdit)
	panel.cardNotes = qt.NewQTextEdit2()
	panel.cardNotes.SetReadOnly(true)
	panel.cardNotes.SetMinimumHeight(72)
	panel.cardNotes.SetMaximumHeight(120)

	content := qt.NewQWidget2()
	mainLayout := qt.NewQVBoxLayout(content)
	mainLayout.SetSpacing(8)

	// --- Section: Основна інформація ---
	basicGroup := qt.NewQGroupBox3("Основна інформація")
	basicGrid := qt.NewQGridLayout(basicGroup.QWidget)
	basicGrid.SetHorizontalSpacing(12)
	basicGrid.SetVerticalSpacing(6)
	basicGrid.SetColumnStretch(1, 2)
	basicGrid.SetColumnStretch(3, 2)
	row := 0
	row = panel.addCardFields(basicGrid, row, []string{"Номер", "Договір"})
	row = panel.addCardWideField(basicGrid, row, "Назва")
	row = panel.addCardWideField(basicGrid, row, "Адреса")
	row = panel.addCardWideField(basicGrid, row, "Контакт / ГМР")
	row = panel.addCardWideField(basicGrid, row, "Опис об'єкта")
	row = panel.addCardWideField(basicGrid, row, "Розташування")
	_ = row
	mainLayout.AddWidget(basicGroup.QWidget)

	// --- Section: Обладнання та зв'язок ---
	deviceGroup := qt.NewQGroupBox3("Обладнання та зв'язок")
	deviceGrid := qt.NewQGridLayout(deviceGroup.QWidget)
	deviceGrid.SetHorizontalSpacing(12)
	deviceGrid.SetVerticalSpacing(6)
	deviceGrid.SetColumnStretch(1, 2)
	deviceGrid.SetColumnStretch(3, 2)
	row = 0
	row = panel.addCardFieldWithTooltip(deviceGrid, row, 0, "Прилад", "Тип приймально-контрольного приладу на об'єкті")
	row = panel.addCardFieldWithTooltip(deviceGrid, row-1, 2, "Модель ППК", "Марка та модель приймально-контрольного приладу")
	row = panel.addCardFieldWithTooltip(deviceGrid, row-1, 4, "Період тесту", "Як часто прилад відправляє тестовий сигнал на пульт")
	row = panel.addCardWideField(deviceGrid, row, "Групи")
	row = panel.addCardFieldWithTooltip(deviceGrid, row, 0, "Стан охорони", "Поточний стан постановки/зняття з охорони")
	row = panel.addCardFieldWithTooltip(deviceGrid, row-1, 2, "SIM 1", "Основний номер SIM-карти приладу")
	row = panel.addCardFieldWithTooltip(deviceGrid, row-1, 4, "SIM 2", "Резервний номер SIM-карти приладу")
	row = panel.addCardWideField(deviceGrid, row, "SIM-карта")
	row = panel.addCardWideField(deviceGrid, row, "Живлення")
	_ = row
	mainLayout.AddWidget(deviceGroup.QWidget)

	// --- Section: Оперативний стан ---
	stateGroup := qt.NewQGroupBox3("Технічний стан")
	stateGrid := qt.NewQGridLayout(stateGroup.QWidget)
	stateGrid.SetHorizontalSpacing(12)
	stateGrid.SetVerticalSpacing(6)
	stateGrid.SetColumnStretch(1, 2)
	stateGrid.SetColumnStretch(3, 2)
	row = 0
	row = panel.addCardFieldWithTooltip(stateGrid, row, 0, "Охорона (стан)", "Поточний стан охорони об'єкта")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 2, "Зв'язок (стан)", "Стан зв'язку приладу з пультом")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 4, "Остання подія", "Час отримання останнього повідомлення від приладу")
	row = panel.addCardFieldWithTooltip(stateGrid, row, 0, "Батарея (АКБ)", "Стан акумуляторної батареї резервного живлення")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 2, "Канал зв'язку", "Тип каналу зв'язку приладу з пультом (GPRS, автододзвін тощо)")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 4, "Тест-повідомлення", "Останнє тестове повідомлення від приладу")
	row = panel.addCardFieldWithTooltip(stateGrid, row, 0, "Якість зв'язку", "Рівень GSM-сигналу приладу")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 2, "Останній тест", "Дата та час останнього тестового сигналу")
	row = panel.addCardFieldWithTooltip(stateGrid, row-1, 4, "Напрямок", "Напрямок підключення на пульті")
	panel.testMessagesButton = qt.NewQPushButton3("Переглянути тестові повідомлення")
	panel.testMessagesButton.SetVisible(false)
	panel.testMessagesButton.OnClicked(panel.showCurrentObjectTestMessages)
	stateGrid.AddWidget3(panel.testMessagesButton.QWidget, row, 0, 1, 6)
	row++
	_ = row
	mainLayout.AddWidget(stateGroup.QWidget)

	// --- Section: Додаткова інформація ---
	notesGroup := qt.NewQGroupBox3("Додаткова інформація")
	notesLayout := qt.NewQVBoxLayout(notesGroup.QWidget)
	notesLayout.AddWidget(panel.cardNotes.QWidget)
	mainLayout.AddWidget(notesGroup.QWidget)

	locationGroup := qt.NewQGroupBox3("Координати і карта")
	locationLayout := qt.NewQHBoxLayout(locationGroup.QWidget)
	panel.mapCoordinates = qt.NewQLabel3("Координати не вказані")
	panel.mapCoordinates.SetTextInteractionFlags(qt.TextSelectableByMouse)
	panel.mapCoordinates.SetStyleSheet("color: " + qtMutedTextColor + ";")
	panel.mapButton = qt.NewQPushButton3("Відкрити карту")
	panel.mapButton.SetEnabled(false)
	panel.mapButton.OnClicked(panel.openCurrentObjectMap)
	locationLayout.AddWidget(panel.mapCoordinates.QWidget)
	locationLayout.AddStretch()
	locationLayout.AddWidget(panel.mapButton.QWidget)
	mainLayout.AddWidget(locationGroup.QWidget)

	mainLayout.AddStretch()

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

// addStatusCard creates a colored status indicator card.
func (panel *WorkAreaPanel) addStatusCard(layout *qt.QHBoxLayout, key string, title string, initialValue string) {
	frame := qt.NewQFrame2()
	frame.SetStyleSheet(`
		QFrame {
			border: 1px solid ` + qtBorderColor + `;
			border-radius: 6px;
			padding: 6px 12px;
			min-width: 130px;
			background: ` + qtAltSurfaceColor + `;
		}
	`)
	cardLayout := qt.NewQVBoxLayout(frame.QWidget)
	cardLayout.SetSpacing(2)
	cardLayout.SetContentsMargins(6, 4, 6, 4)

	titleLabel := qt.NewQLabel3(title)
	titleLabel.SetStyleSheet("font-size: 9pt; color: " + qtMutedTextColor + "; border: 0; background: transparent; padding: 0;")

	valueLabel := qt.NewQLabel3(initialValue)
	valueLabel.SetStyleSheet("font-weight: 700; font-size: 10pt; color: #333; border: 0; background: transparent; padding: 0;")

	cardLayout.AddWidget(titleLabel.QWidget)
	cardLayout.AddWidget(valueLabel.QWidget)
	layout.AddWidget(frame.QWidget)

	panel.statusCards[key] = &statusCard{
		frame: frame,
		title: titleLabel,
		value: valueLabel,
	}
}

// updateStatusIndicators updates the status cards with the current object state.
func (panel *WorkAreaPanel) updateStatusIndicators(object models.Object) {
	if panel == nil || panel.statusCards == nil {
		return
	}

	// Guard status
	if card, ok := panel.statusCards["guard"]; ok {
		switch object.GuardStatusValue() {
		case models.GuardStatusGuarded:
			panel.setStatusCardState(card, "Під охороною", "#2E7D32", "#E8F5E9")
		case models.GuardStatusDisarmed:
			panel.setStatusCardState(card, "Знято", "#F57F17", "#FFFDE7")
		default:
			panel.setStatusCardState(card, "Невідомо", qtMutedTextColor, qtAltSurfaceColor)
		}
	}

	// Connection status
	if card, ok := panel.statusCards["connection"]; ok {
		switch object.ConnectionStatusValue() {
		case models.ConnectionStatusOnline:
			panel.setStatusCardState(card, "Онлайн", "#2E7D32", "#E8F5E9")
		case models.ConnectionStatusOffline:
			panel.setStatusCardState(card, "Втрата зв'язку", "#C62828", "#FFEBEE")
		default:
			panel.setStatusCardState(card, "Невідомо", qtMutedTextColor, qtAltSurfaceColor)
		}
	}

	// Power status
	if card, ok := panel.statusCards["power"]; ok {
		value, textColor, backgroundColor := objectPowerStatusCardState(object)
		panel.setStatusCardState(card, value, textColor, backgroundColor)
	}

	// Monitoring status
	if card, ok := panel.statusCards["monitoring"]; ok {
		switch object.MonitoringStatusValue() {
		case models.MonitoringStatusActive:
			panel.setStatusCardState(card, "Активний", "#2E7D32", "#E8F5E9")
		case models.MonitoringStatusBlocked:
			panel.setStatusCardState(card, "Заблоковано", "#C62828", "#FFEBEE")
		case models.MonitoringStatusDebug:
			panel.setStatusCardState(card, "Тест/Налагодження", "#F57F17", "#FFFDE7")
		default:
			panel.setStatusCardState(card, "Невідомо", qtMutedTextColor, qtAltSurfaceColor)
		}
	}
}

func objectPowerStatusCardState(object models.Object) (value string, textColor string, backgroundColor string) {
	switch {
	case object.PowerFault > 0:
		return "Аварія 220В", "#C62828", "#FFEBEE"
	case object.AkbState > 0:
		return "Несправність АКБ", "#F57F17", "#FFFDE7"
	case object.PowerFault < 0 && object.AkbState < 0:
		return "Невідомо", qtMutedTextColor, qtAltSurfaceColor
	default:
		return "220В та АКБ в нормі", "#2E7D32", "#E8F5E9"
	}
}

func (panel *WorkAreaPanel) setStatusCardState(card *statusCard, value string, textColor string, bgColor string) {
	if card == nil {
		return
	}
	card.value.SetText(value)
	card.value.SetStyleSheet(fmt.Sprintf(
		"font-weight: 700; font-size: 10pt; color: %s; border: 0; background: transparent; padding: 0;",
		textColor,
	))
	card.frame.SetStyleSheet(fmt.Sprintf(`
		QFrame {
			border: 1px solid %s;
			border-radius: 6px;
			padding: 6px 12px;
			min-width: 130px;
			background: %s;
		}
	`, textColor, bgColor))
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

// addCardFieldWithTooltip adds a single labeled field at a specific grid position with a tooltip.
func (panel *WorkAreaPanel) addCardFieldWithTooltip(grid *qt.QGridLayout, row int, col int, labelText string, tooltip string) int {
	lbl := qt.NewQLabel3(labelText)
	lbl.SetToolTip(tooltip)
	grid.AddWidget3(lbl.QWidget, row, col, 1, 1)
	field := qt.NewQLineEdit2()
	field.SetReadOnly(true)
	field.SetMinimumWidth(150)
	field.SetToolTip(tooltip)
	grid.AddWidget3(field.QWidget, row, col+1, 1, 1)
	panel.cardFields[labelText] = field
	return row + 1
}

func (panel *WorkAreaPanel) setObjectCard(object models.Object, presentation viewmodels.WorkAreaDevicePresentation) {
	// --- Основна інформація ---
	panel.setCardValue("Номер", viewmodels.ObjectDisplayNumber(object))
	panel.setCardValue("Договір", object.ContractNum)
	panel.setCardValue("Контакт / ГМР", presentation.PhoneCopyText)

	panel.setCardValue("Назва", strings.TrimSpace(object.Name))
	panel.setCardValue("Адреса", strings.TrimSpace(object.Address))

	panel.setCardValue("Опис об'єкта", presentation.DescriptionText)
	panel.setCardValue("Розташування", object.Location1)

	// --- Обладнання та зв'язок ---
	panel.setCardValue("Прилад", trimPresentationPrefix(presentation.DeviceTypeText))
	panel.setCardValue("Модель ППК", trimPresentationPrefix(presentation.PanelMarkText))
	panel.setCardValue("Період тесту", trimPresentationPrefix(presentation.TestControlText))

	panel.setCardValue("Групи", trimPresentationPrefix(presentation.GroupsText))
	panel.setCardValue("Стан охорони", trimPresentationPrefix(presentation.GuardText))
	panel.setCardValue("SIM-карта", trimPresentationPrefix(presentation.SIMText))

	panel.setCardValue("SIM 1", trimPresentationPrefix(presentation.SIM1Text))
	panel.setCardValue("SIM 2", trimPresentationPrefix(presentation.SIM2Text))
	panel.setCardValue("Живлення", trimPresentationPrefix(presentation.PowerText))

	// --- Оперативний стан ---
	panel.setCardValue("Охорона (стан)", objectCardGuardText(object, presentation))
	panel.setCardValue("Зв'язок (стан)", objectCardConnectionText(object, presentation))

	lastMsgStr := "—"
	if !object.LastMessageTime.IsZero() {
		lastMsgStr = object.LastMessageTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Остання подія", lastMsgStr)

	panel.setCardValue("Батарея (АКБ)", trimPresentationPrefix(presentation.AkbText))
	panel.setCardValue("Канал зв'язку", trimPresentationPrefix(presentation.ChannelText))
	panel.setCardValue("Тест-повідомлення", "Завантаження...")
	panel.setCardValue("Якість зв'язку", object.SignalStrength)

	lastTestStr := "—"
	if !object.LastTestTime.IsZero() {
		lastTestStr = object.LastTestTime.Format("02.01.2006 15:04:05")
	}
	panel.setCardValue("Останній тест", lastTestStr)
	panel.setCardValue("Напрямок", "")

	panel.cardNotes.SetPlainText(emptyDash(object.Notes1))
	panel.setMapLocation(object)

	// Update status indicator cards
	panel.updateStatusIndicators(object)
}

func (panel *WorkAreaPanel) setMapLocation(object models.Object) {
	if panel == nil || panel.mapCoordinates == nil || panel.mapButton == nil {
		return
	}
	latitude := strings.TrimSpace(object.Latitude)
	longitude := strings.TrimSpace(object.Longitude)
	if latitude == "" || longitude == "" {
		panel.mapCoordinates.SetText("Координати не вказані")
		panel.mapButton.SetEnabled(false)
		return
	}
	panel.mapCoordinates.SetText(latitude + ", " + longitude)
	panel.mapButton.SetEnabled(true)
}

func (panel *WorkAreaPanel) openCurrentObjectMap() {
	if panel == nil || panel.currentObject == nil {
		return
	}
	latitude := strings.TrimSpace(panel.currentObject.Latitude)
	longitude := strings.TrimSpace(panel.currentObject.Longitude)
	if latitude == "" || longitude == "" {
		return
	}
	query := url.QueryEscape(latitude + "," + longitude)
	qt.QDesktopServices_OpenUrl(qt.NewQUrl3("https://www.google.com/maps/search/?api=1&query=" + query))
}

func (panel *WorkAreaPanel) setOperationalOverview(
	object models.Object,
	zones []models.Zone,
	contacts []models.Contact,
	presentation viewmodels.WorkAreaDevicePresentation,
) {
	if panel == nil || panel.overviewVM == nil {
		return
	}
	overview := panel.overviewVM.Build(object, zones, contacts, presentation)

	panel.setOverviewFact("device", overview.Device)
	panel.setOverviewFact("channel", overview.Channel)
	panel.setOverviewFact("signal", overview.Signal)
	panel.setOverviewFact("lastEvent", overview.LastEvent)
	panel.setOverviewFact("lastTest", overview.LastTest)
	panel.setOverviewFact("testControl", overview.TestControl)
	panel.setOverviewFact("phone", overview.Phone)
	panel.setOverviewFact("responseGroup", overview.ResponseGroup)
	panel.setOverviewFact("location", overview.Location)
	panel.setOverviewFact("additionalInfo", overview.AdditionalInfo)
	panel.setOverviewMetric("groups", overview.GroupCount)
	panel.setOverviewMetric("zones", overview.ZoneCount)
	panel.setOverviewMetric("contacts", overview.ContactCount)
	panel.setOverviewContacts(overview.PriorityContacts)
	panel.setOverviewZoneStates(zones)
	panel.updateStatusIndicators(object)
}

func (panel *WorkAreaPanel) setOverviewFact(key string, value string) {
	if panel == nil || panel.overviewFacts == nil {
		return
	}
	if label := panel.overviewFacts[key]; label != nil {
		label.SetText(emptyDash(value))
		label.SetToolTip(emptyDash(value))
	}
}

func (panel *WorkAreaPanel) setOverviewMetric(key string, value int) {
	if panel == nil || panel.overviewMetrics == nil {
		return
	}
	if label := panel.overviewMetrics[key]; label != nil {
		label.SetText(strconv.Itoa(value))
	}
}

func (panel *WorkAreaPanel) setOverviewContacts(contacts []models.Contact) {
	if panel == nil || panel.overviewContactsModel == nil {
		return
	}
	panel.overviewContactsModel.Clear()
	panel.overviewContactsModel.SetHorizontalHeaderLabels([]string{"Особа", "Телефон", "Роль / група"})
	if len(contacts) == 0 {
		addReadOnlyRow(panel.overviewContactsModel, []string{"Відповідальних не вказано", "", ""})
		return
	}
	for _, contact := range contacts {
		role := contactPositionText(contact)
		if group := strings.TrimSpace(contact.GroupName); group != "" {
			if role != "" && role != "—" {
				role += " / "
			}
			role += group
		}
		addReadOnlyRow(panel.overviewContactsModel, []string{
			emptyDash(contact.Name),
			emptyDash(contact.Phone),
			emptyDash(role),
		})
	}
	panel.overviewContactsTable.ResizeColumnsToContents()
	panel.overviewContactsTable.HorizontalHeader().SetStretchLastSection(true)
}

func (panel *WorkAreaPanel) setOverviewZoneStates(zones []models.Zone) {
	if panel == nil || panel.overviewZonesLayout == nil {
		return
	}
	for {
		item := panel.overviewZonesLayout.TakeAt(0)
		if item == nil {
			break
		}
		if widget := item.Widget(); widget != nil {
			widget.Hide()
			widget.Delete()
		}
	}
	if len(zones) == 0 {
		empty := qt.NewQLabel3("Дані про зони відсутні")
		empty.SetStyleSheet("color: " + qtMutedTextColor + "; padding: 8px;")
		panel.overviewZonesLayout.AddWidget3(empty.QWidget, 0, 0, 1, 1)
		return
	}

	const columns = 4
	for index := range zones {
		zone := zones[index]
		label := qt.NewQLabel3(fmt.Sprintf("%d.%s", zone.Number, overviewZoneStatusText(zone)))
		label.SetAlignment(qt.AlignCenter)
		label.SetFixedSize2(88, 34)
		label.SetToolTip(fmt.Sprintf("Зона №%d: %s\n%s", zone.Number, zone.GetStatusDisplay(), emptyDash(zone.Name)))
		label.SetStyleSheet(overviewZoneStyle(zone))
		panel.overviewZonesLayout.AddWidget3(label.QWidget, index/columns, index%columns, 1, 1)
	}
}

func overviewZoneStatusText(zone models.Zone) string {
	if zone.IsBypassed {
		return "Відкл."
	}
	switch zone.Status {
	case models.ZoneNormal:
		return "Норм."
	case models.ZoneAlarm:
		return "Трив."
	case models.ZoneFire:
		return "Пож."
	case models.ZoneBreak:
		return "Обр."
	case models.ZoneShort:
		return "КЗ"
	default:
		return "—"
	}
}

func overviewZoneStyle(zone models.Zone) string {
	background := "#718096"
	if zone.IsBypassed {
		background = "#D79218"
	} else {
		switch zone.Status {
		case models.ZoneNormal:
			background = "#3C9360"
		case models.ZoneAlarm, models.ZoneFire:
			background = "#C53B32"
		case models.ZoneBreak, models.ZoneShort:
			background = "#D79218"
		}
	}
	return "font-weight: 700; color: white; background: " + background +
		"; border: 1px solid rgba(0, 0, 0, 35); border-radius: 2px; padding: 2px;"
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
	panel.eventsLoadSeq++
	loadSeq := panel.eventsLoadSeq
	from, to := panel.objectEventRange(time.Now())

	go func(id int, seq int, rangeStart time.Time, rangeEnd time.Time) {
		events := panel.viewModel.LoadObjectEventsRange(panel.dataProvider, id, eventLimit, rangeStart, rangeEnd)
		updateUI := func() {
			if seq != panel.eventsLoadSeq {
				return
			}
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
	}(objectID, loadSeq, from, to)
}

func (panel *WorkAreaPanel) loadObjectEventsForExport(objectID int) []models.Event {
	if panel == nil || panel.dataProvider == nil || panel.viewModel == nil {
		return nil
	}
	if events, ok := panel.cachedObjectEvents(objectID); ok {
		return events
	}
	eventLimit := config.LoadUIConfig(panel.uiPreferences()).ObjectLogLimit
	from, to := panel.objectEventRange(time.Now())
	events := panel.viewModel.LoadObjectEventsRange(panel.dataProvider, objectID, eventLimit, from, to)
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

func (panel *WorkAreaPanel) SetTableFontSize(objectsSize float32, eventsSize float32) {
	if panel == nil {
		return
	}
	objTables := []*qt.QTableView{
		panel.overviewContactsTable,
		panel.zonesTable,
		panel.contactsTable,
	}
	for _, table := range objTables {
		if table != nil {
			font := table.Font()
			font.SetPointSizeF(float64(objectsSize))
			table.SetFont(font)
			table.VerticalHeader().SetDefaultSectionSize(int(objectsSize * 2))
		}
	}
	if panel.eventsTable != nil {
		font := panel.eventsTable.Font()
		font.SetPointSizeF(float64(eventsSize))
		panel.eventsTable.SetFont(font)
		panel.eventsTable.VerticalHeader().SetDefaultSectionSize(int(eventsSize * 2))
	}
}

