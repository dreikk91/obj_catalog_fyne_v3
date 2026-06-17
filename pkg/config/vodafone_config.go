package config

import (
	"strings"
	"time"
)

const (
	PrefVodafonePhone                = "vodafone.phone"
	PrefVodafoneAccessToken          = "vodafone.access_token"
	PrefVodafoneTokenExpiry          = "vodafone.token_expiry"
	PrefVodafoneLoginMethod          = "vodafone.login_method"
	PrefVodafonePUK                  = "vodafone.puk"
	PrefVodafoneAutoResetEnabled     = "vodafone.auto_reset.enabled"
	PrefVodafoneAutoResetDailyLimit  = "vodafone.auto_reset.daily_limit"
	PrefVodafoneAutoResetWindowHours = "vodafone.auto_reset.window_hours"
	PrefVodafoneAutoResetHistory     = "vodafone.auto_reset.history"

	VodafoneLoginMethodSMS = "sms"
	VodafoneLoginMethodPUK = "puk"

	DefaultVodafoneAutoResetEnabled     = true
	DefaultVodafoneAutoResetDailyLimit  = 2
	DefaultVodafoneAutoResetWindowHours = 24
	MinVodafoneAutoResetWindowHours     = 1
)

// VodafoneConfig зберігає локальні параметри авторизації Vodafone API.
type VodafoneConfig struct {
	Phone                string
	AccessToken          string
	TokenExpiry          string
	LoginMethod          string
	PUK                  string
	AutoResetEnabled     bool
	AutoResetDailyLimit  int
	AutoResetWindowHours int
}

func LoadVodafoneConfig(p Preferences) VodafoneConfig {
	if p == nil {
		return VodafoneConfig{
			AutoResetEnabled:     DefaultVodafoneAutoResetEnabled,
			AutoResetDailyLimit:  DefaultVodafoneAutoResetDailyLimit,
			AutoResetWindowHours: DefaultVodafoneAutoResetWindowHours,
			LoginMethod:          VodafoneLoginMethodSMS,
		}
	}
	cfg := VodafoneConfig{
		Phone:                strings.TrimSpace(p.StringWithFallback(PrefVodafonePhone, "")),
		AccessToken:          strings.TrimSpace(p.StringWithFallback(PrefVodafoneAccessToken, "")),
		TokenExpiry:          strings.TrimSpace(p.StringWithFallback(PrefVodafoneTokenExpiry, "")),
		LoginMethod:          normalizeVodafoneLoginMethod(p.StringWithFallback(PrefVodafoneLoginMethod, VodafoneLoginMethodSMS)),
		PUK:                  strings.TrimSpace(p.StringWithFallback(PrefVodafonePUK, "")),
		AutoResetEnabled:     p.BoolWithFallback(PrefVodafoneAutoResetEnabled, DefaultVodafoneAutoResetEnabled),
		AutoResetDailyLimit:  clampVodafoneAutoResetLimit(p.IntWithFallback(PrefVodafoneAutoResetDailyLimit, DefaultVodafoneAutoResetDailyLimit)),
		AutoResetWindowHours: clampVodafoneAutoResetWindowHours(p.IntWithFallback(PrefVodafoneAutoResetWindowHours, DefaultVodafoneAutoResetWindowHours)),
	}
	return cfg
}

func SaveVodafoneConfig(p Preferences, cfg VodafoneConfig) {
	if p == nil {
		return
	}
	p.SetString(PrefVodafonePhone, strings.TrimSpace(cfg.Phone))
	p.SetString(PrefVodafoneAccessToken, strings.TrimSpace(cfg.AccessToken))
	p.SetString(PrefVodafoneTokenExpiry, strings.TrimSpace(cfg.TokenExpiry))
	p.SetString(PrefVodafoneLoginMethod, normalizeVodafoneLoginMethod(cfg.LoginMethod))
	p.SetString(PrefVodafonePUK, strings.TrimSpace(cfg.PUK))
	p.SetBool(PrefVodafoneAutoResetEnabled, cfg.AutoResetEnabled)
	p.SetInt(PrefVodafoneAutoResetDailyLimit, clampVodafoneAutoResetLimit(cfg.AutoResetDailyLimit))
	p.SetInt(PrefVodafoneAutoResetWindowHours, clampVodafoneAutoResetWindowHours(cfg.AutoResetWindowHours))
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

func (c VodafoneConfig) NormalizedLoginMethod() string {
	return normalizeVodafoneLoginMethod(c.LoginMethod)
}

func (c VodafoneConfig) HasPUKCredentials() bool {
	return strings.TrimSpace(c.Phone) != "" && strings.TrimSpace(c.PUK) != ""
}

func normalizeVodafoneLoginMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case VodafoneLoginMethodPUK:
		return VodafoneLoginMethodPUK
	default:
		return VodafoneLoginMethodSMS
	}
}

func clampVodafoneAutoResetLimit(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func clampVodafoneAutoResetWindowHours(v int) int {
	if v < MinVodafoneAutoResetWindowHours {
		return MinVodafoneAutoResetWindowHours
	}
	if v > 24*30 {
		return 24 * 30
	}
	return v
}

// VodafoneConfigStore абстрагує збереження локальних Vodafone налаштувань.
type VodafoneConfigStore interface {
	LoadVodafoneConfig() VodafoneConfig
	SaveVodafoneConfig(cfg VodafoneConfig)
}

// PreferencesVodafoneConfigStore працює поверх Fyne Preferences.
type PreferencesVodafoneConfigStore struct {
	pref Preferences
}

func NewPreferencesVodafoneConfigStore(pref Preferences) *PreferencesVodafoneConfigStore {
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
