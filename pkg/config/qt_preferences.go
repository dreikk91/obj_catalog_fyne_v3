//go:build qt

package config

import (
	"runtime"

	qt "github.com/mappu/miqt/qt6"
)

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
	k := qt.NewQAnyStringView3(key)
	if !p.settings.Contains(*k) {
		runtime.KeepAlive(k)
		return fallback
	}
	v := qt.NewQVariant8(fallback)
	res := p.settings.Value(*k, v)
	val := res.ToBool()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) FloatWithFallback(key string, fallback float64) float64 {
	k := qt.NewQAnyStringView3(key)
	if !p.settings.Contains(*k) {
		runtime.KeepAlive(k)
		return fallback
	}
	v := qt.NewQVariant9(fallback)
	res := p.settings.Value(*k, v)
	val := res.ToDouble()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) IntWithFallback(key string, fallback int) int {
	k := qt.NewQAnyStringView3(key)
	if !p.settings.Contains(*k) {
		runtime.KeepAlive(k)
		return fallback
	}
	v := qt.NewQVariant4(fallback)
	res := p.settings.Value(*k, v)
	val := res.ToInt()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) String(key string) string {
	return p.StringWithFallback(key, "")
}

func (p *QtPreferences) StringWithFallback(key string, fallback string) string {
	k := qt.NewQAnyStringView3(key)
	if !p.settings.Contains(*k) {
		runtime.KeepAlive(k)
		return fallback
	}
	v := qt.NewQVariant14(fallback)
	res := p.settings.Value(*k, v)
	val := res.ToString()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) SetBool(key string, value bool) {
	k := qt.NewQAnyStringView3(key)
	v := qt.NewQVariant8(value)
	p.settings.SetValue(*k, v)
	p.settings.Sync()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetFloat(key string, value float64) {
	k := qt.NewQAnyStringView3(key)
	v := qt.NewQVariant9(value)
	p.settings.SetValue(*k, v)
	p.settings.Sync()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetInt(key string, value int) {
	k := qt.NewQAnyStringView3(key)
	v := qt.NewQVariant4(value)
	p.settings.SetValue(*k, v)
	p.settings.Sync()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetString(key string, value string) {
	k := qt.NewQAnyStringView3(key)
	v := qt.NewQVariant14(value)
	p.settings.SetValue(*k, v)
	p.settings.Sync()

	runtime.KeepAlive(k)
	runtime.KeepAlive(v)
}
