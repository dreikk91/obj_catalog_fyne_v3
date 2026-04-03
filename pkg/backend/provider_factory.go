package backend

import (
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"

	"github.com/jmoiron/sqlx"
)

// NewDBProvider creates backend data provider implementation and returns it as GUI-facing contract.
func NewDBProvider(db *sqlx.DB, dsn string, opts ...data.DBProviderOption) contracts.DataProvider {
	return data.NewDBDataProvider(db, dsn, opts...)
}

// NewPhoenixProvider creates Phoenix MSSQL backend provider.
func NewPhoenixProvider(db *sqlx.DB, dsn string) contracts.DataProvider {
	return data.NewPhoenixDataProvider(db, dsn)
}

// NewCASLCloudProvider creates CASL Cloud API backend provider.
func NewCASLCloudProvider(baseURL string, token string, pultID int64, credentials ...string) contracts.DataProvider {
	return data.NewCASLCloudProvider(baseURL, token, pultID, credentials...)
}

// NewCombinedProvider creates a composite provider where primary source stays authoritative
// and secondary source augments monitoring data.
func NewCombinedProvider(primary contracts.DataProvider, secondary contracts.DataProvider) contracts.DataProvider {
	if secondary == nil {
		return primary
	}
	if primary == nil {
		return secondary
	}
	return data.NewCombinedDataProvider(primary, secondary)
}

// NewMultiSourceProvider builds a provider over an arbitrary number of sources.
func NewMultiSourceProvider(sources ...data.ProviderSource) contracts.DataProvider {
	return data.NewMultiSourceDataProvider(sources...)
}

type adminProviderResolver interface {
	AdminProvider() contracts.AdminProvider
}

// AsAdminProvider returns admin capabilities when backend implementation supports them.
func AsAdminProvider(provider contracts.DataProvider) (contracts.AdminProvider, bool) {
	admin, ok := provider.(contracts.AdminProvider)
	if ok {
		return admin, true
	}
	resolver, ok := provider.(adminProviderResolver)
	if !ok {
		return nil, false
	}
	admin = resolver.AdminProvider()
	if admin == nil {
		return nil, false
	}
	return admin, true
}

var _ contracts.DataProvider = (*data.DBDataProvider)(nil)
var _ contracts.AdminProvider = (*data.DBDataProvider)(nil)
var _ contracts.DataProvider = (*data.PhoenixDataProvider)(nil)
var _ contracts.DataProvider = (*data.CASLCloudProvider)(nil)
var _ contracts.DataProvider = (*data.CombinedDataProvider)(nil)
var _ config.VodafoneConfigStore = (*config.PreferencesVodafoneConfigStore)(nil)
var _ config.KyivstarConfigStore = (*config.PreferencesKyivstarConfigStore)(nil)
