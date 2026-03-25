package widgets

import (
	"image/color"
	"math"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SortOrder defines sort direction for QTableView.
type SortOrder int

const (
	// SortAscending sorts from A..Z / low..high.
	SortAscending SortOrder = iota
	// SortDescending sorts from Z..A / high..low.
	SortDescending
)

// PenStyle is a compatibility enum similar to Qt grid pen style.
// Fyne currently supports only show/hide separators, so style is stored as metadata.
type PenStyle int

const (
	// PenSolid draws regular separators.
	PenSolid PenStyle = iota
	// PenDash stores preference for dashed lines (rendered as solid in Fyne).
	PenDash
	// PenDot stores preference for dotted lines (rendered as solid in Fyne).
	PenDot
)

// ModelIndex points to model row/column.
type ModelIndex struct {
	Row int
	Col int
}

// IsValid reports whether index points to a model cell.
func (i ModelIndex) IsValid() bool {
	return i.Row >= 0 && i.Col >= 0
}

// TableModel is a minimal model/view contract used by QTableView.
type TableModel interface {
	RowCount() int
	ColumnCount() int
	Data(row, column int) string
	HeaderData(column int) string
}

// RowHeaderProvider allows custom row header text.
type RowHeaderProvider interface {
	RowHeaderData(row int) string
}

// SortableTableModel can sort data in-place on model side.
type SortableTableModel interface {
	SortByColumn(column int, order SortOrder)
}

type cellKey struct {
	row int
	col int
}

type spanData struct {
	rows int
	cols int
}

// QTableView is a Qt-inspired table view wrapper over Fyne's widget.Table.
// It provides common operations (hide/show rows/columns, sorting, sizing, selection)
// and exposes a Go-friendly API with close semantic mapping to QTableView.
type QTableView struct {
	Table *widget.Table

	OnSelected   func(index ModelIndex)
	OnUnselected func(index ModelIndex)

	model TableModel

	showGrid            bool
	wordWrap            bool
	sortingEnabled      bool
	cornerButtonEnabled bool
	gridStyle           PenStyle
	sortColumn          int
	sortOrder           SortOrder

	columnHidden map[int]bool
	rowHidden    map[int]bool

	columnWidths map[int]float32
	rowHeights   map[int]float32

	rowOrder      []int
	visibleRows   []int
	visibleCols   []int
	selectedIndex *ModelIndex

	spans   map[cellKey]spanData
	covered map[cellKey]cellKey

	defaultCellSize fyne.Size
}

// NewQTableView creates a new table view with model/view callbacks.
func NewQTableView(model TableModel) *QTableView {
	view := &QTableView{
		model:               model,
		showGrid:            true,
		wordWrap:            true,
		cornerButtonEnabled: true,
		gridStyle:           PenSolid,
		sortColumn:          -1,
		sortOrder:           SortAscending,
		columnHidden:        make(map[int]bool),
		rowHidden:           make(map[int]bool),
		columnWidths:        make(map[int]float32),
		rowHeights:          make(map[int]float32),
		spans:               make(map[cellKey]spanData),
		covered:             make(map[cellKey]cellKey),
	}

	cellTemplate := view.createCellTemplate()
	headerTemplate := view.createHeaderTemplate()
	view.defaultCellSize = cellTemplate.MinSize().Max(headerTemplate.MinSize())

	table := widget.NewTableWithHeaders(
		func() (int, int) { return view.length() },
		func() fyne.CanvasObject { return view.createCellTemplate() },
		func(id widget.TableCellID, o fyne.CanvasObject) { view.updateCell(id, o) },
	)
	table.CreateHeader = func() fyne.CanvasObject { return view.createHeaderTemplate() }
	table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) { view.updateHeader(id, o) }
	table.HideSeparators = false
	table.OnSelected = func(id widget.TableCellID) {
		idx, ok := view.toModelIndex(id)
		if !ok {
			return
		}
		view.selectedIndex = &idx
		if view.OnSelected != nil {
			view.OnSelected(idx)
		}
	}
	table.OnUnselected = func(id widget.TableCellID) {
		idx, ok := view.toModelIndex(id)
		if !ok {
			return
		}
		view.selectedIndex = nil
		if view.OnUnselected != nil {
			view.OnUnselected(idx)
		}
	}

	view.Table = table
	view.rebuildMappings()
	view.applySizes()
	return view
}

func (v *QTableView) length() (int, int) {
	if v.model == nil {
		return 0, 0
	}
	return len(v.visibleRows), len(v.visibleCols)
}

func (v *QTableView) createCellTemplate() fyne.CanvasObject {
	bg := canvas.NewRectangle(color.Transparent)
	txt := widget.NewLabel("")
	txt.Wrapping = fyne.TextWrapWord
	txt.Alignment = fyne.TextAlignLeading
	return container.NewStack(bg, container.NewPadded(txt))
}

func (v *QTableView) createHeaderTemplate() fyne.CanvasObject {
	l := widget.NewLabel("")
	l.Alignment = fyne.TextAlignCenter
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

func (v *QTableView) updateCell(id widget.TableCellID, o fyne.CanvasObject) {
	stack, ok := o.(*fyne.Container)
	if !ok || len(stack.Objects) < 2 {
		return
	}
	bg, ok := stack.Objects[0].(*canvas.Rectangle)
	if !ok {
		return
	}
	padded, ok := stack.Objects[1].(*fyne.Container)
	if !ok || len(padded.Objects) == 0 {
		return
	}
	lbl, ok := padded.Objects[0].(*widget.Label)
	if !ok {
		return
	}

	bg.FillColor = color.Transparent
	bg.Refresh()

	idx, valid := v.toModelIndex(id)
	if !valid {
		lbl.SetText("")
		return
	}
	if v.isCovered(idx.Row, idx.Col) {
		lbl.SetText("")
		return
	}

	lbl.Wrapping = fyne.TextWrapOff
	lbl.Truncation = fyne.TextTruncateEllipsis
	if v.wordWrap {
		lbl.Wrapping = fyne.TextWrapWord
		lbl.Truncation = fyne.TextTruncateClip
	}
	lbl.SetText(v.safeData(idx.Row, idx.Col))
}

func (v *QTableView) updateHeader(id widget.TableCellID, o fyne.CanvasObject) {
	lbl, ok := o.(*widget.Label)
	if !ok {
		return
	}

	switch {
	case id.Row < 0 && id.Col >= 0:
		if id.Col >= len(v.visibleCols) {
			lbl.SetText("")
			return
		}
		col := v.visibleCols[id.Col]
		lbl.SetText(v.model.HeaderData(col))
	case id.Col < 0 && id.Row >= 0:
		if id.Row >= len(v.visibleRows) {
			lbl.SetText("")
			return
		}
		row := v.visibleRows[id.Row]
		if provider, ok := v.model.(RowHeaderProvider); ok {
			lbl.SetText(provider.RowHeaderData(row))
			return
		}
		lbl.SetText(strconv.Itoa(row + 1))
	default:
		lbl.SetText("")
	}
}

func (v *QTableView) safeData(row, col int) string {
	if v.model == nil {
		return ""
	}
	if row < 0 || row >= v.model.RowCount() || col < 0 || col >= v.model.ColumnCount() {
		return ""
	}
	return v.model.Data(row, col)
}

func (v *QTableView) toModelIndex(id widget.TableCellID) (ModelIndex, bool) {
	if id.Row < 0 || id.Col < 0 {
		return ModelIndex{}, false
	}
	if id.Row >= len(v.visibleRows) || id.Col >= len(v.visibleCols) {
		return ModelIndex{}, false
	}
	return ModelIndex{
		Row: v.visibleRows[id.Row],
		Col: v.visibleCols[id.Col],
	}, true
}

func (v *QTableView) viewRow(modelRow int) int {
	for i, row := range v.visibleRows {
		if row == modelRow {
			return i
		}
	}
	return -1
}

func (v *QTableView) viewCol(modelCol int) int {
	for i, col := range v.visibleCols {
		if col == modelCol {
			return i
		}
	}
	return -1
}

func (v *QTableView) rebuildMappings() {
	if v.model == nil {
		v.rowOrder = nil
		v.visibleRows = nil
		v.visibleCols = nil
		return
	}

	rows := v.model.RowCount()
	cols := v.model.ColumnCount()

	if len(v.rowOrder) != rows {
		v.rowOrder = make([]int, rows)
		for i := 0; i < rows; i++ {
			v.rowOrder[i] = i
		}
	}

	v.visibleRows = v.visibleRows[:0]
	for _, row := range v.rowOrder {
		if row < 0 || row >= rows {
			continue
		}
		if v.rowHidden[row] {
			continue
		}
		v.visibleRows = append(v.visibleRows, row)
	}

	v.visibleCols = v.visibleCols[:0]
	for c := 0; c < cols; c++ {
		if v.columnHidden[c] {
			continue
		}
		v.visibleCols = append(v.visibleCols, c)
	}
}

func (v *QTableView) applySizes() {
	if v.Table == nil {
		return
	}
	for viewCol, modelCol := range v.visibleCols {
		v.Table.SetColumnWidth(viewCol, v.columnWidthRaw(modelCol))
	}
	for viewRow, modelRow := range v.visibleRows {
		v.Table.SetRowHeight(viewRow, v.rowHeightRaw(modelRow))
	}
}

func (v *QTableView) separatorSize() float32 {
	return v.Table.Theme().Size(theme.SizeNamePadding)
}

func (v *QTableView) columnWidthRaw(modelCol int) float32 {
	if width, ok := v.columnWidths[modelCol]; ok && width > 0 {
		return width
	}
	return v.defaultCellSize.Width
}

func (v *QTableView) rowHeightRaw(modelRow int) float32 {
	if height, ok := v.rowHeights[modelRow]; ok && height > 0 {
		return height
	}
	return v.defaultCellSize.Height
}

func (v *QTableView) rebuildCoveredMap() {
	v.covered = make(map[cellKey]cellKey)
	for origin, span := range v.spans {
		if span.rows <= 1 && span.cols <= 1 {
			continue
		}
		for r := origin.row; r < origin.row+span.rows; r++ {
			for c := origin.col; c < origin.col+span.cols; c++ {
				key := cellKey{row: r, col: c}
				if key == origin {
					continue
				}
				v.covered[key] = origin
			}
		}
	}
}

func (v *QTableView) isCovered(row, col int) bool {
	_, ok := v.covered[cellKey{row: row, col: col}]
	return ok
}

// Widget returns underlying Fyne table widget.
func (v *QTableView) Widget() *widget.Table {
	return v.Table
}

// SetModel replaces the current model and refreshes mappings.
func (v *QTableView) SetModel(model TableModel) {
	v.model = model
	v.rowOrder = nil
	v.selectedIndex = nil
	v.rebuildMappings()
	v.applySizes()
	if v.Table != nil {
		v.Table.UnselectAll()
		v.Table.Refresh()
	}
}

// Refresh rebuilds mappings and refreshes the table.
func (v *QTableView) Refresh() {
	v.rebuildMappings()
	v.applySizes()
	v.Table.Refresh()
}

// IsCornerButtonEnabled returns current corner button setting.
func (v *QTableView) IsCornerButtonEnabled() bool {
	return v.cornerButtonEnabled
}

// SetCornerButtonEnabled stores corner button mode.
// Fyne has no clickable corner button in table headers; use SelectAll() for equivalent behavior.
func (v *QTableView) SetCornerButtonEnabled(enable bool) {
	v.cornerButtonEnabled = enable
}

// SelectAll selects the first visible cell.
// Note: unlike Qt multi-selection, Fyne Table supports single selected cell.
func (v *QTableView) SelectAll() {
	if len(v.visibleRows) == 0 || len(v.visibleCols) == 0 {
		return
	}
	v.Table.Select(widget.TableCellID{Row: 0, Col: 0})
}

// GridStyle returns preferred pen style metadata.
func (v *QTableView) GridStyle() PenStyle {
	return v.gridStyle
}

// SetGridStyle stores preferred pen style metadata.
func (v *QTableView) SetGridStyle(style PenStyle) {
	v.gridStyle = style
}

// ShowGrid reports whether separators are visible.
func (v *QTableView) ShowGrid() bool {
	return v.showGrid
}

// SetShowGrid toggles table separators visibility.
func (v *QTableView) SetShowGrid(show bool) {
	v.showGrid = show
	v.Table.HideSeparators = !show
	v.Table.Refresh()
}

// WordWrap reports whether cell labels wrap text.
func (v *QTableView) WordWrap() bool {
	return v.wordWrap
}

// SetWordWrap toggles label wrapping.
func (v *QTableView) SetWordWrap(on bool) {
	v.wordWrap = on
	v.Table.Refresh()
}

// IsSortingEnabled reports whether auto-sort is enabled.
func (v *QTableView) IsSortingEnabled() bool {
	return v.sortingEnabled
}

// SetSortingEnabled enables/disables sorting.
// Enabling triggers immediate sort by current sort column if it is valid.
func (v *QTableView) SetSortingEnabled(enable bool) {
	v.sortingEnabled = enable
	if !enable || v.sortColumn < 0 {
		return
	}
	v.SortByColumn(v.sortColumn, v.sortOrder)
}

// SortByColumn sorts by model column.
func (v *QTableView) SortByColumn(column int, order SortOrder) {
	v.sortColumn = column
	v.sortOrder = order
	if v.model == nil {
		return
	}
	if column < 0 || column >= v.model.ColumnCount() {
		return
	}

	if sorter, ok := v.model.(SortableTableModel); ok {
		sorter.SortByColumn(column, order)
		rows := v.model.RowCount()
		v.rowOrder = make([]int, rows)
		for i := 0; i < rows; i++ {
			v.rowOrder[i] = i
		}
		v.Refresh()
		return
	}

	if len(v.rowOrder) == 0 {
		rows := v.model.RowCount()
		v.rowOrder = make([]int, rows)
		for i := 0; i < rows; i++ {
			v.rowOrder[i] = i
		}
	}

	sort.SliceStable(v.rowOrder, func(i, j int) bool {
		left := strings.TrimSpace(v.safeData(v.rowOrder[i], column))
		right := strings.TrimSpace(v.safeData(v.rowOrder[j], column))
		if left == right {
			return v.rowOrder[i] < v.rowOrder[j]
		}

		lf, le := strconv.ParseFloat(strings.ReplaceAll(left, ",", "."), 64)
		rf, re := strconv.ParseFloat(strings.ReplaceAll(right, ",", "."), 64)
		if le == nil && re == nil {
			if order == SortDescending {
				return lf > rf
			}
			return lf < rf
		}

		cmp := strings.Compare(strings.ToLower(left), strings.ToLower(right))
		if order == SortDescending {
			return cmp > 0
		}
		return cmp < 0
	})
	v.Refresh()
}

// ClearSpans removes all span metadata.
// Note: Fyne Table does not support true merged cells, spans are metadata + covered-cell suppression.
func (v *QTableView) ClearSpans() {
	v.spans = make(map[cellKey]spanData)
	v.covered = make(map[cellKey]cellKey)
	v.Table.Refresh()
}

// SetSpan stores span metadata for a model cell.
func (v *QTableView) SetSpan(row, column, rowSpanCount, columnSpanCount int) {
	if row < 0 || column < 0 || rowSpanCount < 1 || columnSpanCount < 1 {
		return
	}
	key := cellKey{row: row, col: column}
	v.spans[key] = spanData{rows: rowSpanCount, cols: columnSpanCount}
	v.rebuildCoveredMap()
	v.Table.Refresh()
}

// RowSpan returns row span for model cell.
func (v *QTableView) RowSpan(row, column int) int {
	span, ok := v.spans[cellKey{row: row, col: column}]
	if !ok {
		return 1
	}
	return span.rows
}

// ColumnSpan returns column span for model cell.
func (v *QTableView) ColumnSpan(row, column int) int {
	span, ok := v.spans[cellKey{row: row, col: column}]
	if !ok {
		return 1
	}
	return span.cols
}

// ColumnAt returns model column at x in contents coordinates.
func (v *QTableView) ColumnAt(x int) int {
	curX := float32(0)
	target := float32(x)
	sep := v.separatorSize()
	for _, col := range v.visibleCols {
		width := v.columnWidthRaw(col)
		if target >= curX && target < curX+width {
			return col
		}
		curX += width + sep
	}
	return -1
}

// RowAt returns model row at y in contents coordinates.
func (v *QTableView) RowAt(y int) int {
	curY := float32(0)
	target := float32(y)
	sep := v.separatorSize()
	for _, row := range v.visibleRows {
		height := v.rowHeightRaw(row)
		if target >= curY && target < curY+height {
			return row
		}
		curY += height + sep
	}
	return -1
}

// ColumnViewportPosition returns x in contents coordinates for model column.
func (v *QTableView) ColumnViewportPosition(column int) int {
	curX := float32(0)
	sep := v.separatorSize()
	for _, col := range v.visibleCols {
		if col == column {
			return int(math.Round(float64(curX)))
		}
		curX += v.columnWidthRaw(col) + sep
	}
	return -1
}

// RowViewportPosition returns y in contents coordinates for model row.
func (v *QTableView) RowViewportPosition(row int) int {
	curY := float32(0)
	sep := v.separatorSize()
	for _, r := range v.visibleRows {
		if r == row {
			return int(math.Round(float64(curY)))
		}
		curY += v.rowHeightRaw(r) + sep
	}
	return -1
}

// ColumnWidth returns configured width for model column.
func (v *QTableView) ColumnWidth(column int) int {
	return int(math.Round(float64(v.columnWidthRaw(column))))
}

// RowHeight returns configured height for model row.
func (v *QTableView) RowHeight(row int) int {
	return int(math.Round(float64(v.rowHeightRaw(row))))
}

// SetColumnWidth sets width for model column.
func (v *QTableView) SetColumnWidth(column int, width int) {
	if width < 1 {
		width = 1
	}
	fw := float32(width)
	v.columnWidths[column] = fw
	if idx := v.viewCol(column); idx >= 0 {
		v.Table.SetColumnWidth(idx, fw)
	}
}

// SetRowHeight sets height for model row.
func (v *QTableView) SetRowHeight(row int, height int) {
	if height < 1 {
		height = 1
	}
	fh := float32(height)
	v.rowHeights[row] = fh
	if idx := v.viewRow(row); idx >= 0 {
		v.Table.SetRowHeight(idx, fh)
	}
}

// IsColumnHidden reports whether model column is hidden.
func (v *QTableView) IsColumnHidden(column int) bool {
	return v.columnHidden[column]
}

// IsRowHidden reports whether model row is hidden.
func (v *QTableView) IsRowHidden(row int) bool {
	return v.rowHidden[row]
}

// SetColumnHidden hides/shows model column.
func (v *QTableView) SetColumnHidden(column int, hide bool) {
	if hide {
		v.columnHidden[column] = true
	} else {
		delete(v.columnHidden, column)
	}
	v.Refresh()
}

// SetRowHidden hides/shows model row.
func (v *QTableView) SetRowHidden(row int, hide bool) {
	if hide {
		v.rowHidden[row] = true
	} else {
		delete(v.rowHidden, row)
	}
	v.Refresh()
}

// HideColumn hides model column.
func (v *QTableView) HideColumn(column int) {
	v.SetColumnHidden(column, true)
}

// ShowColumn shows model column.
func (v *QTableView) ShowColumn(column int) {
	v.SetColumnHidden(column, false)
}

// HideRow hides model row.
func (v *QTableView) HideRow(row int) {
	v.SetRowHidden(row, true)
}

// ShowRow shows model row.
func (v *QTableView) ShowRow(row int) {
	v.SetRowHidden(row, false)
}

// SelectRow selects first visible column in model row.
func (v *QTableView) SelectRow(row int) {
	viewRow := v.viewRow(row)
	if viewRow < 0 || len(v.visibleCols) == 0 {
		return
	}
	v.Table.Select(widget.TableCellID{Row: viewRow, Col: 0})
}

// SelectColumn selects first visible row in model column.
func (v *QTableView) SelectColumn(column int) {
	viewCol := v.viewCol(column)
	if viewCol < 0 || len(v.visibleRows) == 0 {
		return
	}
	v.Table.Select(widget.TableCellID{Row: 0, Col: viewCol})
}

// ScrollTo scrolls to model index.
func (v *QTableView) ScrollTo(index ModelIndex) {
	if !index.IsValid() {
		return
	}
	row := v.viewRow(index.Row)
	col := v.viewCol(index.Col)
	if row < 0 || col < 0 {
		return
	}
	v.Table.ScrollTo(widget.TableCellID{Row: row, Col: col})
}

// IndexAt returns model index in contents coordinates.
func (v *QTableView) IndexAt(x, y int) (ModelIndex, bool) {
	row := v.RowAt(y)
	col := v.ColumnAt(x)
	if row < 0 || col < 0 {
		return ModelIndex{}, false
	}
	return ModelIndex{Row: row, Col: col}, true
}

// CurrentIndex returns selected model index.
func (v *QTableView) CurrentIndex() (ModelIndex, bool) {
	if v.selectedIndex == nil {
		return ModelIndex{}, false
	}
	return *v.selectedIndex, true
}

// ResizeColumnToContents computes width by longest header/data text for model column.
func (v *QTableView) ResizeColumnToContents(column int) {
	if v.model == nil || column < 0 || column >= v.model.ColumnCount() {
		return
	}

	fontSize := theme.DefaultTheme().Size(theme.SizeNameText)
	style := fyne.TextStyle{}
	maxWidth := fyne.MeasureText(v.model.HeaderData(column), fontSize, style).Width
	for row := 0; row < v.model.RowCount(); row++ {
		if v.rowHidden[row] {
			continue
		}
		width := fyne.MeasureText(v.safeData(row, column), fontSize, style).Width
		if width > maxWidth {
			maxWidth = width
		}
	}
	padding := v.Table.Theme().Size(theme.SizeNamePadding) * 2
	v.SetColumnWidth(column, int(math.Ceil(float64(maxWidth+padding))))
}

// ResizeColumnsToContents applies ResizeColumnToContents for all visible columns.
func (v *QTableView) ResizeColumnsToContents() {
	for _, col := range v.visibleCols {
		v.ResizeColumnToContents(col)
	}
}

// ResizeRowToContents computes row height based on max wrapped-line count in row.
func (v *QTableView) ResizeRowToContents(row int) {
	if v.model == nil || row < 0 || row >= v.model.RowCount() {
		return
	}
	fontSize := theme.DefaultTheme().Size(theme.SizeNameText)
	style := fyne.TextStyle{}
	lineHeight := fyne.MeasureText("Wg", fontSize, style).Height
	if lineHeight <= 0 {
		lineHeight = v.defaultCellSize.Height
	}

	maxHeight := lineHeight
	for _, col := range v.visibleCols {
		text := v.safeData(row, col)
		cellWidth := v.columnWidthRaw(col)
		if !v.wordWrap || cellWidth <= 1 {
			continue
		}
		textWidth := fyne.MeasureText(text, fontSize, style).Width
		lines := int(math.Ceil(float64(textWidth / cellWidth)))
		if lines < 1 {
			lines = 1
		}
		height := float32(lines) * lineHeight
		if height > maxHeight {
			maxHeight = height
		}
	}
	padding := v.Table.Theme().Size(theme.SizeNamePadding) * 2
	v.SetRowHeight(row, int(math.Ceil(float64(maxHeight+padding))))
}

// ResizeRowsToContents applies ResizeRowToContents for all visible rows.
func (v *QTableView) ResizeRowsToContents() {
	for _, row := range v.visibleRows {
		v.ResizeRowToContents(row)
	}
}
