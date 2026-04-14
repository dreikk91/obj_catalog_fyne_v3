package data

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/ui/viewmodels"
	"sort"
	"strconv"
	"strings"
	"time"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

type caslStatisticReportProvider interface {
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
}

type caslObjectEditorProvider interface {
	contracts.CASLObjectEditorProvider
}

type alarmProcessingProvider interface {
	contracts.AlarmProcessingProvider
}

// Default slice capacities for pre-allocation
const (
	defaultObjectsCapacity = 128
	defaultEventsCapacity  = 256
	defaultAlarmsCapacity  = 64
)

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
			OwnsObjectID: ids.IsCASLObjectID,
			OwnsAlarmID:  ids.IsCASLObjectID,
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

func (p *CombinedDataProvider) Shutdown() {
	if p == nil {
		return
	}
	for _, source := range p.sources {
		shutdowner, ok := source.Provider.(contracts.ShutdownProvider)
		if !ok || shutdowner == nil {
			continue
		}
		shutdowner.Shutdown()
	}
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

func (p *CombinedDataProvider) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	if p == nil {
		return nil, errors.New("combined provider is nil")
	}
	for _, source := range p.sources {
		reporter, ok := source.Provider.(caslStatisticReportProvider)
		if !ok {
			continue
		}
		return reporter.GetStatisticReport(ctx, name, limit)
	}
	return nil, errors.New("casl reports provider is not configured")
}

func (p *CombinedDataProvider) GetCASLObjectEditorSnapshot(ctx context.Context, objectID int64) (contracts.CASLObjectEditorSnapshot, error) {
	provider, err := p.resolveCASLObjectEditorProvider(objectID)
	if err != nil {
		return contracts.CASLObjectEditorSnapshot{}, err
	}
	return provider.GetCASLObjectEditorSnapshot(ctx, objectID)
}

func (p *CombinedDataProvider) CreateCASLObject(ctx context.Context, create contracts.CASLGuardObjectCreate) (string, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return "", err
	}
	return provider.CreateCASLObject(ctx, create)
}

func (p *CombinedDataProvider) UpdateCASLObject(ctx context.Context, update contracts.CASLGuardObjectUpdate) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(update.ObjID))
	if err != nil {
		return err
	}
	return provider.UpdateCASLObject(ctx, update)
}

func (p *CombinedDataProvider) DeleteCASLObject(ctx context.Context, objectID int64) error {
	provider, err := p.resolveCASLObjectEditorProvider(objectID)
	if err != nil {
		return err
	}
	return provider.DeleteCASLObject(ctx, objectID)
}

func (p *CombinedDataProvider) UpdateCASLRoom(ctx context.Context, update contracts.CASLRoomUpdate) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(update.ObjID))
	if err != nil {
		return err
	}
	return provider.UpdateCASLRoom(ctx, update)
}

func (p *CombinedDataProvider) CreateCASLRoom(ctx context.Context, create contracts.CASLRoomCreate) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(create.ObjID))
	if err != nil {
		return err
	}
	return provider.CreateCASLRoom(ctx, create)
}

func (p *CombinedDataProvider) ReadCASLDeviceNumbers(ctx context.Context) ([]int64, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return nil, err
	}
	return provider.ReadCASLDeviceNumbers(ctx)
}

func (p *CombinedDataProvider) IsCASLDeviceNumberInUse(ctx context.Context, deviceNumber int64) (bool, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return false, err
	}
	return provider.IsCASLDeviceNumberInUse(ctx, deviceNumber)
}

func (p *CombinedDataProvider) CreateCASLDevice(ctx context.Context, create contracts.CASLDeviceCreate) (string, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return "", err
	}
	return provider.CreateCASLDevice(ctx, create)
}

func (p *CombinedDataProvider) UpdateCASLDevice(ctx context.Context, update contracts.CASLDeviceUpdate) error {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return err
	}
	return provider.UpdateCASLDevice(ctx, update)
}

func (p *CombinedDataProvider) BlockCASLDevice(ctx context.Context, request contracts.CASLDeviceBlockRequest) error {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return err
	}
	return provider.BlockCASLDevice(ctx, request)
}

func (p *CombinedDataProvider) UnblockCASLDevice(ctx context.Context, deviceID string) error {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return err
	}
	return provider.UnblockCASLDevice(ctx, deviceID)
}

func (p *CombinedDataProvider) UpdateCASLDeviceLine(ctx context.Context, update contracts.CASLDeviceLineMutation) error {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return err
	}
	return provider.UpdateCASLDeviceLine(ctx, update)
}

func (p *CombinedDataProvider) CreateCASLDeviceLine(ctx context.Context, create contracts.CASLDeviceLineMutation) error {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return err
	}
	return provider.CreateCASLDeviceLine(ctx, create)
}

func (p *CombinedDataProvider) AddCASLLineToRoom(ctx context.Context, binding contracts.CASLLineToRoomBinding) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(binding.ObjID))
	if err != nil {
		return err
	}
	return provider.AddCASLLineToRoom(ctx, binding)
}

func (p *CombinedDataProvider) AddCASLUserToRoom(ctx context.Context, request contracts.CASLAddUserToRoomRequest) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(request.ObjID))
	if err != nil {
		return err
	}
	return provider.AddCASLUserToRoom(ctx, request)
}

func (p *CombinedDataProvider) RemoveCASLUserFromRoom(ctx context.Context, request contracts.CASLRemoveUserFromRoomRequest) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(request.ObjID))
	if err != nil {
		return err
	}
	return provider.RemoveCASLUserFromRoom(ctx, request)
}

func (p *CombinedDataProvider) UpdateCASLRoomUserPriorities(ctx context.Context, objectID int64, items []contracts.CASLRoomUserPriority) error {
	provider, err := p.resolveCASLObjectEditorProvider(objectID)
	if err != nil {
		return err
	}
	return provider.UpdateCASLRoomUserPriorities(ctx, objectID, items)
}

func (p *CombinedDataProvider) CreateCASLUser(ctx context.Context, request contracts.CASLUserCreateRequest) (contracts.CASLUserProfile, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return contracts.CASLUserProfile{}, err
	}
	return provider.CreateCASLUser(ctx, request)
}

func (p *CombinedDataProvider) ReadCASLObjectBasket(ctx context.Context) ([]contracts.CASLObjectBasketItem, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return nil, err
	}
	return provider.ReadCASLObjectBasket(ctx)
}

func (p *CombinedDataProvider) CreateCASLImage(ctx context.Context, request contracts.CASLImageCreateRequest) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(request.ObjID))
	if err != nil {
		return err
	}
	return provider.CreateCASLImage(ctx, request)
}

func (p *CombinedDataProvider) DeleteCASLImage(ctx context.Context, request contracts.CASLImageDeleteRequest) error {
	provider, err := p.resolveCASLObjectEditorProvider(parseCASLMutationObjectID(request.ObjID))
	if err != nil {
		return err
	}
	return provider.DeleteCASLImage(ctx, request)
}

func (p *CombinedDataProvider) FetchCASLImagePreview(ctx context.Context, imageID string) ([]byte, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return nil, err
	}
	return provider.FetchCASLImagePreview(ctx, imageID)
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
	objects := make([]models.Object, 0, defaultObjectsCapacity)
	if p != nil {
		for _, source := range p.sources {
			objects = append(objects, source.Provider.GetObjects()...)
		}
	}

	if len(objects) == 0 {
		return nil
	}

	sort.SliceStable(objects, func(i, j int) bool {
		return viewmodels.ObjectDisplayNumber(objects[i]) < viewmodels.ObjectDisplayNumber(objects[j])
	})
	return objects
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
	events := make([]models.Event, 0, defaultEventsCapacity)
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

func (p *CombinedDataProvider) GetAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	if p == nil {
		return nil
	}

	provider := p.providerForObjectID(strconv.Itoa(alarm.ObjectID))
	if historyProvider, ok := provider.(contracts.AlarmHistoryProvider); ok {
		return historyProvider.GetAlarmSourceMessages(alarm)
	}

	for _, source := range p.sources {
		historyProvider, ok := source.Provider.(contracts.AlarmHistoryProvider)
		if !ok {
			continue
		}
		if msgs := historyProvider.GetAlarmSourceMessages(alarm); len(msgs) > 0 {
			return msgs
		}
	}

	return nil
}

func (p *CombinedDataProvider) GetActiveAlarmSourceMessages(alarm models.Alarm) []models.AlarmMsg {
	if p == nil {
		return nil
	}

	provider := p.providerForObjectID(strconv.Itoa(alarm.ObjectID))
	if historyProvider, ok := provider.(contracts.ActiveAlarmHistoryProvider); ok {
		return historyProvider.GetActiveAlarmSourceMessages(alarm)
	}

	for _, source := range p.sources {
		historyProvider, ok := source.Provider.(contracts.ActiveAlarmHistoryProvider)
		if !ok {
			continue
		}
		if msgs := historyProvider.GetActiveAlarmSourceMessages(alarm); len(msgs) > 0 {
			return msgs
		}
	}

	return nil
}

func (p *CombinedDataProvider) GetAlarms() []models.Alarm {
	alarms := make([]models.Alarm, 0, defaultAlarmsCapacity)
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

func (p *CombinedDataProvider) ProcessAlarm(id string, user string, note string) error {
	if p == nil {
		return errors.New("combined provider is nil")
	}

	provider := p.providerForAlarmID(id)
	if provider != nil {
		return provider.ProcessAlarm(id, user, note)
	}

	// Fallback: відправляємо в перше доступне джерело.
	for _, source := range p.sources {
		return source.Provider.ProcessAlarm(id, user, note)
	}
	return errors.New("no data source available to process alarm")
}

func (p *CombinedDataProvider) GetAlarmProcessingOptions(ctx context.Context, alarm models.Alarm) ([]contracts.AlarmProcessingOption, error) {
	if p == nil {
		return nil, errors.New("combined provider is nil")
	}

	provider := p.providerForAlarmID(strconv.Itoa(alarm.ID))
	if advanced, ok := provider.(alarmProcessingProvider); ok {
		return advanced.GetAlarmProcessingOptions(ctx, alarm)
	}
	return nil, nil
}

func (p *CombinedDataProvider) ProcessAlarmWithRequest(ctx context.Context, alarm models.Alarm, user string, request contracts.AlarmProcessingRequest) error {
	if p == nil {
		return errors.New("combined provider is nil")
	}

	provider := p.providerForAlarmID(strconv.Itoa(alarm.ID))
	if advanced, ok := provider.(alarmProcessingProvider); ok {
		return advanced.ProcessAlarmWithRequest(ctx, alarm, user, request)
	}
	if provider != nil {
		return provider.ProcessAlarm(strconv.Itoa(alarm.ID), user, request.Note)
	}
	return errors.New("alarm provider is not configured")
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

func parseCASLMutationObjectID(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func (p *CombinedDataProvider) resolveCASLObjectEditorProvider(objectID int64) (caslObjectEditorProvider, error) {
	if objectID > 0 {
		if source := p.sourceForObjectID(int(objectID)); source != nil {
			if provider, ok := source.Provider.(caslObjectEditorProvider); ok {
				return provider, nil
			}
		}
	}
	return p.resolveAnyCASLObjectEditorProvider()
}

func (p *CombinedDataProvider) resolveAnyCASLObjectEditorProvider() (caslObjectEditorProvider, error) {
	if p == nil {
		return nil, errors.New("combined provider is nil")
	}
	for _, source := range p.sources {
		provider, ok := source.Provider.(caslObjectEditorProvider)
		if ok {
			return provider, nil
		}
	}
	return nil, errors.New("casl object editor provider is not configured")
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
