package application

import (
	"obj_catalog_fyne_v3/pkg/ami"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"

	"github.com/rs/zerolog/log"
)

// amiDialerAdapter адаптує ami.Client до contracts.PhoneDialer.
type amiDialerAdapter struct {
	client *ami.Client
}

func (a *amiDialerAdapter) DialPhone(phone string) (string, error) {
	return a.client.Originate(phone, "")
}

func (a *amiDialerAdapter) HangupCall(callID string) {
	a.client.Hangup(callID)
}

func (a *amiDialerAdapter) IsDialerConnected() bool {
	return a.client.IsConnected()
}

// buildDialerFromSettings будує PhoneDialer з налаштувань. Nil якщо вимкнено.
func buildDialerFromSettings(settings contracts.AMISettings) contracts.PhoneDialer {
	if !settings.Enabled {
		return nil
	}
	cfg := ami.Config{
		Host:      settings.Host,
		Port:      settings.Port,
		Username:  settings.Username,
		Secret:    settings.Secret,
		Extension: settings.Extension,
		Context:   settings.Context,
	}
	client, err := ami.NewClientLazy(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("AMI: не вдалося ініціалізувати клієнт")
		return nil
	}
	log.Info().Str("host", cfg.Host).Int("port", cfg.Port).Msg("AMI: клієнт ініціалізовано")
	return &amiDialerAdapter{client: client}
}

// buildAMIDialer читає конфіг з Fyne Preferences і повертає PhoneDialer або nil.
func buildAMIDialer(app *Application) contracts.PhoneDialer {
	if app == nil {
		return nil
	}
	prefs := app.fyneApp.Preferences()
	enabled, cfg := config.LoadAMIConfig(prefs)
	return buildDialerFromSettings(contracts.AMISettings{
		Enabled:   enabled,
		Host:      cfg.Host,
		Port:      cfg.Port,
		Username:  cfg.Username,
		Secret:    cfg.Secret,
		Extension: cfg.Extension,
		Context:   cfg.Context,
	})
}

// applicationAMISettings реалізує contracts.AMISettingsProvider через Fyne Preferences.
type applicationAMISettings struct {
	app *Application
}

func (s applicationAMISettings) GetAMISettings() contracts.AMISettings {
	prefs := s.app.fyneApp.Preferences()
	enabled, cfg := config.LoadAMIConfig(prefs)
	return contracts.AMISettings{
		Enabled:   enabled,
		Host:      cfg.Host,
		Port:      cfg.Port,
		Username:  cfg.Username,
		Secret:    cfg.Secret,
		Extension: cfg.Extension,
		Context:   cfg.Context,
	}
}

func (s applicationAMISettings) SaveAMISettings(settings contracts.AMISettings) error {
	prefs := s.app.fyneApp.Preferences()
	prefs.SetBool(config.PrefAMIEnabled, settings.Enabled)
	prefs.SetString(config.PrefAMIHost, settings.Host)
	prefs.SetInt(config.PrefAMIPort, settings.Port)
	prefs.SetString(config.PrefAMIUsername, settings.Username)
	prefs.SetString(config.PrefAMISecret, settings.Secret)
	prefs.SetString(config.PrefAMIExtension, settings.Extension)
	prefs.SetString(config.PrefAMIContext, settings.Context)
	return nil
}
