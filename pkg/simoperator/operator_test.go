package simoperator

import "testing"

func TestDetect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  Operator
	}{
		{name: "vodafone local", input: "0501234567", want: Vodafone},
		{name: "vodafone intl", input: "+380991234567", want: Vodafone},
		{name: "kyivstar local", input: "0671234567", want: Kyivstar},
		{name: "kyivstar intl", input: "380971234567", want: Kyivstar},
		{name: "lifecell local", input: "0631234567", want: Lifecell},
		{name: "lifecell intl", input: "380931234567", want: Lifecell},
		{name: "unknown", input: "0911234567", want: Unknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Detect(tt.input); got != tt.want {
				t.Fatalf("Detect(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
