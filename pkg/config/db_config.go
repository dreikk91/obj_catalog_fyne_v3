package config

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"github.com/rs/zerolog/log"
)

const (
	PrefUser     = "db.user"
	PrefPassword = "db.password"
	PrefHost     = "db.host"
	PrefPort     = "db.port"
	PrefPath     = "db.path"
	PrefParams   = "db.params"

	PrefBackendMode = "backend.mode"
	PrefCASLEnabled = "casl.enabled"
	PrefCASLBaseURL = "casl.base_url"
	PrefCASLToken   = "casl.token"
	PrefCASLEmail   = "casl.email"
	PrefCASLPass    = "casl.password"
	PrefCASLPultID  = "casl.pult_id"
)

const (
	BackendModeFirebird  = "firebird"
	BackendModeCASLCloud = "casl_cloud"
)

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Path     string
	Params   string

	CASLEnabled bool
	Mode        string
	CASLBaseURL string
	CASLToken   string
	CASLEmail   string
	CASLPass    string
	CASLPultID  int64
}

func LoadDBConfig(p fyne.Preferences) DBConfig {
	log.Debug().Msg("Завантаження налаштувань БД з преференсів...")
	legacyMode := normalizeBackendMode(p.StringWithFallback(PrefBackendMode, BackendModeFirebird))
	caslEnabled := p.BoolWithFallback(PrefCASLEnabled, legacyMode == BackendModeCASLCloud)
	cfg := DBConfig{
		User:     p.StringWithFallback(PrefUser, "SYSDBA"),
		Password: p.StringWithFallback(PrefPassword, "masterkey"),
		Host:     p.StringWithFallback(PrefHost, "localhost"),
		Port:     p.StringWithFallback(PrefPort, "3050"),
		Path:     p.StringWithFallback(PrefPath, "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:   p.StringWithFallback(PrefParams, "charset=WIN1251&auth_plugin_name=Srp"),

		CASLEnabled: caslEnabled,
		Mode:        legacyMode,
		CASLBaseURL: p.StringWithFallback(PrefCASLBaseURL, "http://127.0.0.1:50003"),
		CASLToken:   p.StringWithFallback(PrefCASLToken, ""),
		CASLEmail:   p.StringWithFallback(PrefCASLEmail, ""),
		CASLPass:    p.StringWithFallback(PrefCASLPass, ""),
		CASLPultID:  int64(p.IntWithFallback(PrefCASLPultID, 0)),
	}

	log.Debug().
		Str("user", cfg.User).
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("path", cfg.Path).
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
	p.SetBool(PrefCASLEnabled, cfg.CASLEnabled)
	mode := normalizeBackendMode(cfg.Mode)
	if cfg.CASLEnabled {
		mode = BackendModeCASLCloud
	} else {
		mode = BackendModeFirebird
	}
	p.SetString(PrefBackendMode, mode)
	p.SetString(PrefCASLBaseURL, cfg.CASLBaseURL)
	p.SetString(PrefCASLToken, cfg.CASLToken)
	p.SetString(PrefCASLEmail, cfg.CASLEmail)
	p.SetString(PrefCASLPass, cfg.CASLPass)
	p.SetInt(PrefCASLPultID, int(cfg.CASLPultID))
	log.Debug().Str("host", cfg.Host).Str("port", cfg.Port).Msg("Налаштування БД збережено")
}

func (c DBConfig) ToDSN() string {
	// Format: user:password@host:port/path?params
	dsn := fmt.Sprintf("%s:%s@%s:%s/%s", c.User, c.Password, c.Host, c.Port, c.Path)
	if c.Params != "" {
		if !strings.HasPrefix(c.Params, "?") {
			dsn += "?"
		}
		dsn += c.Params
	}
	log.Debug().Str("dsn", dsn).Msg("DSN сформовано")
	return dsn
}

func (c DBConfig) NormalizedMode() string {
	return normalizeBackendMode(c.Mode)
}

func normalizeBackendMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case BackendModeCASLCloud:
		return BackendModeCASLCloud
	default:
		return BackendModeFirebird
	}
}
