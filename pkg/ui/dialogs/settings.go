package dialogs

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
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

func ShowSettingsDialog(
	win fyne.Window,
	adminProvider contracts.AdminProvider,
	pref fyne.Preferences,
	isDarkTheme bool,
	onSave func(config.DBConfig, config.UIConfig),
	onColorsChanged func(),
) {
	dbCfg := config.LoadDBConfig(pref)
	uiCfg := config.LoadUIConfig(pref)
	vfCfg := config.LoadVodafoneConfig(pref)
	ksCfg := config.LoadKyivstarConfig(pref)
	vfAuthVM := viewmodels.NewVodafoneAuthViewModel()
	ksAuthVM := viewmodels.NewKyivstarAuthViewModel()

	// Database fields
	userEntry := widget.NewEntry()
	userEntry.SetText(dbCfg.User)
	passEntry := widget.NewPasswordEntry()
	passEntry.SetText(dbCfg.Password)
	hostEntry := widget.NewEntry()
	hostEntry.SetText(dbCfg.Host)
	portEntry := widget.NewEntry()
	portEntry.SetText(dbCfg.Port)
	pathEntry := widget.NewEntry()
	pathEntry.SetText(dbCfg.Path)
	paramsEntry := widget.NewEntry()
	paramsEntry.SetText(dbCfg.Params)
	firebirdEnabledCheck := widget.NewCheck("Увімкнути БД/МІСТ (Firebird)", nil)
	firebirdEnabledCheck.SetChecked(dbCfg.FirebirdEnabled || (!dbCfg.FirebirdEnabled && !dbCfg.PhoenixEnabled && dbCfg.NormalizedMode() != config.BackendModePhoenix))

	// Phoenix MSSQL fields
	phoenixEnabledCheck := widget.NewCheck("Увімкнути Phoenix паралельно з іншими джерелами", nil)
	phoenixEnabledCheck.SetChecked(dbCfg.PhoenixEnabled || dbCfg.NormalizedMode() == config.BackendModePhoenix)
	phoenixUserEntry := widget.NewEntry()
	phoenixUserEntry.SetText(dbCfg.PhoenixUser)
	phoenixPassEntry := widget.NewPasswordEntry()
	phoenixPassEntry.SetText(dbCfg.PhoenixPassword)
	phoenixHostEntry := widget.NewEntry()
	phoenixHostEntry.SetText(dbCfg.PhoenixHost)
	phoenixPortEntry := widget.NewEntry()
	phoenixPortEntry.SetText(dbCfg.PhoenixPort)
	phoenixInstanceEntry := widget.NewEntry()
	phoenixInstanceEntry.SetText(dbCfg.PhoenixInstance)
	phoenixDatabaseEntry := widget.NewEntry()
	phoenixDatabaseEntry.SetText(dbCfg.PhoenixDatabase)
	phoenixParamsEntry := widget.NewEntry()
	phoenixParamsEntry.SetText(dbCfg.PhoenixParams)

	// CASL Cloud fields
	caslBaseURLEntry := widget.NewEntry()
	caslBaseURLEntry.SetText(strings.TrimSpace(dbCfg.CASLBaseURL))
	caslBaseURLEntry.SetPlaceHolder("http://10.32.1.221:50003")

	caslTokenEntry := widget.NewEntry()
	caslTokenEntry.SetText(strings.TrimSpace(dbCfg.CASLToken))
	caslTokenEntry.SetPlaceHolder("JWT токен (необов'язково)")

	caslEmailEntry := widget.NewEntry()
	caslEmailEntry.SetText(strings.TrimSpace(dbCfg.CASLEmail))
	caslEmailEntry.SetPlaceHolder("test@lot.lviv.ua")

	caslPassEntry := widget.NewPasswordEntry()
	caslPassEntry.SetText(strings.TrimSpace(dbCfg.CASLPass))
	caslPassEntry.SetPlaceHolder("Пароль CASL")

	caslPultIDEntry := widget.NewEntry()
	if dbCfg.CASLPultID > 0 {
		caslPultIDEntry.SetText(strconv.FormatInt(dbCfg.CASLPultID, 10))
	}
	caslPultIDEntry.SetPlaceHolder("0 = авто")
	caslEnabledCheck := widget.NewCheck("Увімкнути CASL Cloud паралельно з БД/мостом", nil)
	caslEnabledCheck.SetChecked(dbCfg.CASLEnabled || dbCfg.NormalizedMode() == config.BackendModeCASLCloud)

	vodafonePhoneEntry := widget.NewEntry()
	vodafonePhoneEntry.SetText(strings.TrimSpace(vfCfg.Phone))
	vodafonePhoneEntry.SetPlaceHolder("380501234567")

	vodafoneCodeEntry := widget.NewPasswordEntry()
	vodafoneCodeEntry.SetPlaceHolder("SMS-код")

	vodafoneStatusLabel := widget.NewLabel(vfAuthVM.BuildStatusText(contracts.VodafoneAuthState{
		Phone:          vfCfg.Phone,
		Authorized:     vfCfg.TokenUsableAt(timeNow()),
		TokenExpiresAt: vfCfg.TokenExpiryTime(),
	}))
	vodafoneStatusLabel.Wrapping = fyne.TextWrapWord

	kyivstarClientIDEntry := widget.NewEntry()
	kyivstarClientIDEntry.SetText(strings.TrimSpace(ksCfg.ClientID))
	kyivstarClientIDEntry.SetPlaceHolder("client_id")

	kyivstarClientSecretEntry := widget.NewPasswordEntry()
	kyivstarClientSecretEntry.SetText(strings.TrimSpace(ksCfg.ClientSecret))
	kyivstarClientSecretEntry.SetPlaceHolder("client_secret")

	kyivstarEmailEntry := widget.NewEntry()
	kyivstarEmailEntry.SetText(strings.TrimSpace(ksCfg.UserEmail))
	kyivstarEmailEntry.SetPlaceHolder("company.user@domain.ua")

	kyivstarStatusLabel := widget.NewLabel(ksAuthVM.BuildStatusText(contracts.KyivstarAuthState{
		ClientID:       ksCfg.ClientID,
		UserEmail:      ksCfg.UserEmail,
		Configured:     ksCfg.HasCredentials(),
		Authorized:     ksCfg.TokenUsableAt(timeNow()),
		TokenExpiresAt: ksCfg.TokenExpiryTime(),
	}))
	kyivstarStatusLabel.Wrapping = fyne.TextWrapWord

	setVodafoneBusy := func(busy bool) {
		if busy {
			vodafonePhoneEntry.Disable()
			vodafoneCodeEntry.Disable()
			return
		}
		vodafonePhoneEntry.Enable()
		vodafoneCodeEntry.Enable()
	}

	setKyivstarBusy := func(busy bool) {
		if busy {
			kyivstarClientIDEntry.Disable()
			kyivstarClientSecretEntry.Disable()
			kyivstarEmailEntry.Disable()
			return
		}
		kyivstarClientIDEntry.Enable()
		kyivstarClientSecretEntry.Enable()
		kyivstarEmailEntry.Enable()
	}

	refreshVodafoneStatus := func() {
		state := contracts.VodafoneAuthState{
			Phone:          strings.TrimSpace(vodafonePhoneEntry.Text),
			Authorized:     vfCfg.TokenUsableAt(timeNow()),
			TokenExpiresAt: vfCfg.TokenExpiryTime(),
		}
		if adminProvider != nil {
			if liveState, err := adminProvider.GetVodafoneAuthState(); err == nil {
				state = liveState
				if strings.TrimSpace(liveState.Phone) != "" {
					vodafonePhoneEntry.SetText(strings.TrimSpace(liveState.Phone))
				}
				vfCfg.Phone = liveState.Phone
			}
		}
		vodafoneStatusLabel.SetText(vfAuthVM.BuildStatusText(state))
	}

	refreshKyivstarStatus := func() {
		state := contracts.KyivstarAuthState{
			ClientID:       strings.TrimSpace(kyivstarClientIDEntry.Text),
			UserEmail:      strings.TrimSpace(kyivstarEmailEntry.Text),
			Configured:     strings.TrimSpace(kyivstarClientIDEntry.Text) != "" && strings.TrimSpace(kyivstarClientSecretEntry.Text) != "",
			Authorized:     ksCfg.TokenUsableAt(timeNow()),
			TokenExpiresAt: ksCfg.TokenExpiryTime(),
		}
		if adminProvider != nil {
			if liveState, err := adminProvider.GetKyivstarAuthState(); err == nil {
				state = liveState
				if strings.TrimSpace(liveState.ClientID) != "" {
					kyivstarClientIDEntry.SetText(strings.TrimSpace(liveState.ClientID))
				}
				ksCfg.ClientID = liveState.ClientID
				ksCfg.UserEmail = liveState.UserEmail
			}
		}
		kyivstarStatusLabel.SetText(ksAuthVM.BuildStatusText(state))
	}

	requestVodafoneSMSBtn := widget.NewButton("Надіслати SMS", func() {
		if adminProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		phone := strings.TrimSpace(vodafonePhoneEntry.Text)
		setVodafoneBusy(true)
		vodafoneStatusLabel.SetText("Vodafone: надсилання SMS-коду...")
		go func() {
			err := adminProvider.RequestVodafoneLoginSMS(phone)
			fyne.Do(func() {
				setVodafoneBusy(false)
				if err != nil {
					vodafoneStatusLabel.SetText(err.Error())
					return
				}
				vodafoneStatusLabel.SetText("Vodafone: SMS-код надіслано")
			})
		}()
	})

	verifyVodafoneCodeBtn := widget.NewButton("Підтвердити код", func() {
		if adminProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		phone := strings.TrimSpace(vodafonePhoneEntry.Text)
		code := strings.TrimSpace(vodafoneCodeEntry.Text)
		setVodafoneBusy(true)
		vodafoneStatusLabel.SetText("Vodafone: перевірка коду...")
		go func() {
			state, err := adminProvider.VerifyVodafoneLogin(phone, code)
			fyne.Do(func() {
				setVodafoneBusy(false)
				if err != nil {
					vodafoneStatusLabel.SetText(err.Error())
					return
				}
				vfCfg.Phone = state.Phone
				vfCfg.AccessToken = config.LoadVodafoneConfig(pref).AccessToken
				vfCfg.TokenExpiry = config.LoadVodafoneConfig(pref).TokenExpiry
				vodafoneCodeEntry.SetText("")
				vodafoneStatusLabel.SetText(vfAuthVM.BuildStatusText(state))
			})
		}()
	})

	clearVodafoneTokenBtn := widget.NewButton("Очистити токен", func() {
		if adminProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		setVodafoneBusy(true)
		go func() {
			err := adminProvider.ClearVodafoneLogin()
			fyne.Do(func() {
				setVodafoneBusy(false)
				if err != nil {
					vodafoneStatusLabel.SetText(err.Error())
					return
				}
				latestCfg := config.LoadVodafoneConfig(pref)
				vfCfg = latestCfg
				refreshVodafoneStatus()
			})
		}()
	})

	refreshKyivstarTokenBtn := widget.NewButton("Отримати токен", func() {
		if adminProvider == nil {
			kyivstarStatusLabel.SetText("Kyivstar: сервіс недоступний")
			return
		}
		currentCfg := config.LoadKyivstarConfig(pref)
		currentCfg.ClientID = strings.TrimSpace(kyivstarClientIDEntry.Text)
		currentCfg.ClientSecret = strings.TrimSpace(kyivstarClientSecretEntry.Text)
		currentCfg.UserEmail = strings.TrimSpace(kyivstarEmailEntry.Text)
		currentCfg.AccessToken = ""
		currentCfg.TokenExpiry = ""
		config.SaveKyivstarConfig(pref, currentCfg)
		ksCfg = currentCfg
		setKyivstarBusy(true)
		kyivstarStatusLabel.SetText("Kyivstar: отримання access token...")
		go func() {
			state, err := adminProvider.RefreshKyivstarToken()
			fyne.Do(func() {
				setKyivstarBusy(false)
				if err != nil {
					kyivstarStatusLabel.SetText(err.Error())
					return
				}
				ksCfg = config.LoadKyivstarConfig(pref)
				kyivstarStatusLabel.SetText(ksAuthVM.BuildStatusText(state))
			})
		}()
	})

	clearKyivstarTokenBtn := widget.NewButton("Очистити токен", func() {
		if adminProvider == nil {
			kyivstarStatusLabel.SetText("Kyivstar: сервіс недоступний")
			return
		}
		setKyivstarBusy(true)
		go func() {
			err := adminProvider.ClearKyivstarToken()
			fyne.Do(func() {
				setKyivstarBusy(false)
				if err != nil {
					kyivstarStatusLabel.SetText(err.Error())
					return
				}
				ksCfg = config.LoadKyivstarConfig(pref)
				refreshKyivstarStatus()
			})
		}()
	})

	// UI fields
	fontEntry := widget.NewEntry()
	fontEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSize))
	fontObjEntry := widget.NewEntry()
	fontObjEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeObjects))
	fontEvEntry := widget.NewEntry()
	fontEvEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeEvents))
	fontAlmEntry := widget.NewEntry()
	fontAlmEntry.SetText(fmt.Sprintf("%.1f", uiCfg.FontSizeAlarms))

	eventLimitEntry := widget.NewEntry()
	eventLimitEntry.SetText(strconv.Itoa(uiCfg.EventLogLimit))
	eventLimitEntry.SetPlaceHolder("2000")

	objectLimitEntry := widget.NewEntry()
	objectLimitEntry.SetText(strconv.Itoa(uiCfg.ObjectLogLimit))
	objectLimitEntry.SetPlaceHolder("0 = без обмеження")

	bridgeHistoryModeSelect := widget.NewSelect(bridgeAlarmHistoryModeOptions(), nil)
	bridgeHistoryModeSelect.SetSelected(bridgeAlarmHistoryModeLabel(uiCfg.BridgeAlarmHistoryMode))

	eventProbeIntervalEntry := widget.NewEntry()
	eventProbeIntervalEntry.SetText(strconv.Itoa(uiCfg.EventProbeIntervalSec))
	eventProbeIntervalEntry.SetPlaceHolder(strconv.Itoa(config.DefaultEventProbeIntervalSec))

	eventsReconcileEntry := widget.NewEntry()
	eventsReconcileEntry.SetText(strconv.Itoa(uiCfg.EventsReconcileSec))
	eventsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultEventsReconcileSec))

	alarmsReconcileEntry := widget.NewEntry()
	alarmsReconcileEntry.SetText(strconv.Itoa(uiCfg.AlarmsReconcileSec))
	alarmsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultAlarmsReconcileSec))

	objectsReconcileEntry := widget.NewEntry()
	objectsReconcileEntry.SetText(strconv.Itoa(uiCfg.ObjectsReconcileSec))
	objectsReconcileEntry.SetPlaceHolder(strconv.Itoa(config.DefaultObjectsReconcileSec))

	fallbackRefreshEntry := widget.NewEntry()
	fallbackRefreshEntry.SetText(strconv.Itoa(uiCfg.FallbackRefreshSec))
	fallbackRefreshEntry.SetPlaceHolder(strconv.Itoa(config.DefaultFallbackRefreshSec))

	maxProbeBackoffEntry := widget.NewEntry()
	maxProbeBackoffEntry.SetText(strconv.Itoa(uiCfg.MaxProbeBackoffSec))
	maxProbeBackoffEntry.SetPlaceHolder(strconv.Itoa(config.DefaultMaxProbeBackoffSec))

	schedulerHelpLabel := widget.NewLabel("Оновлення Firebird, сек. Менші значення роблять інтерфейс актуальнішим, але сильніше навантажують сервер.")
	schedulerHelpLabel.Wrapping = fyne.TextWrapWord

	exportDirEntry := widget.NewEntry()
	exportDirEntry.SetText(uiCfg.ExportDir)
	exportDirEntry.SetPlaceHolder("Папка запуску програми")

	browseExportDirBtn := makeIconButton("Обрати...", iconFolder(), widget.MediumImportance, func() {
		dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uri == nil {
				return
			}
			exportDirEntry.SetText(uriPathToLocalPath(uri.Path()))
		}, win).Show()
	})

	clearExportDirBtn := makeIconButton("Очистити", iconClear(), widget.LowImportance, func() {
		exportDirEntry.SetText("")
	})

	exportDirRow := container.NewBorder(
		nil,
		nil,
		nil,
		container.NewHBox(browseExportDirBtn, clearExportDirBtn),
		exportDirEntry,
	)

	colorsBtn := makeIconButton("Налаштувати кольори подій...", iconSearch(), widget.LowImportance, func() {
		ShowColorPaletteDialog(win, isDarkTheme, onColorsChanged)
	})

	logLevelOptions := []string{"debug", "info", "warn", "error"}
	logLevelSelect := widget.NewSelect(logLevelOptions, nil)
	logLevelSelect.SetSelected(strings.ToLower(strings.TrimSpace(dbCfg.LogLevel)))
	if logLevelSelect.Selected == "" {
		logLevelSelect.SetSelected("info")
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("База даних", widget.NewForm(
			widget.NewFormItem("Увімкнення", firebirdEnabledCheck),
			widget.NewFormItem("Користувач", userEntry),
			widget.NewFormItem("Пароль", passEntry),
			widget.NewFormItem("Хост", hostEntry),
			widget.NewFormItem("Порт", portEntry),
			widget.NewFormItem("Шлях до БД", pathEntry),
			widget.NewFormItem("Параметри", paramsEntry),
		)),
		container.NewTabItem("Phoenix", widget.NewForm(
			widget.NewFormItem("Увімкнення", phoenixEnabledCheck),
			widget.NewFormItem("Користувач", phoenixUserEntry),
			widget.NewFormItem("Пароль", phoenixPassEntry),
			widget.NewFormItem("Хост", phoenixHostEntry),
			widget.NewFormItem("Порт", phoenixPortEntry),
			widget.NewFormItem("Інстанс", phoenixInstanceEntry),
			widget.NewFormItem("База", phoenixDatabaseEntry),
			widget.NewFormItem("Параметри", phoenixParamsEntry),
		)),
		container.NewTabItem("CASL Cloud", widget.NewForm(
			widget.NewFormItem("Паралельний режим", caslEnabledCheck),
			widget.NewFormItem("Base URL", caslBaseURLEntry),
			widget.NewFormItem("Token", caslTokenEntry),
			widget.NewFormItem("Email", caslEmailEntry),
			widget.NewFormItem("Password", caslPassEntry),
			widget.NewFormItem("Pult ID", caslPultIDEntry),
		)),
		container.NewTabItem("Vodafone", container.NewVBox(
			widget.NewLabel("Авторизація тільки через SMS-код для батьківського номера Vodafone."),
			widget.NewForm(
				widget.NewFormItem("Номер входу", vodafonePhoneEntry),
				widget.NewFormItem("SMS-код", vodafoneCodeEntry),
			),
			container.NewHBox(requestVodafoneSMSBtn, verifyVodafoneCodeBtn, clearVodafoneTokenBtn),
			vodafoneStatusLabel,
		)),
		container.NewTabItem("Kyivstar", container.NewVBox(
			widget.NewLabel("Kyivstar IoT API використовує client_id/client_secret і email компанії для reset запитів."),
			widget.NewForm(
				widget.NewFormItem("Client ID", kyivstarClientIDEntry),
				widget.NewFormItem("Client Secret", kyivstarClientSecretEntry),
				widget.NewFormItem("Email компанії", kyivstarEmailEntry),
			),
			container.NewHBox(refreshKyivstarTokenBtn, clearKyivstarTokenBtn),
			kyivstarStatusLabel,
		)),
		container.NewTabItem("Інтерфейс", widget.NewForm(
			widget.NewFormItem("Загальний шрифт", fontEntry),
			widget.NewFormItem("Шрифт об'єктів", fontObjEntry),
			widget.NewFormItem("Шрифт подій", fontEvEntry),
			widget.NewFormItem("Шрифт тривог", fontAlmEntry),
			widget.NewFormItem("Режим логування", logLevelSelect),
			widget.NewFormItem("Ліміт загального журналу", eventLimitEntry),
			widget.NewFormItem("Ліміт журналу об'єкта", objectLimitEntry),
			widget.NewFormItem("Хронологія МІСТ", bridgeHistoryModeSelect),
			widget.NewFormItem("Папка експорту", exportDirRow),
			widget.NewFormItem("Кольори подій", colorsBtn),
		)),
		container.NewTabItem("Оновлення", widget.NewForm(
			widget.NewFormItem("Пояснення", schedulerHelpLabel),
			widget.NewFormItem("Probe нових подій", eventProbeIntervalEntry),
			widget.NewFormItem("Reconcile журналу", eventsReconcileEntry),
			widget.NewFormItem("Reconcile тривог", alarmsReconcileEntry),
			widget.NewFormItem("Reconcile об'єктів", objectsReconcileEntry),
			widget.NewFormItem("Fallback без probe", fallbackRefreshEntry),
			widget.NewFormItem("Макс. backoff probe", maxProbeBackoffEntry),
		)),
	)

	d := dialog.NewCustomConfirm(
		"Налаштування системи",
		"Зберегти",
		"Скасувати",
		tabs,
		func(save bool) {
			if save {
				caslEnabled := caslEnabledCheck.Checked
				firebirdEnabled := firebirdEnabledCheck.Checked
				phoenixEnabled := phoenixEnabledCheck.Checked
				mode := config.BackendModeFirebird
				switch {
				case phoenixEnabled && !firebirdEnabled:
					mode = config.BackendModePhoenix
				case caslEnabled && !firebirdEnabled && !phoenixEnabled:
					mode = config.BackendModeCASLCloud
				}

				caslPultID := int64(0)
				if parsed, err := strconv.ParseInt(strings.TrimSpace(caslPultIDEntry.Text), 10, 64); err == nil && parsed > 0 {
					caslPultID = parsed
				}

				newDbCfg := config.DBConfig{
					User:            userEntry.Text,
					Password:        passEntry.Text,
					Host:            hostEntry.Text,
					Port:            portEntry.Text,
					Path:            pathEntry.Text,
					Params:          paramsEntry.Text,
					FirebirdEnabled: firebirdEnabled,
					PhoenixEnabled:  phoenixEnabled,
					PhoenixUser:     strings.TrimSpace(phoenixUserEntry.Text),
					PhoenixPassword: phoenixPassEntry.Text,
					PhoenixHost:     strings.TrimSpace(phoenixHostEntry.Text),
					PhoenixPort:     strings.TrimSpace(phoenixPortEntry.Text),
					PhoenixInstance: strings.TrimSpace(phoenixInstanceEntry.Text),
					PhoenixDatabase: strings.TrimSpace(phoenixDatabaseEntry.Text),
					PhoenixParams:   strings.TrimSpace(phoenixParamsEntry.Text),
					CASLEnabled:     caslEnabled,
					Mode:            mode,
					CASLBaseURL:     strings.TrimSpace(caslBaseURLEntry.Text),
					CASLToken:       strings.TrimSpace(caslTokenEntry.Text),
					CASLEmail:       strings.TrimSpace(caslEmailEntry.Text),
					CASLPass:        strings.TrimSpace(caslPassEntry.Text),
					CASLPultID:      caslPultID,
					LogLevel:        strings.ToLower(strings.TrimSpace(logLevelSelect.Selected)),
				}

				fSize, _ := strconv.ParseFloat(fontEntry.Text, 32)
				fObjSize, _ := strconv.ParseFloat(fontObjEntry.Text, 32)
				fEvSize, _ := strconv.ParseFloat(fontEvEntry.Text, 32)
				fAlmSize, _ := strconv.ParseFloat(fontAlmEntry.Text, 32)
				evLimit, _ := strconv.Atoi(strings.TrimSpace(eventLimitEntry.Text))
				objLimit, _ := strconv.Atoi(strings.TrimSpace(objectLimitEntry.Text))
				eventProbeIntervalSec, _ := strconv.Atoi(strings.TrimSpace(eventProbeIntervalEntry.Text))
				eventsReconcileSec, _ := strconv.Atoi(strings.TrimSpace(eventsReconcileEntry.Text))
				alarmsReconcileSec, _ := strconv.Atoi(strings.TrimSpace(alarmsReconcileEntry.Text))
				objectsReconcileSec, _ := strconv.Atoi(strings.TrimSpace(objectsReconcileEntry.Text))
				fallbackRefreshSec, _ := strconv.Atoi(strings.TrimSpace(fallbackRefreshEntry.Text))
				maxProbeBackoffSec, _ := strconv.Atoi(strings.TrimSpace(maxProbeBackoffEntry.Text))

				newUiCfg := config.UIConfig{
					FontSize:               float32(fSize),
					FontSizeObjects:        float32(fObjSize),
					FontSizeEvents:         float32(fEvSize),
					FontSizeAlarms:         float32(fAlmSize),
					ExportDir:              strings.TrimSpace(exportDirEntry.Text),
					EventLogLimit:          evLimit,
					ObjectLogLimit:         objLimit,
					BridgeAlarmHistoryMode: bridgeAlarmHistoryModeValue(bridgeHistoryModeSelect.Selected),
					EventProbeIntervalSec:  eventProbeIntervalSec,
					EventsReconcileSec:     eventsReconcileSec,
					AlarmsReconcileSec:     alarmsReconcileSec,
					ObjectsReconcileSec:    objectsReconcileSec,
					FallbackRefreshSec:     fallbackRefreshSec,
					MaxProbeBackoffSec:     maxProbeBackoffSec,
				}

				newVodafoneCfg := config.LoadVodafoneConfig(pref)
				newVodafoneCfg.Phone = strings.TrimSpace(vodafonePhoneEntry.Text)

				newKyivstarCfg := config.LoadKyivstarConfig(pref)
				clientIDChanged := strings.TrimSpace(newKyivstarCfg.ClientID) != strings.TrimSpace(kyivstarClientIDEntry.Text)
				clientSecretChanged := strings.TrimSpace(newKyivstarCfg.ClientSecret) != strings.TrimSpace(kyivstarClientSecretEntry.Text)
				newKyivstarCfg.ClientID = strings.TrimSpace(kyivstarClientIDEntry.Text)
				newKyivstarCfg.ClientSecret = strings.TrimSpace(kyivstarClientSecretEntry.Text)
				newKyivstarCfg.UserEmail = strings.TrimSpace(kyivstarEmailEntry.Text)
				if clientIDChanged || clientSecretChanged {
					newKyivstarCfg.AccessToken = ""
					newKyivstarCfg.TokenExpiry = ""
				}

				config.SaveDBConfig(pref, newDbCfg)
				config.SaveUIConfig(pref, newUiCfg)
				config.SaveVodafoneConfig(pref, newVodafoneCfg)
				config.SaveKyivstarConfig(pref, newKyivstarCfg)

				if onSave != nil {
					onSave(newDbCfg, newUiCfg)
				}
			}
		},
		win,
	)

	d.Resize(fyne.NewSize(560, 520))
	refreshVodafoneStatus()
	refreshKyivstarStatus()
	d.Show()
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
