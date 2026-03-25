package dialogs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"obj_catalog_fyne_v3/pkg/data"
)

func ShowNewObjectWizardDialog(parent fyne.Window, provider data.AdminProvider, onSaved func(objn int64)) {
	win := fyne.CurrentApp().NewWindow("Майстер створення об'єкта")
	win.Resize(fyne.NewSize(980, 760))

	statusLabel := widget.NewLabel("Крок 1/6: дані об'єкта")

	var (
		pendingPersonals []data.AdminObjectPersonal
		selectedPersonal = -1
		pendingZones     []data.AdminObjectZone
		selectedZone     = -1
	)

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
	latitudeEntry := widget.NewEntry()
	latitudeEntry.SetPlaceHolder("Широта (LATITUDE)")
	longitudeEntry := widget.NewEntry()
	longitudeEntry.SetPlaceHolder("Довгота (LONGITUDE)")

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
	hiddenNRow := container.NewHBox(
		widget.NewLabel("Для каналу GPRS (5):"),
		container.NewGridWrap(fyne.NewSize(150, 36), hiddenNEntry),
	)

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
	allPPKItems := make([]data.PPKConstructorItem, 0)
	subServerLabelToBind := map[string]string{}

	autoUpdatingFullName := false
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
			hiddenNRow.Show()
			hiddenNEntry.Enable()
		} else {
			hiddenNRow.Hide()
			hiddenNEntry.Disable()
		}
	}
	refreshPPKOptionsByChannel := func(preferredID int64) {
		channelCode := channelLabelToCode[channelCodeSelect.Selected]
		ppkLabelToID = map[string]int64{"—": 0}
		ppkOptions := []string{"—"}
		for _, item := range allPPKItems {
			if channelCode > 0 && item.Channel > 0 && item.Channel != channelCode {
				continue
			}
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

		if preferredID > 0 {
			for _, opt := range ppkSelect.Options {
				if ppkLabelToID[opt] == preferredID {
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

	formatSIMUsageList := func(usages []data.AdminSIMPhoneUsage) string {
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
		usages, err := provider.FindObjectsBySIMPhone(rawPhone, nil)
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
		setSelectByID(regionSelect, regionSelect.Options, regionLabelToID, 0)
		subServerASelect.SetSelected("—")
		subServerBSelect.SetSelected("—")
		latitudeEntry.SetText("")
		longitudeEntry.SetText("")
		pendingPersonals = nil
		selectedPersonal = -1
		pendingZones = nil
		selectedZone = -1
	}

	buildCardFromUI := func() (data.AdminObjectCard, error) {
		var card data.AdminObjectCard

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

	validateStep := func(step int) error {
		objnRaw := strings.TrimSpace(objnEntry.Text)
		if step >= 0 {
			if objnRaw == "" {
				return fmt.Errorf("вкажіть об'єктовий номер")
			}
			if _, err := strconv.ParseInt(objnRaw, 10, 64); err != nil {
				return fmt.Errorf("некоректний об'єктовий номер")
			}
			if strings.TrimSpace(shortNameEntry.Text) == "" {
				return fmt.Errorf("вкажіть коротку назву об'єкта")
			}
			if typeLabelToID[objectTypeSelect.Selected] <= 0 {
				return fmt.Errorf("виберіть тип об'єкта")
			}
			channelCode := channelLabelToCode[channelCodeSelect.Selected]
			if channelCode == 5 && strings.TrimSpace(hiddenNEntry.Text) == "" {
				return fmt.Errorf("для каналу GPRS вкажіть прихований номер")
			}
		}
		return nil
	}

	personalFullName := func(item data.AdminObjectPersonal) string {
		parts := []string{
			strings.TrimSpace(item.Surname),
			strings.TrimSpace(item.Name),
			strings.TrimSpace(item.SecName),
		}
		filtered := make([]string, 0, len(parts))
		for _, p := range parts {
			if p != "" {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return "(без ПІБ)"
		}
		return strings.Join(filtered, " ")
	}

	nextPersonalNumber := func() int64 {
		maxVal := int64(0)
		for _, it := range pendingPersonals {
			if it.Number > maxVal {
				maxVal = it.Number
			}
		}
		return maxVal + 1
	}

	personalTable := widget.NewTable(
		func() (int, int) { return len(pendingPersonals) + 1, 6 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("№")
				case 1:
					lbl.SetText("ПІБ")
				case 2:
					lbl.SetText("Телефон")
				case 3:
					lbl.SetText("Посада")
				case 4:
					lbl.SetText("Доступ")
				default:
					lbl.SetText("Примітка")
				}
				return
			}
			idx := id.Row - 1
			if idx < 0 || idx >= len(pendingPersonals) {
				lbl.SetText("")
				return
			}
			it := pendingPersonals[idx]
			switch id.Col {
			case 0:
				lbl.SetText(strconv.FormatInt(it.Number, 10))
			case 1:
				lbl.SetText(personalFullName(it))
			case 2:
				lbl.SetText(strings.TrimSpace(it.Phones))
			case 3:
				lbl.SetText(strings.TrimSpace(it.Position))
			case 4:
				if it.Access1 > 0 {
					lbl.SetText("Адмін")
				} else {
					lbl.SetText("Оператор")
				}
			case 5:
				lbl.SetText(strings.TrimSpace(it.Notes))
			}
		},
	)
	personalTable.SetColumnWidth(0, 60)
	personalTable.SetColumnWidth(1, 270)
	personalTable.SetColumnWidth(2, 180)
	personalTable.SetColumnWidth(3, 170)
	personalTable.SetColumnWidth(4, 100)
	personalTable.SetColumnWidth(5, 230)
	personalTable.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedPersonal = -1
			return
		}
		idx := id.Row - 1
		if idx < 0 || idx >= len(pendingPersonals) {
			selectedPersonal = -1
			return
		}
		selectedPersonal = idx
	}

	addPersonalBtn := widget.NewButton("Додати", func() {
		showObjectPersonalEditor(win, provider, "Додати В/О", data.AdminObjectPersonal{
			Number: nextPersonalNumber(),
			IsRang: true,
		}, func(item data.AdminObjectPersonal) error {
			if item.Number <= 0 {
				item.Number = nextPersonalNumber()
			}
			pendingPersonals = append(pendingPersonals, item)
			selectedPersonal = len(pendingPersonals) - 1
			personalTable.Refresh()
			return nil
		}, statusLabel, func() {
			statusLabel.SetText(fmt.Sprintf("Додано В/О. Всього: %d", len(pendingPersonals)))
		})
	})
	editPersonalBtn := widget.NewButton("Змінити", func() {
		if selectedPersonal < 0 || selectedPersonal >= len(pendingPersonals) {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		initial := pendingPersonals[selectedPersonal]
		showObjectPersonalEditor(win, provider, "Редагування В/О", initial, func(item data.AdminObjectPersonal) error {
			if item.Number <= 0 {
				item.Number = initial.Number
			}
			pendingPersonals[selectedPersonal] = item
			personalTable.Refresh()
			return nil
		}, statusLabel, func() {
			statusLabel.SetText("В/О оновлено")
		})
	})
	deletePersonalBtn := widget.NewButton("Видалити", func() {
		if selectedPersonal < 0 || selectedPersonal >= len(pendingPersonals) {
			statusLabel.SetText("Виберіть В/О у таблиці")
			return
		}
		target := pendingPersonals[selectedPersonal]
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити В/О \"%s\"?", personalFullName(target)),
			func(ok bool) {
				if !ok {
					return
				}
				pendingPersonals = append(pendingPersonals[:selectedPersonal], pendingPersonals[selectedPersonal+1:]...)
				selectedPersonal = -1
				personalTable.Refresh()
				statusLabel.SetText(fmt.Sprintf("В/О видалено. Залишилось: %d", len(pendingPersonals)))
			},
			win,
		)
	})

	quickZoneNameEntry := widget.NewEntry()
	quickZoneNameEntry.SetPlaceHolder("Назва зони (Enter -> наступна зона)")
	selectedZoneLabel := widget.NewLabel("Зона: —")

	effectiveZoneNumberAt := func(idx int) int64 {
		if idx < 0 || idx >= len(pendingZones) {
			return 0
		}
		if pendingZones[idx].ZoneNumber > 0 {
			return pendingZones[idx].ZoneNumber
		}
		return int64(idx) + 1
	}
	sortPendingZones := func() {
		sort.SliceStable(pendingZones, func(i, j int) bool {
			left := pendingZones[i].ZoneNumber
			right := pendingZones[j].ZoneNumber
			if left <= 0 {
				left = int64(i) + 1
			}
			if right <= 0 {
				right = int64(j) + 1
			}
			return left < right
		})
	}
	findRowByZoneNumber := func(zoneNumber int64) int {
		if zoneNumber <= 0 {
			return -1
		}
		for i := range pendingZones {
			if effectiveZoneNumberAt(i) == zoneNumber {
				return i
			}
		}
		return -1
	}
	updateSelectedZoneLabel := func() {
		if selectedZone < 0 || selectedZone >= len(pendingZones) {
			selectedZoneLabel.SetText("Зона: —")
			return
		}
		selectedZoneLabel.SetText(fmt.Sprintf("Зона: #%d", effectiveZoneNumberAt(selectedZone)))
	}
	ensureZoneExists := func(zoneNumber int64, defaultDescription string) error {
		if zoneNumber <= 0 {
			return fmt.Errorf("некоректний номер зони")
		}
		if findRowByZoneNumber(zoneNumber) >= 0 {
			return nil
		}
		desc := strings.TrimSpace(defaultDescription)
		if desc == "" {
			desc = fmt.Sprintf("Шлейф %d", zoneNumber)
		}
		pendingZones = append(pendingZones, data.AdminObjectZone{
			ZoneNumber:    zoneNumber,
			ZoneType:      1,
			Description:   desc,
			EntryDelaySec: 0,
		})
		sortPendingZones()
		return nil
	}

	zoneTable := widget.NewTable(
		func() (int, int) { return len(pendingZones) + 1, 3 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				switch id.Col {
				case 0:
					lbl.SetText("ZONEN")
				case 1:
					lbl.SetText("Тип")
				default:
					lbl.SetText("Опис")
				}
				return
			}
			idx := id.Row - 1
			if idx < 0 || idx >= len(pendingZones) {
				lbl.SetText("")
				return
			}
			it := pendingZones[idx]
			switch id.Col {
			case 0:
				lbl.SetText(strconv.FormatInt(effectiveZoneNumberAt(idx), 10))
			case 1:
				lbl.SetText("пож.")
			default:
				lbl.SetText(strings.TrimSpace(it.Description))
			}
		},
	)
	zoneTable.StickyRowCount = 1
	zoneTable.StickyColumnCount = 1
	applyZoneTableLayout := func() {
		zoneTable.SetColumnWidth(0, 120)
		zoneTable.SetColumnWidth(1, 120)
		zoneTable.SetColumnWidth(2, 520)
	}
	applyZoneTableLayout()

	var selectZoneByNumber func(zoneNumber int64, focusQuickName bool)
	refreshZoneTable := func(targetZoneNumber int64, focusQuickName bool) {
		sortPendingZones()
		zoneTable.Refresh()
		applyZoneTableLayout()
		if selectZoneByNumber != nil {
			selectZoneByNumber(targetZoneNumber, focusQuickName)
		}
	}
	selectZoneByNumber = func(zoneNumber int64, focusQuickName bool) {
		if len(pendingZones) == 0 {
			selectedZone = -1
			zoneTable.UnselectAll()
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		targetRow := 0
		if zoneNumber > 0 {
			if row := findRowByZoneNumber(zoneNumber); row >= 0 {
				targetRow = row
			}
		}
		selectedZone = targetRow
		zoneTable.Select(widget.TableCellID{Row: targetRow + 1, Col: 0})
		quickZoneNameEntry.SetText(strings.TrimSpace(pendingZones[targetRow].Description))
		updateSelectedZoneLabel()
		if focusQuickName {
			focusIfOnCanvas(win, quickZoneNameEntry)
		}
	}

	zoneTable.OnSelected = func(id widget.TableCellID) {
		if id.Row <= 0 {
			selectedZone = -1
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		idx := id.Row - 1
		if idx < 0 || idx >= len(pendingZones) {
			selectedZone = -1
			quickZoneNameEntry.SetText("")
			updateSelectedZoneLabel()
			return
		}
		selectedZone = idx
		quickZoneNameEntry.SetText(strings.TrimSpace(pendingZones[idx].Description))
		updateSelectedZoneLabel()
		focusIfOnCanvas(win, quickZoneNameEntry)
	}

	moveToNextZone := func() {
		if selectedZone < 0 || selectedZone >= len(pendingZones) {
			if len(pendingZones) == 0 {
				if err := ensureZoneExists(1, strings.TrimSpace(quickZoneNameEntry.Text)); err != nil {
					dialog.ShowError(err, win)
					statusLabel.SetText("Не вдалося додати першу зону")
					return
				}
				refreshZoneTable(1, true)
				statusLabel.SetText("Додано зону #1")
				return
			}
			selectZoneByNumber(effectiveZoneNumberAt(0), true)
		}
		if selectedZone < 0 || selectedZone >= len(pendingZones) {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}

		current := pendingZones[selectedZone]
		currentZoneNumber := effectiveZoneNumberAt(selectedZone)
		if current.ZoneNumber <= 0 {
			current.ZoneNumber = currentZoneNumber
		}
		current.Description = strings.TrimSpace(quickZoneNameEntry.Text)
		pendingZones[selectedZone] = current

		nextZoneNumber := currentZoneNumber + 1
		if err := ensureZoneExists(nextZoneNumber, ""); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося додати наступну зону")
			return
		}

		refreshZoneTable(nextZoneNumber, true)
		statusLabel.SetText(fmt.Sprintf("Збережено зону #%d, перехід на #%d", currentZoneNumber, nextZoneNumber))
	}
	quickZoneNameEntry.OnSubmitted = func(string) {
		moveToNextZone()
	}

	addZoneBtn := widget.NewButton("Додати", func() {
		nextZoneNumber := int64(1)
		if selectedZone >= 0 && selectedZone < len(pendingZones) {
			nextZoneNumber = effectiveZoneNumberAt(selectedZone) + 1
		} else if len(pendingZones) > 0 {
			lastZone := effectiveZoneNumberAt(len(pendingZones) - 1)
			if lastZone > 0 {
				nextZoneNumber = lastZone + 1
			}
		}
		if err := ensureZoneExists(nextZoneNumber, ""); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося додати зону")
			return
		}
		refreshZoneTable(nextZoneNumber, true)
		statusLabel.SetText(fmt.Sprintf("Готово до введення зони #%d", nextZoneNumber))
	})

	editZoneBtn := widget.NewButton("Змінити", func() {
		if len(pendingZones) == 0 {
			if err := ensureZoneExists(1, ""); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося створити першу зону")
				return
			}
			refreshZoneTable(1, true)
			statusLabel.SetText("Створено зону #1, можна вводити назву")
			return
		}
		if selectedZone < 0 || selectedZone >= len(pendingZones) {
			selectZoneByNumber(effectiveZoneNumberAt(0), true)
			statusLabel.SetText("Виберіть зону і вводьте назву")
			return
		}
		updateSelectedZoneLabel()
		focusIfOnCanvas(win, quickZoneNameEntry)
		statusLabel.SetText(fmt.Sprintf("Редагування зони #%d: введіть назву і натисніть Enter", effectiveZoneNumberAt(selectedZone)))
	})

	deleteZoneBtn := widget.NewButton("Видалити", func() {
		if selectedZone < 0 || selectedZone >= len(pendingZones) {
			statusLabel.SetText("Виберіть зону у таблиці")
			return
		}
		targetZone := effectiveZoneNumberAt(selectedZone)
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити зону #%d?", targetZone),
			func(ok bool) {
				if !ok {
					return
				}
				pendingZones = append(pendingZones[:selectedZone], pendingZones[selectedZone+1:]...)
				selectedZone = -1
				refreshZoneTable(0, false)
				statusLabel.SetText(fmt.Sprintf("Зону #%d видалено", targetZone))
			},
			win,
		)
	})

	defaultZoneFillCount := func() int64 {
		maxZone := int64(0)
		for i := range pendingZones {
			if effectiveZoneNumberAt(i) > maxZone {
				maxZone = effectiveZoneNumberAt(i)
			}
		}
		if maxZone > 0 {
			return maxZone
		}
		return 24
	}

	fillZonesBtn := widget.NewButton("Заповнити", func() {
		showZoneFillDialog(win, defaultZoneFillCount(), func(count int64) {
			if count <= 0 {
				statusLabel.SetText("Кількість зон має бути більше 0")
				return
			}
			existingDescriptions := make(map[int64]string, len(pendingZones))
			for i := range pendingZones {
				zoneNumber := effectiveZoneNumberAt(i)
				existingDescriptions[zoneNumber] = strings.TrimSpace(pendingZones[i].Description)
			}
			for zoneNumber := int64(1); zoneNumber <= count; zoneNumber++ {
				if err := ensureZoneExists(zoneNumber, existingDescriptions[zoneNumber]); err != nil {
					dialog.ShowError(err, win)
					statusLabel.SetText("Не вдалося заповнити зони")
					return
				}
			}
			refreshZoneTable(1, false)
			statusLabel.SetText(fmt.Sprintf("Зони заповнено до #%d", count))
		}, statusLabel)
	})

	clearZonesBtn := widget.NewButton("Очистити", func() {
		dialog.ShowConfirm(
			"Підтвердження",
			"Видалити всі зони, додані в майстрі?",
			func(ok bool) {
				if !ok {
					return
				}
				pendingZones = nil
				selectedZone = -1
				refreshZoneTable(0, false)
				statusLabel.SetText("Зони очищено")
			},
			win,
		)
	})

	refreshZonesBtn := widget.NewButton("Оновити", func() {
		refreshZoneTable(0, false)
		statusLabel.SetText(fmt.Sprintf("Зони: %d запис(ів)", len(pendingZones)))
	})
	nextZoneBtn := widget.NewButton("Enter -> Наступна", moveToNextZone)

	mapPickBtn := widget.NewButton("Вибрати на карті", func() {
		showCoordinatesMapPicker(
			win,
			strings.TrimSpace(latitudeEntry.Text),
			strings.TrimSpace(longitudeEntry.Text),
			func(lat, lon string) {
				latitudeEntry.SetText(lat)
				longitudeEntry.SetText(lon)
				statusLabel.SetText("Координати вибрано на карті")
			},
		)
	})
	clearCoordsBtn := widget.NewButton("Очистити", func() {
		latitudeEntry.SetText("")
		longitudeEntry.SetText("")
		statusLabel.SetText("Координати очищено")
	})

	reviewText := widget.NewTextGrid()
	reviewText.SetText("")
	reviewScroll := container.NewScroll(reviewText)
	reviewScroll.SetMinSize(fyne.NewSize(0, 420))

	refreshReview := func() {
		lines := []string{
			"1) Дані об'єкта",
			fmt.Sprintf("№ об'єкта: %s", blankFallback(strings.TrimSpace(objnEntry.Text), "—")),
			fmt.Sprintf("Коротка назва: %s", blankFallback(strings.TrimSpace(shortNameEntry.Text), "—")),
			fmt.Sprintf("Повна назва: %s", blankFallback(strings.TrimSpace(fullNameEntry.Text), "—")),
			fmt.Sprintf("Тип: %s", blankFallback(strings.TrimSpace(objectTypeSelect.Selected), "—")),
			fmt.Sprintf("Район: %s", blankFallback(strings.TrimSpace(regionSelect.Selected), "—")),
			fmt.Sprintf("Прихований №: %s", blankFallback(strings.TrimSpace(hiddenNEntry.Text), "—")),
			fmt.Sprintf("Адреса: %s", blankFallback(strings.TrimSpace(addressEntry.Text), "—")),
			fmt.Sprintf("Телефони: %s", blankFallback(strings.TrimSpace(phonesEntry.Text), "—")),
			fmt.Sprintf("Договір: %s", blankFallback(strings.TrimSpace(contractEntry.Text), "—")),
			fmt.Sprintf("Дата: %s", blankFallback(strings.TrimSpace(dateEntry.Text), "—")),
			fmt.Sprintf("Розташування: %s", blankFallback(strings.TrimSpace(locationEntry.Text), "—")),
			fmt.Sprintf("Інформація: %s", blankFallback(strings.TrimSpace(notesEntry.Text), "—")),
			"",
			"2) Параметри пристрою",
			fmt.Sprintf("Канал: %s", blankFallback(strings.TrimSpace(channelCodeSelect.Selected), "—")),
			fmt.Sprintf("ППК: %s", blankFallback(strings.TrimSpace(ppkSelect.Selected), "—")),
			fmt.Sprintf("Підсервер A: %s", blankFallback(strings.TrimSpace(subServerASelect.Selected), "—")),
			fmt.Sprintf("Підсервер B: %s", blankFallback(strings.TrimSpace(subServerBSelect.Selected), "—")),
			fmt.Sprintf("SIM 1: %s", blankFallback(strings.TrimSpace(sim1Entry.Text), "—")),
			fmt.Sprintf("SIM 2: %s", blankFallback(strings.TrimSpace(sim2Entry.Text), "—")),
			fmt.Sprintf("Контроль тестів: %t", testControlCheck.Checked),
			fmt.Sprintf("Інтервал тесту, хв: %s", blankFallback(strings.TrimSpace(testIntervalEntry.Text), "—")),
			"",
			"3) Зв'язані дані",
			fmt.Sprintf("В/О: %d", len(pendingPersonals)),
			fmt.Sprintf("Зони: %d", len(pendingZones)),
			fmt.Sprintf("Координати: %s / %s", blankFallback(strings.TrimSpace(latitudeEntry.Text), "0"), blankFallback(strings.TrimSpace(longitudeEntry.Text), "0")),
		}
		if _, err := buildCardFromUI(); err != nil {
			lines = append(lines, "")
			lines = append(lines, "Увага: перед створенням виправте:")
			lines = append(lines, "- "+err.Error())
		}
		if len(pendingPersonals) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Список В/О:")
			for i, it := range pendingPersonals {
				role := "Оператор"
				if it.Access1 > 0 {
					role = "Адмін"
				}
				lines = append(lines, fmt.Sprintf("%d) #%d %s, %s, %s", i+1, it.Number, personalFullName(it), role, strings.TrimSpace(it.Phones)))
			}
		}
		if len(pendingZones) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Список зон:")
			for i, it := range pendingZones {
				lines = append(lines, fmt.Sprintf("%d) ZONEN=%d, %s", i+1, effectiveZoneNumberAt(i), strings.TrimSpace(it.Description)))
			}
		}
		reviewText.SetText(strings.Join(lines, "\n"))
	}

	step1 := widget.NewForm(
		widget.NewFormItem("Об'єктовий номер:", objnEntry),
		widget.NewFormItem("Коротка назва:", shortNameEntry),
		widget.NewFormItem("Повна назва:", fullNameEntry),
		widget.NewFormItem("Тип об'єкта:", objectTypeSelect),
		widget.NewFormItem("Район:", regionSelect),
		widget.NewFormItem("Канал:", channelCodeSelect),
		widget.NewFormItem("", hiddenNRow),
		widget.NewFormItem("Адреса:", addressEntry),
		widget.NewFormItem("Телефони:", phonesEntry),
		widget.NewFormItem("Договір:", contractEntry),
		widget.NewFormItem("Дата:", dateRow),
		widget.NewFormItem("Розташування:", locationEntry),
		widget.NewFormItem("Інформація:", notesEntry),
	)

	step2 := widget.NewForm(
		widget.NewFormItem("ППК:", ppkSelect),
		widget.NewFormItem("Підсервер A:", subServerASelect),
		widget.NewFormItem("Підсервер B:", subServerBSelect),
		widget.NewFormItem("SIM 1:", container.NewVBox(sim1Entry, sim1UsageLabel)),
		widget.NewFormItem("SIM 2:", container.NewVBox(sim2Entry, sim2UsageLabel)),
		widget.NewFormItem("Контроль GPRS:", container.NewHBox(testControlCheck, widget.NewLabel("Інтервал (хв.):"), container.NewGridWrap(fyne.NewSize(100, 36), testIntervalEntry))),
	)

	step3 := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addPersonalBtn, editPersonalBtn, deletePersonalBtn),
			widget.NewSeparator(),
		),
		nil,
		nil,
		nil,
		personalTable,
	)

	step4 := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addZoneBtn, editZoneBtn, deleteZoneBtn, fillZonesBtn, clearZonesBtn, layout.NewSpacer(), refreshZonesBtn),
			widget.NewSeparator(),
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewBorder(
				nil,
				nil,
				container.NewHBox(widget.NewLabel("Швидке введення:"), layout.NewSpacer(), selectedZoneLabel),
				nextZoneBtn,
				quickZoneNameEntry,
			),
		),
		nil,
		nil,
		zoneTable,
	)

	step5 := widget.NewForm(
		widget.NewFormItem("Широта:", latitudeEntry),
		widget.NewFormItem("Довгота:", longitudeEntry),
		widget.NewFormItem("", container.NewHBox(mapPickBtn, clearCoordsBtn)),
	)

	step6 := container.NewBorder(
		widget.NewLabel("Підсумок введених даних перед створенням"),
		nil, nil, nil,
		reviewScroll,
	)

	steps := container.NewAppTabs(
		container.NewTabItem("1. Дані об'єкта", container.NewPadded(step1)),
		container.NewTabItem("2. Параметри пристрою", container.NewPadded(step2)),
		container.NewTabItem("3. Відповідальні", container.NewPadded(step3)),
		container.NewTabItem("4. Зони", container.NewPadded(step4)),
		container.NewTabItem("5. Додаткова інфо", container.NewPadded(step5)),
		container.NewTabItem("6. Підтвердження", container.NewPadded(step6)),
	)

	stepTitles := []string{
		"дані об'єкта",
		"параметри пристрою",
		"відповідальні особи",
		"зони",
		"додаткова інформація",
		"підтвердження",
	}

	currentStep := 0
	updateStepState := func() {
		steps.SelectIndex(currentStep)
		stepName := ""
		if currentStep >= 0 && currentStep < len(stepTitles) {
			stepName = stepTitles[currentStep]
		}
		statusLabel.SetText(fmt.Sprintf("Крок %d/%d: %s", currentStep+1, len(stepTitles), stepName))
		if currentStep == len(stepTitles)-1 {
			refreshReview()
		}
	}

	backBtn := widget.NewButton("Назад", nil)
	nextBtn := widget.NewButton("Далі", nil)
	createBtn := widget.NewButton("Створити", nil)
	cancelBtn := widget.NewButton("Скасувати", func() { win.Close() })

	refreshButtons := func() {
		if currentStep <= 0 {
			backBtn.Disable()
		} else {
			backBtn.Enable()
		}
		if currentStep >= len(stepTitles)-1 {
			nextBtn.Disable()
			createBtn.Enable()
		} else {
			nextBtn.Enable()
			createBtn.Disable()
		}
	}

	backBtn.OnTapped = func() {
		if currentStep <= 0 {
			return
		}
		currentStep--
		updateStepState()
		refreshButtons()
	}

	nextBtn.OnTapped = func() {
		if err := validateStep(currentStep); err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		if currentStep >= len(stepTitles)-1 {
			return
		}
		currentStep++
		updateStepState()
		refreshButtons()
	}

	createBtn.OnTapped = func() {
		if err := validateStep(len(stepTitles) - 1); err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		card, err := buildCardFromUI()
		if err != nil {
			statusLabel.SetText(err.Error())
			return
		}
		if err := provider.CreateObject(card); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося створити об'єкт")
			return
		}

		warnings := make([]string, 0, 4)

		for idx, it := range pendingPersonals {
			if err := provider.AddObjectPersonal(card.ObjN, it); err != nil {
				warnings = append(warnings, fmt.Sprintf("В/О #%d не додано: %v", idx+1, err))
			}
		}
		for idx, it := range pendingZones {
			if err := provider.AddObjectZone(card.ObjN, it); err != nil {
				warnings = append(warnings, fmt.Sprintf("Зона #%d не додана: %v", idx+1, err))
			}
		}
		coords := data.AdminObjectCoordinates{
			Latitude:  strings.TrimSpace(latitudeEntry.Text),
			Longitude: strings.TrimSpace(longitudeEntry.Text),
		}
		if err := provider.SaveObjectCoordinates(card.ObjN, coords); err != nil {
			warnings = append(warnings, fmt.Sprintf("Координати не збережено: %v", err))
		}

		if len(warnings) > 0 {
			dialog.ShowInformation(
				"Створено з попередженнями",
				"Об'єкт створено, але частина додаткових даних не збережена:\n\n"+strings.Join(warnings, "\n"),
				win,
			)
		} else {
			statusLabel.SetText("Новий об'єкт створено")
		}
		if onSaved != nil {
			onSaved(card.ObjN)
		}
		win.Close()
	}

	content := container.NewBorder(
		nil,
		container.NewHBox(statusLabel, layout.NewSpacer(), backBtn, nextBtn, createBtn, cancelBtn),
		nil, nil,
		steps,
	)
	win.SetContent(content)

	if err := loadReferenceData(); err != nil {
		dialog.ShowError(err, win)
		statusLabel.SetText("Не вдалося завантажити довідники")
	}
	fillDefaults()
	updateStepState()
	refreshButtons()
	personalTable.Refresh()
	refreshZoneTable(0, false)

	win.Show()
}
