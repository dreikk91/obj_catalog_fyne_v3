//go:build qt

package qtui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestOverviewZoneStatusText(t *testing.T) {
	tests := []struct {
		name string
		zone models.Zone
		want string
	}{
		{name: "normal", zone: models.Zone{Status: models.ZoneNormal}, want: "Норм."},
		{name: "alarm", zone: models.Zone{Status: models.ZoneAlarm}, want: "Трив."},
		{name: "fire", zone: models.Zone{Status: models.ZoneFire}, want: "Пож."},
		{name: "break", zone: models.Zone{Status: models.ZoneBreak}, want: "Обр."},
		{name: "short", zone: models.Zone{Status: models.ZoneShort}, want: "КЗ"},
		{name: "bypassed", zone: models.Zone{Status: models.ZoneNormal, IsBypassed: true}, want: "Відкл."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := overviewZoneStatusText(test.zone); got != test.want {
				t.Fatalf("overviewZoneStatusText() = %q, want %q", got, test.want)
			}
		})
	}
}
