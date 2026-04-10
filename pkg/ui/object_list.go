// Package ui - компонент списку об'єктів для лівої панелі
package ui

import (
	"fmt"
	"image/color"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	appTheme "obj_catalog_fyne_v3/pkg/theme"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/usecases"
)

type ObjectListPanel struct {
	Container    *fyne.Container
	Table        *objectListTable
	SearchEntry  *widget.Entry
	SearchClear  *widget.Button
	FilteredData binding.UntypedList
	FilterSelect *widget.Select
	SourceSelect *widget.Select
	Data         contracts.ObjectProvider
	ViewModel    *viewmodels.ObjectListViewModel
	ColumnHeader *fyne.Container

	// Кеш усіх об'єктів
	AllObjects    []models.Object
	FilteredItems []models.Object
	isUpdating    bool
	mutex         sync.RWMutex

	CurrentFilter string
	CurrentSource string
	LoadingLabel  *widget.Label
	SelectedRow   int
	SelectedCol   int
	TitleText     *canvas.Text
	lastFontSize  float32
	colNameWidth  float32
	colAddrWidth  float32
	// Останній об'єкт, про який повідомили через OnObjectSelected.
	// Потрібно, щоб при авто-виборі гарантовано підвантажувати картку,
	// але не викликати завантаження повторно без зміни вибору.
	lastNotifiedSelectedID int
	hasNotifiedSelection   bool
	searchDebounceTimer    *time.Timer
	searchDebounceDelay    time.Duration
	filterRequestVersion   uint64
	runFilterRequestFn     func(version uint64)

	// Callback при виборі об'єкта
	OnObjectSelected func(object models.Object)
}

const defaultObjectListSearchDebounceDelay = 250 * time.Millisecond

type objectListTable struct {
	widget.Table

	onMoveSelection func(delta int) bool
}

func newObjectListTable(
	length func() (rows int, cols int),
	create func() fyne.CanvasObject,
	update func(id widget.TableCellID, obj fyne.CanvasObject),
	onMoveSelection func(delta int) bool,
) *objectListTable {
	table := &objectListTable{
		onMoveSelection: onMoveSelection,
	}
	table.Length = length
	table.CreateCell = create
	table.UpdateCell = update
	table.ExtendBaseWidget(table)
	return table
}

func (t *objectListTable) TypedKey(event *fyne.KeyEvent) {
	if event == nil {
		return
	}

	switch event.Name {
	case fyne.KeyUp:
		if t.onMoveSelection != nil && t.onMoveSelection(-1) {
			return
		}
	case fyne.KeyDown:
		if t.onMoveSelection != nil && t.onMoveSelection(1) {
			return
		}
	}

	t.Table.TypedKey(event)
}

// NewObjectListPanel створює панель списку об'єктів
func NewObjectListPanel(provider contracts.ObjectProvider) *ObjectListPanel {
	panel := &ObjectListPanel{
		Data:                provider,
		ViewModel:           viewmodels.NewObjectListViewModel(),
		FilteredData:        binding.NewUntypedList(),
		CurrentFilter:       viewmodels.FilterAll,
		CurrentSource:       viewmodels.ObjectSourceAll,
		SelectedRow:         -1,
		SelectedCol:         0,
		colNameWidth:        200,
		colAddrWidth:        250,
		searchDebounceDelay: defaultObjectListSearchDebounceDelay,
	}

	// Заголовок
	panel.TitleText = canvas.NewText("ОБ'ЄКТИ", nil)
	themeSize := fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	panel.TitleText.TextSize = themeSize + 1
	panel.TitleText.TextStyle = fyne.TextStyle{Bold: true}

	// Поле пошуку
	panel.SearchEntry = widget.NewEntry()
	panel.SearchEntry.SetPlaceHolder("🔍 Пошук (№, Назва, Адреса, SIM, Тел...)")
	panel.SearchClear = widget.NewButtonWithIcon("", theme.ContentClearIcon(), func() {
		if panel.SearchEntry == nil {
			return
		}
		panel.SearchEntry.SetText("")
		if panel.Table != nil {
			panel.Table.UnselectAll()
		}
	})
	panel.SearchClear.Disable()
	panel.SearchEntry.OnChanged = func(text string) {
		if panel.SearchClear != nil {
			if strings.TrimSpace(text) == "" {
				panel.SearchClear.Disable()
			} else {
				panel.SearchClear.Enable()
			}
		}
		delay := panel.searchDebounceDelay
		if strings.TrimSpace(text) == "" {
			delay = 0
		}
		panel.scheduleFilterApply(delay)
	}

	// Вибір фільтру
	panel.FilterSelect = widget.NewSelect(panel.ViewModel.BuildFilterOptions(0, 0, 0, 0, 0), func(selected string) {
		if panel.isUpdating {
			return
		}
		panel.CurrentFilter = viewmodels.NormalizeObjectListFilter(selected)
		panel.scheduleFilterApply(0)
	})
	panel.FilterSelect.PlaceHolder = "Фільтр"

	panel.SourceSelect = widget.NewSelect(
		viewmodels.BuildObjectSourceOptions(0, 0, 0, 0),
		func(selected string) {
			if panel.isUpdating {
				return
			}
			panel.CurrentSource = viewmodels.NormalizeObjectSourceFilter(selected)
			panel.scheduleFilterApply(0)
		},
	)
	panel.SourceSelect.PlaceHolder = "Джерело"

	// Лейбл завантаження
	panel.LoadingLabel = widget.NewLabel("Завантаження даних...")
	panel.LoadingLabel.Alignment = fyne.TextAlignCenter

	// Таблиця об'єктів (використовує FilteredItems)
	panel.Table = newObjectListTable(
		func() (int, int) {
			if panel.FilteredData != nil {
				return panel.FilteredData.Length(), 4
			}
			return 0, 4
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Cell Data", color.White)
			// Буде оновлено в UpdateCell
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)
			txt.TextStyle.Monospace = true

			item, ok := panel.objectByRow(id.Row)
			if !ok {
				txt.Text = ""
				txt.Refresh()
				bg.Hide()
				bg.Refresh()
				return
			}

			textColor, rowColor := panel.ViewModel.GetRowColors(item, IsDarkMode())

			panel.mutex.RLock()
			selectedRow := panel.SelectedRow
			panel.mutex.RUnlock()
			if id.Row == selectedRow {
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
				cellText = viewmodels.ObjectDisplayNumber(item)
			case 1:
				cellText = fmt.Sprintf("%s %s", viewmodels.SourceBadgeForObjectID(item.ID), item.Name)
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
		panel.moveSelection,
	)

	panel.Table.OnSelected = panel.handleSelection

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
		container.NewBorder(nil, nil, nil, panel.SearchClear, panel.SearchEntry),
		container.NewGridWithColumns(2, panel.FilterSelect, panel.SourceSelect),
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

func (p *ObjectListPanel) handleSelection(id widget.TableCellID) {
	selectedObj, ok := p.objectByRow(id.Row)
	if !ok {
		return
	}

	p.mutex.Lock()
	p.SelectedRow = id.Row
	p.SelectedCol = id.Col
	p.lastNotifiedSelectedID = selectedObj.ID
	p.hasNotifiedSelection = true
	p.mutex.Unlock()

	if p.OnObjectSelected != nil {
		p.OnObjectSelected(selectedObj)
	}
	if p.Table != nil {
		p.Table.Refresh()
	}
}

func (p *ObjectListPanel) RefreshData() {
	if p.Data == nil {
		return
	}
	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewObjectListViewModel()
	}
	useCase := usecases.NewObjectListUseCase(p.Data)
	objects := p.ViewModel.LoadObjects(useCase)

	p.mutex.Lock()
	p.AllObjects = objects
	p.mutex.Unlock()

	p.scheduleFilterApply(0)
}

func (p *ObjectListPanel) applyFilters() {
	p.scheduleFilterApply(0)
}

func (p *ObjectListPanel) scheduleFilterApply(delay time.Duration) {
	if p == nil {
		return
	}

	p.mutex.Lock()
	p.filterRequestVersion++
	version := p.filterRequestVersion
	if p.searchDebounceTimer != nil {
		p.searchDebounceTimer.Stop()
		p.searchDebounceTimer = nil
	}
	p.mutex.Unlock()

	dispatch := func() {
		p.dispatchFilterRequest(version)
	}

	if delay > 0 {
		p.mutex.Lock()
		p.searchDebounceTimer = time.AfterFunc(delay, dispatch)
		p.mutex.Unlock()
		return
	}

	go dispatch()
}

func (p *ObjectListPanel) dispatchFilterRequest(version uint64) {
	if p == nil {
		return
	}
	if p.runFilterRequestFn != nil {
		p.runFilterRequestFn(version)
		return
	}
	p.runFilterRequest(version)
}

func (p *ObjectListPanel) runFilterRequest(version uint64) {
	if p.Table == nil {
		return
	}
	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewObjectListViewModel()
	}

	if !p.isCurrentFilterRequest(version) {
		return
	}

	type filterState struct {
		query         string
		currentFilter string
		currentSource string
	}

	var (
		state filterState
		ok    bool
	)
	fyne.DoAndWait(func() {
		if !p.isCurrentFilterRequest(version) {
			return
		}
		if p.SearchEntry != nil {
			state.query = p.SearchEntry.Text
		}
		state.currentFilter = p.CurrentFilter
		state.currentSource = p.CurrentSource
		ok = true
	})
	if !ok || !p.isCurrentFilterRequest(version) {
		return
	}

	query := strings.ToLower(strings.TrimSpace(state.query))
	currentFilter := state.currentFilter
	currentSource := state.currentSource

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

	result := p.ViewModel.ApplyFilters(viewmodels.ObjectListFilterInput{
		AllObjects:           all,
		Query:                query,
		CurrentFilter:        currentFilter,
		CurrentSource:        currentSource,
		PreviousSelectedID:   prevSelectedID,
		HadPreviousSelection: hadPrevSelection,
		LastNotifiedID:       lastNotifiedID,
		HasNotifiedSelection: hasNotifiedSelection,
	})
	if !p.isCurrentFilterRequest(version) {
		return
	}

	// Оновлюємо список і UI
	p.mutex.Lock()
	p.FilteredItems = result.Filtered
	p.SelectedRow = result.NewSelectedRow
	if result.NewSelectedRow < 0 {
		p.SelectedCol = 0
		p.hasNotifiedSelection = false
		p.lastNotifiedSelectedID = 0
	}
	p.mutex.Unlock()

	fyne.Do(func() {
		if !p.isCurrentFilterRequest(version) {
			return
		}
		p.isUpdating = true
		defer func() { p.isUpdating = false }()

		// Оновлюємо назви фільтрів з кількістю
		filterOptions := p.ViewModel.BuildFilterOptions(
			result.CountAll,
			result.CountAlarm,
			result.CountOffline,
			result.CountMonitoringOff,
			result.CountDebug,
		)
		updateSelectPreservingValue(p.FilterSelect, filterOptions, currentFilter)

		if p.SourceSelect != nil {
			options := viewmodels.BuildObjectSourceOptions(
				result.CountAll,
				result.CountBridge,
				result.CountPhoenix,
				result.CountCASL,
			)
			updateSelectPreservingValue(p.SourceSelect, options, currentSource)
		}

		if p.TitleText != nil {
			p.TitleText.Text = fmt.Sprintf("ОБ'ЄКТИ (%d)", result.CountAll)
			p.TitleText.Refresh()
		}

		if p.LoadingLabel != nil {
			p.LoadingLabel.Hide()
		}
		if p.Table != nil {
			p.Table.Show()
			_ = SetUntypedList(p.FilteredData, result.Filtered)
			p.Table.Refresh()
		}

		if result.ShouldNotifySelection && result.HasSelectedObject && p.OnObjectSelected != nil {
			p.OnObjectSelected(result.SelectedObject)
			p.mutex.Lock()
			p.lastNotifiedSelectedID = result.SelectedObject.ID
			p.hasNotifiedSelection = true
			p.mutex.Unlock()
		}
	})
}

func (p *ObjectListPanel) isCurrentFilterRequest(version uint64) bool {
	if p == nil {
		return false
	}
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return version == p.filterRequestVersion
}

func (p *ObjectListPanel) objectByRow(row int) (models.Object, bool) {
	if p == nil || p.FilteredData == nil || row < 0 || row >= p.FilteredData.Length() {
		return models.Object{}, false
	}
	value, err := p.FilteredData.GetValue(row)
	if err != nil {
		return models.Object{}, false
	}
	obj, ok := value.(models.Object)
	return obj, ok
}

func (p *ObjectListPanel) moveSelection(delta int) bool {
	if p == nil || p.Table == nil || delta == 0 {
		return false
	}

	p.mutex.RLock()
	total := len(p.FilteredItems)
	selectedRow := p.SelectedRow
	selectedCol := p.SelectedCol
	p.mutex.RUnlock()

	if total == 0 {
		return false
	}

	nextRow := selectedRow
	switch {
	case nextRow < 0 || nextRow >= total:
		if delta > 0 {
			nextRow = 0
		} else {
			nextRow = total - 1
		}
	default:
		nextRow += delta
		if nextRow < 0 {
			nextRow = 0
		}
		if nextRow >= total {
			nextRow = total - 1
		}
	}

	if selectedCol < 0 || selectedCol >= 4 {
		selectedCol = 0
	}

	p.Table.Select(widget.TableCellID{Row: nextRow, Col: selectedCol})
	return true
}

// objectListTableLayout для динамічного ресайзу колонок "Назва" та "Адреса"
type objectListTableLayout struct {
	panel         *ObjectListPanel
	table         *objectListTable
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
