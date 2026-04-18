package v1

import "obj_catalog_fyne_v3/pkg/contracts"

func FromObjectUpsertRequest(request ObjectUpsertRequest) contracts.FrontendObjectUpsertRequest {
	return contracts.FrontendObjectUpsertRequest{
		Source:   toContractSource(request.Source),
		ObjectID: request.ObjectID,
		Core: contracts.FrontendObjectCoreFields{
			Name:        request.Core.Name,
			Address:     request.Core.Address,
			Contract:    request.Core.Contract,
			Description: request.Core.Description,
			Notes:       request.Core.Notes,
			Latitude:    request.Core.Latitude,
			Longitude:   request.Core.Longitude,
		},
		Legacy: fromLegacyPayload(request.Legacy),
		CASL:   fromCASLPayload(request.CASL),
	}
}

func ToCapabilities(capabilities contracts.FrontendCapabilities) Capabilities {
	items := make([]SourceCapability, 0, len(capabilities.Sources))
	for _, item := range capabilities.Sources {
		items = append(items, SourceCapability{
			Source:            toSource(item.Source),
			DisplayName:       item.DisplayName,
			ReadObjects:       item.ReadObjects,
			ReadObjectDetails: item.ReadObjectDetails,
			ReadEvents:        item.ReadEvents,
			ReadAlarms:        item.ReadAlarms,
			CreateObject:      item.CreateObject,
			UpdateObject:      item.UpdateObject,
		})
	}
	return Capabilities{Sources: items}
}

func ToObjectListResponse(items []contracts.FrontendObjectSummary) ObjectListResponse {
	responseItems := make([]ObjectSummary, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, ToObjectSummary(item))
	}
	return ObjectListResponse{Items: responseItems}
}

func ToAlarmListResponse(items []contracts.FrontendAlarmItem) AlarmListResponse {
	responseItems := make([]AlarmItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, ToAlarmItem(item))
	}
	return AlarmListResponse{Items: responseItems}
}

func ToEventListResponse(items []contracts.FrontendEventItem) EventListResponse {
	responseItems := make([]EventItem, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, ToEventItem(item))
	}
	return EventListResponse{Items: responseItems}
}

func ToObjectSummary(item contracts.FrontendObjectSummary) ObjectSummary {
	return ObjectSummary{
		ID:               item.ID,
		Source:           toSource(item.Source),
		NativeID:         item.NativeID,
		DisplayNumber:    item.DisplayNumber,
		Name:             item.Name,
		Address:          item.Address,
		ContractNumber:   item.ContractNumber,
		Phone:            item.Phone,
		StatusCode:       item.StatusCode,
		StatusText:       item.StatusText,
		DeviceType:       item.DeviceType,
		PanelMark:        item.PanelMark,
		SignalStrength:   item.SignalStrength,
		SIM1:             item.SIM1,
		SIM2:             item.SIM2,
		LastTestTime:     item.LastTestTime,
		LastMessageTime:  item.LastMessageTime,
		GuardStatus:      toGuardStatus(item.GuardStatus),
		ConnectionStatus: toConnectionStatus(item.ConnectionStatus),
		MonitoringStatus: toMonitoringStatus(item.MonitoringStatus),
		HasAssignment:    item.HasAssignment,
	}
}

func ToAlarmItem(item contracts.FrontendAlarmItem) AlarmItem {
	return AlarmItem{
		ID:             item.ID,
		Source:         toSource(item.Source),
		ObjectID:       item.ObjectID,
		ObjectNativeID: item.ObjectNativeID,
		ObjectNumber:   item.ObjectNumber,
		ObjectName:     item.ObjectName,
		Address:        item.Address,
		Time:           item.Time,
		Details:        item.Details,
		TypeCode:       item.TypeCode,
		TypeText:       item.TypeText,
		ZoneNumber:     item.ZoneNumber,
		ZoneName:       item.ZoneName,
		IsProcessed:    item.IsProcessed,
		ProcessedBy:    item.ProcessedBy,
		ProcessNote:    item.ProcessNote,
		VisualSeverity: toVisualSeverity(item.VisualSeverity),
	}
}

func ToEventItem(item contracts.FrontendEventItem) EventItem {
	return EventItem{
		ID:             item.ID,
		Source:         toSource(item.Source),
		ObjectID:       item.ObjectID,
		ObjectNativeID: item.ObjectNativeID,
		ObjectNumber:   item.ObjectNumber,
		ObjectName:     item.ObjectName,
		Time:           item.Time,
		TypeCode:       item.TypeCode,
		TypeText:       item.TypeText,
		ZoneNumber:     item.ZoneNumber,
		Details:        item.Details,
		UserName:       item.UserName,
		VisualSeverity: toVisualSeverity(item.VisualSeverity),
	}
}

func ToObjectDetails(item contracts.FrontendObjectDetails) ObjectDetails {
	return ObjectDetails{
		Summary:             ToObjectSummary(item.Summary),
		GSMLevel:            item.GSMLevel,
		PowerSource:         item.PowerSource,
		AutoTestHours:       item.AutoTestHours,
		SubServerA:          item.SubServerA,
		SubServerB:          item.SubServerB,
		ChannelCode:         item.ChannelCode,
		AKBState:            item.AKBState,
		PowerFault:          item.PowerFault,
		TestControl:         item.TestControl,
		TestIntervalMin:     item.TestIntervalMin,
		Phones:              item.Phones,
		Notes:               item.Notes,
		Location:            item.Location,
		LaunchDate:          item.LaunchDate,
		ExternalSignal:      item.ExternalSignal,
		ExternalTestMessage: item.ExternalTestMessage,
		ExternalLastTest:    item.ExternalLastTest,
		ExternalLastMessage: item.ExternalLastMessage,
		Zones:               toZones(item.Zones),
		Contacts:            toContacts(item.Contacts),
		Events:              toEvents(item.Events),
	}
}

func ToObjectMutationResult(item contracts.FrontendObjectMutationResult) ObjectMutationResult {
	return ObjectMutationResult{
		Source:   toSource(item.Source),
		ObjectID: item.ObjectID,
		NativeID: item.NativeID,
	}
}

func toZones(items []contracts.FrontendZone) []Zone {
	result := make([]Zone, 0, len(items))
	for _, item := range items {
		result = append(result, Zone{
			Number:         item.Number,
			Name:           item.Name,
			SensorType:     item.SensorType,
			Status:         item.Status,
			GroupID:        item.GroupID,
			GroupNumber:    item.GroupNumber,
			GroupName:      item.GroupName,
			GroupStateText: item.GroupStateText,
		})
	}
	return result
}

func toContacts(items []contracts.FrontendContact) []Contact {
	result := make([]Contact, 0, len(items))
	for _, item := range items {
		result = append(result, Contact{
			Name:           item.Name,
			Position:       item.Position,
			Phone:          item.Phone,
			Priority:       item.Priority,
			CodeWord:       item.CodeWord,
			GroupID:        item.GroupID,
			GroupNumber:    item.GroupNumber,
			GroupName:      item.GroupName,
			GroupStateText: item.GroupStateText,
		})
	}
	return result
}

func toEvents(items []contracts.FrontendEventItem) []EventItem {
	result := make([]EventItem, 0, len(items))
	for _, item := range items {
		result = append(result, ToEventItem(item))
	}
	return result
}

func fromLegacyPayload(payload *LegacyObjectPayload) *contracts.FrontendLegacyObjectPayload {
	if payload == nil {
		return nil
	}
	return &contracts.FrontendLegacyObjectPayload{
		ObjUIN:             payload.ObjUIN,
		ObjN:               payload.ObjN,
		GrpN:               payload.GrpN,
		ObjTypeID:          payload.ObjTypeID,
		ObjRegID:           payload.ObjRegID,
		ChannelCode:        payload.ChannelCode,
		PPKID:              payload.PPKID,
		GSMHiddenN:         payload.GSMHiddenN,
		TestIntervalMin:    payload.TestIntervalMin,
		ShortName:          payload.ShortName,
		FullName:           payload.FullName,
		Phones:             payload.Phones,
		StartDate:          payload.StartDate,
		Location:           payload.Location,
		GSMPhone1:          payload.GSMPhone1,
		GSMPhone2:          payload.GSMPhone2,
		SubServerA:         payload.SubServerA,
		SubServerB:         payload.SubServerB,
		TestControlEnabled: payload.TestControlEnabled,
	}
}

func fromCASLPayload(payload *CASLObjectPayload) *contracts.FrontendCASLObjectPayload {
	if payload == nil {
		return nil
	}
	return &contracts.FrontendCASLObjectPayload{
		ObjID:          payload.ObjID,
		ManagerID:      payload.ManagerID,
		Status:         payload.Status,
		ObjectType:     payload.ObjectType,
		IDRequest:      payload.IDRequest,
		ReactingPultID: payload.ReactingPultID,
		StartDate:      payload.StartDate,
		GeoZoneID:      payload.GeoZoneID,
		BusinessCoeff:  payload.BusinessCoeff,
	}
}

func toSource(source contracts.FrontendSource) Source {
	switch source {
	case contracts.FrontendSourceBridge:
		return SourceBridge
	case contracts.FrontendSourcePhoenix:
		return SourcePhoenix
	case contracts.FrontendSourceCASL:
		return SourceCASL
	default:
		return SourceUnknown
	}
}

func toContractSource(source Source) contracts.FrontendSource {
	switch source {
	case SourceBridge:
		return contracts.FrontendSourceBridge
	case SourcePhoenix:
		return contracts.FrontendSourcePhoenix
	case SourceCASL:
		return contracts.FrontendSourceCASL
	default:
		return contracts.FrontendSourceUnknown
	}
}

func toConnectionStatus(status contracts.FrontendConnectionStatus) ConnectionStatus {
	switch status {
	case contracts.FrontendConnectionStatusOnline:
		return ConnectionStatusOnline
	case contracts.FrontendConnectionStatusOffline:
		return ConnectionStatusOffline
	default:
		return ConnectionStatusUnknown
	}
}

func toGuardStatus(status contracts.FrontendGuardStatus) GuardStatus {
	switch status {
	case contracts.FrontendGuardStatusGuarded:
		return GuardStatusGuarded
	case contracts.FrontendGuardStatusDisarmed:
		return GuardStatusDisarmed
	default:
		return GuardStatusUnknown
	}
}

func toMonitoringStatus(status contracts.FrontendMonitoringStatus) MonitoringStatus {
	switch status {
	case contracts.FrontendMonitoringStatusActive:
		return MonitoringStatusActive
	case contracts.FrontendMonitoringStatusBlocked:
		return MonitoringStatusBlocked
	case contracts.FrontendMonitoringStatusDebug:
		return MonitoringStatusDebug
	default:
		return MonitoringStatusUnknown
	}
}

func toVisualSeverity(status contracts.FrontendVisualSeverity) VisualSeverity {
	switch status {
	case contracts.FrontendVisualSeverityNormal:
		return VisualSeverityNormal
	case contracts.FrontendVisualSeverityInfo:
		return VisualSeverityInfo
	case contracts.FrontendVisualSeverityWarning:
		return VisualSeverityWarning
	case contracts.FrontendVisualSeverityCritical:
		return VisualSeverityCritical
	default:
		return VisualSeverityUnknown
	}
}
