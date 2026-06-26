package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/caslcompat"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/data"
	"obj_catalog_fyne_v3/pkg/database"
	"obj_catalog_fyne_v3/pkg/ids"
)

func main() {
	configPath := flag.String("config", "casl-fixture-gateway.json", "JSON config path; created with defaults when missing")
	addr := flag.String("addr", "127.0.0.1:50003", "HTTP API listen address")
	wsAddr := flag.String("ws-addr", "127.0.0.1:23322", "WebSocket listen address")
	autoPort := flag.Bool("auto-port", true, "try following ports when the requested HTTP or WebSocket port is unavailable")
	corsOrigin := flag.String("cors-origin", "*", "Access-Control-Allow-Origin value for browser testing")
	serveCASLUI := flag.Bool("serve-casl-ui", true, "serve CASL web UI static files on the HTTP API port")
	caslRoot := flag.String("casl-root", defaultCASLRoot(), "CASL http-api directory containing public, configurator_4L, and casl-technic")
	dataSource := flag.String("data-source", "fixture", "data source: fixture, env, or config")
	shutdownTimeout := flag.Duration("shutdown-timeout", 5*time.Second, "graceful shutdown timeout")
	flag.Parse()

	explicitFlags := visitedFlags()
	gatewayCfg, createdConfig, err := loadOrCreateGatewayConfig(strings.TrimSpace(*configPath))
	if err != nil {
		log.Fatal(err)
	}
	applyExplicitFlagOverrides(&gatewayCfg, explicitFlags, *addr, *wsAddr, *autoPort, *corsOrigin, *serveCASLUI, *caslRoot, *dataSource, *shutdownTimeout)
	if createdConfig {
		log.Printf("created default gateway config: %s", strings.TrimSpace(*configPath))
	}

	httpListener, httpAddr, err := listenTCPWithFallback(strings.TrimSpace(gatewayCfg.Addr), gatewayCfg.AutoPort)
	if err != nil {
		log.Fatal(err)
	}
	defer httpListener.Close()

	wsListener, resolvedWSAddr, err := listenTCPWithFallback(strings.TrimSpace(gatewayCfg.WSAddr), gatewayCfg.AutoPort)
	if err != nil {
		log.Fatal(err)
	}
	defer wsListener.Close()

	wsURL := "ws://" + resolvedWSAddr
	handler, cleanup, err := buildGatewayHandler(strings.TrimSpace(gatewayCfg.DataSource), wsURL, gatewayCfg.Database.toDBConfig())
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	var httpHandler http.Handler = handler
	if gatewayCfg.ServeCASLUI {
		httpHandler = caslcompat.NewStaticSiteHandler(handler, caslcompat.StaticSiteOptions{
			CASLRootDir: strings.TrimSpace(gatewayCfg.CASLRoot),
		})
	}
	httpHandler = withCORS(httpHandler, strings.TrimSpace(gatewayCfg.CORSOrigin))

	log.Printf("CASL fixture HTTP API listening on http://%s", httpAddr)
	log.Printf("CASL fixture WebSocket listening on %s/", wsURL)
	if gatewayCfg.ServeCASLUI && strings.TrimSpace(gatewayCfg.CASLRoot) != "" {
		log.Printf("CASL web UI static root: %s", strings.TrimSpace(gatewayCfg.CASLRoot))
	}

	httpServer := &http.Server{Handler: httpHandler}
	wsServer := &http.Server{Handler: handler}
	serveResults := make(chan serveResult, 2)

	go serveHTTP("HTTP API", httpServer, httpListener, serveResults)
	go serveHTTP("WebSocket", wsServer, wsListener, serveResults)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	waitFor := 2
	var fatalErr error
	select {
	case <-signalCtx.Done():
		log.Printf("shutdown signal received")
	case result := <-serveResults:
		waitFor--
		if result.err != nil {
			fatalErr = result.err
			log.Printf("%s server failed: %v", result.name, result.err)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), gatewayCfg.ShutdownTimeout.Duration())
	defer cancel()
	if err := shutdownServers(shutdownCtx, httpServer, wsServer); err != nil && fatalErr == nil {
		fatalErr = err
	}

	for ; waitFor > 0; waitFor-- {
		result := <-serveResults
		if result.err != nil && fatalErr == nil {
			fatalErr = result.err
			log.Printf("%s server failed: %v", result.name, result.err)
		}
	}

	if fatalErr != nil {
		log.Fatal(fatalErr)
	}
	log.Printf("CASL fixture gateway stopped")
}

type serveResult struct {
	name string
	err  error
}

func serveHTTP(name string, server *http.Server, listener net.Listener, results chan<- serveResult) {
	err := server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		err = nil
	}
	results <- serveResult{name: name, err: err}
}

func shutdownServers(ctx context.Context, servers ...*http.Server) error {
	errorsText := make([]string, 0, len(servers))
	for _, server := range servers {
		if server == nil {
			continue
		}
		if err := server.Shutdown(ctx); err != nil {
			errorsText = append(errorsText, err.Error())
		}
	}
	if len(errorsText) > 0 {
		return loggableError("server shutdown failed: " + strings.Join(errorsText, "; "))
	}
	return nil
}

func buildGatewayHandler(source string, wsURL string, cfg config.DBConfig) (*caslcompat.Handler, func(), error) {
	switch strings.ToLower(strings.TrimSpace(source)) {
	case "", "fixture":
		return caslcompat.NewFixtureHandlerWithWSURL(wsURL), func() {}, nil
	case "env", "config":
		provider, upstream, cleanup, err := buildDataProvider(cfg)
		if err != nil {
			return nil, func() {}, err
		}
		options := caslcompat.ProviderFixtureOptions{
			SourceName: "env",
			DeviceType: caslcompat.UnifiedDeviceType{
				Type:      "ENV_PROVIDER_GENERIC",
				NameUK:    "Зовнішнє джерело",
				NameRU:    "Внешний источник",
				NameEN:    "External source",
				MaxLines:  999,
				MaxGroups: 999,
			},
		}
		handler := caslcompat.NewProviderHandlerWithWSURL(provider, options, wsURL)
		handler.SetCommandUpstream(upstream)
		return handler, cleanup, nil
	default:
		return nil, func() {}, loggableError("unsupported data source: " + source)
	}
}

func buildDataProvider(cfg config.DBConfig) (contracts.DataProvider, caslcompat.CommandUpstream, func(), error) {
	firebirdEnabled, phoenixEnabled, caslEnabled := resolveEnabledSources(cfg)
	sources := make([]data.ProviderSource, 0, 3)
	cleanup := make([]func(), 0, 2)
	initErrors := make([]string, 0, 3)
	var caslUpstream caslcompat.CommandUpstream

	if firebirdEnabled {
		dsn := cfg.FirebirdDSN()
		db, err := database.InitNamedDB("firebirdsql", dsn, "БД/МІСТ")
		if err != nil {
			initErrors = append(initErrors, "firebird: "+err.Error())
		} else {
			cleanup = append(cleanup, func() { _ = db.Close() })
			sources = append(sources, data.ProviderSource{
				Name:     "bridge",
				Provider: backend.NewDBProvider(db, dsn),
			})
		}
	}

	if phoenixEnabled {
		dsn := cfg.PhoenixDSN()
		db, err := database.InitNamedDB("sqlserver", dsn, "Phoenix")
		if err != nil {
			initErrors = append(initErrors, "phoenix: "+err.Error())
		} else {
			cleanup = append(cleanup, func() { _ = db.Close() })
			sources = append(sources, data.ProviderSource{
				Name:         "phoenix",
				Provider:     backend.NewPhoenixProvider(db, dsn),
				OwnsObjectID: ids.IsPhoenixObjectID,
				OwnsAlarmID:  ids.IsPhoenixObjectID,
			})
		}
	}

	if caslEnabled {
		caslProvider := data.NewCASLCloudProvider(
			cfg.CASLBaseURL,
			cfg.CASLToken,
			cfg.CASLPultID,
			cfg.CASLEmail,
			cfg.CASLPass,
		)
		caslUpstream = caslProvider
		sources = append(sources, data.ProviderSource{
			Name:         "casl",
			Provider:     caslProvider,
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
		})
	}

	if len(sources) == 0 {
		for _, closeFn := range cleanup {
			closeFn()
		}
		return nil, nil, func() {}, loggableError("failed to initialize any data source: " + strings.Join(initErrors, "; "))
	}

	provider := backend.NewMultiSourceProvider(sources...)
	return provider, caslUpstream, func() {
		if shutdowner, ok := provider.(contracts.ShutdownProvider); ok {
			shutdowner.Shutdown()
		}
		for _, closeFn := range cleanup {
			closeFn()
		}
	}, nil
}

func resolveEnabledSources(cfg config.DBConfig) (firebird bool, phoenix bool, casl bool) {
	firebird = cfg.FirebirdEnabled
	phoenix = cfg.PhoenixEnabled
	casl = cfg.CASLEnabled

	switch cfg.NormalizedMode() {
	case config.BackendModePhoenix:
		phoenix = true
	case config.BackendModeCASLCloud:
		casl = true
	}
	if !firebird && !phoenix && !casl {
		firebird = true
	}
	return firebird, phoenix, casl
}

type gatewayConfig struct {
	Addr            string          `json:"addr"`
	WSAddr          string          `json:"ws_addr"`
	AutoPort        bool            `json:"auto_port"`
	CORSOrigin      string          `json:"cors_origin"`
	ServeCASLUI     bool            `json:"serve_casl_ui"`
	CASLRoot        string          `json:"casl_root"`
	DataSource      string          `json:"data_source"`
	ShutdownTimeout configDuration  `json:"shutdown_timeout"`
	Database        gatewayDBConfig `json:"database"`
}

type gatewayDBConfig struct {
	User            string `json:"user"`
	Password        string `json:"password"`
	Host            string `json:"host"`
	Port            string `json:"port"`
	Path            string `json:"path"`
	Params          string `json:"params"`
	FirebirdEnabled bool   `json:"firebird_enabled"`
	PhoenixEnabled  bool   `json:"phoenix_enabled"`
	PhoenixUser     string `json:"phoenix_user"`
	PhoenixPassword string `json:"phoenix_password"`
	PhoenixHost     string `json:"phoenix_host"`
	PhoenixPort     string `json:"phoenix_port"`
	PhoenixInstance string `json:"phoenix_instance"`
	PhoenixDatabase string `json:"phoenix_database"`
	PhoenixParams   string `json:"phoenix_params"`
	CASLEnabled     bool   `json:"casl_enabled"`
	Mode            string `json:"mode"`
	CASLBaseURL     string `json:"casl_base_url"`
	CASLToken       string `json:"casl_token"`
	CASLEmail       string `json:"casl_email"`
	CASLPass        string `json:"casl_password"`
	CASLPultID      int64  `json:"casl_pult_id"`
}

type configDuration time.Duration

func (d configDuration) Duration() time.Duration {
	if d <= 0 {
		return 5 * time.Second
	}
	return time.Duration(d)
}

func (d configDuration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *configDuration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed, parseErr := time.ParseDuration(strings.TrimSpace(text))
		if parseErr != nil {
			return fmt.Errorf("parse duration %q: %w", text, parseErr)
		}
		*d = configDuration(parsed)
		return nil
	}

	var seconds float64
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}
	*d = configDuration(time.Duration(seconds * float64(time.Second)))
	return nil
}

func defaultGatewayConfig() gatewayConfig {
	return gatewayConfig{
		Addr:            "127.0.0.1:50003",
		WSAddr:          "127.0.0.1:23322",
		AutoPort:        true,
		CORSOrigin:      "*",
		ServeCASLUI:     true,
		CASLRoot:        defaultCASLRoot(),
		DataSource:      "fixture",
		ShutdownTimeout: configDuration(5 * time.Second),
		Database: gatewayDBConfig{
			User:            "SYSDBA",
			Password:        "masterkey",
			Host:            "localhost",
			Port:            "3050",
			Path:            "C:/MOST.PM/BASE/MOST5.FDB",
			Params:          "charset=WIN1251&auth_plugin_name=Srp",
			FirebirdEnabled: true,
			PhoenixEnabled:  false,
			PhoenixUser:     "sa",
			PhoenixPassword: "",
			PhoenixHost:     "localhost",
			PhoenixPort:     "",
			PhoenixInstance: "PHOENIX4",
			PhoenixDatabase: "Pult4DB",
			PhoenixParams:   "encrypt=disable&trustservercertificate=true",
			CASLEnabled:     false,
			Mode:            config.BackendModeFirebird,
			CASLBaseURL:     "http://127.0.0.1:50003",
			CASLToken:       "",
			CASLEmail:       "",
			CASLPass:        "",
			CASLPultID:      0,
		},
	}
}

func loadOrCreateGatewayConfig(path string) (gatewayConfig, bool, error) {
	cfg := defaultGatewayConfig()
	if path == "" {
		return cfg, false, nil
	}

	body, err := os.ReadFile(path)
	if err == nil {
		if err := json.Unmarshal(body, &cfg); err != nil {
			return gatewayConfig{}, false, fmt.Errorf("read gateway config %q: %w", path, err)
		}
		cfg.applyDefaults()
		return cfg, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return gatewayConfig{}, false, fmt.Errorf("read gateway config %q: %w", path, err)
	}

	if err := writeGatewayConfig(path, cfg); err != nil {
		return gatewayConfig{}, false, err
	}
	return cfg, true, nil
}

func writeGatewayConfig(path string, cfg gatewayConfig) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create gateway config directory %q: %w", dir, err)
		}
	}

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode gateway config: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write gateway config %q: %w", path, err)
	}
	return nil
}

func (cfg *gatewayConfig) applyDefaults() {
	defaults := defaultGatewayConfig()
	if strings.TrimSpace(cfg.Addr) == "" {
		cfg.Addr = defaults.Addr
	}
	if strings.TrimSpace(cfg.WSAddr) == "" {
		cfg.WSAddr = defaults.WSAddr
	}
	if strings.TrimSpace(cfg.CORSOrigin) == "" {
		cfg.CORSOrigin = defaults.CORSOrigin
	}
	if strings.TrimSpace(cfg.DataSource) == "" {
		cfg.DataSource = defaults.DataSource
	}
	if cfg.ShutdownTimeout <= 0 {
		cfg.ShutdownTimeout = defaults.ShutdownTimeout
	}
	cfg.Database.applyDefaults()
}

func (cfg *gatewayDBConfig) applyDefaults() {
	defaults := defaultGatewayConfig().Database
	if strings.TrimSpace(cfg.User) == "" {
		cfg.User = defaults.User
	}
	if strings.TrimSpace(cfg.Password) == "" {
		cfg.Password = defaults.Password
	}
	if strings.TrimSpace(cfg.Host) == "" {
		cfg.Host = defaults.Host
	}
	if strings.TrimSpace(cfg.Port) == "" {
		cfg.Port = defaults.Port
	}
	if strings.TrimSpace(cfg.Path) == "" {
		cfg.Path = defaults.Path
	}
	if strings.TrimSpace(cfg.Params) == "" {
		cfg.Params = defaults.Params
	}
	if strings.TrimSpace(cfg.PhoenixUser) == "" {
		cfg.PhoenixUser = defaults.PhoenixUser
	}
	if strings.TrimSpace(cfg.PhoenixHost) == "" {
		cfg.PhoenixHost = defaults.PhoenixHost
	}
	if strings.TrimSpace(cfg.PhoenixInstance) == "" {
		cfg.PhoenixInstance = defaults.PhoenixInstance
	}
	if strings.TrimSpace(cfg.PhoenixDatabase) == "" {
		cfg.PhoenixDatabase = defaults.PhoenixDatabase
	}
	if strings.TrimSpace(cfg.PhoenixParams) == "" {
		cfg.PhoenixParams = defaults.PhoenixParams
	}
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = defaults.Mode
	}
	if strings.TrimSpace(cfg.CASLBaseURL) == "" {
		cfg.CASLBaseURL = defaults.CASLBaseURL
	}
}

func (cfg gatewayDBConfig) toDBConfig() config.DBConfig {
	return config.DBConfig{
		User:            cfg.User,
		Password:        cfg.Password,
		Host:            cfg.Host,
		Port:            cfg.Port,
		Path:            cfg.Path,
		Params:          cfg.Params,
		FirebirdEnabled: cfg.FirebirdEnabled,
		PhoenixEnabled:  cfg.PhoenixEnabled,
		PhoenixUser:     cfg.PhoenixUser,
		PhoenixPassword: cfg.PhoenixPassword,
		PhoenixHost:     cfg.PhoenixHost,
		PhoenixPort:     cfg.PhoenixPort,
		PhoenixInstance: cfg.PhoenixInstance,
		PhoenixDatabase: cfg.PhoenixDatabase,
		PhoenixParams:   cfg.PhoenixParams,
		CASLEnabled:     cfg.CASLEnabled,
		Mode:            cfg.Mode,
		CASLBaseURL:     cfg.CASLBaseURL,
		CASLToken:       cfg.CASLToken,
		CASLEmail:       cfg.CASLEmail,
		CASLPass:        cfg.CASLPass,
		CASLPultID:      cfg.CASLPultID,
	}
}

func visitedFlags() map[string]bool {
	visited := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		visited[f.Name] = true
	})
	return visited
}

func applyExplicitFlagOverrides(
	cfg *gatewayConfig,
	visited map[string]bool,
	addr string,
	wsAddr string,
	autoPort bool,
	corsOrigin string,
	serveCASLUI bool,
	caslRoot string,
	dataSource string,
	shutdownTimeout time.Duration,
) {
	if cfg == nil {
		return
	}
	if visited["addr"] {
		cfg.Addr = addr
	}
	if visited["ws-addr"] {
		cfg.WSAddr = wsAddr
	}
	if visited["auto-port"] {
		cfg.AutoPort = autoPort
	}
	if visited["cors-origin"] {
		cfg.CORSOrigin = corsOrigin
	}
	if visited["serve-casl-ui"] {
		cfg.ServeCASLUI = serveCASLUI
	}
	if visited["casl-root"] {
		cfg.CASLRoot = caslRoot
	}
	if visited["data-source"] {
		cfg.DataSource = dataSource
	}
	if visited["shutdown-timeout"] {
		cfg.ShutdownTimeout = configDuration(shutdownTimeout)
	}
}

type loggableError string

func (e loggableError) Error() string {
	return string(e)
}

func withCORS(next http.Handler, origin string) http.Handler {
	if strings.TrimSpace(origin) == "" {
		origin = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func listenTCPWithFallback(addr string, autoPort bool) (net.Listener, string, error) {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:0"
	}

	listener, err := net.Listen("tcp", addr)
	if err == nil {
		return listener, listener.Addr().String(), nil
	}
	if !autoPort {
		return nil, "", err
	}

	host, port, splitErr := net.SplitHostPort(addr)
	if splitErr != nil {
		return nil, "", err
	}
	start, parseErr := strconv.Atoi(port)
	if parseErr != nil || start <= 0 {
		return nil, "", err
	}

	for candidate := start + 1; candidate <= start+100; candidate++ {
		nextAddr := net.JoinHostPort(host, strconv.Itoa(candidate))
		listener, listenErr := net.Listen("tcp", nextAddr)
		if listenErr == nil {
			log.Printf("requested address %s is unavailable (%v); using %s", addr, err, listener.Addr().String())
			return listener, listener.Addr().String(), nil
		}
	}

	return nil, "", err
}

func defaultCASLRoot() string {
	const path = `C:\casl_cloud\http-api`
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return path
	}
	return ""
}
