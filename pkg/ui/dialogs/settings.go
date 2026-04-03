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
	vodafoneProvider contracts.AdminObjectVodafoneService,
	pref fyne.Preferences,
	isDarkTheme bool,
	onSave func(config.DBConfig, config.UIConfig),
	onColorsChanged func(),
) {
	dbCfg := config.LoadDBConfig(pref)
	uiCfg := config.LoadUIConfig(pref)
	vfCfg := config.LoadVodafoneConfig(pref)
	vfAuthVM := viewmodels.NewVodafoneAuthViewModel()

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

	setVodafoneBusy := func(busy bool) {
		if busy {
			vodafonePhoneEntry.Disable()
			vodafoneCodeEntry.Disable()
			return
		}
		vodafonePhoneEntry.Enable()
		vodafoneCodeEntry.Enable()
	}

	refreshVodafoneStatus := func() {
		state := contracts.VodafoneAuthState{
			Phone:          strings.TrimSpace(vodafonePhoneEntry.Text),
			Authorized:     vfCfg.TokenUsableAt(timeNow()),
			TokenExpiresAt: vfCfg.TokenExpiryTime(),
		}
		if vodafoneProvider != nil {
			if liveState, err := vodafoneProvider.GetVodafoneAuthState(); err == nil {
				state = liveState
				if strings.TrimSpace(liveState.Phone) != "" {
					vodafonePhoneEntry.SetText(strings.TrimSpace(liveState.Phone))
				}
				vfCfg.Phone = liveState.Phone
			}
		}
		vodafoneStatusLabel.SetText(vfAuthVM.BuildStatusText(state))
	}

	requestVodafoneSMSBtn := widget.NewButton("Надіслати SMS", func() {
		if vodafoneProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		phone := strings.TrimSpace(vodafonePhoneEntry.Text)
		setVodafoneBusy(true)
		vodafoneStatusLabel.SetText("Vodafone: надсилання SMS-коду...")
		go func() {
			err := vodafoneProvider.RequestVodafoneLoginSMS(phone)
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
		if vodafoneProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		phone := strings.TrimSpace(vodafonePhoneEntry.Text)
		code := strings.TrimSpace(vodafoneCodeEntry.Text)
		setVodafoneBusy(true)
		vodafoneStatusLabel.SetText("Vodafone: перевірка коду...")
		go func() {
			state, err := vodafoneProvider.VerifyVodafoneLogin(phone, code)
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
		if vodafoneProvider == nil {
			vodafoneStatusLabel.SetText("Vodafone: сервіс недоступний")
			return
		}
		setVodafoneBusy(true)
		go func() {
			err := vodafoneProvider.ClearVodafoneLogin()
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

	colorsBtn := makeIconButton("Налаштувати кольори...", iconSearch(), widget.LowImportance, func() {
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
		container.NewTabItem("Інтерфейс", widget.NewForm(
			widget.NewFormItem("Загальний шрифт", fontEntry),
			widget.NewFormItem("Шрифт об'єктів", fontObjEntry),
			widget.NewFormItem("Шрифт подій", fontEvEntry),
			widget.NewFormItem("Шрифт тривог", fontAlmEntry),
			widget.NewFormItem("Режим логування", logLevelSelect),
			widget.NewFormItem("Ліміт загального журналу", eventLimitEntry),
			widget.NewFormItem("Ліміт журналу об'єкта", objectLimitEntry),
			widget.NewFormItem("Папка експорту", exportDirRow),
			widget.NewFormItem("Кольори подій/об'єктів", colorsBtn),
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

				newUiCfg := config.UIConfig{
					FontSize:        float32(fSize),
					FontSizeObjects: float32(fObjSize),
					FontSizeEvents:  float32(fEvSize),
					FontSizeAlarms:  float32(fAlmSize),
					ExportDir:       strings.TrimSpace(exportDirEntry.Text),
					EventLogLimit:   evLimit,
					ObjectLogLimit:  objLimit,
				}

				newVodafoneCfg := config.LoadVodafoneConfig(pref)
				newVodafoneCfg.Phone = strings.TrimSpace(vodafonePhoneEntry.Text)

				config.SaveDBConfig(pref, newDbCfg)
				config.SaveUIConfig(pref, newUiCfg)
				config.SaveVodafoneConfig(pref, newVodafoneCfg)

				if onSave != nil {
					onSave(newDbCfg, newUiCfg)
				}
			}
		},
		win,
	)

	d.Resize(fyne.NewSize(560, 520))
	refreshVodafoneStatus()
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
