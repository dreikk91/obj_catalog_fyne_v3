package data

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"obj_catalog_fyne_v3/pkg/config"
	"strings"
	"testing"
	"time"
)

type kyivstarConfigStoreStub struct {
	cfg config.KyivstarConfig
}

func (s *kyivstarConfigStoreStub) LoadKyivstarConfig() config.KyivstarConfig {
	return s.cfg
}

func (s *kyivstarConfigStoreStub) SaveKyivstarConfig(cfg config.KyivstarConfig) {
	s.cfg = cfg
}

func TestNormalizeKyivstarMSISDN(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "local 067", input: "0671234567", want: "380671234567"},
		{name: "intl 097", input: "+380971234567", want: "380971234567"},
		{name: "formatted 077", input: "+38 (077) 123-45-67", want: "380771234567"},
		{name: "unsupported operator", input: "0631234567", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeKyivstarMSISDN(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeKyivstarMSISDN(%q) error = nil, want error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeKyivstarMSISDN(%q) error = %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeKyivstarMSISDN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestKyivstarService_RefreshToken_PersistsToken(t *testing.T) {
	t.Parallel()

	store := &kyivstarConfigStoreStub{
		cfg: config.KyivstarConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/idp/oauth2/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got == "" {
			t.Fatal("expected Basic Authorization header")
		}
		if err := json.NewEncoder(w).Encode(map[string]string{
			"access_token": "ks-token",
			"expires_in":   "3600",
		}); err != nil {
			t.Fatalf("encode token response: %v", err)
		}
	}))
	defer server.Close()

	service := NewKyivstarService(store, WithKyivstarBaseURL(server.URL))

	state, err := service.RefreshToken()
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if !state.Authorized {
		t.Fatal("RefreshToken() authorized = false, want true")
	}
	if got := strings.TrimSpace(store.cfg.AccessToken); got != "ks-token" {
		t.Fatalf("stored access token = %q, want %q", got, "ks-token")
	}
	if store.cfg.TokenExpiry == "" {
		t.Fatal("expected token expiry to be persisted")
	}
}

func TestKyivstarService_RefreshToken_AcceptsNumericExpiresIn(t *testing.T) {
	t.Parallel()

	store := &kyivstarConfigStoreStub{
		cfg: config.KyivstarConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]any{
			"access_token": "ks-token",
			"expires_in":   28799,
		}); err != nil {
			t.Fatalf("encode token response: %v", err)
		}
	}))
	defer server.Close()

	service := NewKyivstarService(store, WithKyivstarBaseURL(server.URL))

	state, err := service.RefreshToken()
	if err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
	if !state.Authorized {
		t.Fatal("RefreshToken() authorized = false, want true")
	}
	if store.cfg.TokenExpiry == "" {
		t.Fatal("expected token expiry to be persisted")
	}
}

func TestKyivstarService_GetSIMStatus_AggregatesInfo(t *testing.T) {
	t.Parallel()

	store := &kyivstarConfigStoreStub{
		cfg: config.KyivstarConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AccessToken:  "cached-token",
			TokenExpiry:  time.Now().Add(30 * time.Minute).Format(time.RFC3339),
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer cached-token" {
			t.Fatalf("Authorization = %q, want Bearer cached-token", got)
		}
		switch r.URL.Path {
		case "/rest/iot/company-numbers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{
					{
						"number":       "380671234567",
						"deviceName":   "Obj 1001",
						"deviceId":     "1001",
						"iccid":        "iccid-1",
						"imei":         "imei-1",
						"tariffPlan":   "IoT Base",
						"account":      "ACC-1",
						"dataUsage":    "1024",
						"smsUsage":     "3",
						"voiceUsage":   "0",
						"isOnline":     true,
						"isTestPeriod": false,
					},
				},
			})
		case "/rest/iot/company-numbers/statuses":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":           "ACTIVE",
				"availableActions": []string{"pause"},
			})
		case "/rest/iot/company-numbers/services":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"serviceId":        "10",
					"name":             "DATA",
					"status":           "ACTIVE",
					"availableActions": []string{"pause"},
				},
			})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := NewKyivstarService(store, WithKyivstarBaseURL(server.URL))

	status, err := service.GetSIMStatus("0671234567")
	if err != nil {
		t.Fatalf("GetSIMStatus() error = %v", err)
	}
	if !status.Available {
		t.Fatal("Available = false, want true")
	}
	if status.MSISDN != "380671234567" {
		t.Fatalf("MSISDN = %q, want %q", status.MSISDN, "380671234567")
	}
	if status.DeviceName != "Obj 1001" {
		t.Fatalf("DeviceName = %q, want %q", status.DeviceName, "Obj 1001")
	}
	if status.NumberStatus != "ACTIVE" {
		t.Fatalf("NumberStatus = %q, want %q", status.NumberStatus, "ACTIVE")
	}
	if len(status.Services) != 1 || status.Services[0].ServiceID != "10" {
		t.Fatalf("unexpected services: %+v", status.Services)
	}
}

func TestKyivstarService_PauseSIMServices_SendsSelectedIDs(t *testing.T) {
	t.Parallel()

	store := &kyivstarConfigStoreStub{
		cfg: config.KyivstarConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			AccessToken:  "cached-token",
			TokenExpiry:  time.Now().Add(30 * time.Minute).Format(time.RFC3339),
		},
	}

	var bodyText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/iot/company-numbers":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"content": []map[string]any{{"number": "380971234567"}},
			})
		case "/rest/iot/company-numbers/services":
			payload, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}
			bodyText = string(payload)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	service := NewKyivstarService(store, WithKyivstarBaseURL(server.URL))

	result, err := service.PauseSIMServices("0971234567", []string{"10", "20"})
	if err != nil {
		t.Fatalf("PauseSIMServices() error = %v", err)
	}
	if result.MSISDN != "380971234567" {
		t.Fatalf("MSISDN = %q, want %q", result.MSISDN, "380971234567")
	}
	if !strings.Contains(bodyText, `"serviceId":"10"`) || !strings.Contains(bodyText, `"serviceId":"20"`) {
		t.Fatalf("request body does not contain selected service ids: %s", bodyText)
	}
	if !strings.Contains(bodyText, `"action":"pause"`) {
		t.Fatalf("request body does not contain pause action: %s", bodyText)
	}
}

func TestKyivstarService_UpdateSIMMetadata_Unsupported(t *testing.T) {
	t.Parallel()

	service := NewKyivstarService(&kyivstarConfigStoreStub{})
	err := service.UpdateSIMMetadata("0671234567", "Obj 1001", "1001")
	if !errorsIs(err, errKyivstarMetadataUnsupported) {
		t.Fatalf("UpdateSIMMetadata() error = %v, want %v", err, errKyivstarMetadataUnsupported)
	}
}

func errorsIs(err error, target error) bool {
	return err != nil && target != nil && strings.Contains(err.Error(), target.Error())
}

func TestKyivstarService_RefreshToken_SendsBasicAuth(t *testing.T) {
	t.Parallel()

	store := &kyivstarConfigStoreStub{
		cfg: config.KyivstarConfig{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Fatalf("Authorization = %q, want %q", got, wantAuth)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "ks-token",
			"expires_in":   "60",
		})
	}))
	defer server.Close()

	service := NewKyivstarService(store, WithKyivstarBaseURL(server.URL))
	if _, err := service.RefreshToken(); err != nil {
		t.Fatalf("RefreshToken() error = %v", err)
	}
}
