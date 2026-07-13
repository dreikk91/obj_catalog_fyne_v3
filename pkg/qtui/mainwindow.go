//go:build qt

package qtui

import (
	"encoding/base64"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
)

const (
	prefQtWindowWidth      = "qt.window.width"
	prefQtWindowHeight     = "qt.window.height"
	prefQtTopSplitterSizes = "qt.splitter.top.sizes"
	prefQtTablePrefix      = "qt.table."
	prefQtDockState        = "qt.window.dock_state_v3"
)

type MainWindow struct {
	*qt.QMainWindow

	app          *App
	topSplitter  *qt.QSplitter
	persistTimer *qt.QTimer

	objectList            *ObjectListPanel
	workArea              *WorkAreaPanel
	alarmPanel            *AlarmPanel
	eventLog              *EventLogPanel
	alarmDock             *qt.QDockWidget
	eventDock             *qt.QDockWidget
	alarmDockAction       *qt.QAction
	eventDockAction       *qt.QAction
	allowDetachedJournals bool

	statusLabel *qt.QLabel

	OnSettingsRequested       func()
	OnRefreshRequested        func()
	OnDiagnosticsRequested    func()
	OnResponseGroupsRequested func()
	OnOperationalMapRequested func()
	OnNewObjectsRequested     func()
	OnCreateObjectRequested   func()
	OnCreateCASLRequested     func()
}

func NewMainWindow(app *App) *MainWindow {
	mw := &MainWindow{
		QMainWindow: qt.NewQMainWindow2(),
		app:         app,
	}

	mw.SetWindowTitle("АРМ Пожежної Безпеки - Qt UI")
	mw.restoreWindowSize()
	mw.SetStyleSheet(NativeWindowsStyleSheet)

	mw.buildLayout()
	mw.restoreDockState()
	mw.buildMenuBar()
	mw.ApplyJournalDockPolicy(config.LoadUIConfig(mw.preferences()).AllowDetachedJournals)
	mw.buildStatusBar()
	mw.restoreTableColumnWidths()
	mw.installTableColumnPersistence()
	mw.registerShortcuts()
	mw.installClosePersistence()

	return mw
}

func (mw *MainWindow) buildMenuBar() {
	menuBar := qt.NewQMenuBar(mw.QWidget)
	fileMenu := menuBar.AddMenuWithTitle("Файл")
	createAction := fileMenu.AddActionWithText("Новий об'єкт МІСТ")
	createAction.SetShortcut(qt.NewQKeySequence2("Ctrl+N"))
	createAction.OnTriggered(func() {
		if mw.OnCreateObjectRequested != nil {
			mw.OnCreateObjectRequested()
		}
	})
	createCASLAction := fileMenu.AddActionWithText("Новий об'єкт CASL")
	createCASLAction.SetShortcut(qt.NewQKeySequence2("Ctrl+Shift+N"))
	createCASLAction.OnTriggered(func() {
		if mw.OnCreateCASLRequested != nil {
			mw.OnCreateCASLRequested()
		}
	})
	fileMenu.AddSeparator()
	settingsAction := fileMenu.AddActionWithText("Налаштування")
	settingsAction.SetShortcut(qt.NewQKeySequence2("Ctrl+,"))
	settingsAction.OnTriggered(func() {
		if mw.OnSettingsRequested != nil {
			mw.OnSettingsRequested()
		}
	})
	refreshAction := fileMenu.AddActionWithText("Оновити")
	refreshAction.SetShortcut(qt.NewQKeySequence2("Ctrl+R"))
	refreshAction.OnTriggered(func() {
		if mw.OnRefreshRequested != nil {
			mw.OnRefreshRequested()
		}
	})
	fileMenu.AddActionWithText("Експорт")
	fileMenu.AddSeparator()
	exitAction := fileMenu.AddActionWithText("Вийти")
	exitAction.SetShortcut(qt.NewQKeySequence2("Ctrl+Q"))
	exitAction.OnTriggered(func() {
		qt.QCoreApplication_Quit()
	})

	viewMenu := menuBar.AddMenuWithTitle("Вигляд")
	responseGroupsAction := viewMenu.AddActionWithText("Групи реагування")
	responseGroupsAction.SetShortcut(qt.NewQKeySequence2("Ctrl+Shift+G"))
	responseGroupsAction.OnTriggered(func() {
		if mw.OnResponseGroupsRequested != nil {
			mw.OnResponseGroupsRequested()
		}
	})
	operationalMapAction := viewMenu.AddActionWithText("Оперативна карта")
	operationalMapAction.SetShortcut(qt.NewQKeySequence2("Ctrl+Shift+M"))
	operationalMapAction.OnTriggered(func() {
		if mw.OnOperationalMapRequested != nil {
			mw.OnOperationalMapRequested()
		}
	})
	newObjectsAction := viewMenu.AddActionWithText("Нові об'єкти за період")
	newObjectsAction.SetShortcut(qt.NewQKeySequence2("Ctrl+Shift+O"))
	newObjectsAction.OnTriggered(func() {
		if mw.OnNewObjectsRequested != nil {
			mw.OnNewObjectsRequested()
		}
	})
	viewMenu.AddSeparator()
	if mw.alarmDock != nil {
		toggleAlarmsAction := mw.alarmDock.ToggleViewAction()
		toggleAlarmsAction.SetText("Показати журнал тривог")
		viewMenu.AddAction(toggleAlarmsAction)
		detachAlarmsAction := viewMenu.AddActionWithText("Прикріпити / відкріпити журнал тривог")
		detachAlarmsAction.OnTriggered(func() {
			mw.toggleDockFloating(mw.alarmDock)
		})
		mw.alarmDockAction = detachAlarmsAction
	}
	if mw.eventDock != nil {
		toggleEventsAction := mw.eventDock.ToggleViewAction()
		toggleEventsAction.SetText("Показати журнал подій")
		viewMenu.AddAction(toggleEventsAction)
		detachEventsAction := viewMenu.AddActionWithText("Прикріпити / відкріпити журнал подій")
		detachEventsAction.OnTriggered(func() {
			mw.toggleDockFloating(mw.eventDock)
		})
		mw.eventDockAction = detachEventsAction
	}
	viewMenu.AddSeparator()
	viewMenu.AddActionWithText("Світла тема")
	viewMenu.AddActionWithText("Темна тема")

	helpMenu := menuBar.AddMenuWithTitle("Допомога")
	diagnosticsAction := helpMenu.AddActionWithText("Діагностика")
	diagnosticsAction.OnTriggered(func() {
		if mw.OnDiagnosticsRequested != nil {
			mw.OnDiagnosticsRequested()
		}
	})
	helpMenu.AddActionWithText("Про програму")

	mw.SetMenuBar(menuBar)
}

func (mw *MainWindow) buildStatusBar() {
	statusBar := qt.NewQStatusBar(mw.QWidget)
	mw.statusLabel = qt.NewQLabel3("БД: не підключено | Phoenix: не підключено | Ctrl+F - пошук")
	statusBar.AddWidget(mw.statusLabel.QWidget)
	mw.SetStatusBar(statusBar)
}

func (mw *MainWindow) SetStatus(text string) {
	if mw == nil || mw.statusLabel == nil {
		return
	}
	mw.statusLabel.SetText(text)
}

func (mw *MainWindow) buildLayout() {
	mw.objectList = NewObjectListPanel(mw.app.Preferences())
	mw.workArea = NewWorkAreaPanel(mw.app.Preferences())
	mw.alarmPanel = NewAlarmPanel(mw.app.Preferences())
	mw.eventLog = NewEventLogPanel(mw.app.Preferences())
	mw.objectList.SetSizePolicy2(qt.QSizePolicy__Preferred, qt.QSizePolicy__Ignored)
	mw.workArea.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Ignored)
	mw.alarmPanel.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Ignored)
	mw.eventLog.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Ignored)
	mw.alarmPanel.OnCountChanged = func(count int) {
		mw.setDockCount(mw.alarmDock, "Тривоги", count)
	}
	mw.eventLog.OnCountChanged = func(count int) {
		mw.setDockCount(mw.eventDock, "Журнал подій", count)
	}

	mw.topSplitter = qt.NewQSplitter3(qt.Horizontal)
	mw.topSplitter.AddWidget(mw.objectList.QWidget)
	mw.topSplitter.AddWidget(mw.workArea.QWidget)
	mw.topSplitter.SetSizePolicy2(qt.QSizePolicy__Expanding, qt.QSizePolicy__Ignored)
	topSizes := mw.savedSplitterSizes(prefQtTopSplitterSizes, []int{320, 1040})
	mw.topSplitter.SetSizes(normalizeTopSplitterSizes(topSizes, mw.availableLayoutWidth()))
	mw.SetCentralWidget(mw.topSplitter.QWidget)

	dockFeatures := qt.QDockWidget__DockWidgetClosable |
		qt.QDockWidget__DockWidgetMovable |
		qt.QDockWidget__DockWidgetFloatable
	mw.alarmDock = qt.NewQDockWidget4("Тривоги", mw.QWidget)
	alarmDockName := qt.NewQAnyStringView3("alarmJournalDock")
	mw.alarmDock.SetObjectName(*alarmDockName)
	alarmDockName.Delete()
	mw.alarmDock.SetFeatures(dockFeatures)
	mw.alarmDock.SetWidget(mw.alarmPanel.QWidget)
	disableDockDoubleClick(mw.alarmDock)
	mw.activateDockAfterAttach(mw.alarmDock)
	mw.AddDockWidget(qt.BottomDockWidgetArea, mw.alarmDock)

	mw.eventDock = qt.NewQDockWidget4("Журнал подій", mw.QWidget)
	eventDockName := qt.NewQAnyStringView3("eventJournalDock")
	mw.eventDock.SetObjectName(*eventDockName)
	eventDockName.Delete()
	mw.eventDock.SetFeatures(dockFeatures)
	mw.eventDock.SetWidget(mw.eventLog.QWidget)
	disableDockDoubleClick(mw.eventDock)
	mw.activateDockAfterAttach(mw.eventDock)
	mw.AddDockWidget(qt.BottomDockWidgetArea, mw.eventDock)

	mw.SetDockOptions(qt.QMainWindow__AllowTabbedDocks)
	mw.TabifyDockWidget(mw.alarmDock, mw.eventDock)
	mw.alarmDock.Raise()
	dockHeight := journalDockHeight(mw.availableLayoutHeight())
	mw.ResizeDocks([]*qt.QDockWidget{mw.alarmDock, mw.eventDock}, []int{dockHeight, dockHeight}, qt.Vertical)
}

func disableDockDoubleClick(dock *qt.QDockWidget) {
	if dock == nil {
		return
	}
	dock.OnMouseDoubleClickEvent(func(_ func(event *qt.QMouseEvent), event *qt.QMouseEvent) {
		if event != nil {
			event.Accept()
		}
	})
}

func (mw *MainWindow) activateDockAfterAttach(dock *qt.QDockWidget) {
	if mw == nil || dock == nil {
		return
	}
	dock.OnTopLevelChanged(func(topLevel bool) {
		if topLevel {
			return
		}
		DeferOnMainThread(func() {
			if dock.IsFloating() {
				return
			}
			dock.SetVisible(true)
			dock.Raise()
			dock.Update()
			mw.Update()
		})
	})
}

func (mw *MainWindow) restoreDockState() {
	prefs := mw.preferences()
	if prefs == nil {
		return
	}
	encoded := strings.TrimSpace(prefs.StringWithFallback(prefQtDockState, ""))
	if encoded == "" {
		return
	}
	state, err := base64.StdEncoding.DecodeString(encoded)
	if err == nil {
		mw.RestoreState(state)
	}
}

func (mw *MainWindow) setDockCount(dock *qt.QDockWidget, title string, count int) {
	if mw == nil || dock == nil {
		return
	}
	if count > 0 {
		dock.SetWindowTitle(title + " (" + strconv.Itoa(count) + ")")
		return
	}
	dock.SetWindowTitle(title)
}

func (mw *MainWindow) toggleDockFloating(dock *qt.QDockWidget) {
	if mw == nil || dock == nil || !mw.allowDetachedJournals {
		return
	}
	dock.SetVisible(true)
	dock.SetFloating(!dock.IsFloating())
	dock.Raise()
}

// ApplyJournalDockPolicy enables or disables floating journal windows.
func (mw *MainWindow) ApplyJournalDockPolicy(allowDetached bool) {
	if mw == nil {
		return
	}
	mw.allowDetachedJournals = allowDetached

	features := qt.QDockWidget__DockWidgetClosable
	if allowDetached {
		features |= qt.QDockWidget__DockWidgetMovable | qt.QDockWidget__DockWidgetFloatable
	}
	for _, dock := range []*qt.QDockWidget{mw.alarmDock, mw.eventDock} {
		if dock == nil {
			continue
		}
		if !allowDetached && dock.IsFloating() {
			dock.SetFloating(false)
			dock.Raise()
		}
		dock.SetFeatures(features)
	}
	for _, action := range []*qt.QAction{mw.alarmDockAction, mw.eventDockAction} {
		if action != nil {
			action.SetVisible(allowDetached)
			action.SetEnabled(allowDetached)
		}
	}
}

func (mw *MainWindow) registerShortcuts() {
	mw.addShortcut("Ctrl+F", func() {
		if mw.objectList != nil {
			mw.objectList.FocusSearch()
			mw.SetStatus("Пошук активний")
		}
	})
	mw.addShortcut("Ctrl+1", func() { mw.selectWorkAreaTab(0) })
	mw.addShortcut("Ctrl+2", func() { mw.selectWorkAreaTab(1) })
	mw.addShortcut("Ctrl+3", func() { mw.selectWorkAreaTab(2) })
	mw.addShortcut("Ctrl+4", func() { mw.selectWorkAreaTab(3) })
	mw.addShortcut("Ctrl+5", func() { mw.selectWorkAreaTab(4) })
	mw.addShortcut("Ctrl+6", func() { mw.selectWorkAreaTab(5) })
	mw.addShortcut("Ctrl+7", func() { mw.selectWorkAreaTab(6) })
}

func (mw *MainWindow) addShortcut(sequence string, handler func()) {
	shortcut := qt.NewQShortcut2(qt.NewQKeySequence2(sequence), mw.QObject)
	shortcut.SetContext(qt.ApplicationShortcut)
	shortcut.OnActivated(handler)
}

func (mw *MainWindow) selectWorkAreaTab(index int) {
	if mw == nil || mw.workArea == nil {
		return
	}
	mw.workArea.SelectTab(index)
}

func (mw *MainWindow) restoreWindowSize() {
	prefs := mw.preferences()
	width, height := 1440, 900
	if prefs != nil {
		width = prefs.IntWithFallback(prefQtWindowWidth, width)
		height = prefs.IntWithFallback(prefQtWindowHeight, height)
	}
	availableWidth, availableHeight := mw.availableScreenDimensions()
	width, height = constrainWindowSize(width, height, availableWidth, availableHeight)
	mw.Resize(width, height)
}

func constrainWindowSize(width int, height int, availableWidth int, availableHeight int) (int, int) {
	width = max(width, 800)
	height = max(height, 600)
	if availableWidth > 0 {
		width = min(width, availableWidth)
	}
	if availableHeight > 0 {
		height = min(height, availableHeight)
	}
	return width, height
}

func journalDockHeight(availableHeight int) int {
	if availableHeight <= 0 {
		return 240
	}
	return max(170, min(260, availableHeight*24/100))
}

func (mw *MainWindow) installClosePersistence() {
	mw.OnCloseEvent(func(super func(event *qt.QCloseEvent), event *qt.QCloseEvent) {
		mw.persistWindowState()
		super(event)
	})
}

func (mw *MainWindow) persistWindowState() {
	prefs := mw.preferences()
	if prefs == nil {
		return
	}
	size := mw.Size()
	if size != nil {
		prefs.SetInt(prefQtWindowWidth, size.Width())
		prefs.SetInt(prefQtWindowHeight, size.Height())
	}
	if mw.topSplitter != nil {
		prefs.SetString(prefQtTopSplitterSizes, encodeSizes(mw.topSplitter.Sizes()))
	}
	if state := mw.SaveState(); len(state) > 0 {
		prefs.SetString(prefQtDockState, base64.StdEncoding.EncodeToString(state))
	}
	if mw.alarmPanel != nil {
		mw.alarmPanel.saveSplitterSizes()
	}
	mw.persistTableColumnWidths()
}

func (mw *MainWindow) tableRegistry() map[string]*qt.QTableView {
	tables := map[string]*qt.QTableView{}
	if mw.objectList != nil {
		tables["objects"] = mw.objectList.table
	}
	if mw.eventLog != nil {
		tables["events"] = mw.eventLog.table
	}
	if mw.workArea != nil {
		tables["object_zones_flat"] = mw.workArea.zonesTable
		tables["object_contacts"] = mw.workArea.contactsTable
		tables["object_events"] = mw.workArea.eventsTable
	}
	return tables
}

func (mw *MainWindow) treeRegistry() map[string]*qt.QTreeView {
	trees := map[string]*qt.QTreeView{}
	if mw.alarmPanel != nil {
		trees["alarms"] = mw.alarmPanel.table
	}
	if mw.workArea != nil {
		trees["object_zones"] = mw.workArea.zonesTree
	}
	return trees
}

func (mw *MainWindow) restoreTableColumnWidths() {
	prefs := mw.preferences()
	if prefs == nil {
		return
	}
	for key, table := range mw.tableRegistry() {
		if table == nil || table.Model() == nil {
			continue
		}
		widths := decodeSizes(prefs.StringWithFallback(prefQtTablePrefix+key+".widths", ""))
		if len(widths) == 0 {
			continue
		}
		if len(widths) != table.Model().ColumnCount(qt.NewQModelIndex()) {
			continue
		}
		widths = normalizedColumnWidths(key, widths)
		applyTableColumnMinimums(key, table)
		for column, width := range widths {
			if width > 0 {
				table.SetColumnWidth(column, width)
			}
		}
		table.HorizontalHeader().SetSectionResizeMode(qt.QHeaderView__Interactive)
		table.HorizontalHeader().SetStretchLastSection(false)
		mw.markTableManuallySized(key)
	}
	for key, tree := range mw.treeRegistry() {
		if tree == nil || tree.Model() == nil {
			continue
		}
		widths := decodeSizes(prefs.StringWithFallback(prefQtTablePrefix+key+".widths", ""))
		if len(widths) == 0 || len(widths) != tree.Model().ColumnCount(qt.NewQModelIndex()) {
			continue
		}
		widths = normalizedColumnWidths(key, widths)
		applyTreeColumnMinimums(key, tree)
		for column, width := range widths {
			if width > 0 {
				tree.SetColumnWidth(column, width)
			}
		}
		tree.Header().SetSectionResizeMode(qt.QHeaderView__Interactive)
		tree.Header().SetStretchLastSection(false)
		mw.markTableManuallySized(key)
	}
}

func (mw *MainWindow) installTableColumnPersistence() {
	for key, table := range mw.tableRegistry() {
		if table == nil || table.HorizontalHeader() == nil {
			continue
		}
		tableKey := key
		table.HorizontalHeader().OnSectionResized(func(logicalIndex int, oldSize int, newSize int) {
			if oldSize == newSize || logicalIndex < 0 {
				return
			}
			if isProgrammaticColumnResize() {
				return
			}
			mw.markTableManuallySized(tableKey)
			mw.scheduleTableColumnPersistence()
		})
	}
	for key, tree := range mw.treeRegistry() {
		if tree == nil || tree.Header() == nil {
			continue
		}
		treeKey := key
		tree.Header().OnSectionResized(func(logicalIndex int, oldSize int, newSize int) {
			if oldSize == newSize || logicalIndex < 0 {
				return
			}
			if isProgrammaticColumnResize() {
				return
			}
			mw.markTableManuallySized(treeKey)
			mw.scheduleTableColumnPersistence()
		})
	}
}

func (mw *MainWindow) scheduleTableColumnPersistence() {
	if mw == nil {
		return
	}
	if mw.persistTimer == nil {
		mw.persistTimer = qt.NewQTimer()
		mw.persistTimer.SetSingleShot(true)
		mw.persistTimer.SetInterval(500)
		mw.persistTimer.OnTimeout(func() {
			mw.persistTableColumnWidths()
		})
	}
	mw.persistTimer.Start2()
}

func (mw *MainWindow) markTableManuallySized(key string) {
	switch key {
	case "objects":
		if mw.objectList != nil {
			mw.objectList.autoSized = true
		}
	case "alarms":
		if mw.alarmPanel != nil {
			mw.alarmPanel.autoSized = true
		}
	case "events":
		if mw.eventLog != nil {
			mw.eventLog.autoSized = true
		}
	case "object_zones", "object_zones_flat", "object_contacts", "object_events":
		if mw.workArea != nil {
			mw.workArea.markColumnsSized(key)
		}
	}
}

func (mw *MainWindow) persistTableColumnWidths() {
	prefs := mw.preferences()
	if prefs == nil {
		return
	}
	for key, table := range mw.tableRegistry() {
		if table == nil || table.Model() == nil {
			continue
		}
		count := table.Model().ColumnCount(qt.NewQModelIndex())
		widths := make([]int, 0, count)
		for column := 0; column < count; column++ {
			widths = append(widths, table.ColumnWidth(column))
		}
		widths = normalizedColumnWidths(key, widths)
		prefs.SetString(prefQtTablePrefix+key+".widths", encodeSizes(widths))
	}
	for key, tree := range mw.treeRegistry() {
		if tree == nil || tree.Model() == nil {
			continue
		}
		count := tree.Model().ColumnCount(qt.NewQModelIndex())
		widths := make([]int, 0, count)
		for column := 0; column < count; column++ {
			widths = append(widths, tree.ColumnWidth(column))
		}
		widths = normalizedColumnWidths(key, widths)
		prefs.SetString(prefQtTablePrefix+key+".widths", encodeSizes(widths))
	}
}

func normalizedColumnWidths(key string, widths []int) []int {
	minimums := minimumColumnWidths(key)
	if len(minimums) == 0 || len(widths) == 0 {
		return widths
	}
	normalized := append([]int(nil), widths...)
	for column := range normalized {
		if column < len(minimums) && normalized[column] < minimums[column] {
			normalized[column] = minimums[column]
		}
	}
	return normalized
}

func applyTableColumnMinimums(key string, table *qt.QTableView) {
	if table == nil || table.HorizontalHeader() == nil {
		return
	}
	minimums := minimumColumnWidths(key)
	if len(minimums) == 0 {
		table.HorizontalHeader().SetMinimumSectionSize(48)
		return
	}
	table.HorizontalHeader().SetMinimumSectionSize(minSliceValue(minimums, 48))
	for column, width := range minimums {
		table.SetColumnWidth(column, maxInt(table.ColumnWidth(column), width))
	}
}

func applyTreeColumnMinimums(key string, tree *qt.QTreeView) {
	if tree == nil || tree.Header() == nil {
		return
	}
	minimums := minimumColumnWidths(key)
	if len(minimums) == 0 {
		tree.Header().SetMinimumSectionSize(48)
		return
	}
	tree.Header().SetMinimumSectionSize(minSliceValue(minimums, 48))
	for column, width := range minimums {
		tree.SetColumnWidth(column, maxInt(tree.ColumnWidth(column), width))
	}
}

func minimumColumnWidths(key string) []int {
	switch key {
	case "objects":
		return []int{72, 190, 240}
	case "alarms":
		return []int{92, 70, 180, 260, 92, 92}
	case "events":
		return []int{120, 70, 150, 70, 220, 280, 92}
	case "object_zones_flat":
		return []int{72, 180, 120, 110}
	case "object_zones":
		return []int{82, 170, 72, 180, 120, 110}
	case "object_contacts":
		return []int{160, 130, 150, 110}
	case "object_events":
		return []int{128, 140, 70, 320}
	default:
		return nil
	}
}

func minSliceValue(values []int, fallback int) int {
	if len(values) == 0 {
		return fallback
	}
	minimum := values[0]
	for _, value := range values[1:] {
		if value < minimum {
			minimum = value
		}
	}
	return minimum
}

func (mw *MainWindow) savedSplitterSizes(key string, fallback []int) []int {
	prefs := mw.preferences()
	if prefs == nil {
		return fallback
	}
	sizes := decodeSizes(prefs.StringWithFallback(key, ""))
	if len(sizes) != len(fallback) {
		return fallback
	}
	return sizes
}

func normalizeTopSplitterSizes(sizes []int, availableWidth int) []int {
	if len(sizes) != 2 {
		sizes = []int{320, 1040}
	}
	if availableWidth <= 0 {
		availableWidth = sizes[0] + sizes[1]
	}
	if availableWidth < 800 {
		availableWidth = 800
	}

	const (
		minObjectListWidth = 240
		minWorkAreaWidth   = 560
	)
	maxLeft := availableWidth - minWorkAreaWidth
	if maxLeft < minObjectListWidth {
		maxLeft = minObjectListWidth
	}

	left := sizes[0]
	if left < minObjectListWidth {
		left = minObjectListWidth
	}
	if left > maxLeft {
		left = maxLeft
	}
	return []int{left, availableWidth - left}
}

func (mw *MainWindow) availableLayoutWidth() int {
	if mw == nil {
		return 0
	}
	width := mw.Width()
	availableWidth, _ := mw.availableScreenDimensions()
	if availableWidth > 0 && (width <= 0 || availableWidth < width) {
		return availableWidth
	}
	return width
}

func (mw *MainWindow) availableLayoutHeight() int {
	if mw == nil {
		return 0
	}
	height := mw.Height()
	_, availableHeight := mw.availableScreenDimensions()
	if availableHeight > 0 && (height <= 0 || availableHeight < height) {
		return availableHeight
	}
	return height
}

func (mw *MainWindow) availableScreenDimensions() (int, int) {
	if mw == nil {
		return 0, 0
	}
	screen := mw.Screen()
	if screen == nil {
		return 0, 0
	}
	geometry := screen.AvailableGeometry()
	if geometry == nil {
		return 0, 0
	}
	return geometry.Width(), geometry.Height()
}

func (mw *MainWindow) preferences() config.Preferences {
	if mw == nil || mw.app == nil {
		return nil
	}
	return mw.app.Preferences()
}

func encodeSizes(sizes []int) string {
	parts := make([]string, 0, len(sizes))
	for _, size := range sizes {
		parts = append(parts, strconv.Itoa(size))
	}
	return strings.Join(parts, ",")
}

func decodeSizes(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	sizes := make([]int, 0, len(parts))
	for _, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < 0 {
			return nil
		}
		sizes = append(sizes, value)
	}
	return sizes
}
