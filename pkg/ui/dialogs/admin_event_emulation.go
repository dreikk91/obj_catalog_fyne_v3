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

type adminEventEmulationProvider interface {
	ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error)
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
	ListMessageProtocols() ([]int64, error)
	EmulateEvent(objn int64, zone int64, messageUIN int64) error
}

func ShowEventEmulationDialog(parent fyne.Window, provider adminEventEmulationProvider, onEmulated func()) {
	win := fyne.CurrentApp().NewWindow("Емуляція подій")
	win.Resize(fyne.NewSize(1120, 680))

	var (
		objects            []contracts.DisplayBlockObject
		messages           []contracts.AdminMessage
		selectedMessageID  int64
		selectedMessageRow = -1
		protocolOptionID   = map[string]int64{}
	)

	statusLabel := widget.NewLabel("Готово")

	objNumEntry := widget.NewEntry()
	objNumEntry.SetPlaceHolder("№ об'єкта")
	zoneEntry := widget.NewEntry()
	zoneEntry.SetText("1")

	objectFilterEntry := widget.NewEntry()
	objectFilterEntry.SetPlaceHolder("Фільтр об'єктів")
	messageFilterEntry := widget.NewEntry()
	messageFilterEntry.SetPlaceHolder("Фільтр повідомлень")

	protocolSelect := widget.NewSelect([]string{"Всі"}, nil)
	protocolSelect.SetSelected("Всі")

	chAlarm := widget.NewCheck("тривожні", nil)
	chTech := widget.NewCheck("технічні", nil)
	chRestore := widget.NewCheck("відновлення", nil)
	chTest := widget.NewCheck("тестові", nil)
	chInfo := widget.NewCheck("інформаційні", nil)
	for _, ch := range []*widget.Check{chAlarm, chTech, chRestore, chTest, chInfo} {
		ch.SetChecked(true)
	}

	objectList := widget.NewList(
		func() int { return len(objects) },
		func() fyne.CanvasObject { return widget.NewLabel("object") },
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			if id < 0 || id >= len(objects) {
				label.SetText("")
				return
			}
			label.SetText(fmt.Sprintf("%d  %s", objects[id].ObjN, objects[id].Name))
		},
	)
	objectList.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(objects) {
			return
		}
		objNumEntry.SetText(strconv.FormatInt(objects[id].ObjN, 10))
		statusLabel.SetText(fmt.Sprintf("Обрано об'єкт №%d", objects[id].ObjN))
	}

	messageTable := widget.NewTable(
		func() (int, int) { return len(messages), 4 },
		func() fyne.CanvasObject { return newColoredTableCell() },
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			if id.Row < 0 || id.Row >= len(messages) {
				updateColoredMessageCell(obj, "", nil, false)
				return
			}
			m := messages[id.Row]
			cellText := ""
			switch id.Col {
			case 0:
				if m.MessageID != nil {
					cellText = strconv.FormatInt(*m.MessageID, 10)
				} else {
					cellText = strconv.FormatInt(m.UIN, 10)
				}
			case 1:
				cellText = strings.TrimSpace(m.Text)
			case 2:
				cellText = messageTypeLabel(m.SC1)
			default:
				if m.ProtocolID != nil {
					cellText = protocolDisplayName(*m.ProtocolID)
				} else {
					cellText = "—"
				}
			}
			updateColoredMessageCell(obj, cellText, m.SC1, id.Row == selectedMessageRow)
		},
	)
	messageTable.SetColumnWidth(0, 90)
	messageTable.SetColumnWidth(1, 620)
	messageTable.SetColumnWidth(2, 170)
	messageTable.SetColumnWidth(3, 100)
	messageTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(messages) {
			return
		}
		selectedMessageID = messages[id.Row].UIN
		selectedMessageRow = id.Row
		messageTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Обрано повідомлення UIN=%d", selectedMessageID))
	}

	selectedProtocol := func() (*int64, error) {
		v := strings.TrimSpace(protocolSelect.Selected)
		if v == "" || v == "Всі" {
			return nil, nil
		}
		if id, ok := protocolOptionID[v]; ok {
			return &id, nil
		}
		return nil, fmt.Errorf("unknown protocol option: %s", v)
	}

	messagePasses := func(m contracts.AdminMessage) bool {
		families := []struct {
			name  string
			check *widget.Check
		}{
			{name: "alarm", check: chAlarm},
			{name: "tech", check: chTech},
			{name: "restore", check: chRestore},
			{name: "test", check: chTest},
			{name: "info", check: chInfo},
		}
		for _, fam := range families {
			if fam.check.Checked && sc1MatchesFamily(m.SC1, fam.name) {
				return true
			}
		}
		return false
	}

	reloadObjects := func() {
		loaded, err := provider.ListDisplayBlockObjects(strings.TrimSpace(objectFilterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження об'єктів")
			return
		}
		objects = loaded
		objectList.Refresh()
	}

	reloadMessages := func() {
		protocolID, err := selectedProtocol()
		if err != nil {
			statusLabel.SetText("Некоректний номер протоколу")
			return
		}

		loaded, err := provider.ListMessages(protocolID, strings.TrimSpace(messageFilterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження повідомлень")
			return
		}

		filtered := make([]contracts.AdminMessage, 0, len(loaded))
		for _, m := range loaded {
			if messagePasses(m) {
				filtered = append(filtered, m)
			}
		}
		messages = filtered
		selectedMessageID = 0
		selectedMessageRow = -1
		messageTable.UnselectAll()
		messageTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Повідомлень для емуляції: %d", len(messages)))
	}

	loadProtocols := func() {
		protocols, err := provider.ListMessageProtocols()
		if err != nil {
			dialog.ShowError(err, win)
			return
		}
		protocolOptionID = map[string]int64{}
		opts := []string{"Всі"}
		for _, p := range protocols {
			option := protocolOptionLabel(p)
			opts = append(opts, option)
			protocolOptionID[option] = p
		}
		protocolSelect.Options = opts
		protocolSelect.SetSelected("Всі")
		protocolSelect.Refresh()
	}

	emulateBtn := widget.NewButton("Емулювати", func() {
		objRaw := strings.TrimSpace(objNumEntry.Text)
		if objRaw == "" {
			statusLabel.SetText("Вкажіть або виберіть № об'єкта")
			return
		}
		objN, err := strconv.ParseInt(objRaw, 10, 64)
		if err != nil || objN <= 0 {
			statusLabel.SetText("Некоректний № об'єкта")
			return
		}

		zoneRaw := strings.TrimSpace(zoneEntry.Text)
		zone, err := strconv.ParseInt(zoneRaw, 10, 64)
		if err != nil || zone < 0 {
			statusLabel.SetText("Некоректний номер зони")
			return
		}

		if selectedMessageID <= 0 {
			statusLabel.SetText("Виберіть подію для емуляції")
			return
		}

		if err := provider.EmulateEvent(objN, zone, selectedMessageID); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося виконати емуляцію")
			return
		}

		statusLabel.SetText(fmt.Sprintf("Емуляцію виконано: об'єкт №%d, зона %d", objN, zone))
		if onEmulated != nil {
			onEmulated()
		}
	})

	refreshBtn := widget.NewButton("Оновити", func() {
		reloadObjects()
		reloadMessages()
	})
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	objectFilterEntry.OnSubmitted = func(_ string) { reloadObjects() }
	messageFilterEntry.OnSubmitted = func(_ string) { reloadMessages() }
	protocolSelect.OnChanged = func(_ string) { reloadMessages() }
	for _, ch := range []*widget.Check{chAlarm, chTech, chRestore, chTest, chInfo} {
		ch.OnChanged = func(_ bool) { reloadMessages() }
	}

	left := container.NewBorder(
		container.NewVBox(
			widget.NewLabel("Об'єкт"),
			objectFilterEntry,
		),
		nil, nil, nil,
		objectList,
	)

	headers := container.NewGridWithColumns(
		4,
		widget.NewLabel("Код"),
		widget.NewLabel("Повідомлення"),
		widget.NewLabel("Тип"),
		widget.NewLabel("Прот."),
	)
	right := container.NewBorder(
		container.NewVBox(
			container.NewHBox(
				widget.NewLabel("Протокол:"),
				protocolSelect,
				widget.NewLabel("Фільтр:"),
				messageFilterEntry,
			),
			container.NewHBox(
				chAlarm, chTech, chRestore, chTest, chInfo,
			),
			widget.NewSeparator(),
			headers,
		),
		nil, nil, nil,
		messageTable,
	)

	split := container.NewHSplit(left, right)
	split.SetOffset(0.30)

	top := container.NewHBox(
		widget.NewLabel("№пр.:"),
		container.NewGridWrap(fyne.NewSize(120, 36), objNumEntry),
		widget.NewLabel("Зона:"),
		container.NewGridWrap(fyne.NewSize(90, 36), zoneEntry),
		layout.NewSpacer(),
		refreshBtn,
		emulateBtn,
	)

	content := container.NewBorder(
		top,
		container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		nil, nil,
		split,
	)
	win.SetContent(content)

	loadProtocols()
	reloadObjects()
	reloadMessages()
	win.Show()
}

// protocolDisplayName повертає читабельну назву протоколу.
// У наявній БД окремого довідника протоколів немає, тому використовуємо
// мапу з типових протоколів з адмінського мануалу.
func protocolDisplayName(id int64) string {
	switch id {
	case 18:
		return "Контакт ID"
	case 3:
		return "20bps / ADEMCO"
	case 4:
		return "МОСТ"
	default:
		return fmt.Sprintf("Протокол %d", id)
	}
}

func protocolOptionLabel(id int64) string {
	return fmt.Sprintf("%s [%d]", protocolDisplayName(id), id)
}
