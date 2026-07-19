package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rs/zerolog/log"
	applogger "obj_catalog_fyne_v3/pkg/logger"
)

const (
	PrefUser     = "db.user"
	PrefPassword = "db.password"
	PrefHost     = "db.host"
	PrefPort     = "db.port"
	PrefPath     = "db.path"
	PrefParams   = "db.params"

	PrefFirebirdEnabled         = "firebird.enabled"
	PrefPhoenixEnabled          = "phoenix.enabled"
	PrefPhoenixUser             = "phoenix.user"
	PrefPhoenixPassword         = "phoenix.password"
	PrefPhoenixHost             = "phoenix.host"
	PrefPhoenixPort             = "phoenix.port"
	PrefPhoenixInstance         = "phoenix.instance"
	PrefPhoenixDatabase         = "phoenix.database"
	PrefPhoenixParams           = "phoenix.params"
	PrefPhoenixControlHost      = "phoenix.control_center.host"
	PrefPhoenixOperatorID       = "phoenix.operator.id"
	PrefPhoenixOperatorName     = "phoenix.operator.name"
	PrefPhoenixOperatorPassword = "phoenix.operator.password"
	PrefPhoenixClientRole       = "phoenix.client.role"

	PrefBackendMode = "backend.mode"
	PrefCASLEnabled = "casl.enabled"
	PrefCASLBaseURL = "casl.base_url"
	PrefCASLToken   = "casl.token"
	PrefCASLEmail   = "casl.email"
	PrefCASLPass    = "casl.password"
	PrefCASLPultID  = "casl.pult_id"
	PrefLogLevel    = "log.level"
)

const (
	BackendModeFirebird  = "firebird"
	BackendModePhoenix   = "phoenix"
	BackendModeCASLCloud = "casl_cloud"
)

const (
	PhoenixClientRoleDuty          = "duty_operator"
	PhoenixClientRoleAdministrator = "administrator"
)

// NormalizePhoenixClientRole returns a supported Phoenix workstation role.
func NormalizePhoenixClientRole(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case PhoenixClientRoleAdministrator, "admin":
		return PhoenixClientRoleAdministrator
	default:
		return PhoenixClientRoleDuty
	}
}

// PhoenixLoginConfigured reports whether automatic Phoenix login has credentials.
func PhoenixLoginConfigured(cfg DBConfig) bool {
	return strings.TrimSpace(cfg.PhoenixControlHost) != "" &&
		cfg.PhoenixOperatorID > 0 &&
		cfg.PhoenixOperatorPassword != ""
}

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Path     string
	Params   string

	FirebirdEnabled         bool
	PhoenixEnabled          bool
	PhoenixUser             string
	PhoenixPassword         string
	PhoenixHost             string
	PhoenixPort             string
	PhoenixInstance         string
	PhoenixDatabase         string
	PhoenixParams           string
	PhoenixControlHost      string
	PhoenixOperatorID       int64
	PhoenixOperatorName     string
	PhoenixOperatorPassword string
	PhoenixClientRole       string

	CASLEnabled bool
	Mode        string
	CASLBaseURL string
	CASLToken   string
	CASLEmail   string
	CASLPass    string
	CASLPultID  int64
	LogLevel    string
}

func LoadDBConfig(p Preferences) DBConfig {
	log.Debug().Msg("Завантаження налаштувань БД з преференсів...")
	legacyMode := normalizeBackendMode(p.StringWithFallback(PrefBackendMode, BackendModeFirebird))
	caslEnabled := p.BoolWithFallback(PrefCASLEnabled, legacyMode == BackendModeCASLCloud)
	firebirdEnabled := p.BoolWithFallback(PrefFirebirdEnabled, legacyMode != BackendModePhoenix)
	phoenixEnabled := p.BoolWithFallback(PrefPhoenixEnabled, legacyMode == BackendModePhoenix)
	cfg := DBConfig{
		User:     stringWithTrimmedFallback(p, PrefUser, "SYSDBA"),
		Password: p.StringWithFallback(PrefPassword, "masterkey"),
		Host:     stringWithTrimmedFallback(p, PrefHost, "localhost"),
		Port:     stringWithTrimmedFallback(p, PrefPort, "3050"),
		Path:     stringWithTrimmedFallback(p, PrefPath, "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:   stringWithTrimmedFallback(p, PrefParams, "charset=WIN1251&auth_plugin_name=Srp"),

		FirebirdEnabled:         firebirdEnabled,
		PhoenixEnabled:          phoenixEnabled,
		PhoenixUser:             stringWithTrimmedFallback(p, PrefPhoenixUser, "sa"),
		PhoenixPassword:         p.StringWithFallback(PrefPhoenixPassword, ""),
		PhoenixHost:             stringWithTrimmedFallback(p, PrefPhoenixHost, "localhost"),
		PhoenixPort:             strings.TrimSpace(p.StringWithFallback(PrefPhoenixPort, "")),
		PhoenixInstance:         stringWithTrimmedFallback(p, PrefPhoenixInstance, "PHOENIX4"),
		PhoenixDatabase:         stringWithTrimmedFallback(p, PrefPhoenixDatabase, "Pult4DB"),
		PhoenixParams:           stringWithTrimmedFallback(p, PrefPhoenixParams, "encrypt=disable&trustservercertificate=true"),
		PhoenixControlHost:      strings.TrimSpace(p.StringWithFallback(PrefPhoenixControlHost, "")),
		PhoenixOperatorID:       int64(p.IntWithFallback(PrefPhoenixOperatorID, 0)),
		PhoenixOperatorName:     strings.TrimSpace(p.StringWithFallback(PrefPhoenixOperatorName, "")),
		PhoenixOperatorPassword: p.StringWithFallback(PrefPhoenixOperatorPassword, ""),
		PhoenixClientRole:       NormalizePhoenixClientRole(p.StringWithFallback(PrefPhoenixClientRole, PhoenixClientRoleDuty)),

		CASLEnabled: caslEnabled,
		Mode:        legacyMode,
		CASLBaseURL: p.StringWithFallback(PrefCASLBaseURL, "http://127.0.0.1:50003"),
		CASLToken:   p.StringWithFallback(PrefCASLToken, ""),
		CASLEmail:   p.StringWithFallback(PrefCASLEmail, ""),
		CASLPass:    p.StringWithFallback(PrefCASLPass, ""),
		CASLPultID:  int64(p.IntWithFallback(PrefCASLPultID, 0)),
		LogLevel:    applogger.NormalizeLogLevel(p.StringWithFallback(PrefLogLevel, "info")),
	}

	log.Debug().
		Str("user", cfg.User).
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("path", cfg.Path).
		Bool("firebirdEnabled", cfg.FirebirdEnabled).
		Bool("phoenixEnabled", cfg.PhoenixEnabled).
		Str("phoenixHost", cfg.PhoenixHost).
		Str("phoenixInstance", cfg.PhoenixInstance).
		Str("phoenixDatabase", cfg.PhoenixDatabase).
		Str("mode", cfg.Mode).
		Bool("caslEnabled", cfg.CASLEnabled).
		Msg("Налаштування БД завантажено")

	// Якщо жодного ключа ще немає в преференсах, записуємо дефолтні значення
	// Це гарантує, що конфіг з'явиться на диску одразу після першого запуску
	if p.String(PrefUser) == "" {
		log.Debug().Msg("Преференси порожні - записуємо дефолтні значення...")
		SaveDBConfig(p, cfg)
		log.Debug().Msg("Дефолтні налаштування записано")
	}

	return cfg
}

func SaveDBConfig(p Preferences, cfg DBConfig) {
	log.Debug().Msg("Збереження налаштувань БД...")
	p.SetString(PrefUser, strings.TrimSpace(cfg.User))
	p.SetString(PrefPassword, cfg.Password)
	p.SetString(PrefHost, strings.TrimSpace(cfg.Host))
	p.SetString(PrefPort, strings.TrimSpace(cfg.Port))
	p.SetString(PrefPath, strings.TrimSpace(cfg.Path))
	p.SetString(PrefParams, strings.TrimSpace(cfg.Params))
	p.SetBool(PrefFirebirdEnabled, cfg.FirebirdEnabled)
	p.SetBool(PrefPhoenixEnabled, cfg.PhoenixEnabled)
	p.SetString(PrefPhoenixUser, strings.TrimSpace(cfg.PhoenixUser))
	p.SetString(PrefPhoenixPassword, cfg.PhoenixPassword)
	p.SetString(PrefPhoenixHost, strings.TrimSpace(cfg.PhoenixHost))
	p.SetString(PrefPhoenixPort, strings.TrimSpace(cfg.PhoenixPort))
	p.SetString(PrefPhoenixInstance, strings.TrimSpace(cfg.PhoenixInstance))
	p.SetString(PrefPhoenixDatabase, strings.TrimSpace(cfg.PhoenixDatabase))
	p.SetString(PrefPhoenixParams, strings.TrimSpace(cfg.PhoenixParams))
	p.SetString(PrefPhoenixControlHost, strings.TrimSpace(cfg.PhoenixControlHost))
	p.SetInt(PrefPhoenixOperatorID, int(cfg.PhoenixOperatorID))
	p.SetString(PrefPhoenixOperatorName, strings.TrimSpace(cfg.PhoenixOperatorName))
	p.SetString(PrefPhoenixOperatorPassword, cfg.PhoenixOperatorPassword)
	p.SetString(PrefPhoenixClientRole, NormalizePhoenixClientRole(cfg.PhoenixClientRole))
	p.SetBool(PrefCASLEnabled, cfg.CASLEnabled)
	p.SetString(PrefBackendMode, normalizeBackendMode(cfg.Mode))
	p.SetString(PrefCASLBaseURL, cfg.CASLBaseURL)
	p.SetString(PrefCASLToken, cfg.CASLToken)
	p.SetString(PrefCASLEmail, cfg.CASLEmail)
	p.SetString(PrefCASLPass, cfg.CASLPass)
	p.SetInt(PrefCASLPultID, int(cfg.CASLPultID))
	p.SetString(PrefLogLevel, applogger.NormalizeLogLevel(cfg.LogLevel))
	log.Debug().Str("host", cfg.Host).Str("port", cfg.Port).Msg("Налаштування БД збережено")
}

func (c DBConfig) FirebirdDSN() string {
	// Format: user:password@host:port/path?params
	host := strings.TrimSpace(c.Host)
	if host == "" {
		host = "localhost"
	}
	connectHost := firebirdConnectHost(host)
	port := strings.TrimSpace(c.Port)
	if port == "" {
		port = "3050"
	}
	path := strings.TrimSpace(c.Path)
	if path == "" {
		path = "C:/MOST.PM/BASE/MOST5.FDB"
	}
	dsn := fmt.Sprintf("%s:%s@%s:%s/%s", strings.TrimSpace(c.User), c.Password, connectHost, port, path)
	if params := strings.TrimSpace(c.Params); params != "" {
		if !strings.HasPrefix(params, "?") {
			dsn += "?"
		}
		dsn += params
	}
	log.Debug().Str("host", connectHost).Str("configuredHost", host).Str("port", port).Str("path", path).Msg("Firebird DSN сформовано")
	return dsn
}

func firebirdConnectHost(host string) string {
	switch strings.ToLower(strings.TrimSpace(host)) {
	case "", "localhost":
		return "127.0.0.1"
	default:
		return strings.TrimSpace(host)
	}
}

func (c DBConfig) PhoenixDSN() string {
	host := strings.TrimSpace(c.PhoenixHost)
	if host == "" {
		host = "localhost"
	}

	authority := host
	if port := strings.TrimSpace(c.PhoenixPort); port != "" {
		authority = authority + ":" + port
	}

	u := &url.URL{
		Scheme: "sqlserver",
		Host:   authority,
	}
	if user := strings.TrimSpace(c.PhoenixUser); user != "" {
		if password := c.PhoenixPassword; password != "" {
			u.User = url.UserPassword(user, password)
		} else {
			u.User = url.User(user)
		}
	}

	q := u.Query()
	if database := strings.TrimSpace(c.PhoenixDatabase); database != "" {
		q.Set("database", database)
	}
	if instance := strings.TrimSpace(c.PhoenixInstance); instance != "" {
		q.Set("instance", instance)
	}
	applyURLQueryParams(q, c.PhoenixParams)
	if _, configured := q["dial timeout"]; !configured {
		q.Set("dial timeout", "5")
	}
	if _, configured := q["keepalive"]; !configured {
		q.Set("keepalive", "5")
	}
	u.RawQuery = q.Encode()

	dsn := u.String()
	log.Debug().
		Str("phoenixHost", host).
		Str("phoenixInstance", strings.TrimSpace(c.PhoenixInstance)).
		Str("phoenixDatabase", strings.TrimSpace(c.PhoenixDatabase)).
		Msg("Phoenix DSN сформовано")
	return dsn
}

func (c DBConfig) NormalizedMode() string {
	return normalizeBackendMode(c.Mode)
}

func normalizeBackendMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case BackendModePhoenix:
		return BackendModePhoenix
	case BackendModeCASLCloud:
		return BackendModeCASLCloud
	default:
		return BackendModeFirebird
	}
}

func applyURLQueryParams(values url.Values, raw string) {
	text := strings.TrimSpace(strings.TrimPrefix(raw, "?"))
	if text == "" {
		return
	}

	for _, part := range strings.Split(text, "&") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, found := strings.Cut(part, "=")
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if !found {
			values.Set(key, "")
			continue
		}
		values.Set(key, strings.TrimSpace(value))
	}
}

func stringWithTrimmedFallback(p Preferences, key string, fallback string) string {
	value := strings.TrimSpace(p.StringWithFallback(key, fallback))
	if value == "" {
		return fallback
	}
	return value
}
