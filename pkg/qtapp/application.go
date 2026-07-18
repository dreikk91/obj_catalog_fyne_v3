//go:build qt

package qtapp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	qt "github.com/mappu/miqt/qt6"
	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/dataruntime"
	"obj_catalog_fyne_v3/pkg/eventbus"
	objexport "obj_catalog_fyne_v3/pkg/export"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/omnicell"
	"obj_catalog_fyne_v3/pkg/qtui"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const (
	mainThreadQueueSize         = 1000
	mainThreadQueueDrainLimit   = 128
	mainThreadEnqueueWarnAfter  = 5 * time.Second
	mainThreadEnqueueRetryDelay = 200 * time.Millisecond
	dataRefreshTimeout          = 30 * time.Second
)

type Application struct {
	ui                      *qtui.App
	runtime                 *dataruntime.Runtime
	uiData                  *backend.FrontendUIDataProvider
	workVM                  *viewmodels.WorkAreaViewModel
	currentObject           *models.Object
	currentObjectZones      int
	currentObjectContacts   int
	currentObjectsCount     int
	currentAlarmsCount      int
	currentEventsCount      int
	selectionSeq            int
	objectsRefreshSeq       int
	alarmsRefreshSeq        int
	eventsRefreshSeq        int
	objectsRefreshActiveSeq int
	alarmsRefreshActiveSeq  int
	eventsRefreshActiveSeq  int
	eventBus                *eventbus.Bus
	mainThreadQueue         chan func()
	refreshLoopCancel       context.CancelFunc
	refreshStateMu          sync.Mutex
	objectsRefreshInFlight  bool
	objectsRefreshPending   bool
	alarmsRefreshInFlight   bool
	alarmsRefreshPending    bool
	eventsRefreshInFlight   bool
	eventsRefreshPending    bool
	refreshCoalesceMu       sync.Mutex
	pendingRefresh          eventbus.DataRefreshEvent
	refreshCoalescePending  bool
	responseGroupsMu        sync.Mutex
	responseGroupsCache     map[contracts.FrontendSource]responseGroupsCacheEntry
	responseDialogMu        sync.Mutex
	responseDialogAlarmID   int
	responseDialogActive    bool
	phoneDialer             contracts.PhoneDialer
	backendStatusTimer      *qt.QTimer
	lastBackendStatus       string
}

type responseGroupsCacheEntry struct {
	loadedAt time.Time
	provider *backend.FrontendUIDataProvider
	groups   []contracts.FrontendResponseGroup
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
		mainThreadQueue: make(chan func(), mainThreadQueueSize),
	}

	dispatcherTimer := qt.NewQTimer()
	dispatcherTimer.SetInterval(20)
	dispatcherTimer.OnTimeout(func() {
		for i := 0; i < mainThreadQueueDrainLimit; i++ {
			select {
			case f := <-app.mainThreadQueue:
				app.safeRunOnMainThread(f)
			default:
				return
			}
		}
	})
	dispatcherTimer.Start2()

	app.backendStatusTimer = qt.NewQTimer()
	app.backendStatusTimer.SetInterval(1000)
	app.backendStatusTimer.OnTimeout(app.updateBackendStatus)
	app.backendStatusTimer.Start2()

	app.ui.OnSettingsSaved = app.applySettings
	app.ui.OnRefreshRequested = app.refreshData
	app.ui.OnDiagnosticsRequested = app.showDiagnostics
	app.ui.OnResponseGroupsRequested = app.showResponseGroups
	app.ui.OnOperationalMapRequested = app.showOperationalMap
	app.ui.OnNewObjectsRequested = app.showNewObjectsReport
	app.ui.OnExportContacts = app.exportContacts
	app.ui.OnCreateObject = app.createObject
	app.ui.OnCreateCASLObject = app.createCASLObject
	app.ui.OnEditObject = app.editCurrentObject
	app.ui.OnSIMManagement = app.showCurrentObjectSIM
	app.ui.OnBridgeMode = app.setBridgeMonitoringMode
	app.ui.OnCASLBlock = app.showCASLObjectBlock
	app.ui.OnSendSIMSMS = app.sendSIMSMS
	app.ui.OnDialPhone = app.dialPhone
	app.ui.OnProcessAlarms = app.processAlarms
	app.ui.OnPickAlarms = app.pickAlarms
	app.ui.OnRespondAlarm = app.respondToAlarm
	app.ui.OnRunOnMainThread = app.runOnMainThread
	app.ui.OnAlarmSelected = app.handleAlarmSelected
	app.ui.OnEventSelected = app.handleEventSelected
	app.ui.OnStarted = app.showPhoenixLoginIfNeeded
	app.registerEventBusHandlers()
	app.initializeRuntime(preferences)
	return app
}

func (a *Application) showNewObjectsReport() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	a.ui.ShowNewObjectsReport(a.uiData, a.applyObjectContext)
}

func (a *Application) exportContacts() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}

	initialDir := strings.TrimSpace(config.LoadUIConfig(a.ui.Preferences()).ExportDir)
	if initialDir == "" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			initialDir = filepath.Join(homeDir, "Downloads")
		}
	}
	filePath, ok := a.ui.ChooseContactsCSVPath(initialDir)
	if !ok {
		return
	}

	provider := a.uiData
	a.ui.SetStatus("Експорт контактів: завантаження об'єктів...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		objects := provider.GetObjectsContext(ctx)
		if ctx.Err() != nil {
			a.runOnMainThread(func() {
				if a.uiData == provider {
					a.ui.ShowError("Експорт контактів", "Не вдалося завантажити список об'єктів за відведений час.")
					a.ui.SetStatus("Експорт контактів не виконано")
				}
			})
			return
		}

		const workers = 8
		type exportJob struct {
			index  int
			object models.Object
		}
		jobs := make(chan exportJob)
		exportObjects := make([]objexport.ContactExportObject, len(objects))
		var wg sync.WaitGroup
		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobs {
					objectNumber := viewmodels.ObjectDisplayNumber(job.object)
					exportObjects[job.index] = objexport.ContactExportObject{
						Source:       contracts.DetectFrontendSourceByObjectID(job.object.ID),
						ObjectNumber: objectNumber,
						Object:       job.object,
						Contacts:     provider.GetEmployees(strconv.Itoa(job.object.ID)),
					}
				}
			}()
		}
		for index, object := range objects {
			jobs <- exportJob{index: index, object: object}
		}
		close(jobs)
		wg.Wait()

		count, err := objexport.WriteContactsCSV(filePath, exportObjects)
		a.runOnMainThread(func() {
			if a.uiData != provider {
				return
			}
			if err != nil {
				a.ui.ShowError("Експорт контактів", "Не вдалося створити CSV: "+err.Error())
				a.ui.SetStatus("Експорт контактів не виконано")
				return
			}
			a.ui.ShowInfo(
				"Експорт контактів",
				fmt.Sprintf("Експортовано контактів: %d\nФайл: %s", count, filePath),
			)
			a.ui.SetStatus(fmt.Sprintf("Експортовано контактів: %d", count))
		})
	}()
}

func (a *Application) showPhoenixLoginIfNeeded() {
	if a == nil || a.ui == nil {
		return
	}
	cfg := config.LoadDBConfig(a.ui.Preferences())
	phoenixEnabled := cfg.PhoenixEnabled || cfg.NormalizedMode() == config.BackendModePhoenix
	if !phoenixEnabled || config.PhoenixLoginConfigured(cfg) {
		return
	}
	a.ui.ShowPhoenixLogin(func(saved config.DBConfig) {
		a.applySettings(saved, config.LoadUIConfig(a.ui.Preferences()))
	})
}

func (a *Application) initializeRuntime(preferences config.Preferences) {
	dbCfg := config.LoadDBConfig(preferences)
	a.applySettings(dbCfg, config.LoadUIConfig(preferences))
}

func (a *Application) applySettings(dbCfg config.DBConfig, uiCfg config.UIConfig) {
	if a == nil || a.ui == nil {
		return
	}
	if a.refreshLoopCancel != nil {
		a.refreshLoopCancel()
		a.refreshLoopCancel = nil
	}
	a.objectsRefreshSeq++
	a.alarmsRefreshSeq++
	a.eventsRefreshSeq++
	a.uiData = nil
	a.ui.SetDataProvider(nil)
	a.ui.SetAdminProvider(nil)
	if a.runtime != nil {
		a.runtime.Close()
		a.runtime = nil
	}
	a.lastBackendStatus = ""
	a.currentObject = nil
	a.responseGroupsMu.Lock()
	a.responseGroupsCache = nil
	a.responseGroupsMu.Unlock()
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
	a.updateBackendStatus()
	a.ui.ApplyFontSizes(uiCfg)
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
	a.updateBackendStatus()
}

func (a *Application) refreshObjects() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	a.refreshStateMu.Lock()
	if a.objectsRefreshInFlight {
		a.objectsRefreshPending = true
		a.objectsRefreshSeq++
		a.refreshStateMu.Unlock()
		return
	}
	a.objectsRefreshInFlight = true
	a.objectsRefreshSeq++
	seq := a.objectsRefreshSeq
	a.objectsRefreshActiveSeq = seq
	a.refreshStateMu.Unlock()
	go func() {
		defer traceQtOperation("refreshObjects")()
		ctx, cancel := context.WithTimeout(context.Background(), dataRefreshTimeout)
		defer cancel()
		result := make(chan []models.Object, 1)
		go func() { result <- provider.GetObjectsContext(ctx) }()
		var objects []models.Object
		select {
		case objects = <-result:
		case <-ctx.Done():
			log.Warn().Str("operation", "refreshObjects").Dur("timeout", dataRefreshTimeout).Msg("Qt data refresh timed out")
			a.finishObjectsRefresh(seq)
			return
		}
		a.runOnMainThread(func() {
			defer a.finishObjectsRefresh(seq)
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
	a.refreshStateMu.Lock()
	if a.alarmsRefreshInFlight {
		a.alarmsRefreshPending = true
		a.alarmsRefreshSeq++
		a.refreshStateMu.Unlock()
		return
	}
	a.alarmsRefreshInFlight = true
	a.alarmsRefreshSeq++
	seq := a.alarmsRefreshSeq
	a.alarmsRefreshActiveSeq = seq
	a.refreshStateMu.Unlock()
	go func() {
		defer traceQtOperation("refreshAlarms")()
		ctx, cancel := context.WithTimeout(context.Background(), dataRefreshTimeout)
		defer cancel()
		result := make(chan []models.Alarm, 1)
		go func() { result <- provider.GetAlarmsContext(ctx) }()
		var alarms []models.Alarm
		select {
		case alarms = <-result:
		case <-ctx.Done():
			log.Warn().Str("operation", "refreshAlarms").Dur("timeout", dataRefreshTimeout).Msg("Qt data refresh timed out")
			a.finishAlarmsRefresh(seq)
			return
		}
		a.runOnMainThread(func() {
			defer a.finishAlarmsRefresh(seq)
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
	a.refreshStateMu.Lock()
	if a.eventsRefreshInFlight {
		a.eventsRefreshPending = true
		a.eventsRefreshSeq++
		a.refreshStateMu.Unlock()
		return
	}
	a.eventsRefreshInFlight = true
	a.eventsRefreshSeq++
	seq := a.eventsRefreshSeq
	a.eventsRefreshActiveSeq = seq
	a.refreshStateMu.Unlock()
	go func() {
		defer traceQtOperation("refreshEvents")()
		ctx, cancel := context.WithTimeout(context.Background(), dataRefreshTimeout)
		defer cancel()
		result := make(chan []models.Event, 1)
		go func() { result <- provider.GetEventsContext(ctx) }()
		var events []models.Event
		select {
		case events = <-result:
		case <-ctx.Done():
			log.Warn().Str("operation", "refreshEvents").Dur("timeout", dataRefreshTimeout).Msg("Qt data refresh timed out")
			a.finishEventsRefresh(seq)
			return
		}
		if uiCfg.EventLogLimit > 0 && len(events) > uiCfg.EventLogLimit {
			events = events[:uiCfg.EventLogLimit]
		}
		a.runOnMainThread(func() {
			defer a.finishEventsRefresh(seq)
			if a == nil || a.ui == nil || a.uiData != provider || seq != a.eventsRefreshSeq {
				return
			}
			a.currentEventsCount = len(events)
			a.ui.SetEvents(events)
		})
	}()
}

func (a *Application) finishObjectsRefresh(seq int) {
	if a == nil {
		return
	}
	a.refreshStateMu.Lock()
	if seq != a.objectsRefreshActiveSeq {
		a.refreshStateMu.Unlock()
		return
	}
	restart := a.objectsRefreshPending
	a.objectsRefreshInFlight = false
	a.objectsRefreshPending = false
	a.refreshStateMu.Unlock()
	if restart {
		a.refreshObjects()
	}
}

func (a *Application) finishAlarmsRefresh(seq int) {
	if a == nil {
		return
	}
	a.refreshStateMu.Lock()
	if seq != a.alarmsRefreshActiveSeq {
		a.refreshStateMu.Unlock()
		return
	}
	restart := a.alarmsRefreshPending
	a.alarmsRefreshInFlight = false
	a.alarmsRefreshPending = false
	a.refreshStateMu.Unlock()
	if restart {
		a.refreshAlarms()
	}
}

func (a *Application) finishEventsRefresh(seq int) {
	if a == nil {
		return
	}
	a.refreshStateMu.Lock()
	if seq != a.eventsRefreshActiveSeq {
		a.refreshStateMu.Unlock()
		return
	}
	restart := a.eventsRefreshPending
	a.eventsRefreshInFlight = false
	a.eventsRefreshPending = false
	a.refreshStateMu.Unlock()
	if restart {
		a.refreshEvents()
	}
}

func (a *Application) showResponseGroups() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	a.ui.SetStatus("Завантаження груп реагування...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		groups, err := provider.ListResponseGroups(ctx)
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			if err != nil {
				a.ui.ShowError("Групи реагування", "Не вдалося завантажити групи: "+err.Error())
				a.ui.SetStatus("Групи реагування недоступні")
				return
			}
			a.ui.SetStatus(fmt.Sprintf("Груп реагування: %d", len(groups)))
			a.ui.ShowResponseGroups(groups, func(done func([]contracts.FrontendResponseGroup, error)) {
				go func() {
					refreshCtx, refreshCancel := context.WithTimeout(context.Background(), 20*time.Second)
					defer refreshCancel()
					updated, refreshErr := provider.ListResponseGroups(refreshCtx)
					a.runOnMainThread(func() {
						if a == nil || a.ui == nil || a.uiData != provider {
							return
						}
						done(updated, refreshErr)
					})
				}()
			})
		})
	}()
}

func (a *Application) showOperationalMap() {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	provider := a.uiData
	a.ui.SetStatus("Завантаження оперативної карти...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		objects := provider.GetObjects()
		alarms := provider.GetAlarms()
		locations, locationsErr := provider.ListObjectLocations(ctx)
		groups, groupsErr := provider.ListResponseGroups(ctx)
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			if groupsErr != nil {
				a.ui.SetStatus("Карта: групи реагування недоступні")
				groups = nil
			}
			if locationsErr == nil {
				applyObjectLocations(objects, locations)
			}
			objectID, selected := a.ui.ShowOperationalMap(objects, alarms, groups)
			if selected {
				a.reselectObject(objectID)
			}
		})
	}()
}

func applyObjectLocations(objects []models.Object, locations []contracts.ObjectLocation) {
	byID := make(map[int]contracts.ObjectLocation, len(locations))
	for _, location := range locations {
		byID[location.ObjectID] = location
	}
	for index := range objects {
		location, ok := byID[objects[index].ID]
		if !ok {
			continue
		}
		objects[index].Latitude = strings.TrimSpace(location.Latitude)
		objects[index].Longitude = strings.TrimSpace(location.Longitude)
	}
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
	if ids.IsCASLObjectID(a.currentObject.ID) {
		a.openCASLObjectEditor(int64(a.currentObject.ID), false)
		return
	}
	if a.runtime == nil || a.runtime.Provider == nil {
		a.ui.ShowInfo("Редагування об'єкта", "Джерела даних ще не підключені.")
		return
	}
	admin, ok := backend.AsAdminProvider(a.runtime.Provider)
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
	updated, accepted := a.ui.EditObjectCard(admin, card)
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

func (a *Application) createCASLObject() {
	a.openCASLObjectEditor(0, true)
}

func (a *Application) setBridgeMonitoringMode(object models.Object, mode contracts.DisplayBlockMode) {
	if a == nil || a.runtime == nil || a.runtime.Provider == nil {
		return
	}
	admin, ok := backend.AsAdminProvider(a.runtime.Provider)
	if !ok {
		a.ui.ShowInfo("Спостереження МІСТ", "Поточне джерело не підтримує зміну режиму спостереження.")
		return
	}
	number := int64(viewmodels.NumericObjectDisplayNumber(object))
	a.ui.SetStatus("Зміна режиму об'єкта МІСТ №" + strconv.FormatInt(number, 10) + "...")
	go func() {
		err := admin.SetDisplayBlockMode(number, mode)
		a.runOnMainThread(func() {
			if err != nil {
				a.ui.SetStatus("Не вдалося змінити режим об'єкта МІСТ №" + strconv.FormatInt(number, 10))
				a.ui.ShowError("Спостереження МІСТ", err.Error())
				return
			}
			a.refreshData()
			a.ui.SetStatus("Режим об'єкта МІСТ №" + strconv.FormatInt(number, 10) + " оновлено")
		})
	}()
}

func (a *Application) showCASLObjectBlock(object models.Object) {
	if a == nil || a.runtime == nil || a.runtime.Provider == nil {
		return
	}
	provider, ok := a.runtime.Provider.(contracts.CASLObjectEditorProvider)
	if !ok {
		a.ui.ShowInfo("Блокування CASL", "Поточне джерело не підтримує блокування CASL.")
		return
	}
	a.ui.ShowCASLObjectBlock(provider, int64(object.ID), func() {
		a.refreshData()
		a.ui.SetStatus("Стан блокування CASL оновлено")
	})
}

func (a *Application) openCASLObjectEditor(objectID int64, creating bool) {
	if a == nil || a.ui == nil {
		return
	}
	if a.runtime == nil || a.runtime.Provider == nil {
		a.ui.ShowInfo("CASL", "Джерела даних ще не підключені.")
		return
	}
	provider, ok := a.runtime.Provider.(contracts.CASLObjectEditorProvider)
	if !ok {
		a.ui.ShowInfo("CASL", "CASL editor provider недоступний.")
		return
	}
	a.ui.SetStatus("CASL: завантаження редактора...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		snapshot, err := provider.GetCASLObjectEditorSnapshot(ctx, objectID)
		a.runOnMainThread(func() {
			if err != nil {
				a.ui.ShowError("CASL", "Не вдалося завантажити редактор: "+err.Error())
				return
			}
			qtui.DeferOnMainThread(func() {
				savedID, accepted := a.ui.ShowCASLObjectEditor(provider, snapshot, creating)
				if !accepted {
					a.ui.SetStatus("CASL: редагування скасовано")
					return
				}
				a.refreshData()
				a.ui.SetStatus("CASL: об'єкт збережено, obj_id=" + strconv.FormatInt(savedID, 10))
			})
		})
	}()
}

func (a *Application) createObject() {
	if a == nil || a.ui == nil {
		return
	}
	if a.runtime == nil || a.runtime.Provider == nil {
		a.ui.ShowInfo("Створення об'єкта", "Джерела даних ще не підключені.")
		return
	}
	admin, ok := backend.AsAdminProvider(a.runtime.Provider)
	if !ok {
		a.ui.ShowInfo("Створення об'єкта", "Поточне джерело даних не підтримує створення об'єктів.")
		return
	}
	card, warnings, accepted := a.ui.CreateObjectCard(admin)
	if !accepted {
		return
	}
	a.refreshData()
	a.reselectObject(int(card.ObjN))
	a.ui.SetStatus("Об'єкт створено: " + strconv.FormatInt(card.ObjN, 10))
	if len(warnings) > 0 {
		a.ui.ShowInfo("Створено з попередженнями", "Об'єкт створено, але частина додаткових даних не збережена:\n\n"+strings.Join(warnings, "\n"))
	}
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

	provider := a.uiData
	selected := append([]models.Alarm(nil), alarms...)
	a.ui.SetStatus("Завантаження причин відпрацювання...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		options := a.alarmProcessingOptions(provider, ctx, selected)
		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			input, accepted := a.ui.ProcessAlarmsDialog(selected, options)
			if !accepted {
				a.ui.SetStatus("Відпрацювання тривоги скасовано")
				return
			}
			a.executeProcessAlarms(provider, selected, options, input)
		})
	}()
}

func (a *Application) executeProcessAlarms(
	provider *backend.FrontendUIDataProvider,
	alarms []models.Alarm,
	options []contracts.AlarmProcessingOption,
	input qtui.AlarmProcessInput,
) {
	if a == nil || a.ui == nil || provider == nil || len(alarms) == 0 {
		return
	}
	a.ui.SetStatus("Відпрацювання тривог...")
	go func() {
		const operator = contracts.DefaultOperatorName
		successCount := 0
		errorMsgs := make([]string, 0)
		for _, alarm := range alarms {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			var err error
			if len(options) > 0 {
				err = provider.ProcessAlarmWithRequest(ctx, alarm, operator, contracts.AlarmProcessingRequest{
					CauseCode: input.CauseCode,
					Note:      input.Note,
				})
			} else {
				err = provider.ProcessAlarm(strconv.Itoa(alarm.ID), operator, input.Note)
			}
			cancel()
			if err != nil {
				errorMsgs = append(errorMsgs, fmt.Sprintf("№%s: %v", alarm.GetObjectNumberDisplay(), err))
				continue
			}
			successCount++
		}

		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			for _, alarm := range alarms {
				a.invalidateResponseGroupsCache(alarm)
			}
			a.refreshAlarms()
			if len(errorMsgs) > 0 {
				a.ui.ShowError("Відпрацювання тривог", fmt.Sprintf("Успішно відпрацьовано: %d. Помилки:\n%s", successCount, strings.Join(errorMsgs, "\n")))
				return
			}
			if len(alarms) > 1 {
				a.ui.SetStatus(fmt.Sprintf("Відпрацьовано групу з %d тривог", len(alarms)))
				return
			}
			alarm := alarms[0]
			a.ui.SetStatus("Тривогу відпрацьовано: " + alarm.GetObjectNumberDisplay() + " | " + strings.TrimSpace(alarm.GetTypeDisplay()))
		})
	}()
}

func (a *Application) alarmProcessingOptions(
	provider *backend.FrontendUIDataProvider,
	ctx context.Context,
	alarms []models.Alarm,
) []contracts.AlarmProcessingOption {
	if a == nil || provider == nil || len(alarms) == 0 || !sameAlarmProcessingSource(alarms) {
		return nil
	}

	optionSets := make([][]contracts.AlarmProcessingOption, 0, len(alarms))
	for _, alarm := range alarms {
		options, err := provider.GetAlarmProcessingOptions(ctx, alarm)
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

	provider := a.uiData
	selected := append([]models.Alarm(nil), alarms...)
	a.ui.SetStatus("Взяття тривог у роботу...")
	go func() {
		const operator = contracts.DefaultOperatorName
		successCount := 0
		errorMsgs := make([]string, 0)
		for _, alarm := range selected {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			err := provider.PickAlarm(ctx, alarm, operator)
			cancel()
			if err != nil {
				errorMsgs = append(errorMsgs, fmt.Sprintf("№%s: %v", alarm.GetObjectNumberDisplay(), err))
				continue
			}
			successCount++
		}

		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			for _, alarm := range selected {
				a.invalidateResponseGroupsCache(alarm)
			}
			a.refreshAlarms()
			if len(errorMsgs) > 0 {
				a.ui.ShowError("Взяття тривог у роботу", fmt.Sprintf("Успішно взято: %d. Помилки:\n%s", successCount, strings.Join(errorMsgs, "\n")))
				return
			}
			if len(selected) > 1 {
				a.ui.SetStatus(fmt.Sprintf("Взято в роботу групу з %d тривог", len(selected)))
				return
			}
			a.ui.SetStatus("Тривогу взято в роботу: " + selected[0].GetObjectNumberDisplay() + " | " + strings.TrimSpace(selected[0].GetTypeDisplay()))
		})
	}()
}

func (a *Application) respondToAlarm(alarm models.Alarm) {
	if a == nil || a.ui == nil || a.uiData == nil {
		return
	}
	if !a.beginAlarmResponseDialog(alarm.ID) {
		a.ui.SetStatus("Картка цієї тривоги вже завантажується або відкрита")
		return
	}

	provider := a.uiData
	a.ui.SetAlarmResponseLoading(alarm.ID, true)
	a.ui.SetStatus("Завантаження картки реагування...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var (
			groups    []contracts.FrontendResponseGroup
			groupsErr error
			history   []models.AlarmMsg
		)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			groups, groupsErr = a.responseGroupsForAlarm(ctx, provider, alarm)
		}()
		go func() {
			defer wg.Done()
			history = append([]models.AlarmMsg(nil), alarm.SourceMsgs...)
			if len(history) == 0 {
				history = provider.GetAlarmSourceMessages(alarm)
			}
		}()
		wg.Wait()

		a.runOnMainThread(func() {
			defer func() {
				a.endAlarmResponseDialog(alarm.ID)
				if a.ui != nil {
					a.ui.SetAlarmResponseLoading(alarm.ID, false)
				}
			}()
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			if groupsErr != nil {
				a.ui.ShowError("Картка реагування", "Не вдалося завантажити групи реагування: "+groupsErr.Error())
				a.ui.SetStatus("Групи реагування недоступні")
				return
			}

			input, accepted := a.ui.ShowAlarmResponseDialog(alarm, groups, history)
			if !accepted {
				a.ui.SetStatus("Картку реагування закрито")
				return
			}
			switch input.Action {
			case qtui.AlarmResponseTake:
				a.pickAlarms([]models.Alarm{alarm})
			case qtui.AlarmResponseProcess:
				a.processAlarms([]models.Alarm{alarm})
			default:
				a.executeAlarmResponseAction(provider, alarm, input)
			}
		})
	}()
}

func (a *Application) beginAlarmResponseDialog(alarmID int) bool {
	a.responseDialogMu.Lock()
	defer a.responseDialogMu.Unlock()
	if a.responseDialogActive {
		return false
	}
	a.responseDialogActive = true
	a.responseDialogAlarmID = alarmID
	return true
}

func (a *Application) endAlarmResponseDialog(alarmID int) {
	a.responseDialogMu.Lock()
	defer a.responseDialogMu.Unlock()
	if a.responseDialogActive && a.responseDialogAlarmID == alarmID {
		a.responseDialogActive = false
		a.responseDialogAlarmID = 0
	}
}

func (a *Application) responseGroupsForAlarm(
	ctx context.Context,
	provider *backend.FrontendUIDataProvider,
	alarm models.Alarm,
) ([]contracts.FrontendResponseGroup, error) {
	const cacheTTL = 2 * time.Minute
	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)

	a.responseGroupsMu.Lock()
	cached, ok := a.responseGroupsCache[source]
	if ok && cached.provider == provider && time.Since(cached.loadedAt) < cacheTTL {
		groups := append([]contracts.FrontendResponseGroup(nil), cached.groups...)
		a.responseGroupsMu.Unlock()
		return groups, nil
	}
	a.responseGroupsMu.Unlock()

	groups, err := provider.ListResponseGroupsForAlarm(ctx, alarm)
	if err != nil {
		return nil, err
	}

	a.responseGroupsMu.Lock()
	if a.responseGroupsCache == nil {
		a.responseGroupsCache = make(map[contracts.FrontendSource]responseGroupsCacheEntry)
	}
	a.responseGroupsCache[source] = responseGroupsCacheEntry{
		loadedAt: time.Now(),
		provider: provider,
		groups:   append([]contracts.FrontendResponseGroup(nil), groups...),
	}
	a.responseGroupsMu.Unlock()
	return groups, nil
}

func (a *Application) executeAlarmResponseAction(provider *backend.FrontendUIDataProvider, alarm models.Alarm, input qtui.AlarmResponseInput) {
	if a == nil || a.ui == nil || provider == nil {
		return
	}

	actionText := alarmResponseActionText(input.Action)
	a.ui.SetStatus(actionText + "...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		var err error
		switch input.Action {
		case qtui.AlarmResponseAssign:
			err = provider.AssignResponseGroup(ctx, alarm, input.GroupID)
		case qtui.AlarmResponseArrived:
			err = provider.NotifyGroupArrived(ctx, alarm)
		case qtui.AlarmResponseCancel:
			err = provider.CancelResponseGroup(ctx, alarm)
		default:
			return
		}

		a.runOnMainThread(func() {
			if a == nil || a.ui == nil || a.uiData != provider {
				return
			}
			if err != nil {
				a.ui.ShowError("Реагування на тривогу", actionText+": "+err.Error())
				a.ui.SetStatus(actionText + ": помилка")
				return
			}
			a.invalidateResponseGroupsCache(alarm)
			a.refreshAlarms()
			a.ui.SetStatus(actionText + ": виконано")
		})
	}()
}

func (a *Application) invalidateResponseGroupsCache(alarm models.Alarm) {
	if a == nil {
		return
	}
	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)
	a.responseGroupsMu.Lock()
	delete(a.responseGroupsCache, source)
	a.responseGroupsMu.Unlock()
}

func alarmResponseActionText(action qtui.AlarmResponseAction) string {
	switch action {
	case qtui.AlarmResponseTake:
		return "Взяття тривоги в роботу"
	case qtui.AlarmResponseProcess:
		return "Відпрацювання тривоги"
	case qtui.AlarmResponseAssign:
		return "Направлення МГР"
	case qtui.AlarmResponseArrived:
		return "Фіксація прибуття МГР"
	case qtui.AlarmResponseCancel:
		return "Скасування МГР"
	default:
		return "Операція реагування"
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
	health := runtime.SourceHealth()
	if len(health) == 0 {
		return "Джерела даних: не налаштовано"
	}

	parts := make([]string, 0, len(health))
	for _, source := range health {
		name := source.Source.DisplayName()
		if source.Source == contracts.FrontendSourceBridge {
			name = "БД/МІСТ"
		}
		state := "перевірка..."
		switch source.Status {
		case contracts.FrontendSourceHealthStatusOnline:
			state = "підключено"
		case contracts.FrontendSourceHealthStatusDegraded:
			state = "нестабільно"
		case contracts.FrontendSourceHealthStatusOffline:
			state = "недоступно"
		}
		if source.Status != contracts.FrontendSourceHealthStatusOnline {
			if detail := strings.TrimSpace(source.HealthText); detail != "" {
				parts = append(parts, detail)
				continue
			}
		}
		parts = append(parts, name+": "+state)
	}
	return "Джерела даних: " + strings.Join(parts, " | ")
}

func (a *Application) updateBackendStatus() {
	if a == nil || a.ui == nil {
		return
	}
	status := backendStatusText(a.runtime)
	if status == a.lastBackendStatus {
		return
	}
	a.lastBackendStatus = status
	a.ui.SetStatus(status)
}
