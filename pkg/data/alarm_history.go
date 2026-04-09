package data

import (
	"obj_catalog_fyne_v3/pkg/models"
	"sort"
	"time"
)

type alarmEventGroup struct {
	Root   models.Event
	Events []models.Event
}

func buildAlarmSourceMessagesFromEvents(alarm models.Alarm, events []models.Event) []models.AlarmMsg {
	if len(events) == 0 {
		return nil
	}

	groups := groupAlarmEvents(events)
	if group, ok := selectAlarmEventGroup(groups, alarm); ok {
		return mapAlarmEventsToSourceMsgs(filterAlarmEventsSince(group.Events, alarm.Time))
	}

	return mapAlarmEventsToSourceMsgs(filterAlarmEventsSince(events, alarm.Time))
}

func filterAlarmEventsSince(events []models.Event, since time.Time) []models.Event {
	if len(events) == 0 || since.IsZero() {
		return append([]models.Event(nil), events...)
	}

	filtered := make([]models.Event, 0, len(events))
	for _, event := range events {
		if !event.Time.IsZero() && event.Time.Before(since) {
			continue
		}
		filtered = append(filtered, event)
	}
	return filtered
}

func groupAlarmEvents(events []models.Event) []alarmEventGroup {
	if len(events) == 0 {
		return nil
	}

	ordered := append([]models.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.Time.Equal(right.Time) {
			return left.ID < right.ID
		}
		return left.Time.Before(right.Time)
	})

	groups := make([]alarmEventGroup, 0, 4)
	current := alarmEventGroup{}
	hasCurrent := false

	for _, event := range ordered {
		if isAlarmCaseRootEvent(event) {
			if hasCurrent && !shouldStartNewAlarmGroup(current, event) {
				current.Events = append(current.Events, event)
				continue
			}
			if hasCurrent {
				groups = append(groups, current)
			}
			current = alarmEventGroup{
				Root:   event,
				Events: []models.Event{event},
			}
			hasCurrent = true
			continue
		}
		if hasCurrent {
			current.Events = append(current.Events, event)
		}
	}

	if hasCurrent {
		groups = append(groups, current)
	}

	for left, right := 0, len(groups)-1; left < right; left, right = left+1, right-1 {
		groups[left], groups[right] = groups[right], groups[left]
	}

	return groups
}

func shouldStartNewAlarmGroup(current alarmEventGroup, event models.Event) bool {
	if len(current.Events) == 0 {
		return true
	}
	if isPrimaryAlarmEventType(current.Root.Type) && !isPrimaryAlarmEventType(event.Type) {
		return false
	}
	return true
}

func selectAlarmEventGroup(groups []alarmEventGroup, alarm models.Alarm) (alarmEventGroup, bool) {
	if len(groups) == 0 {
		return alarmEventGroup{}, false
	}

	bestIdx := 0
	bestScore := -1
	bestDelta := time.Duration(1<<63 - 1)

	for idx, group := range groups {
		score := 0
		if alarmEventMatchesAlarm(group.Root.Type, alarm.Type) {
			score += 4
		}
		if alarm.ZoneNumber > 0 && group.Root.ZoneNumber == alarm.ZoneNumber {
			score += 3
		}

		delta := absAlarmDuration(group.Root.Time.Sub(alarm.Time))
		if score > bestScore || (score == bestScore && delta < bestDelta) {
			bestIdx = idx
			bestScore = score
			bestDelta = delta
		}
	}

	return groups[bestIdx], true
}

func mapAlarmEventsToSourceMsgs(events []models.Event) []models.AlarmMsg {
	if len(events) == 0 {
		return nil
	}

	ordered := append([]models.Event(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		left := ordered[i]
		right := ordered[j]
		if left.Time.Equal(right.Time) {
			return left.ID > right.ID
		}
		return left.Time.After(right.Time)
	})

	result := make([]models.AlarmMsg, 0, len(ordered))
	for _, event := range ordered {
		_, isAlarm := mapEventTypeToAlarmType(event.Type)
		result = append(result, models.AlarmMsg{
			Time:    event.Time,
			Number:  event.ZoneNumber,
			Details: event.Details,
			SC1:     event.SC1,
			IsAlarm: isAlarm,
		})
	}

	return result
}

func isAlarmCaseRootEvent(event models.Event) bool {
	switch event.Type {
	case models.EventFire,
		models.EventBurglary,
		models.EventPanic,
		models.EventMedical,
		models.EventGas,
		models.EventTamper,
		models.EventFault,
		models.EventPowerFail,
		models.EventBatteryLow,
		models.EventOffline,
		models.EventAlarmNotification,
		models.EventNotification,
		models.EventDeviceBlocked:
		return true
	default:
		return false
	}
}

func alarmEventMatchesAlarm(eventType models.EventType, alarmType models.AlarmType) bool {
	switch alarmType {
	case models.AlarmFire:
		return eventType == models.EventFire
	case models.AlarmBurglary:
		return eventType == models.EventBurglary
	case models.AlarmPanic:
		return eventType == models.EventPanic
	case models.AlarmMedical:
		return eventType == models.EventMedical
	case models.AlarmGas:
		return eventType == models.EventGas
	case models.AlarmTamper:
		return eventType == models.EventTamper
	case models.AlarmFault, models.AlarmFireTrouble:
		return eventType == models.EventFault
	case models.AlarmPowerFail, models.AlarmAcTrouble:
		return eventType == models.EventPowerFail
	case models.AlarmBatteryLow:
		return eventType == models.EventBatteryLow
	case models.AlarmOffline:
		return eventType == models.EventOffline
	case models.AlarmNotification:
		return eventType == models.EventAlarmNotification || eventType == models.EventNotification
	default:
		return isAlarmCaseRootEvent(models.Event{Type: eventType})
	}
}

func absAlarmDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}
