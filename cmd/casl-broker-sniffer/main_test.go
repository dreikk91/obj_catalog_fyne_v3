package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"testing"

	"github.com/go-zeromq/zmq4"
)

func TestParseTopics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{name: "empty subscribes all", raw: "", want: []string{""}},
		{name: "comma list", raw: "api_in, api_out,ppk_in", want: []string{"api_in", "api_out", "ppk_in"}},
		{name: "dedupe", raw: "api_in,api_in", want: []string{"api_in"}},
		{name: "star means all", raw: "*", want: []string{""}},
		{name: "blank parts ignored", raw: " , api_in, ", want: []string{"api_in"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := parseTopics(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("len(parseTopics(%q)) = %d, want %d: %#v", tt.raw, len(got), len(tt.want), got)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("parseTopics(%q)[%d] = %q, want %q", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestDecodePayloadPlainJSON(t *testing.T) {
	t.Parallel()

	got := decodePayload([]byte(`{"type":"read_events","limit":10}`), 1024)
	if got.Encoding != "plain" {
		t.Fatalf("Encoding = %q, want plain", got.Encoding)
	}
	if got.Truncated {
		t.Fatal("Truncated = true, want false")
	}
	obj, ok := got.JSON.(map[string]any)
	if !ok {
		t.Fatalf("JSON type = %T, want map[string]any", got.JSON)
	}
	if obj["type"] != "read_events" {
		t.Fatalf("JSON[type] = %v, want read_events", obj["type"])
	}
}

func TestDecodePayloadGzipJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(`{"type":"api_out","status":"ok"}`)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	got := decodePayload(buf.Bytes(), 1024)
	if got.Encoding != "gzip" {
		t.Fatalf("Encoding = %q, want gzip", got.Encoding)
	}
	raw, err := json.Marshal(got.JSON)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"status":"ok","type":"api_out"}` {
		t.Fatalf("JSON = %s", raw)
	}
}

func TestRecordFromMessageUsesTopicAndPayloadFrame(t *testing.T) {
	t.Parallel()

	msg := zmq4.NewMsgFrom([]byte("api_in"), []byte(`{"type":"get_object"}`))
	got := recordFromMessage(msg, 1024)
	if got.Topic != "api_in" {
		t.Fatalf("Topic = %q, want api_in", got.Topic)
	}
	if got.Frames != 2 {
		t.Fatalf("Frames = %d, want 2", got.Frames)
	}
	obj, ok := got.PayloadJSON.(map[string]any)
	if !ok {
		t.Fatalf("PayloadJSON type = %T, want map[string]any", got.PayloadJSON)
	}
	if obj["type"] != "get_object" {
		t.Fatalf("PayloadJSON[type] = %v, want get_object", obj["type"])
	}
}

func TestDecodePayloadTruncatesBeforeJSON(t *testing.T) {
	t.Parallel()

	got := decodePayload([]byte(`{"long":"abcdef"}`), 5)
	if !got.Truncated {
		t.Fatal("Truncated = false, want true")
	}
	if got.Text != `{"lon` {
		t.Fatalf("Text = %q, want truncated prefix", got.Text)
	}
	if got.JSON != nil {
		t.Fatalf("JSON = %#v, want nil", got.JSON)
	}
}
