package main

import (
	"os"
	"path/filepath"
	"testing"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestLoadOrCreateGatewayConfigCreatesDefaultFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.json")

	cfg, created, err := loadOrCreateGatewayConfig(path)
	if err != nil {
		t.Fatalf("loadOrCreateGatewayConfig() error = %v", err)
	}
	if !created {
		t.Fatal("loadOrCreateGatewayConfig() created = false, want true")
	}
	if cfg.Addr != "127.0.0.1:50003" || cfg.WSAddr != "127.0.0.1:23322" {
		t.Fatalf("unexpected default server config: %#v", cfg)
	}
	if cfg.Database.Mode != config.BackendModeFirebird {
		t.Fatalf("database mode = %q, want %q", cfg.Database.Mode, config.BackendModeFirebird)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("generated config file missing: %v", err)
	}
}

func TestLoadOrCreateGatewayConfigReadsExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gateway.json")
	body := []byte(`{
  "addr": "127.0.0.1:50100",
  "ws_addr": "127.0.0.1:23400",
  "data_source": "config",
  "shutdown_timeout": "3s",
  "database": {
    "mode": "casl_cloud",
    "casl_enabled": true,
    "casl_base_url": "http://127.0.0.1:50003"
  }
}`)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, created, err := loadOrCreateGatewayConfig(path)
	if err != nil {
		t.Fatalf("loadOrCreateGatewayConfig() error = %v", err)
	}
	if created {
		t.Fatal("loadOrCreateGatewayConfig() created = true, want false")
	}
	if cfg.Addr != "127.0.0.1:50100" || cfg.DataSource != "config" {
		t.Fatalf("existing config was not applied: %#v", cfg)
	}
	if cfg.Database.User != "SYSDBA" {
		t.Fatalf("database defaults were not applied: %#v", cfg.Database)
	}
	if got := cfg.ShutdownTimeout.Duration().String(); got != "3s" {
		t.Fatalf("shutdown timeout = %s, want 3s", got)
	}
}
