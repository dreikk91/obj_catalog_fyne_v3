package viewmodels

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"time"
)

type VodafoneAuthViewModel struct{}

func NewVodafoneAuthViewModel() *VodafoneAuthViewModel {
	return &VodafoneAuthViewModel{}
}

func (vm *VodafoneAuthViewModel) BuildStatusText(state contracts.VodafoneAuthState) string {
	phone := strings.TrimSpace(state.Phone)
	cfg := config.VodafoneConfig{LoginMethod: state.LoginMethod}
	method := cfg.NormalizedLoginMethod()
	methodText := "SMS"
	if method == config.VodafoneLoginMethodPUK {
		methodText = "PUK"
	}
	if phone == "" && !state.Authorized {
		return "Vodafone: номер входу не налаштований"
	}
	if state.Authorized {
		if state.TokenExpiresAt.IsZero() {
			return fmt.Sprintf("Vodafone: авторизовано для %s (%s)", fallbackString(phone, "вказаного номера"), methodText)
		}
		text := fmt.Sprintf(
			"Vodafone: авторизовано для %s до %s",
			fallbackString(phone, "вказаного номера"),
			state.TokenExpiresAt.Local().Format("02.01.2006 15:04"),
		)
		if method == config.VodafoneLoginMethodPUK && state.PUKConfigured {
			text += ", автооновлення через PUK увімкнене"
		}
		return text
	}
	if phone != "" && !state.TokenExpiresAt.IsZero() && state.TokenExpiresAt.Before(time.Now()) {
		return fmt.Sprintf("Vodafone: токен для %s прострочений", phone)
	}
	if phone != "" {
		if method == config.VodafoneLoginMethodPUK {
			if state.PUKConfigured {
				return fmt.Sprintf("Vodafone: потрібен вхід через PUK для %s", phone)
			}
			return fmt.Sprintf("Vodafone: введіть PUK для %s", phone)
		}
		return fmt.Sprintf("Vodafone: потрібна SMS-авторизація для %s", phone)
	}
	return "Vodafone: потрібна авторизація"
}

func fallbackString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
