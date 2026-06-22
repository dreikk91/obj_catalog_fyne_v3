//go:build qt

package qtapp

import (
	"strings"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/ami"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
)

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

func buildAMIDialer(prefs config.Preferences) contracts.PhoneDialer {
	if prefs == nil {
		return nil
	}
	enabled, cfg := config.LoadAMIConfig(prefs)
	if !enabled {
		return nil
	}
	client, err := ami.NewClientLazy(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Qt AMI: не вдалося ініціалізувати клієнт")
		return nil
	}
	log.Info().Str("host", cfg.Host).Int("port", cfg.Port).Msg("Qt AMI: клієнт ініціалізовано")
	return &amiDialerAdapter{client: client}
}

func (a *Application) dialPhone(phone string) {
	if a == nil || a.ui == nil {
		return
	}
	phone = strings.TrimSpace(phone)
	if phone == "" {
		a.ui.ShowInfo("Дзвінок", "У контакта немає номера телефону.")
		return
	}
	if a.phoneDialer == nil {
		a.ui.ShowInfo("Дзвінок", "AMI-команди вимкнені або не налаштовані.")
		return
	}

	a.ui.SetStatus("AMI: виклик " + phone)
	go func() {
		callID, err := a.phoneDialer.DialPhone(phone)
		a.runOnMainThread(func() {
			if err != nil {
				a.ui.ShowError("AMI-команда", "Не вдалося виконати дзвінок: "+err.Error())
				return
			}
			a.ui.SetStatus("AMI: дзвінок створено " + phone + " | " + callID)
		})
	}()
}
