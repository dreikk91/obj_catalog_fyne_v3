package dialogs

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
	"obj_catalog_fyne_v3/pkg/utils"
)

func adminDisplayBlockObjectColors(item adminv1.DisplayBlockObject, isDark bool) (color.NRGBA, color.NRGBA) {
	selectObjectColor := utils.SelectObjectColorNRGBA
	if isDark {
		selectObjectColor = utils.SelectObjectColorNRGBADark
	}

	switch item.MonitoringStatus {
	case frontendv1.MonitoringStatusBlocked:
		if isDark {
			return color.NRGBA{R: 230, G: 220, B: 245, A: 255}, color.NRGBA{R: 98, G: 52, B: 125, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 144, G: 64, B: 196, A: 255}
	case frontendv1.MonitoringStatusDebug:
		if isDark {
			return color.NRGBA{R: 238, G: 236, B: 195, A: 255}, color.NRGBA{R: 95, G: 96, B: 42, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 128, G: 128, B: 0, A: 255}
	}

	if item.VisualSeverity == frontendv1.VisualSeverityCritical {
		return selectObjectColor(1)
	}
	if item.TechAlarmState > 0 {
		return selectObjectColor(2)
	}
	if item.ConnectionStatus == frontendv1.ConnectionStatusOffline {
		if isDark {
			return color.NRGBA{R: 255, G: 250, B: 180, A: 255}, color.NRGBA{R: 90, G: 90, B: 20, A: 255}
		}
		return color.NRGBA{R: 0, G: 0, B: 0, A: 255}, color.NRGBA{R: 225, G: 235, B: 35, A: 255}
	}
	if item.GuardStatus == frontendv1.GuardStatusDisarmed {
		if isDark {
			return color.NRGBA{R: 230, G: 230, B: 250, A: 255}, color.NRGBA{R: 100, G: 15, B: 120, A: 255}
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}, color.NRGBA{R: 170, G: 14, B: 201, A: 255}
	}

	return selectObjectColor(10)
}

func ShowDisplayBlockingDialog(parent fyne.Window, provider adminv1.DisplayBlockingProvider, onUpdated func()) {
	win := fyne.CurrentApp().NewWindow("Блокування відображення інформації")
	win.Resize(fyne.NewSize(1020, 620))

	var (
		objects     []adminv1.DisplayBlockObject
		selectedRow = -1
	)

	blockModeText := func(mode adminv1.DisplayBlockMode) string {
		switch mode {
		case adminv1.DisplayBlockModeTemporaryOff:
			return "Тимчасово зняти із спостереження"
		case adminv1.DisplayBlockModeDebug:
			return "Ввести об'єкт в режим налагодження"
		default:
			return "Немає"
		}
	}

	parseBlockMode := func(label string) adminv1.DisplayBlockMode {
		switch label {
		case "Тимчасово зняти із спостереження":
			return adminv1.DisplayBlockModeTemporaryOff
		case "Ввести об'єкт в режим налагодження":
			return adminv1.DisplayBlockModeDebug
		default:
			return adminv1.DisplayBlockModeNone
		}
	}

	statusLabel := widget.NewLabel("Готово")
	objectNumberEntry := widget.NewEntry()
	objectNumberEntry.SetPlaceHolder("№ об'єкта")
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Фільтр (№ або назва)")

	modeRadio := widget.NewRadioGroup(
		[]string{
			"Немає",
			"Тимчасово зняти із спостереження",
			"Ввести об'єкт в режим налагодження",
		},
		nil,
	)
	modeRadio.Horizontal = true
	modeRadio.SetSelected("Немає")

	isDarkMode := func() bool {
		return fyne.CurrentApp().Settings().ThemeVariant() == theme.VariantDark
	}

	calcObjectColors := func(item adminv1.DisplayBlockObject) (color.NRGBA, color.NRGBA) {
		return adminDisplayBlockObjectColors(item, isDarkMode())
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(objects), 3
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("", color.Black)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtWrap := stack.Objects[1].(*fyne.Container)
			txt := txtWrap.Objects[0].(*canvas.Text)

			if id.Row < 0 || id.Row >= len(objects) {
				txt.Text = ""
				txt.Refresh()
				bg.FillColor = color.Transparent
				bg.Refresh()
				return
			}

			item := objects[id.Row]
			textColor, rowColor := calcObjectColors(item)

			if id.Row == selectedRow {
				bg.FillColor = color.NRGBA{R: 33, G: 112, B: 214, A: 255}
				txt.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			} else {
				bg.FillColor = rowColor
				txt.Color = textColor
			}
			bg.Refresh()

			switch id.Col {
			case 0:
				txt.Text = strconv.FormatInt(item.ObjN, 10)
			case 1:
				txt.Text = item.Name
			default:
				txt.Text = blockModeText(item.BlockMode)
			}
			txt.Refresh()
		},
	)
	table.SetColumnWidth(0, 100)
	table.SetColumnWidth(1, 530)
	table.SetColumnWidth(2, 340)

	table.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(objects) {
			return
		}
		selectedRow = id.Row
		object := objects[id.Row]
		objectNumberEntry.SetText(strconv.FormatInt(object.ObjN, 10))
		modeRadio.SetSelected(blockModeText(object.BlockMode))
		statusLabel.SetText(fmt.Sprintf("Вибрано об'єкт №%d", object.ObjN))
		table.Refresh()
	}

	reload := func(selectObjN int64) {
		loaded, err := provider.ListDisplayBlockObjects(strings.TrimSpace(filterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження об'єктів")
			return
		}

		objects = loaded
		selectedRow = -1
		table.Refresh()
		table.UnselectAll()

		if selectObjN != 0 {
			for i := range objects {
				if objects[i].ObjN == selectObjN {
					table.Select(widget.TableCellID{Row: i, Col: 0})
					return
				}
			}
		}
		if len(objects) > 0 {
			table.Select(widget.TableCellID{Row: 0, Col: 0})
		} else {
			statusLabel.SetText("Об'єкти не знайдено")
		}
	}

	setBtn := widget.NewButton("Встановити", func() {
		rawObjN := strings.TrimSpace(objectNumberEntry.Text)
		if rawObjN == "" {
			statusLabel.SetText("Вкажіть № об'єкта")
			return
		}
		objN, err := strconv.ParseInt(rawObjN, 10, 64)
		if err != nil {
			statusLabel.SetText("Некоректний № об'єкта")
			return
		}

		mode := parseBlockMode(modeRadio.Selected)
		if err := provider.SetDisplayBlockMode(objN, mode); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося змінити режим блокування")
			return
		}

		statusLabel.SetText(fmt.Sprintf("Оновлено блокування для об'єкта №%d", objN))
		reload(objN)
		if onUpdated != nil {
			onUpdated()
		}
	})

	refreshBtn := widget.NewButton("Оновити", func() { reload(0) })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })
	filterEntry.OnSubmitted = func(_ string) { reload(0) }

	headers := container.NewGridWithColumns(
		3,
		widget.NewLabel("№пр."),
		widget.NewLabel("Об'єкт"),
		widget.NewLabel("Блокування"),
	)

	top := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("№пр.:"),
			container.NewGridWrap(fyne.NewSize(120, 36), objectNumberEntry),
			widget.NewLabel("Режим:"),
			modeRadio,
			setBtn,
		),
		container.NewBorder(
			nil, nil,
			widget.NewLabel("Фільтр:"),
			refreshBtn,
			container.NewGridWrap(fyne.NewSize(640, 36), filterEntry),
		),
		widget.NewSeparator(),
	)

	content := container.NewBorder(
		top,
		container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		nil, nil,
		container.NewBorder(headers, nil, nil, nil, table),
	)

	win.SetContent(content)
	reload(0)
	win.Show()
}
