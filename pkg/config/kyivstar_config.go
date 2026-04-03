package config

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
)

const (
	PrefKyivstarClientID     = "kyivstar.client_id"
	PrefKyivstarClientSecret = "kyivstar.client_secret"
	PrefKyivstarUserEmail    = "kyivstar.user_email"
	PrefKyivstarAccessToken  = "kyivstar.access_token"
	PrefKyivstarTokenExpiry  = "kyivstar.token_expiry"
)

// KyivstarConfig зберігає локальні параметри доступу до Kyivstar IoT API.
type KyivstarConfig struct {
	ClientID     string
	ClientSecret string
	UserEmail    string
	AccessToken  string
	TokenExpiry  string
}

func LoadKyivstarConfig(p fyne.Preferences) KyivstarConfig {
	return KyivstarConfig{
		ClientID:     strings.TrimSpace(p.StringWithFallback(PrefKyivstarClientID, "")),
		ClientSecret: strings.TrimSpace(p.StringWithFallback(PrefKyivstarClientSecret, "")),
		UserEmail:    strings.TrimSpace(p.StringWithFallback(PrefKyivstarUserEmail, "")),
		AccessToken:  strings.TrimSpace(p.StringWithFallback(PrefKyivstarAccessToken, "")),
		TokenExpiry:  strings.TrimSpace(p.StringWithFallback(PrefKyivstarTokenExpiry, "")),
	}
}

func SaveKyivstarConfig(p fyne.Preferences, cfg KyivstarConfig) {
	p.SetString(PrefKyivstarClientID, strings.TrimSpace(cfg.ClientID))
	p.SetString(PrefKyivstarClientSecret, strings.TrimSpace(cfg.ClientSecret))
	p.SetString(PrefKyivstarUserEmail, strings.TrimSpace(cfg.UserEmail))
	p.SetString(PrefKyivstarAccessToken, strings.TrimSpace(cfg.AccessToken))
	p.SetString(PrefKyivstarTokenExpiry, strings.TrimSpace(cfg.TokenExpiry))
}

func (c KyivstarConfig) TokenExpiryTime() time.Time {
	if strings.TrimSpace(c.TokenExpiry) == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(c.TokenExpiry))
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c KyivstarConfig) HasCredentials() bool {
	return strings.TrimSpace(c.ClientID) != "" && strings.TrimSpace(c.ClientSecret) != ""
}

func (c KyivstarConfig) HasAccessToken() bool {
	return strings.TrimSpace(c.AccessToken) != ""
}

func (c KyivstarConfig) TokenUsableAt(now time.Time) bool {
	if !c.HasAccessToken() {
		return false
	}
	expiry := c.TokenExpiryTime()
	if expiry.IsZero() {
		return true
	}
	return expiry.After(now)
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
