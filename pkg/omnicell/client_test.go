package omnicell

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestNormalizeMSISDN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "local", input: "067 123-45-67", want: "380671234567"},
		{name: "plus", input: "+380671234567", want: "380671234567"},
		{name: "international", input: "380671234567", want: "380671234567"},
		{name: "invalid", input: "123", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := NormalizeMSISDN(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeMSISDN(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeMSISDN(%q) error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeMSISDN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestClientSendSMS(t *testing.T) {
	t.Parallel()

	var (
		gotAuth        string
		gotContentType string
		gotBody        string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request body: %v", err)
		}
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<response><status>accepted</status></response>"))
	}))
	defer server.Close()

	client := NewClientWithHTTPClient(config.OmnicellConfig{
		Endpoint: server.URL,
		Login:    "login",
		Password: "password",
		Source:   "Alarm",
	}, server.Client())

	resp, err := client.SendSMS(context.Background(), SendRequest{Phone: "0671234567", Text: "Test <ok>"})
	if err != nil {
		t.Fatalf("SendSMS error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if !strings.HasPrefix(gotContentType, "text/xml") {
		t.Fatalf("Content-Type = %q", gotContentType)
	}
	if gotAuth == "" {
		t.Fatal("Authorization header is empty")
	}
	for _, want := range []string{
		`<service id="single" source="Alarm" type="SMS"></service>`,
		`<to>380671234567</to>`,
		`<body content-type="text/plain">Test &lt;ok&gt;</body>`,
	} {
		if !strings.Contains(gotBody, want) {
			t.Fatalf("request body missing %q in %s", want, gotBody)
		}
	}
}
