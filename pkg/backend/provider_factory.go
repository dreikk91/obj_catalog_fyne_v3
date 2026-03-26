package backend

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"

	"github.com/jmoiron/sqlx"
)

// NewDBProvider creates backend data provider implementation and returns it as GUI-facing contract.
func NewDBProvider(db *sqlx.DB, dsn string) contracts.DataProvider {
	return data.NewDBDataProvider(db, dsn)
}

// AsAdminProvider returns admin capabilities when backend implementation supports them.
func AsAdminProvider(provider contracts.DataProvider) (contracts.AdminProvider, bool) {
	admin, ok := provider.(contracts.AdminProvider)
	return admin, ok
}

var _ contracts.DataProvider = (*data.DBDataProvider)(nil)
var _ contracts.AdminProvider = (*data.DBDataProvider)(nil)
