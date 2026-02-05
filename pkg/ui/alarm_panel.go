// Package ui - панель активних тривог
package ui

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/models"
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

	OnAlarmSelected func(alarm models.Alarm)
	OnProcessAlarm  func(alarm models.Alarm)
	TitleText       *canvas.Text
	lastFontSize    float32
}

// NewAlarmPanelWidget створює панель тривог
func NewAlarmPanelWidget(provider data.AlarmProvider) *AlarmPanelWidget {
	panel := &AlarmPanelWidget{
		Data: provider,
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
				txtContainer := stack.Objects[1].(*fyne.Container)
				txt := txtContainer.Objects[0].(*canvas.Text)

				alarm := panel.CurrentAlarms[id]
				icon := "🔴"
				textColor := theme.ColorNameError
				if alarm.Type == models.AlarmFault {
					icon = "🟡"
					textColor = theme.ColorNameWarning
				}
				displayText := icon + " " + alarm.GetTimeDisplay() + " | №" + itoa(alarm.ObjectID) + " " + alarm.ObjectName + " | " + alarm.GetTypeDisplay()
				if alarm.Details != "" {
					displayText += " — " + alarm.Details
				}
				txt.Text = displayText
				variant := fyne.CurrentApp().Settings().ThemeVariant()
				txt.Color = fyne.CurrentApp().Settings().Theme().Color(textColor, variant)

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
		panel.mutex.RLock()
		defer panel.mutex.RUnlock()

		if id < len(panel.CurrentAlarms) && panel.OnAlarmSelected != nil {
			panel.OnAlarmSelected(panel.CurrentAlarms[id])
		}
	}

	panel.Container = container.NewBorder(
		titleContainer,
		nil, nil, nil,
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

	// Оновлюємо кеш та UI
	p.mutex.Lock()
	p.CurrentAlarms = alarms
	p.mutex.Unlock()

	fyne.Do(func() {
		if p.List != nil {
			p.List.Refresh()
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
