//go:build qt

package qtui

import (
	"fmt"
	"image/color"
	"runtime"
	"sort"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type AlarmPanel struct {
	*qt.QWidget
	sourceFilter   *qt.QComboBox
	severityFilter *qt.QComboBox
	statusLabel    *qt.QLabel
	criticalLabel  *qt.QLabel
	normalLabel    *qt.QLabel
	filteredLabel  *qt.QLabel
	selectionLabel *qt.QLabel
	processButton  *qt.QPushButton
	pickButton     *qt.QPushButton
	responseButton *qt.QPushButton
	historyButton  *qt.QPushButton
	hideHistoryBtn *qt.QPushButton
	table          *qt.QTableView
	model          *qt.QStandardItemModel
	vm             *viewmodels.AlarmListViewModel
	caseHistoryVM  *viewmodels.WorkAreaCaseHistoryViewModel
	historyBrowser *qt.QTextBrowser
	dataProvider   contracts.DataProvider
	prefs          config.Preferences

	autoSized       bool
	filterUpdating  bool
	allAlarms       []models.Alarm
	alarmsByID      map[int]models.Alarm
	groupsByKey     map[string]alarmGroup
	rowsSignature   string
	rowsReady       bool
	selectedAlarmID int

	OnAlarmSelected func(models.Alarm)
	OnProcessAlarms func([]models.Alarm)
	OnPickAlarms    func([]models.Alarm)
	OnRespondAlarm  func(models.Alarm)
	OnCountChanged  func(count int)
}

type alarmGroup struct {
	Key           string
	Source        string
	ObjectID      int
	ObjectNumber  string
	ObjectName    string
	Address       string
	Alarms        []models.Alarm
	Primary       models.Alarm
	CriticalCount int
	LatestAt      int64
	LatestTime    string
}

func NewAlarmPanel(prefs config.Preferences) *AlarmPanel {
	panel := &AlarmPanel{
		QWidget:       qt.NewQWidget2(),
		vm:            viewmodels.NewAlarmListViewModel(),
		caseHistoryVM: viewmodels.NewWorkAreaCaseHistoryViewModel(),
		alarmsByID:    map[int]models.Alarm{},
		groupsByKey:   map[string]alarmGroup{},
		prefs:         prefs,
	}
	layout := qt.NewQVBoxLayout(panel.QWidget)
	panel.model = qt.NewQStandardItemModel2(0, 6)
	panel.model.SetHorizontalHeaderLabels(alarmGroupHeaders())
	addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", ""})

	header := qt.NewQFrame2()
	header.SetStyleSheet(`
		QFrame {
			background: ` + qtSurfaceColor + `;
			border: 1px solid ` + qtBorderColor + `;
			border-radius: 3px;
		}
	`)
	headerLayout := qt.NewQVBoxLayout(header.QWidget)
	headerLayout.SetContentsMargins(6, 4, 6, 4)
	headerLayout.SetSpacing(4)

	toolbar := qt.NewQHBoxLayout2()
	panel.statusLabel = qt.NewQLabel3("Тривог немає")
	panel.statusLabel.SetMinimumWidth(168)
	panel.statusLabel.SetStyleSheet("font-weight: 700; color: #3D9C3B; border: 0; background: transparent;")
	toolbar.AddWidget(panel.statusLabel.QWidget)

	panel.criticalLabel = newAlarmMetricLabel("Критичні", "#C62828", "#FFEBEE")
	panel.normalLabel = newAlarmMetricLabel("Звичайні", "#FF8F00", "#FFF3E0")
	panel.filteredLabel = newAlarmMetricLabel("Груп", qtPrimaryColor, qtAltSurfaceColor)
	panel.selectionLabel = newAlarmMetricLabel("Вибрано", "#3D9C3B", "#E8F5E9")
	toolbar.AddWidget(panel.criticalLabel.QWidget)
	toolbar.AddWidget(panel.normalLabel.QWidget)
	toolbar.AddWidget(panel.filteredLabel.QWidget)
	toolbar.AddWidget(panel.selectionLabel.QWidget)

	panel.processButton = qt.NewQPushButton3("Відпрацювати")
	panel.processButton.SetToolTip("Відпрацювати вибрані тривоги")
	panel.processButton.OnClicked(func() {
		panel.processSelectedAlarms()
	})
	panel.pickButton = qt.NewQPushButton3("Взяти в роботу")
	panel.pickButton.SetToolTip("Закріпити вибрані тривоги за оператором")
	panel.pickButton.OnClicked(func() {
		panel.pickSelectedAlarms()
	})
	panel.responseButton = qt.NewQPushButton3("Реагування")
	panel.responseButton.SetToolTip("Відкрити картку реагування та керування МГР")
	panel.responseButton.OnClicked(func() {
		panel.respondToSelectedAlarm()
	})
	panel.historyButton = qt.NewQPushButton3("Хронологія")
	panel.historyButton.SetToolTip("Показати хронологію вибраної групи")
	panel.historyButton.OnClicked(func() {
		panel.showSelectedHistory()
	})
	panel.hideHistoryBtn = qt.NewQPushButton3("Сховати")
	panel.hideHistoryBtn.SetToolTip("Сховати хронологію")
	panel.hideHistoryBtn.OnClicked(func() {
		panel.hideCaseHistory()
	})
	panel.processButton.SetEnabled(false)
	panel.pickButton.SetEnabled(false)
	panel.responseButton.SetEnabled(false)
	panel.historyButton.SetEnabled(false)
	toolbar.AddWidget(panel.processButton.QWidget)
	toolbar.AddWidget(panel.pickButton.QWidget)
	toolbar.AddWidget(panel.responseButton.QWidget)
	toolbar.AddWidget(panel.historyButton.QWidget)
	toolbar.AddWidget(panel.hideHistoryBtn.QWidget)

	panel.sourceFilter = qt.NewQComboBox2()
	panel.sourceFilter.AddItems(viewmodels.BuildObjectSourceOptions(0, 0, 0, 0))
	panel.sourceFilter.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.applyFilters()
	})
	panel.severityFilter = qt.NewQComboBox2()
	panel.severityFilter.AddItems([]string{"Всі тривоги", "Критичні", "Звичайні"})
	panel.severityFilter.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.applyFilters()
	})
	criticalFilterButton := qt.NewQPushButton3("Критичні")
	criticalFilterButton.SetToolTip("Показати тільки критичні тривоги")
	criticalFilterButton.OnClicked(func() {
		panel.severityFilter.SetCurrentText("Критичні")
	})
	allFilterButton := qt.NewQPushButton3("Всі")
	allFilterButton.SetToolTip("Показати всі тривоги")
	allFilterButton.OnClicked(func() {
		panel.severityFilter.SetCurrentText("Всі тривоги")
	})
	normalFilterButton := qt.NewQPushButton3("Звичайні")
	normalFilterButton.SetToolTip("Показати тільки звичайні тривоги")
	normalFilterButton.OnClicked(func() {
		panel.severityFilter.SetCurrentText("Звичайні")
	})
	toolbar.AddStretch()
	toolbar.AddWidget(criticalFilterButton.QWidget)
	toolbar.AddWidget(allFilterButton.QWidget)
	toolbar.AddWidget(normalFilterButton.QWidget)
	toolbar.AddWidget(panel.sourceFilter.QWidget)
	toolbar.AddWidget(panel.severityFilter.QWidget)
	headerLayout.AddLayout(toolbar.QLayout)

	panel.table = qt.NewQTableView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetSelectionMode(qt.QAbstractItemView__ExtendedSelection)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.HorizontalHeader().SetStretchLastSection(true)
	panel.table.VerticalHeader().SetVisible(false)
	panel.table.SetStyleSheet(`
		QTableView {
			gridline-color: #e7e7e7;
			selection-background-color: ` + qtPrimaryColor + `;
			selection-color: white;
		}
		QHeaderView::section {
			background: #f7f7f7;
			border: 0;
			border-bottom: 1px solid #d5d5d5;
			padding: 5px;
			font-weight: 600;
		}
	`)
	panel.table.OnDoubleClicked(func(index *qt.QModelIndex) {
		if alarm, ok := panel.alarmAtIndex(index); ok {
			if panel.OnRespondAlarm != nil {
				panel.OnRespondAlarm(alarm)
			} else if panel.OnProcessAlarms != nil {
				panel.OnProcessAlarms([]models.Alarm{alarm})
			}
		}
	})
	panel.table.SetContextMenuPolicy(qt.CustomContextMenu)
	panel.table.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		panel.showContextMenu(pos)
	})

	panel.table.SelectionModel().OnCurrentRowChanged(func(current *qt.QModelIndex, previous *qt.QModelIndex) {
		if alarm, ok := panel.alarmAtIndex(current); ok {
			panel.selectedAlarmID = alarm.ID
			if panel.OnAlarmSelected != nil {
				panel.OnAlarmSelected(alarm)
			}
			panel.updateSelectionState()
			return
		}
		panel.selectedAlarmID = 0
		panel.updateSelectionState()
	})
	panel.table.SelectionModel().OnSelectionChanged(func(selected *qt.QItemSelection, deselected *qt.QItemSelection) {
		panel.updateSelectionState()
	})

	topWidget := qt.NewQWidget2()
	topLayout := qt.NewQVBoxLayout(topWidget)
	topLayout.AddWidget(header.QWidget)
	topLayout.AddWidget(panel.table.QWidget)
	topWidget.SetLayout(topLayout.QLayout)

	panel.historyBrowser = qt.NewQTextBrowser(nil)
	panel.clearCaseHistory()
	panel.historyBrowser.SetVisible(false)

	splitter := qt.NewQSplitter3(qt.Vertical)
	splitter.AddWidget(topWidget)
	splitter.AddWidget(panel.historyBrowser.QWidget)
	splitter.SetSizes([]int{900, 0})

	layout.AddWidget(splitter.QWidget)
	panel.SetLayout(layout.QLayout)
	panel.updateRibbonStats(nil, nil)
	panel.updateSelectionState()
	return panel
}

func newAlarmMetricLabel(title string, color string, background string) *qt.QLabel {
	label := qt.NewQLabel3(title + ": 0")
	label.SetMinimumWidth(92)
	label.SetStyleSheet(fmt.Sprintf(
		"font-weight: 700; color: %s; background: %s; border: 1px solid %s; border-radius: 3px; padding: 3px 6px;",
		color,
		background,
		color,
	))
	return label
}

func alarmGroupHeaders() []string {
	return []string{"Остання", "№", "Об'єкт", "Кейс", "Пріоритет", "Джерело"}
}

func buildAlarmGroups(alarms []models.Alarm) []alarmGroup {
	if len(alarms) == 0 {
		return nil
	}

	byKey := make(map[string]*alarmGroup)
	order := make([]string, 0, len(alarms))
	for _, alarm := range alarms {
		source := viewmodels.ObjectSourceByID(alarm.ObjectID)
		key := source + ":" + strconv.Itoa(alarm.ObjectID)
		group, ok := byKey[key]
		if !ok {
			group = &alarmGroup{
				Key:          key,
				Source:       source,
				ObjectID:     alarm.ObjectID,
				ObjectNumber: alarm.GetObjectNumberDisplay(),
				ObjectName:   strings.TrimSpace(alarm.ObjectName),
				Address:      strings.TrimSpace(alarm.Address),
				Primary:      alarm,
				LatestAt:     alarm.Time.UnixNano(),
				LatestTime:   alarm.GetTimeDisplay(),
			}
			byKey[key] = group
			order = append(order, key)
		}
		group.Alarms = append(group.Alarms, alarm)
		if alarm.Time.UnixNano() > group.LatestAt {
			group.LatestAt = alarm.Time.UnixNano()
			group.LatestTime = alarm.GetTimeDisplay()
		}
		if alarm.IsCritical() {
			group.CriticalCount++
			if !group.Primary.IsCritical() || alarm.Time.After(group.Primary.Time) {
				group.Primary = alarm
			}
		} else if group.CriticalCount == 0 && alarm.Time.After(group.Primary.Time) {
			group.Primary = alarm
		}
	}

	groups := make([]alarmGroup, 0, len(order))
	for _, key := range order {
		group := byKey[key]
		sort.SliceStable(group.Alarms, func(i, j int) bool {
			return group.Alarms[i].Time.After(group.Alarms[j].Time)
		})
		if group.ObjectName == "" {
			group.ObjectName = "Об'єкт"
		}
		groups = append(groups, *group)
	}
	sort.SliceStable(groups, func(i, j int) bool {
		leftCritical := groups[i].CriticalCount > 0
		rightCritical := groups[j].CriticalCount > 0
		if leftCritical != rightCritical {
			return leftCritical
		}
		return groups[i].Primary.Time.After(groups[j].Primary.Time)
	})
	return groups
}

func alarmGroupCaseText(group alarmGroup) string {
	if len(group.Alarms) == 0 {
		return ""
	}
	parts := []string{strconv.Itoa(len(group.Alarms)) + " трив."}
	if group.CriticalCount > 0 {
		parts = append(parts, strconv.Itoa(group.CriticalCount)+" крит.")
	}
	if group.Primary.ZoneNumber > 0 {
		parts = append(parts, "зона "+strconv.Itoa(group.Primary.ZoneNumber))
	}
	eventText := strings.TrimSpace(group.Primary.GetTypeDisplay())
	if group.Primary.Details != "" {
		eventText += " - " + strings.TrimSpace(group.Primary.Details)
	}
	if eventText != "" {
		parts = append(parts, eventText)
	}
	return strings.Join(parts, " | ")
}

func addColoredReadOnlyGroupRow(model *qt.QStandardItemModel, values []string, groupKey string, textColor color.NRGBA, rowColor color.NRGBA) {
	items := make([]*qt.QStandardItem, 0, len(values))
	foreground := qt.NewQColor11(int(textColor.R), int(textColor.G), int(textColor.B), int(textColor.A)).ToQVariant()
	background := qt.NewQColor11(int(rowColor.R), int(rowColor.G), int(rowColor.B), int(rowColor.A)).ToQVariant()
	for idx, value := range values {
		item := qt.NewQStandardItem2(value)
		item.SetEditable(false)
		item.SetData(foreground, int(qt.ForegroundRole))
		item.SetData(background, int(qt.BackgroundRole))
		if idx == 0 {
			item.SetData(qt.NewQVariant14(groupKey), int(qt.UserRole))
		}
		items = append(items, item)
	}
	model.AppendRow(items)
}

func (panel *AlarmPanel) SetDataProvider(provider contracts.DataProvider) {
	if panel == nil {
		return
	}
	panel.dataProvider = provider
}

func (panel *AlarmPanel) SetAlarms(alarms []models.Alarm) {
	if panel == nil {
		return
	}
	panel.allAlarms = append(panel.allAlarms[:0], alarms...)
	panel.applyFilters()
}

func (panel *AlarmPanel) applyColumnWidths() {
	if panel.autoSized {
		return
	}
	resizeTableToContentsWithMinimums("alarms", panel.table)
	panel.autoSized = true
}

func (panel *AlarmPanel) applyFilters() {
	if panel == nil || panel.model == nil || panel.vm == nil {
		return
	}
	selectedSource := viewmodels.ObjectSourceAll
	if panel.sourceFilter != nil {
		selectedSource = viewmodels.NormalizeObjectSourceFilter(panel.sourceFilter.CurrentText())
	}
	out := panel.vm.BuildRefreshOutput(viewmodels.AlarmRefreshInput{
		Alarms:         panel.allAlarms,
		LastKnownIDs:   map[int]struct{}{},
		SelectedSource: selectedSource,
	})
	if panel.sourceFilter != nil {
		panel.filterUpdating = true
		updateComboItems(panel.sourceFilter, viewmodels.BuildObjectSourceOptions(out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL), selectedSource)
		panel.filterUpdating = false
	}

	filtered := filterAlarmsBySeverity(out.FilteredAlarms, panel.currentSeverityFilter())
	groups := buildAlarmGroups(filtered)
	panel.updateRibbonStats(out.FilteredAlarms, groups)
	if panel.OnCountChanged != nil {
		panel.OnCountChanged(len(filtered))
	}
	panel.alarmsByID = make(map[int]models.Alarm, len(filtered))
	panel.groupsByKey = make(map[string]alarmGroup, len(groups))
	signature := alarmGroupRowsSignature(groups)
	if panel.rowsReady && panel.rowsSignature == signature {
		for _, alarm := range filtered {
			panel.alarmsByID[alarm.ID] = alarm
		}
		for _, group := range groups {
			panel.groupsByKey[group.Key] = group
		}
		panel.updateSelectionState()
		return
	}
	panel.rowsSignature = signature
	panel.rowsReady = true

	var columnWidths []int
	if panel.autoSized {
		columnWidths = captureTableColumnWidths(panel.table)
	}
	panel.model.Clear()
	panel.model.SetHorizontalHeaderLabels(alarmGroupHeaders())
	if len(groups) == 0 {
		addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", ""})
		panel.selectedAlarmID = 0
		panel.hideCaseHistory()
		restoreTableColumnWidthsSnapshot("alarms", panel.table, columnWidths)
		panel.updateSelectionState()
		return
	}
	for _, alarm := range filtered {
		panel.alarmsByID[alarm.ID] = alarm
	}
	for _, group := range groups {
		panel.groupsByKey[group.Key] = group
		priority := "звичайна"
		if group.CriticalCount > 0 {
			priority = "критична"
		}
		textColor, rowColor := eventRowColorsBySeverity(group.Primary.VisualSeverityValue(), group.Primary.SC1)
		addColoredReadOnlyGroupRow(panel.model, []string{
			group.LatestTime,
			group.ObjectNumber,
			strings.TrimSpace(group.ObjectName),
			alarmGroupCaseText(group),
			priority,
			group.Source,
		}, group.Key, textColor, rowColor)
	}
	if restoreTableColumnWidthsSnapshot("alarms", panel.table, columnWidths) {
		panel.updateSelectionState()
		return
	}
	panel.applyColumnWidths()
	panel.updateSelectionState()
}

func alarmGroupRowsSignature(groups []alarmGroup) string {
	var b strings.Builder
	for _, group := range groups {
		fmt.Fprintf(
			&b,
			"%s:%d:%s:%d:%d:%s:%s:%s|",
			group.Key,
			group.ObjectID,
			group.LatestTime,
			group.CriticalCount,
			len(group.Alarms),
			group.ObjectNumber,
			strings.TrimSpace(group.ObjectName),
			alarmGroupCaseText(group),
		)
		for _, alarm := range group.Alarms {
			fmt.Fprintf(&b, "%d:%d:%d:%s;", alarm.ID, alarm.Time.UnixNano(), alarm.SC1, strings.TrimSpace(alarm.Details))
		}
		b.WriteByte('|')
	}
	return b.String()
}

func (panel *AlarmPanel) updateRibbonStats(sourceFiltered []models.Alarm, groups []alarmGroup) {
	if panel == nil {
		return
	}
	criticalCount := 0
	for _, alarm := range sourceFiltered {
		if alarm.IsCritical() {
			criticalCount++
		}
	}
	normalCount := len(sourceFiltered) - criticalCount
	displayedCount := len(groups)
	if panel.statusLabel != nil {
		switch {
		case len(sourceFiltered) == 0:
			panel.statusLabel.SetText("Тривог немає")
			panel.statusLabel.SetStyleSheet("font-weight: 600; color: #3D9C3B; border: 0; background: transparent;")
		case criticalCount > 0:
			panel.statusLabel.SetText("Негайна увага")
			panel.statusLabel.SetStyleSheet("font-weight: 700; color: #b3261e; border: 0; background: transparent;")
		default:
			panel.statusLabel.SetText("Активні події")
			panel.statusLabel.SetStyleSheet("font-weight: 600; color: #8a5a00; border: 0; background: transparent;")
		}
	}
	if panel.criticalLabel != nil {
		panel.criticalLabel.SetText("Критичні: " + strconv.Itoa(criticalCount))
	}
	if panel.normalLabel != nil {
		panel.normalLabel.SetText("Звичайні: " + strconv.Itoa(normalCount))
	}
	if panel.filteredLabel != nil {
		panel.filteredLabel.SetText("Груп: " + strconv.Itoa(displayedCount))
	}
}

func (panel *AlarmPanel) updateSelectionState() {
	if panel == nil {
		return
	}
	selectedGroups, selectedAlarms := panel.selectedGroupAndAlarmCounts()
	alarms := panel.selectedAlarms()
	if panel.selectionLabel != nil {
		panel.selectionLabel.SetText("Вибрано: " + strconv.Itoa(selectedGroups) + "/" + strconv.Itoa(selectedAlarms))
	}
	hasSelection := selectedAlarms > 0
	if panel.processButton != nil {
		panel.processButton.SetEnabled(hasSelection && canProcessAlarms(alarms))
		if selectedAlarms > 1 {
			panel.processButton.SetText("Відпрацювати (" + strconv.Itoa(selectedAlarms) + ")")
		} else {
			panel.processButton.SetText("Відпрацювати")
		}
	}
	if panel.pickButton != nil {
		panel.pickButton.SetEnabled(hasSelection && canTakeAlarms(alarms))
		if selectedAlarms > 1 {
			panel.pickButton.SetText("Взяти в роботу (" + strconv.Itoa(selectedAlarms) + ")")
		} else {
			panel.pickButton.SetText("Взяти в роботу")
		}
	}
	if panel.responseButton != nil {
		panel.responseButton.SetEnabled(selectedGroups == 1)
	}
	if panel.historyButton != nil {
		panel.historyButton.SetEnabled(selectedGroups == 1)
	}
}

func (panel *AlarmPanel) currentSeverityFilter() string {
	if panel == nil || panel.severityFilter == nil {
		return "Всі тривоги"
	}
	return panel.severityFilter.CurrentText()
}

func (panel *AlarmPanel) selectedAlarms() []models.Alarm {
	if panel == nil || panel.table == nil || panel.model == nil {
		return nil
	}
	selection := panel.table.SelectionModel().SelectedRows()
	if len(selection) == 0 {
		return nil
	}
	alarms := make([]models.Alarm, 0, len(selection))
	for _, index := range selection {
		if index.Column() != 0 {
			continue
		}
		groupKey := panel.model.Data(&index, int(qt.UserRole)).ToString()
		if group, ok := panel.groupsByKey[groupKey]; ok {
			alarms = append(alarms, group.Alarms...)
		}
	}
	return alarms
}

func (panel *AlarmPanel) selectedGroupAndAlarmCounts() (int, int) {
	if panel == nil || panel.table == nil || panel.model == nil {
		return 0, 0
	}
	selection := panel.table.SelectionModel().SelectedRows()
	groupCount := 0
	alarmCount := 0
	for _, index := range selection {
		if index.Column() != 0 {
			continue
		}
		groupKey := panel.model.Data(&index, int(qt.UserRole)).ToString()
		if group, ok := panel.groupsByKey[groupKey]; ok {
			groupCount++
			alarmCount += len(group.Alarms)
		}
	}
	return groupCount, alarmCount
}

func (panel *AlarmPanel) processSelectedAlarms() {
	if panel == nil || panel.OnProcessAlarms == nil {
		return
	}
	alarms := panel.selectedAlarms()
	if len(alarms) > 0 {
		panel.OnProcessAlarms(alarms)
	}
}

func (panel *AlarmPanel) pickSelectedAlarms() {
	if panel == nil || panel.OnPickAlarms == nil {
		return
	}
	alarms := panel.selectedAlarms()
	if len(alarms) > 0 {
		panel.OnPickAlarms(alarms)
	}
}

func (panel *AlarmPanel) respondToSelectedAlarm() {
	if panel == nil || panel.OnRespondAlarm == nil {
		return
	}
	index := panel.table.CurrentIndex()
	alarm, ok := panel.alarmAtIndex(index)
	if !ok {
		return
	}
	panel.OnRespondAlarm(alarm)
}

func (panel *AlarmPanel) showContextMenu(pos *qt.QPoint) {
	if panel == nil || panel.table == nil || pos == nil {
		return
	}
	index := panel.table.IndexAt(pos)
	if !index.IsValid() {
		return
	}
	group, ok := panel.groupAtIndex(index)
	if !ok {
		return
	}
	alarms := contextMenuAlarms(group.Alarms, panel.selectedAlarms())
	if !selectionContainsAnyAlarm(panel.selectedAlarms(), group.Alarms) {
		panel.table.SelectRow(index.Row())
	}

	menu := qt.NewQMenu(panel.table.QWidget)
	processAction := menu.AddActionWithText(alarmActionText("Відпрацювати", alarms))
	processAction.SetEnabled(canProcessAlarms(alarms))
	processAction.OnTriggered(func() {
		if panel.OnProcessAlarms != nil {
			panel.OnProcessAlarms(alarms)
		}
	})

	pickAction := menu.AddActionWithText(alarmActionText("Взяти в роботу", alarms))
	pickAction.SetEnabled(canTakeAlarms(alarms))
	pickAction.OnTriggered(func() {
		if panel.OnPickAlarms != nil {
			panel.OnPickAlarms(alarms)
		}
	})

	if panel.OnRespondAlarm != nil {
		responseAction := menu.AddActionWithText("Відкрити картку реагування")
		responseAction.SetEnabled(len(group.Alarms) > 0)
		responseAction.OnTriggered(func() {
			panel.OnRespondAlarm(group.Primary)
		})
	}

	menu.AddSeparator()
	historyAction := menu.AddActionWithText("Переглянути хронологію групи")
	historyAction.OnTriggered(func() {
		panel.showGroupHistory(group)
	})

	menu.AddSeparator()
	addTableCopyActions(menu, panel.table, index)
	menu.AddSeparator()
	addTableColumnActions(menu, "alarms", panel.table, panel.prefs, func() {
		panel.autoSized = true
	}, func() {
		panel.autoSized = false
	})
	menu.ExecWithPos(panel.table.MapToGlobalWithQPoint(pos))
}

func canProcessAlarms(alarms []models.Alarm) bool {
	if len(alarms) == 0 {
		return false
	}
	for _, alarm := range alarms {
		if !alarm.CanProcess {
			return false
		}
	}
	return true
}

func canTakeAlarms(alarms []models.Alarm) bool {
	if len(alarms) == 0 {
		return false
	}
	for _, alarm := range alarms {
		if alarm.IsInProgress && !alarm.CanTakeOver {
			return false
		}
	}
	return true
}

func contextMenuAlarms(clicked []models.Alarm, selected []models.Alarm) []models.Alarm {
	if len(selected) <= 1 {
		return clicked
	}
	if selectionContainsAnyAlarm(selected, clicked) {
		return selected
	}
	return clicked
}

func selectionContainsAnyAlarm(selected []models.Alarm, candidates []models.Alarm) bool {
	for _, candidate := range candidates {
		for _, alarm := range selected {
			if alarm.ID == candidate.ID {
				return true
			}
		}
	}
	return false
}

func alarmActionText(action string, alarms []models.Alarm) string {
	if len(alarms) <= 1 {
		return action
	}
	return fmt.Sprintf("%s (%d)", action, len(alarms))
}

func (panel *AlarmPanel) alarmAtIndex(index *qt.QModelIndex) (models.Alarm, bool) {
	group, ok := panel.groupAtIndex(index)
	if !ok {
		return models.Alarm{}, false
	}
	return group.Primary, true
}

func (panel *AlarmPanel) groupAtIndex(index *qt.QModelIndex) (alarmGroup, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return alarmGroup{}, false
	}
	parent := qt.NewQModelIndex()
	rowIndex := panel.model.Index(index.Row(), 0, parent)
	if rowIndex == nil || !rowIndex.IsValid() {
		runtime.KeepAlive(parent)
		return alarmGroup{}, false
	}
	groupKey := panel.model.Data(rowIndex, int(qt.UserRole)).ToString()
	group, ok := panel.groupsByKey[groupKey]
	runtime.KeepAlive(parent)
	runtime.KeepAlive(rowIndex)
	return group, ok
}

func (panel *AlarmPanel) showSelectedHistory() {
	if panel == nil || panel.table == nil {
		return
	}
	current := panel.table.SelectionModel().CurrentIndex()
	group, ok := panel.groupAtIndex(current)
	if !ok {
		return
	}
	panel.showGroupHistory(group)
}

func (panel *AlarmPanel) showGroupHistory(group alarmGroup) {
	if panel == nil {
		return
	}
	if panel.historyBrowser != nil {
		panel.historyBrowser.SetVisible(true)
	}
	panel.viewAlarmHistory(group.Primary)
}

func (panel *AlarmPanel) hideCaseHistory() {
	if panel == nil || panel.historyBrowser == nil {
		return
	}
	panel.clearCaseHistory()
	panel.historyBrowser.SetVisible(false)
}

func (panel *AlarmPanel) loadCaseHistoryForAlarm(alarm models.Alarm) {
	if panel == nil || panel.caseHistoryVM == nil || panel.dataProvider == nil {
		panel.clearCaseHistory()
		return
	}

	uiCfg := config.LoadUIConfig(panel.prefs)
	useBridgeActiveHistory := !ids.IsCASLObjectID(alarm.ObjectID) &&
		!ids.IsPhoenixObjectID(alarm.ObjectID) &&
		uiCfg.NormalizedBridgeAlarmHistoryMode() == config.BridgeAlarmHistoryModeActiveOnly

	if useBridgeActiveHistory {
		if historyProvider, ok := panel.dataProvider.(contracts.ActiveAlarmHistoryProvider); ok {
			panel.showCaseHistoryLoading(alarm)
			go func(selected models.Alarm) {
				msgs := historyProvider.GetActiveAlarmSourceMessages(selected)
				runOnMainThread(func() {
					if panel.selectedAlarmID != selected.ID {
						return
					}
					if len(msgs) == 0 {
						panel.showEmptyCaseHistory(selected)
						return
					}
					panel.showCaseHistorySourceMessages(selected, msgs)
				})
			}(alarm)
			return
		}
	}

	if len(alarm.SourceMsgs) > 0 {
		panel.showCaseHistorySourceMessages(alarm, alarm.SourceMsgs)
		return
	}

	if historyProvider, ok := panel.dataProvider.(contracts.AlarmHistoryProvider); ok && !ids.IsCASLObjectID(alarm.ObjectID) {
		panel.showCaseHistoryLoading(alarm)
		go func(selected models.Alarm) {
			msgs := historyProvider.GetAlarmSourceMessages(selected)
			runOnMainThread(func() {
				if panel.selectedAlarmID != selected.ID {
					return
				}
				if len(msgs) == 0 {
					panel.showEmptyCaseHistory(selected)
					return
				}
				panel.showCaseHistorySourceMessages(selected, msgs)
			})
		}(alarm)
		return
	}

	if !ids.IsCASLObjectID(alarm.ObjectID) {
		panel.clearCaseHistory()
		return
	}

	panel.showCaseHistoryLoading(alarm)
	go func(selected models.Alarm) {
		events := panel.dataProvider.GetObjectEvents(strconv.Itoa(selected.ObjectID))
		object := &models.Object{ID: selected.ObjectID, Name: selected.ObjectName}
		group, ok := panel.caseHistoryVM.FindGroupForAlarm(object, selected, events)
		runOnMainThread(func() {
			if panel.selectedAlarmID != selected.ID {
				return
			}
			if !ok {
				panel.showEmptyCaseHistory(selected)
				return
			}
			panel.showCaseHistoryGroup(selected, group)
		})
	}(alarm)
}

func (panel *AlarmPanel) showCaseHistoryLoading(alarm models.Alarm) {
	title := alarmSourceDisplayName(alarm.ObjectID) + ": №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}
	panel.historyBrowser.SetHtml(fmt.Sprintf("<div style='color: %s; font-family: Segoe UI; padding: 10px;'><b>%s</b><br/>Завантаження хронології...</div>", qtPrimaryColor, title))
}

func (panel *AlarmPanel) showEmptyCaseHistory(alarm models.Alarm) {
	title := alarmSourceDisplayName(alarm.ObjectID) + ": №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}
	panel.historyBrowser.SetHtml(fmt.Sprintf("<div style='color: #666; font-family: Segoe UI; padding: 10px;'><b>%s</b><br/>Подій за період тривоги не знайдено</div>", title))
}

func (panel *AlarmPanel) showCaseHistorySourceMessages(alarm models.Alarm, sourceMsgs []models.AlarmMsg) {
	msgs := prepareSourceMessagesForDisplay(alarm, sourceMsgs, panel.prefs.StringWithFallback(config.PrefBridgeAlarmHistoryMode, ""))
	if len(msgs) == 0 {
		panel.showEmptyCaseHistory(alarm)
		return
	}

	title := alarmSourceDisplayName(alarm.ObjectID) + ": №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`
		<div style="font-family: 'Segoe UI', sans-serif; font-size: 10pt; padding: 10px;">
			<h4 style="margin: 0 0 10px 0; color: %s;">%s</h4>
			<table width="100%%" cellpadding="4" cellspacing="0" style="border-collapse: collapse;">
	`, qtPrimaryColor, title))

	for _, msg := range msgs {
		textColor, rowColor := eventColorsForSC1(alarmSourceMessageSC1(msg))
		text := formatAlarmSourceMessageText(msg)
		weight := "normal"
		if msg.IsAlarm {
			weight = "bold"
		}
		sb.WriteString(fmt.Sprintf(`
			<tr style="background-color: %s; color: %s; font-weight: %s;">
				<td style="border-bottom: 1px solid #eee; padding: 4px;">%s</td>
			</tr>
		`, rowColor, textColor, weight, htmlEscape(text)))
	}

	sb.WriteString(`
			</table>
		</div>
	`)
	panel.historyBrowser.SetHtml(sb.String())
}

func (panel *AlarmPanel) showCaseHistoryGroup(alarm models.Alarm, group viewmodels.WorkAreaCaseHistoryGroup) {
	title := "CASL: №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}
	if summary := strings.TrimSpace(group.Title); summary != "" {
		title += " | " + summary
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`
		<div style="font-family: 'Segoe UI', sans-serif; font-size: 10pt; padding: 10px;">
			<h4 style="margin: 0 0 10px 0; color: %s;">%s</h4>
			<table width="100%%" cellpadding="4" cellspacing="0" style="border-collapse: collapse;">
	`, qtPrimaryColor, title))

	for _, event := range group.Events {
		textColor, rowColor := eventRowColors(event)
		text := caseHistoryEventText(event)
		weight := "normal"
		if event.IsCritical() {
			weight = "bold"
		}
		sb.WriteString(fmt.Sprintf(`
			<tr style="background-color: %s; color: %s; font-weight: %s;">
				<td style="border-bottom: 1px solid #eee; padding: 4px;">%s</td>
			</tr>
		`, colorToHTML(rowColor), colorToHTML(textColor), weight, htmlEscape(text)))
	}

	sb.WriteString(`
			</table>
		</div>
	`)
	panel.historyBrowser.SetHtml(sb.String())
}

func (panel *AlarmPanel) clearCaseHistory() {
	if panel == nil || panel.historyBrowser == nil {
		return
	}
	panel.historyBrowser.SetHtml("<div style='color: #666; font-family: Segoe UI; padding: 10px;'>Оберіть тривогу для перегляду хронології</div>")
}

func (panel *AlarmPanel) ReloadSelectedCaseHistory() {
	if panel == nil {
		return
	}
	if alarm, ok := panel.alarmsByID[panel.selectedAlarmID]; ok {
		panel.loadCaseHistoryForAlarm(alarm)
	}
}

func (panel *AlarmPanel) viewAlarmHistory(alarm models.Alarm) {
	// Optional callback or dialog for full history
	// Currently just reloads in the browser pane
	panel.loadCaseHistoryForAlarm(alarm)
}
