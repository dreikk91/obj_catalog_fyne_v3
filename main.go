package main

import (
	"context"
	"fmt"
	// "math/rand"
	"runtime/debug"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/models"
	apptheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Application зберігає стан додатку
type Application struct {
	fyneApp        fyne.App
	mainWindow     fyne.Window
	db             *sqlx.DB
	dbHealthCancel context.CancelFunc

	// Поточний вибраний об'єкт (для заголовка, контекстних фільтрів тощо)
	currentObject *models.Object
	// Поточна кількість активних тривог (для заголовка)
	currentAlarmsTotal int

	// Сховище даних (інтерфейс)
	dataProvider data.DataProvider
	// Пряме посилання на MockData ТІЛЬКИ для симуляції
	// mockData *data.MockData

	// UI компоненти (нові структури)
	alarmPanel *ui.AlarmPanelWidget
	objectList *ui.ObjectListPanel
	workArea   *ui.WorkAreaPanel
	eventLog   *ui.EventLogPanel

	// Праві вкладки (картка об'єкта / журнал / тривоги)
	rightTabs *container.AppTabs

	// Поточна тема
	isDarkTheme bool

	statusLabel *widget.Label
}

// updateWindowTitle оновлює заголовок вікна з урахуванням
// вибраного об'єкта та кількості активних тривог.
func (a *Application) updateWindowTitle() {
	base := "Каталог об'єктів"

	if a.currentObject != nil {
		base = fmt.Sprintf("Каталог об'єктів — %s (№%d)", a.currentObject.Name, a.currentObject.ID)
	}
	if a.currentAlarmsTotal > 0 {
		base = fmt.Sprintf("%s — Тривоги: %d", base, a.currentAlarmsTotal)
	}

	if a.mainWindow != nil {
		a.mainWindow.SetTitle(base)
	}
}

const (
	prefKeyObjectListSplitOffset = "ui.objectList.splitOffset"
	prefKeyDarkTheme             = "ui.theme.dark"
)

func main() {
	// Ініціалізація логера
	logConfig := logger.DefaultConfig()
	if err := logger.Setup(logConfig); err != nil {
		fmt.Printf("Помилка налаштування логера: %v\n", err)
	}

	log.Info().Str("level", logConfig.LogLevel).Str("logDir", logConfig.LogDir).Msg("Запуск програми - АРМ Пожежної Безпеки v1.0")

	// Додаємо базове відновлення після паніки
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("КРИТИЧНА ПОМИЛКА (Panic)")
		}
		log.Info().Msg("Завершення програми")
	}()

	// Створюємо додаток
	log.Debug().Msg("Ініціалізація додатку...")
	application := NewApplication()
	log.Info().Msg("Додаток ініціалізовано. Запуск UI...")
	application.Run()
}

// NewApplication створює новий екземпляр додатку
func NewApplication() *Application {
	// Ініціалізація Fyne з унікальним ID для збереження налаштувань
	log.Info().Msg("Ініціалізація Fyne додатку...")
	fyneApp := app.NewWithID("com.most.obj_catalog_fyne_v3")
	log.Debug().Str("appID", "com.most.obj_catalog_fyne_v3").Msg("Fyne додаток створено")

	// Завантажуємо збережену тему (за замовчуванням - темна)
	isDark := fyneApp.Preferences().BoolWithFallback(prefKeyDarkTheme, true)

	// Створюємо головне вікно
	log.Debug().Msg("Створення головного вікна...")
	mainWindow := fyneApp.NewWindow("Каталог об'єктів")
	mainWindow.Resize(fyne.NewSize(1024, 768))
	log.Debug().Str("size", "1024x768").Msg("Головне вікно налаштовано")

	// Завантажуємо налаштування БД
	log.Info().Msg("Завантаження налаштувань БД...")
	dbCfg := config.LoadDBConfig(fyneApp.Preferences())
	dsn := dbCfg.ToDSN()
	log.Info().Str("host", dbCfg.Host).Str("port", dbCfg.Port).Str("user", dbCfg.User).Msg("Налаштування БД завантажено")

	// Ініціалізуємо БД
	log.Info().Msg("Підключення до бази даних...")
	db := database.InitDB(dsn)
	log.Info().Msg("БД підключена, запуск перевірки здоров'я...")
	healthCancel := database.StartHealthCheck(db)

	// Створюємо mock дані
	// mockData := data.NewMockData()

	// ВИБІР ПРОВАЙДЕРА
	log.Info().Msg("Ініціалізація провайдера даних...")
	dataProvider := data.NewDBDataProvider(db, dsn)
	log.Debug().Msg("Провайдер даних БД створено")

	log.Info().Msg("Створення структури додатку...")
	application := &Application{
		fyneApp:        fyneApp,
		mainWindow:     mainWindow,
		db:             db,
		dbHealthCancel: healthCancel,
		dataProvider:   dataProvider,
		// mockData:   mockData,
		isDarkTheme: isDark,
	}
	log.Debug().Msg("Структура додатку готова")

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
	a.alarmPanel = ui.NewAlarmPanelWidget(a.dataProvider)
	log.Debug().Msg("AlarmPanel створена")

	log.Debug().Msg("Створення ObjectListPanel...")
	a.objectList = ui.NewObjectListPanel(a.dataProvider)
	log.Debug().Msg("ObjectListPanel створена")

	log.Debug().Msg("Створення WorkAreaPanel...")
	a.workArea = ui.NewWorkAreaPanel(a.dataProvider, a.mainWindow)
	log.Debug().Msg("WorkAreaPanel створена")

	log.Debug().Msg("Створення EventLogPanel...")
	a.eventLog = ui.NewEventLogPanel(a.dataProvider)
	log.Debug().Msg("EventLogPanel створена")

	log.Debug().Msg("Налаштування callbacks...")

	// Налаштовуємо callbacks
	a.objectList.OnObjectSelected = func(object models.Object) {
		log.Debug().Int("objectID", object.ID).Str("objectName", object.Name).Msg("Об'єкт вибраний з списку")
		// Зберігаємо поточний об'єкт для заголовка та контекстних фільтрів
		a.currentObject = &object
		a.updateWindowTitle()
		a.workArea.SetObject(object)
		// Синхронізуємо глобальний журнал подій з вибраним об'єктом
		if a.eventLog != nil {
			a.eventLog.SetCurrentObject(&object)
		}
		// Для адміністратора при виборі об'єкта завжди показуємо його картку
		if a.rightTabs != nil {
			a.rightTabs.SelectIndex(0)
		}
	}

	a.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога вибрана (одинарний клік)")
		obj := a.dataProvider.GetObjectByID(fmt.Sprintf("%d", alarm.ObjectID))
		if obj != nil {
			// Оновлюємо контекст вибраного об'єкта, але залишаємо вкладку "Тривоги" відкритою.
			a.currentObject = obj
			a.updateWindowTitle()
			a.workArea.SetObject(*obj)
			if a.eventLog != nil {
				a.eventLog.SetCurrentObject(obj)
			}
		}
	}

	a.alarmPanel.OnAlarmActivated = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога активована (подвійний клік)")
		// Подвійний клік: відкриваємо вкладку деталей для вже вибраного об'єкта.
		if a.rightTabs != nil {
			a.rightTabs.SelectIndex(0)
		}
	}

	a.eventLog.OnEventSelected = func(event models.Event) {
		log.Debug().Int("eventID", event.ID).Int("objectID", event.ObjectID).Msg("Подія вибрана")
		obj := a.dataProvider.GetObjectByID(fmt.Sprintf("%d", event.ObjectID))
		if obj != nil {
			// Оновлюємо контекст вибраного об'єкта
			a.currentObject = obj
			a.updateWindowTitle()
			a.workArea.SetObject(*obj)
			if a.eventLog != nil {
				a.eventLog.SetCurrentObject(obj)
			}
			if a.rightTabs != nil {
				a.rightTabs.SelectIndex(0)
			}
		}
	}

	a.alarmPanel.OnProcessAlarm = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Msg("Початок обробки тривоги...")
		dialogs.ShowProcessAlarmDialog(a.mainWindow, alarm, func(result dialogs.ProcessAlarmResult) {
			log.Info().Int("alarmID", alarm.ID).Str("action", result.Action).Str("note", result.Note).Msg("Тривога оброблена")
			a.dataProvider.ProcessAlarm(fmt.Sprintf("%d", alarm.ID), "Диспетчер", result.Note)
			a.alarmPanel.Refresh()
			dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Тривогу оброблено: "+result.Action)
		})
	}

	log.Debug().Msg("Callbacks налаштовані")

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
		// Оновлюємо панелі, щоб застосувати нові кольори
		a.objectList.Refresh()
		a.eventLog.Refresh()
	}
	updateThemeButton()

	// Кнопка налаштування кольорів подій/об'єктів
	colorsBtn := widget.NewButtonWithIcon("Кольори", fyneTheme.ColorPaletteIcon(), func() {
		log.Debug().Bool("darkTheme", a.isDarkTheme).Msg("Відкриття діалогу кольорів...")
		dialogs.ShowColorPaletteDialog(a.mainWindow, a.isDarkTheme, func() {
			// Після зміни кольорів оновлюємо всі панелі, які їх використовують
			if a.alarmPanel != nil {
				a.alarmPanel.Refresh()
			}
			if a.eventLog != nil {
				a.eventLog.Refresh()
			}
			if a.objectList != nil {
				a.objectList.Refresh()
			}
			if a.workArea != nil && a.workArea.EventsList != nil {
				a.workArea.EventsList.Refresh()
			}
		})
	})

	// Кнопка налаштувань
	settingsBtn := widget.NewButtonWithIcon("Налаштування", fyneTheme.SettingsIcon(), func() {
		log.Debug().Msg("Відкриття діалогу налаштувань...")
		dialogs.ShowSettingsDialog(a.mainWindow, a.fyneApp.Preferences(), func(dbCfg config.DBConfig, uiCfg config.UIConfig) {
			log.Info().Str("host", dbCfg.Host).Msg("Параметри в діалозі налаштувань змінено")
			a.Reconnect(dbCfg)
			a.RefreshUI(uiCfg)
		})
	})

	title := widget.NewLabel("Каталог об'єктів")
	toolbar := container.NewHBox(title, layout.NewSpacer(), themeBtn, colorsBtn, settingsBtn)

	// Таби: показуємо найважливіше першим (тривоги), додаємо лічильники.
	detailsTab := container.NewTabItem("КАРТКА ОБ'ЄКТА", a.workArea.Container)
	eventsTab := container.NewTabItem("ЖУРНАЛ ПОДІЙ", a.eventLog.Container)
	alarmsTab := container.NewTabItem("ТРИВОГИ", a.alarmPanel.Container)
	rightTabs := container.NewAppTabs(detailsTab, eventsTab, alarmsTab)
	// Зберігаємо посилання на вкладки для подальшого керування
	a.rightTabs = rightTabs
	// Для адміністративного сценарію за замовчуванням показуємо картку об'єкта
	rightTabs.Select(detailsTab)

	// Хелпер для badge-оновлення табів.
	lastAlarmsCount := 0
	lastFireCount := 0
	lastEventsCount := 0
	updateTabBadges := func(alarmsCount int, fireCount int, eventsCount int) {
		if alarmsCount >= 0 {
			lastAlarmsCount = alarmsCount
			lastFireCount = fireCount
		}
		if eventsCount >= 0 {
			lastEventsCount = eventsCount
		}

		// Алгоритм простий: показуємо тільки те, що реально важливо користувачу.
		alarmTitle := "АКТИВНІ ТРИВОГИ"
		if lastAlarmsCount > 0 {
			alarmTitle = fmt.Sprintf("АКТИВНІ ТРИВОГИ (%d)", lastAlarmsCount)
			if lastFireCount > 0 {
				alarmTitle = fmt.Sprintf("АКТИВНІ ТРИВОГИ (%d, ПОЖЕЖА: %d)", lastAlarmsCount, lastFireCount)
			}
		}
		alarmsTab.Text = alarmTitle

		eventsTitle := "ЖУРНАЛ ПОДІЙ"
		if lastEventsCount > 0 {
			eventsTitle = fmt.Sprintf("ЖУРНАЛ ПОДІЙ (%d)", lastEventsCount)
		}
		eventsTab.Text = eventsTitle

		rightTabs.Refresh()
		// Оновлюємо заголовок вікна з урахуванням кількості тривог
		a.currentAlarmsTotal = lastAlarmsCount
		a.updateWindowTitle()
	}

	// Синхронізуємо лічильники з панелями (викличеться після їх Refresh()).
	if a.alarmPanel != nil {
		a.alarmPanel.OnCountsChanged = func(total int, fire int) {
			// eventsCount тут не знаємо — не чіпаємо.
			updateTabBadges(total, fire, -1)
		}
		a.alarmPanel.OnNewCriticalAlarm = func(alarm models.Alarm) {
			// Для адміністратора не перемикаємо вкладку автоматично,
			// а лише м'яко сповіщаємо про нову тривогу.
			ui.ShowToast(a.mainWindow, fmt.Sprintf("Нова тривога: №%d %s", alarm.ObjectID, alarm.GetTypeDisplay()))
		}
	}
	if a.eventLog != nil {
		a.eventLog.OnCountChanged = func(count int) {
			updateTabBadges(-1, 0, count)
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

	a.statusLabel = widget.NewLabel("БД : підключено")
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
		a.fyneApp.Preferences().SetFloat(prefKeyObjectListSplitOffset, rootSplit.Offset)
		a.mainWindow.Close()
	})

	a.mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if themeBtn.OnTapped != nil {
			themeBtn.OnTapped()
		}
	})
	a.mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		if a.objectList != nil && a.objectList.SearchEntry != nil {
			a.mainWindow.Canvas().Focus(a.objectList.SearchEntry)
		}
	})
	a.mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key1, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		rightTabs.SelectIndex(0)
	})
	a.mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key2, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		rightTabs.SelectIndex(1)
	})
	a.mainWindow.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.Key3, Modifier: fyne.KeyModifierControl}, func(shortcut fyne.Shortcut) {
		rightTabs.SelectIndex(2)
	})
}

// startGettingEvents запускає симуляцію подій
func (a *Application) startGettingEvents() {
	go func() {
		secTicker := time.NewTicker(2 * time.Second) // Трохи повільніше
		defer secTicker.Stop()

		minTicker := time.NewTicker(60 * time.Second)
		defer minTicker.Stop()

		for {
			select {
			case <-secTicker.C:
				// Симуляція тільки якщо використовуємо мок-дані або для візуального ефекту
				// В реальному проекті тут краще робити фонове оновлення через провайдера
				// if a.mockData != nil && rand.Intn(3) == 0 {
				// 	a.mockData.SimulateRandomEvent()
				// 	a.mockData.SimulateNewAlarm()
				// }

				fyne.Do(func() {
					if a.alarmPanel != nil {
						a.alarmPanel.Refresh()
					}
					if a.eventLog != nil {
						a.eventLog.Refresh()
					}
					if a.objectList != nil {
						a.objectList.Refresh()
					}
				})

			case <-minTicker.C:
				// if a.mockData != nil {
				// 	changedObj := a.mockData.SimulateObjectChange()
				fyne.Do(func() {
					a.objectList.Refresh()
					// if a.workArea.CurrentObject != nil && a.workArea.CurrentObject.ID == changedObj.ID {
					// 	a.workArea.SetObject(*changedObj)
					// }
				})
				// }
			}
		}
	}()
}

// Run запускає додаток
func (a *Application) Run() {
	log.Info().Msg("Запуск основного цикла додатку (UI loop)...")
	if a.db != nil {
		defer func() {
			log.Debug().Msg("Закриття з'єднання з БД...")
			if a.dbHealthCancel != nil {
				a.dbHealthCancel()
				a.dbHealthCancel = nil
			}
			a.db.Close()
			log.Debug().Msg("✓ З'єднання з БД закрито")
		}()
	}

	// Робимо рядок пошуку активним (виділеним) одразу після старту.
	if a.objectList != nil && a.objectList.SearchEntry != nil {
		a.mainWindow.Canvas().Focus(a.objectList.SearchEntry)
	}
	a.mainWindow.ShowAndRun()
	log.Info().Msg("Основний цикл завершено")
}

// Reconnect перепідключає базу даних та оновлює провайдери
func (a *Application) Reconnect(cfg config.DBConfig) {
	dsn := cfg.ToDSN()
	log.Warn().Str("dsn", dsn).Msg("🔄 Перепідключення до бази даних...")
	if a.statusLabel != nil {
		a.statusLabel.SetText("БД : перепідключення...")
	}
	log.Debug().Msg("Ініціалізація нового з'єднання з БД...")
	newDB := database.InitDB(dsn)
	if err := newDB.Ping(); err != nil {
		log.Error().Err(err).Msg("❌ Помилка перевірки з'єднання з новою БД")
		if a.statusLabel != nil {
			a.statusLabel.SetText("БД : помилка підключення")
		}
		dialogs.ShowErrorDialog(a.mainWindow, "Помилка підключення", err)
		return
	}
	log.Debug().Msg("✓ Нове з'єднання з БД успішне")

	// Закриваємо стару базу
	if a.db != nil {
		log.Debug().Msg("Закриття старого з'єднання з БД...")
		if a.dbHealthCancel != nil {
			a.dbHealthCancel()
			a.dbHealthCancel = nil
		}
		a.db.Close()
		log.Debug().Msg("✓ Старе з'єднання закрито")
	}

	a.db = newDB
	a.dataProvider = data.NewDBDataProvider(newDB, dsn)
	a.dbHealthCancel = database.StartHealthCheck(newDB)
	log.Debug().Msg("Провайдер даних оновлено")

	// Оновлюємо посилання в панелях
	log.Debug().Msg("Оновлення посилань на БД у панелях...")
	a.alarmPanel.Data = a.dataProvider
	a.objectList.Data = a.dataProvider
	a.workArea.Data = a.dataProvider
	a.eventLog.Data = a.dataProvider
	log.Debug().Msg("✓ Посилання оновлено")

	// Перезавантажуємо дані
	log.Debug().Msg("Перезавантаження даних у всіх панелях...")
	a.alarmPanel.Refresh()
	a.objectList.Refresh()
	a.eventLog.Refresh()
	log.Debug().Msg("✓ Дані перезавантажено")

	log.Info().Msg("✅ Перепідключення до БД завершено успішно")
	if a.statusLabel != nil {
		a.statusLabel.SetText("БД : підключено")
	}
	dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Підключення до бази даних оновлено")
}

// RefreshUI оновлює інтерфейс (тему, шрифти)
func (a *Application) RefreshUI(cfg config.UIConfig) {
	log.Info().Float32("fontSize", cfg.FontSize).Msg("🎨 Оновлення параметрів інтерфейсу...")
	log.Debug().Float32("fontSizeAlarms", cfg.FontSizeAlarms).Float32("fontSizeObjects", cfg.FontSizeObjects).Float32("fontSizeEvents", cfg.FontSizeEvents).Msg("Нові розміри шрифтів")

	a.setTheme(a.isDarkTheme)

	// Оновлюємо панелі
	log.Debug().Msg("Оновлення AlarmPanel...")
	a.alarmPanel.OnThemeChanged(cfg.FontSizeAlarms)
	a.alarmPanel.Refresh()

	log.Debug().Msg("Оновлення ObjectListPanel...")
	a.objectList.OnThemeChanged(cfg.FontSizeObjects)
	a.objectList.Refresh()

	log.Debug().Msg("Оновлення WorkAreaPanel...")
	a.workArea.OnThemeChanged(cfg.FontSize)

	log.Debug().Msg("Оновлення EventLogPanel...")
	a.eventLog.OnThemeChanged(cfg.FontSizeEvents)
	a.eventLog.Refresh()

	log.Info().Msg("✅ Параметри інтерфейсу оновлено")
}
