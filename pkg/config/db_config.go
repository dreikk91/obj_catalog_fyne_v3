package config

import (
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
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

	PrefFirebirdEnabled = "firebird.enabled"
	PrefPhoenixEnabled  = "phoenix.enabled"
	PrefPhoenixUser     = "phoenix.user"
	PrefPhoenixPassword = "phoenix.password"
	PrefPhoenixHost     = "phoenix.host"
	PrefPhoenixPort     = "phoenix.port"
	PrefPhoenixInstance = "phoenix.instance"
	PrefPhoenixDatabase = "phoenix.database"
	PrefPhoenixParams   = "phoenix.params"

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

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Path     string
	Params   string

	FirebirdEnabled bool
	PhoenixEnabled  bool
	PhoenixUser     string
	PhoenixPassword string
	PhoenixHost     string
	PhoenixPort     string
	PhoenixInstance string
	PhoenixDatabase string
	PhoenixParams   string

	CASLEnabled bool
	Mode        string
	CASLBaseURL string
	CASLToken   string
	CASLEmail   string
	CASLPass    string
	CASLPultID  int64
	LogLevel    string
}

func LoadDBConfig(p fyne.Preferences) DBConfig {
	log.Debug().Msg("Завантаження налаштувань БД з преференсів...")
	legacyMode := normalizeBackendMode(p.StringWithFallback(PrefBackendMode, BackendModeFirebird))
	caslEnabled := p.BoolWithFallback(PrefCASLEnabled, legacyMode == BackendModeCASLCloud)
	firebirdEnabled := p.BoolWithFallback(PrefFirebirdEnabled, legacyMode != BackendModePhoenix)
	phoenixEnabled := p.BoolWithFallback(PrefPhoenixEnabled, legacyMode == BackendModePhoenix)
	cfg := DBConfig{
		User:     p.StringWithFallback(PrefUser, "SYSDBA"),
		Password: p.StringWithFallback(PrefPassword, "masterkey"),
		Host:     p.StringWithFallback(PrefHost, "localhost"),
		Port:     p.StringWithFallback(PrefPort, "3050"),
		Path:     p.StringWithFallback(PrefPath, "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:   p.StringWithFallback(PrefParams, "charset=WIN1251&auth_plugin_name=Srp"),

		FirebirdEnabled: firebirdEnabled,
		PhoenixEnabled:  phoenixEnabled,
		PhoenixUser:     p.StringWithFallback(PrefPhoenixUser, "sa"),
		PhoenixPassword: p.StringWithFallback(PrefPhoenixPassword, ""),
		PhoenixHost:     p.StringWithFallback(PrefPhoenixHost, "localhost"),
		PhoenixPort:     p.StringWithFallback(PrefPhoenixPort, ""),
		PhoenixInstance: p.StringWithFallback(PrefPhoenixInstance, "PHOENIX4"),
		PhoenixDatabase: p.StringWithFallback(PrefPhoenixDatabase, "Pult4DB"),
		PhoenixParams:   p.StringWithFallback(PrefPhoenixParams, "encrypt=disable&trustservercertificate=true"),

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

func SaveDBConfig(p fyne.Preferences, cfg DBConfig) {
	log.Debug().Msg("Збереження налаштувань БД...")
	p.SetString(PrefUser, cfg.User)
	p.SetString(PrefPassword, cfg.Password)
	p.SetString(PrefHost, cfg.Host)
	p.SetString(PrefPort, cfg.Port)
	p.SetString(PrefPath, cfg.Path)
	p.SetString(PrefParams, cfg.Params)
	p.SetBool(PrefFirebirdEnabled, cfg.FirebirdEnabled)
	p.SetBool(PrefPhoenixEnabled, cfg.PhoenixEnabled)
	p.SetString(PrefPhoenixUser, cfg.PhoenixUser)
	p.SetString(PrefPhoenixPassword, cfg.PhoenixPassword)
	p.SetString(PrefPhoenixHost, cfg.PhoenixHost)
	p.SetString(PrefPhoenixPort, cfg.PhoenixPort)
	p.SetString(PrefPhoenixInstance, cfg.PhoenixInstance)
	p.SetString(PrefPhoenixDatabase, cfg.PhoenixDatabase)
	p.SetString(PrefPhoenixParams, cfg.PhoenixParams)
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
	dsn := fmt.Sprintf("%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.Path)
	if c.Params != "" {
		if !strings.HasPrefix(c.Params, "?") {
			dsn += "?"
		}
		dsn += c.Params
	}
	log.Debug().Str("host", c.Host).Str("port", c.Port).Str("path", c.Path).Msg("Firebird DSN сформовано")
	return dsn
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
