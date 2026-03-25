package dialogs

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/data"
)

func ShowPPKConstructorDialog(parent fyne.Window, provider data.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Конструктор ППК")
	win.Resize(fyne.NewSize(900, 560))

	var (
		items       []data.PPKConstructorItem
		selectedRow = -1
		selectedID  int64
		mode        = "view" // view | add | edit
	)

	statusLabel := widget.NewLabel("Готово")

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Назва типу ППК")
	channelEntry := widget.NewEntry()
	channelEntry.SetPlaceHolder("Код каналу (наприклад, 0, 1, 7)")
	zoneCountEntry := widget.NewEntry()
	zoneCountEntry.SetPlaceHolder("Кількість ШС")

	setMode := func(next string) {
		mode = next
		editable := mode == "add" || mode == "edit"
		if editable {
			nameEntry.Enable()
			channelEntry.Enable()
			zoneCountEntry.Enable()
			return
		}
		nameEntry.Disable()
		channelEntry.Disable()
		zoneCountEntry.Disable()
	}

	fillEditor := func(item data.PPKConstructorItem) {
		nameEntry.SetText(strings.TrimSpace(item.Name))
		channelEntry.SetText(strconv.FormatInt(item.Channel, 10))
		zoneCountEntry.SetText(strconv.FormatInt(item.ZoneCount, 10))
	}

	clearEditor := func() {
		nameEntry.SetText("")
		channelEntry.SetText("0")
		zoneCountEntry.SetText("4")
	}

	table := widget.NewTable(
		func() (int, int) { return len(items), 4 },
		func() fyne.CanvasObject { return widget.NewLabel("cell") },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row < 0 || id.Row >= len(items) {
				lbl.SetText("")
				return
			}
			it := items[id.Row]
			switch id.Col {
			case 0:
				lbl.SetText(strconv.FormatInt(it.ID, 10))
			case 1:
				lbl.SetText(strings.TrimSpace(it.Name))
			case 2:
				lbl.SetText(strconv.FormatInt(it.Channel, 10))
			default:
				lbl.SetText(strconv.FormatInt(it.ZoneCount, 10))
			}
		},
	)
	table.SetColumnWidth(0, 80)
	table.SetColumnWidth(1, 500)
	table.SetColumnWidth(2, 120)
	table.SetColumnWidth(3, 140)

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(items) {
			return
		}
		selectedRow = id.Row
		selectedID = items[id.Row].ID
		fillEditor(items[id.Row])
		if mode != "add" {
			setMode("view")
		}
		statusLabel.SetText(fmt.Sprintf("Вибрано ППК ID=%d", selectedID))
	}

	reload := func(reselectID int64) {
		loaded, err := provider.ListPPKConstructor()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити довідник ППК")
			return
		}

		items = loaded
		table.Refresh()
		table.UnselectAll()
		selectedRow = -1
		selectedID = 0

		if reselectID > 0 {
			for i := range items {
				if items[i].ID == reselectID {
					table.Select(widget.TableCellID{Row: i, Col: 0})
					return
				}
			}
		}
		if len(items) > 0 {
			table.Select(widget.TableCellID{Row: 0, Col: 0})
		} else {
			clearEditor()
			statusLabel.SetText("Довідник ППК порожній")
		}
	}

	parseEditor := func() (name string, channel int64, zoneCount int64, err error) {
		name = strings.TrimSpace(nameEntry.Text)
		if name == "" {
			return "", 0, 0, fmt.Errorf("назва ППК не може бути порожньою")
		}

		channelRaw := strings.TrimSpace(channelEntry.Text)
		if channelRaw == "" {
			channelRaw = "0"
		}
		channel, err = strconv.ParseInt(channelRaw, 10, 64)
		if err != nil || channel < 0 {
			return "", 0, 0, fmt.Errorf("некоректний код каналу")
		}

		zoneRaw := strings.TrimSpace(zoneCountEntry.Text)
		zoneCount, err = strconv.ParseInt(zoneRaw, 10, 64)
		if err != nil || zoneCount <= 0 {
			return "", 0, 0, fmt.Errorf("некоректна кількість ШС")
		}

		return name, channel, zoneCount, nil
	}

	addBtn := widget.NewButton("Додати", func() {
		selectedRow = -1
		selectedID = 0
		clearEditor()
		setMode("add")
		win.Canvas().Focus(nameEntry)
		statusLabel.SetText("Режим додавання")
	})

	editBtn := widget.NewButton("Змінити", func() {
		if selectedID <= 0 {
			statusLabel.SetText("Спочатку виберіть рядок у таблиці")
			return
		}
		setMode("edit")
		statusLabel.SetText("Режим редагування")
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		if selectedID <= 0 {
			statusLabel.SetText("Спочатку виберіть рядок у таблиці")
			return
		}
		targetID := selectedID
		targetName := strings.TrimSpace(nameEntry.Text)
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити ППК \"%s\"?", targetName),
			func(ok bool) {
				if !ok {
					return
				}
				if err := provider.DeletePPKConstructor(targetID); err != nil {
					dialog.ShowError(err, win)
					statusLabel.SetText("Не вдалося видалити ППК")
					return
				}
				statusLabel.SetText("ППК видалено")
				reload(0)
			},
			win,
		)
	})

	applyBtn := widget.NewButton("Застосувати", func() {
		name, channel, zoneCount, err := parseEditor()
		if err != nil {
			statusLabel.SetText(err.Error())
			return
		}

		switch mode {
		case "add":
			if err := provider.AddPPKConstructor(name, channel, zoneCount); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося додати ППК")
				return
			}
			setMode("view")
			statusLabel.SetText("ППК додано")
			reload(0)
		case "edit":
			if selectedID <= 0 {
				statusLabel.SetText("Немає вибраного рядка")
				return
			}
			if err := provider.UpdatePPKConstructor(selectedID, name, channel, zoneCount); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося зберегти зміни")
				return
			}
			setMode("view")
			statusLabel.SetText("Зміни збережено")
			reload(selectedID)
		default:
			statusLabel.SetText("Оберіть Додати або Змінити")
		}
	})

	cancelBtn := widget.NewButton("Відміна", func() {
		setMode("view")
		if selectedRow >= 0 && selectedRow < len(items) {
			fillEditor(items[selectedRow])
		} else {
			clearEditor()
		}
		statusLabel.SetText("Редагування скасовано")
	})

	refreshBtn := widget.NewButton("Оновити", func() { reload(selectedID) })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	headers := container.NewGridWithColumns(
		4,
		widget.NewLabel("ID"),
		widget.NewLabel("Назва ППК"),
		widget.NewLabel("Канал"),
		widget.NewLabel("ШС"),
	)

	editor := widget.NewForm(
		widget.NewFormItem("Назва:", nameEntry),
		widget.NewFormItem("Код каналу:", channelEntry),
		widget.NewFormItem("Кількість ШС:", zoneCountEntry),
	)

	content := container.NewBorder(
		container.NewVBox(
			container.NewHBox(addBtn, editBtn, deleteBtn, layout.NewSpacer(), refreshBtn),
			widget.NewSeparator(),
			headers,
		),
		container.NewVBox(
			widget.NewSeparator(),
			container.NewBorder(nil, nil, nil, container.NewHBox(applyBtn, cancelBtn), editor),
			widget.NewSeparator(),
			container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		),
		nil, nil,
		table,
	)

	win.SetContent(content)
	setMode("view")
	reload(0)
	win.Show()
}
