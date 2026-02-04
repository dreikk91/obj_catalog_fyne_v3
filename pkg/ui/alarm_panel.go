// Package ui - панель активних тривог
package ui

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

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
}

// NewAlarmPanelWidget створює панель тривог
func NewAlarmPanelWidget(provider data.AlarmProvider) *AlarmPanelWidget {
	panel := &AlarmPanelWidget{
		Data: provider,
	}

	// Заголовок
	titleText := canvas.NewText("🔔 АКТИВНІ ТРИВОГИ", color.White)
	titleText.TextSize = 14
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	titleBg := canvas.NewRectangle(color.NRGBA{R: 100, G: 0, B: 0, A: 255})
	titleContainer := container.NewStack(titleBg, container.NewPadded(titleText))

	// Список тривог (тепер використовує кеш)
	panel.List = widget.NewList(
		func() int {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()
			return len(panel.CurrentAlarms)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Тривога")
			label.Importance = widget.DangerImportance
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()

			if id < len(panel.CurrentAlarms) {
				label := obj.(*widget.Label)
				alarm := panel.CurrentAlarms[id]
				icon := "🔴"
				if alarm.Type == models.AlarmFault {
					icon = "🟡"
					label.Importance = widget.WarningImportance
				} else {
					label.Importance = widget.DangerImportance
				}
				displayText := icon + " " + alarm.GetTimeDisplay() + " | №" + itoa(alarm.ObjectID) + " " + alarm.ObjectName + " | " + alarm.GetTypeDisplay()
				if alarm.Details != "" {
					displayText += " — " + alarm.Details
				}
				label.SetText(displayText)
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
