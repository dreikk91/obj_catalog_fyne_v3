package application

import (
	"fyne.io/fyne/v2"

	"obj_catalog_fyne_v3/pkg/eventbus"
)

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
	a.eventBus.Publish(eventbus.TopicDataRefresh, refresh)
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

func (a *Application) focusObjectByID(objectID int64) {
	a.applyObjectContextByID(objectID, true)
}
