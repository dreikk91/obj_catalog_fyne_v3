package application

import (
	"time"

	"fyne.io/fyne/v2"

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
		fyne.Do(func() {
			a.handleDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true, RefreshAlarms: true, RefreshEvents: true})
			a.focusObjectByID(event.ObjectID)
		})
	})

	a.eventBus.Subscribe(eventbus.TopicObjectDeleted, func(payload any) {
		event, ok := payload.(eventbus.ObjectDeletedEvent)
		if !ok {
			return
		}
		fyne.Do(func() {
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
		fyne.Do(func() {
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
	if refresh.RefreshObjects && a.objectList != nil {
		a.objectList.Refresh()
	}
	if refresh.RefreshAlarms && a.alarmPanel != nil {
		a.alarmPanel.Refresh()
	}
	if refresh.RefreshEvents && a.eventLog != nil {
		a.eventLog.Refresh()
	}
	if refresh.RefreshEvents && a.workArea != nil {
		a.workArea.RefreshCurrentObjectEvents()
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
	a.eventBus.Publish(eventbus.TopicDataRefresh, refresh)
}

func mergeDataRefreshEvents(base, extra eventbus.DataRefreshEvent) eventbus.DataRefreshEvent {
	base.RefreshObjects = base.RefreshObjects || extra.RefreshObjects
	base.RefreshAlarms = base.RefreshAlarms || extra.RefreshAlarms
	base.RefreshEvents = base.RefreshEvents || extra.RefreshEvents
	return base
}

func (a *Application) focusObjectByID(objectID int64) {
	a.applyObjectContextByID(objectID, true)
}
