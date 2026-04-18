// Package ui - панель активних тривог
package ui

import (
	"fmt"
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
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/usecases"
)

// AlarmPanelWidget - структура для панелі тривог
type AlarmPanelWidget struct {
	Container     *fyne.Container
	List          *widget.List
	listData      binding.UntypedList
	SourceSelect  *widget.Select
	Data          contracts.DataProvider
	ViewModel     *viewmodels.AlarmListViewModel
	CaseHistoryVM *viewmodels.WorkAreaCaseHistoryViewModel

	// Кеш даних
	AllAlarms             []models.Alarm
	CurrentAlarms         []models.Alarm
	mutex                 sync.RWMutex
	isRefreshing          bool
	currentSource         string
	selectedIndex         int
	selectedID            int
	lastClickTime         time.Time
	processBtn            *widget.Button
	lastKnownIDs          map[int]struct{}
	CaseHistoryTitle      *widget.Label
	CaseHistoryAccordion  *widget.Accordion
	CaseHistorySection    *fyne.Container
	caseHistoryLoadingID  int
	caseHistoryAlarm      models.Alarm
	caseHistoryGroup      viewmodels.WorkAreaCaseHistoryGroup
	hasCaseHistoryGroup   bool
	caseHistorySourceMsgs []models.AlarmMsg
	hasCaseHistorySource  bool

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
			panel.Refresh()
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

			textColor, rowColor := eventRowColorsBySeverity(alarm.VisualSeverityValue(), alarm.SC1)

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

	panel.CaseHistoryTitle = widget.NewLabel("Хронологія вибраної тривоги")
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
	panel.Refresh()

	return panel
}

// Refresh оновлює панель асинхронно
func (p *AlarmPanelWidget) Refresh() {
	go p.refreshData()
}

func (p *AlarmPanelWidget) refreshData() {
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())

	if p.Data == nil {
		return
	}
	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewAlarmListViewModel()
	}
	useCase := usecases.NewAlarmListUseCase(p.Data)

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

	alarms := p.ViewModel.LoadAlarms(useCase)
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
			updateSelectPreservingValue(p.SourceSelect, options, currentSource)
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
		alarm := p.caseHistoryAlarm
		group := p.caseHistoryGroup
		hasGroup := p.hasCaseHistoryGroup
		sourceMsgs := append([]models.AlarmMsg(nil), p.caseHistorySourceMsgs...)
		hasSource := p.hasCaseHistorySource
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
		if hasSource {
			p.showCaseHistorySourceMessages(alarm, sourceMsgs)
			return
		}
		if hasGroup {
			p.showCaseHistoryGroup(alarm, group)
		}
	})
}

func formatAlarmListText(alarm models.Alarm) string {
	objNum := alarm.GetObjectNumberDisplay()
	displayText := alarm.GetTimeDisplay() + " — " + alarm.GetTypeDisplay() + " — №" + objNum
	if alarm.ZoneNumber > 0 {
		displayText += "-" + strconv.Itoa(alarm.ZoneNumber)
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
	if p == nil || p.CaseHistoryVM == nil || p.Data == nil {
		p.clearCaseHistory()
		return
	}

	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	useBridgeActiveHistory := !ids.IsCASLObjectID(alarm.ObjectID) &&
		!ids.IsPhoenixObjectID(alarm.ObjectID) &&
		uiCfg.NormalizedBridgeAlarmHistoryMode() == config.BridgeAlarmHistoryModeActiveOnly

	if useBridgeActiveHistory {
		if historyProvider, ok := p.Data.(contracts.ActiveAlarmHistoryProvider); ok {
			p.mutex.Lock()
			p.caseHistoryLoadingID = alarm.ID
			p.mutex.Unlock()

			fyne.Do(func() {
				p.showCaseHistoryLoading(alarm)
			})

			go func(selected models.Alarm) {
				msgs := historyProvider.GetActiveAlarmSourceMessages(selected)

				fyne.Do(func() {
					p.mutex.RLock()
					stillSelected := p.selectedID == selected.ID && p.caseHistoryLoadingID == selected.ID
					p.mutex.RUnlock()
					if !stillSelected {
						return
					}
					if len(msgs) == 0 {
						p.showEmptyCaseHistory(selected)
						return
					}
					p.showCaseHistorySourceMessages(selected, msgs)
				})
			}(alarm)
			return
		}
	}

	if len(alarm.SourceMsgs) > 0 {
		p.showCaseHistorySourceMessages(alarm, alarm.SourceMsgs)
		return
	}

	if historyProvider, ok := p.Data.(contracts.AlarmHistoryProvider); ok && !ids.IsCASLObjectID(alarm.ObjectID) {
		p.mutex.Lock()
		p.caseHistoryLoadingID = alarm.ID
		p.mutex.Unlock()

		fyne.Do(func() {
			p.showCaseHistoryLoading(alarm)
		})

		go func(selected models.Alarm) {
			msgs := historyProvider.GetAlarmSourceMessages(selected)

			fyne.Do(func() {
				p.mutex.RLock()
				stillSelected := p.selectedID == selected.ID && p.caseHistoryLoadingID == selected.ID
				p.mutex.RUnlock()
				if !stillSelected {
					return
				}
				if len(msgs) == 0 {
					p.showEmptyCaseHistory(selected)
					return
				}
				p.showCaseHistorySourceMessages(selected, msgs)
			})
		}(alarm)
		return
	}

	if !ids.IsCASLObjectID(alarm.ObjectID) {
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

	p.mutex.Lock()
	p.caseHistoryAlarm = alarm
	p.hasCaseHistoryGroup = false
	p.caseHistorySourceMsgs = nil
	p.hasCaseHistorySource = false
	p.mutex.Unlock()

	if p.CaseHistoryTitle != nil {
		p.CaseHistoryTitle.SetText(alarmSourceDisplayName(alarm.ObjectID) + ": завантаження хронології для №" + alarm.GetObjectNumberDisplay())
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

	p.mutex.Lock()
	p.caseHistoryAlarm = alarm
	p.hasCaseHistoryGroup = false
	p.caseHistorySourceMsgs = nil
	p.hasCaseHistorySource = false
	p.mutex.Unlock()

	if p.CaseHistoryTitle != nil {
		title := alarmSourceDisplayName(alarm.ObjectID) + ": №" + alarm.GetObjectNumberDisplay()
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

	p.mutex.Lock()
	p.caseHistoryAlarm = alarm
	p.caseHistoryGroup = group
	p.hasCaseHistoryGroup = true
	p.caseHistorySourceMsgs = nil
	p.hasCaseHistorySource = false
	p.mutex.Unlock()

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
	fyne.Do(func() {
		if len(p.CaseHistoryAccordion.Items) == 0 {
			return
		}
		scrollCaseHistoryToBottom(p.CaseHistoryAccordion.Items[0].Detail)
	})
}

func (p *AlarmPanelWidget) showCaseHistorySourceMessages(alarm models.Alarm, sourceMsgs []models.AlarmMsg) {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	msgs := prepareSourceMessagesForDisplay(alarm, sourceMsgs, uiCfg.BridgeAlarmHistoryMode)
	if len(msgs) == 0 {
		p.clearCaseHistory()
		return
	}

	p.mutex.Lock()
	p.caseHistoryAlarm = alarm
	p.caseHistoryGroup = viewmodels.WorkAreaCaseHistoryGroup{}
	p.hasCaseHistoryGroup = false
	p.caseHistorySourceMsgs = append([]models.AlarmMsg(nil), msgs...)
	p.hasCaseHistorySource = true
	p.mutex.Unlock()

	sourceTitle := alarmSourceDisplayName(alarm.ObjectID)
	if p.CaseHistoryTitle != nil {
		title := sourceTitle + ": №" + alarm.GetObjectNumberDisplay()
		if name := strings.TrimSpace(alarm.ObjectName); name != "" {
			title += " " + name
		}
		p.CaseHistoryTitle.SetText(title)
	}

	itemTitle := fmt.Sprintf("Хронологія подій (%d)", len(msgs))
	p.CaseHistoryAccordion.Items = []*widget.AccordionItem{
		widget.NewAccordionItem(itemTitle, buildAlarmSourceMessagesList(msgs)),
	}
	p.CaseHistoryAccordion.Open(0)
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Show()
}

func filterAlarmSourceMessagesSince(alarm models.Alarm, sourceMsgs []models.AlarmMsg) []models.AlarmMsg {
	if len(sourceMsgs) == 0 {
		return nil
	}

	msgs := append([]models.AlarmMsg(nil), sourceMsgs...)
	if alarm.Time.IsZero() {
		return msgs
	}

	filtered := make([]models.AlarmMsg, 0, len(msgs))
	for _, msg := range msgs {
		if !msg.Time.IsZero() && msg.Time.Before(alarm.Time) {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered
}

func prepareSourceMessagesForDisplay(alarm models.Alarm, sourceMsgs []models.AlarmMsg, bridgeHistoryMode string) []models.AlarmMsg {
	if len(sourceMsgs) == 0 {
		return nil
	}

	msgs := append([]models.AlarmMsg(nil), sourceMsgs...)
	if ids.IsCASLObjectID(alarm.ObjectID) {
		return msgs
	}
	if !ids.IsCASLObjectID(alarm.ObjectID) &&
		!ids.IsPhoenixObjectID(alarm.ObjectID) &&
		config.NormalizeBridgeAlarmHistoryMode(bridgeHistoryMode) == config.BridgeAlarmHistoryModeActiveOnly {
		return msgs
	}

	return filterAlarmSourceMessagesSince(alarm, msgs)
}

func alarmSourceDisplayName(objectID int) string {
	switch {
	case ids.IsCASLObjectID(objectID):
		return "CASL"
	case ids.IsPhoenixObjectID(objectID):
		return "Phoenix"
	default:
		return "БД/МІСТ"
	}
}

func buildAlarmSourceMessagesList(messages []models.AlarmMsg) fyne.CanvasObject {
	if len(messages) == 0 {
		label := widget.NewLabel("Події для цього кейсу відсутні.")
		label.Wrapping = fyne.TextWrapWord
		return container.NewPadded(label)
	}

	msgs := append([]models.AlarmMsg(nil), messages...)
	rowsText := make([]string, len(msgs))
	for i, msg := range msgs {
		rowsText[i] = formatAlarmSourceMessageText(msg)
	}

	list := widget.NewList(
		func() int {
			return len(msgs)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("", color.White)
			txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || int(id) >= len(msgs) {
				return
			}

			stack := obj.(*fyne.Container)
			bg := stack.Objects[0].(*canvas.Rectangle)
			txtContainer := stack.Objects[1].(*fyne.Container)
			txt := txtContainer.Objects[0].(*canvas.Text)

			msg := msgs[id]
			textColor, rowColor := eventRowColors(alarmSourceMessageSC1(msg))

			bg.FillColor = rowColor
			bg.Refresh()

			txt.Color = textColor
			txt.TextStyle = fyne.TextStyle{Bold: msg.IsAlarm}
			txt.Text = rowsText[id]
			txt.Refresh()
		},
	)

	listHeight := float32(len(msgs)) * 40
	if len(msgs) > caseHistoryVisibleEventRows {
		listHeight = 220
	}
	if listHeight < 80 {
		listHeight = 80
	}

	widthGuide := canvas.NewRectangle(color.Transparent)
	widthGuide.SetMinSize(fyne.NewSize(journalListMinWidth, 1))
	heightGuide := canvas.NewRectangle(color.Transparent)
	heightGuide.SetMinSize(fyne.NewSize(1, listHeight))

	hScroll := container.NewHScroll(container.NewStack(list, widthGuide, heightGuide))
	ensureJournalListMinWidth(widthGuide, rowsText, fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText), fyne.TextStyle{Bold: true})
	return hScroll
}

func formatAlarmSourceMessageText(msg models.AlarmMsg) string {
	text := "—"
	if !msg.Time.IsZero() {
		text = msg.Time.Local().Format("02.01.2006 15:04:05")
	}

	state := "Подія"
	if msg.IsAlarm {
		state = "Тривога"
	}
	text += " | " + state

	if msg.Number > 0 {
		text += " | Зона " + strconv.Itoa(msg.Number)
	}

	details := strings.TrimSpace(msg.Details)
	code := strings.TrimSpace(msg.Code)
	contactID := strings.TrimSpace(msg.ContactID)
	switch {
	case details != "":
		text += " — " + details
	case code != "":
		text += " — " + code
	case contactID != "":
		text += " — " + contactID
	}

	if code != "" && details != "" {
		text += " [code=" + code + "]"
	}
	if contactID != "" && details != "" {
		text += " [cid=" + contactID + "]"
	}
	return text
}

func alarmSourceMessageSC1(msg models.AlarmMsg) int {
	sc1 := msg.SC1
	if sc1 == 0 {
		if msg.IsAlarm {
			sc1 = 1
		} else {
			sc1 = 6
		}
	}
	return sc1
}

func (p *AlarmPanelWidget) clearCaseHistory() {
	if p == nil || p.CaseHistorySection == nil || p.CaseHistoryAccordion == nil {
		return
	}

	p.mutex.Lock()
	p.caseHistoryLoadingID = 0
	p.caseHistoryAlarm = models.Alarm{}
	p.caseHistoryGroup = viewmodels.WorkAreaCaseHistoryGroup{}
	p.hasCaseHistoryGroup = false
	p.caseHistorySourceMsgs = nil
	p.hasCaseHistorySource = false
	p.mutex.Unlock()

	if p.CaseHistoryTitle != nil {
		p.CaseHistoryTitle.SetText("Хронологія вибраної тривоги")
	}
	p.CaseHistoryAccordion.Items = nil
	p.CaseHistoryAccordion.Refresh()
	p.CaseHistorySection.Hide()
}

func (p *AlarmPanelWidget) ReloadSelectedCaseHistory() {
	if p == nil {
		return
	}

	p.mutex.RLock()
	if p.selectedIndex < 0 || p.selectedIndex >= len(p.CurrentAlarms) {
		p.mutex.RUnlock()
		return
	}
	selected := p.CurrentAlarms[p.selectedIndex]
	p.mutex.RUnlock()

	p.loadCaseHistoryForAlarm(selected)
}
