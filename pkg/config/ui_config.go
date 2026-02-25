package config

import (
	"fyne.io/fyne/v2"
)

const (
	PrefFontSize        = "ui.font_size"
	PrefFontSizeObjects = "ui.font_size_objects"
	PrefFontSizeEvents  = "ui.font_size_events"
	PrefFontSizeAlarms  = "ui.font_size_alarms"
	PrefExportDir       = "ui.export_dir"
	PrefEventLogLimit   = "ui.event_log_limit"
	PrefObjectLogLimit  = "ui.object_log_limit"

	// MinFontSize - мінімальний дозволений розмір шрифту
	MinFontSize = 8.0
	// MaxFontSize - максимальний дозволений розмір шрифту
	MaxFontSize = 30.0
)

type UIConfig struct {
	FontSize        float32
	FontSizeObjects float32
	FontSizeEvents  float32
	FontSizeAlarms  float32
	ExportDir       string
	EventLogLimit   int
	ObjectLogLimit  int
}

// clampFontSize обмежує значення шрифту зверху MaxFontSize
func clampFontSize(v float32) float32 {
	if v < float32(MinFontSize) {
		return float32(MinFontSize)
	}
	if v > float32(MaxFontSize) {
		return float32(MaxFontSize)
	}

	return v
}

func clampEventLimit(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100000 {
		return 100000
	}
	return v
}

func LoadUIConfig(p fyne.Preferences) UIConfig {
	fontSize := clampFontSize(float32(p.FloatWithFallback(PrefFontSize, 13.0)))
	fontSizeObjects := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeObjects, 13.0)))
	fontSizeEvents := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeEvents, 12.0)))
	fontSizeAlarms := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeAlarms, 13.0)))

	return UIConfig{
		FontSize:        fontSize,
		FontSizeObjects: fontSizeObjects,
		FontSizeEvents:  fontSizeEvents,
		FontSizeAlarms:  fontSizeAlarms,
		ExportDir:       p.StringWithFallback(PrefExportDir, ""),
		EventLogLimit:   clampEventLimit(int(p.IntWithFallback(PrefEventLogLimit, 2000))),
		ObjectLogLimit:  clampEventLimit(int(p.IntWithFallback(PrefObjectLogLimit, 0))),
	}
}

func SaveUIConfig(p fyne.Preferences, cfg UIConfig) {
	p.SetFloat(PrefFontSize, float64(clampFontSize(cfg.FontSize)))
	p.SetFloat(PrefFontSizeObjects, float64(clampFontSize(cfg.FontSizeObjects)))
	p.SetFloat(PrefFontSizeEvents, float64(clampFontSize(cfg.FontSizeEvents)))
	p.SetFloat(PrefFontSizeAlarms, float64(clampFontSize(cfg.FontSizeAlarms)))
	p.SetString(PrefExportDir, cfg.ExportDir)
	p.SetInt(PrefEventLogLimit, clampEventLimit(cfg.EventLogLimit))
	p.SetInt(PrefObjectLogLimit, clampEventLimit(cfg.ObjectLogLimit))
}
