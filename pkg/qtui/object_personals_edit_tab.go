//go:build qt

package qtui

import (
	"fmt"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectPersonalsEditTab struct {
	parent      *qt.QWidget
	provider    contracts.AdminObjectPersonalService
	objn        int64
	statusLabel *qt.QLabel
	table       *qt.QTableWidget
	items       []contracts.AdminObjectPersonal
	loaded      bool
}

func newObjectPersonalsEditTab(parent *qt.QWidget, provider contracts.AdminObjectPersonalService, objn int64, statusLabel *qt.QLabel) (*qt.QWidget, func()) {
	tab := &objectPersonalsEditTab{
		parent:      parent,
		provider:    provider,
		objn:        objn,
		statusLabel: statusLabel,
		table:       qt.NewQTableWidget3(0, 6),
	}
	return tab.build(), tab.ensureLoaded
}

func (t *objectPersonalsEditTab) build() *qt.QWidget {
	t.table.SetHorizontalHeaderLabels([]string{"№", "ПІБ", "Телефон", "Посада", "Доступ", "Примітка"})
	t.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	t.table.OnCellDoubleClicked(func(row int, _ int) {
		t.editPersonal(row)
	})
	t.applyColumnWidths()

	addBtn := qt.NewQPushButton3("Додати")
	editBtn := qt.NewQPushButton3("Змінити")
	deleteBtn := qt.NewQPushButton3("Видалити")
	refreshBtn := qt.NewQPushButton3("Оновити")

	addBtn.OnClicked(t.addPersonal)
	editBtn.OnClicked(func() { t.editPersonal(t.table.CurrentRow()) })
	deleteBtn.OnClicked(t.deletePersonal)
	refreshBtn.OnClicked(t.reload)

	toolbar := qt.NewQHBoxLayout2()
	toolbar.AddWidget(addBtn.QWidget)
	toolbar.AddWidget(editBtn.QWidget)
	toolbar.AddWidget(deleteBtn.QWidget)
	toolbar.AddStretch()
	toolbar.AddWidget(refreshBtn.QWidget)

	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)
	layout.AddLayout(toolbar.QLayout)
	layout.AddWidget(t.table.QWidget)
	widget.SetLayout(layout.QLayout)

	return widget
}

func (t *objectPersonalsEditTab) ensureLoaded() {
	if t.loaded {
		return
	}
	t.reload()
}

func (t *objectPersonalsEditTab) reload() {
	items, err := t.provider.ListObjectPersonals(t.objn)
	if err != nil {
		t.showError("Не вдалося завантажити В/О", err)
		return
	}
	t.loaded = true
	t.items = items
	t.table.SetRowCount(len(items))
	for row, item := range items {
		t.table.SetItem(row, 0, qt.NewQTableWidgetItem2(formatInt64NonZero(item.Number)))
		t.table.SetItem(row, 1, qt.NewQTableWidgetItem2(objectPersonalFullName(item)))
		t.table.SetItem(row, 2, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Phones)))
		t.table.SetItem(row, 3, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Position)))
		if item.Access1 > 0 {
			t.table.SetItem(row, 4, qt.NewQTableWidgetItem2("Адмін"))
		} else {
			t.table.SetItem(row, 4, qt.NewQTableWidgetItem2("Оператор"))
		}
		t.table.SetItem(row, 5, qt.NewQTableWidgetItem2(strings.TrimSpace(item.Notes)))
	}
	t.applyColumnWidths()
	t.setStatus(fmt.Sprintf("В/О: %d запис(ів)", len(items)))
}

func (t *objectPersonalsEditTab) addPersonal() {
	nextNumber := int64(len(t.items) + 1)
	item, ok := showObjectPersonalEditDialog(t.parent, contracts.AdminObjectPersonal{
		Number: nextNumber,
		IsRang: true,
	})
	if !ok {
		return
	}
	if err := t.provider.AddObjectPersonal(t.objn, item); err != nil {
		t.showError("Не вдалося додати В/О", err)
		return
	}
	t.reload()
	t.setStatus("В/О додано")
}

func (t *objectPersonalsEditTab) editPersonal(row int) {
	item, ok := t.itemAt(row)
	if !ok {
		t.setStatus("Виберіть В/О у таблиці")
		return
	}
	updated, ok := showObjectPersonalEditDialog(t.parent, item)
	if !ok {
		return
	}
	updated.ID = item.ID
	if strings.TrimSpace(updated.CreatedAt) == "" {
		updated.CreatedAt = item.CreatedAt
	}
	if err := t.provider.UpdateObjectPersonal(t.objn, updated); err != nil {
		t.showError("Не вдалося оновити В/О", err)
		return
	}
	t.reload()
	t.setStatus("В/О оновлено")
}

func (t *objectPersonalsEditTab) deletePersonal() {
	item, ok := t.itemAt(t.table.CurrentRow())
	if !ok {
		t.setStatus("Виберіть В/О у таблиці")
		return
	}
	if qt.QMessageBox_Question(t.parent, "Підтвердження", fmt.Sprintf("Видалити запис \"%s\"?", objectPersonalFullName(item))) != qt.QMessageBox__Yes {
		return
	}
	if err := t.provider.DeleteObjectPersonal(t.objn, item.ID); err != nil {
		t.showError("Не вдалося видалити В/О", err)
		return
	}
	t.reload()
	t.setStatus("В/О видалено")
}

func (t *objectPersonalsEditTab) itemAt(row int) (contracts.AdminObjectPersonal, bool) {
	if row < 0 || row >= len(t.items) {
		return contracts.AdminObjectPersonal{}, false
	}
	return t.items[row], true
}

func (t *objectPersonalsEditTab) applyColumnWidths() {
	t.table.SetColumnWidth(0, 60)
	t.table.SetColumnWidth(1, 260)
	t.table.SetColumnWidth(2, 180)
	t.table.SetColumnWidth(3, 160)
	t.table.SetColumnWidth(4, 100)
	t.table.SetColumnWidth(5, 220)
}

func (t *objectPersonalsEditTab) setStatus(text string) {
	if t.statusLabel != nil {
		t.statusLabel.SetText(strings.TrimSpace(text))
	}
}

func (t *objectPersonalsEditTab) showError(prefix string, err error) {
	message := strings.TrimSpace(prefix)
	if err != nil {
		message += ": " + err.Error()
	}
	t.setStatus(message)
	qt.QMessageBox_Critical(t.parent, prefix, message)
}

func showObjectPersonalEditDialog(parent *qt.QWidget, initial contracts.AdminObjectPersonal) (contracts.AdminObjectPersonal, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Відповідальна особа")
	dialog.Resize(620, 520)

	number := newLineEdit(formatInt64NonZero(initial.Number))
	surname := newLineEdit(initial.Surname)
	name := newLineEdit(initial.Name)
	secName := newLineEdit(initial.SecName)
	address := newLineEdit(initial.Address)
	phones := newLineEdit(initial.Phones)
	position := newLineEdit(initial.Position)
	notes := newLineEdit(initial.Notes)
	viberID := newLineEdit(initial.ViberID)
	telegramID := newLineEdit(initial.TelegramID)
	createdAt := qt.NewQLabel3(emptyDash(initial.CreatedAt))
	isRang := qt.NewQCheckBox3("ISRANG (старший/ранг)")
	isRang.SetChecked(initial.IsRang || initial.ID == 0)
	access := qt.NewQCheckBox3("Повний доступ до адмін-функцій (ACCESS1=1)")
	access.SetChecked(initial.Access1 > 0)
	trk := qt.NewQCheckBox3("Перевіряючий ТРК")
	trk.SetChecked(initial.IsTRKTester)

	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("№", number.QWidget)
	form.AddRow3("Створено", createdAt.QWidget)
	form.AddRow3("Прізвище", surname.QWidget)
	form.AddRow3("Ім'я", name.QWidget)
	form.AddRow3("По батькові", secName.QWidget)
	form.AddRow3("Адреса", address.QWidget)
	form.AddRow3("Телефон", phones.QWidget)
	form.AddRow3("Посада", position.QWidget)
	form.AddRow3("Примітка", notes.QWidget)
	form.AddRow3("", isRang.QWidget)
	form.AddRow3("", access.QWidget)
	form.AddRow3("Viber ID", viberID.QWidget)
	form.AddRow3("Telegram ID", telegramID.QWidget)
	form.AddRow3("", trk.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Save | qt.QDialogButtonBox__Cancel)
	buttons.OnAccepted(dialog.Accept)
	buttons.OnRejected(dialog.Reject)

	layout := qt.NewQVBoxLayout(dialog.QWidget)
	layout.AddLayout(form.QLayout)
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	if dialog.Exec() != int(qt.QDialog__Accepted) {
		return initial, false
	}
	updated := initial
	numberRaw := strings.TrimSpace(number.Text())
	if numberRaw != "" {
		parsed, err := strconv.ParseInt(numberRaw, 10, 64)
		if err != nil {
			qt.QMessageBox_Information(parent, "Відповідальна особа", "Некоректний номер В/О")
			return initial, false
		}
		updated.Number = parsed
	} else {
		updated.Number = 0
	}
	updated.Surname = strings.TrimSpace(surname.Text())
	updated.Name = strings.TrimSpace(name.Text())
	updated.SecName = strings.TrimSpace(secName.Text())
	updated.Address = strings.TrimSpace(address.Text())
	updated.Phones = strings.TrimSpace(phones.Text())
	updated.Position = strings.TrimSpace(position.Text())
	updated.Notes = strings.TrimSpace(notes.Text())
	updated.IsRang = isRang.IsChecked()
	updated.Access1 = boolToInt64(access.IsChecked())
	updated.ViberID = strings.TrimSpace(viberID.Text())
	updated.TelegramID = strings.TrimSpace(telegramID.Text())
	updated.IsTRKTester = trk.IsChecked()
	return updated, true
}

func objectPersonalFullName(item contracts.AdminObjectPersonal) string {
	parts := make([]string, 0, 3)
	for _, value := range []string{item.Surname, item.Name, item.SecName} {
		if text := strings.TrimSpace(value); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "(без ПІБ)"
	}
	return strings.Join(parts, " ")
}

func formatInt64NonZero(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}
