// Package ui - панель активних тривог
package ui

import (
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

// AlarmPanelWidget - структура для панелі тривог
type AlarmPanelWidget struct {
	Container *fyne.Container
	List      *widget.List
	Data      data.AlarmProvider

	// Кеш даних
	CurrentAlarms []models.Alarm
	mutex         sync.RWMutex
	isRefreshing  bool
	selectedIndex int
	lastClickTime time.Time
	processBtn    *widget.Button
	lastKnownIDs  map[int]struct{}

	// OnAlarmSelected викликається при кожному кліку по тривозі (одинарному).
	OnAlarmSelected func(alarm models.Alarm)
	// OnAlarmActivated викликається тільки при подвійному кліку по одній і тій самій тривозі.
	OnAlarmActivated func(alarm models.Alarm)

	OnProcessAlarm     func(alarm models.Alarm)
	OnCountsChanged    func(total int, fire int)
	OnNewCriticalAlarm func(alarm models.Alarm)
	TitleText          *canvas.Text
	lastFontSize       float32
}

// NewAlarmPanelWidget створює панель тривог
func NewAlarmPanelWidget(provider data.AlarmProvider) *AlarmPanelWidget {
	panel := &AlarmPanelWidget{
		Data:          provider,
		selectedIndex: -1,
		lastKnownIDs:  make(map[int]struct{}),
	}

	// Заголовок
	panel.TitleText = canvas.NewText("🔔 АКТИВНІ ТРИВОГИ", color.White)
	appTheme := fyne.CurrentApp().Settings().Theme()
	panel.TitleText.TextSize = appTheme.Size(theme.SizeNameText) + 1
	panel.TitleText.TextStyle = fyne.TextStyle{Bold: true}

	titleBg := canvas.NewRectangle(color.NRGBA{R: 100, G: 0, B: 0, A: 255})
	titleContainer := container.NewStack(titleBg, container.NewPadded(panel.TitleText))

	// Список тривог (тепер використовує кеш)
	panel.List = widget.NewList(
		func() int {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()
			return len(panel.CurrentAlarms)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Тривога", color.White)
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()

			if id < len(panel.CurrentAlarms) {
				stack := obj.(*fyne.Container)
				bg := stack.Objects[0].(*canvas.Rectangle)
				txtContainer := stack.Objects[1].(*fyne.Container)
				txt := txtContainer.Objects[0].(*canvas.Text)

				alarm := panel.CurrentAlarms[id]

				// Вибираємо палітру кольорів залежно від теми
				var textColor, rowColor color.NRGBA
				if IsDarkMode() {
					textColor, rowColor = utils.SelectColorNRGBADark(alarm.SC1)
				} else {
					textColor, rowColor = utils.SelectColorNRGBA(alarm.SC1)
				}

				// Базовий колір рядка
				rowBg := rowColor
				// Якщо рядок вибраний — робимо підсвічування трохи яскравішим/темнішим,
				// щоб користувач чітко бачив поточний вибір.
				if int(id) == panel.selectedIndex {
					rowBg = adjustAlarmRowColor(rowColor)
				}
				bg.FillColor = rowBg
				bg.Refresh()

				txt.Color = textColor

				// Для непідготовленого користувача: стабільний читабельний формат рядка.
				// [час] — [тип] — №[об'єкт] [назва] — [зона/деталі]
				if alarm.Type == models.AlarmFire {
					txt.TextStyle.Bold = true
				} else {
					txt.TextStyle.Bold = false
				}
				displayText := alarm.GetTimeDisplay() + " — " + alarm.GetTypeDisplay() + " — №" + itoa(alarm.ObjectID)
				if alarm.ZoneNumber > 0 {
					displayText += "-" + itoa(alarm.ZoneNumber)
				}
				displayText += " " + alarm.ObjectName
				if alarm.Details != "" {
					displayText += " — " + alarm.Details
				}
				txt.Text = displayText

				if panel.lastFontSize > 0 {
					txt.TextSize = panel.lastFontSize
				} else {
					txt.TextSize = fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText)
				}
				txt.Refresh()
			}
		},
	)

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

		panel.selectedIndex = int(id)
		panel.lastClickTime = now
		selected := panel.CurrentAlarms[id]
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

	actions := container.NewPadded(container.NewBorder(nil, nil, nil, nil, panel.processBtn))

	panel.Container = container.NewBorder(
		titleContainer,
		actions, nil, nil,
		panel.List,
	)

	// Перший запуск завантаження
	go panel.Refresh()

	return panel
}

// Refresh оновлює панель асинхронно
func (p *AlarmPanelWidget) Refresh() {
	uiCfg := config.LoadUIConfig(fyne.CurrentApp().Preferences())
	p.OnThemeChanged(uiCfg.FontSizeAlarms)
	if p.Data == nil {
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

	// Отримуємо дані з БД (може бути довго)
	alarms := p.Data.GetAlarms()

	// Порахуємо лічильники та визначимо "нові критичні" тривоги.
	total := len(alarms)
	fireCount := 0
	var newCritical *models.Alarm
	for i := range alarms {
		if alarms[i].Type == models.AlarmFire && !alarms[i].IsProcessed {
			fireCount++
		}
		if _, ok := p.lastKnownIDs[alarms[i].ID]; !ok {
			// Вважаємо критичною в першу чергу пожежу.
			if newCritical == nil && alarms[i].Type == models.AlarmFire && !alarms[i].IsProcessed {
				newCritical = &alarms[i]
			}
		}
	}

	// Оновлюємо кеш та UI
	p.mutex.Lock()
	p.CurrentAlarms = alarms
	// Оновлюємо набір відомих ID
	p.lastKnownIDs = make(map[int]struct{}, len(alarms))
	for i := range alarms {
		p.lastKnownIDs[alarms[i].ID] = struct{}{}
	}
	p.mutex.Unlock()

	fyne.Do(func() {
		if p.List != nil {
			p.List.Refresh()
		}
		if p.OnCountsChanged != nil {
			p.OnCountsChanged(total, fireCount)
		}
		if newCritical != nil && p.OnNewCriticalAlarm != nil {
			p.OnNewCriticalAlarm(*newCritical)
		}
	})
}

func (p *AlarmPanelWidget) OnThemeChanged(fontSize float32) {
	p.lastFontSize = fontSize
	if p.TitleText != nil {
		p.TitleText.TextSize = fontSize + 1
		p.TitleText.Refresh()
	}
	if p.List != nil {
		p.List.Refresh()
	}
}

// adjustAlarmRowColor трохи змінює яскравість кольору рядка,
// щоб підсвітити вибраний елемент у списку тривог.
func adjustAlarmRowColor(c color.NRGBA) color.NRGBA {
	const factor = 1.15 // 15% яскравіше
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
