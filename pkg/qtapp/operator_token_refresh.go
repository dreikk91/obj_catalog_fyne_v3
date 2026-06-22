//go:build qt

package qtapp

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
)

const (
	operatorTokenRefreshInterval = 5 * time.Minute
	operatorTokenRefreshAhead    = 15 * time.Minute
)

type vodafoneTokenRefresher interface {
	RefreshVodafoneToken() (contracts.VodafoneAuthState, error)
}

type kyivstarTokenRefresher interface {
	RefreshKyivstarToken() (contracts.KyivstarAuthState, error)
}

func (a *Application) startOperatorTokenRefreshMonitor(ctx context.Context) {
	if a == nil {
		return
	}

	a.refreshOperatorTokens()

	ticker := time.NewTicker(operatorTokenRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.refreshOperatorTokens()
		}
	}
}

func (a *Application) refreshOperatorTokens() {
	prefs := a.preferences()
	if prefs == nil {
		return
	}
	now := time.Now().UTC()
	refreshAt := now.Add(operatorTokenRefreshAhead)

	if shouldRefreshVodafoneToken(config.LoadVodafoneConfig(prefs), refreshAt) {
		refresher, ok := resolveAdminCapability[vodafoneTokenRefresher](a)
		if ok {
			if state, err := refresher.RefreshVodafoneToken(); err != nil {
				log.Warn().Err(err).Msg("Не вдалося оновити Vodafone token")
			} else {
				log.Info().Time("expiresAt", state.TokenExpiresAt).Msg("Vodafone token оновлено")
			}
		}
	}

	if shouldRefreshKyivstarToken(config.LoadKyivstarConfig(prefs), refreshAt) {
		refresher, ok := resolveAdminCapability[kyivstarTokenRefresher](a)
		if ok {
			if state, err := refresher.RefreshKyivstarToken(); err != nil {
				log.Warn().Err(err).Msg("Не вдалося оновити Kyivstar token")
			} else {
				log.Info().Time("expiresAt", state.TokenExpiresAt).Msg("Kyivstar token оновлено")
			}
		}
	}
}

func shouldRefreshVodafoneToken(cfg config.VodafoneConfig, at time.Time) bool {
	return cfg.NormalizedLoginMethod() == config.VodafoneLoginMethodPUK &&
		cfg.HasPUKCredentials() &&
		!cfg.TokenUsableAt(at)
}

func shouldRefreshKyivstarToken(cfg config.KyivstarConfig, at time.Time) bool {
	return cfg.HasCredentials() && !cfg.TokenUsableAt(at)
}
