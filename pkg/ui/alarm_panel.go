// Package ui - панель активних тривог
package ui

import (
	"image/color"
	"strconv"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/usecases"
	"obj_catalog_fyne_v3/pkg/utils"
)

// AlarmPanelWidget - структура для панелі тривог
type AlarmPanelWidget struct {
	Container     *fyne.Container
	List          *widget.List
	listData      binding.UntypedList
	SourceSelect  *widget.Select
	Data          contracts.DataProvider
	UseCase       *usecases.AlarmListUseCase
	ViewModel     *viewmodels.AlarmListViewModel
	CaseHistoryVM *viewmodels.WorkAreaCaseHistoryViewModel

	// Кеш даних
	AllAlarms            []models.Alarm
	CurrentAlarms        []models.Alarm
	mutex                sync.RWMutex
	isRefreshing         bool
	currentSource        string
	selectedIndex        int
	selectedID           int
	lastClickTime        time.Time
	processBtn           *widget.Button
	lastKnownIDs         map[int]struct{}
	CaseHistoryTitle     *widget.Label
	CaseHistoryAccordion *widget.Accordion
	CaseHistorySection   *fyne.Container
	caseHistoryLoadingID int

	// OnAlarmSelected викликається при кожному кліку по тривозі (одинарному).
	OnAlarmSelected func(alarm models.Alarm)
	// OnAlarmActivated викликається тільки при подвійному кліку по одній і тій самій тривозі.
	OnAlarmActivated func(alarm models.Alarm)

	OnProcessAlarm     func(alarm models.Alarm)
	OnCountsChanged    func(total int, critical int)
	OnNewCriticalAlarm func(alarm models.Alarm)
	TitleText          *canvas.Text
	lastFontSize       float32
	listWidthGuide     *canvas.Rectangle
}

// NewAlarmPanelWidget створює панель тривог
func NewAlarmPanelWidget(provider contracts.DataProvider) *AlarmPanelWidget {
	panel := &AlarmPanelWidget{
		Data:          provider,
		UseCase:       usecases.NewAlarmListUseCase(provider),
		ViewModel:     viewmodels.NewAlarmListViewModel(),
		CaseHistoryVM: viewmodels.NewWorkAreaCaseHistoryViewModel(),
		listData:      binding.NewUntypedList(),
		currentSource: viewmodels.ObjectSourceAll,
		selectedIndex: -1,
		lastKnownIDs:  make(map[int]struct{}),
	}

	// Заголовок
	panel.TitleText = canvas.NewText("🔔 АКТИВНІ ТРИВОГИ", color.White)
	appTheme := fyne.CurrentApp().Settings().Theme()
	panel.TitleText.TextSize = appTheme.Size(theme.SizeNameText) + 1
	panel.TitleText.TextStyle = fyne.TextStyle{Bold: true}

	panel.SourceSelect = widget.NewSelect(
		viewmodels.BuildObjectSourceOptions(0, 0, 0, 0),
		func(selected string) {
			panel.mutex.Lock()
			panel.currentSource = viewmodels.NormalizeObjectSourceFilter(selected)
			panel.mutex.Unlock()
			go panel.Refresh()
		},
	)
	panel.SourceSelect.SetSelected(panel.SourceSelect.Options[0])
	panel.SourceSelect.PlaceHolder = "Джерело"

	titleBg := canvas.NewRectangle(color.NRGBA{R: 100, G: 0, B: 0, A: 255})
	titleContainer := container.NewStack(titleBg, container.NewPadded(panel.TitleText))
	header := container.NewHBox(
		titleContainer,
		layout.NewSpacer(),
		panel.SourceSelect,
	)

	// Список тривог (тепер використовує кеш)
	panel.List = widget.NewListWithData(
		panel.listData,
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Тривога", color.White)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(item binding.DataItem, obj fyne.CanvasObject) {
			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)

			data, ok := item.(binding.Untyped)
			if !ok {
				txt.Text = ""
				bg.FillColor = color.Transparent
				bg.Refresh()
				txt.Refresh()
				return
			}
			value, err := data.Get()
			if err != nil {
				txt.Text = ""
				bg.FillColor = color.Transparent
				bg.Refresh()
				txt.Refresh()
				return
			}
			alarm, ok := value.(models.Alarm)
			if !ok {
				txt.Text = ""
				bg.FillColor = color.Transparent
				bg.Refresh()
				txt.Refresh()
				return
			}

			// Вибираємо палітру кольорів залежно від теми
			var textColor, rowColor color.NRGBA
			if IsDarkMode() {
				textColor, rowColor = utils.SelectColorNRGBADark(alarm.SC1)
			} else {
				textColor, rowColor = utils.SelectColorNRGBA(alarm.SC1)
			}

			// Базовий колір рядка. Для вибраної тривоги додаємо підсвітку.
			rowBg := rowColor
			panel.mutex.RLock()
			isSelected := panel.selectedID > 0 && panel.selectedID == alarm.ID
			panel.mutex.RUnlock()
			if isSelected {
				rowBg = adjustAlarmRowColor(rowColor)
			}
			bg.FillColor = rowBg
			bg.Refresh()

			txt.Color = textColor

			// Для непідготовленого користувача: стабільний читабельний формат рядка.
			// [час] — [тип] — №[об'єкт] [назва] — [зона/деталі]
			if alarm.IsCritical() {
				txt.TextStyle.Bold = true
			} else {
				txt.TextStyle.Bold = false
			}
			txt.Text = formatAlarmListText(alarm)

			if panel.lastFontSize > 0 {
				txt.TextSize = panel.lastFontSize
			} else {
				txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
			}
			txt.Refresh()
		},
	)
	alarmsScroll, alarmsWidthGuide := newHorizontalJournalScroll(panel.List)
	panel.listWidthGuide = alarmsWidthGuide

	panel.List.OnSelected = func(id widget.ListItemID) {
		panel.mutex.Lock()
		valid := int(id) < len(panel.CurrentAlarms)
		if !valid {
			panel.mutex.Unlock()
			return
		}

		now := time.Now()
		prevIndex := panel.selectedIndex
		prevTime := panel.lastClickTime

		selected := panel.CurrentAlarms[id]
		panel.selectedIndex = int(id)
		panel.selectedID = selected.ID
		panel.lastClickTime = now
		panel.mutex.Unlock()

		// Оновлюємо стан кнопки обробки та підсвічування рядка.
		if panel.processBtn != nil {
			panel.processBtn.Enable()
		}
		if panel.List != nil {
			panel.List.Refresh()
		}

		// Одинарний клік: вибираємо об'єкт (оновлюємо картку/контекст без зміни вкладки).
		if panel.OnAlarmSelected != nil {
			panel.OnAlarmSelected(selected)
		}
		panel.loadCaseHistoryForAlarm(selected)

		// Подвійний клік по тому самому елементу в межах інтервалу
		// додатково викликає "активацію" (відкриття деталей).
		if prevIndex == int(id) && !prevTime.IsZero() && now.Sub(prevTime) < 600*time.Millisecond {
			if panel.OnAlarmActivated != nil {
				panel.OnAlarmActivated(selected)
			}
		}
	}

	panel.processBtn = widget.NewButton("Обробити вибрану тривогу", func() {
		panel.mutex.RLock()
		defer panel.mutex.RUnlock()
		if panel.selectedIndex < 0 || panel.selectedIndex >= len(panel.CurrentAlarms) {
			return
		}
		if panel.OnProcessAlarm != nil {
			panel.OnProcessAlarm(panel.CurrentAlarms[panel.selectedIndex])
		}
	})
	panel.processBtn.Disable()

	panel.CaseHistoryTitle = widget.NewLabel("Хронологія вибраної тривоги CASL")
	panel.CaseHistoryTitle.TextStyle = fyne.TextStyle{Bold: true}
	panel.CaseHistoryTitle.Wrapping = fyne.TextWrapWord
	panel.CaseHistoryAccordion = widget.NewAccordion()
	panel.CaseHistoryAccordion.MultiOpen = true
	panel.CaseHistorySection = container.NewVBox(
		widget.NewSeparator(),
		panel.CaseHistoryTitle,
		panel.CaseHistoryAccordion,
	)
	panel.CaseHistorySection.Hide()

	actions := container.NewPadded(container.NewBorder(nil, nil, nil, nil, panel.processBtn))
	body := container.NewBorder(nil, panel.CaseHistorySection, nil, nil, alarmsScroll)

	panel.Container = container.NewBorder(
		header,
		actions, nil, nil,
		body,
	)

	// Перший запуск завантаження
	go panel.Refresh()

	return panel
}

// Refresh оновлює панель асинхронно
func (p *AlarmPanelWidget) Refresh() {
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())

	if p.Data == nil {
		return
	}
	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewAlarmListViewModel()
	}
	p.UseCase = usecases.NewAlarmListUseCase(p.Data)

	p.mutex.Lock()
	if p.isRefreshing {
		p.mutex.Unlock()
		return
	}
	p.isRefreshing = true
	p.mutex.Unlock()

	defer func() {
		p.mutex.Lock()
		p.isRefreshing = false
		p.mutex.Unlock()
	}()

	alarms := p.ViewModel.LoadAlarms(p.UseCase)
	currentSource := viewmodels.ObjectSourceAll
	p.mutex.RLock()
	if strings.TrimSpace(p.currentSource) != "" {
		currentSource = p.currentSource
	}
	p.mutex.RUnlock()

	p.mutex.RLock()
	lastKnown := make(map[int]struct{}, len(p.lastKnownIDs))
	for id := range p.lastKnownIDs {
		lastKnown[id] = struct{}{}
	}
	p.mutex.RUnlock()

	result := p.ViewModel.BuildRefreshOutput(viewmodels.AlarmRefreshInput{
		Alarms:         alarms,
		LastKnownIDs:   lastKnown,
		SelectedSource: currentSource,
	})

	p.mutex.Lock()
	p.AllAlarms = result.CurrentAlarms
	p.CurrentAlarms = result.FilteredAlarms
	p.lastKnownIDs = result.KnownIDs
	selectionCleared := false
	if p.selectedID > 0 {
		found := -1
		for i := range p.CurrentAlarms {
			if p.CurrentAlarms[i].ID == p.selectedID {
				found = i
				break
			}
		}
		if found >= 0 {
			p.selectedIndex = found
		} else {
			p.selectedIndex = -1
			p.selectedID = 0
			p.caseHistoryLoadingID = 0
			selectionCleared = true
		}
	}
	p.mutex.Unlock()

	fyne.Do(func() {
		// Оновлюємо розмір шрифту та UI-елементи теми — тільки на GUI-треді
		p.lastFontSize = uiCfg.FontSizeAlarms
		if p.TitleText != nil {
			p.TitleText.TextSize = uiCfg.FontSizeAlarms + 1
			p.TitleText.Refresh()
		}

		if p.SourceSelect != nil {
			options := viewmodels.BuildObjectSourceOptions(result.CountAll, result.CountBridge, result.CountPhoenix, result.CountCASL)
			p.SourceSelect.Options = options
			target := options[0]
			for _, option := range options {
				if strings.HasPrefix(option, currentSource+" (") || option == currentSource {
					target = option
					break
				}
			}
			handler := p.SourceSelect.OnChanged
			p.SourceSelect.OnChanged = nil
			p.SourceSelect.SetSelected(target)
			p.SourceSelect.OnChanged = handler
			p.SourceSelect.Refresh()
		}

		_ = SetUntypedList(p.listData, result.FilteredAlarms)
		ensureJournalListMinWidth(p.listWidthGuide, alarmListTexts(result.FilteredAlarms), p.lastFontSize, fyne.TextStyle{Bold: true})
		if p.List != nil {
			p.List.Refresh()
		}
		if p.processBtn != nil && p.selectedIndex < 0 {
			p.processBtn.Disable()
		}
		if selectionCleared {
			p.clearCaseHistory()
		}
		if p.OnCountsChanged != nil {
			p.OnCountsChanged(result.Total, result.CriticalCount)
		}
		if result.HasNewCritical && p.OnNewCriticalAlarm != nil {
			p.OnNewCriticalAlarm(result.NewCritical)
		}
	})
}

func (p *AlarmPanelWidget) OnThemeChanged(fontSize float32) {
	p.lastFontSize = fontSize // safe: just a float32, no UI call

	fyne.Do(func() {
		p.mutex.RLock()
		texts := alarmListTexts(p.CurrentAlarms)
		p.mutex.RUnlock()

		if p.TitleText != nil {
			p.TitleText.TextSize = fontSize + 1
			p.TitleText.Refresh()
		}
		if p.List != nil {
			ensureJournalListMinWidth(p.listWidthGuide, texts, fontSize, fyne.TextStyle{Bold: true})
			p.List.Refresh()
		}
		if p.SourceSelect != nil {
			p.SourceSelect.Refresh()
		}
		if p.CaseHistoryTitle != nil {
			p.CaseHistoryTitle.Refresh()
		}
		if p.CaseHistoryAccordion != nil {
			p.CaseHistoryAccordion.Refresh()
		}
	})
}

func formatAlarmListText(alarm models.Alarm) string {
	objNum := alarm.GetObjectNumberDisplay()
	displayText := alarm.GetTimeDisplay() + " — " + alarm.GetTypeDisplay() + " — №" + objNum
	if alarm.ZoneNumber > 0 {
		displayText += "-" + itoa(alarm.ZoneNumber)
	}
	displayText += " " + alarm.ObjectName
	if alarm.Details != "" {
		displayText += " — " + alarm.Details
	}
	return displayText
}

func alarmListTexts(alarms []models.Alarm) []string {
	texts := make([]string, 0, len(alarms))
	for _, alarm := range alarms {
		texts = append(texts, formatAlarmListText(alarm))
	}
	return texts
}

// adjustAlarmRowColor трохи змінює яскравість кольору рядка,
// щоб підсвітити вибраний елемент у списку тривог.
func adjustAlarmRowColor(c color.NRGBA) color.NRGBA {
	const factor = 0.8 // 15% яскравіше
	scale := func(v uint8) uint8 {
		f := float32(v) * factor
		if f > 255 {
			f = 255
		}
		return uint8(f)
	}

	return color.NRGBA{
		R: scale(c.R),
		G: scale(c.G),
		B: scale(c.B),
		A: c.A,
	}
}

func (p *AlarmPanelWidget) loadCaseHistoryForAlarm(alarm models.Alarm) {
	if p == nil || p.CaseHistoryVM == nil || p.Data == nil || !viewmodels.IsCASLObjectID(alarm.ObjectID) {
		p.clearCaseHistory()
		return
	}

	p.mutex.Lock()
	p.caseHistoryLoadingID = alarm.ID
	p.mutex.Unlock()

	fyne.Do(func() {
		p.showCaseHistoryLoading(alarm)
	})

	go func(selected models.Alarm) {
		events := p.Data.GetObjectEvents(strconv.Itoa(selected.ObjectID))
		object := &models.Object{ID: selected.ObjectID, Name: selected.ObjectName}
		group, ok := p.CaseHistoryVM.FindGroupForAlarm(object, selected, events)

		fyne.Do(func() {
			p.mutex.RLock()
			stillSelected := p.selectedID == selected.ID && p.caseHistoryLoadingID == selected.ID
			p.mutex.RUnlock()
			if !stillSelected {
				return
			}
			if !ok {
				p.showEmptyCaseHistory(selected)
				return
			}
			p.showCaseHistoryGroup(selected, group)
		})
	}(alarm)
}

func (p *AlarmPanelWidget) showCaseHistoryLoading(alarm models.Alarm) {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	if p.CaseHistoryTitle != nil {
		p.CaseHistoryTitle.SetText("CASL: завантаження хронології для №" + alarm.GetObjectNumberDisplay())
	}
	loading := widget.NewProgressBarInfinite()
	loading.Start()
	p.CaseHistoryAccordion.Items = []*widget.AccordionItem{
		widget.NewAccordionItem("Завантаження подій кейсу", container.NewPadded(loading)),
	}
	p.CaseHistoryAccordion.Open(0)
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Show()
}

func (p *AlarmPanelWidget) showEmptyCaseHistory(alarm models.Alarm) {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	if p.CaseHistoryTitle != nil {
		title := "CASL: №" + alarm.GetObjectNumberDisplay()
		if name := strings.TrimSpace(alarm.ObjectName); name != "" {
			title += " " + name
		}
		p.CaseHistoryTitle.SetText(title)
	}
	label := widget.NewLabel("Для цієї тривоги не вдалося знайти деталізовану хронологію подій.")
	label.Wrapping = fyne.TextWrapWord
	p.CaseHistoryAccordion.Items = []*widget.AccordionItem{
		widget.NewAccordionItem("Хронологія недоступна", container.NewPadded(label)),
	}
	p.CaseHistoryAccordion.Open(0)
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Show()
}

func (p *AlarmPanelWidget) showCaseHistoryGroup(alarm models.Alarm, group viewmodels.WorkAreaCaseHistoryGroup) {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	if p.CaseHistoryTitle != nil {
		title := "CASL: №" + alarm.GetObjectNumberDisplay()
		if name := strings.TrimSpace(alarm.ObjectName); name != "" {
			title += " " + name
		}
		if summary := strings.TrimSpace(group.Title); summary != "" {
			title += " | " + summary
		}
		p.CaseHistoryTitle.SetText(title)
	}

	p.CaseHistoryAccordion.Items = []*widget.AccordionItem{
		widget.NewAccordionItem(
			group.Title,
			buildCaseHistoryEventList(group),
		),
	}
	p.CaseHistoryAccordion.Open(0)
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Show()
}

func (p *AlarmPanelWidget) clearCaseHistory() {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	if p.CaseHistoryTitle != nil {
		p.CaseHistoryTitle.SetText("Хронологія вибраної тривоги CASL")
	}
	p.CaseHistoryAccordion.Items = nil
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Hide()
}
