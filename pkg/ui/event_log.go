// Package ui - глобальний журнал подій
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
	"obj_catalog_fyne_v3/pkg/utils"
)

// EventLogPanel - структура журналу подій
type EventLogPanel struct {
	Container       *fyne.Container
	List            *widget.List
	Data            data.EventProvider
	IsPaused        bool
	PauseBtn        *widget.Button
	OnEventSelected func(models.Event)

	// Кеш даних
	CurrentEvents []models.Event
	mutex         sync.RWMutex
	isRefreshing  bool
}

// NewEventLogPanel створює панель журналу подій
func NewEventLogPanel(provider data.EventProvider) *EventLogPanel {
	panel := &EventLogPanel{
		Data:     provider,
		IsPaused: false,
	}

	// Заголовок
	titleText := canvas.NewText("📜 ЖУРНАЛ ПОДІЙ", nil)
	titleText.TextSize = 14
	titleText.TextStyle = fyne.TextStyle{Bold: true}

	// Кнопка паузи
	panel.PauseBtn = widget.NewButton("⏸ Пауза", func() {
		panel.IsPaused = !panel.IsPaused
		if panel.IsPaused {
			panel.PauseBtn.SetText("▶ Продовжити")
		} else {
			panel.PauseBtn.SetText("⏸ Пауза")
		}
	})

	header := container.NewBorder(
		nil, nil,
		container.NewPadded(titleText),
		panel.PauseBtn,
		nil,
	)

	// Список подій (тепер використовує кеш)
	panel.List = widget.NewList(
		func() int {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()
			return len(panel.CurrentEvents)
		},
		func() fyne.CanvasObject {
			bg := canvas.NewRectangle(color.Transparent)
			txt := canvas.NewText("Подія", color.White)
			txt.TextSize = 13 // Співпадає з темою
			return container.NewStack(bg, container.NewPadded(txt))
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			panel.mutex.RLock()
			defer panel.mutex.RUnlock()

			if id < len(panel.CurrentEvents) {
				stack := obj.(*fyne.Container)
				bg := stack.Objects[0].(*canvas.Rectangle)
				txtContainer := stack.Objects[1].(*fyne.Container)
				txt := txtContainer.Objects[0].(*canvas.Text)

				event := panel.CurrentEvents[id]

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

				icon := getEventIcon(event.Type)
				text := icon + " " + event.GetDateTimeDisplay() + " | №" + itoa(event.ObjectID) + " " + event.ObjectName + " | " + event.GetTypeDisplay()
				if event.Details != "" {
					text += " — " + event.Details
				}
				txt.Text = text
				txt.Refresh()
			}
		},
	)

	panel.List.OnSelected = func(id widget.ListItemID) {
		panel.mutex.RLock()
		defer panel.mutex.RUnlock()
		if id < len(panel.CurrentEvents) && panel.OnEventSelected != nil {
			panel.OnEventSelected(panel.CurrentEvents[id])
		}
		panel.List.Unselect(id)
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
func (e *EventLogPanel) Refresh() {
	if e.Data == nil || e.IsPaused {
		return
	}

	e.mutex.Lock()
	if e.isRefreshing {
		e.mutex.Unlock()
		return
	}
	e.isRefreshing = true
	e.mutex.Unlock()

	defer func() {
		e.mutex.Lock()
		e.isRefreshing = false
		e.mutex.Unlock()
	}()

	// Отримуємо дані з БД (може заблокувати горутину, але не UI)
	events := e.Data.GetEvents()

	// Оновлюємо кеш
	e.mutex.Lock()
	e.CurrentEvents = events
	e.mutex.Unlock()

	// Оновлюємо UI у головному вікні
	fyne.Do(func() {
		if e.List != nil {
			e.List.Refresh()
		}
	})
}

// Решта функцій (getEventIcon, getEventImportance) залишаються незмінними (вони в тому ж файлі були?)
// Так, вони були в кінці файлу. Я їх додам сюди для цілісності.

func getEventIcon(eventType models.EventType) string {
	switch eventType {
	case models.EventFire:
		return "🔴"
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

func getEventImportance(event models.Event) widget.Importance {
	if event.IsCritical() {
		return widget.DangerImportance
	}
	if event.IsWarning() {
		return widget.WarningImportance
	}
	return widget.MediumImportance
}
