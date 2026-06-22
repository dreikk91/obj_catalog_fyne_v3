//go:build qt

package qtui

import (
	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
)

type settingsDialog struct {
	dialog *qt.QDialog
	prefs  config.Preferences

	firebirdEnabled *qt.QCheckBox
	dbUser          *qt.QLineEdit
	dbPassword      *qt.QLineEdit
	dbHost          *qt.QLineEdit
	dbPort          *qt.QLineEdit
	dbPath          *qt.QLineEdit
	dbParams        *qt.QLineEdit

	phoenixEnabled  *qt.QCheckBox
	phoenixUser     *qt.QLineEdit
	phoenixPassword *qt.QLineEdit
	phoenixHost     *qt.QLineEdit
	phoenixPort     *qt.QLineEdit
	phoenixInstance *qt.QLineEdit
	phoenixDatabase *qt.QLineEdit
	phoenixParams   *qt.QLineEdit

	caslEnabled *qt.QCheckBox
	caslBaseURL *qt.QLineEdit
	caslToken   *qt.QLineEdit
	caslEmail   *qt.QLineEdit
	caslPass    *qt.QLineEdit
	caslPultID  *qt.QSpinBox
	logLevel    *qt.QComboBox

	fontSizeObjects        *qt.QDoubleSpinBox
	fontSizeEvents         *qt.QDoubleSpinBox
	fontSizeAlarms         *qt.QDoubleSpinBox
	showBottomAlarmJournal *qt.QCheckBox
	showBottomEventJournal *qt.QCheckBox
	eventLogLimit          *qt.QSpinBox
	objectLogLimit         *qt.QSpinBox
	exportDir              *qt.QLineEdit
}

func ShowSettingsDialog(parent *qt.QWidget, prefs config.Preferences, onSaved func(config.DBConfig, config.UIConfig)) {
	if prefs == nil {
		return
	}
	d := newSettingsDialog(parent, prefs)
	if d.dialog.Exec() != int(qt.QDialog__Accepted) {
		return
	}
	dbCfg, uiCfg := d.values()
	config.SaveDBConfig(prefs, dbCfg)
	config.SaveUIConfig(prefs, uiCfg)
	if onSaved != nil {
		onSaved(dbCfg, uiCfg)
	}
}

func newSettingsDialog(parent *qt.QWidget, prefs config.Preferences) *settingsDialog {
	d := &settingsDialog{
		dialog: qt.NewQDialog(parent),
		prefs:  prefs,
	}
	d.dialog.SetWindowTitle("Налаштування")
	d.dialog.Resize(720, 640)

	root := qt.NewQVBoxLayout(d.dialog.QWidget)
	tabs := qt.NewQTabWidget2()
	tabs.AddTab(d.buildDataSourcesTab(), "Джерела даних")
	tabs.AddTab(d.buildInterfaceTab(), "Інтерфейс")
	root.AddWidget(tabs.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(func() { d.dialog.Accept() })
	buttons.OnRejected(func() { d.dialog.Reject() })
	root.AddWidget(buttons.QWidget)
	d.dialog.SetLayout(root.QLayout)

	d.load(config.LoadDBConfig(prefs), config.LoadUIConfig(prefs))
	return d
}

func (d *settingsDialog) buildDataSourcesTab() *qt.QWidget {
	tab := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(tab)

	form := qt.NewQFormLayout2()
	d.firebirdEnabled = qt.NewQCheckBox3("Увімкнути БД/МІСТ")
	form.AddRow3("БД/МІСТ", d.firebirdEnabled.QWidget)
	d.dbHost = lineEdit()
	form.AddRow3("Host", d.dbHost.QWidget)
	d.dbPort = lineEdit()
	form.AddRow3("Port", d.dbPort.QWidget)
	d.dbPath = lineEdit()
	form.AddRow3("Path", d.dbPath.QWidget)
	d.dbUser = lineEdit()
	form.AddRow3("User", d.dbUser.QWidget)
	d.dbPassword = passwordEdit()
	form.AddRow3("Password", d.dbPassword.QWidget)
	d.dbParams = lineEdit()
	form.AddRow3("Params", d.dbParams.QWidget)

	d.phoenixEnabled = qt.NewQCheckBox3("Увімкнути Phoenix")
	form.AddRow3("Phoenix", d.phoenixEnabled.QWidget)
	d.phoenixHost = lineEdit()
	form.AddRow3("Phoenix host", d.phoenixHost.QWidget)
	d.phoenixPort = lineEdit()
	form.AddRow3("Phoenix port", d.phoenixPort.QWidget)
	d.phoenixInstance = lineEdit()
	form.AddRow3("Instance", d.phoenixInstance.QWidget)
	d.phoenixDatabase = lineEdit()
	form.AddRow3("Database", d.phoenixDatabase.QWidget)
	d.phoenixUser = lineEdit()
	form.AddRow3("Phoenix user", d.phoenixUser.QWidget)
	d.phoenixPassword = passwordEdit()
	form.AddRow3("Phoenix password", d.phoenixPassword.QWidget)
	d.phoenixParams = lineEdit()
	form.AddRow3("Phoenix params", d.phoenixParams.QWidget)

	d.caslEnabled = qt.NewQCheckBox3("Увімкнути CASL Cloud")
	form.AddRow3("CASL", d.caslEnabled.QWidget)
	d.caslBaseURL = lineEdit()
	form.AddRow3("CASL URL", d.caslBaseURL.QWidget)
	d.caslToken = passwordEdit()
	form.AddRow3("CASL token", d.caslToken.QWidget)
	d.caslEmail = lineEdit()
	form.AddRow3("CASL email", d.caslEmail.QWidget)
	d.caslPass = passwordEdit()
	form.AddRow3("CASL password", d.caslPass.QWidget)
	d.caslPultID = spinBox(0, 1000000000)
	form.AddRow3("CASL pult ID", d.caslPultID.QWidget)
	d.logLevel = qt.NewQComboBox2()
	d.logLevel.AddItems([]string{"debug", "info", "warn", "error"})
	form.AddRow3("Log level", d.logLevel.QWidget)

	layout.AddLayout(form.QLayout)
	tab.SetLayout(layout.QLayout)
	return tab
}

func (d *settingsDialog) buildInterfaceTab() *qt.QWidget {
	tab := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(tab)
	form := qt.NewQFormLayout2()

	d.fontSizeObjects = doubleSpinBox(config.MinFontSize, config.MaxFontSize)
	form.AddRow3("Шрифт об'єктів", d.fontSizeObjects.QWidget)
	d.fontSizeEvents = doubleSpinBox(config.MinFontSize, config.MaxFontSize)
	form.AddRow3("Шрифт подій", d.fontSizeEvents.QWidget)
	d.fontSizeAlarms = doubleSpinBox(config.MinFontSize, config.MaxFontSize)
	form.AddRow3("Шрифт тривог", d.fontSizeAlarms.QWidget)
	d.showBottomAlarmJournal = qt.NewQCheckBox3("Показувати тривоги в нижній панелі")
	form.AddRow3("Тривоги", d.showBottomAlarmJournal.QWidget)
	d.showBottomEventJournal = qt.NewQCheckBox3("Показувати журнал у нижній панелі")
	form.AddRow3("Журнал", d.showBottomEventJournal.QWidget)
	d.eventLogLimit = spinBox(0, 100000)
	form.AddRow3("Ліміт загального журналу", d.eventLogLimit.QWidget)
	d.objectLogLimit = spinBox(0, 100000)
	form.AddRow3("Ліміт журналу об'єкта", d.objectLogLimit.QWidget)
	d.exportDir = lineEdit()
	form.AddRow3("Папка експорту", d.exportDir.QWidget)

	layout.AddLayout(form.QLayout)
	tab.SetLayout(layout.QLayout)
	return tab
}

func (d *settingsDialog) load(dbCfg config.DBConfig, uiCfg config.UIConfig) {
	d.firebirdEnabled.SetChecked(dbCfg.FirebirdEnabled)
	d.dbUser.SetText(dbCfg.User)
	d.dbPassword.SetText(dbCfg.Password)
	d.dbHost.SetText(dbCfg.Host)
	d.dbPort.SetText(dbCfg.Port)
	d.dbPath.SetText(dbCfg.Path)
	d.dbParams.SetText(dbCfg.Params)

	d.phoenixEnabled.SetChecked(dbCfg.PhoenixEnabled)
	d.phoenixUser.SetText(dbCfg.PhoenixUser)
	d.phoenixPassword.SetText(dbCfg.PhoenixPassword)
	d.phoenixHost.SetText(dbCfg.PhoenixHost)
	d.phoenixPort.SetText(dbCfg.PhoenixPort)
	d.phoenixInstance.SetText(dbCfg.PhoenixInstance)
	d.phoenixDatabase.SetText(dbCfg.PhoenixDatabase)
	d.phoenixParams.SetText(dbCfg.PhoenixParams)

	d.caslEnabled.SetChecked(dbCfg.CASLEnabled)
	d.caslBaseURL.SetText(dbCfg.CASLBaseURL)
	d.caslToken.SetText(dbCfg.CASLToken)
	d.caslEmail.SetText(dbCfg.CASLEmail)
	d.caslPass.SetText(dbCfg.CASLPass)
	d.caslPultID.SetValue(int(dbCfg.CASLPultID))
	setComboText(d.logLevel, dbCfg.LogLevel)

	d.fontSizeObjects.SetValue(float64(uiCfg.FontSizeObjects))
	d.fontSizeEvents.SetValue(float64(uiCfg.FontSizeEvents))
	d.fontSizeAlarms.SetValue(float64(uiCfg.FontSizeAlarms))
	d.showBottomAlarmJournal.SetChecked(uiCfg.ShowBottomAlarmJournal)
	d.showBottomEventJournal.SetChecked(uiCfg.ShowBottomEventJournal)
	d.eventLogLimit.SetValue(uiCfg.EventLogLimit)
	d.objectLogLimit.SetValue(uiCfg.ObjectLogLimit)
	d.exportDir.SetText(uiCfg.ExportDir)
}

func (d *settingsDialog) values() (config.DBConfig, config.UIConfig) {
	dbCfg := config.LoadDBConfig(d.prefs)
	dbCfg.FirebirdEnabled = d.firebirdEnabled.IsChecked()
	dbCfg.User = d.dbUser.Text()
	dbCfg.Password = d.dbPassword.Text()
	dbCfg.Host = d.dbHost.Text()
	dbCfg.Port = d.dbPort.Text()
	dbCfg.Path = d.dbPath.Text()
	dbCfg.Params = d.dbParams.Text()

	dbCfg.PhoenixEnabled = d.phoenixEnabled.IsChecked()
	dbCfg.PhoenixUser = d.phoenixUser.Text()
	dbCfg.PhoenixPassword = d.phoenixPassword.Text()
	dbCfg.PhoenixHost = d.phoenixHost.Text()
	dbCfg.PhoenixPort = d.phoenixPort.Text()
	dbCfg.PhoenixInstance = d.phoenixInstance.Text()
	dbCfg.PhoenixDatabase = d.phoenixDatabase.Text()
	dbCfg.PhoenixParams = d.phoenixParams.Text()

	dbCfg.CASLEnabled = d.caslEnabled.IsChecked()
	dbCfg.CASLBaseURL = d.caslBaseURL.Text()
	dbCfg.CASLToken = d.caslToken.Text()
	dbCfg.CASLEmail = d.caslEmail.Text()
	dbCfg.CASLPass = d.caslPass.Text()
	dbCfg.CASLPultID = int64(d.caslPultID.Value())
	dbCfg.LogLevel = d.logLevel.CurrentText()
	dbCfg.Mode = backendModeFromEnabled(dbCfg)

	uiCfg := config.LoadUIConfig(d.prefs)
	uiCfg.FontSizeObjects = float32(d.fontSizeObjects.Value())
	uiCfg.FontSizeEvents = float32(d.fontSizeEvents.Value())
	uiCfg.FontSizeAlarms = float32(d.fontSizeAlarms.Value())
	uiCfg.FontSize = uiCfg.FontSizeObjects
	uiCfg.ShowBottomAlarmJournal = d.showBottomAlarmJournal.IsChecked()
	uiCfg.ShowBottomEventJournal = d.showBottomEventJournal.IsChecked()
	uiCfg.EventLogLimit = d.eventLogLimit.Value()
	uiCfg.ObjectLogLimit = d.objectLogLimit.Value()
	uiCfg.ExportDir = d.exportDir.Text()

	return dbCfg, uiCfg
}

func backendModeFromEnabled(cfg config.DBConfig) string {
	switch {
	case cfg.CASLEnabled:
		return config.BackendModeCASLCloud
	case cfg.PhoenixEnabled && !cfg.FirebirdEnabled:
		return config.BackendModePhoenix
	default:
		return config.BackendModeFirebird
	}
}

func lineEdit() *qt.QLineEdit {
	return qt.NewQLineEdit2()
}

func passwordEdit() *qt.QLineEdit {
	edit := qt.NewQLineEdit2()
	edit.SetEchoMode(qt.QLineEdit__Password)
	return edit
}

func spinBox(min int, max int) *qt.QSpinBox {
	box := qt.NewQSpinBox2()
	box.SetRange(min, max)
	return box
}

func doubleSpinBox(min float64, max float64) *qt.QDoubleSpinBox {
	box := qt.NewQDoubleSpinBox2()
	box.SetRange(min, max)
	box.SetDecimals(1)
	box.SetSingleStep(0.5)
	return box
}

func setComboText(combo *qt.QComboBox, value string) {
	idx := combo.FindText(value)
	if idx < 0 {
		idx = combo.FindText("info")
	}
	if idx >= 0 {
		combo.SetCurrentIndex(idx)
	}
}
