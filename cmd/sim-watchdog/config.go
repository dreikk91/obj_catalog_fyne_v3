package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
)

type serviceConfig struct {
	PollInterval     configDuration        `json:"poll_interval"`
	HistoryPath      string                `json:"history_path"`
	DryRun           bool                  `json:"dry_run"`
	IncludeNonBridge bool                  `json:"include_non_bridge"`
	MaxLastTestAge   configDuration        `json:"max_last_test_age"`
	VerifyDB         bool                  `json:"verify_db"`
	Database         serviceDatabaseConfig `json:"database"`
	Kyivstar         serviceKyivstarConfig `json:"kyivstar"`
	Vodafone         serviceVodafoneConfig `json:"vodafone"`
}

type serviceDatabaseConfig struct {
	User            string `json:"user"`
	Password        string `json:"password"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	Path            string `json:"path"`
	Params          string `json:"params"`
	FirebirdEnabled bool   `json:"firebird_enabled"`
	PhoenixEnabled  bool   `json:"phoenix_enabled"`
	PhoenixUser     string `json:"phoenix_user"`
	PhoenixPassword string `json:"phoenix_password"`
	PhoenixHost     string `json:"phoenix_host"`
	PhoenixPort     string `json:"phoenix_port"`
	PhoenixInstance string `json:"phoenix_instance"`
	PhoenixDatabase string `json:"phoenix_database"`
	PhoenixParams   string `json:"phoenix_params"`
	CASLEnabled     bool   `json:"casl_enabled"`
	Mode            string `json:"mode"`
	CASLBaseURL     string `json:"casl_base_url"`
	CASLToken       string `json:"casl_token"`
	CASLEmail       string `json:"casl_email"`
	CASLPass        string `json:"casl_password"`
	CASLPultID      int64  `json:"casl_pult_id"`
	LogLevel        string `json:"log_level"`
}

type serviceKyivstarConfig struct {
	ClientID             string `json:"client_id"`
	ClientSecret         string `json:"client_secret"`
	UserEmail            string `json:"user_email"`
	AccessToken          string `json:"access_token"`
	TokenExpiry          string `json:"token_expiry"`
	AutoResetEnabled     bool   `json:"auto_reset_enabled"`
	AutoResetDailyLimit  int    `json:"auto_reset_daily_limit"`
	AutoResetWindowHours int    `json:"auto_reset_window_hours"`
}

type serviceVodafoneConfig struct {
	Phone                string `json:"phone"`
	AccessToken          string `json:"access_token"`
	TokenExpiry          string `json:"token_expiry"`
	LoginMethod          string `json:"login_method"`
	PUK                  string `json:"puk"`
	AutoResetEnabled     bool   `json:"auto_reset_enabled"`
	AutoResetDailyLimit  int    `json:"auto_reset_daily_limit"`
	AutoResetWindowHours int    `json:"auto_reset_window_hours"`
}

type configDuration time.Duration

func (d configDuration) Duration() time.Duration {
	if d <= 0 {
		return 3 * time.Minute
	}
	return time.Duration(d)
}

func (d configDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *configDuration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed, parseErr := time.ParseDuration(strings.TrimSpace(text))
		if parseErr != nil {
			return fmt.Errorf("parse duration %q: %w", text, parseErr)
		}
		*d = configDuration(parsed)
		return nil
	}

	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}
	*d = configDuration(time.Duration(seconds * float64(time.Second)))
	return nil
}

func defaultServiceConfig() serviceConfig {
	return serviceConfig{
		PollInterval:     configDuration(3 * time.Minute),
		HistoryPath:      "log/sim-watchdog-history.json",
		DryRun:           false,
		IncludeNonBridge: false,
		MaxLastTestAge:   configDuration(7 * 24 * time.Hour),
		VerifyDB:         true,
		Database: serviceDatabaseConfig{
			User:            "SYSDBA",
			Password:        "masterkey",
			Host:            "localhost",
			Port:            "3050",
			Path:            "C:/MOST.PM/BASE/MOST5.FDB",
			Params:          "charset=WIN1251&auth_plugin_name=Srp",
			FirebirdEnabled: true,
			PhoenixEnabled:  false,
			PhoenixUser:     "sa",
			PhoenixHost:     "localhost",
			PhoenixInstance: "PHOENIX4",
			PhoenixDatabase: "Pult4DB",
			PhoenixParams:   "encrypt=disable&trustservercertificate=true",
			Mode:            config.BackendModeFirebird,
			CASLBaseURL:     "http://127.0.0.1:50003",
			LogLevel:        "info",
		},
		Kyivstar: serviceKyivstarConfig{
			AutoResetEnabled:     config.DefaultKyivstarAutoResetEnabled,
			AutoResetDailyLimit:  config.DefaultKyivstarAutoResetDailyLimit,
			AutoResetWindowHours: config.DefaultKyivstarAutoResetWindowHours,
		},
		Vodafone: serviceVodafoneConfig{
			LoginMethod:          config.VodafoneLoginMethodSMS,
			AutoResetEnabled:     config.DefaultVodafoneAutoResetEnabled,
			AutoResetDailyLimit:  config.DefaultVodafoneAutoResetDailyLimit,
			AutoResetWindowHours: config.DefaultVodafoneAutoResetWindowHours,
		},
	}
}

func loadServiceConfig(path string) (serviceConfig, error) {
	cfg := defaultServiceConfig()
	body, err := os.ReadFile(path)
	if err != nil {
		return serviceConfig{}, fmt.Errorf("read service config %q: %w", path, err)
	}
	if err := json.Unmarshal(body, &cfg); err != nil {
		return serviceConfig{}, fmt.Errorf("decode service config %q: %w", path, err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func writeServiceConfig(path string, cfg serviceConfig) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("service config path is empty")
	}
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config directory %q: %w", dir, err)
		}
	}
	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode service config: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write service config %q: %w", path, err)
	}
	return nil
}

func (cfg *serviceConfig) applyDefaults() {
	defaults := defaultServiceConfig()
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaults.PollInterval
	}
	if strings.TrimSpace(cfg.HistoryPath) == "" {
		cfg.HistoryPath = defaults.HistoryPath
	}
	if cfg.MaxLastTestAge <= 0 {
		cfg.MaxLastTestAge = defaults.MaxLastTestAge
	}
	cfg.Database.applyDefaults()
	cfg.Kyivstar.applyDefaults()
	cfg.Vodafone.applyDefaults()
}

func (cfg *serviceDatabaseConfig) applyDefaults() {
	defaults := defaultServiceConfig().Database
	if strings.TrimSpace(cfg.User) == "" {
		cfg.User = defaults.User
	}
	if strings.TrimSpace(cfg.Host) == "" {
		cfg.Host = defaults.Host
	}
	if strings.TrimSpace(cfg.Port) == "" {
		cfg.Port = defaults.Port
	}
	if strings.TrimSpace(cfg.Path) == "" {
		cfg.Path = defaults.Path
	}
	if strings.TrimSpace(cfg.Params) == "" {
		cfg.Params = defaults.Params
	}
	if strings.TrimSpace(cfg.PhoenixUser) == "" {
		cfg.PhoenixUser = defaults.PhoenixUser
	}
	if strings.TrimSpace(cfg.PhoenixHost) == "" {
		cfg.PhoenixHost = defaults.PhoenixHost
	}
	if strings.TrimSpace(cfg.PhoenixInstance) == "" {
		cfg.PhoenixInstance = defaults.PhoenixInstance
	}
	if strings.TrimSpace(cfg.PhoenixDatabase) == "" {
		cfg.PhoenixDatabase = defaults.PhoenixDatabase
	}
	if strings.TrimSpace(cfg.PhoenixParams) == "" {
		cfg.PhoenixParams = defaults.PhoenixParams
	}
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = defaults.Mode
	}
	if strings.TrimSpace(cfg.CASLBaseURL) == "" {
		cfg.CASLBaseURL = defaults.CASLBaseURL
	}
	if strings.TrimSpace(cfg.LogLevel) == "" {
		cfg.LogLevel = defaults.LogLevel
	}
}

func (cfg *serviceKyivstarConfig) applyDefaults() {
	if cfg.AutoResetDailyLimit == 0 {
		cfg.AutoResetDailyLimit = config.DefaultKyivstarAutoResetDailyLimit
	}
	if cfg.AutoResetWindowHours == 0 {
		cfg.AutoResetWindowHours = config.DefaultKyivstarAutoResetWindowHours
	}
}

func (cfg *serviceVodafoneConfig) applyDefaults() {
	if strings.TrimSpace(cfg.LoginMethod) == "" {
		cfg.LoginMethod = config.VodafoneLoginMethodSMS
	}
	if cfg.AutoResetDailyLimit == 0 {
		cfg.AutoResetDailyLimit = config.DefaultVodafoneAutoResetDailyLimit
	}
	if cfg.AutoResetWindowHours == 0 {
		cfg.AutoResetWindowHours = config.DefaultVodafoneAutoResetWindowHours
	}
}

func (cfg serviceConfig) dbConfig() config.DBConfig {
	return cfg.Database.toDBConfig()
}

func (cfg serviceDatabaseConfig) toDBConfig() config.DBConfig {
	return config.DBConfig{
		User:            cfg.User,
		Password:        cfg.Password,
		Host:            cfg.Host,
		Port:            cfg.Port,
		Path:            cfg.Path,
		Params:          cfg.Params,
		FirebirdEnabled: cfg.FirebirdEnabled,
		PhoenixEnabled:  cfg.PhoenixEnabled,
		PhoenixUser:     cfg.PhoenixUser,
		PhoenixPassword: cfg.PhoenixPassword,
		PhoenixHost:     cfg.PhoenixHost,
		PhoenixPort:     cfg.PhoenixPort,
		PhoenixInstance: cfg.PhoenixInstance,
		PhoenixDatabase: cfg.PhoenixDatabase,
		PhoenixParams:   cfg.PhoenixParams,
		CASLEnabled:     cfg.CASLEnabled,
		Mode:            cfg.Mode,
		CASLBaseURL:     cfg.CASLBaseURL,
		CASLToken:       cfg.CASLToken,
		CASLEmail:       cfg.CASLEmail,
		CASLPass:        cfg.CASLPass,
		CASLPultID:      cfg.CASLPultID,
		LogLevel:        cfg.LogLevel,
	}
}

func serviceDatabaseFromDBConfig(cfg config.DBConfig) serviceDatabaseConfig {
	return serviceDatabaseConfig{
		User:            cfg.User,
		Password:        cfg.Password,
		Host:            cfg.Host,
		Port:            cfg.Port,
		Path:            cfg.Path,
		Params:          cfg.Params,
		FirebirdEnabled: cfg.FirebirdEnabled,
		PhoenixEnabled:  cfg.PhoenixEnabled,
		PhoenixUser:     cfg.PhoenixUser,
		PhoenixPassword: cfg.PhoenixPassword,
		PhoenixHost:     cfg.PhoenixHost,
		PhoenixPort:     cfg.PhoenixPort,
		PhoenixInstance: cfg.PhoenixInstance,
		PhoenixDatabase: cfg.PhoenixDatabase,
		PhoenixParams:   cfg.PhoenixParams,
		CASLEnabled:     cfg.CASLEnabled,
		Mode:            cfg.Mode,
		CASLBaseURL:     cfg.CASLBaseURL,
		CASLToken:       cfg.CASLToken,
		CASLEmail:       cfg.CASLEmail,
		CASLPass:        cfg.CASLPass,
		CASLPultID:      cfg.CASLPultID,
		LogLevel:        cfg.LogLevel,
	}
}

func serviceKyivstarFromConfig(cfg config.KyivstarConfig) serviceKyivstarConfig {
	return serviceKyivstarConfig{
		ClientID:             cfg.ClientID,
		ClientSecret:         cfg.ClientSecret,
		UserEmail:            cfg.UserEmail,
		AccessToken:          cfg.AccessToken,
		TokenExpiry:          cfg.TokenExpiry,
		AutoResetEnabled:     cfg.AutoResetEnabled,
		AutoResetDailyLimit:  cfg.AutoResetDailyLimit,
		AutoResetWindowHours: cfg.AutoResetWindowHours,
	}
}

func (cfg serviceKyivstarConfig) toConfig() config.KyivstarConfig {
	return config.KyivstarConfig{
		ClientID:             cfg.ClientID,
		ClientSecret:         cfg.ClientSecret,
		UserEmail:            cfg.UserEmail,
		AccessToken:          cfg.AccessToken,
		TokenExpiry:          cfg.TokenExpiry,
		AutoResetEnabled:     cfg.AutoResetEnabled,
		AutoResetDailyLimit:  cfg.AutoResetDailyLimit,
		AutoResetWindowHours: cfg.AutoResetWindowHours,
	}
}

func serviceKyivstarFromRuntimeConfig(cfg config.KyivstarConfig) serviceKyivstarConfig {
	return serviceKyivstarFromConfig(cfg)
}

func serviceVodafoneFromConfig(cfg config.VodafoneConfig) serviceVodafoneConfig {
	return serviceVodafoneConfig{
		Phone:                cfg.Phone,
		AccessToken:          cfg.AccessToken,
		TokenExpiry:          cfg.TokenExpiry,
		LoginMethod:          cfg.LoginMethod,
		PUK:                  cfg.PUK,
		AutoResetEnabled:     cfg.AutoResetEnabled,
		AutoResetDailyLimit:  cfg.AutoResetDailyLimit,
		AutoResetWindowHours: cfg.AutoResetWindowHours,
	}
}

func (cfg serviceVodafoneConfig) toConfig() config.VodafoneConfig {
	return config.VodafoneConfig{
		Phone:                cfg.Phone,
		AccessToken:          cfg.AccessToken,
		TokenExpiry:          cfg.TokenExpiry,
		LoginMethod:          cfg.LoginMethod,
		PUK:                  cfg.PUK,
		AutoResetEnabled:     cfg.AutoResetEnabled,
		AutoResetDailyLimit:  cfg.AutoResetDailyLimit,
		AutoResetWindowHours: cfg.AutoResetWindowHours,
	}
}

type fileConfigStore struct {
	path string
	mu   sync.Mutex
	cfg  serviceConfig
}

func newFileConfigStore(path string, cfg serviceConfig) *fileConfigStore {
	return &fileConfigStore{path: path, cfg: cfg}
}

func (s *fileConfigStore) LoadKyivstarConfig() config.KyivstarConfig {
	if s == nil {
		return config.KyivstarConfig{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Kyivstar.toConfig()
}

func (s *fileConfigStore) SaveKyivstarConfig(cfg config.KyivstarConfig) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cfg.Kyivstar = serviceKyivstarFromRuntimeConfig(cfg)
	snapshot := s.cfg
	path := s.path
	s.mu.Unlock()
	if err := writeServiceConfig(path, snapshot); err != nil {
		// Token persistence errors are logged by callers as API errors only if surfaced.
		return
	}
}

func (s *fileConfigStore) LoadVodafoneConfig() config.VodafoneConfig {
	if s == nil {
		return config.VodafoneConfig{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Vodafone.toConfig()
}

func (s *fileConfigStore) SaveVodafoneConfig(cfg config.VodafoneConfig) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.cfg.Vodafone = serviceVodafoneFromConfig(cfg)
	snapshot := s.cfg
	path := s.path
	s.mu.Unlock()
	if err := writeServiceConfig(path, snapshot); err != nil {
		return
	}
}
