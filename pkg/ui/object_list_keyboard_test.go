package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestObjectListTableTypedKeyMovesSelectionDown(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	panel, selectedIDs := newKeyboardTestPanel()
	panel.SelectedRow = 1
	panel.SelectedCol = 2

	panel.Table.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})

	if panel.SelectedRow != 2 {
		t.Fatalf("expected row 2 after KeyDown, got %d", panel.SelectedRow)
	}
	if panel.SelectedCol != 2 {
		t.Fatalf("expected selected column 2 to be preserved, got %d", panel.SelectedCol)
	}
	if len(*selectedIDs) != 1 || (*selectedIDs)[0] != 3 {
		t.Fatalf("expected object ID 3 to be selected, got %v", *selectedIDs)
	}
}

func TestObjectListTableTypedKeySelectsLastRowWhenNothingSelected(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	panel, selectedIDs := newKeyboardTestPanel()

	panel.Table.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})

	if panel.SelectedRow != 2 {
		t.Fatalf("expected last row to be selected on KeyUp without prior selection, got %d", panel.SelectedRow)
	}
	if panel.SelectedCol != 0 {
		t.Fatalf("expected default column 0, got %d", panel.SelectedCol)
	}
	if len(*selectedIDs) != 1 || (*selectedIDs)[0] != 3 {
		t.Fatalf("expected object ID 3 to be selected, got %v", *selectedIDs)
	}
}

func newKeyboardTestPanel() (*ObjectListPanel, *[]int) {
	objects := []models.Object{
		{ID: 1, Name: "One"},
		{ID: 2, Name: "Two"},
		{ID: 3, Name: "Three"},
	}

	panel := &ObjectListPanel{
		FilteredData:  binding.NewUntypedList(),
		FilteredItems: append([]models.Object(nil), objects...),
		SelectedRow:   -1,
		SelectedCol:   0,
	}
	if err := SetUntypedList(panel.FilteredData, objects); err != nil {
		panic(err)
	}

	panel.Table = newObjectListTable(
		func() (int, int) {
			return len(panel.FilteredItems), 4
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(widget.TableCellID, fyne.CanvasObject) {},
		panel.moveSelection,
	)

	selectedIDs := make([]int, 0, 1)
	panel.OnObjectSelected = func(object models.Object) {
		selectedIDs = append(selectedIDs, object.ID)
	}
	panel.Table.OnSelected = panel.handleSelection

	return panel, &selectedIDs
}
