//go:build qt

package qtapp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"obj_catalog_fyne_v3/pkg/backend"
	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/eventbus"
	"obj_catalog_fyne_v3/pkg/simoperator"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

const (
	simAutoResetInterval      = 60 * time.Second
	simAutoResetRecoveryAfter = 10 * time.Minute
	simAutoResetLimit         = 500
)

type simAutoResetProvider interface {
	CollectObjectStatistics(filter contracts.AdminStatisticsFilter, limit int) ([]contracts.AdminStatisticsRow, error)
	GetObjectCard(objn int64) (contracts.AdminObjectCard, error)
	RebootVodafoneSIM(msisdn string) (contracts.VodafoneSIMRebootResult, error)
	RebootKyivstarSIM(msisdn string) (contracts.KyivstarSIMResetResult, error)
}

type simAutoResetOperator struct {
	key        string
	label      string
	journal    simAutoResetJournal
	throttle   simAutoResetThrottle
	enabled    bool
	dailyLimit int
	window     time.Duration
}

type simAutoResetEpisode struct {
	objN        int64
	sim1        string
	operatorKey string
	result      string
	resetAt     time.Time
	checkAt     time.Time
	recovered   bool
	expired     bool
	throttled   bool
}

type simAutoResetJournal struct {
	path string
}

type simAutoResetThrottle struct {
	prefs      config.Preferences
	historyKey string
}

type simAutoResetHistory map[string][]string

func newSimAutoResetJournal(path string) simAutoResetJournal {
	return simAutoResetJournal{path: path}
}

func (j simAutoResetJournal) Appendf(format string, args ...any) {
	if strings.TrimSpace(j.path) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(j.path), 0o755); err != nil {
		log.Warn().Err(err).Str("path", j.path).Msg("Не вдалося створити каталог журналу SIM auto reset")
		return
	}
	file, err := os.OpenFile(j.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		log.Warn().Err(err).Str("path", j.path).Msg("Не вдалося відкрити журнал SIM auto reset")
		return
	}
	defer file.Close()

	line := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	if _, err := fmt.Fprintf(file, "%s %s\n", timestamp, line); err != nil {
		log.Warn().Err(err).Str("path", j.path).Msg("Не вдалося записати журнал SIM auto reset")
	}
}

func newSimAutoResetThrottle(prefs config.Preferences, historyKey string) simAutoResetThrottle {
	return simAutoResetThrottle{prefs: prefs, historyKey: historyKey}
}

func (t simAutoResetThrottle) AllowAndRecord(objN int64, sim1 string, limit int, window time.Duration, now time.Time) (bool, time.Time, int) {
	if limit <= 0 {
		return false, now.Add(window), 0
	}
	if window <= 0 {
		window = time.Hour
	}
	history := t.load()
	key := simAutoResetThrottleKey(objN, sim1)
	attempts := pruneSimAutoResetAttempts(history[key], window, now)
	if len(attempts) >= limit {
		history[key] = attempts
		t.save(history)
		return false, nextSimAutoResetAllowedAt(attempts, window), len(attempts)
	}

	attempts = append(attempts, now.UTC().Format(time.RFC3339Nano))
	history[key] = attempts
	t.save(history)
	return true, time.Time{}, len(attempts)
}

func (t simAutoResetThrottle) load() simAutoResetHistory {
	if t.prefs == nil {
		return simAutoResetHistory{}
	}
	raw := strings.TrimSpace(t.prefs.StringWithFallback(t.historyKey, ""))
	if raw == "" {
		return simAutoResetHistory{}
	}
	var history simAutoResetHistory
	if err := json.Unmarshal([]byte(raw), &history); err != nil {
		log.Warn().Err(err).Msg("Не вдалося прочитати історію SIM auto reset throttle")
		return simAutoResetHistory{}
	}
	if history == nil {
		return simAutoResetHistory{}
	}
	return history
}

func (t simAutoResetThrottle) save(history simAutoResetHistory) {
	if t.prefs == nil {
		return
	}
	data, err := json.Marshal(history)
	if err != nil {
		log.Warn().Err(err).Msg("Не вдалося зберегти історію SIM auto reset throttle")
		return
	}
	t.prefs.SetString(t.historyKey, string(data))
}

func simAutoResetThrottleKey(objN int64, sim1 string) string {
	return fmt.Sprintf("%d|%s", objN, strings.TrimSpace(sim1))
}

func pruneSimAutoResetAttempts(attempts []string, window time.Duration, now time.Time) []string {
	if len(attempts) == 0 {
		return nil
	}
	cutoff := now.UTC().Add(-window)
	pruned := make([]string, 0, len(attempts))
	for _, raw := range attempts {
		at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
		if err != nil || at.Before(cutoff) {
			continue
		}
		pruned = append(pruned, at.UTC().Format(time.RFC3339Nano))
	}
	return pruned
}

func nextSimAutoResetAllowedAt(attempts []string, window time.Duration) time.Time {
	if len(attempts) == 0 {
		return time.Now().UTC()
	}
	oldest := time.Time{}
	for _, raw := range attempts {
		at, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		if oldest.IsZero() || at.Before(oldest) {
			oldest = at
		}
	}
	if oldest.IsZero() {
		return time.Now().UTC()
	}
	return oldest.Add(window)
}

func shouldRefreshForLatestEventID(latestID, lastKnownID int64, hasLastKnownID bool) (refresh bool, nextLastKnownID int64, nextHasLastKnownID bool) {
	if !hasLastKnownID {
		return false, latestID, true
	}
	if latestID != lastKnownID {
		return true, latestID, true
	}
	return false, lastKnownID, true
}

func (a *Application) startGettingEvents() {
	if a == nil {
		return
	}

	if a.refreshLoopCancel != nil {
		a.refreshLoopCancel()
		a.refreshLoopCancel = nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.refreshLoopCancel = cancel

	uiCfg := config.LoadUIConfig(a.preferences())
	eventProbeInterval := time.Duration(uiCfg.EventProbeIntervalSec) * time.Second
	eventsReconcileInterval := time.Duration(uiCfg.EventsReconcileSec) * time.Second
	alarmsReconcileInterval := time.Duration(uiCfg.AlarmsReconcileSec) * time.Second
	objectsReconcileInterval := time.Duration(uiCfg.ObjectsReconcileSec) * time.Second
	fallbackRefreshInterval := time.Duration(uiCfg.FallbackRefreshSec) * time.Second
	maxProbeBackoffInterval := time.Duration(uiCfg.MaxProbeBackoffSec) * time.Second

	go func() {
		go a.startSIMAutoResetMonitor(ctx)

		eventProbeTicker := time.NewTicker(eventProbeInterval)
		eventsReconcileTicker := time.NewTicker(eventsReconcileInterval)
		alarmsReconcileTicker := time.NewTicker(alarmsReconcileInterval)
		objectsReconcileTicker := time.NewTicker(objectsReconcileInterval)
		fallbackTicker := time.NewTicker(fallbackRefreshInterval)
		defer eventProbeTicker.Stop()
		defer eventsReconcileTicker.Stop()
		defer alarmsReconcileTicker.Stop()
		defer objectsReconcileTicker.Stop()
		defer fallbackTicker.Stop()

		var (
			lastKnownEventID int64
			hasLastKnownID   bool
			probeBackoff     time.Duration
			nextProbeAt      time.Time
		)

		for {
			select {
			case <-ctx.Done():
				return

			case <-eventProbeTicker.C:
				now := time.Now()
				if !nextProbeAt.IsZero() && now.Before(nextProbeAt) {
					continue
				}

				provider := a.getDataProvider()
				probe, ok := provider.(latestEventIDProvider)
				if !ok {
					probeBackoff = 0
					nextProbeAt = time.Time{}
					continue
				}

				latestID, err := probe.GetLatestEventID()
				if err != nil {
					if probeBackoff == 0 {
						probeBackoff = eventProbeInterval
					} else {
						probeBackoff *= 2
						if probeBackoff > maxProbeBackoffInterval {
							probeBackoff = maxProbeBackoffInterval
						}
					}
					nextProbeAt = now.Add(probeBackoff)
					log.Debug().Err(err).Msg("Не вдалося виконати probe останнього event ID")
					continue
				}
				probeBackoff = 0
				nextProbeAt = time.Time{}

				refresh, nextID, hasNext := shouldRefreshForLatestEventID(latestID, lastKnownEventID, hasLastKnownID)
				lastKnownEventID = nextID
				hasLastKnownID = hasNext
				if refresh {
					a.publishDataRefresh(eventbus.DataRefreshEvent{
						RefreshAlarms: true,
						RefreshEvents: true,
					})
				}

			case <-eventsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshEvents: true})

			case <-alarmsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshAlarms: true})

			case <-objectsReconcileTicker.C:
				a.publishDataRefresh(eventbus.DataRefreshEvent{RefreshObjects: true})

			case <-fallbackTicker.C:
				if _, ok := a.getDataProvider().(latestEventIDProvider); !ok {
					a.publishDataRefresh(eventbus.DataRefreshEvent{
						RefreshObjects: true,
						RefreshAlarms:  true,
						RefreshEvents:  true,
					})
				}
			}
		}
	}()
}

func (a *Application) startSIMAutoResetMonitor(ctx context.Context) {
	if a == nil {
		return
	}

	episodes := make(map[string]simAutoResetEpisode)

	ticker := time.NewTicker(simAutoResetInterval)
	defer ticker.Stop()

	a.runSIMAutoResetCycle(ctx, episodes)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runSIMAutoResetCycle(ctx, episodes)
		}
	}
}

func (a *Application) runSIMAutoResetCycle(ctx context.Context, episodes map[string]simAutoResetEpisode) {
	if ctx.Err() != nil {
		return
	}

	operators := a.simAutoResetOperators()
	if len(operators) == 0 {
		for key := range episodes {
			delete(episodes, key)
		}
		return
	}

	provider, ok := resolveAdminCapability[simAutoResetProvider](a)
	if !ok {
		return
	}

	now := time.Now()
	rows, err := provider.CollectObjectStatistics(simAutoResetStatisticsFilter(), simAutoResetLimit)
	if err != nil {
		log.Warn().Err(err).Msg("Не вдалося отримати offline MOST для SIM auto reset")
		for _, operator := range operators {
			operator.journal.Appendf("offline MOST: помилка отримання статистики: %v", err)
		}
		return
	}

	offline := make(map[int64]contracts.AdminStatisticsRow, len(rows))
	for _, row := range rows {
		if row.ObjN <= 0 {
			continue
		}
		offline[row.ObjN] = row
	}

	for key, episode := range episodes {
		operator, ok := operators[episode.operatorKey]
		if !ok {
			delete(episodes, key)
			continue
		}
		if episode.throttled && !now.Before(episode.checkAt) {
			delete(episodes, key)
			continue
		}
		if episode.recovered {
			if _, offlineAgain := offline[episode.objN]; offlineAgain {
				delete(episodes, key)
			}
			continue
		}
		if now.Before(episode.checkAt) {
			continue
		}
		if _, stillOffline := offline[episode.objN]; stillOffline {
			if !episode.expired {
				episode.expired = true
				episodes[key] = episode
				operator.journal.Appendf("об'єкт %d не відновив зв'язок протягом 10 хвилин після reset sim %s; %s", episode.objN, operator.label, episode.result)
			}
			continue
		}
		episode.recovered = true
		episodes[key] = episode
		if episode.expired {
			operator.journal.Appendf("об'єкт %d відновив зв'язок після 10 хвилин очікування; %s", episode.objN, episode.result)
			continue
		}
		operator.journal.Appendf("об'єкт %d відновив зв'язок протягом 10 хвилин після reset sim; %s", episode.objN, episode.result)
	}

	for _, row := range rows {
		card, err := provider.GetObjectCard(row.ObjN)
		if err != nil {
			log.Warn().Err(err).Int64("objN", row.ObjN).Msg("Не вдалося прочитати картку об'єкта для SIM auto reset")
			for _, operator := range operators {
				operator.journal.Appendf("об'єкт %d втрата зв'язку: не вдалося прочитати картку: %v", row.ObjN, err)
			}
			continue
		}

		sim1 := strings.TrimSpace(card.GSMPhone1)
		operator, ok := selectSIMAutoResetOperator(sim1, operators)
		if !ok {
			continue
		}
		key := simAutoResetEpisodeKey(operator.key, row.ObjN)
		if _, exists := episodes[key]; exists {
			continue
		}
		allowed, nextAllowedAt, used := operator.throttle.AllowAndRecord(
			row.ObjN,
			sim1,
			operator.dailyLimit,
			operator.window,
			now,
		)
		if !allowed {
			operator.journal.Appendf(
				"об'єкт %d втрата зв'язку, SIM1=%s %s: reset sim пропущено через throttle (%d/%d, наступна спроба після %s)",
				row.ObjN,
				sim1,
				operator.label,
				used,
				operator.dailyLimit,
				nextAllowedAt.Local().Format("2006-01-02 15:04:05"),
			)
			episodes[key] = simAutoResetEpisode{
				objN:        row.ObjN,
				sim1:        sim1,
				operatorKey: operator.key,
				checkAt:     nextAllowedAt,
				throttled:   true,
			}
			continue
		}

		operator.journal.Appendf("об'єкт %d втрата зв'язку, SIM1=%s %s, надсилаємо запит reset sim", row.ObjN, sim1, operator.label)
		result, err := sendSIMAutoResetRequest(provider, operator.key, sim1)
		if err != nil {
			log.Warn().Err(err).Int64("objN", row.ObjN).Str("sim1", sim1).Str("operator", operator.key).Msg("SIM auto reset failed")
			operator.journal.Appendf("об'єкт %d reset sim %s результат: помилка: %v", row.ObjN, operator.label, err)
			episodes[key] = simAutoResetEpisode{
				objN:        row.ObjN,
				sim1:        sim1,
				operatorKey: operator.key,
				resetAt:     now,
				checkAt:     now.Add(simAutoResetRecoveryAfter),
				result:      fmt.Sprintf("помилка: %v", err),
			}
			continue
		}

		operator.journal.Appendf("об'єкт %d reset sim %s результат: %s", row.ObjN, operator.label, result)
		episodes[key] = simAutoResetEpisode{
			objN:        row.ObjN,
			sim1:        sim1,
			operatorKey: operator.key,
			resetAt:     now,
			checkAt:     now.Add(simAutoResetRecoveryAfter),
			result:      result,
		}
	}
}

func (a *Application) simAutoResetOperators() map[string]simAutoResetOperator {
	prefs := a.preferences()
	operators := make(map[string]simAutoResetOperator, 2)

	vfCfg := config.LoadVodafoneConfig(prefs)
	if vfCfg.AutoResetEnabled {
		operators[string(simoperator.Vodafone)] = simAutoResetOperator{
			key:        string(simoperator.Vodafone),
			label:      simoperator.Label(simoperator.Vodafone),
			journal:    newSimAutoResetJournal(filepath.Join("log", "vodafone_auto_reset.log")),
			throttle:   newSimAutoResetThrottle(prefs, config.PrefVodafoneAutoResetHistory),
			enabled:    vfCfg.AutoResetEnabled,
			dailyLimit: vfCfg.AutoResetDailyLimit,
			window:     time.Duration(vfCfg.AutoResetWindowHours) * time.Hour,
		}
	}

	ksCfg := config.LoadKyivstarConfig(prefs)
	if ksCfg.AutoResetEnabled {
		operators[string(simoperator.Kyivstar)] = simAutoResetOperator{
			key:        string(simoperator.Kyivstar),
			label:      simoperator.Label(simoperator.Kyivstar),
			journal:    newSimAutoResetJournal(filepath.Join("log", "kyivstar_auto_reset.log")),
			throttle:   newSimAutoResetThrottle(prefs, config.PrefKyivstarAutoResetHistory),
			enabled:    ksCfg.AutoResetEnabled,
			dailyLimit: ksCfg.AutoResetDailyLimit,
			window:     time.Duration(ksCfg.AutoResetWindowHours) * time.Hour,
		}
	}
	return operators
}

func selectSIMAutoResetOperator(sim1 string, operators map[string]simAutoResetOperator) (simAutoResetOperator, bool) {
	operator := simoperator.Detect(sim1)
	if operator == simoperator.Unknown {
		return simAutoResetOperator{}, false
	}
	cfg, ok := operators[string(operator)]
	return cfg, ok && cfg.enabled
}

func sendSIMAutoResetRequest(provider simAutoResetProvider, operatorKey string, sim1 string) (string, error) {
	switch operatorKey {
	case string(simoperator.Vodafone):
		result, err := provider.RebootVodafoneSIM(sim1)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("orderID=%s state=%s", result.OrderID, result.State), nil
	case string(simoperator.Kyivstar):
		result, err := provider.RebootKyivstarSIM(sim1)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("msisdn=%s email=%s", result.MSISDN, result.Email), nil
	default:
		return "", fmt.Errorf("невідомий оператор SIM1: %s", operatorKey)
	}
}

func simAutoResetEpisodeKey(operatorKey string, objN int64) string {
	return fmt.Sprintf("%s|%d", operatorKey, objN)
}

func simAutoResetStatisticsFilter() contracts.AdminStatisticsFilter {
	channelCode := int64(5)
	guardState := int64(1)
	blockMode := contracts.DisplayBlockNone
	return contracts.AdminStatisticsFilter{
		ConnectionMode: contracts.StatsConnectionOffline,
		ProtocolFilter: contracts.StatsProtocolMost,
		ChannelCode:    &channelCode,
		GuardState:     &guardState,
		BlockMode:      &blockMode,
	}
}

func (a *Application) preferences() config.Preferences {
	if a == nil || a.ui == nil {
		return nil
	}
	return a.ui.Preferences()
}

func (a *Application) getDataProvider() contracts.DataProvider {
	if a == nil || a.runtime == nil {
		return nil
	}
	return a.runtime.Provider
}

func (a *Application) resolveAdminProvider() contracts.AdminProvider {
	if a == nil || a.runtime == nil || a.runtime.Provider == nil {
		return nil
	}
	adminProvider, ok := backend.AsAdminProvider(a.runtime.Provider)
	if !ok {
		return nil
	}
	return adminProvider
}

func resolveAdminCapability[T any](a *Application) (T, bool) {
	var zero T
	adminProvider := a.resolveAdminProvider()
	if adminProvider == nil {
		return zero, false
	}
	capability, ok := any(adminProvider).(T)
	if !ok {
		return zero, false
	}
	return capability, true
}
