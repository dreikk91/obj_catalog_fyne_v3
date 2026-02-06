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

	// Callback при виборі об'єкта
	OnObjectSelected func(object models.Object)
}

// NewObjectListPanel створює панель списку об'єктів
func NewObjectListPanel(provider data.ObjectProvider) *ObjectListPanel {
	panel := &ObjectListPanel{
		Data:          provider,
		CurrentFilter: "Всі",
		SelectedRow:   -1,
	}

	// Заголовок
	panel.TitleText = canvas.NewText("📋 ОБ'ЄКТИ", nil)
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
	panel.FilterSelect = widget.NewSelect([]string{"Всі", "Тривога", "Без зв'язку", "Знято"}, func(selected string) {
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
				cellText = getStatusIcon(item.Status) + " " + item.Name
			case 2:
				cellText = item.Address
			case 3:
				cellText = item.ContractNum
			}
			txt.Text = cellText
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
	panel.Table.SetColumnWidth(0, 50)  // ID
	panel.Table.SetColumnWidth(1, 200) // Назва (буде динамічною)
	panel.Table.SetColumnWidth(2, 180) // Адреса
	panel.Table.SetColumnWidth(3, 80)  // Контракт

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
			&objectListTableLayout{table: panel.Table},
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
		case "Тривога":
			if obj.Status != models.StatusFire && obj.Status != models.StatusFault {
				statusMatch = false
			}
		case "Без зв'язку":
			if !(obj.IsConnState == 0 && obj.GuardState != 0) {
				statusMatch = false
			}
		case "Знято зі спостереження":
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
			fmt.Sprintf("Тривога (%d)", countAlarm),
			fmt.Sprintf("Без зв'язку (%d)", countOffline),
			fmt.Sprintf("Знято зі спостереження (%d)", countDisarmed),
		}

		// Знаходимо поточний вибраний фільтр в оновленому списку, щоб він не зникав
		for _, opt := range p.FilterSelect.Options {
			if strings.HasPrefix(opt, currentFilter+" (") || opt == currentFilter {
				p.FilterSelect.SetSelected(opt)
				break
			}
		}
		p.FilterSelect.Refresh()

		if p.LoadingLabel != nil {
			p.LoadingLabel.Hide()
		}
		if p.Table != nil {
			p.Table.Show()
			p.Table.Refresh()
		}
	})
}

// objectListTableLayout для динамічного ресайзу колонки "Назва"
type objectListTableLayout struct {
	table     *widget.Table
	lastWidth float32
}

func (l *objectListTableLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	// Фіксовані колонки: ID(50) + Адреса(180) + Контракт(80) = 310
	fixedWidth := float32(310)

	// Розраховуємо ширину для колонки "Назва" (індекс 1)
	newWidth := size.Width - fixedWidth - 10 // Буфер під скролл
	if newWidth < 120 {
		newWidth = 120
	}

	if l.lastWidth != newWidth {
		l.table.SetColumnWidth(1, newWidth)
		l.lastWidth = newWidth
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
