//go:build qt

package qtui

import (
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type ObjectListPanel struct {
	*qt.QWidget
	search           *qt.QLineEdit
	statusFilter     *qt.QComboBox
	sourceFilter     *qt.QComboBox
	table            *qt.QTableView
	model            *qt.QStandardItemModel
	vm               *viewmodels.ObjectListViewModel
	allObjects       []models.Object
	objectsByID      map[int]models.Object
	autoSized        bool
	OnObjectSelected func(models.Object)
}

func NewObjectListPanel() *ObjectListPanel {
	panel := &ObjectListPanel{
		QWidget: qt.NewQWidget2(),
		vm:      viewmodels.NewObjectListViewModel(),
	}
	panel.SetMinimumWidth(320)

	layout := qt.NewQVBoxLayout(panel.QWidget)
	title := qt.NewQLabel3("Список об'єктів")
	title.SetStyleSheet("font-weight: 600; font-size: 11pt; padding: 4px 0;")

	panel.search = qt.NewQLineEdit2()
	panel.search.SetPlaceholderText("Пошук за назвою, адресою або номером")
	panel.search.SetClearButtonEnabled(true)

	filtersLayout := qt.NewQHBoxLayout2()
	panel.statusFilter = qt.NewQComboBox2()
	panel.statusFilter.AddItems(panel.vm.BuildFilterOptions(0, 0, 0, 0, 0))
	panel.sourceFilter = qt.NewQComboBox2()
	panel.sourceFilter.AddItems(viewmodels.BuildObjectSourceOptions(0, 0, 0, 0))
	filtersLayout.AddWidget(panel.statusFilter.QWidget)
	filtersLayout.AddWidget(panel.sourceFilter.QWidget)

	panel.model = qt.NewQStandardItemModel2(0, 3)
	panel.model.SetHorizontalHeaderLabels([]string{"№", "Назва", "Адреса"})

	panel.table = qt.NewQTableView2()
	panel.table.SetModel(panel.model.QAbstractItemModel)
	panel.table.SetSortingEnabled(true)
	panel.table.SetAlternatingRowColors(true)
	panel.table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	panel.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	panel.table.HorizontalHeader().SetStretchLastSection(true)
	panel.table.SelectionModel().OnCurrentRowChanged(func(current *qt.QModelIndex, previous *qt.QModelIndex) {
		panel.notifyObjectSelection(current)
	})

	panel.search.OnTextChanged(func(string) {
		panel.applyFilters()
	})
	panel.statusFilter.OnCurrentTextChanged(func(string) {
		panel.applyFilters()
	})
	panel.sourceFilter.OnCurrentTextChanged(func(string) {
		panel.applyFilters()
	})

	layout.AddWidget(title.QWidget)
	layout.AddWidget(panel.search.QWidget)
	layout.AddLayout(filtersLayout.QLayout)
	layout.AddWidget(panel.table.QWidget)
	panel.SetLayout(layout.QLayout)
	panel.applyFilters()

	return panel
}

func (panel *ObjectListPanel) SetObjects(objects []models.Object) {
	if panel == nil {
		return
	}
	panel.allObjects = append(panel.allObjects[:0], objects...)
	panel.objectsByID = make(map[int]models.Object, len(objects))
	for _, object := range objects {
		panel.objectsByID[object.ID] = object
	}
	panel.applyFilters()
}

func (panel *ObjectListPanel) FocusSearch() {
	if panel == nil || panel.search == nil {
		return
	}
	panel.search.SetFocus()
	panel.search.SelectAll()
}

func (panel *ObjectListPanel) notifyObjectSelection(index *qt.QModelIndex) {
	if panel == nil || panel.OnObjectSelected == nil || index == nil || !index.IsValid() {
		return
	}
	id := index.SiblingAtColumn(0).DataWithRole(int(qt.UserRole)).ToInt()
	if object, ok := panel.objectsByID[id]; ok {
		panel.OnObjectSelected(object)
	}
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
	panel.model.Clear()
	panel.model.SetHorizontalHeaderLabels([]string{"№", "Назва", "Адреса"})
	for _, object := range objects {
		values := []string{
			viewmodels.ObjectDisplayNumber(object),
			strings.TrimSpace(object.Name),
			strings.TrimSpace(object.Address),
		}
		textColor, rowColor := panel.vm.GetRowColors(object, false)
		addColoredReadOnlyRow(panel.model, values, object.ID, textColor, rowColor)
	}
	if !panel.autoSized {
		resizeObjectListColumns(panel.table)
		panel.autoSized = true
	}
}

func (panel *ObjectListPanel) SelectObject(id int) {
	if panel == nil || panel.model == nil || panel.table == nil {
		return
	}
	selectionModel := panel.table.SelectionModel()
	if selectionModel != nil {
		currentIndex := selectionModel.CurrentIndex()
		if currentIndex != nil && currentIndex.IsValid() {
			currentID := indexToID(panel.model, currentIndex)
			if currentID == id {
				return
			}
		}
	}

	for row := 0; row < panel.model.RowCount(qt.NewQModelIndex()); row++ {
		index := panel.model.Index(row, 0, nil)
		if index != nil && index.IsValid() {
			val := panel.model.Data(index, int(qt.UserRole)).ToInt()
			if val == id {
				panel.table.SelectRow(row)
				panel.table.ScrollTo(index, qt.QAbstractItemView__PositionAtCenter)
				return
			}
		}
	}
}

func indexToID(model *qt.QStandardItemModel, index *qt.QModelIndex) int {
	if model == nil || index == nil || !index.IsValid() {
		return 0
	}
	rowIndex := model.Index(index.Row(), 0, nil)
	if rowIndex == nil || !rowIndex.IsValid() {
		return 0
	}
	return model.Data(rowIndex, int(qt.UserRole)).ToInt()
}

