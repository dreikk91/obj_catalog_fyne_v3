package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"obj_catalog_fyne_v3/pkg/broker"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
)

func main() {
	cfgPath := flag.String("config", "casl-bridge.json", "JSON config path; created with defaults when missing")
	flag.Parse()

	cfg, created, err := loadOrCreateConfig(strings.TrimSpace(*cfgPath))
	if err != nil {
		log.Fatal(err)
	}
	if created {
		log.Printf("created default config: %s", *cfgPath)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
	log.Println("casl-bridge stopped")
}

func run(ctx context.Context, cfg bridgeConfig) error {
	// ── data sources ──────────────────────────────────────────────────────────
	var fbProvider *data.DBDataProvider
	var phProvider *data.PhoenixDataProvider
	var cleanups []func()

	if cfg.Database.FirebirdEnabled {
		dbCfg := cfg.Database.toDBConfig()
		db, err := database.InitNamedDB("firebirdsql", dbCfg.FirebirdDSN(), "МІСТ")
		if err != nil {
			return fmt.Errorf("firebird connect: %w", err)
		}
		cleanups = append(cleanups, func() { _ = db.Close() })
		fbProvider = data.NewDBDataProvider(db, dbCfg.FirebirdDSN())
		log.Println("casl-bridge: Firebird connected")
	}

	if cfg.Database.PhoenixEnabled {
		dbCfg := cfg.Database.toDBConfig()
		db, err := database.InitNamedDB("sqlserver", dbCfg.PhoenixDSN(), "Phoenix")
		if err != nil {
			return fmt.Errorf("phoenix connect: %w", err)
		}
		cleanups = append(cleanups, func() { _ = db.Close() })
		phProvider = data.NewPhoenixDataProvider(db, dbCfg.PhoenixDSN())
		log.Println("casl-bridge: Phoenix connected")
	}

	defer func() {
		for _, fn := range cleanups {
			fn()
		}
	}()

	if fbProvider == nil && phProvider == nil {
		return fmt.Errorf("no data source enabled; set firebird_enabled or phoenix_enabled in config")
	}

	// ── broker ────────────────────────────────────────────────────────────────
	client, err := broker.New(ctx,
		cfg.BrokerHost,
		cfg.BrokerPubPort,
		cfg.BrokerSubPort,
		"casl-bridge",
	)
	if err != nil {
		return fmt.Errorf("broker connect: %w", err)
	}
	defer client.Close()
	log.Printf("casl-bridge: broker connected %s pub=%d sub=%d", cfg.BrokerHost, cfg.BrokerPubPort, cfg.BrokerSubPort)

	// ── bridge ────────────────────────────────────────────────────────────────
	b := newBridge(client, fbProvider, phProvider, cfg)
	b.Run(ctx)
	return ctx.Err()
}

// ── config ────────────────────────────────────────────────────────────────────

type bridgeConfig struct {
	BrokerHost           string         `json:"broker_host"`
	BrokerPubPort        int            `json:"broker_pub_port"`
	BrokerSubPort        int            `json:"broker_sub_port"`
	PultID               int            `json:"pult_id"`
	PublishInitialEvents bool           `json:"publish_initial_events"`
	DisableDeviceSync    bool           `json:"disable_existing_device_sync"`
	PhoenixProbeInterval configDuration `json:"phoenix_probe_interval"`
	PollInterval         configDuration `json:"poll_interval"`
	HeartbeatInterval    configDuration `json:"heartbeat_interval"`
	DeviceTimeout        configDuration `json:"device_timeout"`
	Database             bridgeDBConfig `json:"database"`
}

type bridgeDBConfig struct {
	FirebirdEnabled bool   `json:"firebird_enabled"`
	User            string `json:"user"`
	Password        string `json:"password"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	Path            string `json:"path"`
	Params          string `json:"params"`
	PhoenixEnabled  bool   `json:"phoenix_enabled"`
	PhoenixUser     string `json:"phoenix_user"`
	PhoenixPassword string `json:"phoenix_password"`
	PhoenixHost     string `json:"phoenix_host"`
	PhoenixPort     string `json:"phoenix_port"`
	PhoenixInstance string `json:"phoenix_instance"`
	PhoenixDatabase string `json:"phoenix_database"`
	PhoenixParams   string `json:"phoenix_params"`
}

func (d bridgeDBConfig) toDBConfig() config.DBConfig {
	return config.DBConfig{
		User:            d.User,
		Password:        d.Password,
		Host:            d.Host,
		Port:            d.Port,
		Path:            d.Path,
		Params:          d.Params,
		FirebirdEnabled: d.FirebirdEnabled,
		PhoenixEnabled:  d.PhoenixEnabled,
		PhoenixUser:     d.PhoenixUser,
		PhoenixPassword: d.PhoenixPassword,
		PhoenixHost:     d.PhoenixHost,
		PhoenixPort:     d.PhoenixPort,
		PhoenixInstance: d.PhoenixInstance,
		PhoenixDatabase: d.PhoenixDatabase,
		PhoenixParams:   d.PhoenixParams,
	}
}

type configDuration time.Duration

func (d configDuration) Duration() time.Duration {
	if d <= 0 {
		return 10 * time.Second
	}
	return time.Duration(d)
}

func (d configDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *configDuration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		v, err := time.ParseDuration(strings.TrimSpace(s))
		if err != nil {
			return fmt.Errorf("parse duration %q: %w", s, err)
		}
		*d = configDuration(v)
		return nil
	}
	var secs float64
	if err := json.Unmarshal(b, &secs); err != nil {
		return err
	}
	*d = configDuration(time.Duration(secs * float64(time.Second)))
	return nil
}

func defaultConfig() bridgeConfig {
	return bridgeConfig{
		BrokerHost:           "127.0.0.1",
		BrokerPubPort:        27001,
		BrokerSubPort:        27002,
		PultID:               0,
		PublishInitialEvents: false,
		DisableDeviceSync:    false,
		PhoenixProbeInterval: configDuration(2 * time.Second),
		PollInterval:         configDuration(10 * time.Second),
		HeartbeatInterval:    configDuration(60 * time.Second),
		DeviceTimeout:        configDuration(300 * time.Second),
		Database: bridgeDBConfig{
			FirebirdEnabled: true,
			User:            "SYSDBA",
			Password:        "masterkey",
			Host:            "localhost",
			Port:            "3050",
			Path:            "C:/MOST.PM/BASE/MOST5.FDB",
			Params:          "charset=WIN1251&auth_plugin_name=Srp",
			PhoenixEnabled:  false,
			PhoenixUser:     "sa",
			PhoenixPassword: "",
			PhoenixHost:     "localhost",
			PhoenixPort:     "",
			PhoenixInstance: "PHOENIX4",
			PhoenixDatabase: "Pult4DB",
			PhoenixParams:   "encrypt=disable&trustservercertificate=true",
		},
	}
}

func loadOrCreateConfig(path string) (bridgeConfig, bool, error) {
	cfg := defaultConfig()
	if path == "" {
		return cfg, false, nil
	}

	body, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(body, &cfg); err != nil {
			return bridgeConfig{}, false, fmt.Errorf("read config %q: %w", path, err)
		}
		applyConfigDefaults(&cfg)
		return cfg, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return bridgeConfig{}, false, fmt.Errorf("read config %q: %w", path, err)
	}

	if dir := filepath.Dir(path); dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	body, _ = json.MarshalIndent(cfg, "", "  ")
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return bridgeConfig{}, false, fmt.Errorf("write config %q: %w", path, err)
	}
	return cfg, true, nil
}

func applyConfigDefaults(cfg *bridgeConfig) {
	d := defaultConfig()
	if strings.TrimSpace(cfg.BrokerHost) == "" {
		cfg.BrokerHost = d.BrokerHost
	}
	if cfg.BrokerPubPort == 0 {
		cfg.BrokerPubPort = d.BrokerPubPort
	}
	if cfg.BrokerSubPort == 0 {
		cfg.BrokerSubPort = d.BrokerSubPort
	}
	if cfg.PultID < 0 {
		cfg.PultID = d.PultID
	}
	if cfg.PhoenixProbeInterval <= 0 {
		cfg.PhoenixProbeInterval = d.PhoenixProbeInterval
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = d.PollInterval
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = d.HeartbeatInterval
	}
	if cfg.DeviceTimeout <= 0 {
		cfg.DeviceTimeout = d.DeviceTimeout
	}
	if strings.TrimSpace(cfg.Database.User) == "" {
		cfg.Database.User = d.Database.User
	}
	if strings.TrimSpace(cfg.Database.Host) == "" {
		cfg.Database.Host = d.Database.Host
	}
	if strings.TrimSpace(cfg.Database.Port) == "" {
		cfg.Database.Port = d.Database.Port
	}
}

func randRead(b []byte) (int, error) {
	return rand.Read(b)
}
