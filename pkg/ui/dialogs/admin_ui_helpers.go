package dialogs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ─────────────────────────────────────────────────────────────────────────────
// Section header
// ─────────────────────────────────────────────────────────────────────────────

// makeSectionHeader повертає жирний заголовок секції з горизонтальним роздільником.
func makeSectionHeader(title string) fyne.CanvasObject {
	lbl := widget.NewLabel(title)
	lbl.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewVBox(lbl, widget.NewSeparator())
}

// ─────────────────────────────────────────────────────────────────────────────
// Typed buttons
// ─────────────────────────────────────────────────────────────────────────────

// makePrimaryButton повертає кнопку з підвищеним пріоритетом (синя/акцентна).
func makePrimaryButton(label string, fn func()) *widget.Button {
	btn := widget.NewButton(label, fn)
	btn.Importance = widget.HighImportance
	return btn
}

// makeDangerButton повертає кнопку для деструктивних дій (червона).
func makeDangerButton(label string, fn func()) *widget.Button {
	btn := widget.NewButton(label, fn)
	btn.Importance = widget.DangerImportance
	return btn
}

// makeLowButton повертає другорядну кнопку (приглушена).
func makeLowButton(label string, fn func()) *widget.Button {
	btn := widget.NewButton(label, fn)
	btn.Importance = widget.LowImportance
	return btn
}

// makeIconButton повертає кнопку з іконкою та заданим пріоритетом.
func makeIconButton(label string, icon fyne.Resource, importance widget.ButtonImportance, fn func()) *widget.Button {
	btn := widget.NewButtonWithIcon(label, icon, fn)
	btn.Importance = importance
	return btn
}

// makeIconOnlyButton повертає кнопку-іконку без підпису.
func makeIconOnlyButton(icon fyne.Resource, importance widget.ButtonImportance, fn func()) *widget.Button {
	btn := widget.NewButtonWithIcon("", icon, fn)
	btn.Importance = importance
	return btn
}

// ─────────────────────────────────────────────────────────────────────────────
// Status label
// ─────────────────────────────────────────────────────────────────────────────

// makeStatusLabel повертає Label, стилізований під рядок статусу
// (перенесення слів увімкнено для довгих повідомлень).
func makeStatusLabel(initial string) *widget.Label {
	lbl := widget.NewLabel(initial)
	lbl.Wrapping = fyne.TextWrapWord
	return lbl
}

// ─────────────────────────────────────────────────────────────────────────────
// Common icon shortcuts (helps avoid long fyneTheme.XxxIcon() chains across files)
// ─────────────────────────────────────────────────────────────────────────────

func iconAdd() fyne.Resource    { return fyneTheme.ContentAddIcon() }
func iconEdit() fyne.Resource   { return fyneTheme.DocumentCreateIcon() }
func iconDelete() fyne.Resource { return fyneTheme.DeleteIcon() }
func iconUp() fyne.Resource     { return fyneTheme.MoveUpIcon() }
func iconDown() fyne.Resource   { return fyneTheme.MoveDownIcon() }
func iconRefresh() fyne.Resource { return fyneTheme.ViewRefreshIcon() }
func iconExport() fyne.Resource { return fyneTheme.DocumentSaveIcon() }
func iconClose() fyne.Resource  { return fyneTheme.CancelIcon() }
func iconFolder() fyne.Resource { return fyneTheme.FolderOpenIcon() }
func iconClear() fyne.Resource  { return fyneTheme.ContentClearIcon() }
func iconSearch() fyne.Resource { return fyneTheme.SearchIcon() }
func iconColors() fyne.Resource { return fyneTheme.ColorPaletteIcon() }
