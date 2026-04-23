package backend

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

var legacyFrontendAlarmProcessingOptions = []contracts.FrontendAlarmProcessingOption{
	{Code: "false_alarm", Label: "Помилкова тривога"},
	{Code: "firefighters", Label: "Виклик пожежників"},
	{Code: "response_team", Label: "Виклик ГШР"},
	{Code: "technical_fault", Label: "Технічна несправність"},
	{Code: "control_check", Label: "Контрольна перевірка"},
	{Code: "other", Label: "Інше"},
}

type FrontendAdminObjectMutator interface {
	GetObjectCard(objn int64) (contracts.AdminObjectCard, error)
	CreateObject(card contracts.AdminObjectCard) error
	UpdateObject(card contracts.AdminObjectCard) error
}

type FrontendCASLObjectMutator interface {
	GetCASLObjectEditorSnapshot(ctx context.Context, objectID int64) (contracts.CASLObjectEditorSnapshot, error)
	CreateCASLObject(ctx context.Context, create contracts.CASLGuardObjectCreate) (string, error)
	UpdateCASLObject(ctx context.Context, update contracts.CASLGuardObjectUpdate) error
}

type FrontendSourceCapabilityProvider interface {
	FrontendSourceCapabilities() []contracts.FrontendSourceCapability
}

type FrontendAdapterOption func(*FrontendAdapter)

type FrontendAdapter struct {
	dataProvider       contracts.DataProvider
	adminMutator       FrontendAdminObjectMutator
	caslMutator        FrontendCASLObjectMutator
	capabilityProvider FrontendSourceCapabilityProvider

	// pickedAlarmsMu guards pickedAlarmIDs for Bridge alarms picked locally (no DB state).
	pickedAlarmsMu sync.Mutex
	pickedAlarmIDs map[int]bool
}

func WithFrontendAdminObjectMutator(mutator FrontendAdminObjectMutator) FrontendAdapterOption {
	return func(adapter *FrontendAdapter) {
		if adapter != nil {
			adapter.adminMutator = mutator
		}
	}
}

func WithFrontendCASLObjectMutator(mutator FrontendCASLObjectMutator) FrontendAdapterOption {
	return func(adapter *FrontendAdapter) {
		if adapter != nil {
			adapter.caslMutator = mutator
		}
	}
}

func WithFrontendSourceCapabilityProvider(provider FrontendSourceCapabilityProvider) FrontendAdapterOption {
	return func(adapter *FrontendAdapter) {
		if adapter != nil {
			adapter.capabilityProvider = provider
		}
	}
}

func NewFrontendAdapter(dataProvider contracts.DataProvider, opts ...FrontendAdapterOption) *FrontendAdapter {
	adapter := &FrontendAdapter{
		dataProvider: dataProvider,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(adapter)
		}
	}
	if adapter.adminMutator == nil {
		if admin, ok := AsAdminProvider(dataProvider); ok {
			adapter.adminMutator = admin
		}
	}
	if adapter.caslMutator == nil {
		if casl, ok := dataProvider.(FrontendCASLObjectMutator); ok {
			adapter.caslMutator = casl
		}
	}
	if adapter.capabilityProvider == nil {
		if provider, ok := dataProvider.(FrontendSourceCapabilityProvider); ok {
			adapter.capabilityProvider = provider
		}
	}
	return adapter
}

func (a *FrontendAdapter) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	if a == nil || a.dataProvider == nil {
		return contracts.FrontendCapabilities{}, contracts.ErrFrontendBackendUnavailable
	}
	if a.capabilityProvider != nil {
		return contracts.FrontendCapabilities{
			Sources: slices.Clone(a.capabilityProvider.FrontendSourceCapabilities()),
		}, nil
	}
	return contracts.FrontendCapabilities{Sources: a.fallbackCapabilities()}, nil
}

func (a *FrontendAdapter) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	objects := a.dataProvider.GetObjects()
	result := make([]contracts.FrontendObjectSummary, 0, len(objects))
	for _, object := range objects {
		result = append(result, mapFrontendObjectSummary(object))
	}
	return result, nil
}

func (a *FrontendAdapter) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	alarms := a.dataProvider.GetAlarms()
	var result []contracts.FrontendAlarmItem
	for _, alarm := range alarms {
		isPickedLocally := a.isBridgeAlarmPickedLocally(alarm.ID)

		if len(alarm.SourceMsgs) > 0 {
			for idx, msg := range alarm.SourceMsgs {
				item := mapFrontendAlarmItem(alarm)
				item.ID = item.ID*1000 + idx
				item.Time = msg.Time
				item.TypeCode = msg.Code
				typeText := strings.TrimSpace(msg.Details)
				if typeText == "" {
					typeText = msg.Code
				}
				item.TypeText = typeText
				if msg.Details != "" {
					item.Details = strings.TrimSpace(msg.Details)
				}
				if msg.Number > 0 {
					item.ZoneNumber = msg.Number
					if alarm.ZoneNumber != msg.Number {
						item.ZoneName = ""
					}
				} else {
					item.ZoneNumber = 0
					item.ZoneName = ""
				}
				item.VisualSeverity = frontendAlarmSeverityFromMsg(msg)

				if !item.IsOwnedByMe && isPickedLocally {
					item.IsOwnedByMe = true
					item.IsInProgress = true
					item.CanProcess = true
				}
				result = append(result, item)
			}
		} else {
			item := mapFrontendAlarmItem(alarm)
			if !item.IsOwnedByMe && isPickedLocally {
				item.IsOwnedByMe = true
				item.IsInProgress = true
				item.CanProcess = true
			}
			result = append(result, item)
		}
	}
	return result, nil
}

func (a *FrontendAdapter) isBridgeAlarmPickedLocally(alarmID int) bool {
	if a == nil {
		return false
	}
	a.pickedAlarmsMu.Lock()
	defer a.pickedAlarmsMu.Unlock()
	return a.pickedAlarmIDs[alarmID]
}

func (a *FrontendAdapter) setBridgeAlarmPicked(alarmID int, picked bool) {
	if a == nil {
		return
	}
	a.pickedAlarmsMu.Lock()
	defer a.pickedAlarmsMu.Unlock()
	if picked {
		if a.pickedAlarmIDs == nil {
			a.pickedAlarmIDs = make(map[int]bool)
		}
		a.pickedAlarmIDs[alarmID] = true
	} else {
		delete(a.pickedAlarmIDs, alarmID)
	}
}

func (a *FrontendAdapter) GetAlarmProcessingOptions(ctx context.Context, alarmID int) ([]contracts.FrontendAlarmProcessingOption, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return nil, err
	}

	if advanced, ok := a.dataProvider.(contracts.AlarmProcessingProvider); ok {
		options, err := advanced.GetAlarmProcessingOptions(ctx, alarm)
		if err != nil {
			return nil, err
		}
		if len(options) > 0 {
			result := make([]contracts.FrontendAlarmProcessingOption, 0, len(options))
			for _, item := range options {
				result = append(result, contracts.FrontendAlarmProcessingOption{
					Code:  strings.TrimSpace(item.Code),
					Label: strings.TrimSpace(item.Label),
				})
			}
			return result, nil
		}
	}

	return append([]contracts.FrontendAlarmProcessingOption(nil), legacyFrontendAlarmProcessingOptions...), nil
}

func (a *FrontendAdapter) PickAlarm(ctx context.Context, alarmID int, request contracts.FrontendAlarmPickRequest) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}

	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)
	if advanced, ok := a.dataProvider.(contracts.AlarmTakeoverProvider); ok {
		if err := advanced.PickAlarm(ctx, alarm, strings.TrimSpace(request.User)); err != nil {
			return err
		}
		if source == contracts.FrontendSourceBridge {
			a.setBridgeAlarmPicked(alarmID, true)
		}
		return nil
	}
	return fmt.Errorf("alarm pick is not supported for source %s", source)
}

func (a *FrontendAdapter) ProcessAlarm(ctx context.Context, alarmID int, request contracts.FrontendAlarmProcessRequest) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}

	user := strings.TrimSpace(request.User)
	note := strings.TrimSpace(request.Note)
	if advanced, ok := a.dataProvider.(contracts.AlarmProcessingProvider); ok {
		if err := advanced.ProcessAlarmWithRequest(ctx, alarm, user, contracts.AlarmProcessingRequest{
			CauseCode: strings.TrimSpace(request.CauseCode),
			Note:      note,
		}); err != nil {
			return err
		}
		a.setBridgeAlarmPicked(alarmID, false)
		return nil
	}

	return a.dataProvider.ProcessAlarm(strconv.Itoa(alarm.ID), user, note)
}

func (a *FrontendAdapter) GroupProcessAlarm(ctx context.Context, alarmID int, user string) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}

	if provider, ok := a.dataProvider.(contracts.AlarmGroupProcessProvider); ok {
		if err := provider.GroupProcessAlarm(ctx, alarm, strings.TrimSpace(user)); err != nil {
			return err
		}
		a.setBridgeAlarmPicked(alarmID, false)
		return nil
	}
	return fmt.Errorf("групове завершення не підтримується для джерела %s", contracts.DetectFrontendSourceByObjectID(alarm.ObjectID))
}

func (a *FrontendAdapter) ListAlarmProcessingOptionsCached(ctx context.Context) ([]contracts.FrontendAlarmProcessingOption, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	if advanced, ok := a.dataProvider.(contracts.AlarmProcessingProvider); ok {
		options, err := advanced.GetAlarmProcessingOptions(ctx, models.Alarm{})
		if err != nil {
			return nil, err
		}
		result := make([]contracts.FrontendAlarmProcessingOption, 0, len(options))
		for _, item := range options {
			result = append(result, contracts.FrontendAlarmProcessingOption{
				Code:  strings.TrimSpace(item.Code),
				Label: strings.TrimSpace(item.Label),
			})
		}
		if len(result) > 0 {
			return result, nil
		}
	}
	return append([]contracts.FrontendAlarmProcessingOption(nil), legacyFrontendAlarmProcessingOptions...), nil
}

type caslStandbyCapable interface {
	StandbyCASLObject(ctx context.Context, internalID int, req contracts.FrontendStandbyRequest) error
}

func (a *FrontendAdapter) StandbyObject(ctx context.Context, objectID int, request contracts.FrontendStandbyRequest) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	if objectID <= 0 {
		return fmt.Errorf("невірний ID об'єкта")
	}
	source := contracts.DetectFrontendSourceByObjectID(objectID)
	switch source {
	case contracts.FrontendSourceBridge:
		if admin, ok := AsAdminProvider(a.dataProvider); ok {
			return admin.SetDisplayBlockMode(int64(objectID), contracts.DisplayBlockDebug)
		}
	case contracts.FrontendSourceCASL:
		if standby, ok := a.dataProvider.(caslStandbyCapable); ok {
			return standby.StandbyCASLObject(ctx, objectID, request)
		}
	}
	return fmt.Errorf("переведення в стенди не підтримується для джерела %s", source)
}

func (a *FrontendAdapter) ListResponseGroups(ctx context.Context) ([]contracts.FrontendResponseGroup, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	if provider, ok := a.dataProvider.(contracts.ResponseGroupProvider); ok {
		groups, err := provider.ListResponseGroups(ctx)
		if err != nil {
			return nil, err
		}
		result := make([]contracts.FrontendResponseGroup, 0, len(groups))
		for _, g := range groups {
			result = append(result, contracts.FrontendResponseGroup{
				ID:       strings.TrimSpace(g.ID),
				Name:     strings.TrimSpace(g.Name),
				Callsign: strings.TrimSpace(g.Callsign),
				Phone:    strings.TrimSpace(g.Phone),
			})
		}
		return result, nil
	}
	return []contracts.FrontendResponseGroup{}, nil
}

func (a *FrontendAdapter) AssignResponseGroup(ctx context.Context, alarmID int, request contracts.FrontendAlarmGroupActionRequest) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}
	if provider, ok := a.dataProvider.(contracts.ResponseGroupProvider); ok {
		return provider.AssignResponseGroup(ctx, alarm, strings.TrimSpace(request.GroupID))
	}
	return fmt.Errorf("assign response group is not supported for source %s", contracts.DetectFrontendSourceByObjectID(alarm.ObjectID))
}

func (a *FrontendAdapter) NotifyGroupArrived(ctx context.Context, alarmID int) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}
	if provider, ok := a.dataProvider.(contracts.ResponseGroupProvider); ok {
		return provider.NotifyGroupArrived(ctx, alarm)
	}
	return fmt.Errorf("notify group arrived is not supported for source %s", contracts.DetectFrontendSourceByObjectID(alarm.ObjectID))
}

func (a *FrontendAdapter) CancelResponseGroup(ctx context.Context, alarmID int) error {
	if a == nil || a.dataProvider == nil {
		return contracts.ErrFrontendBackendUnavailable
	}
	alarm, err := a.resolveAlarmByID(alarmID)
	if err != nil {
		return err
	}
	if provider, ok := a.dataProvider.(contracts.ResponseGroupProvider); ok {
		return provider.CancelResponseGroup(ctx, alarm)
	}
	return fmt.Errorf("cancel response group is not supported for source %s", contracts.DetectFrontendSourceByObjectID(alarm.ObjectID))
}

func (a *FrontendAdapter) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	if a == nil || a.dataProvider == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	events := a.dataProvider.GetEvents()
	result := make([]contracts.FrontendEventItem, 0, len(events))
	for _, event := range events {
		result = append(result, mapFrontendEventItem(event))
	}
	return result, nil
}

func (a *FrontendAdapter) resolveAlarmByID(alarmID int) (models.Alarm, error) {
	if alarmID <= 0 {
		return models.Alarm{}, fmt.Errorf("invalid alarm id")
	}

	alarms := a.dataProvider.GetAlarms()
	for _, alarm := range alarms {
		if alarm.ID == alarmID {
			return alarm, nil
		}
	}
	// Compound ID: frontend encodes source-message sub-items as alarmID*1000+idx.
	// Try the base alarm ID if exact match was not found.
	if alarmID >= 1000 {
		baseID := alarmID / 1000
		for _, alarm := range alarms {
			if alarm.ID == baseID {
				return alarm, nil
			}
		}
	}
	return models.Alarm{}, fmt.Errorf("alarm #%d not found", alarmID)
}

func (a *FrontendAdapter) ListObjectEvents(ctx context.Context, objectID int, offset int, limit int) (contracts.FrontendEventPage, error) {
	if a == nil || a.dataProvider == nil {
		return contracts.FrontendEventPage{}, contracts.ErrFrontendBackendUnavailable
	}
	if objectID <= 0 {
		return contracts.FrontendEventPage{}, fmt.Errorf("invalid object id")
	}
	if offset < 0 {
		return contracts.FrontendEventPage{}, fmt.Errorf("invalid object events offset")
	}
	if limit <= 0 {
		return contracts.FrontendEventPage{}, fmt.Errorf("invalid object events limit")
	}

	rawID := strconv.Itoa(objectID)
	object := a.dataProvider.GetObjectByID(rawID)
	if object == nil {
		return contracts.FrontendEventPage{}, fmt.Errorf("object #%d not found", objectID)
	}

	_ = ctx
	items := mapFrontendEvents(a.dataProvider.GetObjectEvents(rawID))
	sortFrontendEventsDesc(items)

	totalCount := len(items)
	if offset >= totalCount {
		return contracts.FrontendEventPage{
			Items:      []contracts.FrontendEventItem{},
			TotalCount: totalCount,
			HasMore:    false,
		}, nil
	}

	end := offset + limit
	if end > totalCount {
		end = totalCount
	}

	pageItems := append([]contracts.FrontendEventItem(nil), items[offset:end]...)
	return contracts.FrontendEventPage{
		Items:      pageItems,
		TotalCount: totalCount,
		HasMore:    end < totalCount,
	}, nil
}

func (a *FrontendAdapter) GetObjectDetails(ctx context.Context, objectID int) (contracts.FrontendObjectDetails, error) {
	if a == nil || a.dataProvider == nil {
		return contracts.FrontendObjectDetails{}, contracts.ErrFrontendBackendUnavailable
	}
	if objectID <= 0 {
		return contracts.FrontendObjectDetails{}, fmt.Errorf("invalid object id")
	}

	rawID := strconv.Itoa(objectID)
	object := a.dataProvider.GetObjectByID(rawID)
	if object == nil {
		return contracts.FrontendObjectDetails{}, fmt.Errorf("object #%d not found", objectID)
	}

	summary := mapFrontendObjectSummary(*object)
	nativeID := a.resolveNativeIDForObject(ctx, objectID, *object)
	if nativeID != "" {
		summary.NativeID = nativeID
	}

	signal, testMessage, lastTest, lastMessage := a.dataProvider.GetExternalData(rawID)
	// Merge external timestamps into summary so Bridge/CASL test times are visible.
	if !lastTest.IsZero() && lastTest.After(summary.LastTestTime) {
		summary.LastTestTime = lastTest
	}
	if !lastMessage.IsZero() && lastMessage.After(summary.LastMessageTime) {
		summary.LastMessageTime = lastMessage
	}
	details := contracts.FrontendObjectDetails{
		Summary:                    summary,
		GSMLevel:                   object.GSMLevel,
		PowerSource:                frontendPowerSource(object.PowerSource),
		AutoTestHours:              object.AutoTestHours,
		SubServerA:                 object.SubServerA,
		SubServerB:                 object.SubServerB,
		ChannelCode:                object.ObjChan,
		AKBState:                   object.AkbState,
		PowerFault:                 object.PowerFault,
		TestControl:                object.TestControl > 0,
		TestIntervalMin:            object.TestTime,
		Phones:                     object.Phones1,
		Description:                object.Description1,
		Notes:                      object.Notes1,
		Location:                   object.Location1,
		LaunchDate:                 object.LaunchDate,
		PreferredResponseGroupID:   strings.TrimSpace(object.PreferredResponseGroupID),
		PreferredResponseGroupName: strings.TrimSpace(object.PreferredResponseGroupName),
		ExternalSignal:             signal,
		ExternalTestMessage:        testMessage,
		ExternalLastTest:           lastTest,
		ExternalLastMessage:        lastMessage,
		Zones:                      mapFrontendZones(a.dataProvider.GetZones(rawID)),
		Contacts:                   mapFrontendContacts(a.dataProvider.GetEmployees(rawID)),
		Events:                     mapFrontendEvents(a.dataProvider.GetObjectEvents(rawID)),
	}
	sortFrontendEventsDesc(details.Events)

	return details, nil
}

func sortFrontendEventsDesc(items []contracts.FrontendEventItem) {
	slices.SortStableFunc(items, func(left, right contracts.FrontendEventItem) int {
		switch {
		case left.Time.After(right.Time):
			return -1
		case left.Time.Before(right.Time):
			return 1
		case left.ID > right.ID:
			return -1
		case left.ID < right.ID:
			return 1
		default:
			return 0
		}
	})
}

func (a *FrontendAdapter) CreateObject(ctx context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	source := resolveFrontendMutationSource(request)
	switch source {
	case contracts.FrontendSourceBridge:
		if a.adminMutator == nil {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
		}
		card, err := buildLegacyAdminObjectCard(contracts.AdminObjectCard{}, request, false)
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		if err := a.adminMutator.CreateObject(card); err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		return contracts.FrontendObjectMutationResult{
			Source:   source,
			ObjectID: int(card.ObjN),
			NativeID: strconv.FormatInt(card.ObjN, 10),
		}, nil
	case contracts.FrontendSourceCASL:
		if a.caslMutator == nil {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
		}
		create, err := buildCASLObjectCreate(request)
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		nativeID, err := a.caslMutator.CreateCASLObject(ctx, create)
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		return contracts.FrontendObjectMutationResult{
			Source:   source,
			NativeID: strings.TrimSpace(nativeID),
		}, nil
	default:
		return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
	}
}

func (a *FrontendAdapter) UpdateObject(ctx context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	source := resolveFrontendMutationSource(request)
	switch source {
	case contracts.FrontendSourceBridge:
		if a.adminMutator == nil {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
		}
		if request.ObjectID <= 0 && (request.Legacy == nil || request.Legacy.ObjN <= 0) {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("legacy update requires object id")
		}
		base := contracts.AdminObjectCard{}
		baseObjN := int64(request.ObjectID)
		if request.Legacy != nil && request.Legacy.ObjN > 0 {
			baseObjN = request.Legacy.ObjN
		}
		if baseObjN > 0 {
			loaded, err := a.adminMutator.GetObjectCard(baseObjN)
			if err != nil {
				return contracts.FrontendObjectMutationResult{}, err
			}
			base = loaded
		}
		card, err := buildLegacyAdminObjectCard(base, request, true)
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		if err := a.adminMutator.UpdateObject(card); err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		return contracts.FrontendObjectMutationResult{
			Source:   source,
			ObjectID: int(card.ObjN),
			NativeID: strconv.FormatInt(card.ObjN, 10),
		}, nil
	case contracts.FrontendSourceCASL:
		if a.caslMutator == nil {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
		}
		if request.ObjectID <= 0 {
			return contracts.FrontendObjectMutationResult{}, fmt.Errorf("casl update requires object id")
		}
		snapshot, err := a.caslMutator.GetCASLObjectEditorSnapshot(ctx, int64(request.ObjectID))
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		update, err := buildCASLObjectUpdate(snapshot.Object, request)
		if err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		if err := a.caslMutator.UpdateCASLObject(ctx, update); err != nil {
			return contracts.FrontendObjectMutationResult{}, err
		}
		return contracts.FrontendObjectMutationResult{
			Source:   source,
			ObjectID: request.ObjectID,
			NativeID: update.ObjID,
		}, nil
	default:
		return contracts.FrontendObjectMutationResult{}, fmt.Errorf("%w: %s", contracts.ErrUnsupportedFrontendSource, source)
	}
}

func (a *FrontendAdapter) fallbackCapabilities() []contracts.FrontendSourceCapability {
	capabilities := make([]contracts.FrontendSourceCapability, 0, 2)
	if a.adminMutator != nil {
		capabilities = append(capabilities, contracts.FrontendSourceCapability{
			Source:            contracts.FrontendSourceBridge,
			DisplayName:       contracts.FrontendSourceBridge.DisplayName(),
			ReadObjects:       true,
			ReadObjectDetails: true,
			ReadEvents:        true,
			ReadAlarms:        true,
			CreateObject:      true,
			UpdateObject:      true,
		})
	}
	if a.caslMutator != nil {
		capabilities = append(capabilities, contracts.FrontendSourceCapability{
			Source:            contracts.FrontendSourceCASL,
			DisplayName:       contracts.FrontendSourceCASL.DisplayName(),
			ReadObjects:       true,
			ReadObjectDetails: true,
			ReadEvents:        true,
			ReadAlarms:        true,
			CreateObject:      true,
			UpdateObject:      true,
		})
	}
	if len(capabilities) == 0 {
		capabilities = append(capabilities, contracts.FrontendSourceCapability{
			Source:            contracts.FrontendSourceUnknown,
			DisplayName:       contracts.FrontendSourceUnknown.DisplayName(),
			ReadObjects:       true,
			ReadObjectDetails: true,
			ReadEvents:        true,
			ReadAlarms:        true,
		})
	}
	return capabilities
}

func resolveFrontendMutationSource(request contracts.FrontendObjectUpsertRequest) contracts.FrontendSource {
	if request.Source != "" && request.Source != contracts.FrontendSourceUnknown {
		return request.Source
	}
	if request.ObjectID > 0 {
		return contracts.DetectFrontendSourceByObjectID(request.ObjectID)
	}
	if request.CASL != nil {
		return contracts.FrontendSourceCASL
	}
	if request.Legacy != nil {
		return contracts.FrontendSourceBridge
	}
	return contracts.FrontendSourceUnknown
}

func buildLegacyAdminObjectCard(base contracts.AdminObjectCard, request contracts.FrontendObjectUpsertRequest, isUpdate bool) (contracts.AdminObjectCard, error) {
	payload := request.Legacy
	if payload == nil {
		return contracts.AdminObjectCard{}, contracts.ErrMissingLegacyObjectPayload
	}

	card := base
	if payload.ObjUIN > 0 {
		card.ObjUIN = payload.ObjUIN
	}
	if payload.ObjN > 0 {
		card.ObjN = payload.ObjN
	} else if card.ObjN == 0 && request.ObjectID > 0 {
		card.ObjN = int64(request.ObjectID)
	}
	if payload.GrpN > 0 {
		card.GrpN = payload.GrpN
	}
	if payload.ObjTypeID > 0 {
		card.ObjTypeID = payload.ObjTypeID
	}
	if payload.ObjRegID > 0 {
		card.ObjRegID = payload.ObjRegID
	}
	if payload.ChannelCode > 0 || (!isUpdate && payload.ChannelCode == 0) {
		card.ChannelCode = payload.ChannelCode
	}
	if payload.PPKID > 0 {
		card.PPKID = payload.PPKID
	}
	if payload.GSMHiddenN > 0 || payload.ChannelCode != 5 {
		card.GSMHiddenN = payload.GSMHiddenN
	}
	if payload.TestIntervalMin > 0 || !payload.TestControlEnabled {
		card.TestIntervalMin = payload.TestIntervalMin
	}

	card.ShortName = firstNonEmpty(strings.TrimSpace(payload.ShortName), strings.TrimSpace(request.Core.Name), strings.TrimSpace(card.ShortName))
	card.FullName = firstNonEmpty(strings.TrimSpace(payload.FullName), strings.TrimSpace(request.Core.Name), strings.TrimSpace(card.FullName))
	card.Address = firstNonEmpty(strings.TrimSpace(request.Core.Address), strings.TrimSpace(card.Address))
	card.Phones = firstNonEmpty(strings.TrimSpace(payload.Phones), strings.TrimSpace(card.Phones))
	card.Contract = firstNonEmpty(strings.TrimSpace(request.Core.Contract), strings.TrimSpace(card.Contract))
	card.StartDate = firstNonEmpty(strings.TrimSpace(payload.StartDate), strings.TrimSpace(card.StartDate))
	card.Location = firstNonEmpty(strings.TrimSpace(payload.Location), strings.TrimSpace(card.Location))
	card.Notes = firstNonEmpty(strings.TrimSpace(request.Core.Notes), strings.TrimSpace(card.Notes))
	card.GSMPhone1 = firstNonEmpty(strings.TrimSpace(payload.GSMPhone1), strings.TrimSpace(card.GSMPhone1))
	card.GSMPhone2 = firstNonEmpty(strings.TrimSpace(payload.GSMPhone2), strings.TrimSpace(card.GSMPhone2))
	card.SubServerA = firstNonEmpty(strings.TrimSpace(payload.SubServerA), strings.TrimSpace(card.SubServerA))
	card.SubServerB = firstNonEmpty(strings.TrimSpace(payload.SubServerB), strings.TrimSpace(card.SubServerB))
	card.TestControlEnabled = payload.TestControlEnabled

	return card, nil
}

func buildCASLObjectCreate(request contracts.FrontendObjectUpsertRequest) (contracts.CASLGuardObjectCreate, error) {
	payload := request.CASL
	if payload == nil {
		return contracts.CASLGuardObjectCreate{}, contracts.ErrMissingCASLObjectPayload
	}

	return contracts.CASLGuardObjectCreate{
		Name:           strings.TrimSpace(request.Core.Name),
		Address:        strings.TrimSpace(request.Core.Address),
		Long:           strings.TrimSpace(request.Core.Longitude),
		Lat:            strings.TrimSpace(request.Core.Latitude),
		Description:    strings.TrimSpace(request.Core.Description),
		Contract:       strings.TrimSpace(request.Core.Contract),
		ManagerID:      strings.TrimSpace(payload.ManagerID),
		Note:           strings.TrimSpace(request.Core.Notes),
		StartDate:      payload.StartDate,
		Status:         strings.TrimSpace(payload.Status),
		ObjectType:     strings.TrimSpace(payload.ObjectType),
		IDRequest:      strings.TrimSpace(payload.IDRequest),
		ReactingPultID: strings.TrimSpace(payload.ReactingPultID),
		GeoZoneID:      payload.GeoZoneID,
		BusinessCoeff:  payload.BusinessCoeff,
	}, nil
}

func buildCASLObjectUpdate(base contracts.CASLGuardObjectDetails, request contracts.FrontendObjectUpsertRequest) (contracts.CASLGuardObjectUpdate, error) {
	payload := request.CASL
	if payload == nil {
		return contracts.CASLGuardObjectUpdate{}, contracts.ErrMissingCASLObjectPayload
	}

	objID := strings.TrimSpace(payload.ObjID)
	if objID == "" {
		objID = strings.TrimSpace(base.ObjID)
	}
	if objID == "" {
		return contracts.CASLGuardObjectUpdate{}, fmt.Errorf("casl update requires obj_id")
	}

	return contracts.CASLGuardObjectUpdate{
		ObjID:          objID,
		Name:           firstNonEmpty(strings.TrimSpace(request.Core.Name), strings.TrimSpace(base.Name)),
		Address:        firstNonEmpty(strings.TrimSpace(request.Core.Address), strings.TrimSpace(base.Address)),
		Long:           firstNonEmpty(strings.TrimSpace(request.Core.Longitude), strings.TrimSpace(base.Long)),
		Lat:            firstNonEmpty(strings.TrimSpace(request.Core.Latitude), strings.TrimSpace(base.Lat)),
		Description:    firstNonEmpty(strings.TrimSpace(request.Core.Description), strings.TrimSpace(base.Description)),
		Contract:       firstNonEmpty(strings.TrimSpace(request.Core.Contract), strings.TrimSpace(base.Contract)),
		ManagerID:      firstNonEmpty(strings.TrimSpace(payload.ManagerID), strings.TrimSpace(base.ManagerID)),
		Note:           firstNonEmpty(strings.TrimSpace(request.Core.Notes), strings.TrimSpace(base.Note)),
		StartDate:      firstNonZeroInt64(payload.StartDate, base.StartDate),
		Status:         firstNonEmpty(strings.TrimSpace(payload.Status), strings.TrimSpace(base.ObjectStatus)),
		ObjectType:     firstNonEmpty(strings.TrimSpace(payload.ObjectType), strings.TrimSpace(base.ObjectType)),
		IDRequest:      firstNonEmpty(strings.TrimSpace(payload.IDRequest), strings.TrimSpace(base.IDRequest)),
		ReactingPultID: firstNonEmpty(strings.TrimSpace(payload.ReactingPultID), strings.TrimSpace(base.ReactingPultID)),
		GeoZoneID:      firstNonZeroInt64(payload.GeoZoneID, base.GeoZoneID),
		BusinessCoeff:  firstFloat64Ptr(payload.BusinessCoeff, base.BusinessCoeff),
		Images:         slices.Clone(base.Images),
	}, nil
}

func (a *FrontendAdapter) resolveNativeIDForObject(ctx context.Context, objectID int, object models.Object) string {
	source := contracts.DetectFrontendSourceByObjectID(objectID)
	switch source {
	case contracts.FrontendSourceBridge:
		return strconv.Itoa(objectID)
	case contracts.FrontendSourcePhoenix:
		return firstNonEmpty(strings.TrimSpace(object.DisplayNumber), strconv.Itoa(objectID))
	case contracts.FrontendSourceCASL:
		if a.caslMutator == nil {
			return ""
		}
		snapshot, err := a.caslMutator.GetCASLObjectEditorSnapshot(ctx, int64(objectID))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(snapshot.Object.ObjID)
	default:
		return ""
	}
}

func mapFrontendObjectSummary(object models.Object) contracts.FrontendObjectSummary {
	source := contracts.DetectFrontendSourceByObjectID(object.ID)
	nativeID := ""
	switch source {
	case contracts.FrontendSourceBridge:
		nativeID = strconv.Itoa(object.ID)
	case contracts.FrontendSourcePhoenix:
		nativeID = firstNonEmpty(strings.TrimSpace(object.DisplayNumber), strconv.Itoa(object.ID))
	}

	return contracts.FrontendObjectSummary{
		ID:               object.ID,
		Source:           source,
		NativeID:         nativeID,
		DisplayNumber:    firstNonEmpty(strings.TrimSpace(object.DisplayNumber), strconv.Itoa(object.ID)),
		Name:             strings.TrimSpace(object.Name),
		Address:          strings.TrimSpace(object.Address),
		ContractNumber:   strings.TrimSpace(object.ContractNum),
		Phone:            firstNonEmpty(strings.TrimSpace(object.Phone), strings.TrimSpace(object.Phones1)),
		StatusCode:       frontendObjectStatusCode(object.Status),
		StatusText:       strings.TrimSpace(object.GetStatusDisplay()),
		DeviceType:       strings.TrimSpace(object.DeviceType),
		PanelMark:        strings.TrimSpace(object.PanelMark),
		SignalStrength:   strings.TrimSpace(object.SignalStrength),
		SIM1:             strings.TrimSpace(object.SIM1),
		SIM2:             strings.TrimSpace(object.SIM2),
		LastTestTime:     object.LastTestTime,
		LastMessageTime:  object.LastMessageTime,
		GuardStatus:      frontendGuardStatus(object),
		ConnectionStatus: frontendConnectionStatus(object),
		MonitoringStatus: frontendMonitoringStatus(object),
		HasAssignment:    object.HasAssignment,
	}
}

func mapFrontendZones(zones []models.Zone) []contracts.FrontendZone {
	result := make([]contracts.FrontendZone, 0, len(zones))
	for _, zone := range zones {
		result = append(result, contracts.FrontendZone{
			Number:         zone.Number,
			Name:           strings.TrimSpace(zone.Name),
			SensorType:     strings.TrimSpace(zone.SensorType),
			Status:         frontendZoneStatus(zone),
			GroupID:        strings.TrimSpace(zone.GroupID),
			GroupNumber:    zone.GroupNumber,
			GroupName:      strings.TrimSpace(zone.GroupName),
			GroupStateText: strings.TrimSpace(zone.GroupStateText),
		})
	}
	return result
}

func mapFrontendContacts(contacts []models.Contact) []contracts.FrontendContact {
	result := make([]contracts.FrontendContact, 0, len(contacts))
	for _, contact := range contacts {
		result = append(result, contracts.FrontendContact{
			Name:           strings.TrimSpace(contact.Name),
			Position:       strings.TrimSpace(contact.Position),
			Phone:          strings.TrimSpace(contact.Phone),
			Priority:       contact.Priority,
			CodeWord:       strings.TrimSpace(contact.CodeWord),
			GroupID:        strings.TrimSpace(contact.GroupID),
			GroupNumber:    contact.GroupNumber,
			GroupName:      strings.TrimSpace(contact.GroupName),
			GroupStateText: strings.TrimSpace(contact.GroupStateText),
		})
	}
	return result
}

func mapFrontendEvents(events []models.Event) []contracts.FrontendEventItem {
	result := make([]contracts.FrontendEventItem, 0, len(events))
	for _, event := range events {
		result = append(result, mapFrontendEventItem(event))
	}
	return result
}

func mapFrontendAlarmItem(alarm models.Alarm) contracts.FrontendAlarmItem {
	source := contracts.DetectFrontendSourceByObjectID(alarm.ObjectID)
	return contracts.FrontendAlarmItem{
		ID:                        alarm.ID,
		Source:                    source,
		ObjectID:                  alarm.ObjectID,
		ObjectNumber:              alarm.GetObjectNumberDisplay(),
		ObjectName:                strings.TrimSpace(alarm.ObjectName),
		Address:                   strings.TrimSpace(alarm.Address),
		Time:                      alarm.Time,
		Details:                   strings.TrimSpace(alarm.Details),
		TypeCode:                  string(alarm.Type),
		TypeText:                  strings.TrimSpace(alarm.GetTypeDisplay()),
		ZoneNumber:                alarm.ZoneNumber,
		ZoneName:                  strings.TrimSpace(alarm.ZoneName),
		IsProcessed:               alarm.IsProcessed,
		ProcessedBy:               strings.TrimSpace(alarm.ProcessedBy),
		ProcessNote:               strings.TrimSpace(alarm.ProcessNote),
		IsInProgress:              alarm.IsInProgress,
		InProgressBy:              strings.TrimSpace(alarm.InProgressBy),
		IsOwnedByMe:               alarm.IsOwnedByMe,
		CanTakeOver:               source == contracts.FrontendSourceCASL && alarm.IsInProgress && !alarm.IsOwnedByMe,
		CanProcess:                (source != contracts.FrontendSourceCASL || !alarm.IsInProgress || alarm.IsOwnedByMe) && !alarm.IsResponseGroupDispatched,
		ResponseGroupID:           strings.TrimSpace(alarm.ResponseGroupID),
		IsResponseGroupDispatched: alarm.IsResponseGroupDispatched,
		IsResponseGroupArrived:    alarm.IsResponseGroupArrived,
		ObjectNativeID:            alarm.GetObjectNumberDisplay(),
		VisualSeverity:            frontendAlarmSeverity(alarm),
	}
}

func mapFrontendEventItem(event models.Event) contracts.FrontendEventItem {
	return contracts.FrontendEventItem{
		ID:             event.ID,
		Source:         contracts.DetectFrontendSourceByObjectID(event.ObjectID),
		ObjectID:       event.ObjectID,
		ObjectNumber:   firstNonEmpty(strings.TrimSpace(event.ObjectNumber), strconv.Itoa(event.ObjectID)),
		ObjectName:     strings.TrimSpace(event.ObjectName),
		Time:           event.Time,
		TypeCode:       string(event.Type),
		TypeText:       strings.TrimSpace(event.GetTypeDisplay()),
		ZoneNumber:     event.ZoneNumber,
		Details:        strings.TrimSpace(event.Details),
		UserName:       strings.TrimSpace(event.UserName),
		ObjectNativeID: firstNonEmpty(strings.TrimSpace(event.ObjectNumber), strconv.Itoa(event.ObjectID)),
		VisualSeverity: frontendEventSeverity(event),
	}
}

func frontendObjectStatusCode(status models.ObjectStatus) string {
	switch status {
	case models.StatusNormal:
		return "normal"
	case models.StatusFire:
		return "alarm"
	case models.StatusFault:
		return "fault"
	case models.StatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}

func frontendPowerSource(source models.PowerSource) string {
	switch source {
	case models.PowerMains:
		return "mains"
	case models.PowerBattery:
		return "battery"
	default:
		return "unknown"
	}
}

func frontendGuardStatus(object models.Object) contracts.FrontendGuardStatus {
	switch object.GuardStatus {
	case models.GuardStatusGuarded:
		return contracts.FrontendGuardStatusGuarded
	case models.GuardStatusDisarmed:
		return contracts.FrontendGuardStatusDisarmed
	}
	switch {
	case object.GuardState > 0, object.IsUnderGuard:
		return contracts.FrontendGuardStatusGuarded
	case object.GuardState == 0:
		return contracts.FrontendGuardStatusDisarmed
	default:
		return contracts.FrontendGuardStatusUnknown
	}
}

func frontendConnectionStatus(object models.Object) contracts.FrontendConnectionStatus {
	switch object.ConnectionStatus {
	case models.ConnectionStatusOnline:
		return contracts.FrontendConnectionStatusOnline
	case models.ConnectionStatusOffline:
		return contracts.FrontendConnectionStatusOffline
	}
	// Explicit "online" signals take highest priority.
	if object.IsConnState > 0 || object.IsConnOK {
		return contracts.FrontendConnectionStatusOnline
	}
	// Explicit "offline" status (provider set it deliberately).
	if object.Status == models.StatusOffline {
		return contracts.FrontendConnectionStatusOffline
	}
	// IsConnState == 0 with non-offline status means the provider does not track
	// connection state (e.g. Bridge with NULL IsConnState1 in DB) — report unknown.
	return contracts.FrontendConnectionStatusUnknown
}

func frontendMonitoringStatus(object models.Object) contracts.FrontendMonitoringStatus {
	switch object.MonitoringStatus {
	case models.MonitoringStatusBlocked:
		return contracts.FrontendMonitoringStatusBlocked
	case models.MonitoringStatusDebug:
		return contracts.FrontendMonitoringStatusDebug
	case models.MonitoringStatusActive:
		return contracts.FrontendMonitoringStatusActive
	}
	switch object.BlockedArmedOnOff {
	case 1:
		return contracts.FrontendMonitoringStatusBlocked
	case 2:
		return contracts.FrontendMonitoringStatusDebug
	case 0:
		return contracts.FrontendMonitoringStatusActive
	default:
		return contracts.FrontendMonitoringStatusUnknown
	}
}

func frontendAlarmSeverity(alarm models.Alarm) contracts.FrontendVisualSeverity {
	switch alarm.Type {
	case models.AlarmFire,
		models.AlarmBurglary,
		models.AlarmPanic,
		models.AlarmMedical,
		models.AlarmGas,
		models.AlarmTamper,
		models.AlarmOperator,
		models.AlarmDevice,
		models.AlarmMobile:
		return contracts.FrontendVisualSeverityCritical
	case models.AlarmFault,
		models.AlarmPowerFail,
		models.AlarmBatteryLow,
		models.AlarmOffline,
		models.AlarmAcTrouble,
		models.AlarmFireTrouble:
		return contracts.FrontendVisualSeverityWarning
	case models.AlarmEliminated,
		models.AlarmNotification,
		models.AlarmSystemEvent:
		return contracts.FrontendVisualSeverityInfo
	default:
		if alarm.IsCritical() {
			return contracts.FrontendVisualSeverityCritical
		}
		return contracts.FrontendVisualSeverityNormal
	}
}

func frontendAlarmSeverityFromMsg(msg models.AlarmMsg) contracts.FrontendVisualSeverity {
	if msg.IsAlarm {
		return contracts.FrontendVisualSeverityCritical
	}
	code := strings.ToLower(msg.Code)
	if code == "fault" || code == "power_fail" || code == "batt_low" {
		return contracts.FrontendVisualSeverityWarning
	}
	if code == "restore" || code == "online" || strings.HasPrefix(code, "r") {
		return contracts.FrontendVisualSeverityNormal
	}
	return contracts.FrontendVisualSeverityInfo
}

func frontendEventSeverity(event models.Event) contracts.FrontendVisualSeverity {
	switch {
	case event.IsCritical():
		return contracts.FrontendVisualSeverityCritical
	case event.IsWarning():
		return contracts.FrontendVisualSeverityWarning
	case event.Type == models.EventNotification || event.Type == models.EventOperatorAction || event.Type == models.SystemEvent:
		return contracts.FrontendVisualSeverityInfo
	default:
		return contracts.FrontendVisualSeverityNormal
	}
}

func frontendZoneStatus(zone models.Zone) string {
	if zone.IsBypassed {
		return "bypassed"
	}
	switch zone.Status {
	case models.ZoneNormal:
		return "normal"
	case models.ZoneFire:
		return "fire"
	case models.ZoneAlarm:
		return "alarm"
	case models.ZoneBreak:
		return "break"
	case models.ZoneShort:
		return "short"
	default:
		return "unknown"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstFloat64Ptr(values ...*float64) *float64 {
	for _, value := range values {
		if value != nil {
			copied := *value
			return &copied
		}
	}
	return nil
}
