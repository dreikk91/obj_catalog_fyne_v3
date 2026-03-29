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
	uk := extractDictionaryLanguageMap(dict, "uk")
	for k, v := range uk {
		if k != "" && v != "" { base[k] = v }
	}
	return base
}

func extractDictionaryLanguageMap(dict map[string]any, lang string) map[string]string {
	langCandidates := []string{lang, strings.ToUpper(lang), "uk-UA", "uk_ua", "ua", "UA"}

	resolve := func(node any) map[string]string {
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
		if out := resolve(nested); len(out) > 0 { return out }
	}
	if nested, ok := dict["dictionary"].(map[string]any); ok {
		if tr, ok := nested["translate"]; ok {
			if out := resolve(tr); len(out) > 0 { return out }
		}
	}
	return nil
}

func FlattenTranslatorMap(value any) map[string]string {
	result := make(map[string]string)

	setIfEmpty := func(k, v string) {
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k != "" && v != "" && result[k] == "" { result[k] = v }
	}

	var walk func(v any)
	walk = func(v any) {
		switch typed := v.(type) {
		case map[string]any:
			codes := extractTranslatorCodes(typed)
			text := extractTranslatorText(typed)
			if len(codes) > 0 && text != "" {
				for _, code := range codes { setIfEmpty(code, text) }
			}
			for k, nested := range typed {
				if looksLikeCode(k) {
					if t := extractTranslatorText(nested); t != "" { setIfEmpty(k, t) }
				}
				walk(nested)
			}
		case []any:
			for _, item := range typed { walk(item) }
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
		priority := []string{"msg", "message", "text", "description", "title", "uk"}
		for _, key := range priority {
			if v, ok := typed[key]; ok {
				if s := extractTranslatorText(v); s != "" { return s }
			}
		}
	}
	return ""
}

func looksLikeCode(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" { return false }
	if strings.HasPrefix(key, "TYPE_") { return false }
	return true
}
