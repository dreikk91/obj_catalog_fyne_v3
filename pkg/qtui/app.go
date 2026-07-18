//go:build qt

package qtui

import (
	_ "embed"
	"os"
	"path/filepath"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simcommands"
	"obj_catalog_fyne_v3/pkg/version"
)

//go:embed icon.png
var appIconPNG []byte

type App struct {
	qapp          *qt.QApplication
	preferences   config.Preferences
	mainWindow    *MainWindow
	adminProvider contracts.AdminProvider

	OnSettingsSaved           func(config.DBConfig, config.UIConfig)
	OnRefreshRequested        func()
	OnDiagnosticsRequested    func()
	OnResponseGroupsRequested func()
	OnOperationalMapRequested func()
	OnNewObjectsRequested     func()
	OnExportContacts          func()
	OnCreateObject            func()
	OnCreateCASLObject        func()
	OnEditObject              func()
	OnSIMManagement           func()
	OnBridgeMode              func(models.Object, contracts.DisplayBlockMode)
	OnCASLBlock               func(models.Object)
	OnSendSIMSMS              func(object models.Object, phone string)
	OnDialPhone               func(phone string)
	OnProcessAlarms           func([]models.Alarm)
	OnPickAlarms              func([]models.Alarm)
	OnRespondAlarm            func(models.Alarm)
	OnRunOnMainThread         func(f func())
	OnAlarmSelected           func(models.Alarm)
	OnEventSelected           func(models.Event)
	OnStarted                 func()
}

func NewApp() *App {
	qapp := qt.NewQApplication(os.Args)
	qt.QCoreApplication_SetOrganizationName("MOST")
	qt.QCoreApplication_SetApplicationName("ObjCatalogQt")
	qt.QCoreApplication_SetApplicationVersion(version.Current().String())
	setNativeWindowsStyle()
	setDefaultApplicationFont()
	setApplicationIcon()

	preferences := config.NewQtPreferences("MOST", "ObjCatalogQt")

	app := &App{
		qapp:        qapp,
		preferences: preferences,
	}
	RunOnMainThread = func(f func()) {
		if app.OnRunOnMainThread != nil {
			app.OnRunOnMainThread(f)
		} else {
			fallbackRunOnMainThread(f)
		}
	}
	app.mainWindow = NewMainWindow(app)
	app.mainWindow.OnSettingsRequested = app.ShowSettings
	app.mainWindow.OnRefreshRequested = func() {
		if app.OnRefreshRequested != nil {
			app.OnRefreshRequested()
		}
	}
	app.mainWindow.OnDiagnosticsRequested = func() {
		if app.OnDiagnosticsRequested != nil {
			app.OnDiagnosticsRequested()
		}
	}
	app.mainWindow.OnResponseGroupsRequested = func() {
		if app.OnResponseGroupsRequested != nil {
			app.OnResponseGroupsRequested()
		}
	}
	app.mainWindow.OnOperationalMapRequested = func() {
		if app.OnOperationalMapRequested != nil {
			app.OnOperationalMapRequested()
		}
	}
	app.mainWindow.OnNewObjectsRequested = func() {
		if app.OnNewObjectsRequested != nil {
			app.OnNewObjectsRequested()
		}
	}
	app.mainWindow.OnExportContactsRequested = func() {
		if app.OnExportContacts != nil {
			app.OnExportContacts()
		}
	}
	app.mainWindow.OnCreateObjectRequested = func() {
		if app.OnCreateObject != nil {
			app.OnCreateObject()
		}
	}
	app.mainWindow.OnCreateCASLRequested = func() {
		if app.OnCreateCASLObject != nil {
			app.OnCreateCASLObject()
		}
	}
	app.mainWindow.workArea.OnEditObjectRequested = func() {
		if app.OnEditObject != nil {
			app.OnEditObject()
		}
	}
	app.mainWindow.workArea.OnSIMManagementRequested = func() {
		if app.OnSIMManagement != nil {
			app.OnSIMManagement()
		}
	}
	app.mainWindow.objectList.OnBridgeMode = func(object models.Object, mode contracts.DisplayBlockMode) {
		if app.OnBridgeMode != nil {
			app.OnBridgeMode(object, mode)
		}
	}
	app.mainWindow.objectList.OnCASLBlock = func(object models.Object) {
		if app.OnCASLBlock != nil {
			app.OnCASLBlock(object)
		}
	}
	app.mainWindow.workArea.OnDialPhoneRequested = func(phone string) {
		if app.OnDialPhone != nil {
			app.OnDialPhone(phone)
		}
	}
	app.mainWindow.alarmPanel.OnProcessAlarms = func(alarms []models.Alarm) {
		if app.OnProcessAlarms != nil {
			app.OnProcessAlarms(alarms)
		}
	}
	app.mainWindow.alarmPanel.OnPickAlarms = func(alarms []models.Alarm) {
		if app.OnPickAlarms != nil {
			app.OnPickAlarms(alarms)
		}
	}
	app.mainWindow.alarmPanel.OnRespondAlarm = func(alarm models.Alarm) {
		if app.OnRespondAlarm != nil {
			app.OnRespondAlarm(alarm)
		}
	}
	app.mainWindow.workArea.OnRunOnMainThread = func(f func()) {
		if app.OnRunOnMainThread != nil {
			app.OnRunOnMainThread(f)
		}
	}
	app.mainWindow.alarmPanel.OnAlarmSelected = func(alarm models.Alarm) {
		if app.OnAlarmSelected != nil {
			app.OnAlarmSelected(alarm)
		}
	}
	app.mainWindow.eventLog.OnEventSelected = func(event models.Event) {
		if app.OnEventSelected != nil {
			app.OnEventSelected(event)
		}
	}
	return app
}

func (a *App) SetDataProvider(provider contracts.DataProvider) {
	if a == nil || a.mainWindow == nil {
		return
	}
	if a.mainWindow.alarmPanel != nil {
		a.mainWindow.alarmPanel.SetDataProvider(provider)
	}
	if a.mainWindow.workArea != nil {
		a.mainWindow.workArea.SetDataProvider(provider)
	}
}

func (a *App) Run() int {
	a.ApplyUIConfig(config.LoadUIConfig(a.preferences))
	a.mainWindow.Show()
	if a.OnStarted != nil {
		a.OnStarted()
	}
	return qt.QApplication_Exec()
}

// ApplyUIConfig applies interface settings that can change at runtime.
func (a *App) ApplyUIConfig(uiCfg config.UIConfig) {
	if a == nil || a.mainWindow == nil {
		return
	}
	a.ApplyFontSizes(uiCfg)
	a.mainWindow.ApplyJournalDockPolicy(uiCfg.AllowDetachedJournals)
}

// ApplyFontSizes applies font size settings from UIConfig to all panel tables.
func (a *App) ApplyFontSizes(uiCfg config.UIConfig) {
	if a == nil || a.mainWindow == nil {
		return
	}
	interfaceFont := qt.QApplication_Font()
	interfaceFont.SetPointSizeF(float64(uiCfg.FontSizeInterface))
	qt.QApplication_SetFont(interfaceFont)
	a.mainWindow.SetFont(interfaceFont)
	if a.mainWindow.objectList != nil {
		a.mainWindow.objectList.SetTableFontSize(uiCfg.FontSizeObjects)
	}
	if a.mainWindow.eventLog != nil {
		a.mainWindow.eventLog.SetTableFontSize(uiCfg.FontSizeEvents)
	}
	if a.mainWindow.alarmPanel != nil {
		a.mainWindow.alarmPanel.SetToolbarFontSize(uiCfg.FontSizeInterface)
		a.mainWindow.alarmPanel.SetTableFontSize(uiCfg.FontSizeAlarms)
	}
	if a.mainWindow.workArea != nil {
		a.mainWindow.workArea.SetCardFontSize(uiCfg.FontSizeObjectCard)
		a.mainWindow.workArea.SetTableFontSize(uiCfg.FontSizeObjects, uiCfg.FontSizeEvents)
	}
}

// ShowPhoenixLogin opens the compact Phoenix startup login dialog.
func (a *App) ShowPhoenixLogin(onSaved func(config.DBConfig)) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowPhoenixLoginDialog(a.mainWindow.QWidget, a.preferences, onSaved)
}

// ShowNewObjectsReport opens the new objects report window.
func (a *App) ShowNewObjectsReport(provider contracts.ObjectProvider, onOpen func(models.Object)) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowNewObjectsReport(a.mainWindow.QWidget, provider, onOpen)
}

// ChooseContactsCSVPath opens a save dialog for the contacts CSV file.
func (a *App) ChooseContactsCSVPath(initialDir string) (string, bool) {
	if a == nil || a.mainWindow == nil {
		return "", false
	}
	initialDir = strings.TrimSpace(initialDir)
	dialog := qt.NewQFileDialog6(
		a.mainWindow.QWidget,
		"Експорт контактів",
		initialDir,
		"CSV files (*.csv)",
	)
	defer dialog.Delete()
	dialog.SetAcceptMode(qt.QFileDialog__AcceptSave)
	dialog.SetFileMode(qt.QFileDialog__AnyFile)
	dialog.SetDefaultSuffix("csv")
	dialog.SelectFile("contacts.csv")
	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return "", false
	}
	files := dialog.SelectedFiles()
	if len(files) == 0 || strings.TrimSpace(files[0]) == "" {
		return "", false
	}
	filePath := strings.TrimSpace(files[0])
	if !strings.EqualFold(filepath.Ext(filePath), ".csv") {
		filePath += ".csv"
	}
	return filePath, true
}

func (a *App) Preferences() config.Preferences {
	return a.preferences
}

func (a *App) SetAdminProvider(provider contracts.AdminProvider) {
	if a == nil {
		return
	}
	a.adminProvider = provider
}

func (a *App) SelectObject(id int) {
	if a == nil || a.mainWindow == nil || a.mainWindow.objectList == nil {
		return
	}
	a.mainWindow.objectList.SelectObject(id)
}

func (a *App) ShowSettings() {
	if a == nil || a.mainWindow == nil || a.preferences == nil {
		return
	}
	ShowSettingsDialog(a.mainWindow.QWidget, a.preferences, func(dbCfg config.DBConfig, uiCfg config.UIConfig) {
		a.ApplyUIConfig(uiCfg)
		if a.OnSettingsSaved != nil {
			a.OnSettingsSaved(dbCfg, uiCfg)
		}
	})
}

func (a *App) EditObjectCard(provider contracts.AdminObjectDialogProvider, card contracts.AdminObjectCard) (contracts.AdminObjectCard, bool) {
	if a == nil || a.mainWindow == nil {
		return card, false
	}
	return ShowObjectEditDialog(a.mainWindow.QWidget, provider, card)
}

func (a *App) CreateObjectCard(provider contracts.AdminObjectDialogProvider) (contracts.AdminObjectCard, []string, bool) {
	if a == nil || a.mainWindow == nil {
		return contracts.AdminObjectCard{}, nil, false
	}
	return ShowObjectCreateDialog(a.mainWindow.QWidget, provider)
}

func (a *App) ShowCASLObjectEditor(
	provider contracts.CASLObjectEditorProvider,
	snapshot contracts.CASLObjectEditorSnapshot,
	creating bool,
) (int64, bool) {
	if a == nil || a.mainWindow == nil {
		return 0, false
	}
	return ShowCASLObjectDialog(a.mainWindow.QWidget, provider, snapshot, creating)
}

func (a *App) ShowCASLObjectBlock(
	provider contracts.CASLObjectEditorProvider,
	objectID int64,
	onSuccess func(),
) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowCASLObjectBlockDialog(a.mainWindow.QWidget, provider, objectID, onSuccess)
}

func (a *App) ShowSIMManagement(object models.Object, usageText string) {
	if a == nil || a.mainWindow == nil {
		return
	}
	var (
		vf contracts.AdminObjectVodafoneService
		ks contracts.AdminObjectKyivstarService
	)
	if a.adminProvider != nil {
		vf = a.adminProvider
		ks = a.adminProvider
	}
	ShowSIMManagementDialog(a.mainWindow.QWidget, object, usageText, vf, ks, func(object models.Object, phone string) {
		if a.OnSendSIMSMS != nil {
			a.OnSendSIMSMS(object, phone)
		}
	})
}

func (a *App) ShowSIMSMS(object models.Object, phone string, cfg config.OmnicellConfig) ([]simcommands.SMSCommand, bool) {
	if a == nil || a.mainWindow == nil {
		return nil, false
	}
	return ShowSIMSMSDialog(a.mainWindow.QWidget, object, phone, cfg)
}

func (a *App) ProcessAlarmDialog(alarm models.Alarm, options []contracts.AlarmProcessingOption) (AlarmProcessInput, bool) {
	return a.ProcessAlarmsDialog([]models.Alarm{alarm}, options)
}

func (a *App) ProcessAlarmsDialog(alarms []models.Alarm, options []contracts.AlarmProcessingOption) (AlarmProcessInput, bool) {
	if a == nil || a.mainWindow == nil {
		return AlarmProcessInput{}, false
	}
	return ShowAlarmProcessDialogForAlarms(a.mainWindow.QWidget, alarms, options)
}

func (a *App) ShowAlarmResponseDialog(
	alarm models.Alarm,
	groups []contracts.FrontendResponseGroup,
	history []models.AlarmMsg,
) (AlarmResponseInput, bool) {
	if a == nil || a.mainWindow == nil {
		return AlarmResponseInput{}, false
	}
	return ShowAlarmResponseDialog(a.mainWindow.QWidget, alarm, groups, history)
}

func (a *App) SetAlarmResponseLoading(alarmID int, loading bool) {
	if a == nil || a.mainWindow == nil || a.mainWindow.alarmPanel == nil {
		return
	}
	a.mainWindow.alarmPanel.SetResponseLoading(alarmID, loading)
}

func (a *App) ShowInfo(title string, message string) {
	if a == nil || a.mainWindow == nil {
		return
	}
	qt.QMessageBox_Information(a.mainWindow.QWidget, title, strings.TrimSpace(message))
}

func (a *App) ShowText(title string, message string) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowTextDialog(a.mainWindow.QWidget, title, message)
}

func (a *App) ShowDiagnostics(report DiagnosticsReport) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowDiagnosticsDialog(a.mainWindow.QWidget, report)
}

func (a *App) ShowResponseGroups(groups []contracts.FrontendResponseGroup, reload ResponseGroupsReload) {
	if a == nil || a.mainWindow == nil {
		return
	}
	ShowResponseGroupsDialog(a.mainWindow.QWidget, groups, reload)
}

func (a *App) ShowOperationalMap(
	objects []models.Object,
	alarms []models.Alarm,
	groups []contracts.FrontendResponseGroup,
) (int, bool) {
	if a == nil || a.mainWindow == nil {
		return 0, false
	}
	return ShowOperationalMapDialog(a.mainWindow.QWidget, objects, alarms, groups)
}

func (a *App) ShowError(title string, message string) {
	if a == nil || a.mainWindow == nil {
		return
	}
	qt.QMessageBox_Critical(a.mainWindow.QWidget, title, strings.TrimSpace(message))
}

func (a *App) SetStatus(text string) {
	if a == nil || a.mainWindow == nil {
		return
	}
	a.mainWindow.SetStatus(text)
}

func (a *App) SetObjects(objects []models.Object) {
	if a == nil || a.mainWindow == nil || a.mainWindow.objectList == nil {
		return
	}
	a.mainWindow.objectList.SetObjects(objects)
}

func (a *App) SetAlarms(alarms []models.Alarm) {
	if a == nil || a.mainWindow == nil || a.mainWindow.alarmPanel == nil {
		return
	}
	a.mainWindow.alarmPanel.SetAlarms(alarms)
}

func (a *App) SetEvents(events []models.Event) {
	if a == nil || a.mainWindow == nil || a.mainWindow.eventLog == nil {
		return
	}
	a.mainWindow.eventLog.SetEvents(events)
}

func (a *App) SetObjectDetails(object models.Object, zones []models.Zone, contacts []models.Contact, events []models.Event) {
	if a == nil || a.mainWindow == nil {
		return
	}
	if a.mainWindow.workArea != nil {
		a.mainWindow.workArea.SetObject(object, zones, contacts, events)
	}
	if a.mainWindow.eventLog != nil {
		a.mainWindow.eventLog.SetCurrentObject(&object)
	}
}

func (a *App) RefreshCurrentObjectEvents() {
	if a == nil || a.mainWindow == nil || a.mainWindow.workArea == nil {
		return
	}
	a.mainWindow.workArea.RefreshEventsIfVisible()
}

func (a *App) SetObjectLoading(object models.Object) {
	if a == nil || a.mainWindow == nil {
		return
	}
	if a.mainWindow.workArea != nil {
		a.mainWindow.workArea.SetLoading(object)
	}
	if a.mainWindow.eventLog != nil {
		a.mainWindow.eventLog.SetCurrentObject(&object)
	}
}

func (a *App) SetObjectSelectedHandler(handler func(models.Object)) {
	if a == nil || a.mainWindow == nil || a.mainWindow.objectList == nil {
		return
	}
	a.mainWindow.objectList.OnObjectSelected = handler
}

func setDefaultApplicationFont() {
	font := qt.NewQFont6("Segoe UI", 10)
	if font.PointSize() <= 0 {
		font.SetPointSize(10)
	}
	qt.QApplication_SetFont(font)
}

func setApplicationIcon() {
	pixmap := qt.NewQPixmap()
	pixmap.LoadFromDataWithData(appIconPNG)
	icon := qt.NewQIcon2(pixmap)
	qt.QGuiApplication_SetWindowIcon(icon)
}

func setNativeWindowsStyle() {
	for _, name := range []string{"windowsvista", "windows"} {
		style := qt.QStyleFactory_Create(name)
		if style != nil {
			qt.QApplication_SetStyle(style)
			return
		}
	}
}
