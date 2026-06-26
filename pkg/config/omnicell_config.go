package config

import "strings"

const (
	PrefOmnicellEnabled                  = "omnicell.enabled"
	PrefOmnicellEndpoint                 = "omnicell.endpoint"
	PrefOmnicellLogin                    = "omnicell.login"
	PrefOmnicellPassword                 = "omnicell.password"
	PrefOmnicellSource                   = "omnicell.source"
	PrefOmnicellMCAPrimaryAPN            = "omnicell.mca.primary_apn"
	PrefOmnicellMCAReserveAPN            = "omnicell.mca.reserve_apn"
	PrefOmnicellMCAPrimaryIP             = "omnicell.mca.primary_ip"
	PrefOmnicellMCAReserveIP             = "omnicell.mca.reserve_ip"
	PrefOmnicellMCAPrimaryModulePort     = "omnicell.mca.primary_module_port"
	PrefOmnicellMCAReserveModulePort     = "omnicell.mca.reserve_module_port"
	PrefOmnicellMCAPrimaryReceiverPort   = "omnicell.mca.primary_receiver_port"
	PrefOmnicellMCAReserveReceiverPort   = "omnicell.mca.reserve_receiver_port"
	PrefOmnicellMCAPrimaryTestInterval   = "omnicell.mca.primary_test_interval"
	PrefOmnicellMCAReserveTestInterval   = "omnicell.mca.reserve_test_interval"
	PrefOmnicellMCAInput1ConfirmMode     = "omnicell.mca.input1_confirm_mode"
	PrefOmnicellMCADefaultMessageProfile = "omnicell.mca.default_message_profile"
)

const DefaultOmnicellEndpoint = "https://api.omnicell.com.ua/ip2sms/"

type OmnicellConfig struct {
	Enabled                  bool
	Endpoint                 string
	Login                    string
	Password                 string
	Source                   string
	MCAPrimaryAPN            string
	MCAReserveAPN            string
	MCAPrimaryIP             string
	MCAReserveIP             string
	MCAPrimaryModulePort     int
	MCAReserveModulePort     int
	MCAPrimaryReceiverPort   int
	MCAReserveReceiverPort   int
	MCAPrimaryTestInterval   int
	MCAReserveTestInterval   int
	MCAInput1ConfirmMode     bool
	MCADefaultMessageProfile string
}

func LoadOmnicellConfig(p Preferences) OmnicellConfig {
	if p == nil {
		return defaultOmnicellConfig()
	}
	defaults := defaultOmnicellConfig()
	return OmnicellConfig{
		Enabled:                  p.BoolWithFallback(PrefOmnicellEnabled, false),
		Endpoint:                 stringWithTrimmedFallback(p, PrefOmnicellEndpoint, defaults.Endpoint),
		Login:                    strings.TrimSpace(p.StringWithFallback(PrefOmnicellLogin, "")),
		Password:                 p.StringWithFallback(PrefOmnicellPassword, ""),
		Source:                   strings.TrimSpace(p.StringWithFallback(PrefOmnicellSource, "")),
		MCAPrimaryAPN:            stringWithTrimmedFallback(p, PrefOmnicellMCAPrimaryAPN, defaults.MCAPrimaryAPN),
		MCAReserveAPN:            stringWithTrimmedFallback(p, PrefOmnicellMCAReserveAPN, defaults.MCAReserveAPN),
		MCAPrimaryIP:             stringWithTrimmedFallback(p, PrefOmnicellMCAPrimaryIP, defaults.MCAPrimaryIP),
		MCAReserveIP:             stringWithTrimmedFallback(p, PrefOmnicellMCAReserveIP, defaults.MCAReserveIP),
		MCAPrimaryModulePort:     p.IntWithFallback(PrefOmnicellMCAPrimaryModulePort, defaults.MCAPrimaryModulePort),
		MCAReserveModulePort:     p.IntWithFallback(PrefOmnicellMCAReserveModulePort, defaults.MCAReserveModulePort),
		MCAPrimaryReceiverPort:   p.IntWithFallback(PrefOmnicellMCAPrimaryReceiverPort, defaults.MCAPrimaryReceiverPort),
		MCAReserveReceiverPort:   p.IntWithFallback(PrefOmnicellMCAReserveReceiverPort, defaults.MCAReserveReceiverPort),
		MCAPrimaryTestInterval:   p.IntWithFallback(PrefOmnicellMCAPrimaryTestInterval, defaults.MCAPrimaryTestInterval),
		MCAReserveTestInterval:   p.IntWithFallback(PrefOmnicellMCAReserveTestInterval, defaults.MCAReserveTestInterval),
		MCAInput1ConfirmMode:     p.BoolWithFallback(PrefOmnicellMCAInput1ConfirmMode, false),
		MCADefaultMessageProfile: stringWithTrimmedFallback(p, PrefOmnicellMCADefaultMessageProfile, defaults.MCADefaultMessageProfile),
	}
}

func SaveOmnicellConfig(p Preferences, cfg OmnicellConfig) {
	if p == nil {
		return
	}
	p.SetBool(PrefOmnicellEnabled, cfg.Enabled)
	p.SetString(PrefOmnicellEndpoint, strings.TrimSpace(cfg.Endpoint))
	p.SetString(PrefOmnicellLogin, strings.TrimSpace(cfg.Login))
	p.SetString(PrefOmnicellPassword, cfg.Password)
	p.SetString(PrefOmnicellSource, strings.TrimSpace(cfg.Source))
	p.SetString(PrefOmnicellMCAPrimaryAPN, strings.TrimSpace(cfg.MCAPrimaryAPN))
	p.SetString(PrefOmnicellMCAReserveAPN, strings.TrimSpace(cfg.MCAReserveAPN))
	p.SetString(PrefOmnicellMCAPrimaryIP, strings.TrimSpace(cfg.MCAPrimaryIP))
	p.SetString(PrefOmnicellMCAReserveIP, strings.TrimSpace(cfg.MCAReserveIP))
	p.SetInt(PrefOmnicellMCAPrimaryModulePort, cfg.MCAPrimaryModulePort)
	p.SetInt(PrefOmnicellMCAReserveModulePort, cfg.MCAReserveModulePort)
	p.SetInt(PrefOmnicellMCAPrimaryReceiverPort, cfg.MCAPrimaryReceiverPort)
	p.SetInt(PrefOmnicellMCAReserveReceiverPort, cfg.MCAReserveReceiverPort)
	p.SetInt(PrefOmnicellMCAPrimaryTestInterval, cfg.MCAPrimaryTestInterval)
	p.SetInt(PrefOmnicellMCAReserveTestInterval, cfg.MCAReserveTestInterval)
	p.SetBool(PrefOmnicellMCAInput1ConfirmMode, cfg.MCAInput1ConfirmMode)
	p.SetString(PrefOmnicellMCADefaultMessageProfile, strings.TrimSpace(cfg.MCADefaultMessageProfile))
}

func (cfg OmnicellConfig) Ready() bool {
	return cfg.Enabled &&
		strings.TrimSpace(cfg.Endpoint) != "" &&
		strings.TrimSpace(cfg.Login) != "" &&
		strings.TrimSpace(cfg.Password) != "" &&
		strings.TrimSpace(cfg.Source) != ""
}

func defaultOmnicellConfig() OmnicellConfig {
	return OmnicellConfig{
		Endpoint:                 DefaultOmnicellEndpoint,
		MCAPrimaryAPN:            "internet",
		MCAReserveAPN:            "internet",
		MCAPrimaryIP:             "091.196.053.147",
		MCAReserveIP:             "094.153.183.241",
		MCAPrimaryModulePort:     3312,
		MCAReserveModulePort:     3312,
		MCAPrimaryReceiverPort:   3311,
		MCAReserveReceiverPort:   3311,
		MCAPrimaryTestInterval:   1,
		MCAReserveTestInterval:   4,
		MCADefaultMessageProfile: "МЦА-GSM4",
	}
}
