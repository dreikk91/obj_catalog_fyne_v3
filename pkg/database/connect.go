package database

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
	_ "github.com/nakagami/firebirdsql" // Драйвер Firebird
	zlog "github.com/rs/zerolog/log"
)

// ConnectionHealth is the latest result of an active database connectivity check.
type ConnectionHealth struct {
	state atomic.Uint32
}

const (
	connectionHealthUnchecked uint32 = iota
	connectionHealthOnline
	connectionHealthOffline
)

// Status reports whether a connectivity check has completed and whether it succeeded.
func (h *ConnectionHealth) Status() (checked bool, online bool) {
	if h == nil {
		return false, false
	}
	switch h.state.Load() {
	case connectionHealthOnline:
		return true, true
	case connectionHealthOffline:
		return true, false
	default:
		return false, false
	}
}

func InitDB(connStr string) (*sqlx.DB, error) {
	return InitNamedDB("firebirdsql", connStr, "Firebird")
}

func InitNamedDB(driverName string, connStr string, label string) (*sqlx.DB, error) {
	dbLabel := strings.TrimSpace(label)
	if dbLabel == "" {
		dbLabel = driverName
	}

	zlog.Info().Str("driver", driverName).Str("label", dbLabel).Msg("Початок ініціалізації БД...")

	// sqlx.Open не відкриває фізичне з'єднання одразу
	zlog.Debug().Str("driver", driverName).Str("label", dbLabel).Msg("Відкриття драйвера БД...")
	db, err := sqlx.Open(driverName, connStr)
	if err != nil {
		zlog.Error().Err(err).Str("driver", driverName).Str("label", dbLabel).Msg("Критична помилка: не вдалося налаштувати драйвер БД")
		return nil, fmt.Errorf("database driver setup failed for %s: %w", dbLabel, err)
	}
	zlog.Debug().Msg("Драйвер відкритий")

	// Налаштування пулу з'єднань
	zlog.Debug().Msg("Налаштування пулу з'єднань...")
	db.SetMaxOpenConns(10)                  // Макс. активних з'єднань
	db.SetMaxIdleConns(2)                   // Макс. з'єднань у черзі
	db.SetConnMaxLifetime(time.Minute * 15) // Час життя з'єднання
	db.SetConnMaxIdleTime(time.Minute)
	zlog.Debug().Int("maxOpenConns", 10).Int("maxIdleConns", 2).Str("maxConnLifetime", "15m").Msg("Пул з'єднань налаштовано")

	// Перша фізична перевірка з'єднання
	zlog.Debug().Msg("Виконання першої перевірки з'єднання (ping)...")
	if err := PingWithTimeout(context.Background(), db, 5*time.Second); err != nil {
		zlog.Warn().Err(err).Str("label", dbLabel).Msg("БД недоступна при старті. Продовжуємо роботу, буде повторна спроба...")
		// Не припиняємо роботу - спробуємо пізніше
	} else {
		zlog.Info().Str("label", dbLabel).Msg("З'єднання з БД встановлено успішно")
	}

	return db, nil
}

func StartHealthCheck(db *sqlx.DB) context.CancelFunc {
	return StartNamedHealthCheck(db, "Firebird")
}

func StartNamedHealthCheck(db *sqlx.DB, label string) context.CancelFunc {
	cancel, _ := StartNamedHealthCheckWithStatus(db, label)
	return cancel
}

// StartNamedHealthCheckWithStatus monitors a database and exposes the latest
// connectivity result. The first check runs immediately.
func StartNamedHealthCheckWithStatus(db *sqlx.DB, label string) (context.CancelFunc, *ConnectionHealth) {
	dbLabel := strings.TrimSpace(label)
	if dbLabel == "" {
		dbLabel = "database"
	}

	zlog.Info().Str("label", dbLabel).Msg("Запуск моніторингу здоров'я БД (перевірка кожні 30 сек)...")
	ctx, cancel := context.WithCancel(context.Background())
	health := &ConnectionHealth{}
	go func() {
		checkCount := 0
		failCount := 0
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			checkCount++
			err := PingWithTimeout(ctx, db, 5*time.Second)
			if err == nil {
				health.state.Store(connectionHealthOnline)
			} else {
				health.state.Store(connectionHealthOffline)
			}
			if err != nil {
				if ctx.Err() != nil {
					zlog.Info().Str("label", dbLabel).Msg("Зупинка моніторингу здоров'я БД")
					return
				}
				if err.Error() == "sql: database is closed" {
					zlog.Info().Str("label", dbLabel).Msg("Моніторинг БД зупинено: з'єднання закрито")
					return
				}
				failCount++

				zlog.Warn().Err(err).Str("label", dbLabel).Int("failCount", failCount).Msg("Втрачено зв'язок з БД")
				// Відновлюємо пул при багаторазових збоях
				if failCount >= 3 {
					zlog.Error().Err(err).Str("label", dbLabel).Int("consecutiveFailures", failCount).Msg("Багаторазові відмови з'єднання з БД")
					ResetIdleConnections(db)
				} else {
					// Спроба "м'якого" відновлення - скидаємо простійні з'єднання
					zlog.Warn().Str("label", dbLabel).Msg("Спроба скидання пулу з'єднань...")
					ResetIdleConnections(db)
				}
			} else {
				if failCount > 0 {
					zlog.Info().Str("label", dbLabel).Int("afterFailures", failCount).Msg("З'єднання з БД відновлено")
					failCount = 0
				}
				zlog.Debug().Str("label", dbLabel).Int("checkNumber", checkCount).Msg("Перевірка здоров'я БД - OK")
			}

			select {
			case <-ctx.Done():
				zlog.Info().Str("label", dbLabel).Msg("Зупинка моніторингу здоров'я БД")
				return
			case <-ticker.C:
			}
		}
	}()
	return cancel, health
}

// PingWithTimeout bounds the caller even when a database driver does not
// return promptly after its context is canceled.
func PingWithTimeout(parent context.Context, db *sqlx.DB, timeout time.Duration) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	if parent == nil {
		parent = context.Background()
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	result := make(chan error, 1)
	go func() {
		result <- db.PingContext(ctx)
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ResetIdleConnections discards pooled idle connections after a transport failure.
func ResetIdleConnections(db *sqlx.DB) {
	if db == nil {
		return
	}
	db.SetMaxIdleConns(0)
	db.SetMaxIdleConns(2)
}
