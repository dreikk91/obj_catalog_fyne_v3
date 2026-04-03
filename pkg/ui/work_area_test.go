package ui

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/simoperator"
)

func TestSIMOperatorDetection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  simoperator.Operator
	}{
		{name: "050 local", input: "0501234567", want: simoperator.Vodafone},
		{name: "38095 intl", input: "380951234567", want: simoperator.Vodafone},
		{name: "formatted 099", input: "+38 (099) 123-45-67", want: simoperator.Vodafone},
		{name: "kyivstar", input: "0671234567", want: simoperator.Kyivstar},
		{name: "lifecell", input: "0631234567", want: simoperator.Unknown},
		{name: "empty", input: "", want: simoperator.Unknown},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := simoperator.Detect(tt.input); got != tt.want {
				t.Fatalf("Detect(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
