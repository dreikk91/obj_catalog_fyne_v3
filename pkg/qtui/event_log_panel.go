//go:build qt

package qtui

import (
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type EventLogPanel struct {
	*qt.QWidget
	table              *qt.QTableView
	model              *qt.QStandardItemModel
	vm                 *viewmodels.EventLogViewModel
	allEvents          []models.Event
	filteredEvents     []models.Event
	eventsByID         map[int]models.Event
	currentObject      *models.Object
	showForCurrentOnly bool
	isPaused           bool
	filterUpdating     bool
	autoSized          bool
	prefs              config.Preferences

	pauseBtn      *qt.QPushButton
	rangeSelect   *qt.QComboBox
	sourceSelect  *qt.QComboBox
	importantOnly *qt.QCheckBox
	contextToggle *qt.QCheckBox

	OnEventSelected func(models.Event)
}

func NewEventLogPanel(prefs config.Preferences) *EventLogPanel {
	panel := &EventLogPanel{
		QWidget:    qt.NewQWidget2(),
		vm:         viewmodels.NewEventLogViewModel(),
		eventsByID: make(map[int]models.Event),
		prefs:      prefs,
	}

	layout := qt.NewQVBoxLayout(panel.QWidget)

	toolbar := qt.NewQHBoxLayout2()
	title := qt.NewQLabel3("Журнал подій")
	title.SetStyleSheet("font-weight: 600; font-size: 11pt; padding: 4px 0;")
	toolbar.AddWidget(title.QWidget)
	toolbar.AddStretch()

	panel.contextToggle = qt.NewQCheckBox3("По вибраному")
	panel.contextToggle.OnToggled(func(checked bool) {
		panel.showForCurrentOnly = checked
		panel.applyFilters()
	})

	panel.sourceSelect = qt.NewQComboBox2()
	panel.sourceSelect.AddItems(viewmodels.BuildObjectSourceOptions(0, 0, 0, 0))
	panel.sourceSelect.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.applyFilters()
	})

	panel.rangeSelect = qt.NewQComboBox2()
	panel.rangeSelect.AddItems([]string{"Остання година", "Сьогодні", "Всі"})
	panel.rangeSelect.OnCurrentTextChanged(func(string) {
		if panel.filterUpdating {
			return
		}
		panel.applyFilters()
	})

	panel.importantOnly = qt.NewQCheckBox3("Важливі")
	panel.importantOnly.OnToggled(func(checked bool) {
		panel.applyFilters()
	})

	panel.pauseBtn = qt.NewQPushButton3("⏸ Пауза")
	panel.pauseBtn.OnClicked(func() {
		panel.isPaused = !panel.isPaused
		if panel.isPaused {
			panel.pauseBtn.SetText("▶ Продовжити")
		} else {
			panel.pauseBtn.SetText("⏸ Пауза")
			panel.applyFilters()
		}
	})

	toolbar.AddWidget(panel.contextToggle.QWidget)
	toolbar.AddWidget(panel.sourceSelect.QWidget)
	toolbar.AddWidget(panel.rangeSelect.QWidget)
	toolbar.AddWidget(panel.importantOnly.QWidget)
	toolbar.AddWidget(panel.pauseBtn.QWidget)
	layout.AddLayout(toolbar.QLayout)

	panel.model = qt.NewQStandardItemModel2(0, 6)
	panel.model.SetHorizontalHeaderLabels([]string{"Час", "№", "Подія", "Об'єкт", "Опис", "Джерело"})

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

func (panel *EventLogPanel) applyFilters() {
	if panel == nil || panel.model == nil || panel.vm == nil {
		return
	}
	period := "Остання година"
	if panel.rangeSelect != nil {
		period = panel.rangeSelect.CurrentText()
	}
	importantOnly := false
	if panel.importantOnly != nil {
		importantOnly = panel.importantOnly.IsChecked()
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
		ImportantOnly:      importantOnly,
		ShowForCurrentOnly: panel.showForCurrentOnly,
		MaxEvents:          eventLogLimit,
	}
	if panel.currentObject != nil {
		input.CurrentObjectID = panel.currentObject.ID
		input.HasCurrentObject = true
	}
	out := panel.vm.ApplyFilters(input)

	panel.filteredEvents = out.Filtered

	if panel.sourceSelect != nil {
		panel.filterUpdating = true
		options := viewmodels.BuildObjectSourceOptions(out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL)
		updateComboItems(panel.sourceSelect, options, selectedSource)
		panel.filterUpdating = false
	}

	panel.model.Clear()
	panel.model.SetHorizontalHeaderLabels([]string{"Час", "№", "Подія", "Об'єкт", "Опис", "Джерело"})
	if len(out.Filtered) == 0 {
		addReadOnlyRow(panel.model, []string{"--:--", "-", "-", "Немає подій", "", ""})
		return
	}

	panel.eventsByID = make(map[int]models.Event, len(out.Filtered))
	for _, event := range out.Filtered {
		panel.eventsByID[event.ID] = event
		textColor, rowColor := eventRowColorsBySeverity(event.VisualSeverityValue(), event.SC1)
		addColoredReadOnlyRow(panel.model, []string{
			event.GetDateTimeDisplay(),
			eventObjectNumber(event),
			event.GetTypeDisplay(),
			strings.TrimSpace(event.ObjectName),
			strings.TrimSpace(event.Details),
			viewmodels.ObjectSourceByID(event.ObjectID),
		}, event.ID, textColor, rowColor)
	}
	if !panel.autoSized {
		resizeTableToContents(panel.table)
		panel.autoSized = true
	}
}

func (panel *EventLogPanel) eventAtIndex(index *qt.QModelIndex) (models.Event, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return models.Event{}, false
	}
	rowIndex := panel.model.Index(index.Row(), 0, nil)
	if rowIndex == nil || !rowIndex.IsValid() {
		return models.Event{}, false
	}
	eventID := panel.model.Data(rowIndex, int(qt.UserRole)).ToInt()
	event, ok := panel.eventsByID[eventID]
	return event, ok
}

func (panel *EventLogPanel) SetCurrentObject(obj *models.Object) {
	if panel == nil {
		return
	}
	panel.currentObject = obj
	panel.applyFilters()
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

	menu.ExecWithPos(panel.table.MapToGlobalWithQPoint(pos))
}
