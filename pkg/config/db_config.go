package config

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
)

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

func LoadDBConfig(p fyne.Preferences) DBConfig {
	cfg := DBConfig{
		User:     p.StringWithFallback(PrefUser, "SYSDBA"),
		Password: p.StringWithFallback(PrefPassword, "masterkey"),
		Host:     p.StringWithFallback(PrefHost, "localhost"),
		Port:     p.StringWithFallback(PrefPort, "3050"),
		Path:     p.StringWithFallback(PrefPath, "C:/MOST.PM/BASE/MOST5.FDB"),
		Params:   p.StringWithFallback(PrefParams, "charset=WIN1251&auth_plugin_name=Srp"),
	}

	// Якщо жодного ключа ще немає в преференсах, записуємо дефолтні значення
	// Це гарантує, що конфіг з'явиться на диску одразу після першого запуску
	if p.String(PrefUser) == "" {
		SaveDBConfig(p, cfg)
	}

	return cfg
}

func SaveDBConfig(p fyne.Preferences, cfg DBConfig) {
	p.SetString(PrefUser, cfg.User)
	p.SetString(PrefPassword, cfg.Password)
	p.SetString(PrefHost, cfg.Host)
	p.SetString(PrefPort, cfg.Port)
	p.SetString(PrefPath, cfg.Path)
	p.SetString(PrefParams, cfg.Params)
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
	return dsn
}
