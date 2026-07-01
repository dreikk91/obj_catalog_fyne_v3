package application

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/ids"
)

type managedDBResource struct {
	label        string
	db           *sqlx.DB
	healthCancel context.CancelFunc
}

type providerBuildResult struct {
	provider        contracts.DataProvider
	phoenixProvider *data.PhoenixDataProvider
	managedDBs      []managedDBResource
	firebirdEnabled bool
	phoenixEnabled  bool
	caslEnabled     bool
}

func buildDataProviderFromConfig(cfg config.DBConfig, pref fyne.Preferences, verifyConnectivity bool) (providerBuildResult, error) {
	firebirdEnabled := cfg.FirebirdEnabled
	phoenixEnabled := cfg.PhoenixEnabled
	caslEnabled := cfg.CASLEnabled

	switch cfg.NormalizedMode() {
	case config.BackendModePhoenix:
		phoenixEnabled = true
	case config.BackendModeCASLCloud:
		caslEnabled = true
	}

	if !firebirdEnabled && !phoenixEnabled && !caslEnabled {
		firebirdEnabled = true
	}

	result := providerBuildResult{
		managedDBs:      make([]managedDBResource, 0, 2),
		firebirdEnabled: firebirdEnabled,
		phoenixEnabled:  phoenixEnabled,
		caslEnabled:     caslEnabled,
	}

	sources := make([]data.ProviderSource, 0, 3)

	if firebirdEnabled {
		dsn := cfg.FirebirdDSN()
		db, err := database.InitNamedDB("firebirdsql", dsn, "БД/МІСТ")
		if err != nil {
			return providerBuildResult{}, err
		}
		if verifyConnectivity {
			if err := db.Ping(); err != nil {
				_ = db.Close()
				return providerBuildResult{}, fmt.Errorf("firebird ping failed: %w", err)
			}
		}
		result.managedDBs = append(result.managedDBs, managedDBResource{
			label:        "БД/МІСТ",
			db:           db,
			healthCancel: database.StartNamedHealthCheck(db, "БД/МІСТ"),
		})
		sources = append(sources, data.ProviderSource{
			Name: "bridge",
			Provider: backend.NewDBProvider(
				db,
				dsn,
				data.WithVodafoneConfigStore(config.NewPreferencesVodafoneConfigStore(pref)),
				data.WithKyivstarConfigStore(config.NewPreferencesKyivstarConfigStore(pref)),
			),
		})
	}

	if phoenixEnabled {
		dsn := cfg.PhoenixDSN()
		db, err := database.InitNamedDB("sqlserver", dsn, "Phoenix")
		if err != nil {
			closeManagedDBResources(result.managedDBs)
			return providerBuildResult{}, err
		}
		if verifyConnectivity {
			if err := db.Ping(); err != nil {
				_ = db.Close()
				closeManagedDBResources(result.managedDBs)
				return providerBuildResult{}, fmt.Errorf("phoenix ping failed: %w", err)
			}
		}
		phoenixProvider := data.NewPhoenixDataProvider(db, dsn)
		settingsCtx, settingsCancel := context.WithTimeout(context.Background(), 10*time.Second)
		settingsErr := phoenixProvider.ConfigureRuntime(settingsCtx, cfg)
		settingsCancel()
		if settingsErr != nil {
			_ = db.Close()
			closeManagedDBResources(result.managedDBs)
			return providerBuildResult{}, fmt.Errorf("phoenix runtime settings: %w", settingsErr)
		}
		if !verifyConnectivity {
			if err := phoenixProvider.StartControlCenterSession(); err != nil {
				log.Warn().Err(err).Msg("Phoenix UDP недоступний; робота з БД продовжується")
			}
		}
		result.phoenixProvider = phoenixProvider
		result.managedDBs = append(result.managedDBs, managedDBResource{
			label:        "Phoenix",
			db:           db,
			healthCancel: database.StartNamedHealthCheck(db, "Phoenix"),
		})
		sources = append(sources, data.ProviderSource{
			Name:         "phoenix",
			Provider:     phoenixProvider,
			OwnsObjectID: ids.IsPhoenixObjectID,
			OwnsAlarmID:  ids.IsPhoenixObjectID,
		})
	}

	if caslEnabled {
		caslProvider := backend.NewCASLCloudProvider(
			cfg.CASLBaseURL,
			cfg.CASLToken,
			cfg.CASLPultID,
			cfg.CASLEmail,
			cfg.CASLPass,
		)
		sources = append(sources, data.ProviderSource{
			Name:         "casl",
			Provider:     caslProvider,
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
		})
	}

	result.provider = backend.NewMultiSourceProvider(sources...)
	return result, nil
}

func closeManagedDBResources(resources []managedDBResource) {
	for _, resource := range resources {
		if resource.healthCancel != nil {
			resource.healthCancel()
		}
		if resource.db != nil {
			_ = resource.db.Close()
		}
	}
}
