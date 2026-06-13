package config

import (
	"testing"
	"time"
)

func TestVodafoneConfig_TokenExpiryTime(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, time.April, 2, 15, 4, 5, 0, time.UTC)
	cfg := VodafoneConfig{TokenExpiry: want.Format(time.RFC3339)}

	got := cfg.TokenExpiryTime()
	if !got.Equal(want) {
		t.Fatalf("TokenExpiryTime() = %v, want %v", got, want)
	}
}

func TestVodafoneConfig_TokenExpiryTime_Invalid(t *testing.T) {
	t.Parallel()

	cfg := VodafoneConfig{TokenExpiry: "invalid"}
	if got := cfg.TokenExpiryTime(); !got.IsZero() {
		t.Fatalf("TokenExpiryTime() must return zero time for invalid value, got %v", got)
	}
}

func TestVodafoneConfig_TokenUsableAt_WithoutExpiry(t *testing.T) {
	t.Parallel()

	cfg := VodafoneConfig{AccessToken: "token-without-exp"}
	if !cfg.TokenUsableAt(time.Now()) {
		t.Fatalf("TokenUsableAt() must accept stored token without expiry")
	}
}

func TestVodafoneConfig_NormalizedLoginMethod(t *testing.T) {
	t.Parallel()

	if got := (VodafoneConfig{}).NormalizedLoginMethod(); got != VodafoneLoginMethodSMS {
		t.Fatalf("empty LoginMethod = %q, want %q", got, VodafoneLoginMethodSMS)
	}
	if got := (VodafoneConfig{LoginMethod: " PUK "}).NormalizedLoginMethod(); got != VodafoneLoginMethodPUK {
		t.Fatalf("PUK LoginMethod = %q, want %q", got, VodafoneLoginMethodPUK)
	}
}
