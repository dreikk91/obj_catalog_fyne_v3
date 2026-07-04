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
	"obj_catalog_fyne_v3/pkg/utils"
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
	table          *qt.QTreeView
	model          *qt.QStandardItemModel
	vm             *viewmodels.AlarmListViewModel
	caseHistoryVM  *viewmodels.WorkAreaCaseHistoryViewModel
	historyTree    *qt.QTreeView
	historyModel   *qt.QStandardItemModel
	splitter       *qt.QSplitter
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
	responseLoading bool

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
	panel.model = qt.NewQStandardItemModel2(0, 7)
	panel.model.SetHorizontalHeaderLabels(alarmGroupHeaders())
	addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", "", ""})

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
	panel.processButton.SetStyleSheet(`
		QPushButton { background: #2E7D32; color: white; border: 1px solid #256829; border-radius: 3px; font-weight: 700; }
		QPushButton:disabled { background: #D8E1E6; color: #8796A1; border-color: #C4D0D8; }
	`)
	panel.processButton.OnClicked(func() {
		panel.processSelectedAlarms()
	})
	panel.pickButton = qt.NewQPushButton3("Взяти в роботу")
	panel.pickButton.SetToolTip("Закріпити вибрані тривоги за оператором")
	panel.pickButton.SetStyleSheet(`
		QPushButton { background: #1E78B4; color: white; border: 1px solid #176496; border-radius: 3px; font-weight: 700; }
		QPushButton:disabled { background: #D8E1E6; color: #8796A1; border-color: #C4D0D8; }
	`)
	panel.pickButton.OnClicked(func() {
		panel.pickSelectedAlarms()
	})
	panel.responseButton = qt.NewQPushButton3("Реагування")
	panel.responseButton.SetToolTip("Відкрити картку реагування та керування МГР")
	panel.responseButton.OnClicked(func() {
		panel.respondToSelectedAlarm()
	})
	panel.historyButton = qt.NewQPushButton3("Хронологія ▾")
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

	panel.table = qt.NewQTreeView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetSelectionMode(qt.QAbstractItemView__ExtendedSelection)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.SetRootIsDecorated(true)
	panel.table.SetItemsExpandable(true)
	panel.table.SetAnimated(true)
	panel.table.SetExpandsOnDoubleClick(false)
	panel.table.SetIndentation(20)
	panel.table.Header().SetStretchLastSection(true)
	panel.table.SetStyleSheet(`
		QTreeView {
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

	panel.historyModel = qt.NewQStandardItemModel2(0, 4)
	panel.historyTree = qt.NewQTreeView2()
	panel.historyTree.SetModel(panel.historyModel.QAbstractItemModel)
	panel.historyTree.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.historyTree.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.historyTree.SetAlternatingRowColors(true)
	panel.historyTree.SetAnimated(true)
	panel.historyTree.SetRootIsDecorated(true)
	panel.historyTree.SetItemsExpandable(true)
	panel.historyTree.SetWordWrap(true)
	panel.historyTree.SetUniformRowHeights(false)
	panel.historyTree.Header().SetStretchLastSection(true)
	panel.historyTree.SetStyleSheet(`
		QTreeView {
			border: 1px solid #d5d5d5;
			background: #ffffff;
			alternate-background-color: #f8fafb;
			selection-background-color: ` + qtPrimaryColor + `;
			selection-color: white;
		}
		QHeaderView::section {
			background: #f2f5f7;
			border: 0;
			border-bottom: 1px solid #d5d5d5;
			padding: 5px;
			font-weight: 600;
		}
	`)
	panel.clearCaseHistory()
	panel.historyTree.SetVisible(false)

	panel.splitter = qt.NewQSplitter3(qt.Vertical)
	panel.splitter.AddWidget(topWidget)
	panel.splitter.AddWidget(panel.historyTree.QWidget)
	panel.splitter.SetSizes([]int{900, 0})

	layout.AddWidget(panel.splitter.QWidget)
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
	return []string{"Остання / час", "№", "Об'єкт / зона", "Кейс / тривога", "Оператор", "Пріоритет", "Джерело"}
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

func alarmGroupOperatorText(group alarmGroup) string {
	for _, alarm := range group.Alarms {
		if !alarm.IsInProgress {
			continue
		}
		if operator := strings.TrimSpace(alarm.InProgressBy); operator != "" {
			return operator
		}
		return "У роботі"
	}
	return "Не взята"
}

func newColoredReadOnlyAlarmRow(values []string, groupKey string, alarmID int, textColor color.NRGBA, rowColor color.NRGBA) []*qt.QStandardItem {
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
			item.SetData(qt.NewQVariant4(alarmID), int(qt.UserRole)+1)
		}
		items = append(items, item)
	}
	return items
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
	resizeTreeToContentsWithMinimums("alarms", panel.table)
	panel.table.CollapseAll()
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
	expandedGroups := panel.expandedAlarmGroupKeys()
	scrollValue := 0
	scrollWasAtBottom := false
	if panel.table != nil {
		scrollBar := panel.table.VerticalScrollBar()
		scrollValue = scrollBar.Value()
		scrollWasAtBottom = scrollValue >= scrollBar.Maximum()
	}
	if panel.autoSized {
		columnWidths = captureTreeColumnWidths(panel.table)
	}
	panel.model.Clear()
	panel.model.SetHorizontalHeaderLabels(alarmGroupHeaders())
	if len(groups) == 0 {
		addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", "", ""})
		panel.selectedAlarmID = 0
		panel.hideCaseHistory()
		restoreTreeColumnWidthsSnapshot("alarms", panel.table, columnWidths)
		restoreAlarmTreeScroll(panel.table, scrollValue, scrollWasAtBottom)
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
		parentItems := newColoredReadOnlyAlarmRow([]string{
			group.LatestTime,
			group.ObjectNumber,
			strings.TrimSpace(group.ObjectName),
			alarmGroupCaseText(group),
			alarmGroupOperatorText(group),
			priority,
			group.Source,
		}, group.Key, 0, textColor, rowColor)
		panel.model.AppendRow(parentItems)
		if !alarmGroupHasChildren(group) {
			continue
		}
		for _, alarm := range group.Alarms {
			childTextColor, childRowColor := eventRowColorsBySeverity(alarm.VisualSeverityValue(), alarm.SC1)
			parentItems[0].AppendRow(newColoredReadOnlyAlarmRow(
				alarmTreeChildValues(alarm),
				group.Key,
				alarm.ID,
				childTextColor,
				childRowColor,
			))
		}
	}
	panel.restoreExpandedAlarmGroups(expandedGroups)
	if restoreTreeColumnWidthsSnapshot("alarms", panel.table, columnWidths) {
		restoreAlarmTreeScroll(panel.table, scrollValue, scrollWasAtBottom)
		panel.updateSelectionState()
		return
	}
	panel.applyColumnWidths()
	panel.restoreExpandedAlarmGroups(expandedGroups)
	restoreAlarmTreeScroll(panel.table, scrollValue, scrollWasAtBottom)
	panel.updateSelectionState()
}

func alarmGroupHasChildren(group alarmGroup) bool {
	return len(group.Alarms) > 1
}

func alarmTreeChildValues(alarm models.Alarm) []string {
	zone := "Подія"
	if alarm.ZoneNumber > 0 {
		zone = "Зона " + strconv.Itoa(alarm.ZoneNumber)
	}
	eventText := strings.TrimSpace(alarm.GetTypeDisplay())
	if details := strings.TrimSpace(alarm.Details); details != "" {
		eventText += " — " + details
	}
	operator := "Не взята"
	if alarm.IsInProgress {
		operator = strings.TrimSpace(alarm.InProgressBy)
		if operator == "" {
			operator = "У роботі"
		}
	}
	priority := "звичайна"
	if alarm.IsCritical() {
		priority = "критична"
	}
	return []string{
		alarm.GetTimeDisplay(),
		"",
		"↳ " + zone,
		eventText,
		operator,
		priority,
		viewmodels.ObjectSourceByID(alarm.ObjectID),
	}
}

func restoreAlarmTreeScroll(table *qt.QTreeView, value int, wasAtBottom bool) {
	if table == nil {
		return
	}
	scrollBar := table.VerticalScrollBar()
	if wasAtBottom {
		scrollBar.SetValue(scrollBar.Maximum())
		return
	}
	scrollBar.SetValue(value)
}

func (panel *AlarmPanel) expandedAlarmGroupKeys() map[string]struct{} {
	result := make(map[string]struct{})
	if panel == nil || panel.table == nil || panel.model == nil {
		return result
	}
	root := qt.NewQModelIndex()
	for row := 0; row < panel.model.RowCount(root); row++ {
		index := panel.model.Index(row, 0, root)
		if index == nil || !index.IsValid() || !panel.table.IsExpanded(index) {
			continue
		}
		if key := strings.TrimSpace(index.DataWithRole(int(qt.UserRole)).ToString()); key != "" {
			result[key] = struct{}{}
		}
	}
	runtime.KeepAlive(root)
	return result
}

func (panel *AlarmPanel) restoreExpandedAlarmGroups(expanded map[string]struct{}) {
	if panel == nil || panel.table == nil || panel.model == nil || len(expanded) == 0 {
		return
	}
	root := qt.NewQModelIndex()
	for row := 0; row < panel.model.RowCount(root); row++ {
		index := panel.model.Index(row, 0, root)
		if index == nil || !index.IsValid() {
			continue
		}
		key := strings.TrimSpace(index.DataWithRole(int(qt.UserRole)).ToString())
		if _, ok := expanded[key]; ok {
			panel.table.Expand(index)
		}
	}
	runtime.KeepAlive(root)
}

func alarmGroupRowsSignature(groups []alarmGroup) string {
	var b strings.Builder
	for _, group := range groups {
		fmt.Fprintf(
			&b,
			"%s:%d:%s:%d:%d:%s:%s:%s:%s|",
			group.Key,
			group.ObjectID,
			group.LatestTime,
			group.CriticalCount,
			len(group.Alarms),
			group.ObjectNumber,
			strings.TrimSpace(group.ObjectName),
			alarmGroupCaseText(group),
			alarmGroupOperatorText(group),
		)
		for _, alarm := range group.Alarms {
			fmt.Fprintf(
				&b,
				"%d:%d:%d:%s:%s:%d:%t:%s:%t:%t;",
				alarm.ID,
				alarm.Time.UnixNano(),
				alarm.SC1,
				strings.TrimSpace(alarm.GetTypeDisplay()),
				strings.TrimSpace(alarm.Details),
				alarm.ZoneNumber,
				alarm.IsInProgress,
				strings.TrimSpace(alarm.InProgressBy),
				alarm.CanTakeOver,
				alarm.CanProcess,
			)
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
		panel.pickButton.SetText(alarmActionText(alarmPickActionVerb(alarms), alarms))
	}
	if panel.responseButton != nil {
		panel.responseButton.SetEnabled(selectedGroups == 1 && !panel.responseLoading)
		if panel.responseLoading {
			panel.responseButton.SetText("Завантаження...")
		} else {
			panel.responseButton.SetText("Реагування")
		}
	}
	if panel.historyButton != nil {
		panel.historyButton.SetEnabled(selectedGroups == 1)
	}
}

func (panel *AlarmPanel) SetResponseLoading(_ int, loading bool) {
	if panel == nil {
		return
	}
	panel.responseLoading = loading
	panel.updateSelectionState()
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
	seen := make(map[int]struct{}, len(selection))
	for _, index := range selection {
		if index.Column() != 0 {
			continue
		}
		rowIndex := panel.alarmTreeFirstColumnIndex(&index)
		if rowIndex == nil || !rowIndex.IsValid() {
			continue
		}
		alarmID := panel.model.Data(rowIndex, int(qt.UserRole)+1).ToInt()
		if alarmID > 0 {
			if alarm, ok := panel.alarmsByID[alarmID]; ok {
				if _, exists := seen[alarm.ID]; !exists {
					alarms = append(alarms, alarm)
					seen[alarm.ID] = struct{}{}
				}
			}
			continue
		}
		groupKey := panel.model.Data(rowIndex, int(qt.UserRole)).ToString()
		if group, ok := panel.groupsByKey[groupKey]; ok {
			for _, alarm := range group.Alarms {
				if _, exists := seen[alarm.ID]; exists {
					continue
				}
				alarms = append(alarms, alarm)
				seen[alarm.ID] = struct{}{}
			}
		}
	}
	return alarms
}

func (panel *AlarmPanel) selectedGroupAndAlarmCounts() (int, int) {
	if panel == nil || panel.table == nil || panel.model == nil {
		return 0, 0
	}
	selection := panel.table.SelectionModel().SelectedRows()
	groupKeys := make(map[string]struct{})
	alarmIDs := make(map[int]struct{})
	for _, index := range selection {
		if index.Column() != 0 {
			continue
		}
		rowIndex := panel.alarmTreeFirstColumnIndex(&index)
		if rowIndex == nil || !rowIndex.IsValid() {
			continue
		}
		groupKey := panel.model.Data(rowIndex, int(qt.UserRole)).ToString()
		if group, ok := panel.groupsByKey[groupKey]; ok {
			groupKeys[groupKey] = struct{}{}
			alarmID := panel.model.Data(rowIndex, int(qt.UserRole)+1).ToInt()
			if alarmID > 0 {
				alarmIDs[alarmID] = struct{}{}
				continue
			}
			for _, alarm := range group.Alarms {
				alarmIDs[alarm.ID] = struct{}{}
			}
		}
	}
	return len(groupKeys), len(alarmIDs)
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
	panel.pickAlarmsWithConfirmation(alarms)
}

func (panel *AlarmPanel) pickAlarmsWithConfirmation(alarms []models.Alarm) {
	if panel == nil || panel.OnPickAlarms == nil || len(alarms) == 0 {
		return
	}
	if alarmsRequireTakeover(alarms) {
		answer := qt.QMessageBox_Question(
			panel.QWidget,
			"Перехоплення тривоги",
			"Тривогу вже обробляє інший оператор. Перехопити її?",
		)
		if answer != qt.QMessageBox__Yes {
			return
		}
	}
	panel.OnPickAlarms(alarms)
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
	clickedAlarms := group.Alarms
	if alarm, exact := panel.exactAlarmAtIndex(index); exact {
		clickedAlarms = []models.Alarm{alarm}
	}
	alarms := contextMenuAlarms(clickedAlarms, panel.selectedAlarms())
	if !selectionContainsAnyAlarm(panel.selectedAlarms(), clickedAlarms) {
		panel.table.ClearSelection()
		panel.table.SetCurrentIndex(index)
		panel.table.SelectionModel().Select(
			index,
			qt.QItemSelectionModel__ClearAndSelect|qt.QItemSelectionModel__Rows,
		)
	}

	menu := qt.NewQMenu(panel.table.QWidget)
	processAction := menu.AddActionWithText(alarmActionText("Відпрацювати", alarms))
	processAction.SetEnabled(canProcessAlarms(alarms))
	processAction.OnTriggered(func() {
		if panel.OnProcessAlarms != nil {
			panel.OnProcessAlarms(alarms)
		}
	})

	pickAction := menu.AddActionWithText(alarmActionText(alarmPickActionVerb(alarms), alarms))
	pickAction.SetEnabled(canTakeAlarms(alarms))
	pickAction.OnTriggered(func() {
		panel.pickAlarmsWithConfirmation(alarms)
	})

	if panel.OnRespondAlarm != nil {
		responseAction := menu.AddActionWithText("Відкрити картку реагування")
		responseAction.SetEnabled(len(group.Alarms) > 0)
		targetAlarm := group.Primary
		if alarm, exact := panel.exactAlarmAtIndex(index); exact {
			targetAlarm = alarm
		}
		responseAction.OnTriggered(func() {
			panel.OnRespondAlarm(targetAlarm)
		})
	}

	menu.AddSeparator()
	historyAction := menu.AddActionWithText("Переглянути хронологію групи")
	historyAction.OnTriggered(func() {
		panel.showGroupHistory(group)
	})

	menu.AddSeparator()
	addTreeCopyActions(menu, panel.table, index)
	menu.AddSeparator()
	addTreeColumnActions(menu, "alarms", panel.table, panel.prefs, func() {
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

func alarmsRequireTakeover(alarms []models.Alarm) bool {
	for _, alarm := range alarms {
		if alarm.IsInProgress && !alarm.IsOwnedByMe && alarm.CanTakeOver {
			return true
		}
	}
	return false
}

func alarmPickActionVerb(alarms []models.Alarm) string {
	if alarmsRequireTakeover(alarms) {
		return "Перехопити"
	}
	return "Взяти в роботу"
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
	if alarm, ok := panel.exactAlarmAtIndex(index); ok {
		return alarm, true
	}
	group, ok := panel.groupAtIndex(index)
	if !ok {
		return models.Alarm{}, false
	}
	return group.Primary, true
}

func (panel *AlarmPanel) exactAlarmAtIndex(index *qt.QModelIndex) (models.Alarm, bool) {
	rowIndex := panel.alarmTreeFirstColumnIndex(index)
	if rowIndex == nil || !rowIndex.IsValid() {
		return models.Alarm{}, false
	}
	alarmID := panel.model.Data(rowIndex, int(qt.UserRole)+1).ToInt()
	alarm, ok := panel.alarmsByID[alarmID]
	return alarm, ok && alarmID > 0
}

func (panel *AlarmPanel) groupAtIndex(index *qt.QModelIndex) (alarmGroup, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return alarmGroup{}, false
	}
	rowIndex := panel.alarmTreeFirstColumnIndex(index)
	if rowIndex == nil || !rowIndex.IsValid() {
		return alarmGroup{}, false
	}
	groupKey := panel.model.Data(rowIndex, int(qt.UserRole)).ToString()
	group, ok := panel.groupsByKey[groupKey]
	runtime.KeepAlive(rowIndex)
	return group, ok
}

func (panel *AlarmPanel) alarmTreeFirstColumnIndex(index *qt.QModelIndex) *qt.QModelIndex {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return nil
	}
	parent := index.Parent()
	rowIndex := panel.model.Index(index.Row(), 0, parent)
	runtime.KeepAlive(parent)
	return rowIndex
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
	if panel.historyTree != nil {
		panel.historyTree.SetVisible(true)
	}
	panel.ensureCaseHistoryVisible()
	panel.viewAlarmHistory(group.Primary)
}

func (panel *AlarmPanel) ensureCaseHistoryVisible() {
	if panel == nil || panel.splitter == nil {
		return
	}
	panel.splitter.SetSizes(caseHistorySplitterSizes(panel.splitter.Sizes()))
}

func caseHistorySplitterSizes(sizes []int) []int {
	total := 0
	for _, size := range sizes {
		total += size
	}
	if total <= 0 {
		total = 900
	}
	historySize := total / 3
	if historySize < 220 {
		historySize = 220
	}
	if historySize >= total {
		historySize = total / 2
	}
	return []int{total - historySize, historySize}
}

func (panel *AlarmPanel) hideCaseHistory() {
	if panel == nil || panel.historyTree == nil {
		return
	}
	panel.clearCaseHistory()
	panel.historyTree.SetVisible(false)
	if panel.splitter != nil {
		panel.splitter.SetSizes([]int{900, 0})
	}
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
	panel.setCaseHistoryTreeStatus(title, "Завантаження хронології...")
}

func (panel *AlarmPanel) showEmptyCaseHistory(alarm models.Alarm) {
	title := alarmSourceDisplayName(alarm.ObjectID) + ": №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}
	panel.setCaseHistoryTreeStatus(title, "Подій за період тривоги не знайдено")
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

	rows := make([]caseHistoryTreeRow, 0, len(msgs))
	for _, msg := range msgs {
		textColor, rowColor := utils.SelectColorNRGBA(alarmSourceMessageSC1(msg))
		rows = append(rows, alarmMessageHistoryTreeRow(msg, colorToQtName(textColor), colorToQtName(rowColor)))
	}
	panel.setCaseHistoryTreeRows(title, rows)
}

func (panel *AlarmPanel) showCaseHistoryGroup(alarm models.Alarm, group viewmodels.WorkAreaCaseHistoryGroup) {
	title := "CASL: №" + alarm.GetObjectNumberDisplay()
	if name := strings.TrimSpace(alarm.ObjectName); name != "" {
		title += " " + name
	}
	if summary := strings.TrimSpace(group.Title); summary != "" {
		title += " | " + summary
	}

	rows := make([]caseHistoryTreeRow, 0, len(group.Events))
	for _, event := range group.Events {
		textColor, rowColor := eventRowColors(event)
		rows = append(rows, eventHistoryTreeRow(event, colorToQtName(textColor), colorToQtName(rowColor)))
	}
	panel.setCaseHistoryTreeRows(title, rows)
}

func (panel *AlarmPanel) clearCaseHistory() {
	if panel == nil || panel.historyTree == nil || panel.historyModel == nil {
		return
	}
	panel.setCaseHistoryTreeStatus("Хронологія кейсу", "Оберіть тривогу для перегляду")
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

func (panel *AlarmPanel) SetTableFontSize(size float32) {
	if panel == nil || panel.table == nil {
		return
	}
	font := panel.table.Font()
	font.SetPointSizeF(float64(size))
	panel.table.SetFont(font)
}
