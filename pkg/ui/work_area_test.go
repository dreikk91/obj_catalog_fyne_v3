package ui

import "testing"

func TestIsVodafonePhone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "050 local", input: "0501234567", want: true},
		{name: "38095 intl", input: "380951234567", want: true},
		{name: "formatted 099", input: "+38 (099) 123-45-67", want: true},
		{name: "kyivstar", input: "0671234567", want: false},
		{name: "lifecell", input: "0631234567", want: false},
		{name: "empty", input: "", want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isVodafonePhone(tt.input); got != tt.want {
				t.Fatalf("isVodafonePhone(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
