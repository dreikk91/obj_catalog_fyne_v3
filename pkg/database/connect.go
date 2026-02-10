package database

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/nakagami/firebirdsql" // Драйвер Firebird
	zlog "github.com/rs/zerolog/log"
)

func InitDB(connStr string) *sqlx.DB {
	zlog.Info().Msg("Початок ініціалізації БД Firebird...")

	// sqlx.Open не відкриває фізичне з'єднання одразу
	zlog.Debug().Msg("Відкриття драйвера БД...")
	db, err := sqlx.Open("firebirdsql", connStr)
	if err != nil {
		zlog.Error().Err(err).Msg("Критична помилка: не вдалося налаштувати драйвер Firebird")
		log.Fatalf("Помилка конфігурації БД: %v", err)
	}
	zlog.Debug().Msg("Драйвер відкритий")

	// Налаштування пулу з'єднань
	zlog.Debug().Msg("Налаштування пулу з'єднань...")
	db.SetMaxOpenConns(10)                  // Макс. активних з'єднань
	db.SetMaxIdleConns(2)                   // Макс. з'єднань у черзі
	db.SetConnMaxLifetime(time.Minute * 15) // Час життя з'єднання
	zlog.Debug().Int("maxOpenConns", 10).Int("maxIdleConns", 2).Str("maxConnLifetime", "15m").Msg("Пул з'єднань налаштовано")

	// Перша фізична перевірка з'єднання
	zlog.Debug().Msg("Виконання першої перевірки з'єднання (ping)...")
	if err := db.Ping(); err != nil {
		zlog.Warn().Err(err).Msg("БД недоступна при старті. Продовжуємо роботу, буде повторна спроба...")
		// Не припиняємо роботу - спробуємо пізніше
	} else {
		zlog.Info().Msg("З'єднання з БД встановлено успішно")
	}

	return db
}

func StartHealthCheck(db *sqlx.DB) context.CancelFunc {
	zlog.Info().Msg("Запуск моніторингу здоров'я БД (перевірка кожні 60 сек)...")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		checkCount := 0
		failCount := 0
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				zlog.Info().Msg("Зупинка моніторингу здоров'я БД")
				return
			case <-ticker.C:
			}
			checkCount++
			if err := db.Ping(); err != nil {
				if err.Error() == "sql: database is closed" {
					zlog.Info().Msg("Моніторинг БД зупинено: з'єднання закрито")
					return
				}
				failCount++

				zlog.Warn().Err(err).Int("failCount", failCount).Msg("Втрачено зв'язок з Firebird!")
				// Відновлюємо пул при багаторазових збоях
				if failCount >= 3 {
					zlog.Error().Err(err).Int("consecutiveFailures", failCount).Msg("Багаторазові відмови з'єднання з БД!")
					// Видаляємо мертві з'єднання з пулу
					db.SetMaxIdleConns(0)
					time.Sleep(500 * time.Millisecond)
					db.SetMaxIdleConns(2)

				} else {
					// Спроба "м'якого" відновлення - скидаємо простійні з'єднання
					zlog.Warn().Msg("Спроба скидання пулу з'єднань...")
					db.SetMaxIdleConns(0)
					time.Sleep(500 * time.Millisecond)
					db.SetMaxIdleConns(2)
				}
			} else {
				if failCount > 0 {
					zlog.Info().Int("afterFailures", failCount).Msg("З'єднання з БД відновлено")
					failCount = 0
				}
				zlog.Debug().Int("checkNumber", checkCount).Msg("Перевірка здоров'я БД - OK")
			}
		}
	}()
	return cancel
}
