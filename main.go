package main

import (
	"fmt"
	"runtime/debug"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

func main() {
	// Ініціалізація логера
	logConfig := logger.DefaultConfig()
	if err := logger.Setup(logConfig); err != nil {
		fmt.Printf("Помилка налаштування логера: %v\n", err)
	}

	log.Info().Str("level", logConfig.LogLevel).Str("logDir", logConfig.LogDir).Msg("Запуск програми - АРМ Пожежної Безпеки v1.0")

	// Додаємо базове відновлення після паніки
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("КРИТИЧНА ПОМИЛКА (Panic)")
		}
		log.Info().Msg("Завершення програми")
	}()

	// Створюємо додаток
	log.Debug().Msg("Ініціалізація додатку...")

	prefs := config.NewSimplePreferences("settings.json")

	// Завантажуємо налаштування БД
	dbCfg := config.LoadDBConfig(prefs)
	dsn := dbCfg.ToDSN()

	// Ініціалізуємо БД
	db := database.InitDB(dsn)
	database.StartHealthCheck(db)

	// Ініціалізація провайдера даних
	dataProvider := data.NewDBDataProvider(db, dsn)

	p := tea.NewProgram(tui.NewModel(dataProvider), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Error().Err(err).Msg("Помилка запуску TUI")
	}

	if db != nil {
		db.Close()
	}
}
