package application

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestBuildJournalLayoutPlan(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.UIConfig
		want journalLayoutPlan
	}{
		{
			name: "all journals on right by default",
			cfg:  config.UIConfig{},
			want: journalLayoutPlan{
				rightShowsEvents: true,
				rightShowsAlarms: true,
			},
		},
		{
			name: "only alarms moved to bottom",
			cfg: config.UIConfig{
				ShowBottomAlarmJournal: true,
			},
			want: journalLayoutPlan{
				rightShowsEvents:  true,
				bottomShowsAlarms: true,
			},
		},
		{
			name: "only events moved to bottom",
			cfg: config.UIConfig{
				ShowBottomEventJournal: true,
			},
			want: journalLayoutPlan{
				rightShowsAlarms:  true,
				bottomShowsEvents: true,
			},
		},
		{
			name: "both journals moved to bottom",
			cfg: config.UIConfig{
				ShowBottomAlarmJournal: true,
				ShowBottomEventJournal: true,
			},
			want: journalLayoutPlan{
				bottomShowsEvents: true,
				bottomShowsAlarms: true,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := buildJournalLayoutPlan(tt.cfg)
			if got != tt.want {
				t.Fatalf("buildJournalLayoutPlan(%+v) = %+v, want %+v", tt.cfg, got, tt.want)
			}
		})
	}
}
