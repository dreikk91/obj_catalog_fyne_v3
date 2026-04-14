package application

import (
	"context"
	"fmt"

	// "math/rand"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/eventbus"
	applogger "obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/models"
	apptheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	appversion "obj_catalog_fyne_v3/pkg/version"

	"github.com/rs/zerolog/log"
)

// Application зберігає стан додатку
type Application struct {
	fyneApp           fyne.App
	mainWindow        fyne.Window
	managedDBs        []managedDBResource
	refreshLoopCancel context.CancelFunc
	isShuttingDown    bool

	// Поточний вибраний об'єкт (для заголовка, контекстних фільтрів тощо)
	currentObject *models.Object
	// Поточна кількість активних тривог (для заголовка)
	currentAlarmsTotal int

	// Сховище даних (інтерфейс)
	dataProvider contracts.DataProvider
	providerMu   sync.RWMutex
	// Внутрішня шина подій для розв'язування UI-компонентів.
	eventBus *eventbus.Bus
	// Коротке вікно для об'єднання частих refresh-подій в одну.
	refreshCoalesceMu      sync.Mutex
	pendingRefresh         eventbus.DataRefreshEvent
	refreshCoalescePending bool
	// Пряме посилання на MockData ТІЛЬКИ для симуляції
	// mockData *contracts.MockData

	// UI компоненти (нові структури)
	alarmPanel *ui.AlarmPanelWidget
	objectList *ui.ObjectListPanel
	workArea   *ui.WorkAreaPanel
	eventLog   *ui.EventLogPanel

	// Праві вкладки (картка об'єкта / журнал / тривоги)
	rightTabs   *container.AppTabs
	bottomTabs  *container.AppTabs
	eventsTab   *container.TabItem
	alarmsTab   *container.TabItem
	objectSplit *container.Split
	bottomSplit *container.Split

	// Стан лічильників для бейджів правих вкладок.
	lastAlarmsCount   int
	lastCriticalCount int
	lastEventsCount   int

	// Поточна тема
	isDarkTheme bool

	statusLabel     *widget.Label
	themeBtn        *widget.Button
	settingsBtn     *widget.Button
	versionInfo     appversion.Info
	firebirdEnabled bool
	phoenixEnabled  bool
	caslEnabled     bool
}

func (a *Application) getDataProvider() contracts.DataProvider {
	if a == nil {
		return nil
	}
	a.providerMu.RLock()
	defer a.providerMu.RUnlock()
	return a.dataProvider
}

func (a *Application) setDataProvider(provider contracts.DataProvider) {
	if a == nil {
		return
	}
	a.providerMu.Lock()
	a.dataProvider = provider
	a.providerMu.Unlock()
}

// updateWindowTitle оновлює заголовок вікна з урахуванням
// вибраного об'єкта та кількості активних тривог.
func (a *Application) updateWindowTitle() {
	versionLabel := ""
	if strings.TrimSpace(a.versionInfo.Label()) != "" {
		versionLabel = fmt.Sprintf(" [%s]", a.versionInfo.Label())
	}
	base := "Каталог об'єктів"

	if a.currentObject != nil {
		base = fmt.Sprintf("Каталог об'єктів%s — %s (№%s)", versionLabel, a.currentObject.Name, viewmodels.ObjectDisplayNumber(*a.currentObject))
	}
	if a.currentAlarmsTotal > 0 {
		base = fmt.Sprintf("%s — Тривоги: %d", base, a.currentAlarmsTotal)
	}

	if a.mainWindow != nil {
		a.mainWindow.SetTitle(base)
	}
}

func (a *Application) backendStatusConnectedText() string {
	if a == nil {
		return "Джерело даних: —"
	}
	parts := make([]string, 0, 3)
	if a.firebirdEnabled {
		parts = append(parts, "БД/МІСТ: підключено")
	}
	if a.phoenixEnabled {
		parts = append(parts, "Phoenix: підключено")
	}
	if a.caslEnabled {
		parts = append(parts, "CASL Cloud: підключено")
	}
	if len(parts) == 0 {
		return "Джерела даних: не налаштовано"
	}
	return strings.Join(parts, " | ")
}

const (
	prefKeyObjectListSplitOffset = "ui.objectList.splitOffset"
	prefKeyBottomSplitOffset     = "ui.bottom.splitOffset"
	prefKeyDarkTheme             = "ui.theme.dark"
)

// NewApplication створює новий екземпляр додатку
func NewApplication() *Application {
	ver := appversion.Current()

	// Ініціалізація Fyne з унікальним ID для збереження налаштувань
	log.Info().Msg("Ініціалізація Fyne додатку...")
	fyneApp := app.NewWithID("com.most.obj_catalog_fyne_v3")
	log.Debug().Str("appID", "com.most.obj_catalog_fyne_v3").Msg("Fyne додаток створено")

	// Завантажуємо збережену тему (за замовчуванням - темна)
	isDark := fyneApp.Preferences().BoolWithFallback(prefKeyDarkTheme, true)

	// Створюємо головне вікно
	mainWindow := fyneApp.NewWindow(fmt.Sprintf("Каталог об'єктів [%s]", ver.Label()))
	mainWindow.Resize(fyne.NewSize(1024, 768))

	// Завантажуємо налаштування БД
	log.Info().Msg("Завантаження налаштувань БД...")
	dbCfg := config.LoadDBConfig(fyneApp.Preferences())
	dbCfg.LogLevel = applogger.SetLogLevel(dbCfg.LogLevel)
	log.Info().
		Str("host", dbCfg.Host).
		Str("port", dbCfg.Port).
		Str("user", dbCfg.User).
		Bool("firebirdEnabled", dbCfg.FirebirdEnabled).
		Bool("phoenixEnabled", dbCfg.PhoenixEnabled).
		Bool("caslEnabled", dbCfg.CASLEnabled || dbCfg.NormalizedMode() == config.BackendModeCASLCloud).
		Msg("Налаштування джерела даних завантажено")

	// Створюємо mock дані
	// mockData := contracts.NewMockData()

	// Ініціалізація основного провайдера БД/мосту
	log.Info().Msg("Ініціалізація провайдера даних...")
	buildResult, err := buildDataProviderFromConfig(dbCfg, fyneApp.Preferences(), false)
	if err != nil {
		log.Error().Err(err).Msg("Не вдалося повністю ініціалізувати джерела даних")
	}

	log.Info().Msg("Створення структури додатку...")
	application := &Application{
		fyneApp:      fyneApp,
		mainWindow:   mainWindow,
		managedDBs:   buildResult.managedDBs,
		dataProvider: buildResult.provider,
		eventBus:     eventbus.NewBus(),
		// mockData:   mockData,
		isDarkTheme:     isDark,
		versionInfo:     ver,
		firebirdEnabled: buildResult.firebirdEnabled,
		phoenixEnabled:  buildResult.phoenixEnabled,
		caslEnabled:     buildResult.caslEnabled,
	}
	log.Info().Str("version", ver.String()).Msg("Версія застосунку")

	// Встановлюємо тему
	application.setTheme(isDark)

	// Будуємо інтерфейс (це тепер швидко, бо все асинхронно)
	log.Info().Msg("Побудова UI компонентів...")
	application.buildUI()
	log.Info().Msg("UI побудовано успішно")

	// Показуємо вікно ЯКНАЙШВИДШЕ
	// А дані будуть підтягуватись у фоні (вже запущено в конструкторах панелей)

	// Запускаємо симуляцію подій / фонове оновлення
	application.startGettingEvents()

	log.Info().Msg("Ініціалізація завершена. Програма готова до роботи.")
	return application
}

// setTheme встановлює тему (темну або світлу)
func (a *Application) setTheme(dark bool) {
	a.isDarkTheme = dark
	// Зберігаємо вибір теми в налаштуваннях
	a.fyneApp.Preferences().SetBool(prefKeyDarkTheme, dark)

	uiCfg := config.LoadUIConfig(a.fyneApp.Preferences())
	if dark {
		a.fyneApp.Settings().SetTheme(apptheme.NewDarkTheme(uiCfg.FontSize))
	} else {
		a.fyneApp.Settings().SetTheme(apptheme.NewLightTheme(uiCfg.FontSize))
	}
}

// buildUI будує головний інтерфейс
func (a *Application) buildUI() {
	a.buildUIPanels()
	a.registerEventBusHandlers()
	a.configurePanelCallbacks()

	// Головне меню (в т.ч. адмінський функціонал з документації)
	a.mainWindow.SetMainMenu(a.buildMainMenu())

	a.themeBtn = a.buildThemeButton()
	a.settingsBtn = a.buildSettingsButton()
	a.bindTabBadgeHandlers()
	a.rebuildMainWindowLayout(config.LoadUIConfig(a.fyneApp.Preferences()))
	a.registerShortcuts(a.themeBtn)
}

func (a *Application) buildUIPanels() {
	provider := a.getDataProvider()
	a.alarmPanel = ui.NewAlarmPanelWidget(provider)
	a.objectList = ui.NewObjectListPanel(provider)
	a.workArea = ui.NewWorkAreaPanel(provider, a.mainWindow)
	a.eventLog = ui.NewEventLogPanel(provider)
}

func (a *Application) configurePanelCallbacks() {
	a.objectList.OnObjectSelected = func(object models.Object) {
		log.Debug().Int("objectID", object.ID).Str("objectName", object.Name).Msg("Об'єкт вибраний з списку")
		a.applyObjectContext(&object, true)
	}

	a.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога вибрана (одинарний клік)")
		a.applyObjectContextByID(int64(alarm.ObjectID), false)
	}

	a.alarmPanel.OnAlarmActivated = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога активована (подвійний клік)")
		a.selectDetailsTab()
	}

	a.eventLog.OnEventSelected = func(event models.Event) {
		log.Debug().Int("eventID", event.ID).Int("objectID", event.ObjectID).Msg("Подія вибрана")
		a.applyObjectContextByID(int64(event.ObjectID), true)
	}

	a.alarmPanel.OnProcessAlarm = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Msg("Початок обробки тривоги...")
		provider := a.getDataProvider()
		if provider == nil {
			dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "Провайдер даних недоступний.")
			return
		}

		dialogs.ShowProcessAlarmDialog(a.mainWindow, alarm, provider, "Диспетчер", func() {
			log.Info().Int("alarmID", alarm.ID).Msg("Тривога відпрацьована")
			a.publishDataRefresh(eventbus.DataRefreshEvent{
				RefreshAlarms: true,
				RefreshEvents: true,
			})
			dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Тривогу відпрацьовано.")
		})
	}
}

func (a *Application) buildThemeButton() *widget.Button {
	themeBtn := widget.NewButtonWithIcon("", fyneTheme.ColorPaletteIcon(), nil)
	a.updateThemeButtonLabel(themeBtn)
	themeBtn.OnTapped = func() {
		a.toggleTheme(themeBtn)
	}
	return themeBtn
}

func (a *Application) updateThemeButtonLabel(themeBtn *widget.Button) {
	if themeBtn == nil {
		return
	}
	if a.isDarkTheme {
		themeBtn.SetText("Світла")
		return
	}
	themeBtn.SetText("Темна")
}

func (a *Application) toggleTheme(themeBtn *widget.Button) {
	newDark := !a.isDarkTheme
	a.setTheme(newDark)
	a.updateThemeButtonLabel(themeBtn)

	uiCfg := config.LoadUIConfig(a.fyneApp.Preferences())
	a.applyThemeToPanels(uiCfg)
	a.publishDataRefresh(eventbus.DataRefreshEvent{
		RefreshObjects: true,
		RefreshAlarms:  true,
		RefreshEvents:  true,
	})
}

func (a *Application) applyThemeToPanels(uiCfg config.UIConfig) {
	if a.alarmPanel != nil {
		a.alarmPanel.OnThemeChanged(uiCfg.FontSizeAlarms)
	}
	if a.objectList != nil {
		a.objectList.OnThemeChanged(uiCfg.FontSizeObjects)
	}
	if a.workArea != nil {
		a.workArea.OnThemeChanged(uiCfg.FontSize)
	}
	if a.eventLog != nil {
		a.eventLog.OnThemeChanged(uiCfg.FontSizeEvents)
	}
}

func (a *Application) buildSettingsButton() *widget.Button {
	return widget.NewButtonWithIcon("Налаштування", fyneTheme.SettingsIcon(), func() {
		dialogs.ShowSettingsDialog(
			a.mainWindow,
			a.resolveAdminProvider(),
			a.fyneApp.Preferences(),
			a.isDarkTheme,
			func(dbCfg config.DBConfig, uiCfg config.UIConfig) {
				log.Info().Str("host", dbCfg.Host).Msg("Параметри в діалозі налаштувань змінено")
				a.Reconnect(dbCfg)
				a.RefreshUI(uiCfg)
			},
			func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{
					RefreshObjects: true,
					RefreshAlarms:  true,
					RefreshEvents:  true,
				})
				if a.workArea != nil && a.workArea.EventsList != nil {
					a.workArea.EventsList.Refresh()
				}
			},
		)
	})
}

func (a *Application) buildToolbar(themeBtn *widget.Button, settingsBtn *widget.Button) fyne.CanvasObject {
	title := widget.NewLabel("Каталог об'єктів")
	return container.NewHBox(title, layout.NewSpacer(), themeBtn, settingsBtn)
}

type journalLayoutPlan struct {
	rightShowsEvents  bool
	rightShowsAlarms  bool
	bottomShowsEvents bool
	bottomShowsAlarms bool
}

func buildJournalLayoutPlan(uiCfg config.UIConfig) journalLayoutPlan {
	return journalLayoutPlan{
		rightShowsEvents:  !uiCfg.ShowBottomEventJournal,
		rightShowsAlarms:  !uiCfg.ShowBottomAlarmJournal,
		bottomShowsEvents: uiCfg.ShowBottomEventJournal,
		bottomShowsAlarms: uiCfg.ShowBottomAlarmJournal,
	}
}

func (a *Application) rebuildMainWindowLayout(uiCfg config.UIConfig) {
	if a == nil || a.mainWindow == nil || a.objectList == nil || a.workArea == nil || a.alarmPanel == nil || a.eventLog == nil {
		return
	}

	log.Debug().
		Bool("bottomAlarms", uiCfg.ShowBottomAlarmJournal).
		Bool("bottomEvents", uiCfg.ShowBottomEventJournal).
		Msg("Компонування макета...")

	toolbar := a.buildToolbar(a.themeBtn, a.settingsBtn)
	rightTabs, bottomTabs := a.buildJournalTabs(uiCfg)
	content := a.buildMainContent(rightTabs, bottomTabs)
	statusBar := a.buildStatusBar()

	finalLayout := container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		statusBar, nil, nil,
		content,
	)
	a.mainWindow.SetContent(finalLayout)

	a.installCloseIntercept()
	a.updateTabBadges(a.lastAlarmsCount, a.lastCriticalCount, a.lastEventsCount)
}

func (a *Application) buildJournalTabs(uiCfg config.UIConfig) (*container.AppTabs, *container.AppTabs) {
	plan := buildJournalLayoutPlan(uiCfg)

	detailsTab := container.NewTabItem("КАРТКА ОБ'ЄКТА", a.workArea.Container)
	rightItems := []*container.TabItem{detailsTab}
	var eventsTab *container.TabItem
	var alarmsTab *container.TabItem

	if plan.rightShowsEvents {
		eventsTab = container.NewTabItem("ЖУРНАЛ ПОДІЙ", a.eventLog.Container)
		rightItems = append(rightItems, eventsTab)
	}
	if plan.rightShowsAlarms {
		alarmsTab = container.NewTabItem("АКТИВНІ ТРИВОГИ", a.alarmPanel.Container)
		rightItems = append(rightItems, alarmsTab)
	}

	rightTabs := container.NewAppTabs(rightItems...)

	var bottomTabs *container.AppTabs
	bottomItems := make([]*container.TabItem, 0, 2)
	if plan.bottomShowsAlarms {
		alarmsTab = container.NewTabItem("АКТИВНІ ТРИВОГИ", a.alarmPanel.Container)
		bottomItems = append(bottomItems, alarmsTab)
	}
	if plan.bottomShowsEvents {
		eventsTab = container.NewTabItem("ЖУРНАЛ ПОДІЙ", a.eventLog.Container)
		bottomItems = append(bottomItems, eventsTab)
	}
	if len(bottomItems) > 0 {
		bottomTabs = container.NewAppTabs(bottomItems...)
	}

	a.configureTabsState(detailsTab, eventsTab, alarmsTab, rightTabs, bottomTabs)
	return rightTabs, bottomTabs
}

func (a *Application) bindTabBadgeHandlers() {
	if a.alarmPanel != nil {
		a.alarmPanel.OnCountsChanged = func(total int, critical int) {
			a.updateTabBadges(total, critical, -1)
		}
		a.alarmPanel.OnNewCriticalAlarm = func(alarm models.Alarm) {
			ui.ShowToast(a.mainWindow, fmt.Sprintf("Нова тривога: №%s %s", alarm.GetObjectNumberDisplay(), alarm.GetTypeDisplay()))
		}
	}
	if a.eventLog != nil {
		a.eventLog.OnCountChanged = func(count int) {
			a.updateTabBadges(-1, 0, count)
		}
	}
}

func (a *Application) buildMainContent(rightTabs *container.AppTabs, bottomTabs *container.AppTabs) fyne.CanvasObject {
	a.objectSplit = container.NewHSplit(a.objectList.Container, rightTabs)
	a.objectSplit.SetOffset(a.savedObjectListSplitOffset())

	if bottomTabs == nil {
		a.bottomSplit = nil
		return a.objectSplit
	}

	a.bottomSplit = container.NewVSplit(a.objectSplit, bottomTabs)
	a.bottomSplit.SetOffset(a.savedBottomSplitOffset())
	return a.bottomSplit
}

func (a *Application) savedObjectListSplitOffset() float64 {
	savedOffset := a.fyneApp.Preferences().FloatWithFallback(prefKeyObjectListSplitOffset, 0.32)
	if savedOffset < 0.10 || savedOffset > 0.90 {
		return 0.32
	}
	return savedOffset
}

func (a *Application) savedBottomSplitOffset() float64 {
	savedOffset := a.fyneApp.Preferences().FloatWithFallback(prefKeyBottomSplitOffset, 0.68)
	if savedOffset < 0.35 || savedOffset > 0.90 {
		return 0.68
	}
	return savedOffset
}

func (a *Application) buildStatusBar() fyne.CanvasObject {
	a.statusLabel = widget.NewLabel(a.backendStatusConnectedText())
	shortcutsLabel := widget.NewLabel("Ctrl+1..3: вкладки | Ctrl+T: тема | Ctrl+F: пошук")
	return container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(a.statusLabel, layout.NewSpacer(), shortcutsLabel),
	)
}

func (a *Application) persistCurrentLayoutOffsets() {
	if a == nil || a.fyneApp == nil {
		return
	}
	if a.objectSplit != nil {
		a.fyneApp.Preferences().SetFloat(prefKeyObjectListSplitOffset, a.objectSplit.Offset)
	}
	if a.bottomSplit != nil {
		a.fyneApp.Preferences().SetFloat(prefKeyBottomSplitOffset, a.bottomSplit.Offset)
	}
}

func (a *Application) installCloseIntercept() {
	a.mainWindow.SetCloseIntercept(func() {
		if a.isShuttingDown {
			return
		}
		a.isShuttingDown = true
		if a.refreshLoopCancel != nil {
			a.refreshLoopCancel()
			a.refreshLoopCancel = nil
		}
		if provider := a.getDataProvider(); provider != nil {
			if shutdowner, ok := provider.(contracts.ShutdownProvider); ok {
				shutdowner.Shutdown()
			}
		}

		a.persistCurrentLayoutOffsets()

		otherWindows := append([]fyne.Window(nil), a.fyneApp.Driver().AllWindows()...)
		for _, w := range otherWindows {
			if w == nil || w == a.mainWindow {
				continue
			}
			w.Close()
		}

		a.fyneApp.Quit()
	})
}

func (a *Application) buildMainMenu() *fyne.MainMenu {
	adminMenu := fyne.NewMenu("Адмін",
		fyne.NewMenuItem("Блокування відображення інформації", withAdminCapability(a, func(admin adminDisplayBlockingProvider) {
			dialogs.ShowDisplayBlockingDialog(a.mainWindow, admin, func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true})
			})
		})),
		fyne.NewMenuItem("Емуляція подій", withAdminCapability(a, func(admin adminEventEmulationProvider) {
			dialogs.ShowEventEmulationDialog(a.mainWindow, admin, func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{
					RefreshObjects: true,
					RefreshAlarms:  true,
					RefreshEvents:  true,
				})
			})
		})),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Об'єкти", nil),
		fyne.NewMenuItem("Налаштування", nil),
		fyne.NewMenuItem("Моніторинг", nil),
	)

	adminObjects := fyne.NewMenu("Об'єкти",
		fyne.NewMenuItem("Новий об'єкт", withAdminCapability(a, func(admin contracts.AdminObjectWizardProvider) {
			a.openNewObjectDialog(admin)
		})),
		fyne.NewMenuItem("Змінити поточний", withAdminCapability(a, func(admin contracts.AdminObjectCardProvider) {
			a.openEditCurrentObjectDialog(admin)
		})),
		fyne.NewMenuItem("Видалити поточний", withAdminCapability(a, func(admin adminObjectDeleteProvider) {
			a.confirmDeleteCurrentObject(admin)
		})),
	)

	adminSettings := fyne.NewMenu("Налаштування",
		fyne.NewMenuItem("Перевизначення подій", withAdminCapability(a, func(admin adminEventOverrideProvider) {
			dialogs.ShowEventOverrideDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Управління повідомленнями адміністратора", withAdminCapability(a, func(admin adminMessagesProvider) {
			dialogs.ShowAdminMessagesDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Контроль системи (БД/логи)", withAdminCapability(a, func(admin adminSystemControlProvider) {
			dialogs.ShowAdminSystemControlDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Налаштування пожежного моніторингу", withAdminCapability(a, func(admin adminFireMonitoringProvider) {
			dialogs.ShowFireMonitoringSettingsDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Керування об'єктами підсерверів", withAdminCapability(a, func(admin adminSubServerObjectsProvider) {
			dialogs.ShowSubServerObjectsDialog(a.mainWindow, admin, func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{
					RefreshObjects: true,
					RefreshAlarms:  true,
				})
			})
		})),
	)

	adminMonitoringItems := []*fyne.MenuItem{
		fyne.NewMenuItem("Збір статистики", withAdminCapability(a, func(admin adminStatisticsProvider) {
			dialogs.ShowStatisticsDialog(a.mainWindow, admin)
		})),
	}
	if _, ok := a.resolveSIMInventoryReportProvider(); ok {
		adminMonitoringItems = append(adminMonitoringItems, fyne.NewMenuItem("Звіт по SIM-картах", func() {
			a.openSIMInventoryReportDialog()
		}))
	}
	adminMonitoring := fyne.NewMenu("Моніторинг", adminMonitoringItems...)

	adminDirectories := fyne.NewMenu("Довідники",
		fyne.NewMenuItem("Конструктор ППК", withAdminCapability(a, func(admin adminPPKConstructorProvider) {
			dialogs.ShowPPKConstructorDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Типи об'єктів", withAdminCapability(a, func(admin adminObjectTypesProvider) {
			dialogs.ShowObjectTypesDictionaryDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Регіони", withAdminCapability(a, func(admin adminRegionsProvider) {
			dialogs.ShowRegionsDictionaryDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Причини тривог", withAdminCapability(a, func(admin adminAlarmReasonsProvider) {
			dialogs.ShowAlarmReasonsDictionaryDialog(a.mainWindow, admin)
		})),
	)

	// В Fyne вкладений пункт меню задається через ChildMenu.
	adminMenu.Items[3].ChildMenu = adminObjects
	adminMenu.Items[4].ChildMenu = adminSettings
	adminMenu.Items[5].ChildMenu = adminMonitoring
	adminMenu.Items = append(adminMenu.Items, fyne.NewMenuItem("Довідники", nil))
	adminMenu.Items[len(adminMenu.Items)-1].ChildMenu = adminDirectories

	helpMenu := fyne.NewMenu("Довідка",
		fyne.NewMenuItem("Про версію", func() {
			dialogs.ShowInfoDialog(a.mainWindow, "Про версію", a.versionInfo.FullText())
		}),
	)

	menus := []*fyne.Menu{adminMenu}
	if _, reportsOK := a.resolveCASLReportsProvider(); reportsOK {
		caslMenuItems := make([]*fyne.MenuItem, 0, 8)
		caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Звіти", func() {
			a.openCASLReportsDialog()
		}))
		if _, ok := a.resolveCASLObjectEditorProvider(); ok {
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItemSeparator())
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Створити новий об'єкт", func() {
				a.openCASLObjectCreator()
			}))
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Редагувати поточний об'єкт", func() {
				a.openCASLObjectEditor()
			}))
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItemSeparator())
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Блокування поточного об'єкта", func() {
				a.openCASLObjectBlockDialog()
			}))
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Видалити поточний об'єкт", func() {
				a.openCASLObjectDeleteDialog()
			}))
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Корзина об'єктів", func() {
				a.openCASLObjectBasketDialog()
			}))
		}
		caslMenu := fyne.NewMenu("CASL", caslMenuItems...)
		menus = append(menus, caslMenu)
	} else if _, editorOK := a.resolveCASLObjectEditorProvider(); editorOK {
		caslMenu := fyne.NewMenu("CASL",
			fyne.NewMenuItem("Створити новий об'єкт", func() {
				a.openCASLObjectCreator()
			}),
			fyne.NewMenuItem("Редагувати поточний об'єкт", func() {
				a.openCASLObjectEditor()
			}),
			fyne.NewMenuItemSeparator(),
			fyne.NewMenuItem("Блокування поточного об'єкта", func() {
				a.openCASLObjectBlockDialog()
			}),
			fyne.NewMenuItem("Видалити поточний об'єкт", func() {
				a.openCASLObjectDeleteDialog()
			}),
			fyne.NewMenuItem("Корзина об'єктів", func() {
				a.openCASLObjectBasketDialog()
			}),
		)
		menus = append(menus, caslMenu)
	}
	menus = append(menus, helpMenu)

	return fyne.NewMainMenu(menus...)
}

func (a *Application) resolveAdminProvider() contracts.AdminProvider {
	if a == nil {
		return nil
	}
	provider := a.getDataProvider()
	if provider == nil {
		return nil
	}
	admin, ok := backend.AsAdminProvider(provider)
	if !ok || admin == nil {
		return nil
	}
	return admin
}

// Run запускає додаток
func (a *Application) Run() {
	log.Info().Msg("Запуск основного цикла додатку (UI loop)...")
	defer a.closeManagedDBs()

	// Робимо рядок пошуку активним (виділеним) одразу після старту.
	if a.objectList != nil && a.objectList.SearchEntry != nil {
		a.mainWindow.Canvas().Focus(a.objectList.SearchEntry)
	}
	a.mainWindow.ShowAndRun()
	log.Info().Msg("Основний цикл завершено")
}

// Reconnect перепідключає джерело даних та оновлює провайдери.
func (a *Application) Reconnect(cfg config.DBConfig) {
	cfg.LogLevel = applogger.SetLogLevel(cfg.LogLevel)
	log.Warn().Msg("🔄 Перепідключення до джерел даних...")

	// Виконуємо операції з БД у горутині, щоб не блокувати UI
	go func() {
		fyne.Do(func() {
			if a.statusLabel != nil {
				a.statusLabel.SetText("Джерела даних: перепідключення...")
			}
		})

		buildResult, err := buildDataProviderFromConfig(cfg, a.fyneApp.Preferences(), true)
		if err != nil {
			log.Error().Err(err).Msg("❌ Помилка перевірки з'єднання з новими джерелами")
			fyne.Do(func() {
				if a.statusLabel != nil {
					a.statusLabel.SetText("Джерела даних: помилка підключення")
				}
				dialogs.ShowErrorDialog(a.mainWindow, "Помилка підключення", err)
			})
			return
		}

		a.closeManagedDBs()
		a.managedDBs = buildResult.managedDBs
		a.setDataProvider(buildResult.provider)
		a.firebirdEnabled = buildResult.firebirdEnabled
		a.phoenixEnabled = buildResult.phoenixEnabled
		a.caslEnabled = buildResult.caslEnabled
		log.Debug().Msg("Провайдер даних оновлено")

		// Оновлюємо посилання в панелях та перезавантажуємо дані
		provider := a.getDataProvider()
		a.alarmPanel.Data = provider
		a.objectList.Data = provider
		a.workArea.Data = provider
		a.eventLog.Data = provider

		a.publishDataRefresh(eventbus.DataRefreshEvent{
			RefreshObjects: true,
			RefreshAlarms:  true,
			RefreshEvents:  true,
		})

		log.Info().
			Bool("firebirdEnabled", buildResult.firebirdEnabled).
			Bool("phoenixEnabled", buildResult.phoenixEnabled).
			Bool("caslEnabled", buildResult.caslEnabled).
			Msg("✅ Перепідключення джерел даних завершено успішно")

		fyne.Do(func() {
			if a.statusLabel != nil {
				a.statusLabel.SetText(a.backendStatusConnectedText())
			}
			dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Підключення до джерел даних оновлено")
		})
	}()
}

func (a *Application) closeManagedDBs() {
	if a == nil || len(a.managedDBs) == 0 {
		return
	}
	log.Debug().Int("count", len(a.managedDBs)).Msg("Закриття з'єднань із джерелами даних...")
	closeManagedDBResources(a.managedDBs)
	a.managedDBs = nil
}

// RefreshUI оновлює інтерфейс (тему, шрифти)
func (a *Application) RefreshUI(cfg config.UIConfig) {
	log.Info().Float32("fontSize", cfg.FontSize).Msg("🎨 Оновлення параметрів інтерфейсу...")
	log.Debug().Float32("fontSizeAlarms", cfg.FontSizeAlarms).Float32("fontSizeObjects", cfg.FontSizeObjects).Float32("fontSizeEvents", cfg.FontSizeEvents).Msg("Нові розміри шрифтів")

	a.persistCurrentLayoutOffsets()
	a.setTheme(a.isDarkTheme)

	a.applyThemeToPanels(cfg)
	a.rebuildMainWindowLayout(cfg)
	a.publishDataRefresh(eventbus.DataRefreshEvent{
		RefreshObjects: true,
		RefreshAlarms:  true,
		RefreshEvents:  true,
	})
	a.startGettingEvents()
	if a.alarmPanel != nil {
		a.alarmPanel.ReloadSelectedCaseHistory()
	}

	log.Info().Msg("✅ Параметри інтерфейсу оновлено")
}
