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

	"obj_catalog_fyne_v3/pkg/contracts"
)

func ShowAdminMessagesDialog(parent fyne.Window, provider contracts.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Управління повідомленнями адміністратора")
	win.Resize(fyne.NewSize(1024, 768))

	var (
		regularMessages []contracts.AdminMessage
		adminMessages   []contracts.AdminMessage
		selectedLeft    = -1
		selectedRight   = -1
		protocolOption  = map[string]int64{}
	)

	statusLabel := widget.NewLabel("Готово")

	protocolSelect := widget.NewSelect([]string{"Всі"}, nil)
	protocolSelect.SetSelected("Всі")

	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Фільтр (текст / hex / код)")

	messageDisplay := func(m contracts.AdminMessage) string {
		idText := "?"
		if m.MessageID != nil {
			idText = strconv.FormatInt(*m.MessageID, 10)
		}
		if strings.TrimSpace(m.MessageHex) != "" {
			idText = m.MessageHex
		}
		text := strings.TrimSpace(m.Text)
		if text == "" {
			text = "(без тексту)"
		}
		return fmt.Sprintf("%s  %s", idText, text)
	}

	leftList := widget.NewList(
		func() int { return len(regularMessages) },
		func() fyne.CanvasObject { return widget.NewLabel("message") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(regularMessages) {
				label.SetText("")
				return
			}
			label.SetText(messageDisplay(regularMessages[id]))
		},
	)
	leftList.OnSelected = func(id widget.ListItemID) {
		selectedLeft = id
		selectedRight = -1
	}

	rightList := widget.NewList(
		func() int { return len(adminMessages) },
		func() fyne.CanvasObject { return widget.NewLabel("message") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(adminMessages) {
				label.SetText("")
				return
			}
			label.SetText(messageDisplay(adminMessages[id]))
		},
	)
	rightList.OnSelected = func(id widget.ListItemID) {
		selectedRight = id
		selectedLeft = -1
	}

	selectedProtocol := func() (*int64, error) {
		val := strings.TrimSpace(protocolSelect.Selected)
		if val == "" || val == "Всі" {
			return nil, nil
		}
		if id, ok := protocolOption[val]; ok {
			return &id, nil
		}
		return nil, fmt.Errorf("unknown protocol option: %s", val)
	}

	reload := func() {
		protocolID, err := selectedProtocol()
		if err != nil {
			statusLabel.SetText("Некоректний протокол")
			return
		}

		messages, err := provider.ListMessages(protocolID, strings.TrimSpace(filterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження повідомлень")
			return
		}

		regularMessages = regularMessages[:0]
		adminMessages = adminMessages[:0]
		for _, m := range messages {
			if m.ForAdminOnly {
				adminMessages = append(adminMessages, m)
			} else {
				regularMessages = append(regularMessages, m)
			}
		}

		selectedLeft = -1
		selectedRight = -1
		leftList.UnselectAll()
		rightList.UnselectAll()
		leftList.Refresh()
		rightList.Refresh()
		statusLabel.SetText(fmt.Sprintf("Завантажено: %d (оператор: %d, адміністратор: %d)", len(messages), len(regularMessages), len(adminMessages)))
	}

	loadProtocols := func() {
		protocols, err := provider.ListMessageProtocols()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження протоколів")
			return
		}
		protocolOption = map[string]int64{}
		opts := []string{"Всі"}
		for _, p := range protocols {
			option := protocolOptionLabel(p)
			opts = append(opts, option)
			protocolOption[option] = p
		}
		protocolSelect.Options = opts
		protocolSelect.SetSelected("Всі")
		protocolSelect.Refresh()
	}

	toAdminBtn := widget.NewButton(">", func() {
		if selectedLeft < 0 || selectedLeft >= len(regularMessages) {
			statusLabel.SetText("Виберіть повідомлення зліва")
			return
		}
		msg := regularMessages[selectedLeft]
		if err := provider.SetMessageAdminOnly(msg.UIN, true); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося перенести повідомлення")
			return
		}
		reload()
	})

	fromAdminBtn := widget.NewButton("<", func() {
		if selectedRight < 0 || selectedRight >= len(adminMessages) {
			statusLabel.SetText("Виберіть повідомлення справа")
			return
		}
		msg := adminMessages[selectedRight]
		if err := provider.SetMessageAdminOnly(msg.UIN, false); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося повернути повідомлення")
			return
		}
		reload()
	})

	refreshBtn := widget.NewButton("Оновити", func() { reload() })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	filterEntry.OnSubmitted = func(_ string) { reload() }
	protocolSelect.OnChanged = func(_ string) { reload() }

	top := container.NewHBox(
		widget.NewLabel("Повідомлення:"),
		layout.NewSpacer(),
		widget.NewLabel("Протокол:"),
		protocolSelect,
		widget.NewLabel("Фільтр:"),
		filterEntry,
		refreshBtn,
	)

	leftPanel := container.NewBorder(
		widget.NewLabel("Повідомлення (оператор та адміністратор)"),
		nil, nil, nil,
		leftList,
	)
	rightPanel := container.NewBorder(
		widget.NewLabel("Адмінські"),
		nil, nil, nil,
		rightList,
	)
	midButtons := container.NewVBox(
		layout.NewSpacer(),
		toAdminBtn,
		fromAdminBtn,
		layout.NewSpacer(),
	)

	center := container.NewBorder(
		nil, nil,
		nil, nil,
		container.NewHBox(
			container.NewGridWrap(fyne.NewSize(470, 470), leftPanel),
			container.NewPadded(midButtons),
			container.NewGridWrap(fyne.NewSize(470, 470), rightPanel),
		),
	)

	content := container.NewBorder(
		top,
		container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		nil, nil,
		center,
	)
	win.SetContent(content)

	loadProtocols()
	reload()
	win.Show()
}
