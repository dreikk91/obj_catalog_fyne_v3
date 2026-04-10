package dialogs

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type eventOverrideDialogProvider interface {
	adminMessagesDialogProvider
	admin220VConstructorProvider
	ListMessages(protocolID *int64, filter string) ([]contracts.AdminMessage, error)
	SetMessageAdminOnly(uin int64, adminOnly bool) error
	SetMessageCategory(uin int64, sc1 *int64) error
}

func ShowEventOverrideDialog(parent fyne.Window, provider eventOverrideDialogProvider) {
	win := fyne.CurrentApp().NewWindow("Глобальне перевизначення подій")
	win.Resize(fyne.NewSize(980, 700))

	var (
		messages           []contracts.AdminMessage
		selectedUIN        int64
		selectedRowIdx           = -1
		selectedProtocolID int64 = 18
	)

	statusLabel := widget.NewLabel("Готово")
	filterEntry := widget.NewEntry()
	filterEntry.SetPlaceHolder("Фільтр")
	filterByCheck := widget.NewCheck("Фільтрувати", nil)
	filterByCheck.SetChecked(true)

	searchMode := widget.NewRadioGroup([]string{"Код", "Повідомлення"}, nil)
	searchMode.Horizontal = true
	searchMode.SetSelected("Повідомлення")

	applyFilterControlsState := func() {
		if filterByCheck.Checked {
			filterEntry.Enable()
		} else {
			filterEntry.Disable()
		}
	}

	scenarioLabelFromSC1 := func(sc1 *int64) string {
		if sc1 == nil {
			return "Інформація"
		}
		switch *sc1 {
		case 1:
			return "Тривога"
		case 2, 3:
			return "Тривога техн."
		case 5, 9, 13, 17:
			return "Відновлення"
		case 12:
			return "Подію заборонено"
		case 16:
			return "Тестове повідомлення"
		default:
			return "Інформація"
		}
	}

	scenarioSC1FromLabel := func(label string) *int64 {
		switch label {
		case "Тривога":
			return i64(1)
		case "Тривога техн.":
			return i64(2)
		case "Відновлення":
			return i64(5)
		case "Подію заборонено":
			return i64(12)
		case "Тестове повідомлення":
			return i64(16)
		case "Інформація":
			return i64(6)
		default:
			return i64(6)
		}
	}

	scenarioRadio := widget.NewRadioGroup(
		[]string{
			"Тривога",
			"Тривога техн.",
			"Відновлення",
			"Інформація",
			"Подію заборонено",
			"Тестове повідомлення",
		},
		nil,
	)
	scenarioRadio.Disable()
	adminOnlyCheck := widget.NewCheck("Для адміністратора", nil)
	adminOnlyCheck.Disable()

	eventTextEntry := widget.NewEntry()
	eventTextEntry.Disable()

	clearSelectionDetails := func() {
		scenarioRadio.SetSelected("Інформація")
		adminOnlyCheck.SetChecked(false)
		adminOnlyCheck.Disable()
		eventTextEntry.SetText("")
	}

	updateSelectionDetails := func(m *contracts.AdminMessage) {
		if m == nil {
			clearSelectionDetails()
			scenarioRadio.Disable()
			return
		}
		scenarioRadio.Enable()
		adminOnlyCheck.Enable()
		scenarioRadio.SetSelected(scenarioLabelFromSC1(m.SC1))
		adminOnlyCheck.SetChecked(m.ForAdminOnly)
		eventTextEntry.SetText(strings.TrimSpace(m.Text))
	}

	applyCodeOnlyFilter := func(in []contracts.AdminMessage, needle string) []contracts.AdminMessage {
		needle = strings.ToLower(strings.TrimSpace(needle))
		if needle == "" {
			return in
		}
		out := make([]contracts.AdminMessage, 0, len(in))
		for _, m := range in {
			decCode := ""
			if m.MessageID != nil {
				decCode = strconv.FormatInt(*m.MessageID, 10)
			} else {
				decCode = strconv.FormatInt(m.UIN, 10)
			}
			hexCode := strings.TrimSpace(m.MessageHex)
			if strings.Contains(strings.ToLower(decCode), needle) || strings.Contains(strings.ToLower(hexCode), needle) {
				out = append(out, m)
			}
		}
		return out
	}

	codeForSort := func(m contracts.AdminMessage) int64 {
		if m.MessageID != nil {
			return *m.MessageID
		}
		return m.UIN
	}

	sortMessages := func(in []contracts.AdminMessage) {
		if strings.EqualFold(searchMode.Selected, "Повідомлення") {
			sort.SliceStable(in, func(i, j int) bool {
				ti := strings.ToLower(strings.TrimSpace(in[i].Text))
				tj := strings.ToLower(strings.TrimSpace(in[j].Text))
				if ti == tj {
					return codeForSort(in[i]) < codeForSort(in[j])
				}
				return ti < tj
			})
			return
		}
		sort.SliceStable(in, func(i, j int) bool {
			ci := codeForSort(in[i])
			cj := codeForSort(in[j])
			if ci == cj {
				ti := strings.ToLower(strings.TrimSpace(in[i].Text))
				tj := strings.ToLower(strings.TrimSpace(in[j].Text))
				return ti < tj
			}
			return ci < cj
		})
	}

	messageTable := widget.NewTable(
		func() (int, int) { return len(messages), 3 },
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
			default:
				cellText = scenarioLabelFromSC1(m.SC1)
			}
			updateColoredMessageCell(obj, cellText, m.SC1, id.Row == selectedRowIdx)
		},
	)
	messageTable.SetColumnWidth(0, 110)
	messageTable.SetColumnWidth(1, 500)
	messageTable.SetColumnWidth(2, 210)
	messageTable.OnSelected = func(id widget.TableCellID) {
		if id.Row < 0 || id.Row >= len(messages) {
			return
		}
		selectedRowIdx = id.Row
		selectedUIN = messages[id.Row].UIN
		updateSelectionDetails(&messages[id.Row])
		messageTable.Refresh()
		statusLabel.SetText(fmt.Sprintf("Вибрано повідомлення UIN=%d", selectedUIN))
	}

	reload := func(reselectUIN int64) {
		protocolID := &selectedProtocolID
		filterValue := ""
		if filterByCheck.Checked {
			filterValue = strings.TrimSpace(filterEntry.Text)
		}

		providerFilter := filterValue
		if strings.EqualFold(searchMode.Selected, "Код") {
			providerFilter = ""
		}

		loaded, err := provider.ListMessages(protocolID, providerFilter)
		if err != nil {
			dialog.ShowError(err, win)
			statusLabel.SetText("Помилка завантаження повідомлень")
			return
		}
		if strings.EqualFold(searchMode.Selected, "Код") && filterValue != "" {
			loaded = applyCodeOnlyFilter(loaded, filterValue)
		}
		sortMessages(loaded)

		messages = loaded
		selectedUIN = 0
		selectedRowIdx = -1
		messageTable.UnselectAll()
		messageTable.Refresh()
		updateSelectionDetails(nil)

		if reselectUIN != 0 {
			for i := range messages {
				if messages[i].UIN == reselectUIN {
					selectedUIN = reselectUIN
					selectedRowIdx = i
					messageTable.Select(widget.TableCellID{Row: i, Col: 0})
					statusLabel.SetText(fmt.Sprintf("Знайдено повідомлень: %d", len(messages)))
					return
				}
			}
		}
		if len(messages) > 0 {
			messageTable.Select(widget.TableCellID{Row: 0, Col: 0})
		}
		statusLabel.SetText(fmt.Sprintf("Знайдено повідомлень: %d", len(messages)))
	}

	applyBtn := widget.NewButton("Змінити", func() {
		if selectedUIN <= 0 || selectedRowIdx < 0 || selectedRowIdx >= len(messages) {
			statusLabel.SetText("Спочатку виберіть повідомлення в таблиці")
			return
		}
		selectedCategory := scenarioSC1FromLabel(scenarioRadio.Selected)
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
	open220Btn := widget.NewButton("Пропажа / відновлення 220В", func() {
		Show220VConstructorDialog(win, provider)
	})
	openAIBtn := widget.NewButton("+", func() {
		ShowAdminPlaceholderDialog(win, "Додатково", "Кнопка '+' збережена для візуальної сумісності з legacy-формою.")
	})
	refreshBtn := widget.NewButton("Оновити", func() { reload(selectedUIN) })
	closeBtn := widget.NewButton("Закрити", func() { win.Close() })
	executeFilterBtn := widget.NewButton("Виконати", func() { reload(selectedUIN) })

	filterEntry.OnSubmitted = func(_ string) { reload(selectedUIN) }
	filterByCheck.OnChanged = func(_ bool) {
		applyFilterControlsState()
		reload(selectedUIN)
	}
	searchMode.OnChanged = func(v string) {
		if strings.EqualFold(v, "Код") {
			filterEntry.SetPlaceHolder("Код / hex")
		} else {
			filterEntry.SetPlaceHolder("Текст повідомлення")
		}
		reload(selectedUIN)
	}

	tabDefinitions := []struct {
		Label string
		ID    int64
	}{
		{Label: "Contact ID", ID: 18},
		{Label: "20BPS / Ademko-Express", ID: 3},
		{Label: "Мост", ID: 4},
	}
	tabProtocol := map[string]int64{}
	tabItems := make([]*container.TabItem, 0, len(tabDefinitions))
	for _, def := range tabDefinitions {
		tabProtocol[def.Label] = def.ID
		tabItems = append(tabItems, container.NewTabItem(def.Label, container.NewStack()))
	}
	protocolTabs := container.NewAppTabs(tabItems...)
	protocolTabs.SetTabLocation(container.TabLocationTop)
	protocolTabs.OnSelected = func(item *container.TabItem) {
		if item == nil {
			return
		}
		if id, ok := tabProtocol[item.Text]; ok {
			selectedProtocolID = id
			reload(selectedUIN)
		}
	}
	protocolTabs.SelectIndex(0)

	filterRow := container.NewBorder(
		nil, nil,
		container.NewHBox(searchMode, filterByCheck),
		executeFilterBtn,
		filterEntry,
	)

	header := container.NewGridWithColumns(
		3,
		widget.NewLabel("Код"),
		widget.NewLabel("Повідомлення"),
		widget.NewLabel("Сценарій"),
	)

	detailsGroup := widget.NewCard("Текст події:", "", container.NewVBox(eventTextEntry, adminOnlyCheck))

	left := container.NewBorder(
		container.NewVBox(filterRow, widget.NewSeparator(), header),
		detailsGroup,
		nil, nil,
		messageTable,
	)

	scenarioGroup := widget.NewCard("Сценарій:", "", scenarioRadio)
	right := container.NewVBox(
		scenarioGroup,
		widget.NewSeparator(),
		applyBtn,
		layout.NewSpacer(),
	)

	body := container.NewHSplit(left, right)
	body.SetOffset(0.74)

	bottomButtons := container.NewHBox(
		openAdminMessagesBtn,
		open220Btn,
		openAIBtn,
		layout.NewSpacer(),
		refreshBtn,
		closeBtn,
	)

	content := container.NewBorder(
		protocolTabs,
		container.NewVBox(
			widget.NewSeparator(),
			bottomButtons,
			widget.NewSeparator(),
			statusLabel,
		),
		nil, nil,
		body,
	)
	win.SetContent(content)

	clearSelectionDetails()
	applyFilterControlsState()
	reload(0)
	win.Show()
}
