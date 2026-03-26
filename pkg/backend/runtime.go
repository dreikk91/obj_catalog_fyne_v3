package backend

import (
	"fmt"
	"sync"

	"obj_catalog_fyne_v3/pkg/config"
	contracts "obj_catalog_fyne_v3/pkg/contracts"
	dataimpl "obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"

	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
)

// Service описує backend-частину застосунку.
// GUI працює з бекендом тільки через цей контракт.
type Service interface {
	Provider() contracts.DataProvider
	Reconnect(cfg config.DBConfig) error
	Close()
}

// Runtime керує життєвим циклом з'єднання БД та провайдера даних.
type Runtime struct {
	mu sync.RWMutex

	db           *sqlx.DB
	healthCancel func()
	provider     contracts.DataProvider
}

func NewRuntime(cfg config.DBConfig) *Runtime {
	dsn := cfg.ToDSN()
	db := database.InitDB(dsn)

	return &Runtime{
		db:           db,
		healthCancel: database.StartHealthCheck(db),
		provider:     dataimpl.NewDBDataProvider(db, dsn),
	}
}

func (r *Runtime) Provider() contracts.DataProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.provider
}

func (r *Runtime) Reconnect(cfg config.DBConfig) error {
	dsn := cfg.ToDSN()
	newDB := database.InitDB(dsn)
	if err := newDB.Ping(); err != nil {
		_ = newDB.Close()
		return fmt.Errorf("database ping failed: %w", err)
	}

	newCancel := database.StartHealthCheck(newDB)
	newProvider := dataimpl.NewDBDataProvider(newDB, dsn)

	r.mu.Lock()
	oldDB := r.db
	oldCancel := r.healthCancel
	r.db = newDB
	r.healthCancel = newCancel
	r.provider = newProvider
	r.mu.Unlock()

	if oldCancel != nil {
		oldCancel()
	}
	if oldDB != nil {
		if err := oldDB.Close(); err != nil {
			log.Warn().Err(err).Msg("не вдалося закрити старе з'єднання БД під час reconnect")
		}
	}

	return nil
}

func (r *Runtime) Close() {
	r.mu.Lock()
	db := r.db
	cancel := r.healthCancel
	r.db = nil
	r.healthCancel = nil
	r.provider = nil
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if db != nil {
		if err := db.Close(); err != nil {
			log.Warn().Err(err).Msg("не вдалося коректно закрити з'єднання БД")
		}
	}
}
