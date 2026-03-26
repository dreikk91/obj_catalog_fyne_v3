package main

import (
	"fmt"
	"runtime/debug"

	"obj_catalog_fyne_v3/pkg/gui"
	"obj_catalog_fyne_v3/pkg/logger"

	"github.com/rs/zerolog/log"
)

func main() {
	logConfig := logger.DefaultConfig()
	if err := logger.Setup(logConfig); err != nil {
		fmt.Printf("Помилка налаштування логера: %v\n", err)
	}

	log.Info().Str("level", logConfig.LogLevel).Str("logDir", logConfig.LogDir).Msg("Запуск програми - АРМ Пожежної Безпеки v1.0")

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("КРИТИЧНА ПОМИЛКА (Panic)")
		}
		log.Info().Msg("Завершення програми")
	}()

	log.Debug().Msg("Ініціалізація GUI додатку...")
	application := gui.NewApplication()
	log.Info().Msg("Додаток ініціалізовано. Запуск UI...")
	application.Run()
}
