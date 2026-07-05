package dialogs

import (
	"context"
	"fmt"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"runtime"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type settingsDialogAdminProvider interface {
	GetVodafoneAuthState() (contracts.VodafoneAuthState, error)
	RequestVodafoneLoginSMS(phone string) error
	VerifyVodafoneLogin(phone string, code string) (contracts.VodafoneAuthState, error)
	ClearVodafoneLogin() error
	GetKyivstarAuthState() (contracts.KyivstarAuthState, error)
	RefreshKyivstarToken() (contracts.KyivstarAuthState, error)
	ClearKyivstarToken() error
}

const (
	phoenixRoleDutyLabel  = "Черговий оператор (порт Duty Operator)"
	phoenixRoleAdminLabel = "Адміністратор (порт Administrator)"
)

type settingsDialogState struct {
	// ... existing fields ...
	dialogCtx       context.Context
	dialogCancel    context.CancelFunc
	win             fyne.Window
	pref            fyne.Preferences
	adminProvider   settingsDialogAdminProvider
	isDarkTheme     bool
	onSave          func(config.DBConfig, config.UIConfig)
	onColorsChanged func()

	dbCfg config.DBConfig
	uiCfg config.UIConfig
	vfCfg config.VodafoneConfig
	ksCfg config.KyivstarConfig

	vfAuthVM *viewmodels.VodafoneAuthViewModel
	ksAuthVM *viewmodels.KyivstarAuthViewModel

	userEntry                    *widget.Entry
	passEntry                    *widget.Entry
	hostEntry                    *widget.Entry
	portEntry                    *widget.Entry
	pathEntry                    *widget.Entry
	paramsEntry                  *widget.Entry
	firebirdEnabledCheck         *widget.Check
	phoenixEnabledCheck          *widget.Check
	phoenixUserEntry             *widget.Entry
	phoenixPassEntry             *widget.Entry
	phoenixHostEntry             *widget.Entry
	phoenixPortEntry             *widget.Entry
	phoenixInstanceEntry         *widget.Entry
	phoenixDatabaseEntry         *widget.Entry
	phoenixParamsEntry           *widget.Entry
	phoenixControlHostEntry      *widget.Entry
	phoenixClientRoleSelect      *widget.Select
	phoenixOperatorSelect        *widget.Select
	phoenixOperatorPassEntry     *widget.Entry
	phoenixRuntimeStatusLabel    *widget.Label
	phoenixOperatorsByLabel      map[string]data.PhoenixOperator
	caslBaseURLEntry             *widget.Entry
	caslTokenEntry               *widget.Entry
	caslEmailEntry               *widget.Entry
	caslPassEntry                *widget.Entry
	caslPultIDEntry              *widget.Entry
	caslEnabledCheck             *widget.Check
	vodafonePhoneEntry           *widget.Entry
	vodafoneLoginMethodRadio     *widget.RadioGroup
	vodafoneCodeEntry            *widget.Entry
	vodafonePUKEntry             *widget.Entry
	vodafoneAutoResetCheck       *widget.Check
	vodafoneAutoResetLimitEntry  *widget.Entry
	vodafoneAutoResetWindowEntry *widget.Entry
	vodafoneStatusLabel          *widget.Label
	kyivstarClientIDEntry        *widget.Entry
	kyivstarClientSecretEntry    *widget.Entry
	kyivstarEmailEntry           *widget.Entry
	kyivstarAutoResetCheck       *widget.Check
	kyivstarAutoResetLimitEntry  *widget.Entry
	kyivstarAutoResetWindowEntry *widget.Entry
	kyivstarStatusLabel          *widget.Label
	fontEntry                    *widget.Entry
	fontObjEntry                 *widget.Entry
	fontEvEntry                  *widget.Entry
	fontAlmEntry                 *widget.Entry
	bottomAlarmJournalCheck      *widget.Check
	bottomEventJournalCheck      *widget.Check
	eventLimitEntry              *widget.Entry
	objectLimitEntry             *widget.Entry
	bridgeHistoryModeSelect      *widget.Select
	eventProbeIntervalEntry      *widget.Entry
	eventsReconcileEntry         *widget.Entry
	alarmsReconcileEntry         *widget.Entry
	objectsReconcileEntry        *widget.Entry
	fallbackRefreshEntry         *widget.Entry
	maxProbeBackoffEntry         *widget.Entry
	schedulerHelpLabel           *widget.Label
	exportDirEntry               *widget.Entry
	logLevelSelect               *widget.Select
}

func ShowSettingsDialog(
	win fyne.Window,
	adminProvider settingsDialogAdminProvider,
	pref fyne.Preferences,
	isDarkTheme bool,
	onSave func(config.DBConfig, config.UIConfig),
	onColorsChanged func(),
) {
	state := newSettingsDialogState(win, pref, adminProvider, isDarkTheme, onSave, onColorsChanged)
	dialog := state.buildDialog()
	state.refreshVodafoneStatus()
	state.refreshKyivstarStatus()
	state.refreshPhoenixRuntimeMetadata()
	dialog.Show()
}

func newSettingsDialogState(
	win fyne.Window,
	pref fyne.Preferences,
	adminProvider settingsDialogAdminProvider,
	isDarkTheme bool,
	onSave func(config.DBConfig, config.UIConfig),
	onColorsChanged func(),
) *settingsDialogState {
	ctx, cancel := context.WithCancel(context.Background())
	s := &settingsDialogState{
		win:             win,
		pref:            pref,
		adminProvider:   adminProvider,
		isDarkTheme:     isDarkTheme,
		dialogCtx:       ctx,
		dialogCancel:    cancel,
		onSave:          onSave,
		onColorsChanged: onColorsChanged,
		dbCfg:           config.LoadDBConfig(pref),
		uiCfg:           config.LoadUIConfig(pref),
		vfCfg:           config.LoadVodafoneConfig(pref),
		ksCfg:           config.LoadKyivstarConfig(pref),
		vfAuthVM:        viewmodels.NewVodafoneAuthViewModel(),
		ksAuthVM:        viewmodels.NewKyivstarAuthViewModel(),
	}

	s.initDatabaseFields()
	s.initCarrierFields()
	s.initUIFields()

	return s
}

func (s *settingsDialogState) initDatabaseFields() {
	s.userEntry = widget.NewEntry()
	s.userEntry.SetText(s.dbCfg.User)
	s.passEntry = widget.NewPasswordEntry()
	s.passEntry.SetText(s.dbCfg.Password)
	s.hostEntry = widget.NewEntry()
	s.hostEntry.SetText(s.dbCfg.Host)
	s.portEntry = widget.NewEntry()
	s.portEntry.SetText(s.dbCfg.Port)
	s.pathEntry = widget.NewEntry()
	s.pathEntry.SetText(s.dbCfg.Path)
	s.paramsEntry = widget.NewEntry()
	s.paramsEntry.SetText(s.dbCfg.Params)

	s.firebirdEnabledCheck = widget.NewCheck("Увімкнути БД/МІСТ (Firebird)", nil)
	s.firebirdEnabledCheck.SetChecked(
		s.dbCfg.FirebirdEnabled ||
			(!s.dbCfg.FirebirdEnabled && !s.dbCfg.PhoenixEnabled && s.dbCfg.NormalizedMode() != config.BackendModePhoenix),
	)

	s.phoenixEnabledCheck = widget.NewCheck("Увімкнути Phoenix паралельно з іншими джерелами", nil)
	s.phoenixEnabledCheck.SetChecked(s.dbCfg.PhoenixEnabled || s.dbCfg.NormalizedMode() == config.BackendModePhoenix)
	s.phoenixUserEntry = widget.NewEntry()
	s.phoenixUserEntry.SetText(s.dbCfg.PhoenixUser)
	s.phoenixPassEntry = widget.NewPasswordEntry()
	s.phoenixPassEntry.SetText(s.dbCfg.PhoenixPassword)
	s.phoenixHostEntry = widget.NewEntry()
	s.phoenixHostEntry.SetText(s.dbCfg.PhoenixHost)
	s.phoenixPortEntry = widget.NewEntry()
	s.phoenixPortEntry.SetText(s.dbCfg.PhoenixPort)
	s.phoenixInstanceEntry = widget.NewEntry()
	s.phoenixInstanceEntry.SetText(s.dbCfg.PhoenixInstance)
	s.phoenixDatabaseEntry = widget.NewEntry()
	s.phoenixDatabaseEntry.SetText(s.dbCfg.PhoenixDatabase)
	s.phoenixParamsEntry = widget.NewEntry()
	s.phoenixParamsEntry.SetText(s.dbCfg.PhoenixParams)
	s.phoenixControlHostEntry = widget.NewEntry()
	s.phoenixControlHostEntry.SetText(s.dbCfg.PhoenixControlHost)
	s.phoenixControlHostEntry.SetPlaceHolder("IP або DNS Phoenix Control Center")
	s.phoenixClientRoleSelect = widget.NewSelect([]string{
		phoenixRoleDutyLabel,
		phoenixRoleAdminLabel,
	}, nil)
	if config.NormalizePhoenixClientRole(s.dbCfg.PhoenixClientRole) == config.PhoenixClientRoleAdministrator {
		s.phoenixClientRoleSelect.SetSelected(phoenixRoleAdminLabel)
	} else {
		s.phoenixClientRoleSelect.SetSelected(phoenixRoleDutyLabel)
	}
	s.phoenixOperatorSelect = widget.NewSelect(nil, nil)
	s.phoenixOperatorSelect.PlaceHolder = "Завантажте користувачів із БД"
	s.phoenixOperatorPassEntry = widget.NewPasswordEntry()
	s.phoenixOperatorPassEntry.SetText(s.dbCfg.PhoenixOperatorPassword)
	s.phoenixRuntimeStatusLabel = widget.NewLabel("Порти та користувачі ще не завантажені")
	s.phoenixRuntimeStatusLabel.Wrapping = fyne.TextWrapWord
	s.phoenixOperatorsByLabel = make(map[string]data.PhoenixOperator)

	s.caslBaseURLEntry = widget.NewEntry()
	s.caslBaseURLEntry.SetText(strings.TrimSpace(s.dbCfg.CASLBaseURL))
	s.caslBaseURLEntry.SetPlaceHolder("http://10.32.1.221:50003")

	s.caslTokenEntry = widget.NewEntry()
	s.caslTokenEntry.SetText(strings.TrimSpace(s.dbCfg.CASLToken))
	s.caslTokenEntry.SetPlaceHolder("JWT токен (необов'язково)")

	s.caslEmailEntry = widget.NewEntry()
	s.caslEmailEntry.SetText(strings.TrimSpace(s.dbCfg.CASLEmail))
	s.caslEmailEntry.SetPlaceHolder("test@lot.lviv.ua")

	s.caslPassEntry = widget.NewPasswordEntry()
	s.caslPassEntry.SetText(strings.TrimSpace(s.dbCfg.CASLPass))
	s.caslPassEntry.SetPlaceHolder("Пароль CASL")

	s.caslPultIDEntry = widget.NewEntry()
	if s.dbCfg.CASLPultID > 0 {
		s.caslPultIDEntry.SetText(strconv.FormatInt(s.dbCfg.CASLPultID, 10))
	}
	s.caslPultIDEntry.SetPlaceHolder("0 = авто")

	s.caslEnabledCheck = widget.NewCheck("Увімкнути CASL Cloud паралельно з БД/мостом", nil)
	s.caslEnabledCheck.SetChecked(s.dbCfg.CASLEnabled || s.dbCfg.NormalizedMode() == config.BackendModeCASLCloud)
}

func (s *settingsDialogState) initCarrierFields() {
	s.vodafonePhoneEntry = widget.NewEntry()
	s.vodafonePhoneEntry.SetText(strings.TrimSpace(s.vfCfg.Phone))
	s.vodafonePhoneEntry.SetPlaceHolder("380501234567")

	s.vodafoneLoginMethodRadio = widget.NewRadioGroup([]string{"SMS-код", "PUK-код"}, nil)
	s.vodafoneLoginMethodRadio.Horizontal = true
	s.vodafoneLoginMethodRadio.Required = true
	if s.vfCfg.NormalizedLoginMethod() == config.VodafoneLoginMethodPUK {
		s.vodafoneLoginMethodRadio.SetSelected("PUK-код")
	} else {
		s.vodafoneLoginMethodRadio.SetSelected("SMS-код")
	}
	s.vodafoneLoginMethodRadio.OnChanged = func(string) {
		s.applyVodafoneLoginMethodState()
		s.vodafoneStatusLabel.SetText(s.vfAuthVM.BuildStatusText(s.currentVodafoneAuthState()))
	}

	s.vodafoneCodeEntry = widget.NewPasswordEntry()
	s.vodafoneCodeEntry.SetPlaceHolder("SMS-код")

	s.vodafonePUKEntry = widget.NewPasswordEntry()
	s.vodafonePUKEntry.SetText(strings.TrimSpace(s.vfCfg.PUK))
	s.vodafonePUKEntry.SetPlaceHolder("PUK-код")

	s.vodafoneAutoResetCheck = widget.NewCheck("Автоматично надсилати reset sim для offline MOST з Vodafone у SIM1", nil)
	s.vodafoneAutoResetCheck.SetChecked(s.vfCfg.AutoResetEnabled)

	s.vodafoneAutoResetLimitEntry = widget.NewEntry()
	s.vodafoneAutoResetLimitEntry.SetText(strconv.Itoa(s.vfCfg.AutoResetDailyLimit))
	s.vodafoneAutoResetLimitEntry.SetPlaceHolder(strconv.Itoa(config.DefaultVodafoneAutoResetDailyLimit))

	s.vodafoneAutoResetWindowEntry = widget.NewEntry()
	s.vodafoneAutoResetWindowEntry.SetText(strconv.Itoa(s.vfCfg.AutoResetWindowHours))
	s.vodafoneAutoResetWindowEntry.SetPlaceHolder(strconv.Itoa(config.DefaultVodafoneAutoResetWindowHours))

	s.vodafoneStatusLabel = widget.NewLabel(s.vfAuthVM.BuildStatusText(s.currentVodafoneAuthState()))
	s.vodafoneStatusLabel.Wrapping = fyne.TextWrapWord
	s.applyVodafoneLoginMethodState()

	s.kyivstarClientIDEntry = widget.NewEntry()
	s.kyivstarClientIDEntry.SetText(strings.TrimSpace(s.ksCfg.ClientID))
	s.kyivstarClientIDEntry.SetPlaceHolder("client_id")

	s.kyivstarClientSecretEntry = widget.NewPasswordEntry()
	s.kyivstarClientSecretEntry.SetText(strings.TrimSpace(s.ksCfg.ClientSecret))
	s.kyivstarClientSecretEntry.SetPlaceHolder("client_secret")

	s.kyivstarEmailEntry = widget.NewEntry()
	s.kyivstarEmailEntry.SetText(strings.TrimSpace(s.ksCfg.UserEmail))
	s.kyivstarEmailEntry.SetPlaceHolder("company.user@domain.ua")

	s.kyivstarAutoResetCheck = widget.NewCheck("Автоматично надсилати reset sim для offline MOST з Kyivstar у SIM1", nil)
	s.kyivstarAutoResetCheck.SetChecked(s.ksCfg.AutoResetEnabled)

	s.kyivstarAutoResetLimitEntry = widget.NewEntry()
	s.kyivstarAutoResetLimitEntry.SetText(strconv.Itoa(s.ksCfg.AutoResetDailyLimit))
	s.kyivstarAutoResetLimitEntry.SetPlaceHolder(strconv.Itoa(config.DefaultKyivstarAutoResetDailyLimit))

	s.kyivstarAutoResetWindowEntry = widget.NewEntry()
	s.kyivstarAutoResetWindowEntry.SetText(strconv.Itoa(s.ksCfg.AutoResetWindowHours))
	s.kyivstarAutoResetWindowEntry.SetPlaceHolder(strconv.Itoa(config.DefaultKyivstarAutoResetWindowHours))

	s.kyivstarStatusLabel = widget.NewLabel(s.ksAuthVM.BuildStatusText(s.currentKyivstarAuthState()))
	s.kyivstarStatusLabel.Wrapping = fyne.TextWrapWord
}

func (s *settingsDialogState) initUIFields() {
	s.fontEntry = widget.NewEntry()
	s.fontEntry.SetText(fmt.Sprintf("%.1f", s.uiCfg.FontSize))
	s.fontObjEntry = widget.NewEntry()
	s.fontObjEntry.SetText(fmt.Sprintf("%.1f", s.uiCfg.FontSizeObjects))
	s.fontEvEntry = widget.NewEntry()
	s.fontEvEntry.SetText(fmt.Sprintf("%.1f", s.uiCfg.FontSizeEvents))
	s.fontAlmEntry = widget.NewEntry()
	s.fontAlmEntry.SetText(fmt.Sprintf("%.1f", s.uiCfg.FontSizeAlarms))

	s.bottomAlarmJournalCheck = widget.NewCheck("Показувати журнал активних тривог знизу на всю ширину", nil)
	s.bottomAlarmJournalCheck.SetChecked(s.uiCfg.ShowBottomAlarmJournal)

	s.bottomEventJournalCheck = widget.NewCheck("Показувати загальний журнал знизу на всю ширину", nil)
	s.bottomEventJournalCheck.SetChecked(s.uiCfg.ShowBottomEventJournal)

	s.eventLimitEntry = widget.NewEntry()
	s.eventLimitEntry.SetText(strconv.Itoa(s.uiCfg.EventLogLimit))
	s.eventLimitEntry.SetPlaceHolder("2000")

	s.objectLimitEntry = widget.NewEntry()
	s.objectLimitEntry.SetText(strconv.Itoa(s.uiCfg.ObjectLogLimit))
	s.objectLimitEntry.SetPlaceHolder("0 = без обмеження")

	s.bridgeHistoryModeSelect = widget.NewSelect(bridgeAlarmHistoryModeOptions(), nil)
	s.bridgeHistoryModeSelect.SetSelected(bridgeAlarmHistoryModeLabel(s.uiCfg.BridgeAlarmHistoryMode))

	s.eventProbeIntervalEntry = widget.NewEntry()
	s.eventProbeIntervalEntry.SetText(strconv.Itoa(s.uiCfg.EventProbeIntervalSec))
	s.eventProbeIntervalEntry.SetPlaceHolder(strconv.Itoa(config.DefaultEventProbeIntervalSec))

	s.eventsReconcileEntry = widget.NewEntry()
	s.eventsReconcileEntry.SetText(strconv.Itoa(s.uiCfg.EventsReconcileSec))
	s.eventsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultEventsReconcileSec))

	s.alarmsReconcileEntry = widget.NewEntry()
	s.alarmsReconcileEntry.SetText(strconv.Itoa(s.uiCfg.AlarmsReconcileSec))
	s.alarmsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultAlarmsReconcileSec))

	s.objectsReconcileEntry = widget.NewEntry()
	s.objectsReconcileEntry.SetText(strconv.Itoa(s.uiCfg.ObjectsReconcileSec))
	s.objectsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultObjectsReconcileSec))

	s.fallbackRefreshEntry = widget.NewEntry()
	s.fallbackRefreshEntry.SetText(strconv.Itoa(s.uiCfg.FallbackRefreshSec))
	s.fallbackRefreshEntry.SetPlaceHolder(strconv.Itoa(config.DefaultFallbackRefreshSec))

	s.maxProbeBackoffEntry = widget.NewEntry()
	s.maxProbeBackoffEntry.SetText(strconv.Itoa(s.uiCfg.MaxProbeBackoffSec))
	s.maxProbeBackoffEntry.SetPlaceHolder(strconv.Itoa(config.DefaultMaxProbeBackoffSec))

	s.schedulerHelpLabel = widget.NewLabel("Оновлення Firebird, сек. Менші значення роблять інтерфейс актуальнішим, але сильніше навантажують сервер.")
	s.schedulerHelpLabel.Wrapping = fyne.TextWrapWord

	s.exportDirEntry = widget.NewEntry()
	s.exportDirEntry.SetText(s.uiCfg.ExportDir)
	s.exportDirEntry.SetPlaceHolder("Папка запуску програми")

	s.logLevelSelect = widget.NewSelect([]string{"debug", "info", "warn", "error"}, nil)
	s.logLevelSelect.SetSelected(strings.ToLower(strings.TrimSpace(s.dbCfg.LogLevel)))
	if s.logLevelSelect.Selected == "" {
		s.logLevelSelect.SetSelected("info")
	}
}

func (s *settingsDialogState) buildDialog() dialog.Dialog {
	d := dialog.NewCustomConfirm(
		"Налаштування системи",
		"Зберегти",
		"Скасувати",
		s.buildTabs(),
		func(save bool) {
			if !save {
				s.dialogCancel()
				return
			}
			s.applySave()
		},
		s.win,
	)
	d.Resize(fyne.NewSize(560, 520))
	return d
}

func (s *settingsDialogState) buildTabs() *container.AppTabs {
	return container.NewAppTabs(
		container.NewTabItem("База даних", s.buildDatabaseTab()),
		container.NewTabItem("Phoenix", s.buildPhoenixTab()),
		container.NewTabItem("CASL Cloud", s.buildCASLTab()),
		container.NewTabItem("Vodafone", s.buildVodafoneTab()),
		container.NewTabItem("Kyivstar", s.buildKyivstarTab()),
		container.NewTabItem("Інтерфейс", s.buildInterfaceTab()),
		container.NewTabItem("Оновлення", s.buildRefreshTab()),
	)
}

func (s *settingsDialogState) buildDatabaseTab() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Увімкнення", s.firebirdEnabledCheck),
		widget.NewFormItem("Користувач", s.userEntry),
		widget.NewFormItem("Пароль", s.passEntry),
		widget.NewFormItem("Хост", s.hostEntry),
		widget.NewFormItem("Порт", s.portEntry),
		widget.NewFormItem("Шлях до БД", s.pathEntry),
		widget.NewFormItem("Параметри", s.paramsEntry),
	)
}

func (s *settingsDialogState) buildPhoenixTab() fyne.CanvasObject {
	form := widget.NewForm(
		widget.NewFormItem("Увімкнення", s.phoenixEnabledCheck),
		widget.NewFormItem("SQL користувач", s.phoenixUserEntry),
		widget.NewFormItem("SQL пароль", s.phoenixPassEntry),
		widget.NewFormItem("SQL хост", s.phoenixHostEntry),
		widget.NewFormItem("SQL порт", s.phoenixPortEntry),
		widget.NewFormItem("Інстанс", s.phoenixInstanceEntry),
		widget.NewFormItem("База", s.phoenixDatabaseEntry),
		widget.NewFormItem("Параметри", s.phoenixParamsEntry),
		widget.NewFormItem("Центр керування", s.phoenixControlHostEntry),
		widget.NewFormItem("Роль клієнта", s.phoenixClientRoleSelect),
		widget.NewFormItem("Оператор", s.phoenixOperatorSelect),
		widget.NewFormItem("Пароль оператора", s.phoenixOperatorPassEntry),
	)
	return container.NewVBox(
		form,
		container.NewHBox(
			widget.NewButton("Оновити з БД", s.refreshPhoenixRuntimeMetadata),
			widget.NewButton("Перевірити оператора", s.verifyPhoenixOperator),
		),
		s.phoenixRuntimeStatusLabel,
	)
}

func (s *settingsDialogState) buildCASLTab() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Паралельний режим", s.caslEnabledCheck),
		widget.NewFormItem("Base URL", s.caslBaseURLEntry),
		widget.NewFormItem("Token", s.caslTokenEntry),
		widget.NewFormItem("Email", s.caslEmailEntry),
		widget.NewFormItem("Password", s.caslPassEntry),
		widget.NewFormItem("Pult ID", s.caslPultIDEntry),
	)
}

func (s *settingsDialogState) buildVodafoneTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Авторизація Vodafone для батьківського номера. PUK-код зберігається локально і використовується для автоматичного поновлення токена."),
		widget.NewForm(
			widget.NewFormItem("Тип входу", s.vodafoneLoginMethodRadio),
			widget.NewFormItem("Номер входу", s.vodafonePhoneEntry),
			widget.NewFormItem("SMS/PUK для входу", s.vodafoneCodeEntry),
			widget.NewFormItem("Збережений PUK", s.vodafonePUKEntry),
			widget.NewFormItem("Авто reset SIM", s.vodafoneAutoResetCheck),
			widget.NewFormItem("Ліміт за вікно", s.vodafoneAutoResetLimitEntry),
			widget.NewFormItem("Вікно throttle, год", s.vodafoneAutoResetWindowEntry),
		),
		container.NewHBox(
			widget.NewButton("Надіслати SMS", s.handleVodafoneSMSRequest),
			widget.NewButton("Увійти", s.handleVodafoneCodeVerify),
			widget.NewButton("Очистити токен", s.handleVodafoneTokenClear),
		),
		s.vodafoneStatusLabel,
	)
}

func (s *settingsDialogState) buildKyivstarTab() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabel("Kyivstar IoT API використовує client_id/client_secret і email компанії для reset запитів."),
		widget.NewForm(
			widget.NewFormItem("Client ID", s.kyivstarClientIDEntry),
			widget.NewFormItem("Client Secret", s.kyivstarClientSecretEntry),
			widget.NewFormItem("Email компанії", s.kyivstarEmailEntry),
			widget.NewFormItem("Авто reset SIM", s.kyivstarAutoResetCheck),
			widget.NewFormItem("Ліміт за вікно", s.kyivstarAutoResetLimitEntry),
			widget.NewFormItem("Вікно throttle, год", s.kyivstarAutoResetWindowEntry),
		),
		container.NewHBox(
			widget.NewButton("Отримати токен", s.handleKyivstarTokenRefresh),
			widget.NewButton("Очистити токен", s.handleKyivstarTokenClear),
		),
		s.kyivstarStatusLabel,
	)
}

func (s *settingsDialogState) buildInterfaceTab() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Загальний шрифт", s.fontEntry),
		widget.NewFormItem("Шрифт об'єктів", s.fontObjEntry),
		widget.NewFormItem("Шрифт подій", s.fontEvEntry),
		widget.NewFormItem("Шрифт тривог", s.fontAlmEntry),
		widget.NewFormItem("Нижній журнал тривог", s.bottomAlarmJournalCheck),
		widget.NewFormItem("Нижній загальний журнал", s.bottomEventJournalCheck),
		widget.NewFormItem("Режим логування", s.logLevelSelect),
		widget.NewFormItem("Ліміт загального журналу", s.eventLimitEntry),
		widget.NewFormItem("Ліміт журналу об'єкта", s.objectLimitEntry),
		widget.NewFormItem("Хронологія МІСТ", s.bridgeHistoryModeSelect),
		widget.NewFormItem("Папка експорту", s.buildExportDirRow()),
		widget.NewFormItem("Кольори подій", s.buildColorsButton()),
	)
}

func (s *settingsDialogState) buildRefreshTab() fyne.CanvasObject {
	return widget.NewForm(
		widget.NewFormItem("Пояснення", s.schedulerHelpLabel),
		widget.NewFormItem("Probe нових подій", s.eventProbeIntervalEntry),
		widget.NewFormItem("Reconcile журналу", s.eventsReconcileEntry),
		widget.NewFormItem("Reconcile тривог", s.alarmsReconcileEntry),
		widget.NewFormItem("Reconcile об'єктів", s.objectsReconcileEntry),
		widget.NewFormItem("Fallback без probe", s.fallbackRefreshEntry),
		widget.NewFormItem("Макс. backoff probe", s.maxProbeBackoffEntry),
	)
}

func (s *settingsDialogState) buildExportDirRow() fyne.CanvasObject {
	browseExportDirBtn := makeIconButton("Обрати...", iconFolder(), widget.MediumImportance, func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, s.win)
				return
			}
			if uri == nil {
				return
			}
			s.exportDirEntry.SetText(uriPathToLocalPath(uri.Path()))
		}, s.win).Show()
	})

	clearExportDirBtn := makeIconButton("Очистити", iconClear(), widget.LowImportance, func() {
		s.exportDirEntry.SetText("")
	})

	return container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(browseExportDirBtn, clearExportDirBtn),
		s.exportDirEntry,
	)
}

func (s *settingsDialogState) buildColorsButton() fyne.CanvasObject {
	return makeIconButton("Налаштувати кольори подій...", iconSearch(), widget.LowImportance, func() {
		ShowColorPaletteDialog(s.win, s.isDarkTheme, s.onColorsChanged)
	})
}

func (s *settingsDialogState) currentVodafoneAuthState() contracts.VodafoneAuthState {
	return contracts.VodafoneAuthState{
		Phone:          strings.TrimSpace(s.vodafonePhoneEntry.Text),
		LoginMethod:    s.selectedVodafoneLoginMethod(),
		PUKConfigured:  s.vodafonePUKEntry != nil && strings.TrimSpace(s.vodafonePUKEntry.Text) != "",
		Authorized:     s.vfCfg.TokenUsableAt(timeNow()),
		TokenExpiresAt: s.vfCfg.TokenExpiryTime(),
	}
}

func (s *settingsDialogState) selectedVodafoneLoginMethod() string {
	if s == nil || s.vodafoneLoginMethodRadio == nil {
		return config.VodafoneLoginMethodSMS
	}
	if strings.TrimSpace(s.vodafoneLoginMethodRadio.Selected) == "PUK-код" {
		return config.VodafoneLoginMethodPUK
	}
	return config.VodafoneLoginMethodSMS
}

func (s *settingsDialogState) applyVodafoneLoginMethodState() {
	if s == nil || s.vodafoneCodeEntry == nil || s.vodafonePUKEntry == nil {
		return
	}
	if s.selectedVodafoneLoginMethod() == config.VodafoneLoginMethodPUK {
		s.vodafoneCodeEntry.SetPlaceHolder("PUK-код для входу")
		s.vodafonePUKEntry.Enable()
		return
	}
	s.vodafoneCodeEntry.SetPlaceHolder("SMS-код")
	s.vodafonePUKEntry.Disable()
}

func (s *settingsDialogState) currentKyivstarAuthState() contracts.KyivstarAuthState {
	return contracts.KyivstarAuthState{
		ClientID:       strings.TrimSpace(s.kyivstarClientIDEntry.Text),
		UserEmail:      strings.TrimSpace(s.kyivstarEmailEntry.Text),
		Configured:     strings.TrimSpace(s.kyivstarClientIDEntry.Text) != "" && strings.TrimSpace(s.kyivstarClientSecretEntry.Text) != "",
		Authorized:     s.ksCfg.TokenUsableAt(timeNow()),
		TokenExpiresAt: s.ksCfg.TokenExpiryTime(),
	}
}

func (s *settingsDialogState) setVodafoneBusy(busy bool) {
	if busy {
		s.vodafonePhoneEntry.Disable()
		s.vodafoneLoginMethodRadio.Disable()
		s.vodafoneCodeEntry.Disable()
		s.vodafonePUKEntry.Disable()
		return
	}
	s.vodafonePhoneEntry.Enable()
	s.vodafoneLoginMethodRadio.Enable()
	s.vodafoneCodeEntry.Enable()
	s.vodafonePUKEntry.Enable()
	s.applyVodafoneLoginMethodState()
}

func (s *settingsDialogState) setKyivstarBusy(busy bool) {
	if busy {
		s.kyivstarClientIDEntry.Disable()
		s.kyivstarClientSecretEntry.Disable()
		s.kyivstarEmailEntry.Disable()
		return
	}
	s.kyivstarClientIDEntry.Enable()
	s.kyivstarClientSecretEntry.Enable()
	s.kyivstarEmailEntry.Enable()
}

func (s *settingsDialogState) refreshVodafoneStatus() {
	state := s.currentVodafoneAuthState()
	if s.adminProvider != nil {
		if liveState, err := s.adminProvider.GetVodafoneAuthState(); err == nil {
			state = liveState
			if strings.TrimSpace(liveState.Phone) != "" {
				s.vodafonePhoneEntry.SetText(strings.TrimSpace(liveState.Phone))
			}
			s.vfCfg.Phone = liveState.Phone
			s.vfCfg.LoginMethod = liveState.LoginMethod
			if liveState.LoginMethod == config.VodafoneLoginMethodPUK {
				s.vodafoneLoginMethodRadio.SetSelected("PUK-код")
			} else {
				s.vodafoneLoginMethodRadio.SetSelected("SMS-код")
			}
		}
	}
	s.vodafoneStatusLabel.SetText(s.vfAuthVM.BuildStatusText(state))
}

func (s *settingsDialogState) refreshKyivstarStatus() {
	state := s.currentKyivstarAuthState()
	if s.adminProvider != nil {
		if liveState, err := s.adminProvider.GetKyivstarAuthState(); err == nil {
			state = liveState
			if strings.TrimSpace(liveState.ClientID) != "" {
				s.kyivstarClientIDEntry.SetText(strings.TrimSpace(liveState.ClientID))
			}
			s.ksCfg.ClientID = liveState.ClientID
			s.ksCfg.UserEmail = liveState.UserEmail
		}
	}
	s.kyivstarStatusLabel.SetText(s.ksAuthVM.BuildStatusText(state))
}

func (s *settingsDialogState) handleVodafoneSMSRequest() {
	if s.adminProvider == nil {
		s.vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
		return
	}
	if s.selectedVodafoneLoginMethod() != config.VodafoneLoginMethodSMS {
		s.vodafoneStatusLabel.SetText("Vodafone: для PUK-входу SMS не потрібне")
		return
	}

	phone := strings.TrimSpace(s.vodafonePhoneEntry.Text)
	s.setVodafoneBusy(true)
	s.vodafoneStatusLabel.SetText("Vodafone: надсилання SMS-коду...")
	go func(ctx context.Context) {
		err := s.adminProvider.RequestVodafoneLoginSMS(phone)
		fyne.Do(func() {
			if ctx.Err() != nil {
				return // Dialog was closed, skip update
			}
			s.setVodafoneBusy(false)
			if err != nil {
				s.vodafoneStatusLabel.SetText(err.Error())
				return
			}
			s.vodafoneStatusLabel.SetText("Vodafone: SMS-код надіслано")
		})
	}(s.dialogCtx)
}

func (s *settingsDialogState) handleVodafoneCodeVerify() {
	if s.adminProvider == nil {
		s.vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
		return
	}

	phone := strings.TrimSpace(s.vodafonePhoneEntry.Text)
	code := strings.TrimSpace(s.vodafoneCodeEntry.Text)
	method := s.selectedVodafoneLoginMethod()
	if method == config.VodafoneLoginMethodPUK && code == "" {
		code = strings.TrimSpace(s.vodafonePUKEntry.Text)
	}
	currentCfg := config.LoadVodafoneConfig(s.pref)
	currentCfg.Phone = phone
	currentCfg.LoginMethod = method
	if method == config.VodafoneLoginMethodPUK {
		currentCfg.PUK = code
	}
	config.SaveVodafoneConfig(s.pref, currentCfg)
	s.vfCfg = currentCfg

	s.setVodafoneBusy(true)
	if method == config.VodafoneLoginMethodPUK {
		s.vodafoneStatusLabel.SetText("Vodafone: вхід через PUK...")
	} else {
		s.vodafoneStatusLabel.SetText("Vodafone: перевірка SMS-коду...")
	}
	go func() {
		state, err := s.adminProvider.VerifyVodafoneLogin(phone, code)
		fyne.Do(func() {
			s.setVodafoneBusy(false)
			if err != nil {
				s.vodafoneStatusLabel.SetText(err.Error())
				return
			}
			s.vfCfg.Phone = state.Phone
			s.vfCfg.LoginMethod = state.LoginMethod
			latestCfg := config.LoadVodafoneConfig(s.pref)
			s.vfCfg.AccessToken = latestCfg.AccessToken
			s.vfCfg.TokenExpiry = latestCfg.TokenExpiry
			s.vfCfg.PUK = latestCfg.PUK
			s.vodafonePUKEntry.SetText(strings.TrimSpace(latestCfg.PUK))
			s.vodafoneCodeEntry.SetText("")
			s.vodafoneStatusLabel.SetText(s.vfAuthVM.BuildStatusText(state))
		})
	}()
}

func (s *settingsDialogState) handleVodafoneTokenClear() {
	if s.adminProvider == nil {
		s.vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
		return
	}

	s.setVodafoneBusy(true)
	go func() {
		err := s.adminProvider.ClearVodafoneLogin()
		fyne.Do(func() {
			s.setVodafoneBusy(false)
			if err != nil {
				s.vodafoneStatusLabel.SetText(err.Error())
				return
			}
			s.vfCfg = config.LoadVodafoneConfig(s.pref)
			s.refreshVodafoneStatus()
		})
	}()
}

func (s *settingsDialogState) handleKyivstarTokenRefresh() {
	if s.adminProvider == nil {
		s.kyivstarStatusLabel.SetText("Kyivstar: сервіс недоступний")
		return
	}

	currentCfg := config.LoadKyivstarConfig(s.pref)
	currentCfg.ClientID = strings.TrimSpace(s.kyivstarClientIDEntry.Text)
	currentCfg.ClientSecret = strings.TrimSpace(s.kyivstarClientSecretEntry.Text)
	currentCfg.UserEmail = strings.TrimSpace(s.kyivstarEmailEntry.Text)
	currentCfg.AccessToken = ""
	currentCfg.TokenExpiry = ""
	config.SaveKyivstarConfig(s.pref, currentCfg)
	s.ksCfg = currentCfg

	s.setKyivstarBusy(true)
	s.kyivstarStatusLabel.SetText("Kyivstar: отримання access token...")
	go func() {
		state, err := s.adminProvider.RefreshKyivstarToken()
		fyne.Do(func() {
			s.setKyivstarBusy(false)
			if err != nil {
				s.kyivstarStatusLabel.SetText(err.Error())
				return
			}
			s.ksCfg = config.LoadKyivstarConfig(s.pref)
			s.kyivstarStatusLabel.SetText(s.ksAuthVM.BuildStatusText(state))
		})
	}()
}

func (s *settingsDialogState) handleKyivstarTokenClear() {
	if s.adminProvider == nil {
		s.kyivstarStatusLabel.SetText("Kyivstar: сервіс недоступний")
		return
	}

	s.setKyivstarBusy(true)
	go func() {
		err := s.adminProvider.ClearKyivstarToken()
		fyne.Do(func() {
			s.setKyivstarBusy(false)
			if err != nil {
				s.kyivstarStatusLabel.SetText(err.Error())
				return
			}
			s.ksCfg = config.LoadKyivstarConfig(s.pref)
			s.refreshKyivstarStatus()
		})
	}()
}

func (s *settingsDialogState) refreshPhoenixRuntimeMetadata() {
	if s == nil || s.phoenixRuntimeStatusLabel == nil {
		return
	}
	cfg := s.buildDBConfigFromForm()
	selectedID := cfg.PhoenixOperatorID
	if selectedID <= 0 {
		selectedID = s.dbCfg.PhoenixOperatorID
	}
	s.phoenixRuntimeStatusLabel.SetText("Завантаження портів і користувачів Phoenix...")

	go func() {
		ctx, cancel := context.WithTimeout(s.dialogCtx, 12*time.Second)
		defer cancel()
		metadata, err := data.LoadPhoenixRuntimeMetadata(ctx, cfg)
		fyne.Do(func() {
			if err != nil {
				s.phoenixRuntimeStatusLabel.SetText("Phoenix: " + err.Error())
				return
			}
			s.phoenixOperatorsByLabel = make(map[string]data.PhoenixOperator, len(metadata.Operators))
			options := make([]string, 0, len(metadata.Operators))
			selectedLabel := ""
			for _, operator := range metadata.Operators {
				label := operator.DisplayName()
				s.phoenixOperatorsByLabel[label] = operator
				options = append(options, label)
				if operator.ID == selectedID {
					selectedLabel = label
				}
			}
			s.phoenixOperatorSelect.Options = options
			s.phoenixOperatorSelect.Refresh()
			switch {
			case selectedLabel != "":
				s.phoenixOperatorSelect.SetSelected(selectedLabel)
			case len(options) > 0:
				s.phoenixOperatorSelect.SetSelected(options[0])
			}
			s.phoenixRuntimeStatusLabel.SetText(fmt.Sprintf(
				"Порти з БД: Control Center %d, Duty Operator %d, Administrator %d, GPS %d. Користувачів: %d",
				metadata.ControlPort,
				metadata.ClientPort,
				metadata.AdminPort,
				metadata.GPSPort,
				len(metadata.Operators),
			))
		})
	}()
}

func (s *settingsDialogState) verifyPhoenixOperator() {
	if s == nil || s.phoenixRuntimeStatusLabel == nil {
		return
	}
	cfg := s.buildDBConfigFromForm()
	s.phoenixRuntimeStatusLabel.SetText("Перевірка користувача Phoenix...")
	go func() {
		ctx, cancel := context.WithTimeout(s.dialogCtx, 12*time.Second)
		defer cancel()
		err := data.ValidatePhoenixOperatorCredentials(ctx, cfg)
		fyne.Do(func() {
			if err != nil {
				s.phoenixRuntimeStatusLabel.SetText("Phoenix: " + err.Error())
				return
			}
			s.phoenixRuntimeStatusLabel.SetText("Користувача Phoenix підтверджено.")
		})
	}()
}

func (s *settingsDialogState) applySave() {
	newDbCfg := s.buildDBConfigFromForm()
	newUiCfg := s.buildUIConfigFromForm()
	newVodafoneCfg := s.buildVodafoneConfigFromForm()
	newKyivstarCfg := s.buildKyivstarConfigFromForm()

	config.SaveDBConfig(s.pref, newDbCfg)
	config.SaveUIConfig(s.pref, newUiCfg)
	config.SaveVodafoneConfig(s.pref, newVodafoneCfg)
	config.SaveKyivstarConfig(s.pref, newKyivstarCfg)

	if s.onSave != nil {
		s.onSave(newDbCfg, newUiCfg)
	}
}

func (s *settingsDialogState) buildDBConfigFromForm() config.DBConfig {
	caslEnabled := s.caslEnabledCheck.Checked
	firebirdEnabled := s.firebirdEnabledCheck.Checked
	phoenixEnabled := s.phoenixEnabledCheck.Checked

	mode := config.BackendModeFirebird
	switch {
	case phoenixEnabled && !firebirdEnabled:
		mode = config.BackendModePhoenix
	case caslEnabled && !firebirdEnabled && !phoenixEnabled:
		mode = config.BackendModeCASLCloud
	}

	caslPultID := int64(0)
	if parsed, err := strconv.ParseInt(strings.TrimSpace(s.caslPultIDEntry.Text), 10, 64); err == nil && parsed > 0 {
		caslPultID = parsed
	}
	selectedOperator := s.phoenixOperatorsByLabel[s.phoenixOperatorSelect.Selected]
	operatorName := strings.TrimSpace(selectedOperator.Login)
	if operatorName == "" {
		operatorName = strings.TrimSpace(selectedOperator.Name)
	}
	if selectedOperator.ID <= 0 && s.dbCfg.PhoenixOperatorID > 0 {
		selectedOperator.ID = s.dbCfg.PhoenixOperatorID
		operatorName = s.dbCfg.PhoenixOperatorName
	}
	clientRole := config.PhoenixClientRoleDuty
	if s.phoenixClientRoleSelect.Selected == phoenixRoleAdminLabel {
		clientRole = config.PhoenixClientRoleAdministrator
	}

	return config.DBConfig{
		User:                    s.userEntry.Text,
		Password:                s.passEntry.Text,
		Host:                    s.hostEntry.Text,
		Port:                    s.portEntry.Text,
		Path:                    s.pathEntry.Text,
		Params:                  s.paramsEntry.Text,
		FirebirdEnabled:         firebirdEnabled,
		PhoenixEnabled:          phoenixEnabled,
		PhoenixUser:             strings.TrimSpace(s.phoenixUserEntry.Text),
		PhoenixPassword:         s.phoenixPassEntry.Text,
		PhoenixHost:             strings.TrimSpace(s.phoenixHostEntry.Text),
		PhoenixPort:             strings.TrimSpace(s.phoenixPortEntry.Text),
		PhoenixInstance:         strings.TrimSpace(s.phoenixInstanceEntry.Text),
		PhoenixDatabase:         strings.TrimSpace(s.phoenixDatabaseEntry.Text),
		PhoenixParams:           strings.TrimSpace(s.phoenixParamsEntry.Text),
		PhoenixControlHost:      strings.TrimSpace(s.phoenixControlHostEntry.Text),
		PhoenixOperatorID:       selectedOperator.ID,
		PhoenixOperatorName:     operatorName,
		PhoenixOperatorPassword: s.phoenixOperatorPassEntry.Text,
		PhoenixClientRole:       clientRole,
		CASLEnabled:             caslEnabled,
		Mode:                    mode,
		CASLBaseURL:             strings.TrimSpace(s.caslBaseURLEntry.Text),
		CASLToken:               strings.TrimSpace(s.caslTokenEntry.Text),
		CASLEmail:               strings.TrimSpace(s.caslEmailEntry.Text),
		CASLPass:                strings.TrimSpace(s.caslPassEntry.Text),
		CASLPultID:              caslPultID,
		LogLevel:                strings.ToLower(strings.TrimSpace(s.logLevelSelect.Selected)),
	}
}

func (s *settingsDialogState) buildUIConfigFromForm() config.UIConfig {
	return config.UIConfig{
		FontSize:               parseFloat32(s.fontEntry.Text),
		FontSizeInterface:      config.LoadUIConfig(s.pref).FontSizeInterface,
		FontSizeObjectCard:     config.LoadUIConfig(s.pref).FontSizeObjectCard,
		FontSizeObjects:        parseFloat32(s.fontObjEntry.Text),
		FontSizeEvents:         parseFloat32(s.fontEvEntry.Text),
		FontSizeAlarms:         parseFloat32(s.fontAlmEntry.Text),
		ShowBottomAlarmJournal: s.bottomAlarmJournalCheck.Checked,
		ShowBottomEventJournal: s.bottomEventJournalCheck.Checked,
		AllowDetachedJournals:  config.LoadUIConfig(s.pref).AllowDetachedJournals,
		ExportDir:              strings.TrimSpace(s.exportDirEntry.Text),
		EventLogLimit:          parseInt(s.eventLimitEntry.Text),
		ObjectLogLimit:         parseInt(s.objectLimitEntry.Text),
		BridgeAlarmHistoryMode: bridgeAlarmHistoryModeValue(s.bridgeHistoryModeSelect.Selected),
		EventProbeIntervalSec:  parseInt(s.eventProbeIntervalEntry.Text),
		EventsReconcileSec:     parseInt(s.eventsReconcileEntry.Text),
		AlarmsReconcileSec:     parseInt(s.alarmsReconcileEntry.Text),
		ObjectsReconcileSec:    parseInt(s.objectsReconcileEntry.Text),
		FallbackRefreshSec:     parseInt(s.fallbackRefreshEntry.Text),
		MaxProbeBackoffSec:     parseInt(s.maxProbeBackoffEntry.Text),
	}
}

func (s *settingsDialogState) buildVodafoneConfigFromForm() config.VodafoneConfig {
	newCfg := config.LoadVodafoneConfig(s.pref)
	oldMethod := newCfg.NormalizedLoginMethod()
	newCfg.Phone = strings.TrimSpace(s.vodafonePhoneEntry.Text)
	newCfg.LoginMethod = s.selectedVodafoneLoginMethod()
	oldPUK := strings.TrimSpace(newCfg.PUK)
	newPUK := strings.TrimSpace(s.vodafonePUKEntry.Text)
	newCfg.PUK = newPUK
	newCfg.AutoResetEnabled = s.vodafoneAutoResetCheck.Checked
	newCfg.AutoResetDailyLimit = parseIntFallback(s.vodafoneAutoResetLimitEntry.Text, config.DefaultVodafoneAutoResetDailyLimit)
	newCfg.AutoResetWindowHours = parseIntFallback(s.vodafoneAutoResetWindowEntry.Text, config.DefaultVodafoneAutoResetWindowHours)
	if oldMethod != newCfg.NormalizedLoginMethod() || oldPUK != newPUK {
		newCfg.AccessToken = ""
		newCfg.TokenExpiry = ""
	}
	return newCfg
}

func (s *settingsDialogState) buildKyivstarConfigFromForm() config.KyivstarConfig {
	newCfg := config.LoadKyivstarConfig(s.pref)
	clientIDChanged := strings.TrimSpace(newCfg.ClientID) != strings.TrimSpace(s.kyivstarClientIDEntry.Text)
	clientSecretChanged := strings.TrimSpace(newCfg.ClientSecret) != strings.TrimSpace(s.kyivstarClientSecretEntry.Text)

	newCfg.ClientID = strings.TrimSpace(s.kyivstarClientIDEntry.Text)
	newCfg.ClientSecret = strings.TrimSpace(s.kyivstarClientSecretEntry.Text)
	newCfg.UserEmail = strings.TrimSpace(s.kyivstarEmailEntry.Text)
	newCfg.AutoResetEnabled = s.kyivstarAutoResetCheck.Checked
	newCfg.AutoResetDailyLimit = parseIntFallback(s.kyivstarAutoResetLimitEntry.Text, config.DefaultKyivstarAutoResetDailyLimit)
	newCfg.AutoResetWindowHours = parseIntFallback(s.kyivstarAutoResetWindowEntry.Text, config.DefaultKyivstarAutoResetWindowHours)
	if clientIDChanged || clientSecretChanged {
		newCfg.AccessToken = ""
		newCfg.TokenExpiry = ""
	}

	return newCfg
}

func parseFloat32(raw string) float32 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(raw), 32)
	if err != nil {
		return 0
	}
	return float32(parsed)
}

func parseInt(raw string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0
	}
	return parsed
}

func parseIntFallback(raw string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return parsed
}

func timeNow() time.Time {
	return time.Now()
}

func uriPathToLocalPath(path string) string {
	if runtime.GOOS == "windows" && len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		return path[1:]
	}
	return path
}

func bridgeAlarmHistoryModeOptions() []string {
	return []string{
		"Тільки активні події з ACTALARMS",
		"Повна хронологія з журналу об'єкта",
	}
}

func bridgeAlarmHistoryModeLabel(mode string) string {
	switch config.NormalizeBridgeAlarmHistoryMode(mode) {
	case config.BridgeAlarmHistoryModeLegacy:
		return "Повна хронологія з журналу об'єкта"
	default:
		return "Тільки активні події з ACTALARMS"
	}
}

func bridgeAlarmHistoryModeValue(label string) string {
	switch strings.TrimSpace(label) {
	case "Повна хронологія з журналу об'єкта":
		return config.BridgeAlarmHistoryModeLegacy
	default:
		return config.BridgeAlarmHistoryModeActiveOnly
	}
}
