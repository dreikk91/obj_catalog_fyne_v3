package main

import (
	"fmt"
	"math/rand"
	"runtime/debug"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

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
	fyneApp    fyne.App
	mainWindow fyne.Window
	db         *sqlx.DB

	// Сховище даних (інтерфейс)
	dataProvider data.DataProvider
	// Пряме посилання на MockData ТІЛЬКИ для симуляції
	mockData *data.MockData

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

	// Додаємо базове відновлення після паніки
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("КРИТИЧНА ПОМИЛКА (Panic)")
		}
	}()

	// Створюємо додаток
	application := NewApplication()
	application.Run()
}

// NewApplication створює новий екземпляр додатку
func NewApplication() *Application {
	// Ініціалізація Fyne
	fyneApp := app.New()

	// Створюємо головне вікно
	mainWindow := fyneApp.NewWindow("АРМ Пожежної Безпеки v1.0")
	mainWindow.Resize(fyne.NewSize(1024, 768))

	// Рядок підключення до Firebird
	dsn := "SYSDBA:masterkey@localhost:3050/C:/MOST.PM/BASE/MOST5.FDB?charset=WIN1251&auth_plugin_name=Srp"

	// Ініціалізуємо БД (це швидко, просто створює структуру)
	db := database.InitDB(dsn)
	database.StartHealthCheck(db)

	// Створюємо mock дані
	mockData := data.NewMockData()

	// ВИБІР ПРОВАЙДЕРА
	dataProvider := data.NewDBDataProvider(db, dsn)

	application := &Application{
		fyneApp:      fyneApp,
		mainWindow:   mainWindow,
		db:           db,
		dataProvider: dataProvider,
		mockData:     mockData,
		isDarkTheme:  true,
	}

	// Встановлюємо тему
	application.setTheme(true)

	// Будуємо інтерфейс (це тепер швидко, бо все асинхронно)
	application.buildUI()

	// Показуємо вікно ЯКНАЙШВИДШЕ
	// А дані будуть підтягуватись у фоні (вже запущено в конструкторах панелей)

	// Запускаємо симуляцію подій / фонове оновлення
	application.startEventSimulation()

	return application
}

// setTheme встановлює тему (темну або світлу)
func (a *Application) setTheme(dark bool) {
	a.isDarkTheme = dark
	if dark {
		a.fyneApp.Settings().SetTheme(theme.NewDarkTheme())
	} else {
		a.fyneApp.Settings().SetTheme(theme.NewLightTheme())
	}
}

// buildUI будує головний інтерфейс
func (a *Application) buildUI() {
	// Створюємо UI компоненти
	a.alarmPanel = ui.NewAlarmPanelWidget(a.dataProvider)
	a.objectList = ui.NewObjectListPanel(a.dataProvider)
	a.workArea = ui.NewWorkAreaPanel(a.dataProvider, a.mainWindow)
	a.eventLog = ui.NewEventLogPanel(a.dataProvider)

	// Налаштовуємо callbacks
	a.objectList.OnObjectSelected = func(object models.Object) {
		a.workArea.SetObject(object)
	}

	a.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		obj := a.dataProvider.GetObjectByID(fmt.Sprintf("%d", alarm.ObjectID))
		if obj != nil {
			a.workArea.SetObject(*obj)
		}
	}

	a.alarmPanel.OnProcessAlarm = func(alarm models.Alarm) {
		dialogs.ShowProcessAlarmDialog(a.mainWindow, alarm, func(result dialogs.ProcessAlarmResult) {
			a.dataProvider.ProcessAlarm(fmt.Sprintf("%d", alarm.ID), "Диспетчер", result.Note)
			a.alarmPanel.Refresh()
			dialogs.ShowInfoDialog(a.mainWindow, "Успішно", "Тривогу оброблено: "+result.Action)
		})
	}

	// Кнопка перемикання теми
	themeBtn := widget.NewButton("🌙 Темна тема", nil)
	themeBtn.OnTapped = func() {
		a.isDarkTheme = !a.isDarkTheme
		a.setTheme(a.isDarkTheme)
		if a.isDarkTheme {
			themeBtn.SetText("🌙 Темна тема")
		} else {
			themeBtn.SetText("☀️ Світла тема")
		}
		// Оновлюємо панелі, щоб застосувати нові кольори
		a.objectList.Refresh()
		a.eventLog.Refresh()
	}

	toolbar := container.NewHBox(
		widget.NewLabel("АРМ Пожежної Безпеки"),
		widget.NewSeparator(),
		themeBtn,
	)

	// Layout
	centerSplit := container.NewHSplit(a.objectList.Container, a.workArea.Container)
	centerSplit.SetOffset(0.45)

	mainSplit := container.NewVSplit(centerSplit, a.eventLog.Container)
	mainSplit.SetOffset(0.75)

	rootSplit := container.NewVSplit(a.alarmPanel.Container, mainSplit)
	rootSplit.SetOffset(0.2)

	finalLayout := container.NewBorder(
		container.NewVBox(toolbar, widget.NewSeparator()),
		nil, nil, nil,
		rootSplit,
	)

	a.mainWindow.SetContent(finalLayout)
}

// startEventSimulation запускає симуляцію подій
func (a *Application) startEventSimulation() {
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
				if a.mockData != nil && rand.Intn(3) == 0 {
					a.mockData.SimulateRandomEvent()
					a.mockData.SimulateNewAlarm()
				}

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
				if a.mockData != nil {
					changedObj := a.mockData.SimulateObjectChange()
					fyne.Do(func() {
						a.objectList.Refresh()
						if a.workArea.CurrentObject != nil && a.workArea.CurrentObject.ID == changedObj.ID {
							a.workArea.SetObject(*changedObj)
						}
					})
				}
			}
		}
	}()
}

// Run запускає додаток
func (a *Application) Run() {
	if a.db != nil {
		defer a.db.Close()
	}
	a.mainWindow.ShowAndRun()
}
