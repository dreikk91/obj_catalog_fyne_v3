package utils

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseAnyTimeRFC3339Nano(t *testing.T) {
	got := ParseAnyTime("2026-04-06T12:34:56.123456789Z")
	if got.IsZero() {
		t.Fatal("expected non-zero time for RFC3339Nano input")
	}
	if got.UTC().Format(time.RFC3339Nano) != "2026-04-06T12:34:56.123456789Z" {
		t.Fatalf("unexpected parsed time: %s", got.UTC().Format(time.RFC3339Nano))
	}
}

func TestParseAnyTimeDefaultStringFallback(t *testing.T) {
	type stringLike string

	got := ParseAnyTime(stringLike("1712406896"))
	if got.IsZero() {
		t.Fatal("expected numeric fallback to parse string-like values")
	}
	if got.Unix() != 1712406896 {
		t.Fatalf("unexpected unix timestamp: %d", got.Unix())
	}
}

func TestParseAnyTimeJSONNumberFloatString(t *testing.T) {
	got := ParseAnyTime(json.Number("1712406896.9"))
	if got.IsZero() {
		t.Fatal("expected non-zero time for JSON float number")
	}
	if got.Unix() != 1712406896 {
		t.Fatalf("unexpected unix timestamp: %d", got.Unix())
	}
}

func TestAsStringFloat64PreservesFraction(t *testing.T) {
	if got := AsString(3.14); got != "3.14" {
		t.Fatalf("AsString(3.14) = %q, want %q", got, "3.14")
	}
	if got := AsString(3.0); got != "3" {
		t.Fatalf("AsString(3.0) = %q, want %q", got, "3")
	}
}
