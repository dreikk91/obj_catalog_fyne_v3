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
	uiwidgets "obj_catalog_fyne_v3/pkg/ui/widgets"
)

func ShowEventOverrideDialog(parent fyne.Window, provider data.AdminProvider) {
	win := fyne.CurrentApp().NewWindow("Глобальне перевизначення подій")
	win.Resize(fyne.NewSize(1160, 700))

	var (
		messages       []data.AdminMessage
		selectedUIN    int64
		selectedRowIdx = -1
	)

	statusLabel := widget.NewLabel("Готово")
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Фільтр (текст / hex / код)")
	protocolSelect := widget.NewSelect([]string{"Всі"}, nil)
	protocolSelect.SetSelected("Всі")

	categoryLabels := make([]string, 0, len(messageCategoryOptions()))
	for _, c := range messageCategoryOptions() {
		categoryLabels = append(categoryLabels, c.Label)
	}
	categorySelect := widget.NewSelect(categoryLabels, nil)
	categorySelect.SetSelected("Інше / без категорії")
	adminOnlyCheck := widget.NewCheck("Для адміністратора", nil)

	messageTableView := uiwidgets.NewQTableViewWithHeaders(
		[]string{"Код", "Hex", "Повідомлення", "Категорія", "Адмін"},
		func() int { return len(messages) },
		func(row, col int) string {
			if row < 0 || row >= len(messages) {
				return ""
			}
			m := messages[row]
			switch col {
			case 0:
				if m.MessageID != nil {
					return strconv.FormatInt(*m.MessageID, 10)
				}
				return strconv.FormatInt(m.UIN, 10)
			case 1:
				if strings.TrimSpace(m.MessageHex) != "" {
					return m.MessageHex
				}
				return "—"
			case 2:
				return strings.TrimSpace(m.Text)
			case 3:
				return categoryLabelFromSC1(m.SC1)
			default:
				if m.ForAdminOnly {
					return "так"
				}
				return "ні"
			}
		},
	)
	messageTableView.SetSelectionBehavior(uiwidgets.SelectRows)
	messageTableView.SetCellRenderer(
		func() fyne.CanvasObject { return newColoredTableCell() },
		func(index uiwidgets.ModelIndex, _ string, selected bool, obj fyne.CanvasObject) {
			if !index.IsValid() || index.Row < 0 || index.Row >= len(messages) {
				updateColoredMessageCell(obj, "", nil, false)
				return
			}
			m := messages[index.Row]
			cellText := ""
			switch index.Col {
			case 0:
				if m.MessageID != nil {
					cellText = strconv.FormatInt(*m.MessageID, 10)
				} else {
					cellText = strconv.FormatInt(m.UIN, 10)
				}
			case 1:
				if strings.TrimSpace(m.MessageHex) != "" {
					cellText = m.MessageHex
				} else {
					cellText = "—"
				}
			case 2:
				cellText = strings.TrimSpace(m.Text)
			case 3:
				cellText = categoryLabelFromSC1(m.SC1)
			default:
				if m.ForAdminOnly {
					cellText = "так"
				} else {
					cellText = "ні"
				}
			}
			updateColoredMessageCell(obj, cellText, m.SC1, selected)
		},
	)
	messageTable := messageTableView.Widget()
	messageTableView.SetColumnWidth(0, 85)
	messageTableView.SetColumnWidth(1, 120)
	messageTableView.SetColumnWidth(2, 590)
	messageTableView.SetColumnWidth(3, 210)
	messageTableView.SetColumnWidth(4, 95)
	messageTableView.OnSelected = func(index uiwidgets.ModelIndex) {
		if index.Row < 0 || index.Row >= len(messages) {
			return
		}
		selectedRowIdx = index.Row
		selectedUIN = messages[index.Row].UIN
		categorySelect.SetSelected(categoryLabelFromSC1(messages[index.Row].SC1))
		adminOnlyCheck.SetChecked(messages[index.Row].ForAdminOnly)
		messageTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Вибрано повідомлення UIN=%d", selectedUIN))
	}

	selectedProtocol := func() (*int64, error) {
		raw := strings.TrimSpace(protocolSelect.Selected)
		if raw == "" || raw == "Всі" {
			return nil, nil
		}
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, err
		}
		return &n, nil
	}

	reload := func(reselectUIN int64) {
		protocolID, err := selectedProtocol()
		if err != nil {
			statusLabel.SetText("Некоректний протокол")
			return
		}

		loaded, err := provider.ListMessages(protocolID, strings.TrimSpace(filterEntry.Text))
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження повідомлень")
			return
		}

		messages = loaded
		selectedUIN = 0
		selectedRowIdx = -1
		messageTable.UnselectAll()
		messageTable.Refresh()

		if reselectUIN != 0 {
			for i := range messages {
				if messages[i].UIN == reselectUIN {
					selectedUIN = reselectUIN
					selectedRowIdx = i
					messageTable.Select(widget.TableCellID{Row: i, Col: 0})
					return
				}
			}
		}
		if len(messages) > 0 {
			messageTable.Select(widget.TableCellID{Row: 0, Col: 0})
		}
		statusLabel.SetText(fmt.Sprintf("Знайдено повідомлень: %d", len(messages)))
	}

	loadProtocols := func() {
		protocols, err := provider.ListMessageProtocols()
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося завантажити протоколи")
			return
		}
		opts := []string{"Всі"}
		for _, p := range protocols {
			opts = append(opts, strconv.FormatInt(p, 10))
		}
		protocolSelect.Options = opts
		protocolSelect.SetSelected("Всі")
		protocolSelect.Refresh()
	}

	applyBtn := widget.NewButton("Змінити", func() {
		if selectedUIN <= 0 || selectedRowIdx < 0 || selectedRowIdx >= len(messages) {
			statusLabel.SetText("Спочатку виберіть повідомлення в таблиці")
			return
		}
		selectedCategory := categorySC1FromLabel(categorySelect.Selected)
		if err := provider.SetMessageCategory(selectedUIN, selectedCategory); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося змінити категорію")
			return
		}
		if err := provider.SetMessageAdminOnly(selectedUIN, adminOnlyCheck.Checked); err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Не вдалося змінити прапорець адміністратора")
			return
		}

		statusLabel.SetText("Зміни застосовано")
		reload(selectedUIN)
	})

	openAdminMessagesBtn := widget.NewButton("Керування адмін повідомленнями", func() {
		ShowAdminMessagesDialog(win, provider)
	})
	refreshBtn := widget.NewButton("Оновити", func() { reload(selectedUIN) })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })

	filterEntry.OnSubmitted = func(_ string) { reload(selectedUIN) }
	protocolSelect.OnChanged = func(_ string) { reload(selectedUIN) }

	top := container.NewHBox(
		widget.NewLabel("Протокол:"),
		protocolSelect,
		widget.NewLabel("Фільтр:"),
		filterEntry,
		refreshBtn,
		openAdminMessagesBtn,
	)

	left := container.NewBorder(
		container.NewVBox(top, widget.NewSeparator()),
		nil, nil, nil,
		messageTable,
	)

	right := container.NewVBox(
		widget.NewLabel("Категорія:"),
		categorySelect,
		adminOnlyCheck,
		widget.NewSeparator(),
		applyBtn,
		layout.NewSpacer(),
	)

	body := container.NewHSplit(left, right)
	body.SetOffset(0.84)

	content := container.NewBorder(
		nil,
		container.NewHBox(statusLabel, layout.NewSpacer(), closeBtn),
		nil, nil,
		body,
	)
	win.SetContent(content)

	loadProtocols()
	reload(0)
	win.Show()
}
