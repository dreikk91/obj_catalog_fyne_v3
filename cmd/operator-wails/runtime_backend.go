package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/ids"

	fyneapp "fyne.io/fyne/v2/app"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

const operatorFyneAppID = "com.most.obj_catalog_fyne_v3"

type managedDBResource struct {
	label        string
	db           *sqlx.DB
	healthCancel context.CancelFunc
}

func bootstrapFrontendBackend() (contracts.FrontendBackend, func(), error) {
	cfg := loadRuntimeDBConfig()
	provider, resources, err := buildDataProviderFromEnvConfig(cfg)
	if err != nil {
		return nil, func() {}, err
	}

	frontend := backend.NewFrontendAdapter(provider)
	cleanup := func() {
		closeManagedDBResources(resources)
		if shutdowner, ok := provider.(contracts.ShutdownProvider); ok {
			shutdowner.Shutdown()
		}
	}

	return frontend, cleanup, nil
}

func loadRuntimeDBConfig() config.DBConfig {
	prefCfg, prefLoaded := loadPreferencesDBConfig()
	if !prefLoaded {
		return loadEnvDBConfig()
	}

	applyEnvOverrides(&prefCfg)
	log.Info().
		Bool("firebirdEnabled", prefCfg.FirebirdEnabled).
		Bool("phoenixEnabled", prefCfg.PhoenixEnabled).
		Bool("caslEnabled", prefCfg.CASLEnabled).
		Str("mode", prefCfg.NormalizedMode()).
		Msg("Operator Wails config loaded from Fyne preferences + env overrides")
	return prefCfg
}

func loadPreferencesDBConfig() (cfg config.DBConfig, ok bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Warn().Interface("panic", recovered).Msg("Operator Wails: failed to read Fyne preferences config")
			cfg = config.DBConfig{}
			ok = false
		}
	}()

	fyneInstance := fyneapp.NewWithID(operatorFyneAppID)
	if fyneInstance == nil {
		return config.DBConfig{}, false
	}
	defer fyneInstance.Quit()

	return config.LoadDBConfig(fyneInstance.Preferences()), true
}

func savePreferencesDBConfig(cfg config.DBConfig) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("operator wails: failed to save Fyne preferences config: %v", recovered)
		}
	}()

	fyneInstance := fyneapp.NewWithID(operatorFyneAppID)
	if fyneInstance == nil {
		return errors.New("operator wails: failed to initialize Fyne app for saving preferences")
	}
	defer fyneInstance.Quit()

	config.SaveDBConfig(fyneInstance.Preferences(), cfg)
	return nil
}

func applyEnvOverrides(cfg *config.DBConfig) {
	if cfg == nil {
		return
	}

	if value, ok := lookupEnvTrimmed("MOST_DB_USER"); ok {
		cfg.User = value
	}
	if value, ok := lookupEnvTrimmed("MOST_DB_PASSWORD"); ok {
		cfg.Password = value
	}
	if value, ok := lookupEnvTrimmed("MOST_DB_HOST"); ok {
		cfg.Host = value
	}
	if value, ok := lookupEnvTrimmed("MOST_DB_PORT"); ok {
		cfg.Port = value
	}
	if value, ok := lookupEnvTrimmed("MOST_DB_PATH"); ok {
		cfg.Path = value
	}
	if value, ok := lookupEnvTrimmed("MOST_DB_PARAMS"); ok {
		cfg.Params = value
	}
	if value, ok := envBoolOverride("MOST_FIREBIRD_ENABLED", cfg.FirebirdEnabled); ok {
		cfg.FirebirdEnabled = value
	}
	if value, ok := envBoolOverride("MOST_PHOENIX_ENABLED", cfg.PhoenixEnabled); ok {
		cfg.PhoenixEnabled = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_USER"); ok {
		cfg.PhoenixUser = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_PASSWORD"); ok {
		cfg.PhoenixPassword = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_HOST"); ok {
		cfg.PhoenixHost = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_PORT"); ok {
		cfg.PhoenixPort = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_INSTANCE"); ok {
		cfg.PhoenixInstance = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_DATABASE"); ok {
		cfg.PhoenixDatabase = value
	}
	if value, ok := lookupEnvTrimmed("MOST_PHOENIX_PARAMS"); ok {
		cfg.PhoenixParams = value
	}
	if value, ok := envBoolOverride("MOST_CASL_ENABLED", cfg.CASLEnabled); ok {
		cfg.CASLEnabled = value
	}
	if value, ok := lookupEnvTrimmed("MOST_BACKEND_MODE"); ok {
		cfg.Mode = value
	}
	if value, ok := lookupEnvTrimmed("MOST_CASL_BASE_URL"); ok {
		cfg.CASLBaseURL = value
	}
	if value, ok := lookupEnvTrimmed("MOST_CASL_TOKEN"); ok {
		cfg.CASLToken = value
	}
	if value, ok := lookupEnvTrimmed("MOST_CASL_EMAIL"); ok {
		cfg.CASLEmail = value
	}
	if value, ok := lookupEnvTrimmed("MOST_CASL_PASSWORD"); ok {
		cfg.CASLPass = value
	}
	if value, ok := envInt64Override("MOST_CASL_PULT_ID", cfg.CASLPultID); ok {
		cfg.CASLPultID = value
	}
	if value, ok := lookupEnvTrimmed("MOST_LOG_LEVEL"); ok {
		cfg.LogLevel = value
	}
}

func envBoolOverride(name string, fallback bool) (bool, bool) {
	raw, ok := lookupEnvTrimmed(name)
	if !ok {
		return fallback, false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().
			Err(err).
			Str("name", name).
			Str("value", raw).
			Bool("fallback", fallback).
			Msg("Invalid boolean environment value, keeping config value")
		return fallback, false
	}
	return parsed, true
}

func envInt64Override(name string, fallback int64) (int64, bool) {
	raw, ok := lookupEnvTrimmed(name)
	if !ok {
		return fallback, false
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Warn().
			Err(err).
			Str("name", name).
			Str("value", raw).
			Int64("fallback", fallback).
			Msg("Invalid int64 environment value, keeping config value")
		return fallback, false
	}
	return parsed, true
}

func buildDataProviderFromEnvConfig(cfg config.DBConfig) (contracts.DataProvider, []managedDBResource, error) {
	firebirdEnabled, phoenixEnabled, caslEnabled := resolveEnabledSources(cfg)

	managed := make([]managedDBResource, 0, 2)
	sources := make([]data.ProviderSource, 0, 3)
	initErrors := make([]string, 0, 3)

	if firebirdEnabled {
		dsn := cfg.FirebirdDSN()
		db, err := database.InitNamedDB("firebirdsql", dsn, "БД/МІСТ")
		if err != nil {
			initErrors = append(initErrors, fmt.Sprintf("firebird init failed: %v", err))
		} else {
			managed = append(managed, managedDBResource{
				label:        "БД/МІСТ",
				db:           db,
				healthCancel: database.StartNamedHealthCheck(db, "БД/МІСТ"),
			})
			sources = append(sources, data.ProviderSource{
				Name:     "bridge",
				Provider: backend.NewDBProvider(db, dsn),
			})
		}
	}

	if phoenixEnabled {
		dsn := cfg.PhoenixDSN()
		db, err := database.InitNamedDB("sqlserver", dsn, "Phoenix")
		if err != nil {
			initErrors = append(initErrors, fmt.Sprintf("phoenix init failed: %v", err))
		} else {
			managed = append(managed, managedDBResource{
				label:        "Phoenix",
				db:           db,
				healthCancel: database.StartNamedHealthCheck(db, "Phoenix"),
			})
			sources = append(sources, data.ProviderSource{
				Name:         "phoenix",
				Provider:     backend.NewPhoenixProvider(db, dsn),
				OwnsObjectID: ids.IsPhoenixObjectID,
				OwnsAlarmID:  ids.IsPhoenixObjectID,
			})
		}
	}

	if caslEnabled {
		casl := backend.NewCASLCloudProvider(
			cfg.CASLBaseURL,
			cfg.CASLToken,
			cfg.CASLPultID,
			cfg.CASLEmail,
			cfg.CASLPass,
		)
		sources = append(sources, data.ProviderSource{
			Name:         "casl",
			Provider:     casl,
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
		})
	}

	if len(sources) == 0 {
		closeManagedDBResources(managed)
		return nil, nil, fmt.Errorf("failed to initialize any data source: %s", strings.Join(initErrors, "; "))
	}

	if len(initErrors) > 0 {
		log.Warn().Strs("errors", initErrors).Msg("Operator Wails: some backend sources failed to initialize")
	}

	return backend.NewMultiSourceProvider(sources...), managed, nil
}

func resolveEnabledSources(cfg config.DBConfig) (firebird bool, phoenix bool, casl bool) {
	firebird = cfg.FirebirdEnabled
	phoenix = cfg.PhoenixEnabled
	casl = cfg.CASLEnabled

	switch cfg.NormalizedMode() {
	case config.BackendModePhoenix:
		phoenix = true
	case config.BackendModeCASLCloud:
		casl = true
	}

	if !firebird && !phoenix && !casl {
		firebird = true
	}

	return firebird, phoenix, casl
}

func closeManagedDBResources(resources []managedDBResource) {
	for _, resource := range resources {
		if resource.healthCancel != nil {
			resource.healthCancel()
		}
		if resource.db != nil {
			if err := resource.db.Close(); err != nil {
				log.Warn().Err(err).Str("label", resource.label).Msg("failed to close data source")
			}
		}
	}
}

func loadEnvDBConfig() config.DBConfig {
	cfg := config.DBConfig{
		User:            envString("MOST_DB_USER", "SYSDBA"),
		Password:        envString("MOST_DB_PASSWORD", "masterkey"),
		Host:            envString("MOST_DB_HOST", "localhost"),
		Port:            envString("MOST_DB_PORT", "3050"),
		Path:            envString("MOST_DB_PATH", "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:          envString("MOST_DB_PARAMS", "charset=WIN1251&auth_plugin_name=Srp"),
		FirebirdEnabled: envBool("MOST_FIREBIRD_ENABLED", true),
		PhoenixEnabled:  envBool("MOST_PHOENIX_ENABLED", false),
		PhoenixUser:     envString("MOST_PHOENIX_USER", "sa"),
		PhoenixPassword: envString("MOST_PHOENIX_PASSWORD", ""),
		PhoenixHost:     envString("MOST_PHOENIX_HOST", "localhost"),
		PhoenixPort:     envString("MOST_PHOENIX_PORT", ""),
		PhoenixInstance: envString("MOST_PHOENIX_INSTANCE", "PHOENIX4"),
		PhoenixDatabase: envString("MOST_PHOENIX_DATABASE", "Pult4DB"),
		PhoenixParams:   envString("MOST_PHOENIX_PARAMS", "encrypt=disable&trustservercertificate=true"),
		CASLEnabled:     envBool("MOST_CASL_ENABLED", false),
		Mode:            envString("MOST_BACKEND_MODE", config.BackendModeFirebird),
		CASLBaseURL:     envString("MOST_CASL_BASE_URL", "http://127.0.0.1:50003"),
		CASLToken:       envString("MOST_CASL_TOKEN", ""),
		CASLEmail:       envString("MOST_CASL_EMAIL", ""),
		CASLPass:        envString("MOST_CASL_PASSWORD", ""),
		CASLPultID:      envInt64("MOST_CASL_PULT_ID", 0),
		LogLevel:        envString("MOST_LOG_LEVEL", "info"),
	}

	log.Info().
		Bool("firebirdEnabled", cfg.FirebirdEnabled).
		Bool("phoenixEnabled", cfg.PhoenixEnabled).
		Bool("caslEnabled", cfg.CASLEnabled).
		Str("mode", cfg.NormalizedMode()).
		Msg("Operator Wails config loaded from environment")

	return cfg
}

func envString(name string, fallback string) string {
	raw, ok := lookupEnvTrimmed(name)
	if !ok {
		return fallback
	}
	return raw
}

func envBool(name string, fallback bool) bool {
	raw, ok := lookupEnvTrimmed(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		log.Warn().
			Err(err).
			Str("name", name).
			Str("value", raw).
			Bool("fallback", fallback).
			Msg("Invalid boolean environment value, using fallback")
		return fallback
	}
	return parsed
}

func envInt64(name string, fallback int64) int64 {
	raw, ok := lookupEnvTrimmed(name)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Warn().
			Err(err).
			Str("name", name).
			Str("value", raw).
			Int64("fallback", fallback).
			Msg("Invalid int64 environment value, using fallback")
		return fallback
	}
	return parsed
}

func lookupEnvTrimmed(name string) (string, bool) {
	value, ok := os.LookupEnv(name)
	if !ok {
		return "", false
	}
	return strings.TrimSpace(value), true
}

type emptyFrontendBackend struct{}

func (emptyFrontendBackend) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	return contracts.FrontendCapabilities{
		Sources: []contracts.FrontendSourceCapability{
			{
				Source:            contracts.FrontendSourceBridge,
				DisplayName:       "МІСТ/Firebird",
				ReadObjects:       true,
				ReadObjectDetails: true,
				ReadEvents:        true,
				ReadAlarms:        true,
			},
		},
	}, nil
}

func (emptyFrontendBackend) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	return []contracts.FrontendObjectSummary{}, nil
}

func (emptyFrontendBackend) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	return []contracts.FrontendAlarmItem{}, nil
}

func (emptyFrontendBackend) GetAlarmProcessingOptions(context.Context, int) ([]contracts.FrontendAlarmProcessingOption, error) {
	return []contracts.FrontendAlarmProcessingOption{}, fmt.Errorf("alarm processing is unavailable in shell-only mode")
}

func (emptyFrontendBackend) PickAlarm(context.Context, int, contracts.FrontendAlarmPickRequest) error {
	return fmt.Errorf("alarm pick is unavailable in shell-only mode")
}

func (emptyFrontendBackend) ProcessAlarm(context.Context, int, contracts.FrontendAlarmProcessRequest) error {
	return fmt.Errorf("alarm processing is unavailable in shell-only mode")
}

func (emptyFrontendBackend) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	return []contracts.FrontendEventItem{}, nil
}

func (emptyFrontendBackend) ListObjectEvents(context.Context, int, int, int) (contracts.FrontendEventPage, error) {
	return contracts.FrontendEventPage{}, errors.New("object events are unavailable in shell-only mode")
}

func (emptyFrontendBackend) GetObjectDetails(context.Context, int) (contracts.FrontendObjectDetails, error) {
	return contracts.FrontendObjectDetails{}, errors.New("object details are unavailable in shell-only mode")
}

func (emptyFrontendBackend) CreateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, fmt.Errorf("create object is unavailable in shell-only mode")
}

func (emptyFrontendBackend) UpdateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, fmt.Errorf("update object is unavailable in shell-only mode")
}

func (emptyFrontendBackend) GroupProcessAlarm(context.Context, int, string) error {
	return fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) ListAlarmProcessingOptionsCached(context.Context) ([]contracts.FrontendAlarmProcessingOption, error) {
	return nil, fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) ListResponseGroups(context.Context) ([]contracts.FrontendResponseGroup, error) {
	return nil, fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) AssignResponseGroup(context.Context, int, contracts.FrontendAlarmGroupActionRequest) error {
	return fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) NotifyGroupArrived(context.Context, int) error {
	return fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) CancelResponseGroup(context.Context, int) error {
	return fmt.Errorf("unavailable in shell-only mode")
}

func (emptyFrontendBackend) StandbyObject(context.Context, int, contracts.FrontendStandbyRequest) error {
	return fmt.Errorf("unavailable in shell-only mode")
}
