package config

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
)

const (
	PrefVodafonePhone       = "vodafone.phone"
	PrefVodafoneAccessToken = "vodafone.access_token"
	PrefVodafoneTokenExpiry = "vodafone.token_expiry"
)

// VodafoneConfig зберігає локальні параметри авторизації Vodafone API.
type VodafoneConfig struct {
	Phone       string
	AccessToken string
	TokenExpiry string
}

func LoadVodafoneConfig(p fyne.Preferences) VodafoneConfig {
	return VodafoneConfig{
		Phone:       strings.TrimSpace(p.StringWithFallback(PrefVodafonePhone, "")),
		AccessToken: strings.TrimSpace(p.StringWithFallback(PrefVodafoneAccessToken, "")),
		TokenExpiry: strings.TrimSpace(p.StringWithFallback(PrefVodafoneTokenExpiry, "")),
	}
}

func SaveVodafoneConfig(p fyne.Preferences, cfg VodafoneConfig) {
	p.SetString(PrefVodafonePhone, strings.TrimSpace(cfg.Phone))
	p.SetString(PrefVodafoneAccessToken, strings.TrimSpace(cfg.AccessToken))
	p.SetString(PrefVodafoneTokenExpiry, strings.TrimSpace(cfg.TokenExpiry))
}

func (c VodafoneConfig) TokenExpiryTime() time.Time {
	return parseTokenExpiry(c.TokenExpiry)
}

func (c VodafoneConfig) HasAccessToken() bool {
	return strings.TrimSpace(c.AccessToken) != ""
}

func (c VodafoneConfig) TokenUsableAt(now time.Time) bool {
	return tokenUsableAt(c.AccessToken, c.TokenExpiry, now)
}

// VodafoneConfigStore абстрагує збереження локальних Vodafone налаштувань.
type VodafoneConfigStore interface {
	LoadVodafoneConfig() VodafoneConfig
	SaveVodafoneConfig(cfg VodafoneConfig)
}

// PreferencesVodafoneConfigStore працює поверх Fyne Preferences.
type PreferencesVodafoneConfigStore struct {
	pref fyne.Preferences
}

func NewPreferencesVodafoneConfigStore(pref fyne.Preferences) *PreferencesVodafoneConfigStore {
	if pref == nil {
		return nil
	}
	return &PreferencesVodafoneConfigStore{pref: pref}
}

func (s *PreferencesVodafoneConfigStore) LoadVodafoneConfig() VodafoneConfig {
	if s == nil || s.pref == nil {
		return VodafoneConfig{}
	}
	return LoadVodafoneConfig(s.pref)
}

func (s *PreferencesVodafoneConfigStore) SaveVodafoneConfig(cfg VodafoneConfig) {
	if s == nil || s.pref == nil {
		return
	}
	SaveVodafoneConfig(s.pref, cfg)
}
