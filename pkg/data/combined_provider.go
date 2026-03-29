package data

import (
	"errors"
	"fmt"
	"hash/fnv"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
	"sort"
	"strconv"
	"strings"
	"time"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

// ProviderSource описує одне джерело даних у мультисистемній конфігурації.
// OwnsObjectID/OwnsAlarmID задають, як маршрутизувати запити до цього джерела.
// Якщо жоден matcher не спрацював, використовується перше (основне) джерело.
type ProviderSource struct {
	Name         string
	Provider     contracts.DataProvider
	OwnsObjectID func(id int) bool
	OwnsAlarmID  func(id int) bool
}

// CombinedDataProvider об'єднує декілька пультових систем в один DataProvider.
// Назва збережена для зворотної сумісності.
type CombinedDataProvider struct {
	sources []ProviderSource
}

func NewCombinedDataProvider(primary contracts.DataProvider, secondary contracts.DataProvider) *CombinedDataProvider {
	sources := make([]ProviderSource, 0, 2)
	if primary != nil {
		sources = append(sources, ProviderSource{
			Name:     "primary",
			Provider: primary,
		})
	}
	if secondary != nil {
		sources = append(sources, ProviderSource{
			Name:         "casl",
			Provider:     secondary,
			OwnsObjectID: isCASLObjectID,
			OwnsAlarmID:  isCASLObjectID,
		})
	}
	return NewMultiSourceDataProvider(sources...)
}

// NewMultiSourceDataProvider створює агрегатор для довільної кількості пультових систем.
func NewMultiSourceDataProvider(sources ...ProviderSource) *CombinedDataProvider {
	filtered := make([]ProviderSource, 0, len(sources))
	for _, source := range sources {
		if source.Provider == nil {
			continue
		}
		filtered = append(filtered, source)
	}
	return &CombinedDataProvider{sources: filtered}
}

func (p *CombinedDataProvider) AdminProvider() contracts.AdminProvider {
	if p == nil {
		return nil
	}
	for _, source := range p.sources {
		admin, ok := source.Provider.(contracts.AdminProvider)
		if ok {
			return admin
		}
	}
	return nil
}

func (p *CombinedDataProvider) CanUseAdminForObjectID(objectID int) bool {
	source := p.sourceForObjectID(objectID)
	if source == nil {
		return false
	}
	_, ok := source.Provider.(contracts.AdminProvider)
	return ok
}

func (p *CombinedDataProvider) SourceNameForObjectID(objectID int) string {
	source := p.sourceForObjectID(objectID)
	if source == nil || strings.TrimSpace(source.Name) == "" {
		return "невідоме джерело"
	}
	return source.Name
}

func (p *CombinedDataProvider) GetObjects() []models.Object {
	objects := make([]models.Object, 0, 128)
	if p != nil {
		for _, source := range p.sources {
			objects = append(objects, source.Provider.GetObjects()...)
		}
	}

	if len(objects) == 0 {
		return nil
	}

	seen := make(map[int]struct{}, len(objects))
	deduped := objects[:0]
	for _, obj := range objects {
		if _, exists := seen[obj.ID]; exists {
			continue
		}
		seen[obj.ID] = struct{}{}
		deduped = append(deduped, obj)
	}

	sort.SliceStable(deduped, func(i, j int) bool {
		return deduped[i].ID < deduped[j].ID
	})
	return deduped
}

func (p *CombinedDataProvider) GetObjectByID(id string) *models.Object {
	if p == nil {
		return nil
	}

	provider := p.providerForObjectID(id)
	if provider == nil {
		return nil
	}
	if obj := provider.GetObjectByID(id); obj != nil {
		return obj
	}
	// Якщо маршрутизатор промахнувся, робимо fallback по всіх джерелах.
	for _, source := range p.sources {
		if source.Provider == provider {
			continue
		}
		if obj := source.Provider.GetObjectByID(id); obj != nil {
			return obj
		}
	}
	return nil
}

func (p *CombinedDataProvider) GetZones(objectID string) []models.Zone {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return nil
	}
	return provider.GetZones(objectID)
}

func (p *CombinedDataProvider) GetEmployees(objectID string) []models.Contact {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return nil
	}
	return provider.GetEmployees(objectID)
}

func (p *CombinedDataProvider) GetEvents() []models.Event {
	events := make([]models.Event, 0, 256)
	if p != nil {
		for _, source := range p.sources {
			events = append(events, source.Provider.GetEvents()...)
		}
	}
	sortEvents(events)
	return events
}

func (p *CombinedDataProvider) GetObjectEvents(objectID string) []models.Event {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return nil
	}
	events := provider.GetObjectEvents(objectID)
	sortEvents(events)
	return events
}

func (p *CombinedDataProvider) GetAlarms() []models.Alarm {
	alarms := make([]models.Alarm, 0, 64)
	if p != nil {
		for _, source := range p.sources {
			alarms = append(alarms, source.Provider.GetAlarms()...)
		}
	}
	sort.SliceStable(alarms, func(i, j int) bool {
		left := alarms[i].Time
		right := alarms[j].Time
		if left.Equal(right) {
			return alarms[i].ID > alarms[j].ID
		}
		return left.After(right)
	})
	return alarms
}

func (p *CombinedDataProvider) ProcessAlarm(id string, user string, note string) {
	if p == nil {
		return
	}

	provider := p.providerForAlarmID(id)
	if provider != nil {
		provider.ProcessAlarm(id, user, note)
		return
	}

	// Fallback: відправляємо в перше доступне джерело.
	for _, source := range p.sources {
		source.Provider.ProcessAlarm(id, user, note)
		return
	}
}

func (p *CombinedDataProvider) GetExternalData(objectID string) (signal string, testMsg string, lastTest time.Time, lastMsg time.Time) {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return "", "", time.Time{}, time.Time{}
	}
	return provider.GetExternalData(objectID)
}

func (p *CombinedDataProvider) GetTestMessages(objectID string) []models.TestMessage {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return nil
	}
	return provider.GetTestMessages(objectID)
}

func (p *CombinedDataProvider) GetLatestEventID() (int64, error) {
	if p == nil {
		return 0, errors.New("combined provider is nil")
	}

	h := fnv.New64a()
	written := false

	writePart := func(tag byte, value int64) {
		_, _ = h.Write([]byte{tag})
		_, _ = h.Write([]byte(fmt.Sprintf("%d", value)))
		_, _ = h.Write([]byte{0})
		written = true
	}

	for i, source := range p.sources {
		latest, ok := source.Provider.(latestEventIDProvider)
		if !ok {
			continue
		}
		id, err := latest.GetLatestEventID()
		if err != nil {
			continue
		}
		tag := byte('a' + (i % 26))
		if source.Name != "" {
			tag = source.Name[0]
		}
		writePart(tag, id)
	}

	if !written {
		return 0, errors.New("no latest event cursor available")
	}
	return int64(h.Sum64() & 0x7fffffffffffffff), nil
}

func (p *CombinedDataProvider) providerForObjectID(objectID string) contracts.DataProvider {
	if p == nil || len(p.sources) == 0 {
		return nil
	}
	if parsedID, ok := parseObjectID(objectID); ok {
		source := p.sourceForObjectID(parsedID)
		if source != nil {
			return source.Provider
		}
	}
	return p.sources[0].Provider
}

func (p *CombinedDataProvider) providerForAlarmID(alarmID string) contracts.DataProvider {
	if p == nil || len(p.sources) == 0 {
		return nil
	}
	if parsedID, ok := parseObjectID(alarmID); ok {
		for _, source := range p.sources {
			if source.OwnsAlarmID != nil && source.OwnsAlarmID(parsedID) {
				return source.Provider
			}
		}
	}
	return p.sources[0].Provider
}

func (p *CombinedDataProvider) sourceForObjectID(objectID int) *ProviderSource {
	if p == nil || len(p.sources) == 0 {
		return nil
	}
	for i := range p.sources {
		if p.sources[i].OwnsObjectID != nil && p.sources[i].OwnsObjectID(objectID) {
			return &p.sources[i]
		}
	}
	return &p.sources[0]
}

func parseObjectID(raw string) (int, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func sortEvents(events []models.Event) {
	sort.SliceStable(events, func(i, j int) bool {
		left := events[i].Time
		right := events[j].Time
		if left.Equal(right) {
			return events[i].ID > events[j].ID
		}
		return left.After(right)
	})
}
