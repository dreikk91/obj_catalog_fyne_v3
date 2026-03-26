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

	"obj_catalog_fyne_v3/pkg/contracts"
)

func ShowStatisticsDialog(parent fyne.Window, provider contracts.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Збір статистики")
	win.Resize(fyne.NewSize(1440, 820))

	statusLabel := widget.NewLabel("Готово")
	summaryLabel := widget.NewLabel("Поки що немає даних")
	summaryLabel.Wrapping = fyne.TextWrapWord

	connectionRadio := widget.NewRadioGroup([]string{
		"Всі об'єкти",
		"Зв'язок норма",
		"Без зв'язку",
	}, nil)
	connectionRadio.Horizontal = true
	connectionRadio.SetSelected("Всі об'єкти")

	channelOptions := []string{"Всі протоколи", "Автододзвон", "Мост", "Нова"}
	channelSelect := widget.NewSelect(channelOptions, nil)
	channelSelect.SetSelected("Всі протоколи")

	guardSelect := widget.NewSelect([]string{
		"Всі режими",
		"0 - знято",
		"1 - під охороною",
		"2 - тривога",
		"3 - інше",
	}, nil)
	guardSelect.SetSelected("Всі режими")

	blockSelect := widget.NewSelect([]string{
		"Всі об'єкти",
		"Тимчасово зняті з нагляду",
		"В режимі налагодження",
	}, nil)
	blockSelect.SetSelected("Всі об'єкти")

	typeSelect := widget.NewSelect([]string{"Всі типи"}, nil)
	typeSelect.SetSelected("Всі типи")
	regionSelect := widget.NewSelect([]string{"Всі райони"}, nil)
	regionSelect.SetSelected("Всі райони")

	searchEntry := widget.NewEntry()
	searchEntry.SetPlaceHolder("Пошук по № об'єкта або назві")

	limitEntry := widget.NewEntry()
	limitEntry.SetText("5000")
	limitEntry.SetPlaceHolder("Ліміт")

	sortSelect := widget.NewSelect([]string{
		"№ об'єкта ↑",
		"№ об'єкта ↓",
		"Назва А-Я",
		"Канал",
		"Режим",
		"Тип",
		"Район",
		"Зв'язок",
	}, nil)
	sortSelect.SetSelected("№ об'єкта ↑")

	typeLabelToID := map[string]int64{"Всі типи": 0}
	regionLabelToID := map[string]int64{"Всі райони": 0}
	channelLabelToProtocol := map[string]contracts.AdminStatisticsProtocolFilter{
		"Всі протоколи": contracts.StatsProtocolAll,
		"Автододзвон":   contracts.StatsProtocolAutodial,
		"Мост":          contracts.StatsProtocolMost,
		"Нова":          contracts.StatsProtocolNova,
	}

	var (
		rows []contracts.AdminStatisticsRow
	)

	columns := []string{
		"OBJN", "Назва", "Повна назва", "Адреса", "Телефони",
		"Договір", "Дата", "Розташування", "Примітка", "Канал",
		"ППК", "SIM1", "SIM2", "HIDN", "SBSA",
		"SBSB", "TESTCTL", "TESTMIN", "Режим", "Зв'язок",
		"Тривога", "Тех", "Блок.", "Тип", "Район", "GRPN", "OBJUIN",
	}

	getCell := func(item contracts.AdminStatisticsRow, col int) string {
		switch col {
		case 0:
			return strconv.FormatInt(item.ObjN, 10)
		case 1:
			return item.ShortName
		case 2:
			return item.FullName
		case 3:
			return item.Address
		case 4:
			return item.Phones
		case 5:
			return item.Contract
		case 6:
			return item.StartDate
		case 7:
			return item.Location
		case 8:
			return item.Notes
		case 9:
			return fmt.Sprintf("%d (%s)", item.ChannelCode, channelCodeCaption(item.ChannelCode))
		case 10:
			return strconv.FormatInt(item.PPKID, 10)
		case 11:
			return item.GSMPhone1
		case 12:
			return item.GSMPhone2
		case 13:
			return strconv.FormatInt(item.GSMHiddenN, 10)
		case 14:
			return item.SubServerA
		case 15:
			return item.SubServerB
		case 16:
			return binaryCaption(item.TestControl)
		case 17:
			return strconv.FormatInt(item.TestTime, 10)
		case 18:
			return guardStateCaption(item.GuardState)
		case 19:
			return connStateCaption(item.IsConnState)
		case 20:
			return binaryCaption(item.AlarmState)
		case 21:
			return binaryCaption(item.TechAlarmState)
		case 22:
			return blockModeCaption(item.BlockMode)
		case 23:
			if strings.TrimSpace(item.ObjTypeName) == "" {
				return strconv.FormatInt(item.ObjTypeID, 10)
			}
			return fmt.Sprintf("%s [%d]", item.ObjTypeName, item.ObjTypeID)
		case 24:
			if strings.TrimSpace(item.RegionName) == "" {
				return strconv.FormatInt(item.RegionID, 10)
			}
			return fmt.Sprintf("%s [%d]", item.RegionName, item.RegionID)
		case 25:
			return strconv.FormatInt(item.GrpN, 10)
		case 26:
			return strconv.FormatInt(item.ObjUIN, 10)
		default:
			return ""
		}
	}

	table := widget.NewTable(
		func() (int, int) { return len(rows) + 1, len(columns) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("cell")
			lbl.Truncation = fyne.TextTruncateClip
			return lbl
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.SetText(columns[id.Col])
				return
			}
			idx := id.Row - 1
			if idx < 0 || idx >= len(rows) {
				lbl.SetText("")
				return
			}
			lbl.SetText(getCell(rows[idx], id.Col))
		},
	)
	table.StickyRowCount = 1

	columnWidths := []float32{
		80, 220, 260, 240, 180,
		140, 120, 220, 240, 130,
		80, 160, 160, 70, 90,
		90, 90, 90, 130, 110,
		80, 80, 140, 200, 180, 70, 90,
	}
	for i := range columns {
		table.SetColumnWidth(i, columnWidths[i])
	}

	sortRows := func() {
		sortMode := sortSelect.Selected
		sort.SliceStable(rows, func(i, j int) bool {
			left := rows[i]
			right := rows[j]
			switch sortMode {
			case "№ об'єкта ↓":
				return left.ObjN > right.ObjN
			case "Назва А-Я":
				return strings.ToLower(left.ShortName) < strings.ToLower(right.ShortName)
			case "Канал":
				if left.ChannelCode == right.ChannelCode {
					return left.ObjN < right.ObjN
				}
				return left.ChannelCode < right.ChannelCode
			case "Режим":
				if left.GuardState == right.GuardState {
					return left.ObjN < right.ObjN
				}
				return left.GuardState < right.GuardState
			case "Тип":
				if left.ObjTypeID == right.ObjTypeID {
					return left.ObjN < right.ObjN
				}
				return left.ObjTypeID < right.ObjTypeID
			case "Район":
				if left.RegionID == right.RegionID {
					return left.ObjN < right.ObjN
				}
				return left.RegionID < right.RegionID
			case "Зв'язок":
				if left.IsConnState == right.IsConnState {
					return left.ObjN < right.ObjN
				}
				return left.IsConnState > right.IsConnState
			default:
				return left.ObjN < right.ObjN
			}
		})
	}

	updateSummary := func() {
		total := len(rows)
		online := 0
		offline := 0
		alarm := 0
		tech := 0
		blocked := 0
		for _, it := range rows {
			if it.IsConnState > 0 {
				online++
			} else {
				offline++
			}
			if it.AlarmState > 0 {
				alarm++
			}
			if it.TechAlarmState > 0 {
				tech++
			}
			if it.BlockMode != contracts.DisplayBlockNone {
				blocked++
			}
		}
		summaryLabel.SetText(fmt.Sprintf(
			"Всього: %d | Зв'язок: %d | Без зв'язку: %d | Тривога: %d | Тех: %d | Блоковані: %d",
			total, online, offline, alarm, tech, blocked,
		))
	}

	parseLimit := func() int {
		raw := strings.TrimSpace(limitEntry.Text)
		if raw == "" {
			return 5000
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return 5000
		}
		if n > 50000 {
			return 50000
		}
		return n
	}

	buildFilter := func() contracts.AdminStatisticsFilter {
		filter := contracts.AdminStatisticsFilter{
			ConnectionMode: contracts.StatsConnectionAll,
			Search:         strings.TrimSpace(searchEntry.Text),
		}
		switch connectionRadio.Selected {
		case "Зв'язок норма":
			filter.ConnectionMode = contracts.StatsConnectionOnline
		case "Без зв'язку":
			filter.ConnectionMode = contracts.StatsConnectionOffline
		}

		if protocol, ok := channelLabelToProtocol[channelSelect.Selected]; ok {
			filter.ProtocolFilter = protocol
		}
		switch guardSelect.Selected {
		case "0 - знято":
			v := int64(0)
			filter.GuardState = &v
		case "1 - під охороною":
			v := int64(1)
			filter.GuardState = &v
		case "2 - тривога":
			v := int64(2)
			filter.GuardState = &v
		case "3 - інше":
			v := int64(3)
			filter.GuardState = &v
		}

		if id, ok := typeLabelToID[typeSelect.Selected]; ok && id > 0 {
			v := id
			filter.ObjTypeID = &v
		}
		if id, ok := regionLabelToID[regionSelect.Selected]; ok && id > 0 {
			v := id
			filter.RegionID = &v
		}

		switch blockSelect.Selected {
		case "Тимчасово зняті з нагляду":
			v := contracts.DisplayBlockTemporaryOff
			filter.BlockMode = &v
		case "В режимі налагодження":
			v := contracts.DisplayBlockDebug
			filter.BlockMode = &v
		}

		return filter
	}

	reload := func() {
		loaded, err := provider.CollectObjectStatistics(buildFilter(), parseLimit())
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося зібрати статистику")
			return
		}
		rows = loaded
		sortRows()
		table.Refresh()
		updateSummary()
		statusLabel.SetText(fmt.Sprintf("Завантажено записів: %d", len(rows)))
	}

	loadReferenceOptions := func() {
		types, err := provider.ListObjectTypes()
		if err == nil {
			options := []string{"Всі типи"}
			typeLabelToID = map[string]int64{"Всі типи": 0}
			for _, it := range types {
				name := strings.TrimSpace(it.Name)
				if name == "" {
					name = fmt.Sprintf("Тип %d", it.ID)
				}
				label := fmt.Sprintf("%s [%d]", name, it.ID)
				options = append(options, label)
				typeLabelToID[label] = it.ID
			}
			typeSelect.Options = options
			typeSelect.SetSelected("Всі типи")
			typeSelect.Refresh()
		}

		regions, err := provider.ListObjectDistricts()
		if err == nil {
			options := []string{"Всі райони"}
			regionLabelToID = map[string]int64{"Всі райони": 0}
			for _, it := range regions {
				name := strings.TrimSpace(it.Name)
				if name == "" {
					name = fmt.Sprintf("Район %d", it.ID)
				}
				label := fmt.Sprintf("%s [%d]", name, it.ID)
				options = append(options, label)
				regionLabelToID[label] = it.ID
			}
			regionSelect.Options = options
			regionSelect.SetSelected("Всі райони")
			regionSelect.Refresh()
		}
	}

	executeBtn := widget.NewButton("Виконати", reload)
	refreshBtn := widget.NewButton("Оновити", reload)
	exportBtn := widget.NewButton("Експорт CSV", func() {
		dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, win)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()

			header := make([]string, 0, len(columns))
			for _, col := range columns {
				header = append(header, escapeCSVCell(col))
			}
			lines := []string{strings.Join(header, ";")}
			for _, it := range rows {
				cells := make([]string, 0, len(columns))
				for colIdx := range columns {
					cells = append(cells, escapeCSVCell(getCell(it, colIdx)))
				}
				lines = append(lines, strings.Join(cells, ";"))
			}
			content := strings.Join(lines, "\n")
			if _, err := uc.Write([]byte(content)); err != nil {
				dialog.ShowError(err, win)
				return
			}
			statusLabel.SetText(fmt.Sprintf("Експортовано: %s", uriPathToLocalPath(uc.URI().Path())))
		}, win).Show()
	})
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	sortSelect.OnChanged = func(string) {
		sortRows()
		table.Refresh()
	}
	searchEntry.OnSubmitted = func(string) { reload() }
	limitEntry.OnSubmitted = func(string) { reload() }

	filters := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Підключення:"),
			connectionRadio,
		),
		container.NewHBox(
			widget.NewLabel("Протокол:"),
			container.NewGridWrap(fyne.NewSize(220, 36), channelSelect),
			widget.NewLabel("Режим:"),
			container.NewGridWrap(fyne.NewSize(180, 36), guardSelect),
			widget.NewLabel("Тип:"),
			container.NewGridWrap(fyne.NewSize(260, 36), typeSelect),
		),
		container.NewHBox(
			widget.NewLabel("Район:"),
			container.NewGridWrap(fyne.NewSize(260, 36), regionSelect),
			widget.NewLabel("Блокування:"),
			container.NewGridWrap(fyne.NewSize(210, 36), blockSelect),
			widget.NewLabel("Ліміт:"),
			container.NewGridWrap(fyne.NewSize(90, 36), limitEntry),
		),
		container.NewHBox(
			widget.NewLabel("Пошук:"),
			searchEntry,
			widget.NewLabel("Сортування:"),
			container.NewGridWrap(fyne.NewSize(180, 36), sortSelect),
			executeBtn,
			refreshBtn,
			exportBtn,
		),
		widget.NewSeparator(),
		summaryLabel,
	)

	content := container.NewBorder(
		filters,
		container.NewHBox(statusLabel, layout.NewSpacer(), widget.NewLabel(time.Now().Format("02.01.2006")), closeBtn),
		nil, nil,
		table,
	)
	win.SetContent(content)

	loadReferenceOptions()
	reload()
	win.Show()
}

func channelCodeCaption(code int64) string {
	switch code {
	case 1:
		return "Автододзвон"
	case 5:
		return "GPRS"
	case 6:
		return "AVD"
	case 7:
		return "GPRS+AVD"
	default:
		if code == 0 {
			return "невизначено"
		}
		return fmt.Sprintf("канал %d", code)
	}
}

func guardStateCaption(state int64) string {
	switch state {
	case 0:
		return "0 (знято)"
	case 1:
		return "1 (охорона)"
	case 2:
		return "2 (тривога)"
	default:
		return strconv.FormatInt(state, 10)
	}
}

func connStateCaption(v int64) string {
	if v > 0 {
		return "є зв'язок"
	}
	return "без зв'язку"
}

func binaryCaption(v int64) string {
	if v > 0 {
		return "1"
	}
	return "0"
}

func blockModeCaption(mode contracts.DisplayBlockMode) string {
	switch mode {
	case contracts.DisplayBlockTemporaryOff:
		return "блок. постановки"
	case contracts.DisplayBlockDebug:
		return "налагодження"
	default:
		return "немає"
	}
}
