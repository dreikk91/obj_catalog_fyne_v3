package v1

import (
	"slices"
	"strconv"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type alarmGroupDraft struct {
	groupID    string
	alertLevel int
	latestAt   time.Time
	latestTime string
	primary    contracts.FrontendAlarmItem
	items      []contracts.FrontendAlarmItem
}

func BuildAlarmGroups(items []contracts.FrontendAlarmItem) []AlarmGroup {
	if len(items) == 0 {
		return []AlarmGroup{}
	}

	byObjectID := make(map[int][]contracts.FrontendAlarmItem)
	for _, item := range items {
		byObjectID[item.ObjectID] = append(byObjectID[item.ObjectID], item)
	}

	groups := make([]alarmGroupDraft, 0, len(items))
	for objectID, objectItems := range byObjectID {
		sortedAsc := slices.Clone(objectItems)
		slices.SortFunc(sortedAsc, func(left, right contracts.FrontendAlarmItem) int {
			if left.Time.Equal(right.Time) {
				switch {
				case left.ID < right.ID:
					return -1
				case left.ID > right.ID:
					return 1
				default:
					return 0
				}
			}
			if left.Time.Before(right.Time) {
				return -1
			}
			return 1
		})

		groupIndex := 0
		var current *alarmGroupDraft
		for _, item := range sortedAsc {
			level := resolveAlarmGroupLevel(item)
			if current == nil || level > current.alertLevel {
				if current != nil {
					groups = append(groups, *current)
				}
				groupIndex++
				current = &alarmGroupDraft{
					groupID:    buildAlarmGroupID(objectID, groupIndex, item.ID),
					alertLevel: level,
					latestAt:   item.Time,
					latestTime: formatTimestamp(item.Time),
					primary:    item,
					items:      []contracts.FrontendAlarmItem{item},
				}
				continue
			}

			current.items = append(current.items, item)
			if item.Time.After(current.latestAt) {
				current.latestAt = item.Time
				current.latestTime = formatTimestamp(item.Time)
			}
		}

		if current != nil {
			groups = append(groups, *current)
		}
	}

	slices.SortFunc(groups, func(left, right alarmGroupDraft) int {
		switch {
		case left.latestAt.After(right.latestAt):
			return -1
		case left.latestAt.Before(right.latestAt):
			return 1
		case left.groupID < right.groupID:
			return -1
		case left.groupID > right.groupID:
			return 1
		default:
			return 0
		}
	})

	result := make([]AlarmGroup, 0, len(groups))
	for _, group := range groups {
		groupItems := make([]AlarmItem, 0, len(group.items))
		for _, item := range group.items {
			groupItems = append(groupItems, ToAlarmItem(item))
		}
		result = append(result, AlarmGroup{
			GroupID:        group.groupID,
			Source:         toSource(group.primary.Source),
			ObjectID:       group.primary.ObjectID,
			ObjectNativeID: group.primary.ObjectNativeID,
			ObjectNumber:   group.primary.ObjectNumber,
			ObjectName:     group.primary.ObjectName,
			Address:        group.primary.Address,
			AlertLevel:     group.alertLevel,
			LatestTime:     group.latestTime,
			Primary:        ToAlarmItem(group.primary),
			Items:          groupItems,
		})
	}
	return result
}

func resolveAlarmGroupLevel(item contracts.FrontendAlarmItem) int {
	switch item.TypeCode {
	case "panic":
		return 4
	case "fire", "FIRE_ALARM":
		return 3
	case "BURGLARY_ALARM", "medical", "gas", "tamper", "ALARM_TYPE_OPERATOR", "ALARM_TYPE_DEVICE", "ALARM_TYPE_MOBILE", "EXIT_ALARM":
		return 2
	case "fault", "power_fail", "battery_low", "offline", "AC_TROUBLE", "FIRE_TROUBLE", "system_event":
		return 1
	default:
		switch item.VisualSeverity {
		case contracts.FrontendVisualSeverityCritical:
			return 2
		case contracts.FrontendVisualSeverityWarning:
			return 1
		default:
			return 0
		}
	}
}

func buildAlarmGroupID(objectID int, groupIndex int, alarmID int) string {
	return formatAlarmGroupIDPart(objectID) + "-" + formatAlarmGroupIDPart(groupIndex) + "-" + formatAlarmGroupIDPart(alarmID)
}

func formatAlarmGroupIDPart(value int) string {
	return strconv.Itoa(value)
}
