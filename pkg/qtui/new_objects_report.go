//go:build qt

package qtui

import (
	"fmt"
	"strings"
	"time"

	qt "github.com/mappu/miqt/qt6"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/objectreport"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

// ShowNewObjectsReport opens the cross-source new objects report.
func ShowNewObjectsReport(
	parent *qt.QWidget,
	provider contracts.ObjectProvider,
	onOpen func(models.Object),
) {
	if provider == nil {
		return
	}
	dialog := qt.NewQDialog(parent)
	dialog.SetWindowTitle("Нові об'єкти")
	dialog.Resize(1000, 640)
	layout := qt.NewQVBoxLayout(dialog.QWidget)

	controls := qt.NewQHBoxLayout2()
	period := qt.NewQComboBox2()
	period.AddItems(objectreport.PeriodOptions())
	period.SetCurrentText(objectreport.PeriodMonth)
	fromEntry := lineEdit()
	toEntry := lineEdit()
	showButton := qt.NewQPushButton3("Показати")
	controls.AddWidget(qt.NewQLabel3("Період").QWidget)
	controls.AddWidget(period.QWidget)
	controls.AddWidget(qt.NewQLabel3("Від").QWidget)
	controls.AddWidget(fromEntry.QWidget)
	controls.AddWidget(qt.NewQLabel3("До").QWidget)
	controls.AddWidget(toEntry.QWidget)
	controls.AddWidget(showButton.QWidget)
	layout.AddLayout(controls.QLayout)

	search := lineEdit()
	search.SetPlaceholderText("Пошук за номером, назвою або адресою")
	layout.AddWidget(search.QWidget)
	status := qt.NewQLabel3("")
	layout.AddWidget(status.QWidget)

	table := qt.NewQTableView2()
	table.SetSelectionBehavior(qt.QAbstractItemView__SelectRows)
	table.SetSelectionMode(qt.QAbstractItemView__SingleSelection)
	table.SetEditTriggers(qt.QAbstractItemView__NoEditTriggers)
	table.SetWordWrap(false)
	model := qt.NewQStandardItemModel2(0, 6)
	model.SetHorizontalHeaderLabels([]string{"Додано", "Джерело", "№", "Назва", "Адреса", "Стан"})
	table.SetModel(model.QAbstractItemModel)
	table.SetSortingEnabled(false)
	layout.AddWidget(table.QWidget)

	buttons := qt.NewQDialogButtonBox4(qt.QDialogButtonBox__Close)
	openButton := buttons.AddButton2("Відкрити картку", qt.QDialogButtonBox__ActionRole)
	buttons.OnRejected(func() { dialog.Reject() })
	layout.AddWidget(buttons.QWidget)
	dialog.SetLayout(layout.QLayout)

	vm := viewmodels.NewObjectListViewModel()
	var allItems []objectreport.Item
	var visibleItems []objectreport.Item

	render := func() {
		query := strings.ToLower(strings.TrimSpace(search.Text()))
		visibleItems = visibleItems[:0]
		for _, item := range allItems {
			line := strings.ToLower(newObjectsQtSearchText(item))
			if query == "" || strings.Contains(line, query) {
				visibleItems = append(visibleItems, item)
			}
		}
		model.Clear()
		model.SetHorizontalHeaderLabels([]string{"Додано", "Джерело", "№", "Назва", "Адреса", "Стан"})
		for _, item := range visibleItems {
			object := item.Object
			textColor, rowColor := vm.GetRowColors(object, false)
			addColoredReadOnlyRow(model, []string{
				item.AddedAt.Format("02.01.2006"),
				viewmodels.ObjectSourceByID(object.ID),
				viewmodels.ObjectDisplayNumber(object),
				strings.TrimSpace(object.Name),
				strings.TrimSpace(object.Address),
				object.GetStatusDisplay(),
			}, object.ID, textColor, rowColor)
		}
		table.ResizeColumnsToContents()
		table.HorizontalHeader().SetStretchLastSection(true)
		status.SetText(fmt.Sprintf("Знайдено об'єктів: %d", len(visibleItems)))
	}

	setPresetRange := func() {
		if period.CurrentText() == objectreport.PeriodCustom {
			return
		}
		from, to := objectreport.RangeForPeriod(period.CurrentText(), time.Now())
		fromEntry.SetText(from.Format("2006-01-02"))
		toEntry.SetText(to.Format("2006-01-02"))
	}
	reload := func() {
		from, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(fromEntry.Text()), time.Local)
		if err != nil {
			status.SetText("Некоректна дата «від». Формат: РРРР-ММ-ДД")
			return
		}
		to, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(toEntry.Text()), time.Local)
		if err != nil || to.Before(from) {
			status.SetText("Некоректна дата «до» або діапазон дат")
			return
		}
		status.SetText("Завантаження об'єктів...")
		allItems = objectreport.Filter(provider.GetObjects(), from, to)
		render()
	}
	openSelected := func() {
		index := table.CurrentIndex()
		if index == nil || !index.IsValid() || index.Row() < 0 || index.Row() >= len(visibleItems) || onOpen == nil {
			return
		}
		object := visibleItems[index.Row()].Object
		dialog.Accept()
		onOpen(object)
	}
	openButton.OnClicked(openSelected)
	table.OnClicked(func(*qt.QModelIndex) { openSelected() })
	search.OnTextChanged(func(string) { render() })
	period.OnCurrentTextChanged(func(string) {
		setPresetRange()
		if period.CurrentText() != objectreport.PeriodCustom {
			reload()
		}
	})
	showButton.OnClicked(reload)

	setPresetRange()
	reload()
	dialog.Exec()
}

func newObjectsQtSearchText(item objectreport.Item) string {
	return strings.Join([]string{
		viewmodels.ObjectDisplayNumber(item.Object),
		item.Object.Name,
		item.Object.Address,
		item.Object.GetStatusDisplay(),
		viewmodels.ObjectSourceByID(item.Object.ID),
	}, " ")
}
