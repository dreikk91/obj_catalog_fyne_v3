package main

import (
	"fmt"
	"strings"
	"sync"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/wailsbridge"
)

type operatorRuntimeController struct {
	mu sync.Mutex

	bridge         *wailsbridge.FrontendV1Service
	backendCleanup func()
}

func newOperatorRuntimeController(bridge *wailsbridge.FrontendV1Service) *operatorRuntimeController {
	return &operatorRuntimeController{
		bridge:         bridge,
		backendCleanup: func() {},
	}
}

func (c *operatorRuntimeController) replaceBackend(frontendBackend contracts.FrontendBackend, cleanup func()) {
	if c == nil {
		return
	}
	if frontendBackend == nil {
		frontendBackend = emptyFrontendBackend{}
	}
	if cleanup == nil {
		cleanup = func() {}
	}

	c.mu.Lock()
	oldCleanup := c.backendCleanup
	c.backendCleanup = cleanup
	if c.bridge != nil {
		c.bridge.SetBackend(frontendBackend)
	}
	c.mu.Unlock()

	oldCleanup()
}

func (c *operatorRuntimeController) reloadWithConfig(cfg config.DBConfig) error {
	provider, resources, err := buildDataProviderFromEnvConfig(cfg)
	if err != nil {
		return err
	}

	frontendBackend := backend.NewFrontendAdapter(provider)
	cleanup := func() {
		closeManagedDBResources(resources)
		if shutdowner, ok := provider.(contracts.ShutdownProvider); ok {
			shutdowner.Shutdown()
		}
	}
	c.replaceBackend(frontendBackend, cleanup)
	return nil
}

func (c *operatorRuntimeController) shutdown() {
	if c == nil {
		return
	}

	c.mu.Lock()
	cleanup := c.backendCleanup
	c.backendCleanup = func() {}
	c.mu.Unlock()

	cleanup()
}

type OperatorDBSettings struct {
	FirebirdEnabled  bool   `json:"FirebirdEnabled"`
	FirebirdUser     string `json:"FirebirdUser"`
	FirebirdPassword string `json:"FirebirdPassword"`
	FirebirdHost     string `json:"FirebirdHost"`
	FirebirdPort     string `json:"FirebirdPort"`
	FirebirdPath     string `json:"FirebirdPath"`
	FirebirdParams   string `json:"FirebirdParams"`

	PhoenixEnabled  bool   `json:"PhoenixEnabled"`
	PhoenixUser     string `json:"PhoenixUser"`
	PhoenixPassword string `json:"PhoenixPassword"`
	PhoenixHost     string `json:"PhoenixHost"`
	PhoenixPort     string `json:"PhoenixPort"`
	PhoenixInstance string `json:"PhoenixInstance"`
	PhoenixDatabase string `json:"PhoenixDatabase"`
	PhoenixParams   string `json:"PhoenixParams"`

	CASLEnabled bool   `json:"CASLEnabled"`
	CASLBaseURL string `json:"CASLBaseURL"`
	CASLToken   string `json:"CASLToken"`
	CASLEmail   string `json:"CASLEmail"`
	CASLPass    string `json:"CASLPass"`
	CASLPultID  int64  `json:"CASLPultID"`

	Mode string `json:"Mode"`
}

type OperatorSettingsService struct {
	controller *operatorRuntimeController
}

func newOperatorSettingsService(controller *operatorRuntimeController) *OperatorSettingsService {
	return &OperatorSettingsService{controller: controller}
}

func (s *OperatorSettingsService) GetDBSettings() (OperatorDBSettings, error) {
	cfg := loadRuntimeDBConfig()
	return mapOperatorDBSettings(cfg), nil
}

func (s *OperatorSettingsService) SaveDBSettings(input OperatorDBSettings) error {
	cfg := mapConfigFromOperatorDBSettings(input)
	if !cfg.FirebirdEnabled && !cfg.PhoenixEnabled && !cfg.CASLEnabled {
		cfg.FirebirdEnabled = true
	}
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = deduceBackendMode(cfg)
	}

	if s == nil || s.controller == nil {
		return fmt.Errorf("operator settings service is unavailable")
	}

	if err := s.controller.reloadWithConfig(cfg); err != nil {
		return err
	}
	return savePreferencesDBConfig(cfg)
}

func mapOperatorDBSettings(cfg config.DBConfig) OperatorDBSettings {
	return OperatorDBSettings{
		FirebirdEnabled:  cfg.FirebirdEnabled,
		FirebirdUser:     cfg.User,
		FirebirdPassword: cfg.Password,
		FirebirdHost:     cfg.Host,
		FirebirdPort:     cfg.Port,
		FirebirdPath:     cfg.Path,
		FirebirdParams:   cfg.Params,

		PhoenixEnabled:  cfg.PhoenixEnabled,
		PhoenixUser:     cfg.PhoenixUser,
		PhoenixPassword: cfg.PhoenixPassword,
		PhoenixHost:     cfg.PhoenixHost,
		PhoenixPort:     cfg.PhoenixPort,
		PhoenixInstance: cfg.PhoenixInstance,
		PhoenixDatabase: cfg.PhoenixDatabase,
		PhoenixParams:   cfg.PhoenixParams,

		CASLEnabled: cfg.CASLEnabled,
		CASLBaseURL: cfg.CASLBaseURL,
		CASLToken:   cfg.CASLToken,
		CASLEmail:   cfg.CASLEmail,
		CASLPass:    cfg.CASLPass,
		CASLPultID:  cfg.CASLPultID,

		Mode: cfg.NormalizedMode(),
	}
}

func mapConfigFromOperatorDBSettings(input OperatorDBSettings) config.DBConfig {
	cfg := config.DBConfig{
		User:            strings.TrimSpace(input.FirebirdUser),
		Password:        input.FirebirdPassword,
		Host:            strings.TrimSpace(input.FirebirdHost),
		Port:            strings.TrimSpace(input.FirebirdPort),
		Path:            strings.TrimSpace(input.FirebirdPath),
		Params:          strings.TrimSpace(input.FirebirdParams),
		FirebirdEnabled: input.FirebirdEnabled,

		PhoenixEnabled:  input.PhoenixEnabled,
		PhoenixUser:     strings.TrimSpace(input.PhoenixUser),
		PhoenixPassword: input.PhoenixPassword,
		PhoenixHost:     strings.TrimSpace(input.PhoenixHost),
		PhoenixPort:     strings.TrimSpace(input.PhoenixPort),
		PhoenixInstance: strings.TrimSpace(input.PhoenixInstance),
		PhoenixDatabase: strings.TrimSpace(input.PhoenixDatabase),
		PhoenixParams:   strings.TrimSpace(input.PhoenixParams),

		CASLEnabled: input.CASLEnabled,
		CASLBaseURL: strings.TrimSpace(input.CASLBaseURL),
		CASLToken:   strings.TrimSpace(input.CASLToken),
		CASLEmail:   strings.TrimSpace(input.CASLEmail),
		CASLPass:    strings.TrimSpace(input.CASLPass),
		CASLPultID:  input.CASLPultID,

		Mode: strings.TrimSpace(input.Mode),
	}

	if cfg.User == "" {
		cfg.User = "SYSDBA"
	}
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == "" {
		cfg.Port = "3050"
	}
	if cfg.Path == "" {
		cfg.Path = "C:/MOST.PM/BASE/MOST5.FDB"
	}
	if cfg.Params == "" {
		cfg.Params = "charset=WIN1251&auth_plugin_name=Srp"
	}

	if cfg.PhoenixUser == "" {
		cfg.PhoenixUser = "sa"
	}
	if cfg.PhoenixHost == "" {
		cfg.PhoenixHost = "localhost"
	}
	if cfg.PhoenixInstance == "" {
		cfg.PhoenixInstance = "PHOENIX4"
	}
	if cfg.PhoenixDatabase == "" {
		cfg.PhoenixDatabase = "Pult4DB"
	}
	if cfg.PhoenixParams == "" {
		cfg.PhoenixParams = "encrypt=disable&trustservercertificate=true"
	}

	if cfg.CASLBaseURL == "" {
		cfg.CASLBaseURL = "http://127.0.0.1:50003"
	}
	return cfg
}

func deduceBackendMode(cfg config.DBConfig) string {
	if cfg.CASLEnabled && !cfg.FirebirdEnabled && !cfg.PhoenixEnabled {
		return config.BackendModeCASLCloud
	}
	if cfg.PhoenixEnabled && !cfg.FirebirdEnabled && !cfg.CASLEnabled {
		return config.BackendModePhoenix
	}
	return config.BackendModeFirebird
}
