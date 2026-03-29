package casl

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

func StableEventID(objID string, ts int64, code string, index int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(strings.TrimSpace(objID)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(fmt.Sprintf("%d", ts)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(strings.TrimSpace(code)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(fmt.Sprintf("%d", index)))

	return int(h.Sum32() & 0x7fffffff)
}

func FlattenDictionaryMap(dict map[string]any) map[string]string {
	base := flattenStringMap(dict)

	// Localization "translate" -> "uk" extraction
	langCandidates := []string{"uk", "uk-UA", "uk_ua", "ua", "UA"}

	extractLang := func(node any) map[string]string {
		root, ok := node.(map[string]any)
		if !ok { return nil }
		for _, key := range langCandidates {
			if nested, exists := root[key]; exists {
				return flattenStringMap(nested)
			}
		}
		return nil
	}

	if nested, ok := dict["translate"]; ok {
		uk := extractLang(nested)
		for k, v := range uk { base[k] = v }
	}

	return base
}

func FlattenTranslatorMap(value any) map[string]string {
	result := make(map[string]string)

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			codes := extractTranslatorCodes(typed)
			text := extractTranslatorText(typed)
			if len(codes) > 0 && text != "" {
				for _, code := range codes { result[code] = text }
			}
			for _, nested := range typed { walk(nested) }
		case []any:
			for _, nested := range typed { walk(nested) }
		}
	}

	walk(value)
	return result
}

func flattenStringMap(value any) map[string]string {
	result := make(map[string]string)
	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			keys := make([]string, 0, len(typed))
			for k := range typed { keys = append(keys, k) }
			sort.Strings(keys)
			for _, k := range keys {
				nested := typed[k]
				if s, ok := nested.(string); ok {
					result[k] = s
				} else {
					walk(nested)
				}
			}
		case []any:
			for _, nested := range typed { walk(nested) }
		}
	}
	walk(value)
	return result
}

func extractTranslatorCodes(entry map[string]any) []string {
	for _, key := range []string{"contact_id", "code", "event_code", "id", "key"} {
		if v, ok := entry[key]; ok {
			return []string{asString(v)}
		}
	}
	return nil
}

func extractTranslatorText(value any) string {
	switch typed := value.(type) {
	case string: return strings.TrimSpace(typed)
	case map[string]any:
		priority := []string{"msg", "message", "text", "description", "name", "uk"}
		for _, key := range priority {
			if v, ok := typed[key]; ok {
				if s := extractTranslatorText(v); s != "" { return s }
			}
		}
	}
	return ""
}
