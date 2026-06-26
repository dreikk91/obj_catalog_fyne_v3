package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/dataruntime"
	"obj_catalog_fyne_v3/pkg/logger"
	"obj_catalog_fyne_v3/pkg/simwatchdog"
	"obj_catalog_fyne_v3/pkg/version"
)

func main() {
	configPath := flag.String("config", "sim-watchdog.json", "JSON config path")
	interval := flag.Duration("interval", 3*time.Minute, "object check interval")
	historyPath := flag.String("history", "log/sim-watchdog-history.json", "JSON reboot history path")
	dryRun := flag.Bool("dry-run", false, "log planned reboots without sending operator API requests")
	includeNonBridge := flag.Bool("include-non-bridge", false, "also process Phoenix/CASL objects")
	verifyDB := flag.Bool("verify-db", true, "ping configured databases on startup")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	visited := visitedFlags()

	ver := version.Current()
	if *showVersion {
		fmt.Println(ver.FullText())
		return
	}

	logConfig := logger.DefaultConfig()
	logConfig.LogDir = "log/sim-watchdog"
	if err := logger.Setup(logConfig); err != nil {
		fmt.Printf("Помилка налаштування логера: %v\n", err)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Str("stack", string(debug.Stack())).
				Msg("sim-watchdog: panic")
			os.Exit(2)
		}
	}()

	opts := runtimeOptions{
		ConfigPath:       strings.TrimSpace(*configPath),
		PollInterval:     *interval,
		HistoryPath:      strings.TrimSpace(*historyPath),
		DryRun:           *dryRun,
		IncludeNonBridge: *includeNonBridge,
		VerifyDB:         *verifyDB,
		visitedFlags:     visited,
	}
	if err := run(opts); err != nil && !errors.Is(err, context.Canceled) {
		log.Error().Err(err).Msg("sim-watchdog stopped with error")
		os.Exit(1)
	}
	log.Info().Msg("sim-watchdog stopped")
}

type runtimeOptions struct {
	ConfigPath       string
	PollInterval     time.Duration
	HistoryPath      string
	DryRun           bool
	IncludeNonBridge bool
	VerifyDB         bool
	visitedFlags     map[string]bool
}

func run(opts runtimeOptions) error {
	dbCfg, store, runCfg, err := resolveRuntimeConfig(opts)
	if err != nil {
		return err
	}
	dbCfg.LogLevel = logger.SetLogLevel(dbCfg.LogLevel)

	runtime, err := dataruntime.New(dbCfg, store, runCfg.VerifyDB)
	if err != nil {
		return err
	}
	defer runtime.Close()

	runner, err := simwatchdog.NewRunner(
		runtime.Provider,
		data.NewKyivstarService(store),
		data.NewVodafoneService(store),
		store,
		simwatchdog.Options{
			PollInterval:     runCfg.PollInterval.Duration(),
			HistoryPath:      runCfg.HistoryPath,
			DryRun:           runCfg.DryRun,
			IncludeNonBridge: runCfg.IncludeNonBridge,
			MaxLastTestAge:   runCfg.MaxLastTestAge.Duration(),
		},
	)
	if err != nil {
		return err
	}

	log.Info().
		Str("config", opts.ConfigPath).
		Dur("interval", runCfg.PollInterval.Duration()).
		Dur("maxLastTestAge", runCfg.MaxLastTestAge.Duration()).
		Str("history", runCfg.HistoryPath).
		Bool("dryRun", runCfg.DryRun).
		Bool("includeNonBridge", runCfg.IncludeNonBridge).
		Bool("firebirdEnabled", runtime.FirebirdEnabled).
		Bool("phoenixEnabled", runtime.PhoenixEnabled).
		Bool("caslEnabled", runtime.CASLEnabled).
		Msg("sim-watchdog started")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runner.Run(ctx)
}

func resolveRuntimeConfig(opts runtimeOptions) (config.DBConfig, simwatchdog.ConfigStore, serviceConfig, error) {
	if strings.TrimSpace(opts.ConfigPath) == "" {
		return config.DBConfig{}, nil, serviceConfig{}, errors.New("config path is empty")
	}
	cfg, err := loadServiceConfig(opts.ConfigPath)
	if err != nil {
		return config.DBConfig{}, nil, serviceConfig{}, err
	}
	applyRuntimeFlagOverrides(&cfg, opts)
	return cfg.dbConfig(), newFileConfigStore(opts.ConfigPath, cfg), cfg, nil
}

func applyRuntimeFlagOverrides(cfg *serviceConfig, opts runtimeOptions) {
	if cfg == nil {
		return
	}
	if opts.visitedFlags["interval"] && opts.PollInterval > 0 {
		cfg.PollInterval = configDuration(opts.PollInterval)
	}
	if opts.visitedFlags["history"] && strings.TrimSpace(opts.HistoryPath) != "" {
		cfg.HistoryPath = strings.TrimSpace(opts.HistoryPath)
	}
	if opts.visitedFlags["dry-run"] {
		cfg.DryRun = opts.DryRun
	}
	if opts.visitedFlags["include-non-bridge"] {
		cfg.IncludeNonBridge = opts.IncludeNonBridge
	}
	if opts.visitedFlags["verify-db"] {
		cfg.VerifyDB = opts.VerifyDB
	}
	cfg.applyDefaults()
}

func visitedFlags() map[string]bool {
	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}
