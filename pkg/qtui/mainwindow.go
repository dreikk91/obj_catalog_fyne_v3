//go:build qt

package qtui

import (
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
)

const (
	prefQtWindowWidth       = "qt.window.width"
	prefQtWindowHeight      = "qt.window.height"
	prefQtMainSplitterSizes = "qt.splitter.main.sizes"
	prefQtTopSplitterSizes  = "qt.splitter.top.sizes"
	prefQtTablePrefix       = "qt.table."
)

type MainWindow struct {
	*qt.QMainWindow

	app          *App
	mainSplitter *qt.QSplitter
	topSplitter  *qt.QSplitter

	objectList *ObjectListPanel
	workArea   *WorkAreaPanel
	alarmPanel *AlarmPanel
	eventLog   *EventLogPanel

	statusLabel *qt.QLabel

	OnSettingsRequested func()
	OnRefreshRequested  func()
}

func NewMainWindow(app *App) *MainWindow {
	mw := &MainWindow{
		QMainWindow: qt.NewQMainWindow2(),
		app:         app,
	}

	mw.SetWindowTitle("АРМ Пожежної Безпеки - Qt UI")
	mw.restoreWindowSize()
	mw.SetStyleSheet(NativeWindowsStyleSheet)

	mw.buildMenuBar()
	mw.buildToolBar()
	mw.buildStatusBar()
	mw.buildLayout()
	mw.restoreTableColumnWidths()
	mw.registerShortcuts()
	mw.installClosePersistence()

	return mw
}

func (mw *MainWindow) buildMenuBar() {
	menuBar := qt.NewQMenuBar(mw.QWidget)
	fileMenu := menuBar.AddMenuWithTitle("Файл")
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
	viewMenu.AddActionWithText("Світла тема")
	viewMenu.AddActionWithText("Темна тема")

	helpMenu := menuBar.AddMenuWithTitle("Допомога")
	helpMenu.AddActionWithText("Про програму")

	mw.SetMenuBar(menuBar)
}

func (mw *MainWindow) buildToolBar() {
	toolbar := qt.NewQToolBar4("Головна панель", mw.QWidget)
	toolbar.SetMovable(false)
	settingsAction := toolbar.AddActionWithText("Налаштування")
	settingsAction.SetShortcut(qt.NewQKeySequence2("Ctrl+,"))
	settingsAction.OnTriggered(func() {
		if mw.OnSettingsRequested != nil {
			mw.OnSettingsRequested()
		}
	})
	refreshAction := toolbar.AddActionWithText("Оновити")
	refreshAction.SetShortcut(qt.NewQKeySequence2("Ctrl+R"))
	refreshAction.OnTriggered(func() {
		if mw.OnRefreshRequested != nil {
			mw.OnRefreshRequested()
		}
	})
	toolbar.AddActionWithText("Експорт")
	toolbar.AddActionWithText("Допомога")
	mw.AddToolBarWithToolbar(toolbar)
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
	mw.objectList = NewObjectListPanel()
	mw.workArea = NewWorkAreaPanel()
	mw.alarmPanel = NewAlarmPanel(mw.app.Preferences())
	mw.eventLog = NewEventLogPanel(mw.app.Preferences())

	mw.topSplitter = qt.NewQSplitter3(qt.Horizontal)
	mw.topSplitter.AddWidget(mw.objectList.QWidget)
	mw.topSplitter.AddWidget(mw.workArea.QWidget)
	mw.topSplitter.SetSizes(mw.savedSplitterSizes(prefQtTopSplitterSizes, []int{360, 920}))

	bottomTabs := qt.NewQTabWidget2()
	bottomTabs.AddTab(mw.alarmPanel.QWidget, "Тривоги")
	bottomTabs.AddTab(mw.eventLog.QWidget, "Журнал подій")

	mw.mainSplitter = qt.NewQSplitter3(qt.Vertical)
	mw.mainSplitter.AddWidget(mw.topSplitter.QWidget)
	mw.mainSplitter.AddWidget(bottomTabs.QWidget)
	mw.mainSplitter.SetSizes(mw.savedSplitterSizes(prefQtMainSplitterSizes, []int{650, 250}))

	mw.SetCentralWidget(mw.mainSplitter.QWidget)
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
	if prefs == nil {
		mw.Resize(1280, 900)
		return
	}
	width := prefs.IntWithFallback(prefQtWindowWidth, 1280)
	height := prefs.IntWithFallback(prefQtWindowHeight, 900)
	if width < 800 {
		width = 800
	}
	if height < 600 {
		height = 600
	}
	mw.Resize(width, height)
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
	if mw.mainSplitter != nil {
		prefs.SetString(prefQtMainSplitterSizes, encodeSizes(mw.mainSplitter.Sizes()))
	}
	if mw.topSplitter != nil {
		prefs.SetString(prefQtTopSplitterSizes, encodeSizes(mw.topSplitter.Sizes()))
	}
	mw.persistTableColumnWidths()
}

func (mw *MainWindow) tableRegistry() map[string]*qt.QTableView {
	tables := map[string]*qt.QTableView{}
	if mw.objectList != nil {
		tables["objects"] = mw.objectList.table
	}
	if mw.alarmPanel != nil {
		tables["alarms"] = mw.alarmPanel.table
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
		for column, width := range widths {
			if width > 0 {
				table.SetColumnWidth(column, width)
			}
		}
		table.HorizontalHeader().SetSectionResizeMode(qt.QHeaderView__Interactive)
		table.HorizontalHeader().SetStretchLastSection(true)
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
		for column, width := range widths {
			if width > 0 {
				tree.SetColumnWidth(column, width)
			}
		}
		tree.Header().SetSectionResizeMode(qt.QHeaderView__Interactive)
		tree.Header().SetStretchLastSection(true)
		mw.markTableManuallySized(key)
	}
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
			mw.workArea.autoSized = true
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
		prefs.SetString(prefQtTablePrefix+key+".widths", encodeSizes(widths))
	}
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
