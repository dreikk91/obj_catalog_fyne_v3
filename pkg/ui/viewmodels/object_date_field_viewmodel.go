package viewmodels

import (
	"strings"
	"time"
)

const objectDateDisplayLayout = "02.01.2006"

// ObjectDateFieldViewModel інкапсулює правила парсингу/форматування дати форми об'єкта.
type ObjectDateFieldViewModel struct{}

func NewObjectDateFieldViewModel() *ObjectDateFieldViewModel {
	return &ObjectDateFieldViewModel{}
}

func (vm *ObjectDateFieldViewModel) Parse(raw string) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}

	formats := []string{
		objectDateDisplayLayout,
		"2006-01-02",
		"2006-01-02 15:04:05",
		time.RFC3339,
	}
	for _, format := range formats {
		if parsed, err := time.ParseInLocation(format, raw, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func (vm *ObjectDateFieldViewModel) ResolvePickerInitial(raw string, fallback time.Time) time.Time {
	if parsed, ok := vm.Parse(raw); ok {
		return parsed
	}
	return fallback
}

func (vm *ObjectDateFieldViewModel) FormatForDisplay(value time.Time) string {
	return value.Format(objectDateDisplayLayout)
}
