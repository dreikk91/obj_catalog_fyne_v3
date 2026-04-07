package viewmodels

import (
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
)

const (
	ObjectSourceAll     = "Всі джерела"
	ObjectSourceBridge  = "БД/МІСТ"
	ObjectSourcePhoenix = "Phoenix"
	ObjectSourceCASL    = "CASL Cloud"
)

func ObjectSourceByID(id int) string {
	if ids.IsPhoenixObjectID(id) {
		return ObjectSourcePhoenix
	}
	if ids.IsCASLObjectID(id) {
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
	case strings.ToLower(ObjectSourcePhoenix), "phoenix":
		return ObjectSourcePhoenix
	case strings.ToLower(ObjectSourceCASL), "casl":
		return ObjectSourceCASL
	case strings.ToLower(ObjectSourceBridge), "bridge", "db", "міст", "бд/міст", "db/bridge":
		return ObjectSourceBridge
	default:
		return ObjectSourceAll
	}
}

func BuildObjectSourceOptions(countAll int, countBridge int, countPhoenix int, countCASL int) []string {
	return []string{
		ObjectSourceAll + " (" + strconv.Itoa(countAll) + ")",
		ObjectSourceBridge + " (" + strconv.Itoa(countBridge) + ")",
		ObjectSourcePhoenix + " (" + strconv.Itoa(countPhoenix) + ")",
		ObjectSourceCASL + " (" + strconv.Itoa(countCASL) + ")",
	}
}

func SourceBadgeForObjectID(id int) string {
	if ids.IsPhoenixObjectID(id) {
		return "[P]"
	}
	if ids.IsCASLObjectID(id) {
		return "[C]"
	}
	return "[М]"
}

func ObjectDisplayNumber(object models.Object) string {
	if strings.TrimSpace(object.DisplayNumber) != "" {
		return object.DisplayNumber
	}
	if !ids.IsCASLObjectID(object.ID) && !ids.IsPhoenixObjectID(object.ID) {
		return strconv.Itoa(object.ID)
	}
	if number := numberFromPanelMark(object.PanelMark); number != "" {
		return number
	}
	if number := utils.LeadingDigits(strings.TrimSpace(object.Name)); number != "" {
		return number
	}
	return strconv.Itoa(object.ID)
}

func sourceMatchesFilter(source string, selectedSource string) bool {
	switch NormalizeObjectSourceFilter(selectedSource) {
	case ObjectSourcePhoenix:
		return source == ObjectSourcePhoenix
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
		if number := utils.LeadingDigits(strings.TrimSpace(text[idx+1:])); number != "" {
			return number
		}
	}
	return utils.LeadingDigits(text)
}
