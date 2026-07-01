package dialogs

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/objectreport"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
)

// ShowNewObjectsReport opens a compact cross-source new objects report.
func ShowNewObjectsReport(
	provider contracts.ObjectProvider,
	isDark bool,
	onOpen func(models.Object),
) {
	reportWindow := fyne.CurrentApp().NewWindow("Нові об'єкти")
	reportWindow.Resize(fyne.NewSize(980, 620))

	period := widget.NewSelect(objectreport.PeriodOptions(), nil)
	period.SetSelected(objectreport.PeriodMonth)
	fromEntry := widget.NewEntry()
	toEntry := widget.NewEntry()
	status := widget.NewLabel("Завантаження...")
	status.Wrapping = fyne.TextWrapWord
	search := widget.NewEntry()
	search.SetPlaceHolder("Пошук за номером, назвою або адресою")

	var allItems []objectreport.Item
	var visibleItems []objectreport.Item
	vm := viewmodels.NewObjectListViewModel()

	list := widget.NewList(
		func() int { return len(visibleItems) },
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			text := canvas.NewText("", color.White)
			return container.NewStack(bg, container.NewPadded(text))
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(visibleItems) {
				return
			}
			stack := item.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			text := stack.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text)
			entry := visibleItems[id]
			textColor, rowColor := vm.GetRowColors(entry.Object, isDark)
			bg.FillColor = rowColor
			bg.Refresh()
			text.Color = textColor
			text.Text = newObjectReportLine(entry)
			text.Refresh()
		},
	)

	applySearch := func() {
		query := strings.ToLower(strings.TrimSpace(search.Text))
		visibleItems = visibleItems[:0]
		for _, item := range allItems {
			line := strings.ToLower(newObjectReportLine(item))
			if query == "" || strings.Contains(line, query) {
				visibleItems = append(visibleItems, item)
			}
		}
		list.Refresh()
		status.SetText(fmt.Sprintf("Знайдено об'єктів: %d", len(visibleItems)))
	}
	search.OnChanged = func(string) { applySearch() }
	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(visibleItems) || onOpen == nil {
			return
		}
		object := visibleItems[id].Object
		reportWindow.Close()
		onOpen(object)
	}

	setPresetRange := func() {
		if period.Selected == objectreport.PeriodCustom {
			return
		}
		from, to := objectreport.RangeForPeriod(period.Selected, time.Now())
		fromEntry.SetText(from.Format("2006-01-02"))
		toEntry.SetText(to.Format("2006-01-02"))
	}
	setPresetRange()

	reload := func() {
		from, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(fromEntry.Text), time.Local)
		if err != nil {
			status.SetText("Некоректна дата «від». Формат: РРРР-ММ-ДД")
			return
		}
		to, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(toEntry.Text), time.Local)
		if err != nil || to.Before(from) {
			status.SetText("Некоректна дата «до» або діапазон дат")
			return
		}
		status.SetText("Завантаження об'єктів...")
		go func() {
			objects := provider.GetObjects()
			items := objectreport.Filter(objects, from, to)
			fyne.Do(func() {
				allItems = items
				applySearch()
			})
		}()
	}
	period.OnChanged = func(string) {
		setPresetRange()
		if period.Selected != objectreport.PeriodCustom {
			reload()
		}
	}

	controls := container.NewGridWithColumns(7,
		widget.NewLabel("Період"), period,
		widget.NewLabel("Від"), fromEntry,
		widget.NewLabel("До"), toEntry,
		widget.NewButton("Показати", reload),
	)
	reportWindow.SetContent(container.NewBorder(
		container.NewVBox(controls, search, status),
		nil, nil, nil,
		list,
	))
	reportWindow.Show()
	reload()
}

func newObjectReportLine(item objectreport.Item) string {
	return fmt.Sprintf(
		"%s   %-8s   №%s   %s   |   %s   |   %s",
		item.AddedAt.Format("02.01.2006"),
		viewmodels.ObjectSourceByID(item.Object.ID),
		viewmodels.ObjectDisplayNumber(item.Object),
		strings.TrimSpace(item.Object.Name),
		strings.TrimSpace(item.Object.Address),
		item.Object.GetStatusDisplay(),
	)
}
