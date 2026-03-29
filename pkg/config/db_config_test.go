package config

import "testing"

func TestNormalizeBackendMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "firebird default", input: "", want: BackendModeFirebird},
		{name: "firebird explicit", input: BackendModeFirebird, want: BackendModeFirebird},
		{name: "casl explicit", input: BackendModeCASLCloud, want: BackendModeCASLCloud},
		{name: "casl mixed case", input: "  CASL_CLOUD  ", want: BackendModeCASLCloud},
		{name: "unknown fallback", input: "unknown", want: BackendModeFirebird},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeBackendMode(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeBackendMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDBConfig_NormalizedMode(t *testing.T) {
	t.Parallel()

	cfg := DBConfig{Mode: BackendModeCASLCloud}
	if got := cfg.NormalizedMode(); got != BackendModeCASLCloud {
		t.Fatalf("NormalizedMode() = %q, want %q", got, BackendModeCASLCloud)
	}

	cfg.Mode = "invalid"
	if got := cfg.NormalizedMode(); got != BackendModeFirebird {
		t.Fatalf("NormalizedMode() fallback = %q, want %q", got, BackendModeFirebird)
	}
}
