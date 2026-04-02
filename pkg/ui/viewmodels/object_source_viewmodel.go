package viewmodels

import (
	"strconv"
	"strings"
	"unicode"

	"obj_catalog_fyne_v3/pkg/models"
)

const (
	ObjectSourceAll    = "Всі джерела"
	ObjectSourceBridge = "БД/МІСТ"
	ObjectSourceCASL   = "CASL Cloud"
)

const (
	caslObjectIDNamespaceStart = 1_500_000_000
	caslObjectIDNamespaceEnd   = 1_999_999_999
)

func IsCASLObjectID(id int) bool {
	return id >= caslObjectIDNamespaceStart && id <= caslObjectIDNamespaceEnd
}

func ObjectSourceByID(id int) string {
	if IsCASLObjectID(id) {
		return ObjectSourceCASL
	}
	return ObjectSourceBridge
}

func NormalizeObjectSourceFilter(selected string) string {
	clean := strings.TrimSpace(selected)
	if idx := strings.Index(clean, " ("); idx != -1 {
		clean = strings.TrimSpace(clean[:idx])
	}
	switch strings.ToLower(clean) {
	case strings.ToLower(ObjectSourceCASL), "casl":
		return ObjectSourceCASL
	case strings.ToLower(ObjectSourceBridge), "bridge", "db", "міст", "бд/міст", "db/bridge":
		return ObjectSourceBridge
	default:
		return ObjectSourceAll
	}
}

func BuildObjectSourceOptions(countAll int, countBridge int, countCASL int) []string {
	return []string{
		ObjectSourceAll + " (" + strconv.Itoa(countAll) + ")",
		ObjectSourceBridge + " (" + strconv.Itoa(countBridge) + ")",
		ObjectSourceCASL + " (" + strconv.Itoa(countCASL) + ")",
	}
}

func SourceBadgeForObjectID(id int) string {
	if IsCASLObjectID(id) {
		return "[C]"
	}
	return "[М]"
}

func ObjectDisplayNumber(object models.Object) string {
	if !IsCASLObjectID(object.ID) {
		return strconv.Itoa(object.ID)
	}

	// if number := numberFromPanelMark(object.PanelMark); number != "" {
	// 	return number
	// }
	// if number := leadingDigits(strings.TrimSpace(object.Name)); number != "" {
	// 	return number
	// }
	return object.DisplayNumber
}

func sourceMatchesFilter(source string, selectedSource string) bool {
	switch NormalizeObjectSourceFilter(selectedSource) {
	case ObjectSourceCASL:
		return source == ObjectSourceCASL
	case ObjectSourceBridge:
		return source == ObjectSourceBridge
	default:
		return true
	}
}

func numberFromPanelMark(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}

	if idx := strings.LastIndex(text, "#"); idx >= 0 && idx < len(text)-1 {
		if number := leadingDigits(strings.TrimSpace(text[idx+1:])); number != "" {
			return number
		}
	}
	return leadingDigits(text)
}

func leadingDigits(value string) string {
	if value == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 {
			break
		}
		if !unicode.IsSpace(r) {
			return ""
		}
	}
	return b.String()
}

// func itoa(v int) string {
// 	if v == 0 {
// 		return "0"
// 	}
// 	neg := v < 0
// 	if neg {
// 		v = -v
// 	}
// 	var buf [20]byte
// 	i := len(buf)
// 	for v > 0 {
// 		i--
// 		buf[i] = byte('0' + (v % 10))
// 		v /= 10
// 	}
// 	if neg {
// 		i--
// 		buf[i] = '-'
// 	}
// 	return string(buf[i:])
// }
