package database

import (
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
	db.SetMaxOpenConns(25)                 // Макс. активних з'єднань
	db.SetMaxIdleConns(5)                  // Макс. з'єднань у черзі
	db.SetConnMaxLifetime(time.Minute * 5) // Час життя з'єднання (менше ніж таймаут Firebird)
	zlog.Debug().Int("maxOpenConns", 25).Int("maxIdleConns", 5).Str("maxConnLifetime", "5m").Msg("Пул з'єднань налаштовано")

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

func StartHealthCheck(db *sqlx.DB) {
	zlog.Info().Msg("Запуск моніторингу здоров'я БД (перевірка кожні 30 сек)...")
	go func() {
		checkCount := 0
		failCount := 0
		for {
			time.Sleep(30 * time.Second)
			checkCount++
			if err := db.Ping(); err != nil {
				failCount++
				zlog.Warn().Err(err).Int("failCount", failCount).Msg("Втрачено зв'язок з Firebird!")
				// Тут можна додати логіку сповіщення
				if failCount >= 3 {
					zlog.Error().Err(err).Int("consecutiveFailures", failCount).Msg("Багаторазові відмови з'єднання з БД!")
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
}
