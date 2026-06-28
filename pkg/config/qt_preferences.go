//go:build qt

package config

import (
	"runtime"
	"strings"

	qt "github.com/mappu/miqt/qt6"
)

const prefQtDanglingViewRepairV1 = "qt.preferences.dangling_view_repair_v1"

// QtPreferences adapts QSettings to the Preferences interface used by config.
type QtPreferences struct {
	settings *qt.QSettings
}

func NewQtPreferences(org, app string) *QtPreferences {
	preferences := &QtPreferences{settings: qt.NewQSettings7(org, app)}
	preferences.repairDanglingViewCorruption()
	return preferences
}

func NewQtPreferencesWithSettings(settings *qt.QSettings) *QtPreferences {
	preferences := &QtPreferences{settings: settings}
	preferences.repairDanglingViewCorruption()
	return preferences
}

func (p *QtPreferences) FileName() string {
	if p == nil || p.settings == nil {
		return ""
	}
	return p.settings.FileName()
}

func (p *QtPreferences) BoolWithFallback(key string, fallback bool) bool {
	if !p.contains(key) {
		return fallback
	}
	v := qt.NewQVariant8(fallback)
	k := qt.NewQAnyStringView3(key)
	res := p.settings.Value(*k, v)
	k.Delete()
	val := res.ToBool()

	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) FloatWithFallback(key string, fallback float64) float64 {
	if !p.contains(key) {
		return fallback
	}
	v := qt.NewQVariant9(fallback)
	k := qt.NewQAnyStringView3(key)
	res := p.settings.Value(*k, v)
	k.Delete()
	val := res.ToDouble()

	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) IntWithFallback(key string, fallback int) int {
	if !p.contains(key) {
		return fallback
	}
	v := qt.NewQVariant4(fallback)
	k := qt.NewQAnyStringView3(key)
	res := p.settings.Value(*k, v)
	k.Delete()
	val := res.ToInt()

	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) String(key string) string {
	return p.StringWithFallback(key, "")
}

func (p *QtPreferences) StringWithFallback(key string, fallback string) string {
	if !p.contains(key) {
		return fallback
	}
	v := qt.NewQVariant14(fallback)
	k := qt.NewQAnyStringView3(key)
	res := p.settings.Value(*k, v)
	k.Delete()
	val := res.ToString()

	runtime.KeepAlive(v)
	runtime.KeepAlive(res)
	return val
}

func (p *QtPreferences) SetBool(key string, value bool) {
	v := qt.NewQVariant8(value)
	k := qt.NewQAnyStringView3(key)
	p.settings.SetValue(*k, v)
	k.Delete()
	p.settings.Sync()

	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetFloat(key string, value float64) {
	v := qt.NewQVariant9(value)
	k := qt.NewQAnyStringView3(key)
	p.settings.SetValue(*k, v)
	k.Delete()
	p.settings.Sync()

	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetInt(key string, value int) {
	v := qt.NewQVariant4(value)
	k := qt.NewQAnyStringView3(key)
	p.settings.SetValue(*k, v)
	k.Delete()
	p.settings.Sync()

	runtime.KeepAlive(v)
}

func (p *QtPreferences) SetString(key string, value string) {
	v := qt.NewQVariant14(value)
	k := qt.NewQAnyStringView3(key)
	p.settings.SetValue(*k, v)
	k.Delete()
	p.settings.Sync()

	runtime.KeepAlive(v)
}

func (p *QtPreferences) contains(key string) bool {
	k := qt.NewQAnyStringView3(key)
	found := p.settings.Contains(*k)
	k.Delete()
	return found
}

func (p *QtPreferences) repairDanglingViewCorruption() {
	if p == nil || p.settings == nil || p.BoolWithFallback(prefQtDanglingViewRepairV1, false) {
		return
	}
	if strings.TrimSpace(p.StringWithFallback(PrefVodafonePhone, "")) == "" {
		for _, key := range p.settings.AllKeys() {
			candidate := strings.TrimSpace(key)
			if !isLegacyVodafonePhone(candidate) || p.StringWithFallback(candidate, "") != candidate {
				continue
			}
			p.SetString(PrefVodafonePhone, candidate)
			break
		}
	}
	p.SetBool(prefQtDanglingViewRepairV1, true)
}

func isLegacyVodafonePhone(value string) bool {
	if len(value) != 12 || !strings.HasPrefix(value, "380") {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	for _, prefix := range []string{"38050", "38066", "38075", "38095", "38099"} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
