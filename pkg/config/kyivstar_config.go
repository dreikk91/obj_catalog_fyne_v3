package config

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
)

const (
	PrefKyivstarClientID             = "kyivstar.client_id"
	PrefKyivstarClientSecret         = "kyivstar.client_secret"
	PrefKyivstarUserEmail            = "kyivstar.user_email"
	PrefKyivstarAccessToken          = "kyivstar.access_token"
	PrefKyivstarTokenExpiry          = "kyivstar.token_expiry"
	PrefKyivstarAutoResetEnabled     = "kyivstar.auto_reset.enabled"
	PrefKyivstarAutoResetDailyLimit  = "kyivstar.auto_reset.daily_limit"
	PrefKyivstarAutoResetWindowHours = "kyivstar.auto_reset.window_hours"
	PrefKyivstarAutoResetHistory     = "kyivstar.auto_reset.history"

	DefaultKyivstarAutoResetEnabled     = true
	DefaultKyivstarAutoResetDailyLimit  = 2
	DefaultKyivstarAutoResetWindowHours = 24
	MinKyivstarAutoResetWindowHours     = 1
)

// KyivstarConfig зберігає локальні параметри доступу до Kyivstar IoT API.
type KyivstarConfig struct {
	ClientID             string
	ClientSecret         string
	UserEmail            string
	AccessToken          string
	TokenExpiry          string
	AutoResetEnabled     bool
	AutoResetDailyLimit  int
	AutoResetWindowHours int
}

func LoadKyivstarConfig(p fyne.Preferences) KyivstarConfig {
	if p == nil {
		return KyivstarConfig{
			AutoResetEnabled:     DefaultKyivstarAutoResetEnabled,
			AutoResetDailyLimit:  DefaultKyivstarAutoResetDailyLimit,
			AutoResetWindowHours: DefaultKyivstarAutoResetWindowHours,
		}
	}
	return KyivstarConfig{
		ClientID:             strings.TrimSpace(p.StringWithFallback(PrefKyivstarClientID, "")),
		ClientSecret:         strings.TrimSpace(p.StringWithFallback(PrefKyivstarClientSecret, "")),
		UserEmail:            strings.TrimSpace(p.StringWithFallback(PrefKyivstarUserEmail, "")),
		AccessToken:          strings.TrimSpace(p.StringWithFallback(PrefKyivstarAccessToken, "")),
		TokenExpiry:          strings.TrimSpace(p.StringWithFallback(PrefKyivstarTokenExpiry, "")),
		AutoResetEnabled:     p.BoolWithFallback(PrefKyivstarAutoResetEnabled, DefaultKyivstarAutoResetEnabled),
		AutoResetDailyLimit:  clampKyivstarAutoResetLimit(p.IntWithFallback(PrefKyivstarAutoResetDailyLimit, DefaultKyivstarAutoResetDailyLimit)),
		AutoResetWindowHours: clampKyivstarAutoResetWindowHours(p.IntWithFallback(PrefKyivstarAutoResetWindowHours, DefaultKyivstarAutoResetWindowHours)),
	}
}

func SaveKyivstarConfig(p fyne.Preferences, cfg KyivstarConfig) {
	if p == nil {
		return
	}
	p.SetString(PrefKyivstarClientID, strings.TrimSpace(cfg.ClientID))
	p.SetString(PrefKyivstarClientSecret, strings.TrimSpace(cfg.ClientSecret))
	p.SetString(PrefKyivstarUserEmail, strings.TrimSpace(cfg.UserEmail))
	p.SetString(PrefKyivstarAccessToken, strings.TrimSpace(cfg.AccessToken))
	p.SetString(PrefKyivstarTokenExpiry, strings.TrimSpace(cfg.TokenExpiry))
	p.SetBool(PrefKyivstarAutoResetEnabled, cfg.AutoResetEnabled)
	p.SetInt(PrefKyivstarAutoResetDailyLimit, clampKyivstarAutoResetLimit(cfg.AutoResetDailyLimit))
	p.SetInt(PrefKyivstarAutoResetWindowHours, clampKyivstarAutoResetWindowHours(cfg.AutoResetWindowHours))
}

func (c KyivstarConfig) TokenExpiryTime() time.Time {
	return parseTokenExpiry(c.TokenExpiry)
}

func (c KyivstarConfig) HasCredentials() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != ""
}

func (c KyivstarConfig) HasAccessToken() bool {
	return strings.TrimSpace(c.AccessToken) != ""
}

func (c KyivstarConfig) TokenUsableAt(now time.Time) bool {
	return tokenUsableAt(c.AccessToken, c.TokenExpiry, now)
}

func clampKyivstarAutoResetLimit(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func clampKyivstarAutoResetWindowHours(v int) int {
	if v < MinKyivstarAutoResetWindowHours {
		return MinKyivstarAutoResetWindowHours
	}
	if v > 24*30 {
		return 24 * 30
	}
	return v
}

// KyivstarConfigStore абстрагує збереження локальних Kyivstar налаштувань.
type KyivstarConfigStore interface {
	LoadKyivstarConfig() KyivstarConfig
	SaveKyivstarConfig(cfg KyivstarConfig)
}

// PreferencesKyivstarConfigStore працює поверх Fyne Preferences.
type PreferencesKyivstarConfigStore struct {
	pref fyne.Preferences
}

func NewPreferencesKyivstarConfigStore(pref fyne.Preferences) *PreferencesKyivstarConfigStore {
	if pref == nil {
		return nil
	}
	return &PreferencesKyivstarConfigStore{pref: pref}
}

func (s *PreferencesKyivstarConfigStore) LoadKyivstarConfig() KyivstarConfig {
	if s == nil || s.pref == nil {
		return KyivstarConfig{}
	}
	return LoadKyivstarConfig(s.pref)
}

func (s *PreferencesKyivstarConfigStore) SaveKyivstarConfig(cfg KyivstarConfig) {
	if s == nil || s.pref == nil {
		return
	}
	SaveKyivstarConfig(s.pref, cfg)
}
