package dataruntime

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/ids"
)

type managedDBResource struct {
	db           *sqlx.DB
	healthCancel context.CancelFunc
	source       contracts.FrontendSource
	health       *database.ConnectionHealth
}

// ConfigStore provides operator settings for Bridge SIM API actions.
type ConfigStore interface {
	config.KyivstarConfigStore
	config.VodafoneConfigStore
}

// Runtime owns a data provider and the resources opened for it.
type Runtime struct {
	Provider        contracts.DataProvider
	FirebirdEnabled bool
	PhoenixEnabled  bool
	CASLEnabled     bool

	managedDBs []managedDBResource
}

// SourceHealth reports the latest known connectivity state of an enabled source.
type SourceHealth struct {
	Source contracts.FrontendSource
	Status contracts.FrontendSourceHealthStatus
}

// New builds the configured backend without importing the GUI application layer.
func New(cfg config.DBConfig, store ConfigStore, verifyConnectivity bool) (*Runtime, error) {
	firebirdEnabled, phoenixEnabled, caslEnabled := enabledSources(cfg)

	runtime := &Runtime{
		managedDBs:      make([]managedDBResource, 0, 2),
		FirebirdEnabled: firebirdEnabled,
		PhoenixEnabled:  phoenixEnabled,
		CASLEnabled:     caslEnabled,
	}
	sources := make([]data.ProviderSource, 0, 3)

	if firebirdEnabled {
		dsn := cfg.FirebirdDSN()
		log.Info().
			Str("label", "БД/МІСТ").
			Str("host", cfg.Host).
			Str("port", cfg.Port).
			Str("path", cfg.Path).
			Msg("Підключення БД/МІСТ з поточного конфігу")
		db, err := database.InitNamedDB("firebirdsql", dsn, "БД/МІСТ")
		if err != nil {
			return nil, err
		}
		if verifyConnectivity {
			if err := db.Ping(); err != nil {
				_ = db.Close()
				return nil, fmt.Errorf("firebird ping failed: %w", err)
			}
		}
		healthCancel, health := database.StartNamedHealthCheckWithStatus(db, "БД/МІСТ")
		runtime.managedDBs = append(runtime.managedDBs, managedDBResource{
			db:           db,
			healthCancel: healthCancel,
			source:       contracts.FrontendSourceBridge,
			health:       health,
		})
		sources = append(sources, data.ProviderSource{
			Name: "bridge",
			Provider: data.NewDBDataProvider(
				db,
				dsn,
				data.WithVodafoneConfigStore(store),
				data.WithKyivstarConfigStore(store),
			),
		})
	}

	if phoenixEnabled {
		dsn := cfg.PhoenixDSN()
		log.Info().
			Str("label", "Phoenix").
			Str("host", cfg.PhoenixHost).
			Str("port", cfg.PhoenixPort).
			Str("instance", cfg.PhoenixInstance).
			Str("database", cfg.PhoenixDatabase).
			Msg("Підключення Phoenix з поточного конфігу")
		db, err := database.InitNamedDB("sqlserver", dsn, "Phoenix")
		if err != nil {
			runtime.Close()
			return nil, err
		}
		if verifyConnectivity {
			if err := db.Ping(); err != nil {
				_ = db.Close()
				runtime.Close()
				return nil, fmt.Errorf("phoenix ping failed: %w", err)
			}
		}
		phoenixProvider := data.NewPhoenixDataProvider(db, dsn)
		settingsCtx, settingsCancel := context.WithTimeout(context.Background(), 10*time.Second)
		settingsErr := phoenixProvider.ConfigureRuntime(settingsCtx, cfg)
		settingsCancel()
		if settingsErr != nil {
			_ = db.Close()
			runtime.Close()
			return nil, fmt.Errorf("phoenix runtime settings: %w", settingsErr)
		}
		if err := phoenixProvider.StartControlCenterSession(); err != nil {
			log.Warn().Err(err).Msg("Phoenix UDP недоступний; робота з БД продовжується")
		}
		healthCancel, health := database.StartNamedHealthCheckWithStatus(db, "Phoenix")
		runtime.managedDBs = append(runtime.managedDBs, managedDBResource{
			db:           db,
			healthCancel: healthCancel,
			source:       contracts.FrontendSourcePhoenix,
			health:       health,
		})
		sources = append(sources, data.ProviderSource{
			Name:         "phoenix",
			Provider:     phoenixProvider,
			OwnsObjectID: ids.IsPhoenixObjectID,
			OwnsAlarmID:  ids.IsPhoenixObjectID,
		})
	}

	if caslEnabled {
		sources = append(sources, data.ProviderSource{
			Name: "casl",
			Provider: data.NewCASLCloudProvider(
				cfg.CASLBaseURL,
				cfg.CASLToken,
				cfg.CASLPultID,
				cfg.CASLEmail,
				cfg.CASLPass,
			),
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
		})
	}

	runtime.Provider = data.NewMultiSourceDataProvider(sources...)
	return runtime, nil
}

// SourceHealth returns a snapshot of the latest health state for enabled sources.
func (r *Runtime) SourceHealth() []SourceHealth {
	if r == nil {
		return nil
	}

	healthBySource := make(map[contracts.FrontendSource]contracts.FrontendSourceHealthStatus, 3)
	for _, resource := range r.managedDBs {
		checked, online := resource.health.Status()
		status := contracts.FrontendSourceHealthStatusUnknown
		if checked {
			status = contracts.FrontendSourceHealthStatusOffline
			if online {
				status = contracts.FrontendSourceHealthStatusOnline
			}
		}
		healthBySource[resource.source] = status
	}

	if capabilityProvider, ok := r.Provider.(interface {
		FrontendSourceCapabilities() []contracts.FrontendSourceCapability
	}); ok {
		for _, capability := range capabilityProvider.FrontendSourceCapabilities() {
			if capability.HealthStatus != "" {
				healthBySource[capability.Source] = capability.HealthStatus
			}
		}
	}

	result := make([]SourceHealth, 0, 3)
	appendSource := func(enabled bool, source contracts.FrontendSource) {
		if !enabled {
			return
		}
		status, ok := healthBySource[source]
		if !ok || status == "" {
			status = contracts.FrontendSourceHealthStatusUnknown
		}
		result = append(result, SourceHealth{Source: source, Status: status})
	}
	appendSource(r.FirebirdEnabled, contracts.FrontendSourceBridge)
	appendSource(r.PhoenixEnabled, contracts.FrontendSourcePhoenix)
	appendSource(r.CASLEnabled, contracts.FrontendSourceCASL)
	return result
}

// Close shuts down provider background work and closes opened databases.
func (r *Runtime) Close() {
	if r == nil {
		return
	}
	if r.Provider != nil {
		if shutdowner, ok := r.Provider.(contracts.ShutdownProvider); ok {
			shutdowner.Shutdown()
		}
	}
	for _, resource := range r.managedDBs {
		if resource.healthCancel != nil {
			resource.healthCancel()
		}
		if resource.db != nil {
			_ = resource.db.Close()
		}
	}
	r.managedDBs = nil
}

func enabledSources(cfg config.DBConfig) (firebird bool, phoenix bool, casl bool) {
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
