package simcommands

import (
	"fmt"
	"strconv"
	"strings"
)

type MCAGSM4Config struct {
	ObjectNumber        int
	HiddenNumber        int
	PrimaryAPN          string
	ReserveAPN          string
	PrimaryIP           string
	ReserveIP           string
	PrimaryModulePort   int
	ReserveModulePort   int
	PrimaryReceiverPort int
	ReserveReceiverPort int
	PrimaryTestInterval int
	ReserveTestInterval int
	Input1ConfirmMode   bool
}

type SMSCommand struct {
	Title string
	Text  string
}

const (
	ProfileMCAGSM4 = "МЦА-GSM4"
	ProfileMCAGSM  = "МЦА-GSM"
	ProfileFreeSMS = "Вільний текст"
)

func DefaultMCAGSM4Config() MCAGSM4Config {
	return MCAGSM4Config{
		PrimaryAPN:          "internet",
		ReserveAPN:          "internet",
		PrimaryIP:           "091.196.053.147",
		ReserveIP:           "094.153.183.241",
		PrimaryModulePort:   3312,
		ReserveModulePort:   3312,
		PrimaryReceiverPort: 3311,
		ReserveReceiverPort: 3311,
		PrimaryTestInterval: 1,
		ReserveTestInterval: 4,
	}
}

func DefaultMCAGSMConfig() MCAGSM4Config {
	cfg := DefaultMCAGSM4Config()
	cfg.PrimaryModulePort = 3315
	cfg.ReserveModulePort = 3315
	cfg.PrimaryReceiverPort = 3311
	cfg.ReserveReceiverPort = 3311
	cfg.PrimaryTestInterval = 240
	cfg.ReserveTestInterval = 240
	return cfg
}

func BuildMCAGSMMessages(cfg MCAGSM4Config) ([]SMSCommand, error) {
	cfg = normalizeMCAGSMConfig(cfg)
	if err := validateMCAGSMConfig(cfg); err != nil {
		return nil, err
	}

	msg1 := fmt.Sprintf(
		"&&1&2&1&%s&%s&%s&%04d&%04d&%03d&0&0&0&",
		formatModuleNumber(cfg.ObjectNumber),
		strings.TrimSpace(cfg.PrimaryAPN),
		formatIPv4(cfg.PrimaryIP),
		cfg.PrimaryModulePort,
		cfg.PrimaryReceiverPort,
		cfg.PrimaryTestInterval,
	)
	msg2 := fmt.Sprintf(
		"&&1&2&2&%s&%s&%s&%04d&%04d&%03d&0&0&0&",
		formatModuleNumber(cfg.HiddenNumber),
		strings.TrimSpace(cfg.ReserveAPN),
		formatIPv4(cfg.ReserveIP),
		cfg.ReserveModulePort,
		cfg.ReserveReceiverPort,
		cfg.ReserveTestInterval,
	)

	return []SMSCommand{
		{Title: ProfileMCAGSM + " SMS №1", Text: msg1},
		{Title: ProfileMCAGSM + " SMS №2", Text: msg2},
	}, nil
}

func BuildMCAGSM4Messages(cfg MCAGSM4Config) ([]SMSCommand, error) {
	cfg = normalizeMCAGSM4Config(cfg)
	if err := validateMCAGSM4Config(cfg); err != nil {
		return nil, err
	}

	msg1 := fmt.Sprintf(
		"&&1&2&0&1&%s&%s&%s&%04d&%04d&%03d&0&0&0&",
		formatModuleNumber(cfg.ObjectNumber),
		strings.TrimSpace(cfg.PrimaryAPN),
		formatIPv4(cfg.PrimaryIP),
		cfg.PrimaryModulePort,
		cfg.PrimaryReceiverPort,
		cfg.PrimaryTestInterval,
	)
	msg2 := fmt.Sprintf(
		"&&1&2&0&2&%s&%s&%s&%04d&%04d&%03d&0&0&0&",
		formatModuleNumber(cfg.HiddenNumber),
		strings.TrimSpace(cfg.ReserveAPN),
		formatIPv4(cfg.ReserveIP),
		cfg.ReserveModulePort,
		cfg.ReserveReceiverPort,
		cfg.ReserveTestInterval,
	)

	inputMode := 0
	if cfg.Input1ConfirmMode {
		inputMode = 1
	}
	msg3 := fmt.Sprintf("&&1&2&0&3&%s&%d&", formatModuleNumber(cfg.ObjectNumber), inputMode)

	return []SMSCommand{
		{Title: ProfileMCAGSM4 + " SMS №1", Text: msg1},
		{Title: ProfileMCAGSM4 + " SMS №2", Text: msg2},
		{Title: ProfileMCAGSM4 + " SMS №3", Text: msg3},
	}, nil
}

func normalizeMCAGSMConfig(cfg MCAGSM4Config) MCAGSM4Config {
	defaults := DefaultMCAGSMConfig()
	return fillMCADefaults(cfg, defaults)
}

func normalizeMCAGSM4Config(cfg MCAGSM4Config) MCAGSM4Config {
	return fillMCADefaults(cfg, DefaultMCAGSM4Config())
}

func fillMCADefaults(cfg MCAGSM4Config, defaults MCAGSM4Config) MCAGSM4Config {
	if strings.TrimSpace(cfg.PrimaryAPN) == "" {
		cfg.PrimaryAPN = defaults.PrimaryAPN
	}
	if strings.TrimSpace(cfg.ReserveAPN) == "" {
		cfg.ReserveAPN = defaults.ReserveAPN
	}
	if cfg.PrimaryModulePort == 0 {
		cfg.PrimaryModulePort = defaults.PrimaryModulePort
	}
	if cfg.ReserveModulePort == 0 {
		cfg.ReserveModulePort = defaults.ReserveModulePort
	}
	if cfg.PrimaryReceiverPort == 0 {
		cfg.PrimaryReceiverPort = defaults.PrimaryReceiverPort
	}
	if cfg.ReserveReceiverPort == 0 {
		cfg.ReserveReceiverPort = defaults.ReserveReceiverPort
	}
	if cfg.PrimaryTestInterval == 0 {
		cfg.PrimaryTestInterval = defaults.PrimaryTestInterval
	}
	if cfg.ReserveTestInterval == 0 {
		cfg.ReserveTestInterval = defaults.ReserveTestInterval
	}
	return cfg
}

func validateMCAGSMConfig(cfg MCAGSM4Config) error {
	return validateMCACommonConfig(cfg)
}

func validateMCAGSM4Config(cfg MCAGSM4Config) error {
	return validateMCACommonConfig(cfg)
}

func validateMCACommonConfig(cfg MCAGSM4Config) error {
	if cfg.ObjectNumber <= 0 {
		return fmt.Errorf("object number is required")
	}
	if cfg.HiddenNumber <= 0 {
		return fmt.Errorf("hidden number is required")
	}
	if parseIPv4(cfg.PrimaryIP) == nil {
		return fmt.Errorf("primary IP is invalid")
	}
	if parseIPv4(cfg.ReserveIP) == nil {
		return fmt.Errorf("reserve IP is invalid")
	}
	for _, item := range []struct {
		name  string
		value int
	}{
		{"primary module port", cfg.PrimaryModulePort},
		{"reserve module port", cfg.ReserveModulePort},
		{"primary receiver port", cfg.PrimaryReceiverPort},
		{"reserve receiver port", cfg.ReserveReceiverPort},
	} {
		if item.value < 1 || item.value > 9999 {
			return fmt.Errorf("%s is out of range", item.name)
		}
	}
	for _, item := range []struct {
		name  string
		value int
	}{
		{"primary test interval", cfg.PrimaryTestInterval},
		{"reserve test interval", cfg.ReserveTestInterval},
	} {
		if item.value < 1 || item.value > 240 {
			return fmt.Errorf("%s is out of range", item.name)
		}
	}
	return nil
}

func formatIPv4(raw string) string {
	ip := parseIPv4(raw)
	if ip == nil {
		return strings.TrimSpace(raw)
	}
	return fmt.Sprintf("%03d.%03d.%03d.%03d", ip[0], ip[1], ip[2], ip[3])
}

func parseIPv4(raw string) []byte {
	parts := strings.Split(strings.TrimSpace(raw), ".")
	if len(parts) != 4 {
		return nil
	}
	ip := make([]byte, 4)
	for i, part := range parts {
		value, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil || value < 0 || value > 255 {
			return nil
		}
		ip[i] = byte(value)
	}
	return ip
}

func formatModuleNumber(value int) string {
	if value < 10000 {
		return fmt.Sprintf("%04d", value)
	}
	return strconv.Itoa(value)
}
