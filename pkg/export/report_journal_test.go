package export

import (
	"testing"
	"time"
)

func TestParseLaunchDate(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Time
	}{
		{
			input:    "18.11.21",
			expected: time.Date(2021, time.November, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "18.11.2021",
			expected: time.Date(2021, time.November, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "2021-11-18",
			expected: time.Date(2021, time.November, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "1001ос 18.11.21",
			expected: time.Date(2021, time.November, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "18.11.21 1001ос",
			expected: time.Date(2021, time.November, 18, 0, 0, 0, 0, time.UTC),
		},
		{
			input:    "invalid-date",
			expected: time.Time{},
		},
		{
			input:    "",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		got := parseLaunchDate(tt.input)
		if !got.Equal(tt.expected) {
			t.Errorf("parseLaunchDate(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}
