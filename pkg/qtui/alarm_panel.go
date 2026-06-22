//go:build qt

package qtui

import (
	"fmt"
	"image/color"
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
	selectedAlarmID int

	OnAlarmSelected func(models.Alarm)
	OnProcessAlarms func([]models.Alarm)
	OnPickAlarms    func([]models.Alarm)
}

func NewAlarmPanel(prefs config.Preferences) *AlarmPanel {
	panel := &AlarmPanel{
		QWidget:       qt.NewQWidget2(),
		vm:            viewmodels.NewAlarmListViewModel(),
		caseHistoryVM: viewmodels.NewWorkAreaCaseHistoryViewModel(),
		alarmsByID:    map[int]models.Alarm{},
		prefs:         prefs,
	}
	layout := qt.NewQVBoxLayout(panel.QWidget)
	panel.model = qt.NewQStandardItemModel2(0, 6)
	panel.model.SetHorizontalHeaderLabels([]string{"Час", "№", "Об'єкт", "Подія", "Пріоритет", "Джерело"})
	addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", ""})

	toolbar := qt.NewQHBoxLayout2()
	processButton := qt.NewQPushButton3("Відпрацювати")
	processButton.OnClicked(func() {
		panel.processSelectedAlarms()
	})
	pickButton := qt.NewQPushButton3("Взяти в роботу")
	pickButton.OnClicked(func() {
		panel.pickSelectedAlarms()
	})
	toolbar.AddWidget(processButton.QWidget)
	toolbar.AddWidget(pickButton.QWidget)
	toolbar.AddStretch()

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
	toolbar.AddWidget(panel.sourceFilter.QWidget)
	toolbar.AddWidget(panel.severityFilter.QWidget)

	panel.table = qt.NewQTableView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetSelectionMode(qt.QAbstractItemView__ExtendedSelection)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.HorizontalHeader().SetStretchLastSection(true)
	panel.table.OnDoubleClicked(func(index *qt.QModelIndex) {
		if alarm, ok := panel.alarmAtIndex(index); ok && panel.OnProcessAlarms != nil {
			panel.OnProcessAlarms([]models.Alarm{alarm})
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
			panel.loadCaseHistoryForAlarm(alarm)
			return
		}
		panel.selectedAlarmID = 0
		panel.clearCaseHistory()
	})

	topWidget := qt.NewQWidget2()
	topLayout := qt.NewQVBoxLayout(topWidget)
	topLayout.AddLayout(toolbar.QLayout)
	topLayout.AddWidget(panel.table.QWidget)
	topWidget.SetLayout(topLayout.QLayout)

	panel.historyBrowser = qt.NewQTextBrowser(nil)
	panel.clearCaseHistory()

	splitter := qt.NewQSplitter3(qt.Vertical)
	splitter.AddWidget(topWidget)
	splitter.AddWidget(panel.historyBrowser.QWidget)
	splitter.SetSizes([]int{500, 200})

	layout.AddWidget(splitter.QWidget)
	panel.SetLayout(layout.QLayout)
	return panel
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
	panel.alarmsByID = make(map[int]models.Alarm, len(filtered))
	panel.model.Clear()
	panel.model.SetHorizontalHeaderLabels([]string{"Час", "№", "Об'єкт", "Подія", "Пріоритет", "Джерело"})
	if len(filtered) == 0 {
		addReadOnlyRow(panel.model, []string{"--:--", "-", "Немає активних тривог", "", "", ""})
		return
	}
	for _, alarm := range filtered {
		panel.alarmsByID[alarm.ID] = alarm
		priority := "звичайна"
		if alarm.IsCritical() {
			priority = "критична"
		}
		rowColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
		textColor := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
		if alarm.IsCritical() {
			rowColor = color.NRGBA{R: 255, G: 220, B: 220, A: 255}
		}
		addColoredReadOnlyRow(panel.model, []string{
			alarm.GetTimeDisplay(),
			alarm.GetObjectNumberDisplay(),
			strings.TrimSpace(alarm.ObjectName),
			alarm.GetTypeDisplay(),
			priority,
			viewmodels.ObjectSourceByID(alarm.ObjectID),
		}, alarm.ID, textColor, rowColor)
	}
	if !panel.autoSized {
		resizeTableToContents(panel.table)
		panel.autoSized = true
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
		eventID := panel.model.Data(&index, int(qt.UserRole)).ToInt()
		if alarm, ok := panel.alarmsByID[eventID]; ok {
			alarms = append(alarms, alarm)
		}
	}
	return alarms
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

func (panel *AlarmPanel) showContextMenu(pos *qt.QPoint) {
	if panel == nil || panel.table == nil || pos == nil {
		return
	}
	index := panel.table.IndexAt(pos)
	if !index.IsValid() {
		return
	}
	alarm, ok := panel.alarmAtIndex(index)
	if !ok {
		return
	}

	menu := qt.NewQMenu(panel.table.QWidget)
	processAction := menu.AddActionWithText("Відпрацювати")
	processAction.OnTriggered(func() {
		if panel.OnProcessAlarms != nil {
			panel.OnProcessAlarms([]models.Alarm{alarm})
		}
	})

	pickAction := menu.AddActionWithText("Взяти в роботу")
	pickAction.OnTriggered(func() {
		if panel.OnPickAlarms != nil {
			panel.OnPickAlarms([]models.Alarm{alarm})
		}
	})

	menu.AddSeparator()
	historyAction := menu.AddActionWithText("Переглянути історію")
	historyAction.OnTriggered(func() {
		panel.viewAlarmHistory(alarm)
	})

	menu.ExecWithPos(panel.table.MapToGlobalWithQPoint(pos))
}

func (panel *AlarmPanel) alarmAtIndex(index *qt.QModelIndex) (models.Alarm, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return models.Alarm{}, false
	}
	rowIndex := panel.model.Index(index.Row(), 0, nil)
	if rowIndex == nil || !rowIndex.IsValid() {
		return models.Alarm{}, false
	}
	alarmID := panel.model.Data(rowIndex, int(qt.UserRole)).ToInt()
	alarm, ok := panel.alarmsByID[alarmID]
	return alarm, ok
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
	panel.historyBrowser.SetHtml(fmt.Sprintf("<div style='color: #1a73e8; font-family: Segoe UI; padding: 10px;'><b>%s</b><br/>Завантаження хронології...</div>", title))
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
			<h4 style="margin: 0 0 10px 0; color: #1a73e8;">%s</h4>
			<table width="100%%" cellpadding="4" cellspacing="0" style="border-collapse: collapse;">
	`, title))

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
			<h4 style="margin: 0 0 10px 0; color: #1a73e8;">%s</h4>
			<table width="100%%" cellpadding="4" cellspacing="0" style="border-collapse: collapse;">
	`, title))

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
