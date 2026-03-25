# QTableView for Fyne

`QTableView` is a Qt-inspired wrapper over `fyne.io/fyne/v2/widget.Table`.

## Quick start

```go
model := widgets.NewSliceTableModel(
	[]string{"ID", "Name", "Address"},
	[][]string{
		{"1", "Alpha", "Kyiv"},
		{"2", "Beta", "Lviv"},
	},
)

tableView := widgets.NewQTableView(model)
tableView.SetSortingEnabled(true)
tableView.SortByColumn(1, widgets.SortAscending)
tableView.SetColumnWidth(1, 260)
tableView.SetShowGrid(true)
tableView.SetWordWrap(true)

content := tableView.Widget()
```

## Implemented features

- model/view callbacks (`TableModel`)
- row/column hide/show
- sort by column
- row/column size control
- auto resize columns/rows to contents
- row/column selection helpers
- content coordinate helpers (`RowAt`, `ColumnAt`)
- header row/column support
- span metadata (`SetSpan`) with covered-cell suppression

## Notes

- Fyne has single-cell selection in `widget.Table`, so `SelectAll` is limited.
- Grid pen style is stored as metadata; rendering is solid or hidden.
- True merged cell painting is not available in Fyne table renderer.
