//go:build qt

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/qtapp"
	"obj_catalog_fyne_v3/pkg/version"

	"github.com/rs/zerolog/log"
)

func main() {
	ver := version.Current()
	if len(os.Args) > 1 {
		arg := strings.TrimSpace(strings.ToLower(os.Args[1]))
		if arg == "--version" || arg == "-version" || arg == "-v" {
			fmt.Println(ver.FullText())
			return
		}
	}

	logConfig := logger.DefaultConfig()
	if executable, err := os.Executable(); err == nil {
		logConfig.LogDir = filepath.Join(filepath.Dir(executable), "log")
	}
	if err := logger.Setup(logConfig); err != nil {
		fmt.Printf("Помилка налаштування логера: %v\n", err)
	}

	log.Info().
		Str("level", logConfig.LogLevel).
		Str("logDir", logConfig.LogDir).
		Str("version", ver.String()).
		Msg("Запуск Qt UI - АРМ Пожежної Безпеки")

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("КРИТИЧНА ПОМИЛКА (Panic)")
		}
		log.Info().Msg("Завершення Qt UI")
	}()

	app := qtapp.NewApplication()
	os.Exit(app.Run())
}
