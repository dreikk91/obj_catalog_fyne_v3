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
	rightTabs *container.AppTabs
	eventsTab *container.TabItem
	alarmsTab *container.TabItem

	// Стан лічильників для бейджів правих вкладок.
	lastAlarmsCount   int
	lastCriticalCount int
	lastEventsCount   int

	// Поточна тема
	isDarkTheme bool

	statusLabel     *widget.Label
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
	log.Debug().Msg("Створення головного вікна...")
	mainWindow := fyneApp.NewWindow(fmt.Sprintf("Каталог об'єктів [%s]", ver.Label()))
	mainWindow.Resize(fyne.NewSize(1024, 768))
	log.Debug().Str("size", "1024x768").Msg("Головне вікно налаштовано")

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
	log.Debug().Msg("Структура додатку готова")
	log.Info().Str("version", ver.String()).Msg("Версія застосунку")

	// Встановлюємо тему
	log.Debug().Msg("Встановлення теми...")
	application.setTheme(isDark)
	log.Debug().Bool("darkTheme", isDark).Msg("Тема встановлена")

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
		log.Debug().Msg("Застосування темної теми...")
		a.fyneApp.Settings().SetTheme(apptheme.NewDarkTheme(uiCfg.FontSize))
	} else {
		log.Debug().Msg("Застосування світлої теми...")
		a.fyneApp.Settings().SetTheme(apptheme.NewLightTheme(uiCfg.FontSize))
	}
	log.Debug().Bool("darkTheme", dark).Float32("fontSize", uiCfg.FontSize).Msg("Тема застосована")
}

// buildUI будує головний інтерфейс
func (a *Application) buildUI() {
	log.Debug().Msg("Початок побудови UI компонентів...")

	// Створюємо UI компоненти
	log.Debug().Msg("Створення AlarmPanel...")
	provider := a.getDataProvider()
	a.alarmPanel = ui.NewAlarmPanelWidget(provider)
	log.Debug().Msg("AlarmPanel створена")

	log.Debug().Msg("Створення ObjectListPanel...")
	a.objectList = ui.NewObjectListPanel(provider)
	log.Debug().Msg("ObjectListPanel створена")

	log.Debug().Msg("Створення WorkAreaPanel...")
	a.workArea = ui.NewWorkAreaPanel(provider, a.mainWindow)
	log.Debug().Msg("WorkAreaPanel створена")

	log.Debug().Msg("Створення EventLogPanel...")
	a.eventLog = ui.NewEventLogPanel(provider)
	log.Debug().Msg("EventLogPanel створена")
	a.registerEventBusHandlers()

	log.Debug().Msg("Налаштування callbacks...")

	// Налаштовуємо callbacks
	a.objectList.OnObjectSelected = func(object models.Object) {
		log.Debug().Int("objectID", object.ID).Str("objectName", object.Name).Msg("Об'єкт вибраний з списку")
		// Для адміністратора при виборі об'єкта відкриваємо картку одразу.
		a.applyObjectContext(&object, true)
	}

	a.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога вибрана (одинарний клік)")
		// Оновлюємо контекст, але залишаємо відкритою вкладку "Тривоги".
		a.applyObjectContextByID(int64(alarm.ObjectID), false)
	}

	a.alarmPanel.OnAlarmActivated = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога активована (подвійний клік)")
		// Подвійний клік: відкриваємо вкладку деталей для вже вибраного об'єкта.
		a.selectDetailsTab()
	}

	a.eventLog.OnEventSelected = func(event models.Event) {
		log.Debug().Int("eventID", event.ID).Int("objectID", event.ObjectID).Msg("Подія вибрана")
		a.applyObjectContextByID(int64(event.ObjectID), true)
	}

	a.alarmPanel.OnProcessAlarm = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Msg("Початок обробки тривоги...")
		dialogs.ShowProcessAlarmDialog(a.mainWindow, alarm, func(result dialogs.ProcessAlarmResult) {
			log.Info().Int("alarmID", alarm.ID).Str("action", result.Action).Str("note", result.Note).Msg("Тривога оброблена")
			provider := a.getDataProvider()
			if provider == nil {
				dialogs.ShowInfoDialog(a.mainWindow, "Недоступно", "Провайдер даних недоступний.")
				return
			}
			provider.ProcessAlarm(fmt.Sprintf("%d", alarm.ID), "Диспетчер", result.Note)
			a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshAlarms: true})
			dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Тривогу оброблено: "+result.Action)
		})
	}

	log.Debug().Msg("Callbacks налаштовані")

	// Головне меню (в т.ч. адмінський функціонал з документації)
	a.mainWindow.SetMainMenu(a.buildMainMenu())

	// Кнопка перемикання теми
	themeBtn := widget.NewButtonWithIcon("", fyneTheme.ColorPaletteIcon(), nil)
	updateThemeButton := func() {
		if a.isDarkTheme {
			themeBtn.SetText("Світла")
		} else {
			themeBtn.SetText("Темна")
		}
	}
	themeBtn.OnTapped = func() {
		newDark := !a.isDarkTheme
		log.Debug().Bool("darkTheme", newDark).Msg("Перемикання теми...")
		a.setTheme(newDark)
		updateThemeButton()

		uiCfg := config.LoadUIConfig(a.fyneApp.Preferences())
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

		// Оновлюємо панелі, щоб застосувати нові кольори та палітри рядків.
		a.publishDataRefresh(eventbus.DataRefreshEvent{
			RefreshObjects: true,
			RefreshAlarms:  true,
			RefreshEvents:  true,
		})
	}
	updateThemeButton()

	// Кнопка налаштувань
	settingsBtn := widget.NewButtonWithIcon("Налаштування", fyneTheme.SettingsIcon(), func() {
		log.Debug().Msg("Відкриття діалогу налаштувань...")
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
				// Після зміни кольорів оновлюємо всі панелі, які їх використовують
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

	title := widget.NewLabel(fmt.Sprintf("Каталог об'єктів"))
	toolbar := container.NewHBox(title, layout.NewSpacer(), themeBtn, settingsBtn)

	// Таби: показуємо найважливіше першим (тривоги), додаємо лічильники.
	detailsTab := container.NewTabItem("КАРТКА ОБ'ЄКТА", a.workArea.Container)
	eventsTab := container.NewTabItem("ЖУРНАЛ ПОДІЙ", a.eventLog.Container)
	alarmsTab := container.NewTabItem("ТРИВОГИ", a.alarmPanel.Container)
	rightTabs := container.NewAppTabs(detailsTab, eventsTab, alarmsTab)
	a.configureTabsState(detailsTab, eventsTab, alarmsTab, rightTabs)

	// Синхронізуємо лічильники з панелями (викличеться після їх Refresh()).
	if a.alarmPanel != nil {
		a.alarmPanel.OnCountsChanged = func(total int, critical int) {
			// eventsCount тут не знаємо — не чіпаємо.
			a.updateTabBadges(total, critical, -1)
		}
		a.alarmPanel.OnNewCriticalAlarm = func(alarm models.Alarm) {
			// Для адміністратора не перемикаємо вкладку автоматично,
			// а лише м'яко сповіщаємо про нову тривогу.
			ui.ShowToast(a.mainWindow, fmt.Sprintf("Нова тривога: №%s %s", alarm.GetObjectNumberDisplay(), alarm.GetTypeDisplay()))
		}
	}
	if a.eventLog != nil {
		a.eventLog.OnCountChanged = func(count int) {
			a.updateTabBadges(-1, 0, count)
		}
	}

	log.Debug().Msg("Компонування макета...")

	// Layout: universal HSplit with right-side tabs (better for 1024x768 and 1920x1080)
	rootSplit := container.NewHSplit(a.objectList.Container, rightTabs)
	savedOffset := a.fyneApp.Preferences().FloatWithFallback(prefKeyObjectListSplitOffset, 0.32)
	// Захист від некоректних значень (щоб не "зламати" макет)
	if savedOffset < 0.10 || savedOffset > 0.90 {
		savedOffset = 0.32
	}
	rootSplit.SetOffset(savedOffset)

	a.statusLabel = widget.NewLabel(a.backendStatusConnectedText())
	shortcutsLabel := widget.NewLabel("Ctrl+1..3: вкладки | Ctrl+T: тема | Ctrl+F: пошук")
	statusBar := container.NewVBox(
		widget.NewSeparator(),
		container.NewHBox(a.statusLabel, layout.NewSpacer(), shortcutsLabel),
	)

	finalLayout := container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		statusBar, nil, nil,
		rootSplit,
	)
	a.mainWindow.SetContent(finalLayout)
	log.Debug().Msg("UI побудований та встановлений на вікно")

	// Запам'ятовуємо ширину (offset) списку об'єктів між запусками.
	// Split не має callback на drag, тому зберігаємо при закритті вікна.
	a.mainWindow.SetCloseIntercept(func() {
		if a.isShuttingDown {
			return
		}
		a.isShuttingDown = true
		if a.refreshLoopCancel != nil {
			a.refreshLoopCancel()
			a.refreshLoopCancel = nil
		}

		a.fyneApp.Preferences().SetFloat(prefKeyObjectListSplitOffset, rootSplit.Offset)

		// Закриваємо всі додаткові вікна (адмінські/службові) перед завершенням додатку.
		otherWindows := append([]fyne.Window(nil), a.fyneApp.Driver().AllWindows()...)
		for _, w := range otherWindows {
			if w == nil || w == a.mainWindow {
				continue
			}
			w.Close()
		}

		a.fyneApp.Quit()
	})

	a.registerShortcuts(themeBtn)
}

func (a *Application) buildMainMenu() *fyne.MainMenu {
	adminMenu := fyne.NewMenu("Адмін",
		fyne.NewMenuItem("Блокування відображення інформації", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowDisplayBlockingDialog(a.mainWindow, admin, func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true})
			})
		})),
		fyne.NewMenuItem("Емуляція подій", a.withAdminProvider(func(admin contracts.AdminProvider) {
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
		fyne.NewMenuItem("Новий об'єкт", a.withAdminProvider(func(admin contracts.AdminProvider) {
			a.openNewObjectDialog(admin)
		})),
		fyne.NewMenuItem("Змінити поточний", a.withAdminProvider(func(admin contracts.AdminProvider) {
			a.openEditCurrentObjectDialog(admin)
		})),
		fyne.NewMenuItem("Видалити поточний", a.withAdminProvider(func(admin contracts.AdminProvider) {
			a.confirmDeleteCurrentObject(admin)
		})),
	)

	adminSettings := fyne.NewMenu("Налаштування",
		fyne.NewMenuItem("Перевизначення подій", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowEventOverrideDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Управління повідомленнями адміністратора", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowAdminMessagesDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Контроль системи (БД/логи)", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowAdminSystemControlDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Налаштування пожежного моніторингу", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowFireMonitoringSettingsDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Керування об'єктами підсерверів", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowSubServerObjectsDialog(a.mainWindow, admin, func() {
				a.publishDataRefresh(eventbus.DataRefreshEvent{
					RefreshObjects: true,
					RefreshAlarms:  true,
				})
			})
		})),
	)

	adminMonitoringItems := []*fyne.MenuItem{
		fyne.NewMenuItem("Збір статистики", a.withAdminProvider(func(admin contracts.AdminProvider) {
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
		fyne.NewMenuItem("Конструктор ППК", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowPPKConstructorDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Типи об'єктів", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowObjectTypesDictionaryDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Регіони", a.withAdminProvider(func(admin contracts.AdminProvider) {
			dialogs.ShowRegionsDictionaryDialog(a.mainWindow, admin)
		})),
		fyne.NewMenuItem("Причини тривог", a.withAdminProvider(func(admin contracts.AdminProvider) {
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
		caslMenuItems := make([]*fyne.MenuItem, 0, 3)
		caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Звіти", func() {
			a.openCASLReportsDialog()
		}))
		if _, ok := a.resolveCASLObjectEditorProvider(); ok {
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Створити новий об'єкт", func() {
				a.openCASLObjectCreator()
			}))
			caslMenuItems = append(caslMenuItems, fyne.NewMenuItem("Редагувати поточний об'єкт", func() {
				a.openCASLObjectEditor()
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
	if a.statusLabel != nil {
		a.statusLabel.SetText("Джерела даних: перепідключення...")
	}

	buildResult, err := buildDataProviderFromConfig(cfg, a.fyneApp.Preferences(), true)
	if err != nil {
		log.Error().Err(err).Msg("❌ Помилка перевірки з'єднання з новими джерелами")
		if a.statusLabel != nil {
			a.statusLabel.SetText("Джерела даних: помилка підключення")
		}
		dialogs.ShowErrorDialog(a.mainWindow, "Помилка підключення", err)
		return
	}

	a.closeManagedDBs()
	a.managedDBs = buildResult.managedDBs
	a.setDataProvider(buildResult.provider)
	a.firebirdEnabled = buildResult.firebirdEnabled
	a.phoenixEnabled = buildResult.phoenixEnabled
	a.caslEnabled = buildResult.caslEnabled
	log.Debug().Msg("Провайдер даних оновлено")

	// Оновлюємо посилання в панелях
	log.Debug().Msg("Оновлення посилань на БД у панелях...")
	provider := a.getDataProvider()
	a.alarmPanel.Data = provider
	a.objectList.Data = provider
	a.workArea.Data = provider
	a.eventLog.Data = provider
	log.Debug().Msg("✓ Посилання оновлено")

	// Перезавантажуємо дані
	log.Debug().Msg("Перезавантаження даних у всіх панелях...")
	a.publishDataRefresh(eventbus.DataRefreshEvent{
		RefreshObjects: true,
		RefreshAlarms:  true,
		RefreshEvents:  true,
	})
	log.Debug().Msg("✓ Дані перезавантажено")

	log.Info().
		Bool("firebirdEnabled", buildResult.firebirdEnabled).
		Bool("phoenixEnabled", buildResult.phoenixEnabled).
		Bool("caslEnabled", buildResult.caslEnabled).
		Msg("✅ Перепідключення джерел даних завершено успішно")
	if a.statusLabel != nil {
		a.statusLabel.SetText(a.backendStatusConnectedText())
	}
	dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Підключення до джерел даних оновлено")
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

	a.setTheme(a.isDarkTheme)

	// Оновлюємо панелі
	log.Debug().Msg("Оновлення AlarmPanel...")
	a.alarmPanel.OnThemeChanged(cfg.FontSizeAlarms)

	log.Debug().Msg("Оновлення ObjectListPanel...")
	a.objectList.OnThemeChanged(cfg.FontSizeObjects)

	log.Debug().Msg("Оновлення WorkAreaPanel...")
	a.workArea.OnThemeChanged(cfg.FontSize)

	log.Debug().Msg("Оновлення EventLogPanel...")
	a.eventLog.OnThemeChanged(cfg.FontSizeEvents)
	a.publishDataRefresh(eventbus.DataRefreshEvent{
		RefreshObjects: true,
		RefreshAlarms:  true,
		RefreshEvents:  true,
	})

	log.Info().Msg("✅ Параметри інтерфейсу оновлено")
}
