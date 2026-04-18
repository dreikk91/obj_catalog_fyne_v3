package dialogs

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

type mostStatisticsDialogState struct {
	win      fyne.Window
	provider adminv1.StatisticsProvider

	statusLabel  *widget.Label
	summaryLabel *widget.Label

	connectionRadio *widget.RadioGroup
	channelSelect   *widget.Select
	guardSelect     *widget.Select
	blockSelect     *widget.Select
	typeSelect      *widget.Select
	regionSelect    *widget.Select
	searchEntry     *widget.Entry
	limitEntry      *widget.Entry
	sortSelect      *widget.Select
	table           *widget.Table

	typeLabelToID          map[string]int64
	regionLabelToID        map[string]int64
	channelLabelToProtocol map[string]adminv1.StatisticsProtocolFilter
	rows                   []adminv1.StatisticsRow
}

var statisticsColumns = []string{
	"OBJN", "Назва", "Повна назва", "Адреса", "Телефони",
	"Договір", "Дата", "Розташування", "Примітка", "Канал",
	"ППК", "SIM1", "SIM2", "HIDN", "SBSA",
	"SBSB", "TESTCTL", "TESTMIN", "Режим", "Зв'язок",
	"Тривога", "Тех", "Блок.", "Тип", "Район", "GRPN", "OBJUIN",
}

var statisticsColumnWidths = []float32{
	80, 220, 260, 240, 180,
	140, 120, 220, 240, 130,
	80, 160, 160, 70, 90,
	90, 90, 90, 130, 110,
	80, 80, 140, 200, 180, 70, 90,
}

func ShowMostStatisticsDialog(parent fyne.Window, provider adminv1.StatisticsProvider) {
	state := newMostStatisticsDialogState(provider)
	state.win.SetContent(state.buildContent())
	state.bindActions()
	state.loadReferenceOptions()
	state.reload()
	state.win.Show()
}

func newMostStatisticsDialogState(provider adminv1.StatisticsProvider) *mostStatisticsDialogState {
	state := &mostStatisticsDialogState{
		win:             fyne.CurrentApp().NewWindow("Мост статистика"),
		provider:        provider,
		typeLabelToID:   map[string]int64{"Всі типи": 0},
		regionLabelToID: map[string]int64{"Всі райони": 0},
		channelLabelToProtocol: map[string]adminv1.StatisticsProtocolFilter{
			"Всі протоколи": adminv1.StatisticsProtocolAll,
			"Автододзвон":   adminv1.StatisticsProtocolAutodial,
			"Мост":          adminv1.StatisticsProtocolMost,
			"Нова":          adminv1.StatisticsProtocolNova,
		},
		rows: []adminv1.StatisticsRow{},
	}
	state.win.Resize(fyne.NewSize(1360, 820))
	state.initControls()
	state.table = state.buildTable()

	return state
}

func (s *mostStatisticsDialogState) initControls() {
	s.statusLabel = widget.NewLabel("Готово")
	s.summaryLabel = widget.NewLabel("Поки що немає даних")
	s.summaryLabel.Wrapping = fyne.TextWrapWord

	s.connectionRadio = widget.NewRadioGroup([]string{
		"Всі об'єкти",
		"Зв'язок норма",
		"Без зв'язку",
	}, nil)
	s.connectionRadio.Horizontal = true
	s.connectionRadio.SetSelected("Всі об'єкти")

	s.channelSelect = widget.NewSelect([]string{"Всі протоколи", "Автододзвон", "Мост", "Нова"}, nil)
	s.channelSelect.SetSelected("Всі протоколи")

	s.guardSelect = widget.NewSelect([]string{
		"Всі режими",
		"0 - знято",
		"1 - під охороною",
		"2 - тривога",
		"3 - інше",
	}, nil)
	s.guardSelect.SetSelected("Всі режими")

	s.blockSelect = widget.NewSelect([]string{
		"Всі об'єкти",
		"Тимчасово зняті з нагляду",
		"В режимі налагодження",
	}, nil)
	s.blockSelect.SetSelected("Всі об'єкти")

	s.typeSelect = widget.NewSelect([]string{"Всі типи"}, nil)
	s.typeSelect.SetSelected("Всі типи")

	s.regionSelect = widget.NewSelect([]string{"Всі райони"}, nil)
	s.regionSelect.SetSelected("Всі райони")

	s.searchEntry = widget.NewEntry()
	s.searchEntry.SetPlaceHolder("Пошук по № об'єкта або назві")

	s.limitEntry = widget.NewEntry()
	s.limitEntry.SetText("5000")
	s.limitEntry.SetPlaceHolder("Ліміт")

	s.sortSelect = widget.NewSelect([]string{
		"№ об'єкта ↑",
		"№ об'єкта ↓",
		"Назва А-Я",
		"Канал",
		"Режим",
		"Тип",
		"Район",
		"Зв'язок",
	}, nil)
	s.sortSelect.SetSelected("№ об'єкта ↑")
}

func (s *mostStatisticsDialogState) buildContent() fyne.CanvasObject {
	closeBtn := makeIconButton("Закрити", iconClose(), widget.LowImportance, func() {
		s.win.Close()
	})

	return container.NewBorder(
		container.NewVBox(s.buildFiltersCard(), widget.NewCard("Зведення", "", s.summaryLabel)),
		container.NewHBox(
			s.statusLabel,
			layout.NewSpacer(),
			widget.NewLabel(time.Now().Format("02.01.2006")),
			closeBtn,
		),
		nil,
		nil,
		s.table,
	)
}

func (s *mostStatisticsDialogState) buildFiltersCard() fyne.CanvasObject {
	executeBtn := makePrimaryButton("Виконати", s.reload)
	refreshBtn := makeIconButton("Оновити", iconRefresh(), widget.MediumImportance, s.reload)
	exportBtn := makeIconButton("Експорт CSV", iconExport(), widget.LowImportance, s.exportCSV)

	return widget.NewCard("Фільтри", "", container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Підключення:"),
			s.connectionRadio,
			layout.NewSpacer(),
			widget.NewLabel("Протокол:"),
			container.NewGridWrap(fyne.NewSize(220, 36), s.channelSelect),
			widget.NewLabel("Режим:"),
			container.NewGridWrap(fyne.NewSize(190, 36), s.guardSelect),
		),
		container.NewHBox(
			widget.NewLabel("Тип:"),
			container.NewGridWrap(fyne.NewSize(300, 36), s.typeSelect),
			widget.NewLabel("Район:"),
			container.NewGridWrap(fyne.NewSize(300, 36), s.regionSelect),
			layout.NewSpacer(),
			widget.NewLabel("Блокування:"),
			container.NewGridWrap(fyne.NewSize(240, 36), s.blockSelect),
			widget.NewLabel("Ліміт:"),
			container.NewGridWrap(fyne.NewSize(100, 36), s.limitEntry),
		),
		container.NewBorder(
			nil,
			nil,
			widget.NewLabel("Пошук:"),
			container.NewHBox(
				widget.NewLabel("Сортування:"),
				container.NewGridWrap(fyne.NewSize(190, 36), s.sortSelect),
				executeBtn,
				refreshBtn,
				exportBtn,
			),
			s.searchEntry,
		),
	))
}

func (s *mostStatisticsDialogState) buildTable() *widget.Table {
	table := widget.NewTable(
		func() (int, int) { return len(s.rows) + 1, len(statisticsColumns) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("cell")
			label.Truncation = fyne.TextTruncateClip
			return label
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			s.updateTableCell(id, obj.(*widget.Label))
		},
	)
	table.StickyRowCount = 1
	for index := range statisticsColumns {
		table.SetColumnWidth(index, statisticsColumnWidths[index])
	}

	return table
}

func (s *mostStatisticsDialogState) bindActions() {
	s.sortSelect.OnChanged = func(string) {
		s.sortRows()
		s.table.Refresh()
	}
	s.searchEntry.OnSubmitted = func(string) {
		s.reload()
	}
	s.limitEntry.OnSubmitted = func(string) {
		s.reload()
	}
}

func (s *mostStatisticsDialogState) updateTableCell(id widget.TableCellID, label *widget.Label) {
	if id.Row == 0 {
		label.TextStyle = fyne.TextStyle{Bold: true}
		label.SetText(statisticsColumns[id.Col])
		return
	}

	label.TextStyle = fyne.TextStyle{}
	index := id.Row - 1
	if index < 0 || index >= len(s.rows) {
		label.SetText("")
		return
	}

	label.SetText(statisticsCellValue(s.rows[index], id.Col))
}

func (s *mostStatisticsDialogState) sortRows() {
	sortMode := s.sortSelect.Selected
	slices.SortStableFunc(s.rows, func(left, right adminv1.StatisticsRow) int {
		switch sortMode {
		case "№ об'єкта ↓":
			return cmp.Compare(right.ObjN, left.ObjN)
		case "Назва А-Я":
			if diff := cmp.Compare(strings.ToLower(left.ShortName), strings.ToLower(right.ShortName)); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		case "Канал":
			if diff := cmp.Compare(left.ChannelCode, right.ChannelCode); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		case "Режим":
			if diff := cmp.Compare(adminGuardSortRank(left.GuardStatus), adminGuardSortRank(right.GuardStatus)); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		case "Тип":
			if diff := cmp.Compare(left.ObjTypeID, right.ObjTypeID); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		case "Район":
			if diff := cmp.Compare(left.RegionID, right.RegionID); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		case "Зв'язок":
			if diff := cmp.Compare(adminConnectionSortRank(right.ConnectionStatus), adminConnectionSortRank(left.ConnectionStatus)); diff != 0 {
				return diff
			}
			return cmp.Compare(left.ObjN, right.ObjN)
		default:
			return cmp.Compare(left.ObjN, right.ObjN)
		}
	})
}

func (s *mostStatisticsDialogState) updateSummary() {
	total := len(s.rows)
	online := 0
	offline := 0
	alarm := 0
	tech := 0
	blocked := 0

	for _, row := range s.rows {
		if row.ConnectionStatus == frontendv1.ConnectionStatusOnline {
			online++
		} else {
			offline++
		}
		if row.VisualSeverity == frontendv1.VisualSeverityCritical {
			alarm++
		}
		if row.TechAlarmState > 0 || (row.VisualSeverity == frontendv1.VisualSeverityWarning && row.AlarmState == 0) {
			tech++
		}
		if row.MonitoringStatus != frontendv1.MonitoringStatusActive {
			blocked++
		}
	}

	s.summaryLabel.SetText(fmt.Sprintf(
		"Всього: %d | Зв'язок: %d | Без зв'язку: %d | Тривога: %d | Тех: %d | Блоковані: %d",
		total,
		online,
		offline,
		alarm,
		tech,
		blocked,
	))
}

func (s *mostStatisticsDialogState) parseLimit() int {
	raw := strings.TrimSpace(s.limitEntry.Text)
	if raw == "" {
		return 5000
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 5000
	}
	if limit > 50000 {
		return 50000
	}

	return limit
}

func (s *mostStatisticsDialogState) buildFilter() adminv1.StatisticsFilter {
	filter := adminv1.StatisticsFilter{
		ConnectionMode: adminv1.StatisticsConnectionModeAll,
		Search:         strings.TrimSpace(s.searchEntry.Text),
	}

	switch s.connectionRadio.Selected {
	case "Зв'язок норма":
		filter.ConnectionMode = adminv1.StatisticsConnectionModeOnline
	case "Без зв'язку":
		filter.ConnectionMode = adminv1.StatisticsConnectionModeOffline
	}

	if protocol, ok := s.channelLabelToProtocol[s.channelSelect.Selected]; ok {
		filter.ProtocolFilter = protocol
	}

	switch s.guardSelect.Selected {
	case "0 - знято":
		filter.GuardState = i64(0)
	case "1 - під охороною":
		filter.GuardState = i64(1)
	case "2 - тривога":
		filter.GuardState = i64(2)
	case "3 - інше":
		filter.GuardState = i64(3)
	}

	if id := s.typeLabelToID[s.typeSelect.Selected]; id > 0 {
		filter.ObjTypeID = &id
	}
	if id := s.regionLabelToID[s.regionSelect.Selected]; id > 0 {
		filter.RegionID = &id
	}

	switch s.blockSelect.Selected {
	case "Тимчасово зняті з нагляду":
		mode := adminv1.DisplayBlockModeTemporaryOff
		filter.BlockMode = &mode
	case "В режимі налагодження":
		mode := adminv1.DisplayBlockModeDebug
		filter.BlockMode = &mode
	}

	return filter
}

func (s *mostStatisticsDialogState) reload() {
	loadedRows, err := s.provider.CollectObjectStatistics(s.buildFilter(), s.parseLimit())
	if err != nil {
		dialog.ShowError(err, s.win)
		s.statusLabel.SetText("Не вдалося зібрати Мост статистику")
		return
	}

	s.rows = loadedRows
	s.sortRows()
	s.table.Refresh()
	s.updateSummary()
	s.statusLabel.SetText(fmt.Sprintf("Завантажено записів: %d", len(s.rows)))
}

func (s *mostStatisticsDialogState) loadReferenceOptions() {
	s.loadObjectTypeOptions()
	s.loadRegionOptions()
}

func (s *mostStatisticsDialogState) loadObjectTypeOptions() {
	types, err := s.provider.ListObjectTypes()
	if err != nil {
		return
	}

	options := []string{"Всі типи"}
	s.typeLabelToID = map[string]int64{"Всі типи": 0}
	for _, item := range types {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("Тип %d", item.ID)
		}
		label := fmt.Sprintf("%s [%d]", name, item.ID)
		options = append(options, label)
		s.typeLabelToID[label] = item.ID
	}

	s.typeSelect.Options = options
	s.typeSelect.SetSelected("Всі типи")
	s.typeSelect.Refresh()
}

func (s *mostStatisticsDialogState) loadRegionOptions() {
	regions, err := s.provider.ListObjectDistricts()
	if err != nil {
		return
	}

	options := []string{"Всі райони"}
	s.regionLabelToID = map[string]int64{"Всі райони": 0}
	for _, item := range regions {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("Район %d", item.ID)
		}
		label := fmt.Sprintf("%s [%d]", name, item.ID)
		options = append(options, label)
		s.regionLabelToID[label] = item.ID
	}

	s.regionSelect.Options = options
	s.regionSelect.SetSelected("Всі райони")
	s.regionSelect.Refresh()
}

func (s *mostStatisticsDialogState) exportCSV() {
	dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, s.win)
			return
		}
		if uc == nil {
			return
		}
		defer uc.Close()

		content := s.buildCSVContent()
		if _, err := uc.Write([]byte(content)); err != nil {
			dialog.ShowError(err, s.win)
			return
		}

		s.statusLabel.SetText(fmt.Sprintf("Експортовано: %s", uriPathToLocalPath(uc.URI().Path())))
	}, s.win).Show()
}

func (s *mostStatisticsDialogState) buildCSVContent() string {
	header := make([]string, 0, len(statisticsColumns))
	for _, column := range statisticsColumns {
		header = append(header, escapeCSVCell(column))
	}

	lines := []string{strings.Join(header, ";")}
	for _, row := range s.rows {
		cells := make([]string, 0, len(statisticsColumns))
		for columnIndex := range statisticsColumns {
			cells = append(cells, escapeCSVCell(statisticsCellValue(row, columnIndex)))
		}
		lines = append(lines, strings.Join(cells, ";"))
	}

	return strings.Join(lines, "\n")
}

func statisticsCellValue(item adminv1.StatisticsRow, column int) string {
	switch column {
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
		return guardStatusCaption(item.GuardStatus, item.GuardState)
	case 19:
		return connectionStatusCaption(item.ConnectionStatus)
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

func guardStatusCaption(status frontendv1.GuardStatus, rawState int64) string {
	switch status {
	case frontendv1.GuardStatusDisarmed:
		return "0 (знято)"
	case frontendv1.GuardStatusGuarded:
		switch rawState {
		case 1:
			return "1 (охорона)"
		case 2:
			return "2 (тривога)"
		default:
			if rawState > 0 {
				return fmt.Sprintf("%d (охорона)", rawState)
			}
		}
	}
	return strconv.FormatInt(rawState, 10)
}

func connectionStatusCaption(status frontendv1.ConnectionStatus) string {
	if status == frontendv1.ConnectionStatusOnline {
		return "є зв'язок"
	}
	return "без зв'язку"
}

func adminGuardSortRank(status frontendv1.GuardStatus) int {
	switch status {
	case frontendv1.GuardStatusDisarmed:
		return 0
	case frontendv1.GuardStatusGuarded:
		return 1
	default:
		return 2
	}
}

func adminConnectionSortRank(status frontendv1.ConnectionStatus) int {
	switch status {
	case frontendv1.ConnectionStatusOnline:
		return 1
	case frontendv1.ConnectionStatusOffline:
		return 0
	default:
		return -1
	}
}

func binaryCaption(v int64) string {
	if v > 0 {
		return "1"
	}
	return "0"
}

func blockModeCaption(mode adminv1.DisplayBlockMode) string {
	switch mode {
	case adminv1.DisplayBlockModeTemporaryOff:
		return "блок. постановки"
	case adminv1.DisplayBlockModeDebug:
		return "налагодження"
	default:
		return "немає"
	}
}
