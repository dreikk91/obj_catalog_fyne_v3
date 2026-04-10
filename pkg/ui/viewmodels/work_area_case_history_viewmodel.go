package viewmodels

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

type WorkAreaCaseHistoryGroup struct {
	Root   models.Event
	Events []models.Event
	Title  string
}

type WorkAreaCaseHistorySection struct {
	Key    string
	Title  string
	Events []models.Event
}

type WorkAreaCaseHistoryViewModel struct{}

const (
	caseHistorySectionRoot     = "root"
	caseHistorySectionRestore  = "restore"
	caseHistorySectionOperator = "operator"
	caseHistorySectionGuard    = "guard"
	caseHistorySectionSystem   = "system"
	caseHistorySectionOther    = "other"
)

func NewWorkAreaCaseHistoryViewModel() *WorkAreaCaseHistoryViewModel {
	return &WorkAreaCaseHistoryViewModel{}
}

func (vm *WorkAreaCaseHistoryViewModel) BuildGroups(currentObject *models.Object, events []models.Event) []WorkAreaCaseHistoryGroup {
	if currentObject == nil || !ids.IsCASLObjectID(currentObject.ID) || len(events) == 0 {
		return nil
	}

	ordered := slices.Clone(events)
	slices.SortStableFunc(ordered, func(left, right models.Event) int {
		leftTime := left.Time
		rightTime := right.Time
		if leftTime.Equal(rightTime) {
			switch {
			case left.ID < right.ID:
				return -1
			case left.ID > right.ID:
				return 1
			default:
				return 0
			}
		}
		switch {
		case leftTime.Before(rightTime):
			return -1
		case leftTime.After(rightTime):
			return 1
		default:
			return 0
		}
	})

	groups := make([]WorkAreaCaseHistoryGroup, 0, 4)
	current := WorkAreaCaseHistoryGroup{}
	hasCurrent := false

	for _, event := range ordered {
		if isWorkAreaCaseRootEvent(event) {
			if hasCurrent {
				current.Title = buildWorkAreaCaseGroupTitle(current.Root, len(current.Events)-1)
				groups = append(groups, current)
			}
			current = WorkAreaCaseHistoryGroup{
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
		current.Title = buildWorkAreaCaseGroupTitle(current.Root, len(current.Events)-1)
		groups = append(groups, current)
	}

	slices.Reverse(groups)

	return groups
}

func (vm *WorkAreaCaseHistoryViewModel) FindGroupForAlarm(
	currentObject *models.Object,
	alarm models.Alarm,
	events []models.Event,
) (WorkAreaCaseHistoryGroup, bool) {
	groups := vm.BuildGroups(currentObject, events)
	if len(groups) == 0 {
		return WorkAreaCaseHistoryGroup{}, false
	}

	bestIdx := 0
	bestScore := -1
	bestDelta := time.Duration(1<<63 - 1)

	for idx, group := range groups {
		score, delta := scoreCaseHistoryGroupForAlarm(group, alarm)
		if score > bestScore || (score == bestScore && delta < bestDelta) {
			bestIdx = idx
			bestScore = score
			bestDelta = delta
		}
	}

	return groups[bestIdx], true
}

func (vm *WorkAreaCaseHistoryViewModel) BuildSections(group WorkAreaCaseHistoryGroup) []WorkAreaCaseHistorySection {
	if len(group.Events) == 0 {
		return nil
	}

	sections := make([]WorkAreaCaseHistorySection, 0, 4)
	indexByKey := make(map[string]int, 6)

	for _, event := range group.Events {
		key, title := caseHistorySectionForEvent(event, group.Root)
		idx, exists := indexByKey[key]
		if !exists {
			idx = len(sections)
			indexByKey[key] = idx
			sections = append(sections, WorkAreaCaseHistorySection{
				Key:   key,
				Title: title,
			})
		}
		sections[idx].Events = append(sections[idx].Events, event)
	}

	for idx := range sections {
		sections[idx].Title = fmt.Sprintf("%s (%d)", sections[idx].Title, len(sections[idx].Events))
	}

	return sections
}

func isWorkAreaCaseRootEvent(event models.Event) bool {
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
		models.EventDeviceBlocked:
		return true
	default:
		return false
	}
}

func scoreCaseHistoryGroupForAlarm(group WorkAreaCaseHistoryGroup, alarm models.Alarm) (int, time.Duration) {
	bestScore := -1
	bestDelta := time.Duration(1<<63 - 1)

	for _, event := range group.Events {
		score := 0
		if caseRootMatchesAlarm(event.Type, alarm.Type) {
			score += 4
		}
		if alarm.ZoneNumber > 0 && event.ZoneNumber == alarm.ZoneNumber {
			score += 3
		}
		delta := absDuration(event.Time.Sub(alarm.Time))
		if score > bestScore || (score == bestScore && delta < bestDelta) {
			bestScore = score
			bestDelta = delta
		}
	}

	if bestScore >= 0 {
		return bestScore, bestDelta
	}
	return 0, absDuration(group.Root.Time.Sub(alarm.Time))
}

func caseRootMatchesAlarm(eventType models.EventType, alarmType models.AlarmType) bool {
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
		return isWorkAreaCaseRootEvent(models.Event{Type: eventType})
	}
}

func caseHistorySectionForEvent(event models.Event, root models.Event) (string, string) {
	if isSameCaseRootEvent(event, root) {
		return caseHistorySectionRoot, "Початок тривоги"
	}

	switch event.Type {
	case models.EventRestore,
		models.EventPowerOK,
		models.EventOnline,
		models.EventAlarmFinished,
		models.EventDeviceUnblocked:
		return caseHistorySectionRestore, "Відновлення"
	case models.EventOperatorAction,
		models.EventManagerAssigned,
		models.EventManagerArrived,
		models.EventManagerCanceled:
		return caseHistorySectionOperator, "Оператор / реагування"
	case models.EventArm,
		models.EventDisarm:
		return caseHistorySectionGuard, "Охорона / режим"
	case models.EventTest,
		models.EventNotification,
		models.EventAlarmNotification,
		models.SystemEvent,
		models.EventService:
		return caseHistorySectionSystem, "Система / сервіс"
	default:
		return caseHistorySectionOther, "Інші події"
	}
}

func isSameCaseRootEvent(event models.Event, root models.Event) bool {
	return event.ID == root.ID &&
		event.Type == root.Type &&
		event.ZoneNumber == root.ZoneNumber &&
		event.Time.Equal(root.Time)
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

func buildWorkAreaCaseGroupTitle(root models.Event, followUpCount int) string {
	parts := []string{root.GetDateTimeDisplay()}
	if root.ZoneNumber > 0 {
		parts = append(parts, fmt.Sprintf("зона %d", root.ZoneNumber))
	}
	parts = append(parts, strings.ToLower(strings.TrimSpace(root.GetTypeDisplay())))

	title := strings.Join(parts, " | ")
	details := strings.TrimSpace(root.Details)
	if details != "" {
		title += " — " + details
	}
	if followUpCount > 0 {
		title += fmt.Sprintf(" (+%d)", followUpCount)
	}
	return title
}
