package viewmodels

import (
	"fmt"
	"obj_catalog_fyne_v3/pkg/contracts"
	"strings"
	"time"
)

type KyivstarAuthViewModel struct{}

func NewKyivstarAuthViewModel() *KyivstarAuthViewModel {
	return &KyivstarAuthViewModel{}
}

func (vm *KyivstarAuthViewModel) BuildStatusText(state contracts.KyivstarAuthState) string {
	clientID := strings.TrimSpace(state.ClientID)
	if !state.Configured && clientID == "" {
		return "Kyivstar: client_id/client_secret не налаштовані"
	}
	if state.Authorized {
		if state.TokenExpiresAt.IsZero() {
			return fmt.Sprintf("Kyivstar: токен активний для %s", fallbackString(clientID, "вказаного client_id"))
		}
		return fmt.Sprintf(
			"Kyivstar: токен активний для %s до %s",
			fallbackString(clientID, "вказаного client_id"),
			state.TokenExpiresAt.Local().Format("02.01.2006 15:04"),
		)
	}
	if clientID != "" && !state.TokenExpiresAt.IsZero() && state.TokenExpiresAt.Before(time.Now()) {
		return fmt.Sprintf("Kyivstar: токен для %s прострочений", clientID)
	}
	if clientID != "" {
		return fmt.Sprintf("Kyivstar: готовий отримати токен для %s", clientID)
	}
	return "Kyivstar: потрібні client_id/client_secret"
}
