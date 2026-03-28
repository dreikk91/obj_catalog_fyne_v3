package dialogs

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func ShowNewObjectDialog(parent fyne.Window, provider contracts.AdminProvider, onSaved func(objn int64)) {
	ShowNewObjectWizardDialog(parent, provider, onSaved)
}

func ShowEditObjectDialog(parent fyne.Window, provider contracts.AdminProvider, objn int64, onSaved func(objn int64)) {
	showObjectCardDialog(parent, provider, &objn, onSaved)
}

func showObjectCardDialog(parent fyne.Window, provider contracts.AdminProvider, editObjN *int64, onSaved func(objn int64)) {
	isEdit := editObjN != nil && *editObjN > 0
	title := "Новий об'єкт"
	if isEdit {
		title = "Редагування/Створення об'єкта"
	} else {
		title = "Редагування/Створення об'єкта"
	}

	win := fyne.CurrentApp().NewWindow(title)
	win.Resize(fyne.NewSize(800, 600))

	statusLabel := widget.NewLabel("Готово")

	objnEntry := widget.NewEntry()
	shortNameEntry := widget.NewEntry()
	fullNameEntry := widget.NewEntry()
	addressEntry := widget.NewEntry()
	phonesEntry := widget.NewEntry()
	contractEntry := widget.NewEntry()
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("дд.мм.рррр")
	locationEntry := widget.NewMultiLineEntry()
	locationEntry.SetMinRowsVisible(2)
	notesEntry := widget.NewMultiLineEntry()
	notesEntry.SetMinRowsVisible(4)

	channelCodeSelect := widget.NewSelect([]string{
		"1 - Автододзвон",
		"5 - GPRS",
	}, nil)
	channelLabelToCode := map[string]int64{
		"1 - Автододзвон": 1,
		"5 - GPRS":        5,
	}
	channelCodeToLabel := map[int64]string{
		1: "1 - Автододзвон",
		5: "5 - GPRS",
	}
	sim1Entry := widget.NewEntry()
	sim2Entry := widget.NewEntry()
	sim1UsageLabel := widget.NewLabel("")
	sim1UsageLabel.Wrapping = fyne.TextWrapWord
	sim2UsageLabel := widget.NewLabel("")
	sim2UsageLabel.Wrapping = fyne.TextWrapWord
	hiddenNEntry := widget.NewEntry()
	hiddenNEntry.SetPlaceHolder("Прихований номер (до 4 цифр)")

	testControlCheck := widget.NewCheck("Контролювати тестові повідомлення", nil)
	testIntervalEntry := widget.NewEntry()
	testIntervalEntry.SetPlaceHolder("хв.")

	objectTypeSelect := widget.NewSelect(nil, nil)
	regionSelect := widget.NewSelect(nil, nil)
	ppkSelect := widget.NewSelect(nil, nil)
	subServerASelect := widget.NewSelect(nil, nil)
	subServerBSelect := widget.NewSelect(nil, nil)

	typeLabelToID := map[string]int64{}
	regionLabelToID := map[string]int64{}
	ppkLabelToID := map[string]int64{}
	allPPKItems := make([]contracts.PPKConstructorItem, 0)
	subServerLabelToBind := map[string]string{}
	hiddenNCard := widget.NewCard(
		"Прихований номер",
		"",
		container.NewVBox(
			widget.NewLabel("Номер (до 4 цифр):"),
			hiddenNEntry,
		),
	)

	autoUpdatingFullName := true
	fullNameSyncedWithShort := true
	fullNameEntry.OnChanged = func(text string) {
		if autoUpdatingFullName {
			return
		}
		fullNameSyncedWithShort = strings.TrimSpace(text) == strings.TrimSpace(shortNameEntry.Text)
	}
	shortNameEntry.OnChanged = func(text string) {
		if autoUpdatingFullName {
			return
		}
		if !fullNameSyncedWithShort {
			return
		}
		autoUpdatingFullName = true
		fullNameEntry.SetText(strings.TrimSpace(text))
		autoUpdatingFullName = false
	}

	enableTestControls := func(enabled bool) {
		if enabled {
			testIntervalEntry.Enable()
			return
		}
		testIntervalEntry.Disable()
	}
	testControlCheck.OnChanged = enableTestControls

	updateChannelSpecificControls := func() {
		channelCode := channelLabelToCode[channelCodeSelect.Selected]
		if channelCode == 5 {
			hiddenNCard.Show()
			hiddenNEntry.Enable()
		} else {
			hiddenNCard.Hide()
			hiddenNEntry.Disable()
		}
	}
	refreshPPKOptionsByChannel := func(preferredID int64) {
		ppkLabelToID = map[string]int64{"—": 0}
		ppkOptions := []string{"—"}
		for _, item := range allPPKItems {
			label := strings.TrimSpace(item.Name)
			if label == "" {
				label = fmt.Sprintf("ППК %d", item.ID)
			}
			label = fmt.Sprintf("%s [%d]", label, item.ID)
			ppkOptions = append(ppkOptions, label)
			ppkLabelToID[label] = item.ID
		}
		ppkSelect.Options = ppkOptions
		ppkSelect.Refresh()

		preferredIDs := make([]int64, 0, 3)
		if preferredID > 0 {
			preferredIDs = append(preferredIDs, preferredID)
			if preferredID > 100 {
				preferredIDs = append(preferredIDs, preferredID-100)
			}
			if preferredID < 100 {
				preferredIDs = append(preferredIDs, preferredID+100)
			}
		}
		for _, wantedID := range preferredIDs {
			for _, opt := range ppkSelect.Options {
				if ppkLabelToID[opt] == wantedID {
					ppkSelect.SetSelected(opt)
					return
				}
			}
		}
		ppkSelect.SetSelected("—")
	}
	channelCodeSelect.OnChanged = func(_ string) {
		selectedPPKID := ppkLabelToID[ppkSelect.Selected]
		updateChannelSpecificControls()
		refreshPPKOptionsByChannel(selectedPPKID)
	}

	formatSIMUsageList := func(usages []contracts.AdminSIMPhoneUsage) string {
		if len(usages) == 0 {
			return ""
		}
		parts := make([]string, 0, len(usages))
		for _, u := range usages {
			name := strings.TrimSpace(u.Name)
			if name != "" {
				parts = append(parts, fmt.Sprintf("#%d (%s, %s)", u.ObjN, name, u.Slot))
				continue
			}
			parts = append(parts, fmt.Sprintf("#%d (%s)", u.ObjN, u.Slot))
		}
		return "Номер вже використовується: " + strings.Join(parts, "; ")
	}

	checkSIMUsage := func(rawPhone string, targetLabel *widget.Label) {
		rawPhone = strings.TrimSpace(rawPhone)
		if rawPhone == "" {
			targetLabel.SetText("")
			return
		}
		usages, err := provider.FindObjectsBySIMPhone(rawPhone, editObjN)
		if err != nil {
			targetLabel.SetText("Не вдалося перевірити номер у базі")
			return
		}
		targetLabel.SetText(formatSIMUsageList(usages))
	}
	sim1Entry.OnChanged = func(text string) {
		checkSIMUsage(text, sim1UsageLabel)
	}
	sim2Entry.OnChanged = func(text string) {
		checkSIMUsage(text, sim2UsageLabel)
	}

	loadReferenceData := func() error {
		typeItems, err := provider.ListObjectTypes()
		if err != nil {
			return fmt.Errorf("не вдалося завантажити типи об'єктів: %w", err)
		}
		regionItems, err := provider.ListObjectDistricts()
		if err != nil {
			return fmt.Errorf("не вдалося завантажити райони: %w", err)
		}
		ppkItems, err := provider.ListPPKConstructor()
		if err != nil {
			return fmt.Errorf("не вдалося завантажити довідник ППК: %w", err)
		}
		subServerItems, err := provider.ListSubServers()
		if err != nil {
			return fmt.Errorf("не вдалося завантажити довідник підсерверів: %w", err)
		}

		typeLabelToID = map[string]int64{}
		typeOptions := make([]string, 0, len(typeItems))
		for _, item := range typeItems {
			label := strings.TrimSpace(item.Name)
			if label == "" {
				label = fmt.Sprintf("Тип %d", item.ID)
			}
			label = fmt.Sprintf("%s [%d]", label, item.ID)
			typeOptions = append(typeOptions, label)
			typeLabelToID[label] = item.ID
		}
		objectTypeSelect.Options = typeOptions
		objectTypeSelect.Refresh()

		regionLabelToID = map[string]int64{}
		regionOptions := []string{"—"}
		regionLabelToID["—"] = 0
		for _, item := range regionItems {
			label := strings.TrimSpace(item.Name)
			if label == "" {
				label = fmt.Sprintf("Район %d", item.ID)
			}
			label = fmt.Sprintf("%s [%d]", label, item.ID)
			regionOptions = append(regionOptions, label)
			regionLabelToID[label] = item.ID
		}
		regionSelect.Options = regionOptions
		regionSelect.Refresh()

		allPPKItems = ppkItems
		refreshPPKOptionsByChannel(ppkLabelToID[ppkSelect.Selected])

		subServerLabelToBind = map[string]string{}
		subServerOptions := []string{"—"}
		subServerLabelToBind["—"] = ""
		subServerTypeLabel := func(t int64) string {
			switch t {
			case 2:
				return "GPRS"
			case 4:
				return "AVD"
			default:
				if t > 0 {
					return fmt.Sprintf("%d", t)
				}
				return "—"
			}
		}
		for _, item := range subServerItems {
			bind := strings.TrimSpace(item.Bind)
			if bind == "" {
				continue
			}
			name := strings.TrimSpace(item.Info)
			if name == "" {
				name = strings.TrimSpace(item.Host)
			}
			if name == "" {
				name = fmt.Sprintf("Підсервер %d", item.ID)
			}
			label := fmt.Sprintf("%s (%s) [%s]", name, subServerTypeLabel(item.Type), bind)
			subServerOptions = append(subServerOptions, label)
			subServerLabelToBind[label] = bind
		}
		subServerASelect.Options = subServerOptions
		subServerASelect.Refresh()
		subServerBSelect.Options = subServerOptions
		subServerBSelect.Refresh()

		return nil
	}

	setSelectByID := func(sel *widget.Select, options []string, labelToID map[string]int64, id int64) {
		for _, opt := range options {
			if labelToID[opt] == id {
				sel.SetSelected(opt)
				return
			}
		}
		if len(options) > 0 {
			sel.SetSelected(options[0])
			return
		}
		sel.ClearSelected()
	}

	setSubServerByBind := func(sel *widget.Select, bind string) {
		bind = strings.TrimSpace(bind)
		if bind == "" {
			sel.SetSelected("—")
			return
		}
		for _, opt := range sel.Options {
			if subServerLabelToBind[opt] == bind {
				sel.SetSelected(opt)
				return
			}
		}
		sel.SetSelected("—")
	}

	fillDefaults := func() {
		objnEntry.SetText("")
		shortNameEntry.SetText("")
		fullNameEntry.SetText("")
		fullNameSyncedWithShort = true
		addressEntry.SetText("")
		phonesEntry.SetText("")
		contractEntry.SetText("")
		dateEntry.SetText(time.Now().Format("02.01.2006"))
		locationEntry.SetText("")
		notesEntry.SetText("")
		channelCodeSelect.SetSelected(channelCodeToLabel[1])
		sim1Entry.SetText("")
		sim2Entry.SetText("")
		sim1UsageLabel.SetText("")
		sim2UsageLabel.SetText("")
		hiddenNEntry.SetText("")
		testControlCheck.SetChecked(true)
		testIntervalEntry.SetText("9")
		enableTestControls(true)
		updateChannelSpecificControls()
		refreshPPKOptionsByChannel(0)

		setSelectByID(objectTypeSelect, objectTypeSelect.Options, typeLabelToID, 0)
		setSelectByID(regionSelect, regionSelect.Options, regionLabelToID, 1)
		subServerASelect.SetSelected("—")
		subServerBSelect.SetSelected("—")
	}

	loadCard := func(objn int64) error {
		card, err := provider.GetObjectCard(objn)
		if err != nil {
			return err
		}

		objnEntry.SetText(strconv.FormatInt(card.ObjN, 10))
		shortNameEntry.SetText(card.ShortName)
		fullNameEntry.SetText(card.FullName)
		addressEntry.SetText(card.Address)
		phonesEntry.SetText(card.Phones)
		contractEntry.SetText(card.Contract)
		dateEntry.SetText(card.StartDate)
		locationEntry.SetText(card.Location)
		notesEntry.SetText(card.Notes)
		if label, ok := channelCodeToLabel[card.ChannelCode]; ok {
			channelCodeSelect.SetSelected(label)
		} else {
			channelCodeSelect.SetSelected(channelCodeToLabel[1])
		}
		refreshPPKOptionsByChannel(card.PPKID)
		sim1Entry.SetText(card.GSMPhone1)
		sim2Entry.SetText(card.GSMPhone2)
		checkSIMUsage(card.GSMPhone1, sim1UsageLabel)
		checkSIMUsage(card.GSMPhone2, sim2UsageLabel)
		if card.GSMHiddenN > 0 {
			hiddenNEntry.SetText(strconv.FormatInt(card.GSMHiddenN, 10))
		} else {
			hiddenNEntry.SetText("")
		}
		testControlCheck.SetChecked(card.TestControlEnabled)
		if card.TestIntervalMin > 0 {
			testIntervalEntry.SetText(strconv.FormatInt(card.TestIntervalMin, 10))
		} else {
			testIntervalEntry.SetText("9")
		}
		enableTestControls(card.TestControlEnabled)

		objTypeID := card.ObjTypeID
		if objTypeID <= 0 && len(objectTypeSelect.Options) > 0 {
			objTypeID = typeLabelToID[objectTypeSelect.Options[0]]
		}
		setSelectByID(objectTypeSelect, objectTypeSelect.Options, typeLabelToID, objTypeID)
		regionID := card.ObjRegID
		if regionID <= 0 {
			regionID = 1
		}
		setSelectByID(regionSelect, regionSelect.Options, regionLabelToID, regionID)
		setSubServerByBind(subServerASelect, card.SubServerA)
		setSubServerByBind(subServerBSelect, card.SubServerB)
		updateChannelSpecificControls()

		return nil
	}

	buildCardFromUI := func() (contracts.AdminObjectCard, error) {
		var card contracts.AdminObjectCard

		objnRaw := strings.TrimSpace(objnEntry.Text)
		objn, err := strconv.ParseInt(objnRaw, 10, 64)
		if err != nil {
			return card, fmt.Errorf("некоректний об'єктовий номер")
		}
		card.ObjN = objn
		card.GrpN = 1
		card.ShortName = strings.TrimSpace(shortNameEntry.Text)
		card.FullName = strings.TrimSpace(fullNameEntry.Text)
		card.Address = strings.TrimSpace(addressEntry.Text)
		card.Phones = strings.TrimSpace(phonesEntry.Text)
		card.Contract = strings.TrimSpace(contractEntry.Text)
		card.StartDate = strings.TrimSpace(dateEntry.Text)
		card.Location = strings.TrimSpace(locationEntry.Text)
		card.Notes = strings.TrimSpace(notesEntry.Text)
		card.GSMPhone1 = strings.TrimSpace(sim1Entry.Text)
		card.GSMPhone2 = strings.TrimSpace(sim2Entry.Text)
		card.TestControlEnabled = testControlCheck.Checked

		channelCode, ok := channelLabelToCode[channelCodeSelect.Selected]
		if !ok {
			return card, fmt.Errorf("виберіть канал зв'язку")
		}
		card.ChannelCode = channelCode
		if channelCode == 5 {
			hiddenRaw := strings.TrimSpace(hiddenNEntry.Text)
			hiddenN, err := strconv.ParseInt(hiddenRaw, 10, 64)
			if err != nil || hiddenN <= 0 {
				return card, fmt.Errorf("для каналу 5 вкажіть коректний прихований номер")
			}
			card.GSMHiddenN = hiddenN
		} else {
			card.GSMHiddenN = 0
		}

		if card.TestControlEnabled {
			testRaw := strings.TrimSpace(testIntervalEntry.Text)
			testInterval, err := strconv.ParseInt(testRaw, 10, 64)
			if err != nil || testInterval <= 0 {
				return card, fmt.Errorf("некоректний інтервал контролю тесту")
			}
			card.TestIntervalMin = testInterval
		}

		objTypeID := typeLabelToID[objectTypeSelect.Selected]
		if objTypeID <= 0 {
			return card, fmt.Errorf("виберіть тип об'єкта")
		}
		card.ObjTypeID = objTypeID
		card.ObjRegID = regionLabelToID[regionSelect.Selected]
		card.PPKID = ppkLabelToID[ppkSelect.Selected]
		card.SubServerA = strings.TrimSpace(subServerLabelToBind[subServerASelect.Selected])
		card.SubServerB = strings.TrimSpace(subServerLabelToBind[subServerBSelect.Selected])

		return card, nil
	}

	parseDateFromEntry := func(raw string) (time.Time, bool) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return time.Time{}, false
		}
		formats := []string{
			"02.01.2006",
			"2006-01-02",
			"2006-01-02 15:04:05",
			time.RFC3339,
		}
		for _, f := range formats {
			if t, err := time.ParseInLocation(f, raw, time.Local); err == nil {
				return t, true
			}
		}
		return time.Time{}, false
	}

	openDatePicker := func() {
		initial := time.Now()
		if parsed, ok := parseDateFromEntry(dateEntry.Text); ok {
			initial = parsed
		}

		var pickerDlg dialog.Dialog
		calendar := xwidget.NewCalendar(initial, func(selected time.Time) {
			dateEntry.SetText(selected.Format("02.01.2006"))
			if pickerDlg != nil {
				pickerDlg.Hide()
			}
		})
		pickerDlg = dialog.NewCustom("Вибір дати", "Закрити", container.NewPadded(calendar), win)
		pickerDlg.Show()
	}
	datePickBtn := widget.NewButton("...", openDatePicker)
	datePickBtn.Importance = widget.LowImportance
	dateRow := container.NewBorder(nil, nil, nil, datePickBtn, dateEntry)

	objectAndTypeRow := container.NewGridWithColumns(
		2,
		shortNameEntry,
		objectTypeSelect,
	)
	phoneContractDateRow := container.NewGridWithColumns(
		3,
		phonesEntry,
		contractEntry,
		dateRow,
	)
	channelAndPPKRow := container.NewGridWithColumns(2, channelCodeSelect, ppkSelect)

	saveBtn := widget.NewButton("Зберегти", func() {
		card, err := buildCardFromUI()
		if err != nil {
			statusLabel.SetText(err.Error())
			return
		}

		if isEdit {
			loaded, err := provider.GetObjectCard(*editObjN)
			if err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося перезавантажити картку")
				return
			}
			card.ObjUIN = loaded.ObjUIN
			if err := provider.UpdateObject(card); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося зберегти зміни об'єкта")
				return
			}
			statusLabel.SetText("Картку об'єкта оновлено")
		} else {
			if err := provider.CreateObject(card); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося створити об'єкт")
				return
			}
			statusLabel.SetText("Новий об'єкт створено")
		}

		if onSaved != nil {
			onSaved(card.ObjN)
		}
		win.Close()
	})

	cancelBtn := widget.NewButton("Відміна", func() { win.Close() })

	mainInfoForm := widget.NewForm(
		widget.NewFormItem("№ об'єкта:", objnEntry),
		widget.NewFormItem("Об'єкт / Тип:", objectAndTypeRow),
		widget.NewFormItem("Повна назва:", fullNameEntry),
		widget.NewFormItem("Телефони / Договір / Дата:", phoneContractDateRow),
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Розташування:", locationEntry),
		widget.NewFormItem("Інформація:", notesEntry),
		widget.NewFormItem("Район:", regionSelect),
		widget.NewFormItem("Канал / ППК:", channelAndPPKRow),
	)

	testControlForm := widget.NewForm(
		widget.NewFormItem("Контролювати:", testControlCheck),
		widget.NewFormItem("Інтервал, хв.:", container.NewGridWrap(fyne.NewSize(90, 36), testIntervalEntry)),
	)
	testControlCard := widget.NewCard("Контроль GPRS/тестів", "", testControlForm)

	simPhonesForm := widget.NewForm(
		widget.NewFormItem("SIM1:", container.NewVBox(sim1Entry, sim1UsageLabel)),
		widget.NewFormItem("SIM2:", container.NewVBox(sim2Entry, sim2UsageLabel)),
	)
	
	simPhonesCard := widget.NewCard("Телефони", "", simPhonesForm)

	subserverForm := widget.NewForm(
		widget.NewFormItem("Підсервер A:", subServerASelect),
		widget.NewFormItem("Підсервер B:", subServerBSelect),
	)
	subserverCard := widget.NewCard("Підсервери", "", subserverForm)

	ppkParamsRow := container.NewGridWithColumns(3, testControlCard, simPhonesCard, hiddenNCard)

	objectTab := container.NewVScroll(container.NewVBox(
		mainInfoForm,
		widget.NewSeparator(),
		ppkParamsRow,
		widget.NewSeparator(),
		subserverCard,
	))

	placeholderText := "Вкладка буде перенесена за формами з D:\\most_output (frmObjChange.dfm)."
	if !isEdit {
		placeholderText = "Для цієї вкладки спочатку збережіть новий об'єкт."
	}

	personalTab := fyne.CanvasObject(container.NewPadded(widget.NewLabel(placeholderText)))
	zonesTab := fyne.CanvasObject(container.NewPadded(widget.NewLabel(placeholderText)))
	additionalTab := fyne.CanvasObject(container.NewPadded(widget.NewLabel(placeholderText)))
	if isEdit {
		personalTab = buildObjectPersonalTab(win, provider, *editObjN, statusLabel)
		zonesTab = buildObjectZonesTab(win, provider, *editObjN, statusLabel)
		additionalTab = buildObjectAdditionalTab(
			win,
			provider,
			*editObjN,
			statusLabel,
			func() string {
				return strings.TrimSpace(addressEntry.Text)
			},
			func(regionID int64) bool {
				for _, opt := range regionSelect.Options {
					if regionLabelToID[opt] == regionID {
						regionSelect.SetSelected(opt)
						return true
					}
				}
				return false
			},
		)
	}

	tabs := container.NewAppTabs(
		container.NewTabItem("Об'єкт", objectTab),
		container.NewTabItem("В/О", personalTab),
		container.NewTabItem("Зображення", container.NewPadded(widget.NewLabel(placeholderText))),
		container.NewTabItem("Зони", zonesTab),
		container.NewTabItem("Додатково", additionalTab),
	)

	content := container.NewBorder(
		nil,
		container.NewHBox(statusLabel, layout.NewSpacer(), saveBtn, cancelBtn),
		nil, nil,
		tabs,
	)
	win.SetContent(content)

	if err := loadReferenceData(); err != nil {
		dialog.ShowError(err, win)
		statusLabel.SetText("Не вдалося завантажити довідники")
	}

	if isEdit {
		objnEntry.Disable()
		if err := loadCard(*editObjN); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити об'єкт для редагування")
		}
	} else {
		fillDefaults()
	}

	win.Show()
}
