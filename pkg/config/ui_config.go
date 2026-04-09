package config

import (
	"fyne.io/fyne/v2"
)

const (
	PrefFontSize               = "ui.font_size"
	PrefFontSizeObjects        = "ui.font_size_objects"
	PrefFontSizeEvents         = "ui.font_size_events"
	PrefFontSizeAlarms         = "ui.font_size_alarms"
	PrefExportDir              = "ui.export_dir"
	PrefEventLogLimit          = "ui.event_log_limit"
	PrefObjectLogLimit         = "ui.object_log_limit"
	PrefBridgeAlarmHistoryMode = "ui.bridge_alarm_history_mode"
	PrefEventProbeIntervalSec  = "ui.event_probe_interval_sec"
	PrefEventsReconcileSec     = "ui.events_reconcile_interval_sec"
	PrefAlarmsReconcileSec     = "ui.alarms_reconcile_interval_sec"
	PrefObjectsReconcileSec    = "ui.objects_reconcile_interval_sec"
	PrefFallbackRefreshSec     = "ui.fallback_refresh_interval_sec"
	PrefMaxProbeBackoffSec     = "ui.max_probe_backoff_interval_sec"

	// MinFontSize - мінімальний дозволений розмір шрифту
	MinFontSize = 8.0
	// MaxFontSize - максимальний дозволений розмір шрифту
	MaxFontSize = 30.0
)

const (
	BridgeAlarmHistoryModeActiveOnly = "active_only"
	BridgeAlarmHistoryModeLegacy     = "legacy_object_events"

	DefaultEventProbeIntervalSec = 5
	DefaultEventsReconcileSec    = 60
	DefaultAlarmsReconcileSec    = 30
	DefaultObjectsReconcileSec   = 180
	DefaultFallbackRefreshSec    = 15
	DefaultMaxProbeBackoffSec    = 60
)

type UIConfig struct {
	FontSize               float32
	FontSizeObjects        float32
	FontSizeEvents         float32
	FontSizeAlarms         float32
	ExportDir              string
	EventLogLimit          int
	ObjectLogLimit         int
	BridgeAlarmHistoryMode string
	EventProbeIntervalSec  int
	EventsReconcileSec     int
	AlarmsReconcileSec     int
	ObjectsReconcileSec    int
	FallbackRefreshSec     int
	MaxProbeBackoffSec     int
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

func clampSchedulerIntervalSec(v int, fallback int) int {
	if v <= 0 {
		return fallback
	}
	if v > 3600 {
		return 3600
	}
	return v
}

func NormalizeBridgeAlarmHistoryMode(v string) string {
	switch v {
	case BridgeAlarmHistoryModeLegacy:
		return BridgeAlarmHistoryModeLegacy
	default:
		return BridgeAlarmHistoryModeActiveOnly
	}
}

func (cfg UIConfig) NormalizedBridgeAlarmHistoryMode() string {
	return NormalizeBridgeAlarmHistoryMode(cfg.BridgeAlarmHistoryMode)
}

func LoadUIConfig(p fyne.Preferences) UIConfig {
	fontSize := clampFontSize(float32(p.FloatWithFallback(PrefFontSize, 13.0)))
	fontSizeObjects := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeObjects, 13.0)))
	fontSizeEvents := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeEvents, 12.0)))
	fontSizeAlarms := clampFontSize(float32(p.FloatWithFallback(PrefFontSizeAlarms, 13.0)))

	return UIConfig{
		FontSize:               fontSize,
		FontSizeObjects:        fontSizeObjects,
		FontSizeEvents:         fontSizeEvents,
		FontSizeAlarms:         fontSizeAlarms,
		ExportDir:              p.StringWithFallback(PrefExportDir, ""),
		EventLogLimit:          clampEventLimit(int(p.IntWithFallback(PrefEventLogLimit, 2000))),
		ObjectLogLimit:         clampEventLimit(int(p.IntWithFallback(PrefObjectLogLimit, 0))),
		BridgeAlarmHistoryMode: NormalizeBridgeAlarmHistoryMode(p.StringWithFallback(PrefBridgeAlarmHistoryMode, BridgeAlarmHistoryModeActiveOnly)),
		EventProbeIntervalSec:  clampSchedulerIntervalSec(int(p.IntWithFallback(PrefEventProbeIntervalSec, DefaultEventProbeIntervalSec)), DefaultEventProbeIntervalSec),
		EventsReconcileSec:     clampSchedulerIntervalSec(int(p.IntWithFallback(PrefEventsReconcileSec, DefaultEventsReconcileSec)), DefaultEventsReconcileSec),
		AlarmsReconcileSec:     clampSchedulerIntervalSec(int(p.IntWithFallback(PrefAlarmsReconcileSec, DefaultAlarmsReconcileSec)), DefaultAlarmsReconcileSec),
		ObjectsReconcileSec:    clampSchedulerIntervalSec(int(p.IntWithFallback(PrefObjectsReconcileSec, DefaultObjectsReconcileSec)), DefaultObjectsReconcileSec),
		FallbackRefreshSec:     clampSchedulerIntervalSec(int(p.IntWithFallback(PrefFallbackRefreshSec, DefaultFallbackRefreshSec)), DefaultFallbackRefreshSec),
		MaxProbeBackoffSec:     clampSchedulerIntervalSec(int(p.IntWithFallback(PrefMaxProbeBackoffSec, DefaultMaxProbeBackoffSec)), DefaultMaxProbeBackoffSec),
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
	p.SetString(PrefBridgeAlarmHistoryMode, NormalizeBridgeAlarmHistoryMode(cfg.BridgeAlarmHistoryMode))
	p.SetInt(PrefEventProbeIntervalSec, clampSchedulerIntervalSec(cfg.EventProbeIntervalSec, DefaultEventProbeIntervalSec))
	p.SetInt(PrefEventsReconcileSec, clampSchedulerIntervalSec(cfg.EventsReconcileSec, DefaultEventsReconcileSec))
	p.SetInt(PrefAlarmsReconcileSec, clampSchedulerIntervalSec(cfg.AlarmsReconcileSec, DefaultAlarmsReconcileSec))
	p.SetInt(PrefObjectsReconcileSec, clampSchedulerIntervalSec(cfg.ObjectsReconcileSec, DefaultObjectsReconcileSec))
	p.SetInt(PrefFallbackRefreshSec, clampSchedulerIntervalSec(cfg.FallbackRefreshSec, DefaultFallbackRefreshSec))
	p.SetInt(PrefMaxProbeBackoffSec, clampSchedulerIntervalSec(cfg.MaxProbeBackoffSec, DefaultMaxProbeBackoffSec))
}
