//go:build qt

package qtapp

import (
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/eventbus"
)

const refreshCoalesceWindow = 120 * time.Millisecond

func (a *Application) registerEventBusHandlers() {
	if a == nil || a.eventBus == nil {
		return
	}

	a.eventBus.Subscribe(eventbus.TopicObjectSaved, func(payload any) {
		event, ok := payload.(eventbus.ObjectSavedEvent)
		if !ok {
			return
		}
		a.runOnMainThread(func() {
			a.handleDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true, RefreshAlarms: true, RefreshEvents: true})
			a.applyObjectContextByID(event.ObjectID)
		})
	})

	a.eventBus.Subscribe(eventbus.TopicObjectDeleted, func(payload any) {
		event, ok := payload.(eventbus.ObjectDeletedEvent)
		if !ok {
			return
		}
		a.runOnMainThread(func() {
			if a.currentObject != nil && int64(a.currentObject.ID) == event.ObjectID {
				a.clearObjectContext()
			}
			a.handleDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true, RefreshAlarms: true, RefreshEvents: true})
		})
	})

	a.eventBus.Subscribe(eventbus.TopicDataRefresh, func(payload any) {
		event, ok := payload.(eventbus.DataRefreshEvent)
		if !ok {
			return
		}
		a.runOnMainThread(func() {
			a.handleDataRefresh(event)
		})
	})
}

func (a *Application) publishObjectSaved(objectID int64) {
	if a == nil || a.eventBus == nil {
		return
	}
	a.eventBus.Publish(eventbus.TopicObjectSaved, eventbus.ObjectSavedEvent{ObjectID: objectID})
}

func (a *Application) publishObjectDeleted(objectID int64) {
	if a == nil || a.eventBus == nil {
		return
	}
	a.eventBus.Publish(eventbus.TopicObjectDeleted, eventbus.ObjectDeletedEvent{ObjectID: objectID})
}

func (a *Application) publishDataRefresh(refresh eventbus.DataRefreshEvent) {
	if a == nil || a.eventBus == nil {
		return
	}
	if !refresh.RefreshObjects && !refresh.RefreshAlarms && !refresh.RefreshEvents {
		return
	}

	a.refreshCoalesceMu.Lock()
	a.pendingRefresh = mergeDataRefreshEvents(a.pendingRefresh, refresh)
	if a.refreshCoalescePending {
		a.refreshCoalesceMu.Unlock()
		return
	}
	a.refreshCoalescePending = true
	a.refreshCoalesceMu.Unlock()

	time.AfterFunc(refreshCoalesceWindow, func() {
		a.flushPendingDataRefresh()
	})
}

func (a *Application) handleDataRefresh(refresh eventbus.DataRefreshEvent) {
	if a == nil || a.ui == nil {
		return
	}
	if !refresh.RefreshObjects && !refresh.RefreshAlarms && !refresh.RefreshEvents {
		a.refreshData()
		return
	}
	if refresh.RefreshObjects {
		a.refreshObjects()
	}
	if refresh.RefreshAlarms {
		a.refreshAlarms()
	}
	if refresh.RefreshEvents {
		a.refreshEvents()
		a.ui.RefreshCurrentObjectEvents()
	}
	if a.runtime != nil {
		a.ui.SetStatus(backendStatusText(a.runtime))
	}
}

func (a *Application) flushPendingDataRefresh() {
	if a == nil || a.eventBus == nil {
		return
	}

	a.refreshCoalesceMu.Lock()
	refresh := a.pendingRefresh
	a.pendingRefresh = eventbus.DataRefreshEvent{}
	a.refreshCoalescePending = false
	a.refreshCoalesceMu.Unlock()

	if !refresh.RefreshObjects && !refresh.RefreshAlarms && !refresh.RefreshEvents {
		return
	}
	a.runOnMainThread(func() {
		a.handleDataRefresh(refresh)
	})
}

func mergeDataRefreshEvents(base, extra eventbus.DataRefreshEvent) eventbus.DataRefreshEvent {
	base.RefreshObjects = base.RefreshObjects || extra.RefreshObjects
	base.RefreshAlarms = base.RefreshAlarms || extra.RefreshAlarms
	base.RefreshEvents = base.RefreshEvents || extra.RefreshEvents
	return base
}

func (a *Application) runOnMainThread(f func()) {
	if a == nil || a.mainThreadQueue == nil || f == nil {
		return
	}
	select {
	case a.mainThreadQueue <- f:
		return
	default:
	}

	timer := time.NewTimer(mainThreadEnqueueWarnAfter)
	select {
	case a.mainThreadQueue <- f:
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return
	case <-timer.C:
		log.Warn().
			Int("queued", len(a.mainThreadQueue)).
			Int("capacity", cap(a.mainThreadQueue)).
			Msg("Qt main-thread queue is full; enqueueing asynchronously")
	}

	go func() {
		ticker := time.NewTicker(mainThreadEnqueueRetryDelay)
		defer ticker.Stop()
		for {
			select {
			case a.mainThreadQueue <- f:
				return
			case <-ticker.C:
			}
		}
	}()
}

func (a *Application) safeRunOnMainThread(f func()) {
	if f == nil {
		return
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Error().Interface("panic", recovered).Msg("Qt main-thread callback panicked")
		}
	}()
	f()
}

func (a *Application) applyObjectContextByID(objectID int64) {
	if a == nil || a.uiData == nil {
		return
	}
	for _, object := range a.uiData.GetObjects() {
		if int64(object.ID) == objectID {
			a.applyObjectContext(object)
			return
		}
	}
}

func (a *Application) clearObjectContext() {
	if a == nil || a.ui == nil {
		return
	}
	a.currentObject = nil
	a.ui.SetStatus("Об'єкт не вибрано")
}
