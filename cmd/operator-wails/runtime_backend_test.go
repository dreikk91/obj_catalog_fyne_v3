package main

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/config"
)

func TestResolveEnabledSourcesFallbackByMode(t *testing.T) {
	tests := []struct {
		name         string
		cfg          config.DBConfig
		wantFirebird bool
		wantPhoenix  bool
		wantCASL     bool
	}{
		{
			name: "defaults to firebird",
			cfg: config.DBConfig{
				Mode: config.BackendModeFirebird,
			},
			wantFirebird: true,
		},
		{
			name: "mode phoenix",
			cfg: config.DBConfig{
				Mode: config.BackendModePhoenix,
			},
			wantPhoenix: true,
		},
		{
			name: "mode casl",
			cfg: config.DBConfig{
				Mode: config.BackendModeCASLCloud,
			},
			wantCASL: true,
		},
		{
			name: "explicit flags preserve values",
			cfg: config.DBConfig{
				FirebirdEnabled: true,
				PhoenixEnabled:  true,
				CASLEnabled:     false,
				Mode:            config.BackendModeCASLCloud,
			},
			wantFirebird: true,
			wantPhoenix:  true,
			wantCASL:     true,
		},
		{
			name: "mode phoenix supplements firebird default",
			cfg: config.DBConfig{
				FirebirdEnabled: true,
				Mode:            config.BackendModePhoenix,
			},
			wantFirebird: true,
			wantPhoenix:  true,
		},
		{
			name: "mode casl supplements firebird default",
			cfg: config.DBConfig{
				FirebirdEnabled: true,
				Mode:            config.BackendModeCASLCloud,
			},
			wantFirebird: true,
			wantCASL:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFirebird, gotPhoenix, gotCASL := resolveEnabledSources(tt.cfg)
			if gotFirebird != tt.wantFirebird || gotPhoenix != tt.wantPhoenix || gotCASL != tt.wantCASL {
				t.Fatalf(
					"resolveEnabledSources() = (%t,%t,%t), want (%t,%t,%t)",
					gotFirebird, gotPhoenix, gotCASL,
					tt.wantFirebird, tt.wantPhoenix, tt.wantCASL,
				)
			}
		})
	}
}

func TestEnvHelpers(t *testing.T) {
	const (
		boolName   = "MOST_TEST_BOOL"
		intName    = "MOST_TEST_INT"
		stringName = "MOST_TEST_STRING"
	)

	t.Setenv(boolName, "TRUE")
	if got := envBool(boolName, false); !got {
		t.Fatalf("envBool(TRUE) = false, want true")
	}

	t.Setenv(boolName, "not-a-bool")
	if got := envBool(boolName, true); !got {
		t.Fatalf("envBool(invalid) should return fallback true")
	}

	t.Setenv(intName, "42")
	if got := envInt64(intName, 7); got != 42 {
		t.Fatalf("envInt64(42) = %d, want 42", got)
	}

	t.Setenv(intName, "bad")
	if got := envInt64(intName, 9); got != 9 {
		t.Fatalf("envInt64(invalid) = %d, want fallback 9", got)
	}

	t.Setenv(stringName, "  value  ")
	if got := envString(stringName, "fallback"); got != "value" {
		t.Fatalf("envString(trimmed) = %q, want %q", got, "value")
	}
}

func TestLoadEnvDBConfig(t *testing.T) {
	t.Setenv("MOST_DB_USER", "user_x")
	t.Setenv("MOST_DB_PASSWORD", "pass_x")
	t.Setenv("MOST_DB_HOST", "host_x")
	t.Setenv("MOST_DB_PORT", "9999")
	t.Setenv("MOST_DB_PATH", "C:/db/test.fdb")
	t.Setenv("MOST_FIREBIRD_ENABLED", "false")
	t.Setenv("MOST_PHOENIX_ENABLED", "true")
	t.Setenv("MOST_BACKEND_MODE", config.BackendModePhoenix)
	t.Setenv("MOST_CASL_ENABLED", "true")
	t.Setenv("MOST_CASL_PULT_ID", "17")

	cfg := loadEnvDBConfig()
	if cfg.User != "user_x" {
		t.Fatalf("cfg.User = %q, want %q", cfg.User, "user_x")
	}
	if cfg.Password != "pass_x" {
		t.Fatalf("cfg.Password = %q, want %q", cfg.Password, "pass_x")
	}
	if cfg.Host != "host_x" {
		t.Fatalf("cfg.Host = %q, want %q", cfg.Host, "host_x")
	}
	if cfg.Port != "9999" {
		t.Fatalf("cfg.Port = %q, want %q", cfg.Port, "9999")
	}
	if cfg.Path != "C:/db/test.fdb" {
		t.Fatalf("cfg.Path = %q, want %q", cfg.Path, "C:/db/test.fdb")
	}
	if cfg.FirebirdEnabled {
		t.Fatalf("cfg.FirebirdEnabled = true, want false")
	}
	if !cfg.PhoenixEnabled {
		t.Fatalf("cfg.PhoenixEnabled = false, want true")
	}
	if !cfg.CASLEnabled {
		t.Fatalf("cfg.CASLEnabled = false, want true")
	}
	if cfg.CASLPultID != 17 {
		t.Fatalf("cfg.CASLPultID = %d, want 17", cfg.CASLPultID)
	}
	if cfg.NormalizedMode() != config.BackendModePhoenix {
		t.Fatalf("cfg.NormalizedMode = %q, want %q", cfg.NormalizedMode(), config.BackendModePhoenix)
	}
}
