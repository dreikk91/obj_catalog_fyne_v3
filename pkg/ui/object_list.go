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
	"obj_catalog_fyne_v3/pkg/utils"
)

type ObjectListPanel struct {
	Container    *fyne.Container
	Table        *widget.Table
	SearchEntry  *widget.Entry
	FilterSelect *widget.Select
	Data         data.ObjectProvider
	ColumnHeader *fyne.Container

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

	// Callback при виборі об'єкта
	OnObjectSelected func(object models.Object)
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
		// Дебоунсинг або просто асинхронний виклик
		go panel.applyFilters()
	}

	// Вибір фільтру
	panel.FilterSelect = widget.NewSelect([]string{"Всі", "Є тривоги", "Нема зв'язку", "Знято з охорони"}, func(selected string) {
		if panel.isUpdating {
			return
		}
		// Видаляємо кількість з назви фільтра перед збереженням
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

	// Таблиця об'єктів (використовує FilteredItems)
	panel.Table = widget.NewTable(
		func() (int, int) {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()
			return len(panel.FilteredItems), 4
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Cell Data", color.White)
			// Буде оновлено в UpdateCell
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()

			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)
			txt.TextStyle.Monospace = true

			if id.Row >= len(panel.FilteredItems) {
				txt.Text = ""
				txt.Refresh()
				bg.Hide()
				bg.Refresh()
				return
			}
			item := panel.FilteredItems[id.Row]

			// Визначаємо кольори на основі комбінації статусів
			textColor, rowColor := utils.ChangeItemColorNRGBA(item.AlarmState, item.GuardState, item.TechAlarmState, item.IsConnState, IsDarkMode())

			if id.Row == panel.SelectedRow {
				bg.FillColor = appTheme.ColorSelection
				bg.Show()
				txt.Color = color.White // Білий для виділеного
			} else {
				// Застосовуємо колір рядка та тексту
				bg.FillColor = rowColor
				bg.Show()
				txt.Color = textColor
			}
			bg.Refresh()

			var cellText string
			switch id.Col {
			case 0:
				cellText = itoa(item.ID)
			case 1:
				// cellText = getStatusIcon(item.Status) + " " + item.Name
				cellText = item.Name
			case 2:
				cellText = item.Address
			case 3:
				cellText = item.ContractNum
			}
			txt.Text = cellText
			if panel.lastFontSize > 0 {
				txt.TextSize = panel.lastFontSize
			} else {
				txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
			}
			txt.Refresh()
		},
	)

	panel.Table.OnSelected = func(id widget.TableCellID) {
		panel.mutex.Lock()
		if id.Row >= len(panel.FilteredItems) {
			panel.mutex.Unlock()
			return
		}

		panel.SelectedRow = id.Row
		selectedObj := panel.FilteredItems[id.Row]
		panel.mutex.Unlock()

		if panel.OnObjectSelected != nil {
			panel.OnObjectSelected(selectedObj)
		}
		panel.Table.Refresh()
	}

	// Ширина колонок (початкова)
	panel.Table.SetColumnWidth(0, 50)  // ID (фіксована)
	panel.Table.SetColumnWidth(1, 200) // Назва (стартове значення, далі динамічна)
	panel.Table.SetColumnWidth(2, 250) // Адреса (стартове значення, далі динамічна)
	panel.Table.SetColumnWidth(3, 80)  // Контракт (фіксована)

	// Заголовки колонок для читабельності таблиці.
	hID := widget.NewLabel("№")
	hName := widget.NewLabel("Об'єкт")
	hAddr := widget.NewLabel("Адреса")
	hContract := widget.NewLabel("Договір")
	for _, l := range []*widget.Label{hID, hName, hAddr, hContract} {
		l.TextStyle = fyne.TextStyle{Bold: true}
	}
	headerRow := container.New(&objectListHeaderLayout{panel: panel}, hID, hName, hAddr, hContract)
	headerBg := canvas.NewRectangle(color.NRGBA{R: 0, G: 0, B: 0, A: 25})
	panel.ColumnHeader = container.NewStack(headerBg, container.NewPadded(headerRow))

	// Збираємо все разом
	header := container.NewVBox(
		container.NewPadded(panel.TitleText),
		panel.SearchEntry,
		panel.FilterSelect,
		panel.ColumnHeader,
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
	// Отримуємо ВСІ об'єкти один раз
	objects := p.Data.GetObjects()

	p.mutex.Lock()
	p.AllObjects = objects
	p.mutex.Unlock()

	// Оновлюємо фільтри асинхронно
	p.applyFilters()
}

func (p *ObjectListPanel) applyFilters() {
	if p.Table == nil {
		return
	}

	// Виконуємо фільтрацію в фоні
	query := strings.ToLower(strings.TrimSpace(p.SearchEntry.Text))
	currentFilter := p.CurrentFilter

	p.mutex.RLock()
	all := p.AllObjects
	p.mutex.RUnlock()

	var filtered []models.Object
	countAll := 0
	countAlarm := 0
	countOffline := 0
	countDisarmed := 0

	for _, obj := range all {
		// 1. Пошук (має працювати для всіх підрахунків)
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

		// Підраховуємо статистику (з урахуванням пошуку)
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

		// 2. Фільтрація для відображення в таблиці
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

	// Оновлюємо список і UI
	p.mutex.Lock()
	p.FilteredItems = filtered
	p.mutex.Unlock()

	fyne.Do(func() {
		p.isUpdating = true
		defer func() { p.isUpdating = false }()

		// Оновлюємо назви фільтрів з кількістю
		p.FilterSelect.Options = []string{
			fmt.Sprintf("Всі (%d)", countAll),
			fmt.Sprintf("Є тривоги (%d)", countAlarm),
			fmt.Sprintf("Нема зв'язку (%d)", countOffline),
			fmt.Sprintf("Знято з охорони (%d)", countDisarmed),
		}

		// Знаходимо поточний вибраний фільтр в оновленому списку, щоб він не зникав
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
			p.Table.Refresh()
		}
	})
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

	// Оновлюємо ширини колонок тільки при зміні значень
	needRefresh := false
	if l.lastNameWidth != nameWidth {
		l.table.SetColumnWidth(1, nameWidth)
		l.lastNameWidth = nameWidth
		if l.panel != nil {
			l.panel.colNameWidth = nameWidth
			if l.panel.ColumnHeader != nil {
				l.panel.ColumnHeader.Refresh()
			}
		}
		needRefresh = true
	}
	if l.lastAddrWidth != addrWidth {
		l.table.SetColumnWidth(2, addrWidth)
		l.lastAddrWidth = addrWidth
		if l.panel != nil {
			l.panel.colAddrWidth = addrWidth
			if l.panel.ColumnHeader != nil {
				l.panel.ColumnHeader.Refresh()
			}
		}
		needRefresh = true
	}
	if needRefresh {
		l.table.Refresh()
	}

	for _, o := range objects {
		o.Resize(size)
		o.Move(fyne.NewPos(0, 0))
	}
}

func (l *objectListTableLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(450, 200)
}

// objectListHeaderLayout вирівнює заголовки колонок так само, як таблицю.
type objectListHeaderLayout struct {
	panel *ObjectListPanel
}

func (l *objectListHeaderLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if l.panel == nil || len(objects) < 4 {
		for _, o := range objects {
			o.Resize(size)
			o.Move(fyne.NewPos(0, 0))
		}
		return
	}

	w0 := float32(50)
	w3 := float32(80)
	w1 := l.panel.colNameWidth
	w2 := l.panel.colAddrWidth
	if w1 <= 0 {
		w1 = 200
	}
	if w2 <= 0 {
		w2 = 250
	}

	x := float32(0)
	objects[0].Resize(fyne.NewSize(w0, size.Height))
	objects[0].Move(fyne.NewPos(x, 0))
	x += w0

	objects[1].Resize(fyne.NewSize(w1, size.Height))
	objects[1].Move(fyne.NewPos(x, 0))
	x += w1

	objects[2].Resize(fyne.NewSize(w2, size.Height))
	objects[2].Move(fyne.NewPos(x, 0))
	x += w2

	objects[3].Resize(fyne.NewSize(w3, size.Height))
	objects[3].Move(fyne.NewPos(x, 0))
}

func (l *objectListHeaderLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(450, 24)
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
	if p.Table != nil {
		p.Table.Refresh()
	}
}
