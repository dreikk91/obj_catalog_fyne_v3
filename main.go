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
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/theme"
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

	// Сховище даних (інтерфейс)
	dataProvider data.DataProvider
	// Пряме посилання на MockData ТІЛЬКИ для симуляції
	// mockData *data.MockData

	// UI компоненти (нові структури)
	alarmPanel *ui.AlarmPanelWidget
	objectList *ui.ObjectListPanel
	workArea   *ui.WorkAreaPanel
	eventLog   *ui.EventLogPanel

	// Поточна тема
	isDarkTheme bool
}

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
		// mockData:     mockData,
		isDarkTheme: true,
	}
	log.Debug().Msg("Структура додатку готова")

	// Встановлюємо тему
	log.Debug().Msg("Встановлення теми...")
	application.setTheme(true)
	log.Debug().Bool("darkTheme", true).Msg("Тема встановлена")

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
	uiCfg := config.LoadUIConfig(a.fyneApp.Preferences())
	if dark {
		log.Debug().Msg("Застосування темної теми...")
		a.fyneApp.Settings().SetTheme(theme.NewDarkTheme(uiCfg.FontSize))
	} else {
		log.Debug().Msg("Застосування світлої теми...")
		a.fyneApp.Settings().SetTheme(theme.NewLightTheme(uiCfg.FontSize))
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
		a.workArea.SetObject(object)
	}

	a.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		log.Debug().Int("alarmID", alarm.ID).Int("objectID", alarm.ObjectID).Msg("Тривога вибрана")
		obj := a.dataProvider.GetObjectByID(fmt.Sprintf("%d", alarm.ObjectID))
		if obj != nil {
			a.workArea.SetObject(*obj)
		}
	}

	a.eventLog.OnEventSelected = func(event models.Event) {
		log.Debug().Int("eventID", event.ID).Int("objectID", event.ObjectID).Msg("Подія вибрана")
		obj := a.dataProvider.GetObjectByID(fmt.Sprintf("%d", event.ObjectID))
		if obj != nil {
			a.workArea.SetObject(*obj)
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
	themeBtn := widget.NewButton("Темна тема", nil)
	themeBtn.OnTapped = func() {
		a.isDarkTheme = !a.isDarkTheme
		log.Debug().Bool("darkTheme", a.isDarkTheme).Msg("Перемикання теми...")
		a.setTheme(a.isDarkTheme)
		if a.isDarkTheme {
			themeBtn.SetText("Темна тема")
		} else {
			themeBtn.SetText("Світла тема")
		}
		// Оновлюємо панелі, щоб застосувати нові кольори
		a.objectList.Refresh()
		a.eventLog.Refresh()
	}

	// Кнопка налаштувань
	settingsBtn := widget.NewButton("Налаштування", func() {
		log.Debug().Msg("Відкриття діалогу налаштувань...")
		dialogs.ShowSettingsDialog(a.mainWindow, a.fyneApp.Preferences(), func(dbCfg config.DBConfig, uiCfg config.UIConfig) {
			log.Info().Str("host", dbCfg.Host).Msg("Параметри в діалозі налаштувань змінено")
			a.Reconnect(dbCfg)
			a.RefreshUI(uiCfg)
		})
	})

	toolbar := container.NewHBox(
		widget.NewLabel("Каталог об'єктів"),
		widget.NewSeparator(),
		themeBtn,
		settingsBtn,
	)

	rightTabs := container.NewAppTabs(
		container.NewTabItem("ДЕТАЛІ", a.workArea.Container),
		container.NewTabItem("ЖУРНАЛ ПОДІЙ", a.eventLog.Container),
		container.NewTabItem("АКТИВНІ ТРИВОГИ", a.alarmPanel.Container),
	)

	log.Debug().Msg("Компонування макета...")

	// Layout: universal HSplit with right-side tabs (better for 1024x768 and 1920x1080)
	rootSplit := container.NewHSplit(a.objectList.Container, rightTabs)
	rootSplit.SetOffset(0.35)

	finalLayout := container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		nil, nil, nil,
		rootSplit,
	)

	a.mainWindow.SetContent(finalLayout)
	log.Debug().Msg("UI побудований та встановлений на вікно")
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
	a.mainWindow.ShowAndRun()
	log.Info().Msg("Основний цикл завершено")
}

// Reconnect перепідключає базу даних та оновлює провайдери
func (a *Application) Reconnect(cfg config.DBConfig) {
	dsn := cfg.ToDSN()
	log.Warn().Str("dsn", dsn).Msg("🔄 Перепідключення до бази даних...")

	log.Debug().Msg("Ініціалізація нового з'єднання з БД...")
	newDB := database.InitDB(dsn)
	if err := newDB.Ping(); err != nil {
		log.Error().Err(err).Msg("❌ Помилка перевірки з'єднання з новою БД")
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

