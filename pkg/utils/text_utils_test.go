package utils

import "testing"

func TestHasCyrillicChars(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "latin only",
			input: "Alarm panel",
			want:  false,
		},
		{
			name:  "ukrainian letters",
			input: "Тривога на об'єкті",
			want:  true,
		},
		{
			name:  "extended cyrillic letter",
			input: "Ѓ",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := HasCyrillicChars(tt.input); got != tt.want {
				t.Fatalf("HasCyrillicChars(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
