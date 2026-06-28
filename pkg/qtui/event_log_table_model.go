//go:build qt

package qtui

import (
	"image/color"
	"sort"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

type eventLogTableModel struct {
	*qt.QAbstractTableModel
	headers []string
	rows    []models.Event
}

func eventLogHeaders() []string {
	return []string{"Час", "№", "Подія", "Об'єкт", "Опис", "Джерело"}
}

func newEventLogTableModel(headers []string) *eventLogTableModel {
	model := &eventLogTableModel{
		QAbstractTableModel: qt.NewQAbstractTableModel(),
		headers:             append([]string(nil), headers...),
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
		event := model.rows[row]
		switch role {
		case int(qt.DisplayRole):
			return qt.NewQVariant14(eventLogCellText(event, column))
		case int(qt.UserRole):
			if column == 0 {
				return qt.NewQVariant4(event.ID)
			}
		case int(qt.ForegroundRole):
			textColor, _ := eventRowColorsBySeverity(event.VisualSeverityValue(), event.SC1)
			return qColorVariant(textColor)
		case int(qt.BackgroundRole):
			_, rowColor := eventRowColorsBySeverity(event.VisualSeverityValue(), event.SC1)
			return qColorVariant(rowColor)
		}
		return qt.NewQVariant()
	})
	model.OnSort(func(super func(column int, order qt.SortOrder), column int, order qt.SortOrder) {
		model.sort(column, order)
	})
	return model
}

func (model *eventLogTableModel) setRows(events []models.Event) {
	model.BeginResetModel()
	model.rows = append(model.rows[:0], events...)
	model.EndResetModel()
}

func (model *eventLogTableModel) eventAt(row int) (models.Event, bool) {
	if model == nil || row < 0 || row >= len(model.rows) {
		return models.Event{}, false
	}
	return model.rows[row], true
}

func (model *eventLogTableModel) sort(column int, order qt.SortOrder) {
	if model == nil || column < 0 || column >= len(model.headers) {
		return
	}
	model.LayoutAboutToBeChanged()
	sort.SliceStable(model.rows, func(i, j int) bool {
		left := eventLogCellText(model.rows[i], column)
		right := eventLogCellText(model.rows[j], column)
		if column == 0 {
			leftTime := model.rows[i].Time
			rightTime := model.rows[j].Time
			if order == qt.DescendingOrder {
				return leftTime.After(rightTime)
			}
			return leftTime.Before(rightTime)
		}
		if order == qt.DescendingOrder {
			return strings.Compare(left, right) > 0
		}
		return strings.Compare(left, right) < 0
	})
	model.LayoutChanged()
}

func eventLogCellText(event models.Event, column int) string {
	switch column {
	case 0:
		return event.GetDateTimeDisplay()
	case 1:
		return eventObjectNumber(event)
	case 2:
		return event.GetTypeDisplay()
	case 3:
		return strings.TrimSpace(event.ObjectName)
	case 4:
		return strings.TrimSpace(event.Details)
	case 5:
		return viewmodels.EventSourceName(event)
	default:
		return ""
	}
}

func qColorVariant(value color.NRGBA) *qt.QVariant {
	return qt.NewQColor11(int(value.R), int(value.G), int(value.B), int(value.A)).ToQVariant()
}
