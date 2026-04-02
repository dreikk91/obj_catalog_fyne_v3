package viewmodels

import (
	"fmt"
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
	if phone == "" && !state.Authorized {
		return "Vodafone: номер входу не налаштований"
	}
	if state.Authorized {
		if state.TokenExpiresAt.IsZero() {
			return fmt.Sprintf("Vodafone: авторизовано для %s", fallbackString(phone, "вказаного номера"))
		}
		return fmt.Sprintf(
			"Vodafone: авторизовано для %s до %s",
			fallbackString(phone, "вказаного номера"),
			state.TokenExpiresAt.Local().Format("02.01.2006 15:04"),
		)
	}
	if phone != "" && !state.TokenExpiresAt.IsZero() && state.TokenExpiresAt.Before(time.Now()) {
		return fmt.Sprintf("Vodafone: токен для %s прострочений", phone)
	}
	if phone != "" {
		return fmt.Sprintf("Vodafone: потрібна авторизація для %s", phone)
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
