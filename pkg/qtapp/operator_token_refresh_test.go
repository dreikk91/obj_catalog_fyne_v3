//go:build qt

package qtapp

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestShouldRefreshKyivstarTokenRequiresCredentials(t *testing.T) {
	t.Parallel()

	at := time.Now().UTC().Add(15 * time.Minute)
	cfg := config.KyivstarConfig{
		ClientID:     "client",
		ClientSecret: "secret",
		AccessToken:  "old-token",
		TokenExpiry:  at.Add(-time.Minute).Format(time.RFC3339),
	}
	if !shouldRefreshKyivstarToken(cfg, at) {
		t.Fatal("shouldRefreshKyivstarToken() = false, want true")
	}

	cfg.ClientSecret = ""
	if shouldRefreshKyivstarToken(cfg, at) {
		t.Fatal("Kyivstar token refresh must require client credentials")
	}
}
