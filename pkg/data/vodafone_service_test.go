package data

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"obj_catalog_fyne_v3/pkg/config"
	"testing"
	"time"
)

type vodafoneConfigStoreStub struct {
	cfg config.VodafoneConfig
}

func (s *vodafoneConfigStoreStub) LoadVodafoneConfig() config.VodafoneConfig {
	return s.cfg
}

func (s *vodafoneConfigStoreStub) SaveVodafoneConfig(cfg config.VodafoneConfig) {
	s.cfg = cfg
}

func TestNormalizeVodafoneMSISDN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ua local", input: "0501234567", want: "380501234567"},
		{name: "intl", input: "380501234567", want: "380501234567"},
		{name: "formatted", input: "+38 (050) 123-45-67", want: "380501234567"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeVodafoneMSISDN(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("normalizeVodafoneMSISDN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestVodafoneService_VerifyLoginPersistsToken(t *testing.T) {
	t.Parallel()

	exp := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	token := buildTestJWT(exp)
	store := &vodafoneConfigStoreStub{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/uaa/oauth/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": token,
		})
	}))
	defer server.Close()

	service := NewVodafoneService(store, WithVodafoneBaseURL(server.URL))
	state, err := service.VerifyLogin("0501234567", "1234")
	if err != nil {
		t.Fatalf("VerifyLogin() error = %v", err)
	}
	if !state.Authorized {
		t.Fatalf("expected authorized state")
	}
	if store.cfg.Phone != "380501234567" {
		t.Fatalf("unexpected stored phone: %q", store.cfg.Phone)
	}
	if store.cfg.AccessToken != token {
		t.Fatalf("unexpected stored token")
	}
	if store.cfg.TokenExpiry != exp.Format(time.RFC3339) {
		t.Fatalf("unexpected token expiry: %q", store.cfg.TokenExpiry)
	}
}

func TestVodafoneService_GetSIMStatus_UsesAvailableIOTList(t *testing.T) {
	t.Parallel()

	store := &vodafoneConfigStoreStub{
		cfg: config.VodafoneConfig{
			Phone:       "380501234567",
			AccessToken: buildTestJWT(time.Now().UTC().Add(2 * time.Hour)),
			TokenExpiry: time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/customer/api/customerManagement/v3/customer":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"account": map[string]any{"id": "295398767704"},
					"relatedParty": []map[string]any{
						{
							"id": "380501234567",
							"characterictics": []map[string]any{
								{"name": "phoneDescription", "value": "Obj 1001"},
							},
						},
					},
				},
			})
		case r.URL.Path == "/customer/api/customerManagement/v3/customer/self" && r.Header.Get("Profile") == "CONNECTIVITY-CHECK-BY-MSISDN":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"relatedParty": []map[string]any{
					{
						"id": "380501234567",
						"characteristics": []map[string]any{
							{"name": "status", "value": "SUCCESS"},
							{"name": "statusSIM", "value": "active"},
							{"name": "statusBS", "value": "active"},
							{"name": "lbsStatusKey", "value": "done"},
							{"name": "connectionTime", "value": "02-04-2026 10:11:12"},
						},
					},
				},
			})
		case r.URL.Path == "/customer/api/customerManagement/v3/customer/self" && r.Header.Get("Profile") == "LASTEVENT-MSISDN-M2M":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"relatedParty": []map[string]any{
						{
							"id": "380501234567",
							"characterictics": []map[string]any{
								{"name": "callType", "value": "DATA_TRANSFER"},
								{"name": "lastEventTime", "value": "2026-04-02T09:10:11Z"},
							},
						},
					},
				},
			})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	service := NewVodafoneService(store, WithVodafoneBaseURL(server.URL))
	status, err := service.GetSIMStatus("0501234567")
	if err != nil {
		t.Fatalf("GetSIMStatus() error = %v", err)
	}
	if !status.Available {
		t.Fatalf("expected SIM to be available")
	}
	if status.SubscriberName != "Obj 1001" {
		t.Fatalf("unexpected subscriber name: %q", status.SubscriberName)
	}
	if status.Connectivity.SIMStatus != "active" {
		t.Fatalf("unexpected SIM status: %q", status.Connectivity.SIMStatus)
	}
	if status.LastEvent.CallType != "DATA_TRANSFER" {
		t.Fatalf("unexpected call type: %q", status.LastEvent.CallType)
	}
}

func buildTestJWT(exp time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"exp": exp.Unix()})
	body := base64.RawURLEncoding.EncodeToString(payload)
	return header + "." + body + "."
}
