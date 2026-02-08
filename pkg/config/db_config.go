package config

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type Preferences interface {
	String(key string) string
	StringWithFallback(key, fallback string) string
	SetString(key, value string)
	Float(key string) float64
	FloatWithFallback(key string, fallback float64) float64
	SetFloat(key string, value float64)
}

const (
	PrefUser     = "db.user"
	PrefPassword = "db.password"
	PrefHost     = "db.host"
	PrefPort     = "db.port"
	PrefPath     = "db.path"
	PrefParams   = "db.params"
)

type DBConfig struct {
	User     string
	Password string
	Host     string
	Port     string
	Path     string
	Params   string
}

func LoadDBConfig(p Preferences) DBConfig {
	log.Debug().Msg("Завантаження налаштувань БД з преференсів...")
	cfg := DBConfig{
		User:     p.StringWithFallback(PrefUser, "SYSDBA"),
		Password: p.StringWithFallback(PrefPassword, "masterkey"),
		Host:     p.StringWithFallback(PrefHost, "localhost"),
		Port:     p.StringWithFallback(PrefPort, "3050"),
		Path:     p.StringWithFallback(PrefPath, "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:   p.StringWithFallback(PrefParams, "charset=WIN1251&auth_plugin_name=Srp"),
	}

	log.Debug().
		Str("user", cfg.User).
		Str("host", cfg.Host).
		Str("port", cfg.Port).
		Str("path", cfg.Path).
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
	p.SetString(PrefUser, cfg.User)
	p.SetString(PrefPassword, cfg.Password)
	p.SetString(PrefHost, cfg.Host)
	p.SetString(PrefPort, cfg.Port)
	p.SetString(PrefPath, cfg.Path)
	p.SetString(PrefParams, cfg.Params)
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
