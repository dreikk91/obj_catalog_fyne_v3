package database

import (
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/nakagami/firebirdsql" // Драйвер Firebird
)

func InitDB(connStr string) *sqlx.DB {
	// sqlx.Open не відкриває фізичне з'єднання одразу
	db, err := sqlx.Open("firebirdsql", connStr)
	if err != nil {
		log.Fatalf("Помилка конфігурації БД: %v", err)
	}

	// Налаштування пулу з'єднань
	db.SetMaxOpenConns(25)           // Макс. активних з'єднань
	db.SetMaxIdleConns(5)            // Макс. з'єднань у черзі
	db.SetConnMaxLifetime(time.Minute * 5) // Час життя з'єднання (менше ніж таймаут Firebird)

	// Перша фізична перевірка з'єднання
	if err := db.Ping(); err != nil {
		log.Printf("БД недоступна при старті: %v", err)
	}

	return db
}

func StartHealthCheck(db *sqlx.DB) {
	go func() {
		for {
			time.Sleep(30 * time.Second)
			if err := db.Ping(); err != nil {
				log.Printf("Увага: Втрачено зв'язок з Firebird! %v", err)
				// Тут можна додати логіку сповіщення
			}
		}
	}()
}