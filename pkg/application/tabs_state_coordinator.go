package application

import (
	"fmt"

	"fyne.io/fyne/v2/container"
)

func (a *Application) configureTabsState(detailsTab *container.TabItem, eventsTab *container.TabItem, alarmsTab *container.TabItem, rightTabs *container.AppTabs) {
	if a == nil {
		return
	}

	a.rightTabs = rightTabs
	a.eventsTab = eventsTab
	a.alarmsTab = alarmsTab
	a.lastAlarmsCount = 0
	a.lastCriticalCount = 0
	a.lastEventsCount = 0

	if rightTabs != nil && detailsTab != nil {
		rightTabs.Select(detailsTab)
	}
}

// updateTabBadges оновлює заголовки вкладок і синхронізує заголовок вікна.
func (a *Application) updateTabBadges(alarmsCount int, criticalCount int, eventsCount int) {
	if a == nil {
		return
	}

	if alarmsCount >= 0 {
		a.lastAlarmsCount = alarmsCount
		a.lastCriticalCount = criticalCount
	}
	if eventsCount >= 0 {
		a.lastEventsCount = eventsCount
	}

	if a.alarmsTab != nil {
		alarmTitle := "АКТИВНІ ТРИВОГИ"
		if a.lastAlarmsCount > 0 {
			alarmTitle = fmt.Sprintf("АКТИВНІ ТРИВОГИ (%d)", a.lastAlarmsCount)
			if a.lastCriticalCount > 0 {
				alarmTitle = fmt.Sprintf("АКТИВНІ ТРИВОГИ (%d, КРИТИЧНІ: %d)", a.lastAlarmsCount, a.lastCriticalCount)
			}
		}
		a.alarmsTab.Text = alarmTitle
	}

	if a.eventsTab != nil {
		eventsTitle := "ЖУРНАЛ ПОДІЙ"
		if a.lastEventsCount > 0 {
			eventsTitle = fmt.Sprintf("ЖУРНАЛ ПОДІЙ (%d)", a.lastEventsCount)
		}
		a.eventsTab.Text = eventsTitle
	}

	if a.rightTabs != nil {
		a.rightTabs.Refresh()
	}

	// Оновлюємо заголовок вікна з урахуванням кількості тривог.
	a.currentAlarmsTotal = a.lastAlarmsCount
	a.updateWindowTitle()
}
