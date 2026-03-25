package widgets

import (
	"sort"
	"strconv"
	"strings"
)

// SliceTableModel is a simple in-memory table model.
type SliceTableModel struct {
	Headers []string
	Rows    [][]string
}

// NewSliceTableModel creates a model from headers and row data.
func NewSliceTableModel(headers []string, rows [][]string) *SliceTableModel {
	m := &SliceTableModel{
		Headers: append([]string(nil), headers...),
		Rows:    make([][]string, len(rows)),
	}
	for i := range rows {
		m.Rows[i] = append([]string(nil), rows[i]...)
	}
	return m
}

// RowCount returns number of rows.
func (m *SliceTableModel) RowCount() int {
	if m == nil {
		return 0
	}
	return len(m.Rows)
}

// ColumnCount returns number of columns.
func (m *SliceTableModel) ColumnCount() int {
	if m == nil {
		return 0
	}
	return len(m.Headers)
}

// HeaderData returns header label for a column.
func (m *SliceTableModel) HeaderData(column int) string {
	if m == nil || column < 0 || column >= len(m.Headers) {
		return ""
	}
	return m.Headers[column]
}

// Data returns cell value by row/column.
func (m *SliceTableModel) Data(row, column int) string {
	if m == nil || row < 0 || row >= len(m.Rows) {
		return ""
	}
	if column < 0 || column >= len(m.Rows[row]) {
		return ""
	}
	return m.Rows[row][column]
}

// SetData updates one cell if indices are valid.
func (m *SliceTableModel) SetData(row, column int, value string) bool {
	if m == nil || row < 0 || row >= len(m.Rows) {
		return false
	}
	if column < 0 || column >= len(m.Rows[row]) {
		return false
	}
	m.Rows[row][column] = value
	return true
}

// SortByColumn sorts rows by a column.
func (m *SliceTableModel) SortByColumn(column int, order SortOrder) {
	if m == nil || column < 0 || column >= m.ColumnCount() {
		return
	}
	sort.SliceStable(m.Rows, func(i, j int) bool {
		left := strings.TrimSpace(m.Data(i, column))
		right := strings.TrimSpace(m.Data(j, column))
		if left == right {
			return i < j
		}

		lf, le := strconv.ParseFloat(strings.ReplaceAll(left, ",", "."), 64)
		rf, re := strconv.ParseFloat(strings.ReplaceAll(right, ",", "."), 64)
		if le == nil && re == nil {
			if order == SortDescending {
				return lf > rf
			}
			return lf < rf
		}

		if order == SortDescending {
			return strings.ToLower(left) > strings.ToLower(right)
		}
		return strings.ToLower(left) < strings.ToLower(right)
	})
}
