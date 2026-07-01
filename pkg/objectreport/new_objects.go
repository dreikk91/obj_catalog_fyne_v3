package objectreport

import (
	"sort"
	"strings"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

const (
	PeriodWeek      = "Тиждень"
	PeriodMonth     = "Місяць"
	PeriodQuarter   = "Квартал"
	PeriodHalfYear  = "Пів року"
	PeriodYear      = "Рік"
	PeriodThreeYear = "Три роки"
	PeriodFiveYear  = "П'ять років"
	PeriodCustom    = "Вибрані дати"
)

// PeriodOptions returns supported report periods in UI order.
func PeriodOptions() []string {
	return []string{
		PeriodWeek,
		PeriodMonth,
		PeriodQuarter,
		PeriodHalfYear,
		PeriodYear,
		PeriodThreeYear,
		PeriodFiveYear,
		PeriodCustom,
	}
}

// Item is an object with its parsed addition date.
type Item struct {
	Object  models.Object
	AddedAt time.Time
}

// RangeForPeriod returns an inclusive calendar range ending on now.
func RangeForPeriod(period string, now time.Time) (from, to time.Time) {
	to = dayStart(now)
	switch strings.TrimSpace(period) {
	case PeriodWeek:
		from = to.AddDate(0, 0, -7)
	case PeriodQuarter:
		from = to.AddDate(0, -3, 0)
	case PeriodHalfYear:
		from = to.AddDate(0, -6, 0)
	case PeriodYear:
		from = to.AddDate(-1, 0, 0)
	case PeriodThreeYear:
		from = to.AddDate(-3, 0, 0)
	case PeriodFiveYear:
		from = to.AddDate(-5, 0, 0)
	default:
		from = to.AddDate(0, -1, 0)
	}
	return from, to
}

// Filter returns objects whose addition date falls inside the inclusive range.
func Filter(objects []models.Object, from, to time.Time) []Item {
	from = dayStart(from)
	toExclusive := dayStart(to).AddDate(0, 0, 1)
	items := make([]Item, 0, len(objects))
	for _, object := range objects {
		addedAt, ok := ParseDate(object.LaunchDate)
		if !ok || addedAt.Before(from) || !addedAt.Before(toExclusive) {
			continue
		}
		items = append(items, Item{Object: object, AddedAt: addedAt})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].AddedAt.After(items[j].AddedAt)
	})
	return items
}

// ParseDate parses addition dates supplied by МІСТ, Phoenix and CASL.
func ParseDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		"02.01.2006",
		"02.01.2006 15:04:05",
		"2006-01-02",
		time.RFC3339,
	} {
		parsed, err := time.ParseInLocation(layout, value, time.Local)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func dayStart(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}
