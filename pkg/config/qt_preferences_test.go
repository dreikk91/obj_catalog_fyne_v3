//go:build qt

package config

import (
	"path/filepath"
	"testing"

	qt "github.com/mappu/miqt/qt6"
)

func TestQtPreferencesVodafoneRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.ini")
	writerSettings := qt.NewQSettings4(path, qt.QSettings__IniFormat)
	writer := NewQtPreferencesWithSettings(writerSettings)
	want := VodafoneConfig{
		Phone:                "380501234567",
		AccessToken:          "access-token",
		TokenExpiry:          "2026-06-27T21:00:00Z",
		LoginMethod:          VodafoneLoginMethodPUK,
		PUK:                  "12345678",
		AutoResetEnabled:     false,
		AutoResetDailyLimit:  4,
		AutoResetWindowHours: 36,
	}
	SaveVodafoneConfig(writer, want)
	writerSettings.Delete()

	readerSettings := qt.NewQSettings4(path, qt.QSettings__IniFormat)
	defer readerSettings.Delete()
	got := LoadVodafoneConfig(NewQtPreferencesWithSettings(readerSettings))
	if got != want {
		t.Fatalf("LoadVodafoneConfig() = %#v, want %#v", got, want)
	}
}

func TestQtPreferencesRepairsLegacyVodafonePhone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.ini")
	settings := qt.NewQSettings4(path, qt.QSettings__IniFormat)
	raw := &QtPreferences{settings: settings}
	raw.SetString("380501234567", "380501234567")

	preferences := NewQtPreferencesWithSettings(settings)
	if got := preferences.String(PrefVodafonePhone); got != "380501234567" {
		t.Fatalf("repaired Vodafone phone = %q", got)
	}
	settings.Delete()
}
