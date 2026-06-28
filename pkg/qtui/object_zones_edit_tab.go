//go:build qt

package qtui

import (
	"fmt"
	"strconv"
	"strings"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type objectZonesEditTab struct {
	parent      *qt.QWidget
	provider    contracts.AdminObjectZoneService
	objn        int64
	statusLabel *qt.QLabel
	table       *qt.QTableWidget
	zones       []contracts.AdminObjectZone
	loaded      bool
}

func newObjectZonesEditTab(parent *qt.QWidget, provider contracts.AdminObjectZoneService, objn int64, statusLabel *qt.QLabel) (*qt.QWidget, func()) {
	tab := &objectZonesEditTab{
		parent:      parent,
		provider:    provider,
		objn:        objn,
		statusLabel: statusLabel,
		table:       qt.NewQTableWidget3(0, 3),
	}
	return tab.build(), tab.ensureLoaded
}

func (t *objectZonesEditTab) build() *qt.QWidget {
	t.table.SetHorizontalHeaderLabels([]string{"ZONEN", "Тип", "Опис"})
	t.table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	t.table.OnCellDoubleClicked(func(row int, _ int) {
		t.editZone(row)
	})
	t.table.SetColumnWidth(0, 90)
	t.table.SetColumnWidth(1, 90)
	t.table.SetColumnWidth(2, 420)

	addBtn := qt.NewQPushButton3("Додати")
	editBtn := qt.NewQPushButton3("Змінити")
	deleteBtn := qt.NewQPushButton3("Видалити")
	fillBtn := qt.NewQPushButton3("Заповнити")
	clearBtn := qt.NewQPushButton3("Очистити")
	refreshBtn := qt.NewQPushButton3("Оновити")

	addBtn.OnClicked(t.addZone)
	editBtn.OnClicked(func() { t.editZone(t.table.CurrentRow()) })
	deleteBtn.OnClicked(t.deleteZone)
	fillBtn.OnClicked(t.fillZones)
	clearBtn.OnClicked(t.clearZones)
	refreshBtn.OnClicked(t.reload)

	toolbar := qt.NewQHBoxLayout2()
	toolbar.AddWidget(addBtn.QWidget)
	toolbar.AddWidget(editBtn.QWidget)
	toolbar.AddWidget(deleteBtn.QWidget)
	toolbar.AddWidget(fillBtn.QWidget)
	toolbar.AddWidget(clearBtn.QWidget)
	toolbar.AddStretch()
	toolbar.AddWidget(refreshBtn.QWidget)

	widget := qt.NewQWidget2()
	layout := qt.NewQVBoxLayout(widget)
	layout.AddLayout(toolbar.QLayout)
	layout.AddWidget(t.table.QWidget)
	widget.SetLayout(layout.QLayout)

	return widget
}

func (t *objectZonesEditTab) ensureLoaded() {
	if t.loaded {
		return
	}
	t.reload()
}

func (t *objectZonesEditTab) reload() {
	zones, err := t.provider.ListObjectZones(t.objn)
	if err != nil {
		t.showError("Не вдалося завантажити зони", err)
		return
	}
	t.loaded = true
	t.zones = zones
	t.table.SetRowCount(len(zones))
	for row, zone := range zones {
		zoneNumber := zone.ZoneNumber
		if zoneNumber <= 0 {
			zoneNumber = int64(row + 1)
		}
		t.table.SetItem(row, 0, qt.NewQTableWidgetItem2(strconv.FormatInt(zoneNumber, 10)))
		t.table.SetItem(row, 1, qt.NewQTableWidgetItem2("пож."))
		t.table.SetItem(row, 2, qt.NewQTableWidgetItem2(strings.TrimSpace(zone.Description)))
	}
	t.table.SetColumnWidth(0, 90)
	t.table.SetColumnWidth(1, 90)
	t.table.SetColumnWidth(2, 420)
	t.setStatus(fmt.Sprintf("Зон: %d", len(zones)))
}

func (t *objectZonesEditTab) addZone() {
	zoneNumber := int64(len(t.zones) + 1)
	if len(t.zones) > 0 {
		last := t.zones[len(t.zones)-1].ZoneNumber
		if last >= zoneNumber {
			zoneNumber = last + 1
		}
	}
	zone, ok := showObjectZoneEditDialog(t.parent, contracts.AdminObjectZone{ZoneNumber: zoneNumber, ZoneType: 1})
	if !ok {
		return
	}
	if err := t.provider.AddObjectZone(t.objn, zone); err != nil {
		t.showError("Не вдалося додати зону", err)
		return
	}
	t.reload()
	t.setStatus(fmt.Sprintf("Додано зону #%d", zone.ZoneNumber))
}

func (t *objectZonesEditTab) editZone(row int) {
	zone, ok := t.zoneAt(row)
	if !ok {
		t.setStatus("Виберіть зону у таблиці")
		return
	}
	updated, ok := showObjectZoneEditDialog(t.parent, zone)
	if !ok {
		return
	}
	updated.ID = zone.ID
	if err := t.provider.UpdateObjectZone(t.objn, updated); err != nil {
		t.showError("Не вдалося оновити зону", err)
		return
	}
	t.reload()
	t.setStatus(fmt.Sprintf("Оновлено зону #%d", updated.ZoneNumber))
}

func (t *objectZonesEditTab) deleteZone() {
	zone, ok := t.zoneAt(t.table.CurrentRow())
	if !ok {
		t.setStatus("Виберіть зону у таблиці")
		return
	}
	if qt.QMessageBox_Question(t.parent, "Підтвердження", fmt.Sprintf("Видалити зону #%d?", zone.ZoneNumber)) != qt.QMessageBox__Yes {
		return
	}
	if err := t.provider.DeleteObjectZone(t.objn, zone.ID); err != nil {
		t.showError("Не вдалося видалити зону", err)
		return
	}
	t.reload()
	t.setStatus("Зону видалено")
}

func (t *objectZonesEditTab) fillZones() {
	count := qt.QInputDialog_GetInt(t.parent, "Заповнити зони", "Кількість зон:")
	if count <= 0 {
		return
	}
	if err := t.provider.FillObjectZones(t.objn, int64(count)); err != nil {
		t.showError("Не вдалося заповнити зони", err)
		return
	}
	t.reload()
	t.setStatus("Зони заповнено")
}

func (t *objectZonesEditTab) clearZones() {
	if qt.QMessageBox_Question(t.parent, "Підтвердження", "Видалити всі зони об'єкта?") != qt.QMessageBox__Yes {
		return
	}
	if err := t.provider.ClearObjectZones(t.objn); err != nil {
		t.showError("Не вдалося очистити зони", err)
		return
	}
	t.reload()
	t.setStatus("Зони очищено")
}

func (t *objectZonesEditTab) zoneAt(row int) (contracts.AdminObjectZone, bool) {
	if row < 0 || row >= len(t.zones) {
		return contracts.AdminObjectZone{}, false
	}
	return t.zones[row], true
}

func (t *objectZonesEditTab) setStatus(text string) {
	if t.statusLabel != nil {
		t.statusLabel.SetText(strings.TrimSpace(text))
	}
}

func (t *objectZonesEditTab) showError(prefix string, err error) {
	message := strings.TrimSpace(prefix)
	if err != nil {
		message += ": " + err.Error()
	}
	t.setStatus(message)
	qt.QMessageBox_Critical(t.parent, prefix, message)
}

func showObjectZoneEditDialog(parent *qt.QWidget, initial contracts.AdminObjectZone) (contracts.AdminObjectZone, bool) {
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Зона об'єкта")
	dialog.Resize(460, 220)

	number := newSpinBox(int(initial.ZoneNumber), 1, 9999)
	description := newLineEdit(initial.Description)

	form := qt.NewQFormLayout2()
	form.SetFieldGrowthPolicy(qt.QFormLayout__AllNonFixedFieldsGrow)
	form.AddRow3("Номер", number.QWidget)
	form.AddRow3("Опис", description.QWidget)

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
	updated.ZoneNumber = int64(number.Value())
	if updated.ZoneType <= 0 {
		updated.ZoneType = 1
	}
	updated.Description = strings.TrimSpace(description.Text())
	return updated, true
}
