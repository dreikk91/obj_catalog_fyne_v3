//go:build qt

package qtapp

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	qt "github.com/mappu/miqt/qt6"
	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/dataruntime"
	"obj_catalog_fyne_v3/pkg/eventbus"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/omnicell"
	"obj_catalog_fyne_v3/pkg/qtui"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type Application struct {
	ui                     *qtui.App
	runtime                *dataruntime.Runtime
	uiData                 *backend.FrontendUIDataProvider
	workVM                 *viewmodels.WorkAreaViewModel
	currentObject          *models.Object
	currentObjectZones     int
	currentObjectContacts  int
	currentObjectsCount    int
	currentAlarmsCount     int
	currentEventsCount     int
	selectionSeq           int
	objectsRefreshSeq      int
	alarmsRefreshSeq       int
	eventsRefreshSeq       int
	eventBus               *eventbus.Bus
	mainThreadQueue        chan func()
	refreshLoopCancel      context.CancelFunc
	refreshCoalesceMu      sync.Mutex
	pendingRefresh         eventbus.DataRefreshEvent
	refreshCoalescePending bool
	phoneDialer            contracts.PhoneDialer
}

type objectDetailsResult struct {
	seq        int
	object     models.Object
	zones      []models.Zone
	contacts   []models.Contact
	events     []models.Event
	statusText string
}

func NewApplication() *Application {
	ui := qtui.NewApp()
	preferences := ui.Preferences()
	if qtPrefs, ok := preferences.(*config.QtPreferences); ok {
		if filename := qtPrefs.FileName(); strings.TrimSpace(filename) != "" {
			log.Info().Str("settingsFile", filename).Msg("Qt UI використовує файл налаштувань")
		}
	}
	app := &Application{
		ui:              ui,
		workVM:          viewmodels.NewWorkAreaViewModel(),
		eventBus:        eventbus.NewBus(),
		mainThreadQueue: make(chan func(), 1000),
	}

	dispatcherTimer := qt.NewQTimer()
	dispatcherTimer.SetInterval(20)
	dispatcherTimer.OnTimeout(func() {
		for {
			select {
			case f := <-app.mainThreadQueue:
				if f != nil {
					f()
				}
			default:
				return
			}
		}
	})
	dispatcherTimer.Start2()

	app.ui.OnSettingsSaved = app.applySettings
	app.ui.OnRefreshRequested = app.refreshData
	app.ui.OnDiagnosticsRequested = app.showDiagnostics
	app.ui.OnEditObject = app.editCurrentObject
	app.ui.OnSIMManagement = app.showCurrentObjectSIM
	app.ui.OnSendSIMSMS = app.sendSIMSMS
	app.ui.OnDialPhone = app.dialPhone
	app.ui.OnProcessAlarms = app.processAlarms
	app.ui.OnPickAlarms = app.pickAlarms
	app.ui.OnRunOnMainThread = app.runOnMainThread
	app.ui.OnAlarmSelected = app.handleAlarmSelected
	app.ui.OnEventSelected = app.handleEventSelected
	app.registerEventBusHandlers()
	app.initializeRuntime(preferences)
	return app
}

func (a *Application) initializeRuntime(preferences config.Preferences) {
	dbCfg := config.LoadDBConfig(preferences)
	a.applySettings(dbCfg, config.LoadUIConfig(preferences))
}

func (a *Application) applySettings(dbCfg config.DBConfig, uiCfg config.UIConfig) {
	if a == nil || a.ui == nil {
		return
	}
	if a.runtime != nil {
		a.runtime.Close()
		a.runtime = nil
	}
	a.currentObject = nil
	a.phoneDialer = buildAMIDialer(a.ui.Preferences())

	store := preferencesConfigStore{preferences: a.ui.Preferences()}

	runtime, err := dataruntime.New(dbCfg, store, false)
	if err != nil {
		log.Error().Err(err).Msg("Qt UI: не вдалося ініціалізувати джерела даних")
		a.ui.SetStatus("Джерела даних: помилка підключення")
		return
	}
	a.runtime = runtime
	if runtime.Provider == nil {
		a.ui.SetStatus("Джерела даних: провайдер недоступний")
		return
	}

	frontend := backend.NewFrontendAdapter(runtime.Provider)
	uiData := backend.NewFrontendUIDataProvider(frontend, runtime.Provider)
	a.uiData = uiData
	a.ui.SetDataProvider(uiData)
	if admin, ok := backend.AsAdminProvider(runtime.Provider); ok {
		a.ui.SetAdminProvider(admin)
	}
	a.ui.SetStatus(backendStatusText(runtime))
	a.ui.SetObjectSelectedHandler(a.applyObjectContext)
	a.refreshData()
	a.startGettingEvents()
}

func (a *Application) refreshData() {
	defer traceQtOperation("refreshData")()
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	a.refreshObjects()
	a.refreshAlarms()
	a.refreshEvents()
	if a.runtime != nil {
		a.ui.SetStatus(backendStatusText(a.runtime))
	}
}

func (a *Application) refreshObjects() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	a.objectsRefreshSeq++
	seq := a.objectsRefreshSeq
	go func() {
		defer traceQtOperation("refreshObjects")()
		objects := provider.GetObjects()
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider || seq != a.objectsRefreshSeq {
				return
			}
			a.currentObjectsCount = len(objects)
			a.ui.SetObjects(objects)
		})
	}()
}

func (a *Application) refreshAlarms() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	a.alarmsRefreshSeq++
	seq := a.alarmsRefreshSeq
	go func() {
		defer traceQtOperation("refreshAlarms")()
		alarms := provider.GetAlarms()
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider || seq != a.alarmsRefreshSeq {
				return
			}
			a.currentAlarmsCount = len(alarms)
			a.ui.SetAlarms(alarms)
		})
	}()
}

func (a *Application) refreshEvents() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	uiCfg := config.LoadUIConfig(a.ui.Preferences())
	a.eventsRefreshSeq++
	seq := a.eventsRefreshSeq
	go func() {
		defer traceQtOperation("refreshEvents")()
		events := provider.GetEvents()
		if uiCfg.EventLogLimit > 0 && len(events) > uiCfg.EventLogLimit {
			events = events[:uiCfg.EventLogLimit]
		}
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider || seq != a.eventsRefreshSeq {
				return
			}
			a.currentEventsCount = len(events)
			a.ui.SetEvents(events)
		})
	}()
}

func (a *Application) applyObjectContext(object models.Object) {
	defer traceQtOperation("selectObject")()
	if a == nil || a.uiData == nil || a.workVM == nil {
		return
	}

	a.ui.SelectObject(object.ID)

	a.selectionSeq++
	seq := a.selectionSeq
	a.currentObject = &object
	a.currentObjectZones = 0
	a.currentObjectContacts = 0
	a.ui.SetObjectLoading(object)
	a.ui.SetStatus("Завантаження об'єкта: " + strconv.Itoa(object.ID) + " | " + strings.TrimSpace(object.Name))

	provider := a.uiData
	workVM := a.workVM
	go func() {
		defer traceQtOperation("loadObjectDetails")()
		details := workVM.LoadObjectBaseDetails(provider, object.ID)
		fullObject := object
		if details.FullObject != nil {
			fullObject = *details.FullObject
		}
		result := objectDetailsResult{
			seq:        seq,
			object:     fullObject,
			zones:      details.Zones,
			contacts:   details.Contacts,
			events:     nil,
			statusText: "Обрано об'єкт: " + strconv.Itoa(object.ID) + " | " + strings.TrimSpace(object.Name),
		}
		a.runOnMainThread(func() {
			a.applyObjectDetailsResult(result)
		})
	}()
}

func (a *Application) handleAlarmSelected(alarm models.Alarm) {
	if a == nil {
		return
	}
	a.reselectObject(alarm.ObjectID)
}

func (a *Application) handleEventSelected(event models.Event) {
	if a == nil {
		return
	}
	a.reselectObject(event.ObjectID)
}

func (a *Application) applyObjectDetailsResult(result objectDetailsResult) {
	defer traceQtOperation("applyObjectDetails")()
	if a == nil || a.ui == nil || result.seq != a.selectionSeq {
		return
	}
	a.currentObject = &result.object
	a.currentObjectZones = len(result.zones)
	a.currentObjectContacts = len(result.contacts)
	a.ui.SetObjectDetails(result.object, result.zones, result.contacts, result.events)
	a.ui.SetStatus(result.statusText)
}

func (a *Application) editCurrentObject() {
	if a == nil || a.ui == nil {
		return
	}
	if a.currentObject == nil {
		a.ui.ShowInfo("Редагування об'єкта", "Оберіть об'єкт у списку.")
		return
	}
	admin, ok := a.adminProvider()
	if !ok {
		a.ui.ShowInfo("Редагування об'єкта", "Поточне джерело даних не підтримує редагування об'єктів.")
		return
	}
	objn := int64(a.currentObject.ID)
	card, err := admin.GetObjectCard(objn)
	if err != nil {
		a.ui.ShowError("Редагування об'єкта", "Не вдалося завантажити картку об'єкта: "+err.Error())
		return
	}
	updated, accepted := a.ui.EditObjectCard(card)
	if !accepted {
		return
	}
	if err := admin.UpdateObject(updated); err != nil {
		a.ui.ShowError("Редагування об'єкта", "Не вдалося зберегти картку об'єкта: "+err.Error())
		return
	}
	a.refreshData()
	a.reselectObject(int(updated.ObjN))
	a.ui.SetStatus("Картку об'єкта оновлено: " + strconv.FormatInt(updated.ObjN, 10))
}

func (a *Application) showCurrentObjectSIM() {
	if a == nil || a.ui == nil {
		return
	}
	if a.currentObject == nil {
		a.ui.ShowInfo("SIM-карти", "Оберіть об'єкт у списку.")
		return
	}
	admin, ok := a.adminProvider()
	if !ok {
		a.ui.ShowSIMManagement(*a.currentObject, "Поточне джерело даних не підтримує перевірку використання SIM-номерів.")
		return
	}
	a.ui.ShowSIMManagement(*a.currentObject, qtui.ObjectSIMUsageText(admin, *a.currentObject))
}

func (a *Application) sendSIMSMS(object models.Object, phone string) {
	if a == nil || a.ui == nil {
		return
	}
	cfg := config.LoadOmnicellConfig(a.ui.Preferences())
	if !cfg.Ready() {
		a.ui.ShowInfo("Omnicell SMS", "Omnicell SMS не налаштовано. Заповніть endpoint, login, password і source в налаштуваннях.")
		return
	}
	commands, accepted := a.ui.ShowSIMSMS(object, phone, cfg)
	if !accepted {
		return
	}
	client := omnicell.NewClient(cfg)
	a.ui.SetStatus("Надсилання SMS через Omnicell на " + strings.TrimSpace(phone))
	go func() {
		results := make([]string, 0, len(commands))
		var sendErr error
		for _, command := range commands {
			resp, err := client.SendSMS(context.Background(), omnicell.SendRequest{
				Phone: phone,
				Text:  command.Text,
			})
			if err != nil {
				sendErr = fmt.Errorf("%s: %w", command.Title, err)
				break
			}
			line := fmt.Sprintf("%s: HTTP %d", command.Title, resp.StatusCode)
			if strings.TrimSpace(resp.Body) != "" {
				line += " | " + strings.TrimSpace(resp.Body)
			}
			results = append(results, line)
		}
		a.runOnMainThread(func() {
			if sendErr != nil {
				a.ui.ShowError("Omnicell SMS", "Не вдалося надіслати SMS: "+sendErr.Error())
				a.ui.SetStatus("Omnicell SMS: помилка")
				return
			}
			msg := fmt.Sprintf("SMS надіслано на %s.\n\n%s", phone, strings.Join(results, "\n"))
			a.ui.ShowInfo("Omnicell SMS", msg)
			a.ui.SetStatus("Omnicell SMS надіслано на " + strings.TrimSpace(phone))
		})
	}()
}

func (a *Application) processAlarms(alarms []models.Alarm) {
	if a == nil || a.ui == nil || len(alarms) == 0 {
		return
	}
	if a.uiData == nil {
		a.ui.ShowInfo("Відпрацювання тривоги", "Джерела даних ще не підключені.")
		return
	}

	ctx := context.Background()
	options := a.alarmProcessingOptions(ctx, alarms)
	alarm := alarms[0]
	input, accepted := a.ui.ProcessAlarmsDialog(alarms, options)
	if !accepted {
		return
	}

	const operator = "Диспетчер"
	var successCount = 0
	var errorMsgs []string

	for _, al := range alarms {
		var err error
		if len(options) > 0 {
			err = a.uiData.ProcessAlarmWithRequest(ctx, al, operator, contracts.AlarmProcessingRequest{
				CauseCode: input.CauseCode,
				Note:      input.Note,
			})
		} else {
			err = a.uiData.ProcessAlarm(strconv.Itoa(al.ID), operator, input.Note)
		}
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("№%s: %v", al.GetObjectNumberDisplay(), err))
		} else {
			successCount++
		}
	}

	a.refreshData()

	if len(errorMsgs) > 0 {
		a.ui.ShowError("Відпрацювання тривог", fmt.Sprintf("Успішно відпрацьовано: %d. Помилки:\n%s", successCount, strings.Join(errorMsgs, "\n")))
	} else {
		if len(alarms) > 1 {
			a.ui.SetStatus(fmt.Sprintf("Відпрацьовано групу з %d тривог", len(alarms)))
		} else {
			a.ui.SetStatus("Тривогу відпрацьовано: " + alarm.GetObjectNumberDisplay() + " | " + strings.TrimSpace(alarm.GetTypeDisplay()))
		}
	}
}

func (a *Application) alarmProcessingOptions(ctx context.Context, alarms []models.Alarm) []contracts.AlarmProcessingOption {
	if a == nil || a.uiData == nil || len(alarms) == 0 || !sameAlarmProcessingSource(alarms) {
		return nil
	}

	optionSets := make([][]contracts.AlarmProcessingOption, 0, len(alarms))
	for _, alarm := range alarms {
		options, err := a.uiData.GetAlarmProcessingOptions(ctx, alarm)
		if err != nil {
			return nil
		}
		optionSets = append(optionSets, options)
	}
	return intersectAlarmProcessingOptions(optionSets...)
}

func (a *Application) pickAlarms(alarms []models.Alarm) {
	if a == nil || a.ui == nil || len(alarms) == 0 {
		return
	}
	if a.uiData == nil {
		a.ui.ShowInfo("Тривоги", "Джерела даних ще не підключені.")
		return
	}

	const operator = "Диспетчер"
	var successCount = 0
	var errorMsgs []string

	for _, al := range alarms {
		if err := a.uiData.PickAlarm(context.Background(), al, operator); err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("№%s: %v", al.GetObjectNumberDisplay(), err))
		} else {
			successCount++
		}
	}

	a.refreshData()

	if len(errorMsgs) > 0 {
		a.ui.ShowError("Взяття тривог у роботу", fmt.Sprintf("Успішно взято: %d. Помилки:\n%s", successCount, strings.Join(errorMsgs, "\n")))
	} else {
		if len(alarms) > 1 {
			a.ui.SetStatus(fmt.Sprintf("Взято в роботу групу з %d тривог", len(alarms)))
		} else {
			a.ui.SetStatus("Тривогу взято в роботу: " + alarms[0].GetObjectNumberDisplay() + " | " + strings.TrimSpace(alarms[0].GetTypeDisplay()))
		}
	}
}

func (a *Application) adminProvider() (interface {
	GetObjectCard(objn int64) (contracts.AdminObjectCard, error)
	UpdateObject(card contracts.AdminObjectCard) error
	FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]viewmodels.SIMPhoneUsage, error)
}, bool) {
	if a == nil || a.runtime == nil || a.runtime.Provider == nil {
		return nil, false
	}
	admin, ok := backend.AsAdminProvider(a.runtime.Provider)
	if !ok {
		return nil, false
	}
	return adminSIMLookupAdapter{admin: admin}, true
}

func (a *Application) reselectObject(objectID int) {
	if a == nil || a.uiData == nil {
		return
	}
	for _, object := range a.uiData.GetObjects() {
		if object.ID == objectID {
			a.applyObjectContext(object)
			return
		}
	}
	a.currentObject = nil
}

type adminSIMLookupAdapter struct {
	admin contracts.AdminProvider
}

func (a adminSIMLookupAdapter) GetObjectCard(objn int64) (contracts.AdminObjectCard, error) {
	return a.admin.GetObjectCard(objn)
}

func (a adminSIMLookupAdapter) UpdateObject(card contracts.AdminObjectCard) error {
	return a.admin.UpdateObject(card)
}

func (a adminSIMLookupAdapter) FindObjectsBySIMPhone(phone string, excludeObjN *int64) ([]viewmodels.SIMPhoneUsage, error) {
	items, err := a.admin.FindObjectsBySIMPhone(phone, excludeObjN)
	if err != nil {
		return nil, err
	}
	return viewmodels.SIMPhoneUsagesFromContracts(items), nil
}

func (a *Application) Run() int {
	defer func() {
		if a.runtime != nil {
			a.runtime.Close()
		}
	}()
	return a.ui.Run()
}

type preferencesConfigStore struct {
	preferences config.Preferences
}

func (s preferencesConfigStore) LoadKyivstarConfig() config.KyivstarConfig {
	return config.LoadKyivstarConfig(s.preferences)
}

func (s preferencesConfigStore) SaveKyivstarConfig(cfg config.KyivstarConfig) {
	config.SaveKyivstarConfig(s.preferences, cfg)
}

func (s preferencesConfigStore) LoadVodafoneConfig() config.VodafoneConfig {
	return config.LoadVodafoneConfig(s.preferences)
}

func (s preferencesConfigStore) SaveVodafoneConfig(cfg config.VodafoneConfig) {
	config.SaveVodafoneConfig(s.preferences, cfg)
}

func backendStatusText(runtime *dataruntime.Runtime) string {
	if runtime == nil {
		return "Джерела даних: не ініціалізовано"
	}
	parts := make([]string, 0, 3)
	if runtime.FirebirdEnabled {
		parts = append(parts, "БД/МІСТ")
	}
	if runtime.PhoenixEnabled {
		parts = append(parts, "Phoenix")
	}
	if runtime.CASLEnabled {
		parts = append(parts, "CASL Cloud")
	}
	if len(parts) == 0 {
		return "Джерела даних: не налаштовано"
	}
	return "Джерела даних: " + strings.Join(parts, " | ") + " підключено"
}
