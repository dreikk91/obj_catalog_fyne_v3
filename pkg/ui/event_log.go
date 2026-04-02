// Package ui - глобальний журнал подій
package ui

import (
	"image/color"
	"obj_catalog_fyne_v3/pkg/config"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"obj_catalog_fyne_v3/pkg/usecases"
	"obj_catalog_fyne_v3/pkg/utils"
)

// EventLogPanel - структура журналу подій
type EventLogPanel struct {
	Container       *fyne.Container
	List            *widget.List
	listData        binding.UntypedList
	Data            contracts.EventProvider
	UseCase         *usecases.EventLogUseCase
	ViewModel       *viewmodels.EventLogViewModel
	IsPaused        bool
	PauseBtn        *widget.Button
	RangeSelect     *widget.Select
	SourceSelect    *widget.Select
	ImportantOnly   *widget.Check
	OnEventSelected func(models.Event)
	OnCountChanged  func(count int)

	// Кеш даних
	AllEvents      []models.Event
	FilteredEvents []models.Event
	mutex          sync.RWMutex
	isRefreshing   bool
	TitleText      *canvas.Text
	lastFontSize   float32

	// Поточний об'єкт для контекстного відображення подій
	currentObject *models.Object
	// Перемикач режиму: всі події чи тільки по вибраному об'єкту
	showForCurrentOnly bool
}

// NewEventLogPanel створює панель журналу подій
func NewEventLogPanel(provider contracts.EventProvider) *EventLogPanel {
	panel := &EventLogPanel{
		Data:      provider,
		listData:  binding.NewUntypedList(),
		UseCase:   usecases.NewEventLogUseCase(provider),
		ViewModel: viewmodels.NewEventLogViewModel(),
		IsPaused:  false,
	}

	// Заголовок
	panel.TitleText = canvas.NewText("📜 ЖУРНАЛ ПОДІЙ", nil)
	themeSize := fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
	panel.TitleText.TextSize = themeSize + 1
	panel.TitleText.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка паузи
	panel.PauseBtn = widget.NewButton("⏸ Пауза", func() {
		panel.IsPaused = !panel.IsPaused
		if panel.IsPaused {
			panel.PauseBtn.SetText("▶ Продовжити")
		} else {
			panel.PauseBtn.SetText("⏸ Пауза")
		}
	})

	panel.RangeSelect = widget.NewSelect([]string{"Остання година", "Сьогодні", "Всі"}, func(string) {
		panel.applyFilters()
	})
	panel.RangeSelect.SetSelected("Остання година")
	panel.RangeSelect.PlaceHolder = "Період"

	panel.SourceSelect = widget.NewSelect(
		viewmodels.BuildObjectSourceOptions(0, 0, 0),
		func(string) {
			panel.applyFilters()
		},
	)
	panel.SourceSelect.SetSelected(panel.SourceSelect.Options[0])
	panel.SourceSelect.PlaceHolder = "Джерело"

	panel.ImportantOnly = widget.NewCheck("Тільки важливі", func(bool) {
		panel.applyFilters()
	})

	// Перемикач контексту: всі події / по вибраному об'єкту
	contextToggle := widget.NewCheck("Тільки по вибраному об'єкту", func(checked bool) {
		panel.showForCurrentOnly = checked
		panel.applyFilters()
	})

	header := container.NewHBox(
		container.NewPadded(panel.TitleText),
		layout.NewSpacer(),
		contextToggle,
		panel.SourceSelect,
		panel.RangeSelect,
		panel.ImportantOnly,
		panel.PauseBtn,
	)

	// Список подій (тепер використовує кеш)
	panel.List = widget.NewListWithData(
		panel.listData,
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Подія", color.White)
			// Буде оновлено в UpdateCell
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
			event, ok := value.(models.Event)
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
				textColor, rowColor = utils.SelectColorNRGBADark(event.SC1)
			} else {
				textColor, rowColor = utils.SelectColorNRGBA(event.SC1)
			}

			bg.FillColor = rowColor
			bg.Refresh()

			txt.Color = textColor

			// Для непідготовленого користувача: стабільний читабельний формат рядка.
			// [дата/час] — [номер | назва об'єкта] — [тип] — [зона/деталі]
			objectName := strings.TrimSpace(event.ObjectName)
			if objectName == "" {
				objectName = "Об'єкт"
			}

			objectNum := event.ObjectNumber
			if objectNum == "" && panel.Data != nil {
				objectNum = panel.Data.GetDisplayNumber(event.ObjectID)
			}

			displayTitle := objectName
			if objectNum != "" && !strings.HasPrefix(objectName, objectNum) {
				displayTitle = objectNum + " | " + objectName
			}

			text := event.GetDateTimeDisplay() + " — " + displayTitle + " — " + event.GetTypeDisplay()
			if event.ZoneNumber > 0 {
				text += " — Зона " + itoa(event.ZoneNumber)
			}
			if event.Details != "" {
				text += " — " + event.Details
			}
			txt.Text = text
			if panel.lastFontSize > 0 {
				txt.TextSize = panel.lastFontSize
			} else {
				txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
			}
			txt.Refresh()
		},
	)

	panel.List.OnSelected = func(id widget.ListItemID) {
		// Забираємо подію під read-lock, а колбек викликаємо вже без блокування,
		// щоб уникнути дедлоку при SetCurrentObject (який використовує write-lock).
		panel.mutex.RLock()
		var ev models.Event
		valid := int(id) < len(panel.FilteredEvents)
		if valid {
			ev = panel.FilteredEvents[id]
		}
		panel.mutex.RUnlock()

		if valid && panel.OnEventSelected != nil {
			panel.OnEventSelected(ev)
		}
		if panel.List != nil {
			panel.List.Unselect(id)
		}
	}

	panel.Container = container.NewBorder(
		header,
		nil, nil, nil,
		panel.List,
	)

	// Перший запуск завантаження
	go panel.Refresh()

	return panel
}

// Refresh оновлює журнал асинхронно
func (p *EventLogPanel) Refresh() {
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	p.OnThemeChanged(uiCfg.FontSizeEvents)
	if p.Data == nil || p.IsPaused || p.isRefreshing {
		return
	}

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

	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewEventLogViewModel()
	}
	// Джерело даних може змінюватися (наприклад, після Reconnect), тому use case перевизначаємо.
	p.UseCase = usecases.NewEventLogUseCase(p.Data)
	events := p.ViewModel.LoadEvents(p.UseCase)

	// Оновлюємо кеш
	p.mutex.Lock()
	p.AllEvents = events
	p.mutex.Unlock()

	// Фільтруємо для відображення
	p.applyFilters()

	// Оновлюємо UI у головному вікні
	fyne.Do(func() {
		if p.List != nil {
			p.List.Refresh()
		}
	})
}

func (p *EventLogPanel) applyFilters() {
	if p.ViewModel == nil {
		p.ViewModel = viewmodels.NewEventLogViewModel()
	}

	p.mutex.RLock()
	all := p.AllEvents
	currentObj := p.currentObject
	showForCurrentOnly := p.showForCurrentOnly
	p.mutex.RUnlock()

	period := ""
	if p.RangeSelect != nil {
		period = p.RangeSelect.Selected
	}
	importantOnly := false
	if p.ImportantOnly != nil {
		importantOnly = p.ImportantOnly.Checked
	}
	selectedSource := viewmodels.ObjectSourceAll
	if p.SourceSelect != nil {
		selectedSource = viewmodels.NormalizeObjectSourceFilter(p.SourceSelect.Selected)
	}

	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	input := viewmodels.EventLogFilterInput{
		AllEvents:          all,
		Period:             period,
		SelectedSource:     selectedSource,
		ImportantOnly:      importantOnly,
		ShowForCurrentOnly: showForCurrentOnly,
		MaxEvents:          uiCfg.EventLogLimit,
	}
	if currentObj != nil {
		input.CurrentObjectID = currentObj.ID
		input.HasCurrentObject = true
	}
	out := p.ViewModel.ApplyFilters(input)

	p.mutex.Lock()
	p.FilteredEvents = out.Filtered
	p.mutex.Unlock()

	fyne.Do(func() {
		if p.SourceSelect != nil {
			options := viewmodels.BuildObjectSourceOptions(out.CountAll, out.CountBridge, out.CountCASL)
			p.SourceSelect.Options = options

			target := options[0]
			for _, option := range options {
				if strings.HasPrefix(option, selectedSource+" (") || option == selectedSource {
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

		_ = SetUntypedList(p.listData, out.Filtered)
		if p.OnCountChanged != nil {
			p.OnCountChanged(out.Count)
		}
		if p.List != nil {
			p.List.Refresh()
		}
	})
}

// Решта функцій (getEventIcon, getEventImportance) залишаються незмінними (вони в тому ж файлі були?)
// Так, вони були в кінці файлу. Я їх додам сюди для цілісності.

func getEventIcon(eventType models.EventType) string {
	switch eventType {
	case models.EventFire:
		return "🔴"
	case models.EventBurglary:
		return "🚨"
	case models.EventPanic:
		return "🆘"
	case models.EventMedical:
		return "🩺"
	case models.EventGas:
		return "☣"
	case models.EventTamper:
		return "🔧"
	case models.EventAlarmNotification:
		return "📥"
	case models.EventOperatorAction:
		return "👤"
	case models.EventManagerAssigned:
		return "🚓"
	case models.EventManagerArrived:
		return "✅"
	case models.EventManagerCanceled:
		return "↩"
	case models.EventAlarmFinished:
		return "✔"
	case models.EventDeviceBlocked:
		return "⛔"
	case models.EventDeviceUnblocked:
		return "🔓"
	case models.EventService, models.SystemEvent:
		return "🛠"
	case models.EventFault, models.EventOffline, models.EventPowerFail, models.EventBatteryLow:
		return "🟡"
	case models.EventArm, models.EventDisarm:
		return "🔵"
	case models.EventRestore, models.EventOnline, models.EventPowerOK:
		return "🟢"
	default:
		return "⚪"
	}
}

func (p *EventLogPanel) OnThemeChanged(fontSize float32) {
	p.lastFontSize = fontSize
	if p.TitleText != nil {
		p.TitleText.TextSize = fontSize + 1
		p.TitleText.Refresh()
	}
	if p.List != nil {
		p.List.Refresh()
	}
	if p.RangeSelect != nil {
		p.RangeSelect.Refresh()
	}
	if p.SourceSelect != nil {
		p.SourceSelect.Refresh()
	}
	if p.ImportantOnly != nil {
		p.ImportantOnly.Refresh()
	}
}

// SetCurrentObject встановлює поточний об'єкт для контекстного журналу.
// При зміні об'єкта фільтрація перераховується автоматично.
func (p *EventLogPanel) SetCurrentObject(obj *models.Object) {
	p.mutex.Lock()
	p.currentObject = obj
	p.mutex.Unlock()
	p.applyFilters()
}

func getEventImportance(event models.Event) widget.Importance {
	if event.IsCritical() {
		return widget.DangerImportance
	}
	if event.IsWarning() {
		return widget.WarningImportance
	}
	return widget.MediumImportance
}
