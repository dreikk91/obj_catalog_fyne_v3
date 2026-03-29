package data

import (
	"obj_catalog_fyne_v3/pkg/data/casl"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty uses default",
			input: "",
			want:  casl.DefaultBaseURL,
		},
		{
			name:  "adds http scheme",
			input: "10.32.1.221:50003",
			want:  "http://10.32.1.221:50003",
		},
		{
			name:  "trims trailing slash",
			input: "http://10.32.1.221:50003/",
			want:  "http://10.32.1.221:50003",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := casl.NewAPIClient(tt.input, "", 0)
			got := client.BaseURL()
			if got != tt.want {
				t.Fatalf("normalizeBaseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsCASLObjectID(t *testing.T) {
	t.Parallel()
	if !isCASLObjectID(casl.ObjectIDNamespaceStart + 100) {
		t.Error("expected true for ID within namespace")
	}
	if isCASLObjectID(casl.ObjectIDNamespaceStart - 1) {
		t.Error("expected false for ID below namespace")
	}
}
