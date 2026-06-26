package simwatchdog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"obj_catalog_fyne_v3/pkg/config"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/simoperator"
	"obj_catalog_fyne_v3/pkg/utils"

	"github.com/rs/zerolog/log"
)

// Options controls the SIM watchdog loop.
type Options struct {
	PollInterval     time.Duration
	HistoryPath      string
	DryRun           bool
	IncludeNonBridge bool
	MaxLastTestAge   time.Duration
}

// ObjectProvider is the minimal object source used by the watchdog.
type ObjectProvider interface {
	GetObjects() []models.Object
}

// LastTestProvider provides the last successful GPRS test timestamp.
type LastTestProvider interface {
	LastGPRSTestTime(ctx context.Context, objectID int) (time.Time, error)
}

// KyivstarRebooter resets Kyivstar SIM cards.
type KyivstarRebooter interface {
	RebootSIM(msisdn string) (contracts.KyivstarSIMResetResult, error)
}

// VodafoneRebooter resets Vodafone SIM cards.
type VodafoneRebooter interface {
	RebootSIM(msisdn string) (contracts.VodafoneSIMRebootResult, error)
}

// Runner periodically checks offline objects and reboots their SIM cards.
type Runner struct {
	objects  ObjectProvider
	kyivstar KyivstarRebooter
	vodafone VodafoneRebooter
	store    ConfigStore
	history  *History
	options  Options
}

// ConfigStore provides operator reset settings.
type ConfigStore interface {
	config.KyivstarConfigStore
	config.VodafoneConfigStore
}

// NewRunner creates a SIM watchdog runner.
func NewRunner(objects ObjectProvider, kyivstar KyivstarRebooter, vodafone VodafoneRebooter, store ConfigStore, opts Options) (*Runner, error) {
	if objects == nil {
		return nil, errors.New("sim watchdog: object provider is not configured")
	}
	if kyivstar == nil {
		return nil, errors.New("sim watchdog: kyivstar service is not configured")
	}
	if vodafone == nil {
		return nil, errors.New("sim watchdog: vodafone service is not configured")
	}
	if store == nil {
		return nil, errors.New("sim watchdog: config store is not configured")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 3 * time.Minute
	}
	if strings.TrimSpace(opts.HistoryPath) == "" {
		opts.HistoryPath = filepath.Join("log", "sim-watchdog-history.json")
	}
	if opts.MaxLastTestAge <= 0 {
		opts.MaxLastTestAge = 7 * 24 * time.Hour
	}
	history, err := LoadHistory(opts.HistoryPath)
	if err != nil {
		return nil, err
	}
	return &Runner{
		objects:  objects,
		kyivstar: kyivstar,
		vodafone: vodafone,
		store:    store,
		history:  history,
		options:  opts,
	}, nil
}

// Run starts the loop until ctx is canceled.
func (r *Runner) Run(ctx context.Context) error {
	if r == nil {
		return errors.New("sim watchdog: runner is nil")
	}
	if err := r.CheckOnce(ctx); err != nil {
		log.Error().Err(err).Msg("sim watchdog: check failed")
	}

	ticker := time.NewTicker(r.options.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.CheckOnce(ctx); err != nil {
				log.Error().Err(err).Msg("sim watchdog: check failed")
			}
		}
	}
}

// CheckOnce performs one offline-object scan.
func (r *Runner) CheckOnce(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	objects := r.objects.GetObjects()
	offline := make([]models.Object, 0)
	skippedByLastTest := 0
	for _, obj := range objects {
		if !r.shouldHandleObject(obj) {
			continue
		}
		if !r.hasRecentLastTest(ctx, obj, time.Now()) {
			skippedByLastTest++
			continue
		}
		offline = append(offline, obj)
	}

	log.Info().
		Int("objects", len(objects)).
		Int("offline", len(offline)).
		Int("skippedByLastTest", skippedByLastTest).
		Bool("dryRun", r.options.DryRun).
		Msg("sim watchdog: object scan completed")

	for _, obj := range offline {
		if err := r.handleObject(ctx, obj); err != nil {
			log.Warn().
				Err(err).
				Int("objectID", obj.ID).
				Str("objectName", obj.Name).
				Msg("sim watchdog: object handling failed")
		}
	}
	return nil
}

func (r *Runner) shouldHandleObject(obj models.Object) bool {
	if !r.options.IncludeNonBridge && (ids.IsPhoenixObjectID(obj.ID) || ids.IsCASLObjectID(obj.ID)) {
		return false
	}
	if obj.ObjChan != 5 {
		return false
	}
	if obj.ConnectionStatusValue() != models.ConnectionStatusOffline && obj.Status != models.StatusOffline {
		return false
	}
	if obj.GuardStatusValue() != models.GuardStatusGuarded {
		return false
	}
	if obj.MonitoringStatusValue() == models.MonitoringStatusBlocked || obj.MonitoringStatusValue() == models.MonitoringStatusDebug {
		return false
	}
	return true
}

func (r *Runner) hasRecentLastTest(ctx context.Context, obj models.Object, now time.Time) bool {
	if r.options.MaxLastTestAge <= 0 {
		return true
	}
	provider, ok := r.objects.(LastTestProvider)
	if !ok {
		log.Warn().
			Int("objectID", obj.ID).
			Msg("sim watchdog: last GPRS test provider is unavailable, object skipped")
		return false
	}

	lastTest, err := provider.LastGPRSTestTime(ctx, obj.ID)
	if err != nil {
		log.Warn().
			Err(err).
			Int("objectID", obj.ID).
			Msg("sim watchdog: failed to read last GPRS test, object skipped")
		return false
	}
	if lastTest.IsZero() {
		log.Warn().
			Int("objectID", obj.ID).
			Msg("sim watchdog: last GPRS test is empty, object skipped")
		return false
	}
	if lastTest.Before(now.Add(-r.options.MaxLastTestAge)) {
		log.Info().
			Int("objectID", obj.ID).
			Time("lastTest", lastTest).
			Dur("maxAge", r.options.MaxLastTestAge).
			Msg("sim watchdog: object skipped because last GPRS test is too old")
		return false
	}
	return true
}

func (r *Runner) handleObject(ctx context.Context, obj models.Object) error {
	sims := objectSIMs(obj)
	for _, sim := range sims {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := r.handleSIM(obj, sim); err != nil {
			log.Warn().
				Err(err).
				Int("objectID", obj.ID).
				Str("slot", sim.slot).
				Str("msisdn", sim.msisdn).
				Str("operator", sim.operatorLabel()).
				Msg("sim watchdog: SIM reboot skipped or failed")
		}
	}
	return nil
}

func (r *Runner) handleSIM(obj models.Object, sim objectSIM) error {
	limit, window, enabled := r.operatorLimit(sim.operator)
	if !enabled {
		return fmt.Errorf("%s auto reset is disabled", sim.operatorLabel())
	}
	if limit <= 0 {
		return fmt.Errorf("%s daily reset limit is zero", sim.operatorLabel())
	}

	key := historyKey(sim.operator, sim.msisdn)
	now := time.Now()
	if !r.history.CanReset(key, now, window, limit) {
		return fmt.Errorf("%s reset limit reached for %s", sim.operatorLabel(), sim.msisdn)
	}

	log.Info().
		Int("objectID", obj.ID).
		Str("objectName", obj.Name).
		Str("slot", sim.slot).
		Str("msisdn", sim.msisdn).
		Str("operator", sim.operatorLabel()).
		Msg("sim watchdog: rebooting SIM for offline object")

	if r.options.DryRun {
		log.Info().
			Int("objectID", obj.ID).
			Str("msisdn", sim.msisdn).
			Msg("sim watchdog: dry-run, reboot not sent")
		return nil
	}

	switch sim.operator {
	case simoperator.Kyivstar:
		if _, err := r.kyivstar.RebootSIM(sim.msisdn); err != nil {
			return err
		}
	case simoperator.Vodafone:
		if _, err := r.vodafone.RebootSIM(sim.msisdn); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported SIM operator %q", sim.operator)
	}

	r.history.Record(key, HistoryEntry{
		Time:       now,
		Operator:   string(sim.operator),
		MSISDN:     sim.msisdn,
		ObjectID:   obj.ID,
		ObjectName: obj.Name,
		Slot:       sim.slot,
	})
	if err := r.history.Save(); err != nil {
		return err
	}
	return nil
}

func (r *Runner) operatorLimit(operator simoperator.Operator) (limit int, window time.Duration, enabled bool) {
	switch operator {
	case simoperator.Kyivstar:
		cfg := r.store.LoadKyivstarConfig()
		return cfg.AutoResetDailyLimit, time.Duration(cfg.AutoResetWindowHours) * time.Hour, cfg.AutoResetEnabled
	case simoperator.Vodafone:
		cfg := r.store.LoadVodafoneConfig()
		return cfg.AutoResetDailyLimit, time.Duration(cfg.AutoResetWindowHours) * time.Hour, cfg.AutoResetEnabled
	default:
		return 0, 0, false
	}
}

type objectSIM struct {
	slot     string
	msisdn   string
	operator simoperator.Operator
}

func (s objectSIM) operatorLabel() string {
	return simoperator.Label(s.operator)
}

func objectSIMs(obj models.Object) []objectSIM {
	candidates := []struct {
		slot string
		raw  string
	}{
		{slot: "SIM1", raw: obj.SIM1},
		{slot: "SIM2", raw: obj.SIM2},
	}

	result := make([]objectSIM, 0, 2)
	seen := make(map[string]struct{}, 2)
	for _, candidate := range candidates {
		msisdn := strings.TrimSpace(candidate.raw)
		if msisdn == "" {
			continue
		}
		operator := simoperator.Detect(msisdn)
		if operator != simoperator.Kyivstar && operator != simoperator.Vodafone {
			continue
		}
		key := historyKey(operator, msisdn)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, objectSIM{
			slot:     candidate.slot,
			msisdn:   msisdn,
			operator: operator,
		})
	}
	return result
}

func historyKey(operator simoperator.Operator, msisdn string) string {
	return string(operator) + ":" + utils.DigitsOnly(msisdn)
}

// History stores reboot attempts by SIM.
type History struct {
	path    string
	mu      sync.Mutex
	Entries map[string][]HistoryEntry `json:"entries"`
}

// HistoryEntry describes one successful reboot request.
type HistoryEntry struct {
	Time       time.Time `json:"time"`
	Operator   string    `json:"operator"`
	MSISDN     string    `json:"msisdn"`
	ObjectID   int       `json:"object_id"`
	ObjectName string    `json:"object_name"`
	Slot       string    `json:"slot"`
}

// LoadHistory reads reset history or creates an empty in-memory store.
func LoadHistory(path string) (*History, error) {
	h := &History{
		path:    strings.TrimSpace(path),
		Entries: make(map[string][]HistoryEntry),
	}
	if h.path == "" {
		return h, nil
	}
	body, err := os.ReadFile(h.path)
	if err == nil {
		if len(strings.TrimSpace(string(body))) == 0 {
			return h, nil
		}
		if err := json.Unmarshal(body, h); err != nil {
			return nil, fmt.Errorf("sim watchdog: read history %q: %w", h.path, err)
		}
		h.path = strings.TrimSpace(path)
		if h.Entries == nil {
			h.Entries = make(map[string][]HistoryEntry)
		}
		return h, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return h, nil
	}
	return nil, fmt.Errorf("sim watchdog: read history %q: %w", h.path, err)
}

// CanReset reports whether key has remaining reset allowance in window.
func (h *History) CanReset(key string, now time.Time, window time.Duration, limit int) bool {
	if h == nil || limit <= 0 {
		return false
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	entries := h.pruneLocked(key, now.Add(-window))
	return len(entries) < limit
}

// Record stores a reboot history entry.
func (h *History) Record(key string, entry HistoryEntry) {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.Entries == nil {
		h.Entries = make(map[string][]HistoryEntry)
	}
	h.Entries[key] = append(h.Entries[key], entry)
	sort.SliceStable(h.Entries[key], func(i, j int) bool {
		return h.Entries[key][i].Time.Before(h.Entries[key][j].Time)
	})
}

// Save writes history to disk.
func (h *History) Save() error {
	if h == nil || strings.TrimSpace(h.path) == "" {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if dir := filepath.Dir(h.path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("sim watchdog: create history directory %q: %w", dir, err)
		}
	}
	body, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("sim watchdog: encode history: %w", err)
	}
	body = append(body, '\n')
	if err := os.WriteFile(h.path, body, 0o600); err != nil {
		return fmt.Errorf("sim watchdog: write history %q: %w", h.path, err)
	}
	return nil
}

func (h *History) pruneLocked(key string, cutoff time.Time) []HistoryEntry {
	if h.Entries == nil {
		h.Entries = make(map[string][]HistoryEntry)
	}
	entries := h.Entries[key]
	kept := entries[:0]
	for _, entry := range entries {
		if entry.Time.IsZero() || entry.Time.Before(cutoff) {
			continue
		}
		kept = append(kept, entry)
	}
	h.Entries[key] = kept
	return kept
}
