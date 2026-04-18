package backend

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

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
	result := make([]contracts.FrontendAlarmItem, 0, len(alarms))
	for _, alarm := range alarms {
		result = append(result, mapFrontendAlarmItem(alarm))
	}
	return result, nil
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
	details := contracts.FrontendObjectDetails{
		Summary:             summary,
		GSMLevel:            object.GSMLevel,
		PowerSource:         frontendPowerSource(object.PowerSource),
		AutoTestHours:       object.AutoTestHours,
		SubServerA:          object.SubServerA,
		SubServerB:          object.SubServerB,
		ChannelCode:         object.ObjChan,
		AKBState:            object.AkbState,
		PowerFault:          object.PowerFault,
		TestControl:         object.TestControl > 0,
		TestIntervalMin:     object.TestTime,
		Phones:              object.Phones1,
		Notes:               object.Notes1,
		Location:            object.Location1,
		LaunchDate:          object.LaunchDate,
		ExternalSignal:      signal,
		ExternalTestMessage: testMessage,
		ExternalLastTest:    lastTest,
		ExternalLastMessage: lastMessage,
		Zones:               mapFrontendZones(a.dataProvider.GetZones(rawID)),
		Contacts:            mapFrontendContacts(a.dataProvider.GetEmployees(rawID)),
		Events:              mapFrontendEvents(a.dataProvider.GetObjectEvents(rawID)),
	}

	return details, nil
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
	return contracts.FrontendAlarmItem{
		ID:             alarm.ID,
		Source:         contracts.DetectFrontendSourceByObjectID(alarm.ObjectID),
		ObjectID:       alarm.ObjectID,
		ObjectNumber:   alarm.GetObjectNumberDisplay(),
		ObjectName:     strings.TrimSpace(alarm.ObjectName),
		Address:        strings.TrimSpace(alarm.Address),
		Time:           alarm.Time,
		Details:        strings.TrimSpace(alarm.Details),
		TypeCode:       string(alarm.Type),
		TypeText:       strings.TrimSpace(alarm.GetTypeDisplay()),
		ZoneNumber:     alarm.ZoneNumber,
		ZoneName:       strings.TrimSpace(alarm.ZoneName),
		IsProcessed:    alarm.IsProcessed,
		ProcessedBy:    strings.TrimSpace(alarm.ProcessedBy),
		ProcessNote:    strings.TrimSpace(alarm.ProcessNote),
		ObjectNativeID: alarm.GetObjectNumberDisplay(),
		VisualSeverity: frontendAlarmSeverity(alarm),
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
	switch {
	case object.IsConnState > 0, object.IsConnOK, object.Status != models.StatusOffline:
		if object.IsConnState == 0 && !object.IsConnOK {
			return contracts.FrontendConnectionStatusOffline
		}
		return contracts.FrontendConnectionStatusOnline
	case object.IsConnState == 0, object.Status == models.StatusOffline:
		return contracts.FrontendConnectionStatusOffline
	default:
		return contracts.FrontendConnectionStatusUnknown
	}
}

func frontendMonitoringStatus(object models.Object) contracts.FrontendMonitoringStatus {
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
