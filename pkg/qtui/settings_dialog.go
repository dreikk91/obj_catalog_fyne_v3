//go:build qt

package qtui

import (
	"context"
	"fmt"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/ami"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/simcommands"
)

const (
	qtPhoenixRoleDutyLabel  = "Черговий оператор (Duty Operator)"
	qtPhoenixRoleAdminLabel = "Адміністратор (Administrator)"
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

	phoenixEnabled          *qt.QCheckBox
	phoenixUser             *qt.QLineEdit
	phoenixPassword         *qt.QLineEdit
	phoenixHost             *qt.QLineEdit
	phoenixPort             *qt.QLineEdit
	phoenixInstance         *qt.QLineEdit
	phoenixDatabase         *qt.QLineEdit
	phoenixParams           *qt.QLineEdit
	phoenixControlHost      *qt.QLineEdit
	phoenixClientRole       *qt.QComboBox
	phoenixOperator         *qt.QComboBox
	phoenixOperatorPassword *qt.QLineEdit
	phoenixRuntimeStatus    *qt.QLabel
	phoenixOperatorsByLabel map[string]data.PhoenixOperator

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

	vodafonePhone       *qt.QLineEdit
	vodafoneAccessToken *qt.QLineEdit
	vodafoneTokenExpiry *qt.QLineEdit
	vodafoneLoginMethod *qt.QComboBox
	vodafonePUK         *qt.QLineEdit
	vodafoneAutoReset   *qt.QCheckBox
	vodafoneDailyLimit  *qt.QSpinBox
	vodafoneWindowHours *qt.QSpinBox

	kyivstarClientID    *qt.QLineEdit
	kyivstarSecret      *qt.QLineEdit
	kyivstarEmail       *qt.QLineEdit
	kyivstarAccessToken *qt.QLineEdit
	kyivstarTokenExpiry *qt.QLineEdit
	kyivstarAutoReset   *qt.QCheckBox
	kyivstarDailyLimit  *qt.QSpinBox
	kyivstarWindowHours *qt.QSpinBox

	omnicellEnabled             *qt.QCheckBox
	omnicellEndpoint            *qt.QLineEdit
	omnicellLogin               *qt.QLineEdit
	omnicellPassword            *qt.QLineEdit
	omnicellSource              *qt.QLineEdit
	omnicellPrimaryAPN          *qt.QLineEdit
	omnicellReserveAPN          *qt.QLineEdit
	omnicellPrimaryIP           *qt.QLineEdit
	omnicellReserveIP           *qt.QLineEdit
	omnicellPrimaryModulePort   *qt.QSpinBox
	omnicellReserveModulePort   *qt.QSpinBox
	omnicellPrimaryReceiverPort *qt.QSpinBox
	omnicellReserveReceiverPort *qt.QSpinBox
	omnicellPrimaryInterval     *qt.QSpinBox
	omnicellReserveInterval     *qt.QSpinBox
	omnicellInput1Confirm       *qt.QCheckBox
	omnicellDefaultProfile      *qt.QComboBox

	amiEnabled   *qt.QCheckBox
	amiHost      *qt.QLineEdit
	amiPort      *qt.QSpinBox
	amiUsername  *qt.QLineEdit
	amiSecret    *qt.QLineEdit
	amiExtension *qt.QLineEdit
	amiContext   *qt.QLineEdit
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
	if dbCfg.PhoenixEnabled && dbCfg.PhoenixOperatorID > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		err := data.ValidatePhoenixOperatorCredentials(ctx, dbCfg)
		cancel()
		if err != nil {
			qt.QMessageBox_Critical(parent, "Користувач Phoenix", err.Error())
			return
		}
	}
	config.SaveDBConfig(prefs, dbCfg)
	config.SaveUIConfig(prefs, uiCfg)
	d.saveOperatorAndCommandSettings()
	if onSaved != nil {
		onSaved(dbCfg, uiCfg)
	}
}

func newSettingsDialog(parent *qt.QWidget, prefs config.Preferences) *settingsDialog {
	d := &settingsDialog{
		dialog:                  qt.NewQDialog(parent),
		prefs:                   prefs,
		phoenixOperatorsByLabel: make(map[string]data.PhoenixOperator),
	}
	d.dialog.SetWindowTitle("Налаштування")
	d.dialog.Resize(640, 520)

	root := qt.NewQVBoxLayout(d.dialog.QWidget)
	tabs := qt.NewQTabWidget2()
	tabs.AddTab(d.buildDataSourcesTab(), "Джерела даних")
	tabs.AddTab(d.buildOperatorsTab(), "Оператори і команди")
	tabs.AddTab(d.buildInterfaceTab(), "Інтерфейс")
	root.AddWidget(tabs.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Ok | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(func() { d.dialog.Accept() })
	buttons.OnRejected(func() { d.dialog.Reject() })
	root.AddWidget(buttons.QWidget)
	d.dialog.SetLayout(root.QLayout)

	d.load(config.LoadDBConfig(prefs), config.LoadUIConfig(prefs))
	d.loadOperatorAndCommandSettings()
	return d
}

func (d *settingsDialog) buildDataSourcesTab() *qt.QWidget {
	tab := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(tab)
	tabs := qt.NewQTabWidget2()
	tabs.SetUsesScrollButtons(true)

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
	tabs.AddTab(wrapForm(form), "БД/МІСТ")

	form = qt.NewQFormLayout2()
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
	d.phoenixControlHost = lineEdit()
	d.phoenixControlHost.SetPlaceholderText("IP або DNS Phoenix Control Center")
	form.AddRow3("Центр керування", d.phoenixControlHost.QWidget)
	d.phoenixClientRole = qt.NewQComboBox2()
	d.phoenixClientRole.AddItems([]string{qtPhoenixRoleDutyLabel, qtPhoenixRoleAdminLabel})
	form.AddRow3("Роль клієнта", d.phoenixClientRole.QWidget)
	d.phoenixOperator = qt.NewQComboBox2()
	form.AddRow3("Оператор", d.phoenixOperator.QWidget)
	d.phoenixOperatorPassword = passwordEdit()
	form.AddRow3("Пароль оператора", d.phoenixOperatorPassword.QWidget)
	refreshPhoenix := qt.NewQPushButton3("Оновити користувачів і порти з БД")
	refreshPhoenix.OnClicked(d.refreshPhoenixRuntimeMetadata)
	form.AddRow3("", refreshPhoenix.QWidget)
	verifyPhoenix := qt.NewQPushButton3("Перевірити оператора")
	verifyPhoenix.OnClicked(d.verifyPhoenixOperator)
	form.AddRow3("", verifyPhoenix.QWidget)
	d.phoenixRuntimeStatus = qt.NewQLabel3("Порти та користувачі ще не завантажені")
	d.phoenixRuntimeStatus.SetWordWrap(true)
	form.AddRow3("Стан", d.phoenixRuntimeStatus.QWidget)
	tabs.AddTab(wrapForm(form), "Phoenix")

	form = qt.NewQFormLayout2()
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
	tabs.AddTab(wrapForm(form), "CASL")

	layout.AddWidget(tabs.QWidget)
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

func (d *settingsDialog) buildOperatorsTab() *qt.QWidget {
	tab := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(tab)
	tabs := qt.NewQTabWidget2()
	tabs.SetUsesScrollButtons(true)

	form := qt.NewQFormLayout2()

	d.vodafonePhone = lineEdit()
	form.AddRow3("Vodafone phone", d.vodafonePhone.QWidget)
	d.vodafoneAccessToken = passwordEdit()
	form.AddRow3("Vodafone token", d.vodafoneAccessToken.QWidget)
	d.vodafoneTokenExpiry = lineEdit()
	form.AddRow3("Vodafone token expiry", d.vodafoneTokenExpiry.QWidget)
	d.vodafoneLoginMethod = qt.NewQComboBox2()
	d.vodafoneLoginMethod.AddItems([]string{config.VodafoneLoginMethodSMS, config.VodafoneLoginMethodPUK})
	form.AddRow3("Vodafone login", d.vodafoneLoginMethod.QWidget)
	d.vodafonePUK = passwordEdit()
	form.AddRow3("Vodafone PUK", d.vodafonePUK.QWidget)
	d.vodafoneAutoReset = qt.NewQCheckBox3("Автоскидання SIM Vodafone")
	form.AddRow3("Vodafone auto reset", d.vodafoneAutoReset.QWidget)
	d.vodafoneDailyLimit = spinBox(0, 100)
	form.AddRow3("Vodafone daily limit", d.vodafoneDailyLimit.QWidget)
	d.vodafoneWindowHours = spinBox(config.MinVodafoneAutoResetWindowHours, 24*30)
	form.AddRow3("Vodafone window, hours", d.vodafoneWindowHours.QWidget)
	tabs.AddTab(wrapForm(form), "Vodafone")

	form = qt.NewQFormLayout2()
	d.kyivstarClientID = lineEdit()
	form.AddRow3("Kyivstar client ID", d.kyivstarClientID.QWidget)
	d.kyivstarSecret = passwordEdit()
	form.AddRow3("Kyivstar secret", d.kyivstarSecret.QWidget)
	d.kyivstarEmail = lineEdit()
	form.AddRow3("Kyivstar email", d.kyivstarEmail.QWidget)
	d.kyivstarAccessToken = passwordEdit()
	form.AddRow3("Kyivstar token", d.kyivstarAccessToken.QWidget)
	d.kyivstarTokenExpiry = lineEdit()
	form.AddRow3("Kyivstar token expiry", d.kyivstarTokenExpiry.QWidget)
	d.kyivstarAutoReset = qt.NewQCheckBox3("Автоскидання SIM Kyivstar")
	form.AddRow3("Kyivstar auto reset", d.kyivstarAutoReset.QWidget)
	d.kyivstarDailyLimit = spinBox(0, 100)
	form.AddRow3("Kyivstar daily limit", d.kyivstarDailyLimit.QWidget)
	d.kyivstarWindowHours = spinBox(config.MinKyivstarAutoResetWindowHours, 24*30)
	form.AddRow3("Kyivstar window, hours", d.kyivstarWindowHours.QWidget)
	tabs.AddTab(wrapForm(form), "Kyivstar")

	form = qt.NewQFormLayout2()
	d.omnicellEnabled = qt.NewQCheckBox3("Увімкнути Omnicell SMS")
	form.AddRow3("Omnicell SMS", d.omnicellEnabled.QWidget)
	d.omnicellEndpoint = lineEdit()
	form.AddRow3("Omnicell endpoint", d.omnicellEndpoint.QWidget)
	d.omnicellLogin = lineEdit()
	form.AddRow3("Omnicell login", d.omnicellLogin.QWidget)
	d.omnicellPassword = passwordEdit()
	form.AddRow3("Omnicell password", d.omnicellPassword.QWidget)
	d.omnicellSource = lineEdit()
	form.AddRow3("Omnicell source", d.omnicellSource.QWidget)
	d.omnicellDefaultProfile = qt.NewQComboBox2()
	d.omnicellDefaultProfile.AddItems([]string{simcommands.ProfileMCAGSM4, simcommands.ProfileMCAGSM, simcommands.ProfileFreeSMS})
	form.AddRow3("SMS профіль за замовчуванням", d.omnicellDefaultProfile.QWidget)
	d.omnicellPrimaryAPN = lineEdit()
	form.AddRow3("МЦА APN основний", d.omnicellPrimaryAPN.QWidget)
	d.omnicellReserveAPN = lineEdit()
	form.AddRow3("МЦА APN резервний", d.omnicellReserveAPN.QWidget)
	d.omnicellPrimaryIP = lineEdit()
	form.AddRow3("МЦА IP основний", d.omnicellPrimaryIP.QWidget)
	d.omnicellReserveIP = lineEdit()
	form.AddRow3("МЦА IP резервний", d.omnicellReserveIP.QWidget)
	d.omnicellPrimaryModulePort = spinBox(1, 9999)
	form.AddRow3("МЦА порт модуля основний", d.omnicellPrimaryModulePort.QWidget)
	d.omnicellReserveModulePort = spinBox(1, 9999)
	form.AddRow3("МЦА порт модуля резервний", d.omnicellReserveModulePort.QWidget)
	d.omnicellPrimaryReceiverPort = spinBox(1, 9999)
	form.AddRow3("МЦА порт ПЦПС основний", d.omnicellPrimaryReceiverPort.QWidget)
	d.omnicellReserveReceiverPort = spinBox(1, 9999)
	form.AddRow3("МЦА порт ПЦПС резервний", d.omnicellReserveReceiverPort.QWidget)
	d.omnicellPrimaryInterval = spinBox(1, 240)
	form.AddRow3("МЦА тест основний, хв", d.omnicellPrimaryInterval.QWidget)
	d.omnicellReserveInterval = spinBox(1, 240)
	form.AddRow3("МЦА тест резервний, хв", d.omnicellReserveInterval.QWidget)
	d.omnicellInput1Confirm = qt.NewQCheckBox3("Вхід 1: підтвердження")
	form.AddRow3("МЦА режим входу 1", d.omnicellInput1Confirm.QWidget)
	tabs.AddTab(wrapForm(form), "Omnicell")

	form = qt.NewQFormLayout2()
	d.amiEnabled = qt.NewQCheckBox3("Увімкнути AMI-команди")
	form.AddRow3("Asterisk AMI", d.amiEnabled.QWidget)
	d.amiHost = lineEdit()
	form.AddRow3("AMI host", d.amiHost.QWidget)
	d.amiPort = spinBox(1, 65535)
	form.AddRow3("AMI port", d.amiPort.QWidget)
	d.amiUsername = lineEdit()
	form.AddRow3("AMI user", d.amiUsername.QWidget)
	d.amiSecret = passwordEdit()
	form.AddRow3("AMI secret", d.amiSecret.QWidget)
	d.amiExtension = lineEdit()
	form.AddRow3("Operator extension", d.amiExtension.QWidget)
	d.amiContext = lineEdit()
	form.AddRow3("Dial context", d.amiContext.QWidget)
	tabs.AddTab(wrapForm(form), "AMI")

	layout.AddWidget(tabs.QWidget)
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
	d.phoenixControlHost.SetText(dbCfg.PhoenixControlHost)
	if config.NormalizePhoenixClientRole(dbCfg.PhoenixClientRole) == config.PhoenixClientRoleAdministrator {
		d.phoenixClientRole.SetCurrentText(qtPhoenixRoleAdminLabel)
	} else {
		d.phoenixClientRole.SetCurrentText(qtPhoenixRoleDutyLabel)
	}
	d.phoenixOperatorPassword.SetText(dbCfg.PhoenixOperatorPassword)

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
	dbCfg.PhoenixControlHost = d.phoenixControlHost.Text()
	dbCfg.PhoenixClientRole = config.PhoenixClientRoleDuty
	if d.phoenixClientRole.CurrentText() == qtPhoenixRoleAdminLabel {
		dbCfg.PhoenixClientRole = config.PhoenixClientRoleAdministrator
	}
	selectedOperator := d.phoenixOperatorsByLabel[d.phoenixOperator.CurrentText()]
	if selectedOperator.ID <= 0 {
		current := config.LoadDBConfig(d.prefs)
		selectedOperator.ID = current.PhoenixOperatorID
		selectedOperator.Login = current.PhoenixOperatorName
	}
	dbCfg.PhoenixOperatorID = selectedOperator.ID
	dbCfg.PhoenixOperatorName = strings.TrimSpace(selectedOperator.Login)
	if dbCfg.PhoenixOperatorName == "" {
		dbCfg.PhoenixOperatorName = strings.TrimSpace(selectedOperator.Name)
	}
	dbCfg.PhoenixOperatorPassword = d.phoenixOperatorPassword.Text()

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

func (d *settingsDialog) refreshPhoenixRuntimeMetadata() {
	if d == nil || d.phoenixRuntimeStatus == nil {
		return
	}
	cfg, _ := d.values()
	selectedID := cfg.PhoenixOperatorID
	if selectedID <= 0 {
		selectedID = config.LoadDBConfig(d.prefs).PhoenixOperatorID
	}
	d.phoenixRuntimeStatus.SetText("Завантаження портів і користувачів Phoenix...")

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	metadata, err := data.LoadPhoenixRuntimeMetadata(ctx, cfg)
	if err != nil {
		d.phoenixRuntimeStatus.SetText("Phoenix: " + err.Error())
		return
	}

	d.phoenixOperatorsByLabel = make(map[string]data.PhoenixOperator, len(metadata.Operators))
	d.phoenixOperator.Clear()
	selectedIndex := -1
	for _, operator := range metadata.Operators {
		label := operator.DisplayName()
		d.phoenixOperatorsByLabel[label] = operator
		d.phoenixOperator.AddItem(label)
		if operator.ID == selectedID {
			selectedIndex = d.phoenixOperator.Count() - 1
		}
	}
	if selectedIndex >= 0 {
		d.phoenixOperator.SetCurrentIndex(selectedIndex)
	}
	d.phoenixRuntimeStatus.SetText(fmt.Sprintf(
		"Порти з БД: Control Center %d, Duty Operator %d, Administrator %d, GPS %d. Користувачів: %d",
		metadata.ControlPort,
		metadata.ClientPort,
		metadata.AdminPort,
		metadata.GPSPort,
		len(metadata.Operators),
	))
}

func (d *settingsDialog) verifyPhoenixOperator() {
	if d == nil || d.phoenixRuntimeStatus == nil {
		return
	}
	cfg, _ := d.values()
	d.phoenixRuntimeStatus.SetText("Перевірка користувача Phoenix...")
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	if err := data.ValidatePhoenixOperatorCredentials(ctx, cfg); err != nil {
		d.phoenixRuntimeStatus.SetText("Phoenix: " + err.Error())
		return
	}
	d.phoenixRuntimeStatus.SetText("Користувача Phoenix підтверджено.")
}

func (d *settingsDialog) loadOperatorAndCommandSettings() {
	vf := config.LoadVodafoneConfig(d.prefs)
	d.vodafonePhone.SetText(vf.Phone)
	d.vodafoneAccessToken.SetText(vf.AccessToken)
	d.vodafoneTokenExpiry.SetText(vf.TokenExpiry)
	setComboTextFallback(d.vodafoneLoginMethod, vf.NormalizedLoginMethod(), config.VodafoneLoginMethodSMS)
	d.vodafonePUK.SetText(vf.PUK)
	d.vodafoneAutoReset.SetChecked(vf.AutoResetEnabled)
	d.vodafoneDailyLimit.SetValue(vf.AutoResetDailyLimit)
	d.vodafoneWindowHours.SetValue(vf.AutoResetWindowHours)

	ks := config.LoadKyivstarConfig(d.prefs)
	d.kyivstarClientID.SetText(ks.ClientID)
	d.kyivstarSecret.SetText(ks.ClientSecret)
	d.kyivstarEmail.SetText(ks.UserEmail)
	d.kyivstarAccessToken.SetText(ks.AccessToken)
	d.kyivstarTokenExpiry.SetText(ks.TokenExpiry)
	d.kyivstarAutoReset.SetChecked(ks.AutoResetEnabled)
	d.kyivstarDailyLimit.SetValue(ks.AutoResetDailyLimit)
	d.kyivstarWindowHours.SetValue(ks.AutoResetWindowHours)

	omni := config.LoadOmnicellConfig(d.prefs)
	d.omnicellEnabled.SetChecked(omni.Enabled)
	d.omnicellEndpoint.SetText(omni.Endpoint)
	d.omnicellLogin.SetText(omni.Login)
	d.omnicellPassword.SetText(omni.Password)
	d.omnicellSource.SetText(omni.Source)
	setComboTextFallback(d.omnicellDefaultProfile, omni.MCADefaultMessageProfile, simcommands.ProfileMCAGSM4)
	d.omnicellPrimaryAPN.SetText(omni.MCAPrimaryAPN)
	d.omnicellReserveAPN.SetText(omni.MCAReserveAPN)
	d.omnicellPrimaryIP.SetText(omni.MCAPrimaryIP)
	d.omnicellReserveIP.SetText(omni.MCAReserveIP)
	d.omnicellPrimaryModulePort.SetValue(omni.MCAPrimaryModulePort)
	d.omnicellReserveModulePort.SetValue(omni.MCAReserveModulePort)
	d.omnicellPrimaryReceiverPort.SetValue(omni.MCAPrimaryReceiverPort)
	d.omnicellReserveReceiverPort.SetValue(omni.MCAReserveReceiverPort)
	d.omnicellPrimaryInterval.SetValue(omni.MCAPrimaryTestInterval)
	d.omnicellReserveInterval.SetValue(omni.MCAReserveTestInterval)
	d.omnicellInput1Confirm.SetChecked(omni.MCAInput1ConfirmMode)

	amiEnabled, amiCfg := config.LoadAMIConfig(d.prefs)
	d.amiEnabled.SetChecked(amiEnabled)
	d.amiHost.SetText(amiCfg.Host)
	d.amiPort.SetValue(amiCfg.Port)
	d.amiUsername.SetText(amiCfg.Username)
	d.amiSecret.SetText(amiCfg.Secret)
	d.amiExtension.SetText(amiCfg.Extension)
	d.amiContext.SetText(amiCfg.Context)
}

func (d *settingsDialog) saveOperatorAndCommandSettings() {
	config.SaveVodafoneConfig(d.prefs, config.VodafoneConfig{
		Phone:                d.vodafonePhone.Text(),
		AccessToken:          d.vodafoneAccessToken.Text(),
		TokenExpiry:          d.vodafoneTokenExpiry.Text(),
		LoginMethod:          d.vodafoneLoginMethod.CurrentText(),
		PUK:                  d.vodafonePUK.Text(),
		AutoResetEnabled:     d.vodafoneAutoReset.IsChecked(),
		AutoResetDailyLimit:  d.vodafoneDailyLimit.Value(),
		AutoResetWindowHours: d.vodafoneWindowHours.Value(),
	})
	config.SaveKyivstarConfig(d.prefs, config.KyivstarConfig{
		ClientID:             d.kyivstarClientID.Text(),
		ClientSecret:         d.kyivstarSecret.Text(),
		UserEmail:            d.kyivstarEmail.Text(),
		AccessToken:          d.kyivstarAccessToken.Text(),
		TokenExpiry:          d.kyivstarTokenExpiry.Text(),
		AutoResetEnabled:     d.kyivstarAutoReset.IsChecked(),
		AutoResetDailyLimit:  d.kyivstarDailyLimit.Value(),
		AutoResetWindowHours: d.kyivstarWindowHours.Value(),
	})
	config.SaveOmnicellConfig(d.prefs, config.OmnicellConfig{
		Enabled:                  d.omnicellEnabled.IsChecked(),
		Endpoint:                 d.omnicellEndpoint.Text(),
		Login:                    d.omnicellLogin.Text(),
		Password:                 d.omnicellPassword.Text(),
		Source:                   d.omnicellSource.Text(),
		MCADefaultMessageProfile: d.omnicellDefaultProfile.CurrentText(),
		MCAPrimaryAPN:            d.omnicellPrimaryAPN.Text(),
		MCAReserveAPN:            d.omnicellReserveAPN.Text(),
		MCAPrimaryIP:             d.omnicellPrimaryIP.Text(),
		MCAReserveIP:             d.omnicellReserveIP.Text(),
		MCAPrimaryModulePort:     d.omnicellPrimaryModulePort.Value(),
		MCAReserveModulePort:     d.omnicellReserveModulePort.Value(),
		MCAPrimaryReceiverPort:   d.omnicellPrimaryReceiverPort.Value(),
		MCAReserveReceiverPort:   d.omnicellReserveReceiverPort.Value(),
		MCAPrimaryTestInterval:   d.omnicellPrimaryInterval.Value(),
		MCAReserveTestInterval:   d.omnicellReserveInterval.Value(),
		MCAInput1ConfirmMode:     d.omnicellInput1Confirm.IsChecked(),
	})
	config.SaveAMIConfig(d.prefs, d.amiEnabled.IsChecked(), ami.Config{
		Host:      d.amiHost.Text(),
		Port:      d.amiPort.Value(),
		Username:  d.amiUsername.Text(),
		Secret:    d.amiSecret.Text(),
		Extension: d.amiExtension.Text(),
		Context:   d.amiContext.Text(),
	})
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

func wrapForm(form *qt.QFormLayout) *qt.QWidget {
	content := qt.NewQWidget2()
	content.SetLayout(form.QLayout)

	scroll := qt.NewQScrollArea2()
	scroll.SetWidgetResizable(true)
	scroll.SetWidget(content)
	return scroll.QWidget
}

func setComboText(combo *qt.QComboBox, value string) {
	setComboTextFallback(combo, value, "info")
}

func setComboTextFallback(combo *qt.QComboBox, value string, fallback string) {
	idx := combo.FindText(value)
	if idx < 0 {
		idx = combo.FindText(fallback)
	}
	if idx >= 0 {
		combo.SetCurrentIndex(idx)
	}
}
