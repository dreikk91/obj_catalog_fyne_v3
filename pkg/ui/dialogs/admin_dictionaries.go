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

	data "obj_catalog_fyne_v3/pkg/contracts"
)

type dictionaryDialogConfig struct {
	Title       string
	NameLabel   string
	ShowCode    bool
	CodeLabel   string
	SupportMove bool

	List   func() ([]data.DictionaryItem, error)
	Add    func(name string, code *int64) error
	Update func(id int64, name string, code *int64) error
	Delete func(id int64) error
	Move   func(id int64, direction int) error
}

func ShowObjectTypesDictionaryDialog(parent fyne.Window, provider data.AdminProvider) {
	showDictionaryDialog(parent, dictionaryDialogConfig{
		Title:     "Типи об'єктів",
		NameLabel: "Тип об'єкта:",
		List:      provider.ListObjectTypes,
		Add:       func(name string, _ *int64) error { return provider.AddObjectType(name) },
		Update:    func(id int64, name string, _ *int64) error { return provider.UpdateObjectType(id, name) },
		Delete:    provider.DeleteObjectType,
	})
}

func ShowRegionsDictionaryDialog(parent fyne.Window, provider data.AdminProvider) {
	showDictionaryDialog(parent, dictionaryDialogConfig{
		Title:     "Регіони",
		NameLabel: "Назва регіону:",
		ShowCode:  true,
		CodeLabel: "Ідентифікатор регіону:",
		List:      provider.ListRegions,
		Add:       provider.AddRegion,
		Update:    provider.UpdateRegion,
		Delete:    provider.DeleteRegion,
	})
}

func ShowAlarmReasonsDictionaryDialog(parent fyne.Window, provider data.AdminProvider) {
	showDictionaryDialog(parent, dictionaryDialogConfig{
		Title:       "Причини тривог",
		NameLabel:   "Причина тривоги:",
		SupportMove: true,
		List:        provider.ListAlarmReasons,
		Add:         func(name string, _ *int64) error { return provider.AddAlarmReason(name) },
		Update:      func(id int64, name string, _ *int64) error { return provider.UpdateAlarmReason(id, name) },
		Delete:      provider.DeleteAlarmReason,
		Move:        provider.MoveAlarmReason,
	})
}

func showDictionaryDialog(parent fyne.Window, cfg dictionaryDialogConfig) {
	win := fyne.CurrentApp().NewWindow(cfg.Title)
	win.Resize(fyne.NewSize(780, 520))

	var (
		items         []data.DictionaryItem
		selectedIndex = -1
		selectedID    int64
		mode          = "view" // view | add | edit
	)

	nameEntry := widget.NewEntry()
	codeEntry := widget.NewEntry()
	if cfg.ShowCode {
		codeEntry.SetPlaceHolder("необов'язково")
	}
	statusLabel := widget.NewLabel("Готово")

	setMode := func(next string) {
		mode = next
		editing := mode == "add" || mode == "edit"
		nameEntry.Enable()
		if !editing {
			nameEntry.Disable()
		}
		if cfg.ShowCode {
			codeEntry.Enable()
			if !editing {
				codeEntry.Disable()
			}
		}
	}

	formatItem := func(item data.DictionaryItem) string {
		if cfg.ShowCode && item.Code != nil {
			return fmt.Sprintf("%s [%d]", item.Name, *item.Code)
		}
		return item.Name
	}

	list := widget.NewList(
		func() int { return len(items) },
		func() fyne.CanvasObject {
			return widget.NewLabel("item")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(items) {
				label.SetText("")
				return
			}
			label.SetText(formatItem(items[id]))
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(items) {
			return
		}
		selectedIndex = id
		selectedID = items[id].ID
		nameEntry.SetText(items[id].Name)
		if cfg.ShowCode {
			if items[id].Code != nil {
				codeEntry.SetText(strconv.FormatInt(*items[id].Code, 10))
			} else {
				codeEntry.SetText("")
			}
		}
		statusLabel.SetText(fmt.Sprintf("Вибрано: %s", formatItem(items[id])))
		if mode != "add" {
			setMode("view")
		}
	}

	reload := func(reselectID int64) {
		loaded, err := cfg.List()
		if err != nil {
			statusLabel.SetText("Помилка завантаження")
			dialog.ShowError(err, win)
			return
		}
		items = loaded
		list.Refresh()

		selectedIndex = -1
		selectedID = 0
		if reselectID != 0 {
			for i := range items {
				if items[i].ID == reselectID {
					selectedIndex = i
					selectedID = reselectID
					list.Select(i)
					return
				}
			}
		}
		if len(items) > 0 {
			list.Select(0)
		} else {
			nameEntry.SetText("")
			codeEntry.SetText("")
			statusLabel.SetText("Довідник порожній")
		}
	}

	clearEditor := func() {
		nameEntry.SetText("")
		codeEntry.SetText("")
	}

	addBtn := widget.NewButton("Додати", func() {
		selectedIndex = -1
		selectedID = 0
		clearEditor()
		setMode("add")
		win.Canvas().Focus(nameEntry)
		statusLabel.SetText("Режим додавання")
	})

	editBtn := widget.NewButton("Змінити", func() {
		if selectedID == 0 {
			statusLabel.SetText("Спочатку виберіть запис")
			return
		}
		setMode("edit")
		statusLabel.SetText("Режим редагування")
	})

	deleteBtn := widget.NewButton("Видалити", func() {
		if selectedID == 0 {
			statusLabel.SetText("Спочатку виберіть запис")
			return
		}
		target := selectedID
		targetName := strings.TrimSpace(nameEntry.Text)
		dialog.ShowConfirm(
			"Підтвердження",
			fmt.Sprintf("Видалити запис \"%s\"?", targetName),
			func(ok bool) {
				if !ok {
					return
				}
				if err := cfg.Delete(target); err != nil {
					dialog.ShowError(err, win)
					statusLabel.SetText("Не вдалося видалити")
					return
				}
				statusLabel.SetText("Запис видалено")
				reload(0)
			},
			win,
		)
	})

	applyBtn := widget.NewButton("Застосувати", func() {
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			statusLabel.SetText("Поле назви не може бути порожнім")
			return
		}

		var code *int64
		if cfg.ShowCode {
			raw := strings.TrimSpace(codeEntry.Text)
			if raw != "" {
				val, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					statusLabel.SetText("Некоректний ідентифікатор регіону")
					return
				}
				code = &val
			}
		}

		switch mode {
		case "add":
			if err := cfg.Add(name, code); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося додати")
				return
			}
			statusLabel.SetText("Додано")
		case "edit":
			if selectedID == 0 {
				statusLabel.SetText("Немає вибраного запису")
				return
			}
			if err := cfg.Update(selectedID, name, code); err != nil {
				dialog.ShowError(err, win)
				statusLabel.SetText("Не вдалося зберегти зміни")
				return
			}
			statusLabel.SetText("Зміни збережено")
		default:
			statusLabel.SetText("Оберіть режим Додати або Змінити")
			return
		}

		setMode("view")
		reload(selectedID)
	})

	cancelBtn := widget.NewButton("Відміна", func() {
		setMode("view")
		if selectedIndex >= 0 && selectedIndex < len(items) {
			nameEntry.SetText(items[selectedIndex].Name)
			if cfg.ShowCode {
				if items[selectedIndex].Code != nil {
					codeEntry.SetText(strconv.FormatInt(*items[selectedIndex].Code, 10))
				} else {
					codeEntry.SetText("")
				}
			}
		} else {
			clearEditor()
		}
		statusLabel.SetText("Зміни скасовано")
	})

	moveUpBtn := widget.NewButton("Підвищити", func() {
		if !cfg.SupportMove || cfg.Move == nil {
			return
		}
		if selectedID == 0 {
			statusLabel.SetText("Спочатку виберіть запис")
			return
		}
		if err := cfg.Move(selectedID, -1); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося підвищити запис")
			return
		}
		reload(selectedID)
		statusLabel.SetText("Запис переміщено вище")
	})

	moveDownBtn := widget.NewButton("Понизити", func() {
		if !cfg.SupportMove || cfg.Move == nil {
			return
		}
		if selectedID == 0 {
			statusLabel.SetText("Спочатку виберіть запис")
			return
		}
		if err := cfg.Move(selectedID, +1); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося понизити запис")
			return
		}
		reload(selectedID)
		statusLabel.SetText("Запис переміщено нижче")
	})

	closeBtn := widget.NewButton("Закрити", func() {
		win.Close()
	})

	controls := []fyne.CanvasObject{
		addBtn,
		editBtn,
		deleteBtn,
	}
	if cfg.SupportMove {
		controls = append(controls, widget.NewSeparator(), moveUpBtn, moveDownBtn)
	}
	controls = append(controls, layout.NewSpacer(), closeBtn)
	rightPanel := container.NewVBox(controls...)

	formItems := []*widget.FormItem{
		widget.NewFormItem(cfg.NameLabel, nameEntry),
	}
	if cfg.ShowCode {
		formItems = append(formItems, widget.NewFormItem(cfg.CodeLabel, codeEntry))
	}
	editorForm := widget.NewForm(formItems...)

	listFrame := container.NewBorder(
		widget.NewLabel("Список:"),
		nil, nil, nil,
		list,
	)

	main := container.NewHSplit(listFrame, rightPanel)
	main.SetOffset(0.78)

	bottom := container.NewBorder(
		nil, nil, nil,
		container.NewHBox(applyBtn, cancelBtn),
		editorForm,
	)

	content := container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), bottom, widget.NewSeparator(), statusLabel),
		nil, nil,
		main,
	)

	win.SetContent(content)
	setMode("view")
	reload(0)
	win.Show()
}
