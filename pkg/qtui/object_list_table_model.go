//go:build qt

package qtui

import (
	"sort"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type objectListTableModel struct {
	*qt.QAbstractTableModel
	headers []string
	rows    []models.Object
	vm      *viewmodels.ObjectListViewModel
}

func newObjectListTableModel(vm *viewmodels.ObjectListViewModel) *objectListTableModel {
	model := &objectListTableModel{
		QAbstractTableModel: qt.NewQAbstractTableModel(),
		headers:             []string{"№", "Назва", "Адреса"},
		vm:                  vm,
	}
	model.OnRowCount(func(parent *qt.QModelIndex) int {
		if parent != nil && parent.IsValid() {
			return 0
		}
		return len(model.rows)
	})
	model.OnColumnCount(func(parent *qt.QModelIndex) int {
		if parent != nil && parent.IsValid() {
			return 0
		}
		return len(model.headers)
	})
	model.OnHeaderData(func(super func(section int, orientation qt.Orientation, role int) *qt.QVariant, section int, orientation qt.Orientation, role int) *qt.QVariant {
		if role == int(qt.DisplayRole) && orientation == qt.Horizontal && section >= 0 && section < len(model.headers) {
			return qt.NewQVariant14(model.headers[section])
		}
		return qt.NewQVariant()
	})
	model.OnData(func(index *qt.QModelIndex, role int) *qt.QVariant {
		if index == nil || !index.IsValid() {
			return qt.NewQVariant()
		}
		row := index.Row()
		column := index.Column()
		if row < 0 || row >= len(model.rows) || column < 0 || column >= len(model.headers) {
			return qt.NewQVariant()
		}
		object := model.rows[row]
		switch role {
		case int(qt.DisplayRole):
			return qt.NewQVariant14(objectListCellText(object, column))
		case int(qt.UserRole):
			if column == 0 {
				return qt.NewQVariant4(object.ID)
			}
		case int(qt.ForegroundRole):
			if model.vm != nil {
				textColor, _ := model.vm.GetRowColors(object, false)
				return qColorVariant(textColor)
			}
		case int(qt.BackgroundRole):
			if model.vm != nil {
				_, rowColor := model.vm.GetRowColors(object, false)
				return qColorVariant(rowColor)
			}
		}
		return qt.NewQVariant()
	})
	model.OnSort(func(super func(column int, order qt.SortOrder), column int, order qt.SortOrder) {
		model.sort(column, order)
	})
	return model
}

func (model *objectListTableModel) setRows(objects []models.Object) {
	model.BeginResetModel()
	model.rows = append(model.rows[:0], objects...)
	model.EndResetModel()
}

func (model *objectListTableModel) objectAt(row int) (models.Object, bool) {
	if model == nil || row < 0 || row >= len(model.rows) {
		return models.Object{}, false
	}
	return model.rows[row], true
}

func (model *objectListTableModel) rowForID(id int) int {
	if model == nil {
		return -1
	}
	for row, object := range model.rows {
		if object.ID == id {
			return row
		}
	}
	return -1
}

func (model *objectListTableModel) sort(column int, order qt.SortOrder) {
	if model == nil || column < 0 || column >= len(model.headers) {
		return
	}
	model.LayoutAboutToBeChanged()
	sort.SliceStable(model.rows, func(i, j int) bool {
		left := objectListCellText(model.rows[i], column)
		right := objectListCellText(model.rows[j], column)
		if order == qt.DescendingOrder {
			return strings.Compare(left, right) > 0
		}
		return strings.Compare(left, right) < 0
	})
	model.LayoutChanged()
}

func objectListCellText(object models.Object, column int) string {
	switch column {
	case 0:
		return viewmodels.ObjectDisplayNumber(object)
	case 1:
		return strings.TrimSpace(object.Name)
	case 2:
		return strings.TrimSpace(object.Address)
	default:
		return ""
	}
}
