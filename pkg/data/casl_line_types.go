package data

import (
	"context"
	"fmt"
	"strings"
)

var caslDefaultZoneTypeLabels = map[string]string{
	"EMPTY":              "Пустий шлейф",
	"NORMAL":             "Нормальна зона",
	"ZONE_ALARM_ON_KZ":   "Тривожний шлейф",
	"ZONE_ALARM":         "Тривожний шлейф",
	"ZONE_ALM":           "Тривожний шлейф",
	"ALM_BTN":            "Тривожна кнопка",
	"ZONE_FIRE":          "Пожежний шлейф",
	"ZONE_NORMAL":        "Нормальна зона",
	"ZONE_COMMON":        "Нормальна зона",
	"ZONE_DELAY":         "Вхідний шлейф",
	"ZONE_PANIC":         "Тривожна кнопка",
	"UNTYPED_ZONE_ALARM": "Нетипізована тривожна зона",
}

func (p *CASLCloudProvider) resolveCASLDeviceLineTypeLabel(ctx context.Context, line caslDeviceLine) string {
	if rawLineType := strings.TrimSpace(line.LineType.String()); rawLineType != "" {
		return p.resolveCASLZoneTypeLabel(ctx, rawLineType)
	}

	rawType := strings.TrimSpace(line.Type.String())
	if rawType == "" {
		return ""
	}

	if label := p.lookupCASLLineTypeInDictionary(ctx, rawType); label != "" {
		return label
	}
	if label := caslFallbackZoneTypeLabel(rawType); label != "" {
		return label
	}
	return rawType
}

func (p *CASLCloudProvider) resolveCASLZoneTypeLabel(ctx context.Context, rawType string) string {
	rawType = strings.TrimSpace(rawType)
	if rawType == "" {
		return ""
	}

	if label := p.lookupCASLLineTypeInDictionary(ctx, rawType); label != "" {
		return label
	}
	if label := caslFallbackZoneTypeLabel(rawType); label != "" {
		return label
	}
	return rawType
}

func (p *CASLCloudProvider) lookupCASLLineTypeInDictionary(ctx context.Context, rawType string) string {
	dict, ok := p.cachedDictionarySnapshot(ctx)
	if !ok || len(dict) == 0 {
		return ""
	}

	lineTypes := extractCASLLineTypesMap(dict)
	if len(lineTypes) == 0 {
		return ""
	}

	for _, key := range []string{rawType, strings.ToUpper(rawType), strings.ToLower(rawType)} {
		if label := strings.TrimSpace(lineTypes[key]); label != "" {
			return label
		}
	}
	return ""
}

func extractCASLLineTypesMap(value any) map[string]string {
	root, ok := value.(map[string]any)
	if !ok || len(root) == 0 {
		return nil
	}

	if mapped := extractCASLLineTypesMapFromRoot(root); len(mapped) > 0 {
		return mapped
	}

	if nestedRaw, exists := root["dictionary"]; exists {
		if nested, ok := nestedRaw.(map[string]any); ok {
			if mapped := extractCASLLineTypesMapFromRoot(nested); len(mapped) > 0 {
				return mapped
			}
		}
	}

	return nil
}

func extractCASLLineTypesMapFromRoot(root map[string]any) map[string]string {
	if len(root) == 0 {
		return nil
	}

	translateMap := extractCASLDictionaryLanguageMap(root, "uk")
	for _, key := range []string{"line_types", "zone_types"} {
		raw, exists := root[key]
		if !exists {
			continue
		}
		if mapped := normalizeCASLLineTypeOptions(raw, translateMap); len(mapped) > 0 {
			return mapped
		}
	}
	return nil
}

func normalizeCASLLineTypeOptions(raw any, translateMap map[string]string) map[string]string {
	switch typed := raw.(type) {
	case map[string]string:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			label := strings.TrimSpace(value)
			if label == "" {
				label = strings.TrimSpace(translateMap[key])
			}
			if label == "" {
				label = key
			}
			result[key] = label
		}
		return result
	case map[string]any:
		result := make(map[string]string, len(typed))
		for key, value := range typed {
			key = strings.TrimSpace(key)
			if key == "" {
				continue
			}
			label := strings.TrimSpace(fmt.Sprint(value))
			if nestedMap, ok := value.(map[string]any); ok {
				if nestedLabel := firstNonEmptyStringMapValue(nestedMap, "label", "name", "title", "description", "value"); nestedLabel != "" {
					label = nestedLabel
				}
			}
			if label == "" || strings.EqualFold(label, key) {
				label = strings.TrimSpace(translateMap[key])
			}
			if label == "" {
				label = key
			}
			result[key] = label
		}
		return result
	case []string:
		return mapCASLLineTypeKeysToLabels(typed, translateMap)
	case []any:
		keys := make([]string, 0, len(typed))
		for _, item := range typed {
			key := strings.TrimSpace(fmt.Sprint(item))
			if key == "" {
				continue
			}
			keys = append(keys, key)
		}
		return mapCASLLineTypeKeysToLabels(keys, translateMap)
	default:
		return nil
	}
}

func mapCASLLineTypeKeysToLabels(keys []string, translateMap map[string]string) map[string]string {
	if len(keys) == 0 {
		return nil
	}

	result := make(map[string]string, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		label := strings.TrimSpace(translateMap[key])
		if label == "" {
			label = key
		}
		result[key] = label
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func firstNonEmptyStringMapValue(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key]; ok {
			text := strings.TrimSpace(fmt.Sprint(value))
			if text != "" {
				return text
			}
		}
	}
	return ""
}

func caslFallbackZoneTypeLabel(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	upper := strings.ToUpper(raw)
	if label, ok := caslDefaultZoneTypeLabels[upper]; ok {
		return label
	}

	switch {
	case strings.HasPrefix(strings.ToLower(raw), "fire_pipeline"), strings.Contains(upper, "FIRE"):
		return "Пожежний шлейф"
	case strings.Contains(upper, "PANIC"), strings.Contains(upper, "ALM_BTN"):
		return "Тривожна кнопка"
	case strings.Contains(upper, "ALARM"), strings.Contains(upper, "ZONE_ALM"):
		return "Тривожний шлейф"
	case strings.Contains(upper, "NORMAL"), strings.Contains(upper, "COMMON"):
		return "Нормальна зона"
	case strings.Contains(upper, "UNTYPED"):
		return "Звичайна зона"
	case strings.Contains(upper, "DELAY"), strings.Contains(upper, "ENTRY"):
		return "Вхідний шлейф"
	default:
		return ""
	}
}
