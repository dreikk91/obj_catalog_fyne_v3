// Package ui - компонент списку об'єктів для лівої панелі
package ui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/models"
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/widgets"
	"obj_catalog_fyne_v3/pkg/utils"
)

type ObjectListPanel struct {
	Container    *fyne.Container
	Table        *widget.Table
	TableView    *widgets.QTableView
	SearchEntry  *widget.Entry
	FilterSelect *widget.Select
	Data         data.ObjectProvider

	// Кеш усіх об'єктів
	AllObjects    []models.Object
	FilteredItems []models.Object
	isUpdating    bool
	mutex         sync.RWMutex

	CurrentFilter string
	LoadingLabel  *widget.Label
	SelectedRow   int
	TitleText     *canvas.Text
	lastFontSize  float32
	colNameWidth  float32
	colAddrWidth  float32
	rowHeight     int

	// Останній об'єкт, про який повідомили через OnObjectSelected.
	// Потрібно, щоб при авто-виборі гарантовано підвантажувати картку,
	// але не викликати завантаження повторно без зміни вибору.
	lastNotifiedSelectedID int
	hasNotifiedSelection   bool

	// Callback при виборі об'єкта
	OnObjectSelected func(object models.Object)
}

const (
	minObjectListRowHeight = 20
	objectListRowDelta     = 8
)

type objectListTableModel struct {
	panel *ObjectListPanel
}

func (m *objectListTableModel) RowCount() int {
	if m == nil || m.panel == nil {
		return 0
	}
	m.panel.mutex.RLock()
	defer m.panel.mutex.RUnlock()
	return len(m.panel.FilteredItems)
}

func (m *objectListTableModel) ColumnCount() int {
	return 4
}

func (m *objectListTableModel) HeaderData(column int) string {
	switch column {
	case 0:
		return "№"
	case 1:
		return "Об'єкт"
	case 2:
		return "Адреса"
	case 3:
		return "Договір"
	default:
		return ""
	}
}

func (m *objectListTableModel) Data(row, column int) string {
	if m == nil || m.panel == nil {
		return ""
	}
	item, ok := m.panel.getFilteredObject(row)
	if !ok {
		return ""
	}
	switch column {
	case 0:
		return itoa(item.ID)
	case 1:
		return item.Name
	case 2:
		return item.Address
	case 3:
		return item.ContractNum
	default:
		return ""
	}
}

func (p *ObjectListPanel) getFilteredObject(row int) (models.Object, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if row < 0 || row >= len(p.FilteredItems) {
		return models.Object{}, false
	}
	return p.FilteredItems[row], true
}

// NewObjectListPanel створює панель списку об'єктів
func NewObjectListPanel(provider data.ObjectProvider) *ObjectListPanel {
	panel := &ObjectListPanel{
		Data:          provider,
		CurrentFilter: "Всі",
		SelectedRow:   -1,
		colNameWidth:  200,
		colAddrWidth:  250,
	}

	// Заголовок
	panel.TitleText = canvas.NewText("ОБ'ЄКТИ", nil)
	themeSize := fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	panel.TitleText.TextSize = themeSize + 1
	panel.TitleText.TextStyle = fyne.TextStyle{Bold: true}

	// Поле пошуку
	panel.SearchEntry = widget.NewEntry()
	panel.SearchEntry.SetPlaceHolder("🔍 Пошук (№, Назва, Адреса, SIM, Тел...)")
	panel.SearchEntry.OnChanged = func(text string) {
		go panel.applyFilters()
	}

	// Вибір фільтру
	panel.FilterSelect = widget.NewSelect([]string{"Всі", "Є тривоги", "Нема зв'язку", "Знято з охорони"}, func(selected string) {
		if panel.isUpdating {
			return
		}
		cleanFilter := selected
		if idx := strings.Index(selected, " ("); idx != -1 {
			cleanFilter = selected[:idx]
		}
		panel.CurrentFilter = cleanFilter
		go panel.applyFilters()
	})
	panel.FilterSelect.PlaceHolder = "Фільтр"

	// Лейбл завантаження
	panel.LoadingLabel = widget.NewLabel("Завантаження даних...")
	panel.LoadingLabel.Alignment = fyne.TextAlignCenter

	// Таблиця об'єктів на QTableView
	model := &objectListTableModel{panel: panel}
	panel.TableView = widgets.NewQTableView(model)
	panel.Table = panel.TableView.Widget()
	panel.TableView.SetHeaderVisible(true, false)
	panel.TableView.SetSortingEnabled(true)
	panel.TableView.SetSelectionBehavior(widgets.SelectRows)
	panel.TableView.SetCornerButtonEnabled(true)
	panel.TableView.SetGridStyle(widgets.PenSolid)
	panel.TableView.SetShowGrid(true)
	panel.TableView.SetWordWrap(false)
	panel.TableView.SetCellRenderer(
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("", color.White)
			txt.TextStyle.Monospace = true
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(index widgets.ModelIndex, _ string, _ bool, obj fyne.CanvasObject) {
			stack, ok := obj.(*fyne.Container)
			if !ok || len(stack.Objects) < 2 {
				return
			}
			bg, ok := stack.Objects[0].(*canvas.Rectangle)
			if !ok {
				return
			}
			padded, ok := stack.Objects[1].(*fyne.Container)
			if !ok || len(padded.Objects) == 0 {
				return
			}
			txt, ok := padded.Objects[0].(*canvas.Text)
			if !ok {
				return
			}

			if !index.IsValid() {
				txt.Text = ""
				bg.Hide()
				bg.Refresh()
				txt.Refresh()
				return
			}

			item, exists := panel.getFilteredObject(index.Row)
			if !exists {
				txt.Text = ""
				bg.Hide()
				bg.Refresh()
				txt.Refresh()
				return
			}

			// Визначаємо кольори на основі комбінації статусів
			textColor, rowColor := utils.ChangeItemColorNRGBA(item.AlarmState, item.GuardState, item.TechAlarmState, item.IsConnState, IsDarkMode())
			if item.BlockedArmedOnOff == 1 {
				// Тимчасово знято із спостереження
				if IsDarkMode() {
					textColor = color.NRGBA{R: 230, G: 220, B: 245, A: 255}
					rowColor = color.NRGBA{R: 98, G: 52, B: 125, A: 255}
				} else {
					textColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					rowColor = color.NRGBA{R: 144, G: 64, B: 196, A: 255}
				}
			} else if item.BlockedArmedOnOff == 2 {
				// Режим налагодження
				if IsDarkMode() {
					textColor = color.NRGBA{R: 238, G: 236, B: 195, A: 255}
					rowColor = color.NRGBA{R: 95, G: 96, B: 42, A: 255}
				} else {
					textColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
					rowColor = color.NRGBA{R: 128, G: 128, B: 0, A: 255}
				}
			}

			missingSubServer := strings.TrimSpace(item.SubServerA) == "" && strings.TrimSpace(item.SubServerB) == ""
			if missingSubServer {
				textColor = color.NRGBA{R: 210, G: 0, B: 0, A: 255}
				rowColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}

			if missingSubServer {
				bg.FillColor = rowColor
				txt.Color = textColor
			} else if index.Row == panel.SelectedRow {
				bg.FillColor = appTheme.ColorSelection
				txt.Color = color.White
			} else {
				bg.FillColor = rowColor
				txt.Color = textColor
			}
			bg.Show()
			bg.Refresh()

			switch index.Col {
			case 0:
				txt.Text = itoa(item.ID)
			case 1:
				txt.Text = item.Name
			case 2:
				txt.Text = item.Address
			case 3:
				txt.Text = item.ContractNum
			default:
				txt.Text = ""
			}
			if panel.lastFontSize > 0 {
				txt.TextSize = panel.lastFontSize
			} else {
				txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
			}
			txt.Refresh()
		},
	)
	panel.initRowHeight()
	panel.TableView.OnSelected = func(index widgets.ModelIndex) {
		panel.mutex.Lock()
		if index.Row < 0 || index.Row >= len(panel.FilteredItems) {
			panel.mutex.Unlock()
			return
		}
		panel.SelectedRow = index.Row
		selectedObj := panel.FilteredItems[index.Row]
		panel.lastNotifiedSelectedID = selectedObj.ID
		panel.hasNotifiedSelection = true
		panel.mutex.Unlock()

		if panel.OnObjectSelected != nil {
			panel.OnObjectSelected(selectedObj)
		}
		panel.TableView.Refresh()
	}

	// Ширина колонок (початкова)
	panel.TableView.SetColumnWidth(0, 50)
	panel.TableView.SetColumnWidth(1, 200)
	panel.TableView.SetColumnWidth(2, 250)
	panel.TableView.SetColumnWidth(3, 80)

	// Збираємо все разом
	header := container.NewVBox(
		container.NewPadded(panel.TitleText),
		panel.SearchEntry,
		panel.FilterSelect,
	)

	panel.Container = container.NewBorder(
		header,
		nil, nil, nil,
		container.New(
			&objectListTableLayout{panel: panel, table: panel.Table},
			container.NewStack(panel.Table, panel.LoadingLabel),
		),
	)

	// Початкове завантаження (асинхронне)
	go panel.RefreshData()

	return panel
}

func (p *ObjectListPanel) RefreshData() {
	if p.Data == nil {
		return
	}
	objects := p.Data.GetObjects()

	p.mutex.Lock()
	p.AllObjects = objects
	p.mutex.Unlock()

	go p.applyFilters()
}

func (p *ObjectListPanel) applyFilters() {
	if p.TableView == nil {
		return
	}

	// Виконуємо фільтрацію в фоні
	query := strings.ToLower(strings.TrimSpace(p.SearchEntry.Text))
	currentFilter := p.CurrentFilter

	p.mutex.RLock()
	all := p.AllObjects
	prevSelectedID := 0
	hadPrevSelection := false
	lastNotifiedID := p.lastNotifiedSelectedID
	hasNotifiedSelection := p.hasNotifiedSelection
	if p.SelectedRow >= 0 && p.SelectedRow < len(p.FilteredItems) {
		prevSelectedID = p.FilteredItems[p.SelectedRow].ID
		hadPrevSelection = true
	}
	p.mutex.RUnlock()

	var filtered []models.Object
	countAll := 0
	countAlarm := 0
	countOffline := 0
	countDisarmed := 0

	for _, obj := range all {
		matchSearch := true
		if query != "" {
			matchSearch = strings.Contains(strings.ToLower(itoa(obj.ID)), query) ||
				strings.Contains(strings.ToLower(obj.Name), query) ||
				strings.Contains(strings.ToLower(obj.Address), query) ||
				strings.Contains(strings.ToLower(obj.ContractNum), query) ||
				strings.Contains(strings.ToLower(obj.SIM1), query) ||
				strings.Contains(strings.ToLower(obj.SIM2), query) ||
				strings.Contains(strings.ToLower(obj.Phone), query)
		}
		if !matchSearch {
			continue
		}

		countAll++
		if obj.Status == models.StatusFire || obj.Status == models.StatusFault {
			countAlarm++
		}
		if obj.IsConnState == 0 && obj.GuardState != 0 {
			countOffline++
		}
		if obj.GuardState == 0 {
			countDisarmed++
		}

		statusMatch := true
		switch currentFilter {
		case "Є тривоги":
			if obj.Status != models.StatusFire && obj.Status != models.StatusFault {
				statusMatch = false
			}
		case "Нема зв'язку":
			if !(obj.IsConnState == 0 && obj.GuardState != 0) {
				statusMatch = false
			}
		case "Знято з охорони":
			if obj.GuardState != 0 {
				statusMatch = false
			}
		}

		if statusMatch {
			filtered = append(filtered, obj)
		}
	}

	// Підтримуємо стабільний вибір:
	// 1) залишаємо поточний об'єкт, якщо він є після фільтрації;
	// 2) якщо вибору немає, автоматично вибираємо перший рядок.
	newSelectedRow := -1
	if hadPrevSelection {
		for i := range filtered {
			if filtered[i].ID == prevSelectedID {
				newSelectedRow = i
				break
			}
		}
	}
	if newSelectedRow == -1 && len(filtered) > 0 {
		newSelectedRow = 0
	}

	var selectedObj models.Object
	shouldNotifySelection := false
	if newSelectedRow >= 0 {
		selectedObj = filtered[newSelectedRow]
		if !hasNotifiedSelection || selectedObj.ID != lastNotifiedID || (!hadPrevSelection || selectedObj.ID != prevSelectedID) {
			shouldNotifySelection = true
		}
	}

	p.mutex.Lock()
	p.FilteredItems = filtered
	p.SelectedRow = newSelectedRow
	if newSelectedRow < 0 {
		p.hasNotifiedSelection = false
		p.lastNotifiedSelectedID = 0
	}
	p.mutex.Unlock()

	fyne.Do(func() {
		p.isUpdating = true
		defer func() { p.isUpdating = false }()

		p.FilterSelect.Options = []string{
			fmt.Sprintf("Всі (%d)", countAll),
			fmt.Sprintf("Є тривоги (%d)", countAlarm),
			fmt.Sprintf("Нема зв'язку (%d)", countOffline),
			fmt.Sprintf("Знято з охорони (%d)", countDisarmed),
		}
		for _, opt := range p.FilterSelect.Options {
			if strings.HasPrefix(opt, currentFilter+" (") || opt == currentFilter {
				p.FilterSelect.SetSelected(opt)
				break
			}
		}
		p.FilterSelect.Refresh()

		if p.TitleText != nil {
			p.TitleText.Text = fmt.Sprintf("ОБ'ЄКТИ (%d)", countAll)
			p.TitleText.Refresh()
		}

		if p.LoadingLabel != nil {
			p.LoadingLabel.Hide()
		}
		if p.Table != nil {
			p.Table.Show()
		}
		if p.TableView != nil {
			p.applyCompactRowHeights(len(filtered))
			p.TableView.Refresh()
		}

		if shouldNotifySelection && p.OnObjectSelected != nil {
			p.OnObjectSelected(selectedObj)
			p.mutex.Lock()
			p.lastNotifiedSelectedID = selectedObj.ID
			p.hasNotifiedSelection = true
			p.mutex.Unlock()
		}
	})
}

func (p *ObjectListPanel) initRowHeight() {
	if p == nil || p.TableView == nil || p.rowHeight > 0 {
		return
	}
	compact := p.TableView.RowHeight(0) - objectListRowDelta
	if compact < minObjectListRowHeight {
		compact = minObjectListRowHeight
	}
	p.rowHeight = compact
}

func (p *ObjectListPanel) applyCompactRowHeights(rowCount int) {
	if p == nil || p.TableView == nil || rowCount <= 0 {
		return
	}
	p.initRowHeight()
	for row := 0; row < rowCount; row++ {
		p.TableView.SetRowHeight(row, p.rowHeight)
	}
}

// objectListTableLayout для динамічного ресайзу колонок "Назва" та "Адреса"
type objectListTableLayout struct {
	panel         *ObjectListPanel
	table         *widget.Table
	lastNameWidth float32
	lastAddrWidth float32
}

func (l *objectListTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Фіксовані колонки: ID(50) + Контракт(80) = 130
	fixedWidth := float32(130)

	// Доступна ширина для динамічних колонок "Назва" (1) та "Адреса" (2)
	available := size.Width - fixedWidth - 10 // невеликий буфер під скролл
	if available < 260 {
		available = 260
	}

	// Розподіляємо доступну ширину між "Назва" та "Адреса" (45% / 55%)
	nameWidth := available * 0.45
	addrWidth := available * 0.55

	// Мінімальні ширини, щоб текст був читабельний
	if nameWidth < 140 {
		nameWidth = 140
	}
	if addrWidth < 160 {
		addrWidth = 160
	}

	needRefresh := false
	if l.lastNameWidth != nameWidth {
		if l.panel != nil && l.panel.TableView != nil {
			l.panel.TableView.SetColumnWidth(1, int(nameWidth))
		} else if l.table != nil {
			l.table.SetColumnWidth(1, nameWidth)
		}
		l.lastNameWidth = nameWidth
		if l.panel != nil {
			l.panel.colNameWidth = nameWidth
		}
		needRefresh = true
	}
	if l.lastAddrWidth != addrWidth {
		if l.panel != nil && l.panel.TableView != nil {
			l.panel.TableView.SetColumnWidth(2, int(addrWidth))
		} else if l.table != nil {
			l.table.SetColumnWidth(2, addrWidth)
		}
		l.lastAddrWidth = addrWidth
		if l.panel != nil {
			l.panel.colAddrWidth = addrWidth
		}
		needRefresh = true
	}
	if needRefresh {
		if l.panel != nil && l.panel.TableView != nil {
			l.panel.TableView.Refresh()
		} else if l.table != nil {
			l.table.Refresh()
		}
	}

	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func (l *objectListTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(450, 200)
}

func (p *ObjectListPanel) Refresh() {
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	p.OnThemeChanged(uiCfg.FontSizeObjects)
	go p.RefreshData()
}

func (p *ObjectListPanel) OnThemeChanged(fontSize float32) {
	p.lastFontSize = fontSize
	if p.TitleText != nil {
		p.TitleText.TextSize = fontSize + 1
		p.TitleText.Refresh()
	}
	if p.TableView != nil {
		p.TableView.Refresh()
	}
}
