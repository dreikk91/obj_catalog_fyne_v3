//go:build qt

package config

import qt "github.com/mappu/miqt/qt6"

// QtPreferences adapts QSettings to the Preferences interface used by config.
type QtPreferences struct {
	settings *qt.QSettings
}

func NewQtPreferences(org, app string) *QtPreferences {
	return &QtPreferences{settings: qt.NewQSettings7(org, app)}
}

func NewQtPreferencesWithSettings(settings *qt.QSettings) *QtPreferences {
	return &QtPreferences{settings: settings}
}

func (p *QtPreferences) FileName() string {
	if p == nil || p.settings == nil {
		return ""
	}
	return p.settings.FileName()
}

func (p *QtPreferences) BoolWithFallback(key string, fallback bool) bool {
	if !p.settings.Contains(qkey(key)) {
		return fallback
	}
	return p.settings.Value(qkey(key), qt.NewQVariant8(fallback)).ToBool()
}

func (p *QtPreferences) FloatWithFallback(key string, fallback float64) float64 {
	if !p.settings.Contains(qkey(key)) {
		return fallback
	}
	return p.settings.Value(qkey(key), qt.NewQVariant9(fallback)).ToDouble()
}

func (p *QtPreferences) IntWithFallback(key string, fallback int) int {
	if !p.settings.Contains(qkey(key)) {
		return fallback
	}
	return p.settings.Value(qkey(key), qt.NewQVariant4(fallback)).ToInt()
}

func (p *QtPreferences) String(key string) string {
	return p.StringWithFallback(key, "")
}

func (p *QtPreferences) StringWithFallback(key string, fallback string) string {
	if !p.settings.Contains(qkey(key)) {
		return fallback
	}
	return p.settings.Value(qkey(key), qt.NewQVariant14(fallback)).ToString()
}

func (p *QtPreferences) SetBool(key string, value bool) {
	p.settings.SetValue(qkey(key), qt.NewQVariant8(value))
	p.settings.Sync()
}

func (p *QtPreferences) SetFloat(key string, value float64) {
	p.settings.SetValue(qkey(key), qt.NewQVariant9(value))
	p.settings.Sync()
}

func (p *QtPreferences) SetInt(key string, value int) {
	p.settings.SetValue(qkey(key), qt.NewQVariant4(value))
	p.settings.Sync()
}

func (p *QtPreferences) SetString(key string, value string) {
	p.settings.SetValue(qkey(key), qt.NewQVariant14(value))
	p.settings.Sync()
}

func qkey(key string) qt.QAnyStringView {
	return *qt.NewQAnyStringView3(key)
}
