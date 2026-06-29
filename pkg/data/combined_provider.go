package data

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ids"
	"obj_catalog_fyne_v3/pkg/models"
	"obj_catalog_fyne_v3/pkg/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type latestEventIDProvider interface {
	GetLatestEventID() (int64, error)
}

type lastGPRSTestTimeProvider interface {
	LastGPRSTestTime(ctx context.Context, objectID int) (time.Time, error)
}

type caslStatisticReportProvider interface {
	GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error)
}

type caslObjectEditorProvider interface {
	contracts.CASLObjectEditorProvider
}

type caslGeoZoneAccessProvider interface {
	ReadManagers(ctx context.Context, skip int, limit int) ([]map[string]any, error)
}

type alarmProcessingProvider interface {
	contracts.AlarmProcessingProvider
}

type timeoutRecoverableProvider interface {
	TriggerReconnect(reason string)
}

// Default slice capacities for pre-allocation
const (
	defaultObjectsCapacity = 128
	defaultEventsCapacity  = 256
	defaultAlarmsCapacity  = 64

	defaultCombinedProviderGetEventsTimeout = 5 * time.Second
	defaultCombinedProviderGetAlarmsTimeout = 3 * time.Second
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
	sources       []ProviderSource
	eventsTimeout time.Duration
	alarmsTimeout time.Duration
}

func (p *CombinedDataProvider) FrontendSourceCapabilities() []contracts.FrontendSourceCapability {
	if p == nil {
		return nil
	}

	capabilities := make([]contracts.FrontendSourceCapability, 0, len(p.sources))
	for _, source := range p.sources {
		frontendSource := frontendSourceFromProviderName(source.Name)
		capability := contracts.FrontendSourceCapability{
			Source:            frontendSource,
			DisplayName:       frontendSource.DisplayName(),
			ReadObjects:       true,
			ReadObjectDetails: true,
			ReadEvents:        true,
			ReadAlarms:        true,
		}

		if _, ok := source.Provider.(contracts.AdminProvider); ok {
			capability.CreateObject = true
			capability.UpdateObject = true
		}
		if _, ok := source.Provider.(contracts.CASLObjectEditorProvider); ok {
			capability.CreateObject = true
			capability.UpdateObject = true
		}
		if healthProvider, ok := source.Provider.(contracts.FrontendSourceHealthProvider); ok {
			health := healthProvider.FrontendSourceHealth()
			capability.HealthStatus = health.HealthStatus
			capability.HealthText = health.HealthText
			capability.APIStatus = health.APIStatus
			capability.RealtimeStatus = health.RealtimeStatus
			capability.LastRealtimePing = health.LastRealtimePing
		}

		capabilities = append(capabilities, capability)
	}

	return capabilities
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

func triggerProviderRecovery(source *ProviderSource, reason string) {
	if source == nil || source.Provider == nil {
		return
	}

	recoverable, ok := source.Provider.(timeoutRecoverableProvider)
	if !ok {
		return
	}

	recoverable.TriggerReconnect(reason)
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
	return &CombinedDataProvider{
		sources:       filtered,
		eventsTimeout: defaultCombinedProviderGetEventsTimeout,
		alarmsTimeout: defaultCombinedProviderGetAlarmsTimeout,
	}
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

func (p *CombinedDataProvider) GetObjectMedia(ctx context.Context, objectID int) ([]contracts.ObjectMedia, error) {
	source := p.sourceForObjectID(objectID)
	if source == nil {
		return nil, fmt.Errorf("media source for object %d was not found", objectID)
	}
	provider, ok := source.Provider.(contracts.ObjectMediaProvider)
	if !ok {
		return nil, nil
	}
	return provider.GetObjectMedia(ctx, objectID)
}

func (p *CombinedDataProvider) FetchObjectMedia(ctx context.Context, media contracts.ObjectMedia) ([]byte, error) {
	for _, source := range p.sources {
		provider, ok := source.Provider.(contracts.ObjectMediaProvider)
		if !ok {
			continue
		}
		body, err := provider.FetchObjectMedia(ctx, media)
		if err == nil {
			return body, nil
		}
	}
	return nil, fmt.Errorf("media %q was not found", media.ID)
}

func (p *CombinedDataProvider) ReadManagers(ctx context.Context, skip int, limit int) ([]map[string]any, error) {
	provider, err := p.resolveAnyCASLObjectEditorProvider()
	if err != nil {
		return nil, err
	}
	accessProvider, ok := provider.(caslGeoZoneAccessProvider)
	if !ok {
		return nil, fmt.Errorf("manager list is not supported by current CASL provider")
	}
	return accessProvider.ReadManagers(ctx, skip, limit)
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
		return combinedObjectDisplayNumber(objects[i]) < combinedObjectDisplayNumber(objects[j])
	})
	return objects
}

func (p *CombinedDataProvider) ListObjectLocations(ctx context.Context) ([]contracts.ObjectLocation, error) {
	if p == nil {
		return nil, nil
	}
	result := make([]contracts.ObjectLocation, 0)
	for _, source := range p.sources {
		if provider, ok := source.Provider.(contracts.ObjectLocationProvider); ok {
			locations, err := provider.ListObjectLocations(ctx)
			if err != nil {
				log.Warn().Err(err).Str("source", source.Name).Msg("ListObjectLocations failed")
				continue
			}
			result = append(result, locations...)
			continue
		}
		for _, object := range source.Provider.GetObjects() {
			if strings.TrimSpace(object.Latitude) == "" || strings.TrimSpace(object.Longitude) == "" {
				continue
			}
			result = append(result, contracts.ObjectLocation{
				ObjectID: object.ID, Latitude: object.Latitude, Longitude: object.Longitude,
			})
		}
	}
	return result, nil
}

func combinedObjectDisplayNumber(object models.Object) string {
	if strings.TrimSpace(object.DisplayNumber) != "" {
		return object.DisplayNumber
	}
	if !ids.IsCASLObjectID(object.ID) && !ids.IsPhoenixObjectID(object.ID) {
		return strconv.Itoa(object.ID)
	}
	if number := combinedNumberFromPanelMark(object.PanelMark); number != "" {
		return number
	}
	if number := utils.LeadingDigits(strings.TrimSpace(object.Name)); number != "" {
		return number
	}
	return strconv.Itoa(object.ID)
}

func combinedNumberFromPanelMark(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if idx := strings.LastIndex(text, "#"); idx >= 0 && idx < len(text)-1 {
		if number := utils.LeadingDigits(strings.TrimSpace(text[idx+1:])); number != "" {
			return number
		}
	}
	return utils.LeadingDigits(text)
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
	if p == nil || len(p.sources) == 0 {
		return nil
	}

	timeout := p.eventsTimeout
	if timeout <= 0 {
		timeout = defaultCombinedProviderGetEventsTimeout
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	events := make([]models.Event, 0, defaultEventsCapacity)

	for i := range p.sources {
		wg.Add(1)
		go func(src *ProviderSource) {
			defer wg.Done()

			resChan := make(chan []models.Event, 1)
			go func() {
				resChan <- src.Provider.GetEvents()
			}()

			select {
			case sourceEvents := <-resChan:
				if len(sourceEvents) > 0 {
					mu.Lock()
					events = append(events, sourceEvents...)
					mu.Unlock()
				}
			case <-time.After(timeout):
				log.Warn().Str("provider", src.Name).Msg("CombinedDataProvider: GetEvents timeout — повертаємо дані без цього джерела")
				triggerProviderRecovery(src, "combined get_events timeout")
			}
		}(&p.sources[i])
	}
	wg.Wait()

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

func (p *CombinedDataProvider) GetObjectEventsRange(objectID string, from time.Time, to time.Time) []models.Event {
	provider := p.providerForObjectID(objectID)
	if provider == nil {
		return nil
	}
	var events []models.Event
	if ranged, ok := provider.(contracts.ObjectEventsRangeProvider); ok {
		events = ranged.GetObjectEventsRange(objectID, from, to)
	} else {
		events = filterEventsByTimeRange(provider.GetObjectEvents(objectID), from, to)
	}
	sortEvents(events)
	return events
}

func filterEventsByTimeRange(events []models.Event, from time.Time, to time.Time) []models.Event {
	result := make([]models.Event, 0, len(events))
	for _, event := range events {
		if !from.IsZero() && event.Time.Before(from) {
			continue
		}
		if !to.IsZero() && event.Time.After(to) {
			continue
		}
		result = append(result, event)
	}
	return result
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
	if p == nil || len(p.sources) == 0 {
		return nil
	}

	timeout := p.alarmsTimeout
	if timeout <= 0 {
		timeout = defaultCombinedProviderGetAlarmsTimeout
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	alarms := make([]models.Alarm, 0, defaultAlarmsCapacity)

	for i := range p.sources {
		wg.Add(1)
		go func(src *ProviderSource) {
			defer wg.Done()

			resChan := make(chan []models.Alarm, 1)
			go func() {
				resChan <- src.Provider.GetAlarms()
			}()

			select {
			case sourceAlarms := <-resChan:
				if len(sourceAlarms) > 0 {
					mu.Lock()
					alarms = append(alarms, sourceAlarms...)
					mu.Unlock()
				}
			case <-time.After(timeout):
				log.Debug().Str("provider", src.Name).Msg("CombinedDataProvider: GetAlarms timeout")
				triggerProviderRecovery(src, "combined get_alarms timeout")
			}
		}(&p.sources[i])
	}
	wg.Wait()

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

func (p *CombinedDataProvider) PickAlarm(ctx context.Context, alarm models.Alarm, user string) error {
	if p == nil {
		return errors.New("combined provider is nil")
	}

	provider := p.providerForAlarmID(strconv.Itoa(alarm.ID))
	if advanced, ok := provider.(contracts.AlarmTakeoverProvider); ok {
		return advanced.PickAlarm(ctx, alarm, user)
	}
	return errors.New("alarm takeover provider is not configured")
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

func (p *CombinedDataProvider) LastGPRSTestTime(ctx context.Context, objectID int) (time.Time, error) {
	provider := p.providerForObjectID(strconv.Itoa(objectID))
	if provider == nil {
		return time.Time{}, errors.New("last GPRS test provider is not configured")
	}
	lastTestProvider, ok := provider.(lastGPRSTestTimeProvider)
	if !ok {
		return time.Time{}, fmt.Errorf("last GPRS test is not supported by provider for object %d", objectID)
	}
	return lastTestProvider.LastGPRSTestTime(ctx, objectID)
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

// ListResponseGroups implements contracts.ResponseGroupProvider.
// Aggregates response groups from all sources that support the interface.
func (p *CombinedDataProvider) ListResponseGroups(ctx context.Context) ([]contracts.ResponseGroup, error) {
	if p == nil {
		return nil, errors.New("combined provider is nil")
	}
	var result []contracts.ResponseGroup
	for _, source := range p.sources {
		rgp, ok := source.Provider.(contracts.ResponseGroupProvider)
		if !ok {
			continue
		}
		groups, err := rgp.ListResponseGroups(ctx)
		if err != nil {
			log.Warn().Err(err).Str("source", source.Name).Msg("ListResponseGroups failed")
			continue
		}
		sourceType := frontendSourceFromProviderName(source.Name)
		for i := range groups {
			if groups[i].Source == contracts.FrontendSourceUnknown || groups[i].Source == "" {
				groups[i].Source = sourceType
			}
		}
		result = append(result, groups...)
	}
	return result, nil
}

// AssignResponseGroup implements contracts.ResponseGroupProvider.
func (p *CombinedDataProvider) AssignResponseGroup(ctx context.Context, alarm models.Alarm, groupID string) error {
	src := p.sourceForObjectID(alarm.ObjectID)
	if src == nil {
		return errors.New("no source available for alarm")
	}
	rgp, ok := src.Provider.(contracts.ResponseGroupProvider)
	if !ok {
		return fmt.Errorf("assign response group not supported for source %s", src.Name)
	}
	return rgp.AssignResponseGroup(ctx, alarm, groupID)
}

// NotifyGroupArrived implements contracts.ResponseGroupProvider.
func (p *CombinedDataProvider) NotifyGroupArrived(ctx context.Context, alarm models.Alarm) error {
	src := p.sourceForObjectID(alarm.ObjectID)
	if src == nil {
		return errors.New("no source available for alarm")
	}
	rgp, ok := src.Provider.(contracts.ResponseGroupProvider)
	if !ok {
		return fmt.Errorf("notify group arrived not supported for source %s", src.Name)
	}
	return rgp.NotifyGroupArrived(ctx, alarm)
}

// CancelResponseGroup implements contracts.ResponseGroupProvider.
func (p *CombinedDataProvider) CancelResponseGroup(ctx context.Context, alarm models.Alarm) error {
	src := p.sourceForObjectID(alarm.ObjectID)
	if src == nil {
		return errors.New("no source available for alarm")
	}
	rgp, ok := src.Provider.(contracts.ResponseGroupProvider)
	if !ok {
		return fmt.Errorf("cancel response group not supported for source %s", src.Name)
	}
	return rgp.CancelResponseGroup(ctx, alarm)
}

// GroupProcessAlarm implements contracts.AlarmGroupProcessProvider.
// Routes to the source that owns the alarm by objectID.
func (p *CombinedDataProvider) GroupProcessAlarm(ctx context.Context, alarm models.Alarm, user string) error {
	src := p.sourceForObjectID(alarm.ObjectID)
	if src == nil {
		return errors.New("no source available for alarm")
	}
	agp, ok := src.Provider.(contracts.AlarmGroupProcessProvider)
	if !ok {
		return fmt.Errorf("group process alarm not supported for source %s", src.Name)
	}
	return agp.GroupProcessAlarm(ctx, alarm, user)
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

// StandbyCASLObject реалізує caslStandbyCapable для CombinedDataProvider.
func (p *CombinedDataProvider) StandbyCASLObject(ctx context.Context, internalID int, req contracts.FrontendStandbyRequest) error {
	source := p.sourceForObjectID(internalID)
	if source == nil {
		return fmt.Errorf("casl standby: джерело для об'єкта %d не знайдено", internalID)
	}
	type standbyCapable interface {
		StandbyCASLObject(ctx context.Context, internalID int, req contracts.FrontendStandbyRequest) error
	}
	capable, ok := source.Provider.(standbyCapable)
	if !ok {
		return fmt.Errorf("casl standby: джерело %q не підтримує стенди", source.Name)
	}
	return capable.StandbyCASLObject(ctx, internalID, req)
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

func frontendSourceFromProviderName(name string) contracts.FrontendSource {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "bridge", "db", "firebird", "most":
		return contracts.FrontendSourceBridge
	case "phoenix":
		return contracts.FrontendSourcePhoenix
	case "casl":
		return contracts.FrontendSourceCASL
	default:
		return contracts.FrontendSourceUnknown
	}
}

// GenerateAcceptedObjectsReport delegates accepted objects Excel reporting to the first underlying source provider that supports it
func (p *CombinedDataProvider) GenerateAcceptedObjectsReport(filePath string) error {
	for _, source := range p.sources {
		if reporter, ok := source.Provider.(contracts.ExcelReportingProvider); ok {
			return reporter.GenerateAcceptedObjectsReport(filePath)
		}
	}
	return fmt.Errorf("no source supports accepted objects report generation")
}

// AppendObjectToDeletedReport delegates deleted objects Excel reporting to the first underlying source provider that supports it
func (p *CombinedDataProvider) AppendObjectToDeletedReport(obj *models.Object, contacts []models.Contact, pdfFilePath string, filePath string) error {
	for _, source := range p.sources {
		if reporter, ok := source.Provider.(contracts.ExcelReportingProvider); ok {
			return reporter.AppendObjectToDeletedReport(obj, contacts, pdfFilePath, filePath)
		}
	}
	return fmt.Errorf("no source supports appending object to deleted report")
}
