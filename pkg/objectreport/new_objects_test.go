package objectreport

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestFilterNewObjectsUsesInclusiveDatesAndSortsNewestFirst(t *testing.T) {
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.Local)
	objects := []models.Object{
		{ID: 1, LaunchDate: "01.06.2026"},
		{ID: 2, LaunchDate: "30.06.2026"},
		{ID: 3, LaunchDate: "31.05.2026"},
		{ID: 4, LaunchDate: ""},
	}

	got := Filter(objects, from, to)

	if len(got) != 2 || got[0].Object.ID != 2 || got[1].Object.ID != 1 {
		t.Fatalf("Filter() = %+v, want objects 2, 1", got)
	}
}

func TestRangeForPeriodQuarter(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.Local)
	from, to := RangeForPeriod(PeriodQuarter, now)
	if want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.Local); !from.Equal(want) {
		t.Fatalf("from = %v, want %v", from, want)
	}
	if to.Hour() != 0 || to.Day() != 1 || to.Month() != time.July {
		t.Fatalf("to = %v", to)
	}
}
