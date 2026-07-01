//go:build qt

package qtui

import (
	"fmt"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

const (
	prefQtObjectListQuery  = "qt.objectList.query"
	prefQtObjectListStatus = "qt.objectList.status"
	prefQtObjectListSource = "qt.objectList.source"
)

type ObjectListPanel struct {
	*qt.QWidget
	search            *qt.QLineEdit
	statusFilter      *qt.QComboBox
	sourceFilter      *qt.QComboBox
	searchTimer       *qt.QTimer
	table             *qt.QTableView
	model             *objectListTableModel
	vm                *viewmodels.ObjectListViewModel
	prefs             config.Preferences
	allObjects        []models.Object
	rowsSignature     string
	rowsReady         bool
	autoSized         bool
	suppressSelection bool
	filterUpdating    bool
	OnObjectSelected  func(models.Object)
	OnBridgeMode      func(models.Object, contracts.DisplayBlockMode)
	OnCASLBlock       func(models.Object)
}

func NewObjectListPanel(prefs config.Preferences) *ObjectListPanel {
	panel := &ObjectListPanel{
		QWidget: qt.NewQWidget2(),
		vm:      viewmodels.NewObjectListViewModel(),
		prefs:   prefs,
	}
	panel.SetMinimumWidth(200)

	layout := qt.NewQVBoxLayout(panel.QWidget)
	title := qt.NewQLabel3("Список об'єктів")
	title.SetStyleSheet("font-weight: 600; font-size: 11pt; padding: 4px 0;")

	panel.search = qt.NewQLineEdit2()
	panel.search.SetPlaceholderText("Номер, назва, адреса, SIM або договір")
	panel.search.SetClearButtonEnabled(true)

	filtersLayout := qt.NewQHBoxLayout2()
	panel.statusFilter = qt.NewQComboBox2()
	panel.statusFilter.AddItems(panel.vm.BuildFilterOptions(0, 0, 0, 0, 0))
	panel.sourceFilter = qt.NewQComboBox2()
	panel.sourceFilter.AddItems(viewmodels.BuildObjectSourceOptions(0, 0, 0, 0))
	filtersLayout.AddWidget(panel.statusFilter.QWidget)
	filtersLayout.AddWidget(panel.sourceFilter.QWidget)

	panel.model = newObjectListTableModel(panel.vm)

	panel.table = qt.NewQTableView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetWordWrap(false)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.HorizontalHeader().SetStretchLastSection(true)
	panel.table.HorizontalHeader().SetMinimumSectionSize(20)
	for i, w := range []int{50, 250, 230} {
		panel.table.SetColumnWidth(i, w)
	}
	panel.table.SelectionModel().OnCurrentRowChanged(func(current *qt.QModelIndex, previous *qt.QModelIndex) {
		panel.notifyObjectSelection(current)
	})
	panel.table.SetContextMenuPolicy(qt.CustomContextMenu)
	panel.table.OnCustomContextMenuRequested(func(pos *qt.QPoint) {
		panel.showContextMenu(pos)
	})
	panel.table.OnDoubleClicked(func(index *qt.QModelIndex) {
		panel.notifyObjectSelection(index)
	})

	panel.searchTimer = qt.NewQTimer()
	panel.searchTimer.SetSingleShot(true)
	panel.searchTimer.SetInterval(200)
	panel.searchTimer.OnTimeout(func() {
		panel.applyFilters()
	})
	panel.search.OnTextChanged(func(string) {
		panel.saveFilterPrefs()
		panel.scheduleSearchFilter()
	})
	panel.statusFilter.OnCurrentTextChanged(func(string) {
		panel.saveFilterPrefs()
		panel.applyFilters()
	})
	panel.sourceFilter.OnCurrentTextChanged(func(string) {
		panel.saveFilterPrefs()
		panel.applyFilters()
	})

	layout.AddWidget(title.QWidget)
	layout.AddWidget(panel.search.QWidget)
	layout.AddLayout(filtersLayout.QLayout)
	layout.AddWidget(panel.table.QWidget)
	panel.SetLayout(layout.QLayout)
	panel.restoreFilterPrefs()
	panel.applyFilters()

	return panel
}

func (panel *ObjectListPanel) SetObjects(objects []models.Object) {
	if panel == nil {
		return
	}
	panel.allObjects = append(panel.allObjects[:0], objects...)
	panel.applyFilters()
}

func (panel *ObjectListPanel) FocusSearch() {
	if panel == nil || panel.search == nil {
		return
	}
	panel.search.SetFocus()
	panel.search.SelectAll()
}

func (panel *ObjectListPanel) scheduleSearchFilter() {
	if panel == nil {
		return
	}
	if panel.searchTimer == nil {
		panel.applyFilters()
		return
	}
	panel.searchTimer.Start2()
}

func (panel *ObjectListPanel) restoreFilterPrefs() {
	if panel == nil || panel.prefs == nil {
		return
	}
	panel.filterUpdating = true
	defer func() {
		panel.filterUpdating = false
	}()

	if panel.search != nil {
		panel.search.SetText(panel.prefs.StringWithFallback(prefQtObjectListQuery, ""))
	}
	if panel.statusFilter != nil {
		status := panel.prefs.StringWithFallback(prefQtObjectListStatus, viewmodels.FilterAll)
		panel.statusFilter.SetCurrentIndex(indexForNormalizedStatusFilter(panel.statusFilter, status))
	}
	if panel.sourceFilter != nil {
		source := panel.prefs.StringWithFallback(prefQtObjectListSource, viewmodels.ObjectSourceAll)
		panel.sourceFilter.SetCurrentIndex(indexForNormalizedSourceFilter(panel.sourceFilter, source))
	}
}

func (panel *ObjectListPanel) saveFilterPrefs() {
	if panel == nil || panel.prefs == nil || panel.filterUpdating {
		return
	}
	if panel.search != nil {
		panel.prefs.SetString(prefQtObjectListQuery, strings.TrimSpace(panel.search.Text()))
	}
	if panel.statusFilter != nil {
		panel.prefs.SetString(prefQtObjectListStatus, viewmodels.NormalizeObjectListFilter(panel.statusFilter.CurrentText()))
	}
	if panel.sourceFilter != nil {
		panel.prefs.SetString(prefQtObjectListSource, viewmodels.NormalizeObjectSourceFilter(panel.sourceFilter.CurrentText()))
	}
}

func (panel *ObjectListPanel) notifyObjectSelection(index *qt.QModelIndex) {
	if panel == nil || panel.suppressSelection || panel.OnObjectSelected == nil || index == nil || !index.IsValid() {
		return
	}
	if object, ok := panel.objectAtIndex(index); ok {
		panel.OnObjectSelected(object)
	}
}

func (panel *ObjectListPanel) objectAtIndex(index *qt.QModelIndex) (models.Object, bool) {
	if panel == nil || panel.model == nil || index == nil || !index.IsValid() {
		return models.Object{}, false
	}
	return panel.model.objectAt(index.Row())
}

func (panel *ObjectListPanel) applyColumnWidths() {
	if panel.autoSized {
		return
	}
	resizeObjectListColumns(panel.table)
	panel.autoSized = true
}

func (panel *ObjectListPanel) applyFilters() {
	if panel == nil || panel.model == nil || panel.vm == nil {
		return
	}
	currentFilter := ""
	if panel.statusFilter != nil {
		currentFilter = panel.statusFilter.CurrentText()
	}
	currentSource := ""
	if panel.sourceFilter != nil {
		currentSource = panel.sourceFilter.CurrentText()
	}
	query := ""
	if panel.search != nil {
		query = panel.search.Text()
	}

	out := panel.vm.ApplyFilters(viewmodels.ObjectListFilterInput{
		AllObjects:    panel.allObjects,
		Query:         query,
		CurrentFilter: currentFilter,
		CurrentSource: currentSource,
	})

	panel.refreshFilterOptions(out, currentFilter, currentSource)
	panel.setFilteredObjects(out.Filtered)
}

func (panel *ObjectListPanel) refreshFilterOptions(out viewmodels.ObjectListFilterOutput, currentFilter string, currentSource string) {
	if panel.statusFilter != nil {
		normalized := viewmodels.NormalizeObjectListFilter(currentFilter)
		wasBlocked := panel.statusFilter.BlockSignals(true)
		panel.statusFilter.Clear()
		panel.statusFilter.AddItems(panel.vm.BuildFilterOptions(out.CountAll, out.CountAlarm, out.CountOffline, out.CountMonitoringOff, out.CountDebug))
		panel.statusFilter.SetCurrentIndex(indexForNormalizedStatusFilter(panel.statusFilter, normalized))
		panel.statusFilter.BlockSignals(wasBlocked)
	}
	if panel.sourceFilter != nil {
		normalized := viewmodels.NormalizeObjectSourceFilter(currentSource)
		wasBlocked := panel.sourceFilter.BlockSignals(true)
		panel.sourceFilter.Clear()
		panel.sourceFilter.AddItems(viewmodels.BuildObjectSourceOptions(out.CountAll, out.CountBridge, out.CountPhoenix, out.CountCASL))
		panel.sourceFilter.SetCurrentIndex(indexForNormalizedSourceFilter(panel.sourceFilter, normalized))
		panel.sourceFilter.BlockSignals(wasBlocked)
	}
}

func (panel *ObjectListPanel) setFilteredObjects(objects []models.Object) {
	signature := objectRowsSignature(objects)
	if panel.rowsReady && panel.rowsSignature == signature {
		return
	}
	panel.rowsSignature = signature
	panel.rowsReady = true

	var columnWidths []int
	if panel.autoSized {
		columnWidths = captureTableColumnWidths(panel.table)
	}
	panel.model.setRows(objects)

	if restoreTableColumnWidthsSnapshot("objects", panel.table, columnWidths) {
		return
	}
	if len(objects) > 0 {
		panel.applyColumnWidths()
	}
}

func objectRowsSignature(objects []models.Object) string {
	var b strings.Builder
	for _, object := range objects {
		fmt.Fprintf(
			&b,
			"%d:%s:%s:%s:%d:%s:%s:%s|",
			object.ID,
			viewmodels.ObjectDisplayNumber(object),
			strings.TrimSpace(object.Name),
			strings.TrimSpace(object.Address),
			object.Status,
			strings.TrimSpace(object.StatusText),
			object.MonitoringStatusValue(),
			viewmodels.ObjectSourceByID(object.ID),
		)
	}
	return b.String()
}

func (panel *ObjectListPanel) SelectObject(id int) {
	if panel == nil || panel.model == nil || panel.table == nil {
		return
	}
	selectionModel := panel.table.SelectionModel()
	if selectionModel != nil {
		currentIndex := selectionModel.CurrentIndex()
		if currentIndex != nil && currentIndex.IsValid() {
			currentID := indexToID(currentIndex)
			if currentID == id {
				return
			}
		}
	}

	row := panel.model.rowForID(id)
	if row < 0 {
		return
	}

	panel.suppressSelection = true
	panel.table.SelectRow(row)
	panel.suppressSelection = false
	parent := qt.NewQModelIndex()
	if index := panel.model.Index(row, 0, parent); index != nil && index.IsValid() {
		panel.table.ScrollTo(index, qt.QAbstractItemView__PositionAtCenter)
	}
}

func (panel *ObjectListPanel) showContextMenu(pos *qt.QPoint) {
	if panel == nil || panel.table == nil || pos == nil {
		return
	}
	index := panel.table.IndexAt(pos)
	if !index.IsValid() {
		return
	}
	object, ok := panel.objectAtIndex(index)
	if !ok {
		return
	}
	panel.table.SelectRow(index.Row())

	menu := qt.NewQMenu(panel.table.QWidget)
	openAction := menu.AddActionWithText("Відкрити картку")
	openAction.OnTriggered(func() {
		if panel.OnObjectSelected != nil {
			panel.OnObjectSelected(object)
		}
	})

	menu.AddSeparator()
	if panel.addMonitoringActions(menu, object) {
		menu.AddSeparator()
	}
	copyNumberAction := menu.AddActionWithText("Копіювати номер")
	copyNumberAction.OnTriggered(func() {
		setClipboardText(viewmodels.ObjectDisplayNumber(object))
	})

	copyNameAction := menu.AddActionWithText("Копіювати назву")
	copyNameAction.OnTriggered(func() {
		setClipboardText(strings.TrimSpace(object.Name))
	})

	copyAddressAction := menu.AddActionWithText("Копіювати адресу")
	copyAddressAction.OnTriggered(func() {
		setClipboardText(strings.TrimSpace(object.Address))
	})

	copySummaryAction := menu.AddActionWithText("Копіювати картку рядка")
	copySummaryAction.OnTriggered(func() {
		setClipboardText(objectListClipboardText(object))
	})

	menu.AddSeparator()
	addTableCopyActions(menu, panel.table, index)
	menu.AddSeparator()
	addTableColumnActions(menu, "objects", panel.table, panel.prefs, func() {
		panel.autoSized = true
	}, func() {
		panel.autoSized = false
	})
	menu.ExecWithPos(panel.table.MapToGlobalWithQPoint(pos))
}

func (panel *ObjectListPanel) addMonitoringActions(menu *qt.QMenu, object models.Object) bool {
	switch viewmodels.ObjectSourceByID(object.ID) {
	case viewmodels.ObjectSourceBridge:
		if panel.OnBridgeMode == nil {
			return false
		}
		submenu := menu.AddMenuWithTitle("Режим спостереження МІСТ")
		current := bridgeDisplayBlockMode(object)
		for _, option := range []struct {
			label string
			mode  contracts.DisplayBlockMode
		}{
			{label: "Активне спостереження", mode: contracts.DisplayBlockNone},
			{label: "Тимчасово зняти зі спостереження", mode: contracts.DisplayBlockTemporaryOff},
			{label: "Режим налагодження", mode: contracts.DisplayBlockDebug},
		} {
			action := submenu.AddActionWithText(option.label)
			action.SetCheckable(true)
			action.SetChecked(option.mode == current)
			action.SetEnabled(option.mode != current)
			mode := option.mode
			label := option.label
			action.OnTriggered(func() {
				if qt.QMessageBox_Question(
					panel.table.QWidget,
					"Режим об'єкта МІСТ",
					fmt.Sprintf("Встановити для об'єкта №%s режим «%s»?", viewmodels.ObjectDisplayNumber(object), label),
				) == qt.QMessageBox__Yes {
					panel.OnBridgeMode(object, mode)
				}
			})
		}
		return true
	case viewmodels.ObjectSourceCASL:
		if panel.OnCASLBlock == nil {
			return false
		}
		label := "Блокувати об'єкт CASL..."
		if object.MonitoringStatusValue() == models.MonitoringStatusBlocked {
			label = "Керування блокуванням CASL..."
		}
		action := menu.AddActionWithText(label)
		action.OnTriggered(func() {
			panel.OnCASLBlock(object)
		})
		return true
	}
	return false
}

func bridgeDisplayBlockMode(object models.Object) contracts.DisplayBlockMode {
	switch object.MonitoringStatusValue() {
	case models.MonitoringStatusBlocked:
		return contracts.DisplayBlockTemporaryOff
	case models.MonitoringStatusDebug:
		return contracts.DisplayBlockDebug
	default:
		return contracts.DisplayBlockNone
	}
}

func objectListClipboardText(object models.Object) string {
	parts := make([]string, 0, 3)
	if number := strings.TrimSpace(viewmodels.ObjectDisplayNumber(object)); number != "" {
		parts = append(parts, "№"+number)
	}
	if name := strings.TrimSpace(object.Name); name != "" {
		parts = append(parts, name)
	}
	if address := strings.TrimSpace(object.Address); address != "" {
		parts = append(parts, address)
	}
	if len(parts) > 0 {
		return strings.Join(parts, " | ")
	}
	return fmt.Sprintf("ID %d", object.ID)
}

func setClipboardText(text string) {
	clipboard := qt.QGuiApplication_Clipboard()
	if clipboard != nil {
		clipboard.SetText(strings.TrimSpace(text))
	}
}

func indexToID(index *qt.QModelIndex) int {
	if index == nil || !index.IsValid() {
		return 0
	}
	return index.SiblingAtColumn(0).DataWithRole(int(qt.UserRole)).ToInt()
}
