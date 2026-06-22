package config

import (
	"obj_catalog_fyne_v3/pkg/ami"
)

const (
	PrefAMIEnabled   = "ami.enabled"
	PrefAMIHost      = "ami.host"
	PrefAMIPort      = "ami.port"
	PrefAMIUsername  = "ami.username"
	PrefAMISecret    = "ami.secret"
	PrefAMIExtension = "ami.extension"
	PrefAMIContext   = "ami.context"
)

func SaveAMIConfig(prefs Preferences, enabled bool, cfg ami.Config) {
	prefs.SetBool(PrefAMIEnabled, enabled)
	prefs.SetString(PrefAMIHost, cfg.Host)
	prefs.SetInt(PrefAMIPort, cfg.Port)
	prefs.SetString(PrefAMIUsername, cfg.Username)
	prefs.SetString(PrefAMISecret, cfg.Secret)
	prefs.SetString(PrefAMIExtension, cfg.Extension)
	prefs.SetString(PrefAMIContext, cfg.Context)
}

func LoadAMIConfig(prefs Preferences) (enabled bool, cfg ami.Config) {
	enabled = prefs.BoolWithFallback(PrefAMIEnabled, false)
	cfg = ami.Config{
		Host:      prefs.StringWithFallback(PrefAMIHost, "127.0.0.1"),
		Port:      prefs.IntWithFallback(PrefAMIPort, 5038),
		Username:  prefs.StringWithFallback(PrefAMIUsername, "admin"),
		Secret:    prefs.String(PrefAMISecret),
		Extension: prefs.StringWithFallback(PrefAMIExtension, "100"),
		Context:   prefs.StringWithFallback(PrefAMIContext, "from-internal"),
	}
	return
}
