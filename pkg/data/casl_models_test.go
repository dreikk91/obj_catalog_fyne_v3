package data

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCASLInt64UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     string
		want        int64
		wantErrPart string
	}{
		{
			name:    "quoted integer",
			payload: `"42"`,
			want:    42,
		},
		{
			name:    "float token truncates to int",
			payload: `42.9`,
			want:    42,
		},
		{
			name:        "invalid string fails",
			payload:     `"oops"`,
			wantErrPart: `invalid numeric value "oops"`,
		},
		{
			name:        "object token fails",
			payload:     `{}`,
			wantErrPart: `invalid numeric token {}`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var value caslInt64
			err := json.Unmarshal([]byte(tt.payload), &value)
			if tt.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrPart, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value.Int64() != tt.want {
				t.Fatalf("unexpected value: got %d, want %d", value.Int64(), tt.want)
			}
		})
	}
}

func TestCASLTextUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     string
		want        string
		wantErrPart string
	}{
		{
			name:    "plain string",
			payload: `"device-23"`,
			want:    "device-23",
		},
		{
			name:    "number stringifies",
			payload: `23`,
			want:    "23",
		},
		{
			name:        "object rejected",
			payload:     `{"nested":true}`,
			wantErrPart: `casl text: expected scalar`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var value caslText
			err := json.Unmarshal([]byte(tt.payload), &value)
			if tt.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrPart, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if value.String() != tt.want {
				t.Fatalf("unexpected value: got %q, want %q", value.String(), tt.want)
			}
		})
	}
}

func TestCASLNullableFloat64UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		payload     string
		want        *float64
		wantErrPart string
	}{
		{
			name:    "quoted float",
			payload: `"1.75"`,
			want:    float64Ptr(1.75),
		},
		{
			name:    "null clears value",
			payload: `null`,
			want:    nil,
		},
		{
			name:        "invalid string fails",
			payload:     `"oops"`,
			wantErrPart: `invalid numeric value "oops"`,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var value caslNullableFloat64
			err := json.Unmarshal([]byte(tt.payload), &value)
			if tt.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErrPart, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := value.Float64Ptr()
			switch {
			case tt.want == nil && got != nil:
				t.Fatalf("expected nil value, got %v", *got)
			case tt.want != nil && got == nil:
				t.Fatalf("expected %v, got nil", *tt.want)
			case tt.want != nil && got != nil && *got != *tt.want:
				t.Fatalf("expected %v, got %v", *tt.want, *got)
			}
		})
	}
}

func TestCASLConnectionRecordUnmarshalJSONRejectsBrokenNestedPayload(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"guardedObject":{
			"obj_id":"24",
			"name":"Obj 24",
			"device_id":"23",
			"device_number":"oops"
		}
	}`)

	var record caslConnectionRecord
	err := json.Unmarshal(raw, &record)
	if err == nil || !strings.Contains(err.Error(), `casl connection guarded object`) {
		t.Fatalf("expected guarded object decode error, got %v", err)
	}
}

func TestDecodeCASLLinePayloads_SharedMappings(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"5": map[string]any{
			"line_id":        173,
			"line_number":    5,
			"group_number":   1,
			"adapter_type":   "SYS",
			"adapter_number": 0,
			"description":    "Штора вікна тил",
			"line_type":      "EMPTY",
			"isBlocked":      true,
			"room_id":        "36",
		},
		"2": "Пожежна зона",
	}

	decoded := decodeCASLLinePayloads(raw)
	if len(decoded) != 2 {
		t.Fatalf("expected 2 decoded lines, got %d", len(decoded))
	}
	if decoded[0].LineNumber != 2 || decoded[1].LineNumber != 5 {
		t.Fatalf("unexpected line order: %+v", decoded)
	}

	deviceLine := mapCASLDecodedLineToDeviceLine(decoded[1])
	if deviceLine.ID.Int64() != 173 || deviceLine.Number.Int64() != 5 {
		t.Fatalf("unexpected device line identity: %+v", deviceLine)
	}
	if deviceLine.Name.String() != "Штора вікна тил" || deviceLine.Type.String() != "EMPTY" {
		t.Fatalf("unexpected device line aliases: %+v", deviceLine)
	}
	if !deviceLine.IsBlocked || deviceLine.RoomID.String() != "36" {
		t.Fatalf("unexpected device line flags: %+v", deviceLine)
	}

	editorLine := mapCASLDecodedLineToEditorLine(decoded[1])
	if editorLine.LineID == nil || *editorLine.LineID != 173 {
		t.Fatalf("unexpected editor line id: %+v", editorLine)
	}
	if editorLine.LineNumber != 5 || editorLine.Description != "Штора вікна тил" {
		t.Fatalf("unexpected editor line details: %+v", editorLine)
	}
	if !editorLine.IsBlocked || editorLine.RoomID != "36" {
		t.Fatalf("unexpected editor line flags: %+v", editorLine)
	}
}

func TestCASLProvider_ReadConnectionsRejectsBrokenNestedDevice(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case caslLoginPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","token":"token-connections","user_id":"1","ws_url":"ws://localhost:23322"}`))
		case caslCommandPath:
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok","data":[{"guardedObject":{"obj_id":"24","name":"Obj 24","device_id":"23","device_number":1003},"device":{"device_id":"23","number":"oops"}}]}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	provider := NewCASLCloudProvider(server.URL, "", 1, "test@lot.lviv.ua", "test123")
	_, err := provider.readConnections(context.Background())
	if err == nil || !strings.Contains(err.Error(), "casl read_connections: decode rows") || !strings.Contains(err.Error(), `invalid numeric value "oops"`) {
		t.Fatalf("expected nested decode error, got %v", err)
	}
}

func float64Ptr(v float64) *float64 {
	return &v
}
