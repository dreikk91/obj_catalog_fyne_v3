package config

import (
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

func TestKyivstarConfig_TokenExpiryTime(t *testing.T) {
	t.Parallel()

	want := time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC)
	cfg := KyivstarConfig{TokenExpiry: want.Format(time.RFC3339)}

	if got := cfg.TokenExpiryTime(); !got.Equal(want) {
		t.Fatalf("TokenExpiryTime() = %v, want %v", got, want)
	}
}

func TestKyivstarConfig_HasCredentials(t *testing.T) {
	t.Parallel()

	cfg := KyivstarConfig{ClientID: " client-id ", ClientSecret: " secret "}
	if !cfg.HasCredentials() {
		t.Fatal("HasCredentials() = false, want true")
	}
}

func TestKyivstarConfig_TokenUsableAt_WithoutExpiry(t *testing.T) {
	t.Parallel()

	cfg := KyivstarConfig{AccessToken: "token-without-exp"}
	if !cfg.TokenUsableAt(time.Now()) {
		t.Fatal("TokenUsableAt() = false, want true")
	}
}

func TestKyivstarConfig_AutoResetWindowMinimum(t *testing.T) {
	t.Parallel()

	cfg := LoadKyivstarConfig(nil)
	if cfg.AutoResetWindowHours != DefaultKyivstarAutoResetWindowHours {
		t.Fatalf("default AutoResetWindowHours = %d, want %d", cfg.AutoResetWindowHours, DefaultKyivstarAutoResetWindowHours)
	}

	app := test.NewApp()
	defer app.Quit()

	cfg.AutoResetWindowHours = 0
	SaveKyivstarConfig(app.Preferences(), cfg)

	got := LoadKyivstarConfig(app.Preferences())
	if got.AutoResetWindowHours != MinKyivstarAutoResetWindowHours {
		t.Fatalf("AutoResetWindowHours = %d, want minimum %d", got.AutoResetWindowHours, MinKyivstarAutoResetWindowHours)
	}
}
