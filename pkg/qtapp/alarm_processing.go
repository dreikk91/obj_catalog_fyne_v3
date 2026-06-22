//go:build qt

package qtapp

import (
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
)

const (
	alarmSourceBridge  = "bridge"
	alarmSourcePhoenix = "phoenix"
	alarmSourceCASL    = "casl"
)

func sameAlarmProcessingSource(alarms []models.Alarm) bool {
	if len(alarms) < 2 {
		return true
	}
	source := alarmProcessingSource(alarms[0])
	for _, alarm := range alarms[1:] {
		if alarmProcessingSource(alarm) != source {
			return false
		}
	}
	return true
}

func alarmProcessingSource(alarm models.Alarm) string {
	switch {
	case ids.IsCASLObjectID(alarm.ObjectID):
		return alarmSourceCASL
	case ids.IsPhoenixObjectID(alarm.ObjectID):
		return alarmSourcePhoenix
	default:
		return alarmSourceBridge
	}
}

func intersectAlarmProcessingOptions(optionSets ...[]contracts.AlarmProcessingOption) []contracts.AlarmProcessingOption {
	if len(optionSets) == 0 {
		return nil
	}

	first := normalizeAlarmProcessingOptions(optionSets[0])
	if len(first) == 0 {
		return nil
	}

	commonCodes := make(map[string]struct{}, len(first))
	for _, option := range first {
		commonCodes[option.Code] = struct{}{}
	}

	for _, set := range optionSets[1:] {
		current := make(map[string]struct{}, len(set))
		for _, option := range normalizeAlarmProcessingOptions(set) {
			current[option.Code] = struct{}{}
		}
		for code := range commonCodes {
			if _, ok := current[code]; !ok {
				delete(commonCodes, code)
			}
		}
	}

	result := make([]contracts.AlarmProcessingOption, 0, len(commonCodes))
	for _, option := range first {
		if _, ok := commonCodes[option.Code]; ok {
			result = append(result, option)
		}
	}
	return result
}

func normalizeAlarmProcessingOptions(options []contracts.AlarmProcessingOption) []contracts.AlarmProcessingOption {
	normalized := make([]contracts.AlarmProcessingOption, 0, len(options))
	seen := make(map[string]struct{}, len(options))
	for _, option := range options {
		code := strings.TrimSpace(option.Code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}

		label := strings.TrimSpace(option.Label)
		if label == "" {
			label = code
		}
		normalized = append(normalized, contracts.AlarmProcessingOption{
			Code:  code,
			Label: label,
		})
	}
	return normalized
}
