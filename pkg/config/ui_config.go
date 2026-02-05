package config

import (
	"fyne.io/fyne/v2"
)

const (
	PrefFontSize        = "ui.font_size"
	PrefFontSizeObjects = "ui.font_size_objects"
	PrefFontSizeEvents  = "ui.font_size_events"
	PrefFontSizeAlarms  = "ui.font_size_alarms"
)

type UIConfig struct {
	FontSize        float32
	FontSizeObjects float32
	FontSizeEvents  float32
	FontSizeAlarms  float32
}

func LoadUIConfig(p fyne.Preferences) UIConfig {
	return UIConfig{
		FontSize:        float32(p.FloatWithFallback(PrefFontSize, 13.0)),
		FontSizeObjects: float32(p.FloatWithFallback(PrefFontSizeObjects, 13.0)),
		FontSizeEvents:  float32(p.FloatWithFallback(PrefFontSizeEvents, 12.0)),
		FontSizeAlarms:  float32(p.FloatWithFallback(PrefFontSizeAlarms, 13.0)),
	}
}

func SaveUIConfig(p fyne.Preferences, cfg UIConfig) {
	p.SetFloat(PrefFontSize, float64(cfg.FontSize))
	p.SetFloat(PrefFontSizeObjects, float64(cfg.FontSizeObjects))
	p.SetFloat(PrefFontSizeEvents, float64(cfg.FontSizeEvents))
	p.SetFloat(PrefFontSizeAlarms, float64(cfg.FontSizeAlarms))
}
