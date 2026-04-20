package main

import (
	"context"
	"os"

	"obj_catalog_fyne_v3/pkg/version"
	"obj_catalog_fyne_v3/pkg/wailsbridge"

	"github.com/rs/zerolog/log"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	bridge := wailsbridge.NewFrontendV1Service(emptyFrontendBackend{})
	runtimeController := newOperatorRuntimeController(bridge)
	settingsService := newOperatorSettingsService(runtimeController)
	journalWSCleanup := func() {}

	frontendBackend, closeFn, err := bootstrapFrontendBackend()
	if err != nil {
		log.Warn().Err(err).Msg("Operator Wails: live backend init failed, fallback to shell-only mode")
	} else {
		runtimeController.replaceBackend(frontendBackend, closeFn)
	}

	streamServer, streamErr := startJournalStreamServer(bridge)
	if streamErr != nil {
		log.Warn().Err(streamErr).Msg("Operator Wails: journal websocket server disabled")
	} else {
		journalWSCleanup = streamServer.shutdown
	}

	err = wails.Run(&options.App{
		Title:            "АРМ Пожежної Безпеки — Operator",
		Width:            1280,
		Height:           768,
		MinWidth:         1180,
		MinHeight:        720,
		BackgroundColour: options.NewRGBA(9, 13, 22, 1),
		AssetServer: &assetserver.Options{
			Assets: os.DirFS("../../frontend/dist"),
		},
		OnStartup: func(ctx context.Context) {
			_ = ctx
			log.Info().
				Str("version", version.Current().String()).
				Msg("Operator Wails shell started")
		},
		OnShutdown: func(ctx context.Context) {
			_ = ctx
			journalWSCleanup()
			runtimeController.shutdown()
		},
		Bind: []interface{}{
			bridge,
			settingsService,
		},
	})
	if err != nil {
		panic(err)
	}
}
