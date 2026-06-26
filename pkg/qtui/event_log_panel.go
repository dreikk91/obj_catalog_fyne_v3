//go:build qt

package qtui

import (
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const (
	prefQtEventLogPeriod          = "qt.eventLog.period"
	prefQtEventLogSource          = "qt.eventLog.source"
	prefQtEventLogSeverity        = "qt.eventLog.severity"
	prefQtEventLogCurrentOnly     = "qt.eventLog.currentOnly"
	eventLogSeverityAll           = "Всі події"
	eventLogSeverityCritical      = "Критичні"
	eventLogSeverityWarning       = "Попередження"
	eventLogSeverityInformational = "Інформаційні"
)

type EventLogPanel struct {
	*qt.QWidget
	table              *qt.QTableView
	model              *eventLogTableModel
	vm                 *viewmodels.EventLogViewModel
	allEvents          []models.Event
	filteredEvents     []models.Event
	currentObject      *models.Object
	rowsSignature      string
	rowsReady          bool
	showForCurrentOnly bool
	isPaused           bool
	filterUpdating     bool
	autoSized          bool
	prefs              config.Preferences

	pauseBtn       *qt.QPushButton
	rangeSelect    *qt.QComboBox
	sourceSelect   *qt.QComboBox
	severitySelect *qt.QComboBox
	contextToggle  *qt.QCheckBox

	OnEventSelected func(models.Event)
	OnCountChanged  func(count int)
}

func NewEventLogPanel(prefs config.Preferences) *EventLogPanel {
	panel := &EventLogPanel{
		QWidget: qt.NewQWidget2(),
		vm:      viewmodels.NewEventLogViewModel(),
		prefs:   prefs,
	}

	layout := qt.NewQVBoxLayout(panel.QWidget)

	toolbar := qt.NewQHBoxLayout2()
	title := qt.NewQLabel3("Журнал подій")
	title.SetStyleSheet("font-weight: 600; font-size: 11pt; padding: 4px 0;")
	toolbar.AddWidget(title.QWidget)
	toolbar.AddStretch()

	panel.contextToggle = qt.NewQCheckBox3("По вибраному")
	panel.contextToggle.SetToolTip("Показувати тільки події поточного вибраного об'єкта (Ctrl+Shift+O)")
	panel.contextToggle.OnToggled(func(checked bool) {
		panel.showForCurrentOnly = checked
		panel.saveFilterPrefs()
		panel.applyFilters()
	})

	panel.sourceSelect = qt.NewQComboBox2()
	panel.sourceSelect.AddItems(viewmodels.BuildObjectSourceOptions(0, 0, 0, 0))
	panel.sourceSelect.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.saveFilterPrefs()
		panel.applyFilters()
	})

	panel.rangeSelect = qt.NewQComboBox2()
	panel.rangeSelect.AddItems([]string{"Остання година", "Сьогодні", "Всі"})
	panel.rangeSelect.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.saveFilterPrefs()
		panel.applyFilters()
	})

	panel.severitySelect = qt.NewQComboBox2()
	panel.severitySelect.AddItems([]string{eventLogSeverityAll, eventLogSeverityCritical, eventLogSeverityWarning, eventLogSeverityInformational})
	panel.severitySelect.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.saveFilterPrefs()
		panel.applyFilters()
	})

	panel.pauseBtn = qt.NewQPushButton3("⏸ Пауза")
	panel.pauseBtn.SetToolTip("Пауза/продовження оновлення журналу (Ctrl+P)")
	panel.pauseBtn.OnClicked(func() {
		panel.TogglePause()
	})

	toolbar.AddWidget(panel.contextToggle.QWidget)
	toolbar.AddWidget(panel.sourceSelect.QWidget)
	toolbar.AddWidget(panel.rangeSelect.QWidget)
	toolbar.AddWidget(panel.severitySelect.QWidget)
	toolbar.AddWidget(panel.pauseBtn.QWidget)
	layout.AddLayout(toolbar.QLayout)

	panel.model = newEventLogTableModel(eventLogHeaders())

	panel.table = qt.NewQTableView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.HorizontalHeader().SetStretchLastSection(true)

	panel.table.SelectionModel().OnCurrentRowChanged(func(current *qt.QModelIndex, previous *qt.QModelIndex) {
		if event, ok := panel.eventAtIndex(current); ok {
			if panel.OnEventSelected != nil {
				panel.OnEventSelected(event)
			}
		}
	})

	panel.table.SetContextMenuPolicy(qt.CustomContextMenu)
	panel.table.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		panel.showContextMenu(pos)
	})

	layout.AddWidget(panel.table.QWidget)
	panel.SetLayout(layout.QLayout)
	panel.restoreFilterPrefs()
	panel.registerShortcuts()
	return panel
}

func (panel *EventLogPanel) SetEvents(events []models.Event) {
	if panel == nil || panel.model == nil {
		return
	}
	if panel.isPaused {
		return
	}
	panel.allEvents = append(panel.allEvents[:0], events...)
	panel.applyFilters()
}

func (panel *EventLogPanel) applyColumnWidths() {
	if panel.autoSized {
		return
	}
	resizeTableToContentsWithMinimums("events", panel.table)
	panel.autoSized = true
}

func (panel *EventLogPanel) applyFilters() {
	if panel == nil || panel.model == nil || panel.vm == nil {
		return
	}
	period := "Остання година"
	if panel.rangeSelect != nil {
		period = panel.rangeSelect.CurrentText()
	}
	severityFilter := eventLogSeverityAll
	if panel.severitySelect != nil {
		severityFilter = panel.severitySelect.CurrentText()
	}
	selectedSource := viewmodels.ObjectSourceAll
	if panel.sourceSelect != nil {
		selectedSource = viewmodels.NormalizeObjectSourceFilter(panel.sourceSelect.CurrentText())
	}

	var eventLogLimit = 1000
	if panel.prefs != nil {
		eventLogLimit = config.LoadUIConfig(panel.prefs).EventLogLimit
	}

	input := viewmodels.EventLogFilterInput{
		AllEvents:          panel.allEvents,
		Period:             period,
		SelectedSource:     selectedSource,
		SeverityFilter:     severityFilter,
		ShowForCurrentOnly: panel.showForCurrentOnly,
		MaxEvents:          eventLogLimit,
	}
	if panel.currentObject != nil {
		input.CurrentObjectID = panel.currentObject.ID
		input.HasCurrentObject = true
	}
	out := panel.vm.ApplyFilters(input)

	panel.filteredEvents = out.Filtered
	if panel.OnCountChanged != nil {
		panel.OnCountChanged(out.Count)
	}

	if panel.sourceSelect != nil {
		panel.filterUpdating = true
		options := viewmodels.BuildObjectSourceOptions(out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
		updateComboItems(panel.sourceSelect, options, selectedSource)
		panel.filterUpdating = false
	}

	signature := eventLogRowsSignature(out.Filtered)
	if panel.rowsReady && panel.rowsSignature == signature {
		return
	}
	panel.rowsSignature = signature
	panel.rowsReady = true

	var columnWidths []int
	if panel.autoSized {
		columnWidths = captureTableColumnWidths(panel.table)
	}
	if len(out.Filtered) == 0 {
		panel.model.setRows(nil)
		restoreTableColumnWidthsSnapshot("events", panel.table, columnWidths)
		return
	}

	panel.model.setRows(out.Filtered)
	if restoreTableColumnWidthsSnapshot("events", panel.table, columnWidths) {
		return
	}
	panel.applyColumnWidths()
}

func eventLogRowsSignature(events []models.Event) string {
	var b strings.Builder
	for _, event := range events {
		b.WriteString(eventRowSignature(event))
		b.WriteByte('|')
	}
	return b.String()
}

func (panel *EventLogPanel) eventAtIndex(index *qt.QModelIndex) (models.Event, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return models.Event{}, false
	}
	return panel.model.eventAt(index.Row())
}

func (panel *EventLogPanel) SetCurrentObject(obj *models.Object) {
	if panel == nil {
		return
	}
	panel.currentObject = obj
	panel.applyFilters()
}

func (panel *EventLogPanel) TogglePause() {
	if panel == nil || panel.pauseBtn == nil {
		return
	}
	panel.isPaused = !panel.isPaused
	if panel.isPaused {
		panel.pauseBtn.SetText("▶ Продовжити")
		return
	}
	panel.pauseBtn.SetText("⏸ Пауза")
	panel.applyFilters()
}

func (panel *EventLogPanel) ToggleCurrentOnly() {
	if panel == nil || panel.contextToggle == nil {
		return
	}
	panel.contextToggle.SetChecked(!panel.contextToggle.IsChecked())
}

func (panel *EventLogPanel) registerShortcuts() {
	if panel == nil {
		return
	}
	pauseShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+P"), panel.QObject)
	pauseShortcut.SetContext(qt.WidgetWithChildrenShortcut)
	pauseShortcut.OnActivated(func() {
		panel.TogglePause()
	})

	currentOnlyShortcut := qt.NewQShortcut2(qt.NewQKeySequence2("Ctrl+Shift+O"), panel.QObject)
	currentOnlyShortcut.SetContext(qt.WidgetWithChildrenShortcut)
	currentOnlyShortcut.OnActivated(func() {
		panel.ToggleCurrentOnly()
	})
}

func (panel *EventLogPanel) restoreFilterPrefs() {
	if panel == nil || panel.prefs == nil {
		return
	}
	panel.filterUpdating = true
	if panel.rangeSelect != nil {
		panel.rangeSelect.SetCurrentText(panel.prefs.StringWithFallback(prefQtEventLogPeriod, "Остання година"))
	}
	if panel.sourceSelect != nil {
		source := panel.prefs.StringWithFallback(prefQtEventLogSource, viewmodels.ObjectSourceAll)
		updateComboItems(panel.sourceSelect, viewmodels.BuildObjectSourceOptions(0, 0, 0, 0), source)
	}
	if panel.severitySelect != nil {
		panel.severitySelect.SetCurrentText(panel.prefs.StringWithFallback(prefQtEventLogSeverity, eventLogSeverityAll))
	}
	if panel.contextToggle != nil {
		panel.contextToggle.SetChecked(panel.prefs.BoolWithFallback(prefQtEventLogCurrentOnly, false))
		panel.showForCurrentOnly = panel.contextToggle.IsChecked()
	}
	panel.filterUpdating = false
}

func (panel *EventLogPanel) saveFilterPrefs() {
	if panel == nil || panel.prefs == nil || panel.filterUpdating {
		return
	}
	if panel.rangeSelect != nil {
		panel.prefs.SetString(prefQtEventLogPeriod, panel.rangeSelect.CurrentText())
	}
	if panel.sourceSelect != nil {
		panel.prefs.SetString(prefQtEventLogSource, viewmodels.NormalizeObjectSourceFilter(panel.sourceSelect.CurrentText()))
	}
	if panel.severitySelect != nil {
		panel.prefs.SetString(prefQtEventLogSeverity, panel.severitySelect.CurrentText())
	}
	if panel.contextToggle != nil {
		panel.prefs.SetBool(prefQtEventLogCurrentOnly, panel.contextToggle.IsChecked())
	}
}

func (panel *EventLogPanel) showContextMenu(pos *qt.QPoint) {
	if panel == nil || panel.table == nil || pos == nil {
		return
	}
	index := panel.table.IndexAt(pos)
	if !index.IsValid() {
		return
	}
	event, ok := panel.eventAtIndex(index)
	if !ok {
		return
	}
	panel.table.SelectRow(index.Row())

	menu := qt.NewQMenu(panel.table.QWidget)
	copyAction := menu.AddActionWithText("Копіювати опис")
	copyAction.OnTriggered(func() {
		clipboard := qt.QGuiApplication_Clipboard()
		if clipboard != nil {
			clipboard.SetText(strings.TrimSpace(event.Details))
		}
	})

	selectAction := menu.AddActionWithText("Перейти до об'єкта")
	selectAction.OnTriggered(func() {
		if panel.OnEventSelected != nil {
			panel.OnEventSelected(event)
		}
	})

	menu.AddSeparator()
	addTableCopyActions(menu, panel.table, index)
	menu.AddSeparator()
	addTableColumnActions(menu, "events", panel.table, panel.prefs, func() {
		panel.autoSized = true
	}, func() {
		panel.autoSized = false
	})
	menu.ExecWithPos(panel.table.MapToGlobalWithQPoint(pos))
}
