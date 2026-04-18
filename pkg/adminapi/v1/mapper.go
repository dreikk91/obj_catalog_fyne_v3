package v1

import (
	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

func ToStatisticsFilter(filter StatisticsFilter) contracts.AdminStatisticsFilter {
	return contracts.AdminStatisticsFilter{
		ConnectionMode: toContractsStatisticsConnectionMode(filter.ConnectionMode),
		ProtocolFilter: toContractsStatisticsProtocolFilter(filter.ProtocolFilter),
		ChannelCode:    filter.ChannelCode,
		GuardState:     filter.GuardState,
		ObjTypeID:      filter.ObjTypeID,
		RegionID:       filter.RegionID,
		BlockMode:      toContractsDisplayBlockModePtr(filter.BlockMode),
		Search:         filter.Search,
	}
}

func ToContractsDisplayBlockMode(mode DisplayBlockMode) contracts.DisplayBlockMode {
	switch mode {
	case DisplayBlockModeTemporaryOff:
		return contracts.DisplayBlockTemporaryOff
	case DisplayBlockModeDebug:
		return contracts.DisplayBlockDebug
	default:
		return contracts.DisplayBlockNone
	}
}

func ToContractsMessage220VMode(mode Message220VMode) contracts.Admin220VMode {
	switch mode {
	case Message220VModeAlarm:
		return contracts.Admin220VAlarm
	case Message220VModeRestore:
		return contracts.Admin220VRestore
	default:
		return contracts.Admin220VNone
	}
}

func ToStatisticsRows(rows []contracts.AdminStatisticsRow) []StatisticsRow {
	result := make([]StatisticsRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, ToStatisticsRow(row))
	}
	return result
}

func ToStatisticsRow(row contracts.AdminStatisticsRow) StatisticsRow {
	return StatisticsRow{
		ObjUIN:           row.ObjUIN,
		ObjN:             row.ObjN,
		GrpN:             row.GrpN,
		ShortName:        row.ShortName,
		FullName:         row.FullName,
		Address:          row.Address,
		Phones:           row.Phones,
		Contract:         row.Contract,
		StartDate:        row.StartDate,
		Location:         row.Location,
		Notes:            row.Notes,
		ChannelCode:      row.ChannelCode,
		PPKID:            row.PPKID,
		PPKName:          row.PPKName,
		GSMPhone1:        row.GSMPhone1,
		GSMPhone2:        row.GSMPhone2,
		GSMHiddenN:       row.GSMHiddenN,
		SubServerA:       row.SubServerA,
		SubServerB:       row.SubServerB,
		TestControl:      row.TestControl,
		TestTime:         row.TestTime,
		GuardState:       row.GuardState,
		IsConnState:      row.IsConnState,
		AlarmState:       row.AlarmState,
		TechAlarmState:   row.TechAlarmState,
		ObjTypeID:        row.ObjTypeID,
		ObjTypeName:      row.ObjTypeName,
		RegionID:         row.RegionID,
		RegionName:       row.RegionName,
		BlockMode:        toDisplayBlockMode(row.BlockMode),
		GuardStatus:      toGuardStatus(row.GuardStatusValue()),
		ConnectionStatus: toConnectionStatus(row.ConnectionStatusValue()),
		MonitoringStatus: toMonitoringStatus(row.MonitoringStatusValue()),
		VisualSeverity:   toVisualSeverity(row.VisualSeverityValue()),
	}
}

func ToDisplayBlockObjects(items []contracts.DisplayBlockObject) []DisplayBlockObject {
	result := make([]DisplayBlockObject, 0, len(items))
	for _, item := range items {
		result = append(result, ToDisplayBlockObject(item))
	}
	return result
}

func ToDisplayBlockObject(item contracts.DisplayBlockObject) DisplayBlockObject {
	return DisplayBlockObject{
		ObjN:             item.ObjN,
		Name:             item.Name,
		BlockMode:        toDisplayBlockMode(item.BlockMode),
		AlarmState:       item.AlarmState,
		GuardState:       item.GuardState,
		TechAlarmState:   item.TechAlarmState,
		IsConnState:      item.IsConnState,
		GuardStatus:      toGuardStatus(item.GuardStatusValue()),
		ConnectionStatus: toConnectionStatus(item.ConnectionStatusValue()),
		MonitoringStatus: toMonitoringStatus(item.MonitoringStatusValue()),
		VisualSeverity:   toVisualSeverity(item.VisualSeverityValue()),
	}
}

func ToDictionaryItems(items []contracts.DictionaryItem) []DictionaryItem {
	result := make([]DictionaryItem, 0, len(items))
	for _, item := range items {
		result = append(result, DictionaryItem{
			ID:    item.ID,
			Name:  item.Name,
			Code:  item.Code,
			Extra: item.Extra,
		})
	}
	return result
}

func ToMessages(items []contracts.AdminMessage) []Message {
	result := make([]Message, 0, len(items))
	for _, item := range items {
		result = append(result, ToMessage(item))
	}
	return result
}

func ToMessage(item contracts.AdminMessage) Message {
	return Message{
		UIN:          item.UIN,
		ProtocolID:   item.ProtocolID,
		MessageID:    item.MessageID,
		MessageHex:   item.MessageHex,
		Text:         item.Text,
		SC1:          item.SC1,
		ForAdminOnly: item.ForAdminOnly,
	}
}

func ToMessage220VBuckets(buckets contracts.Admin220VMessageBuckets) Message220VBuckets {
	return Message220VBuckets{
		Free:    ToMessages(buckets.Free),
		Alarm:   ToMessages(buckets.Alarm),
		Restore: ToMessages(buckets.Restore),
	}
}

func toDisplayBlockMode(mode contracts.DisplayBlockMode) DisplayBlockMode {
	switch mode {
	case contracts.DisplayBlockTemporaryOff:
		return DisplayBlockModeTemporaryOff
	case contracts.DisplayBlockDebug:
		return DisplayBlockModeDebug
	default:
		return DisplayBlockModeNone
	}
}

func toContractsDisplayBlockModePtr(mode *DisplayBlockMode) *contracts.DisplayBlockMode {
	if mode == nil {
		return nil
	}
	value := ToContractsDisplayBlockMode(*mode)
	return &value
}

func toContractsStatisticsConnectionMode(mode StatisticsConnectionMode) contracts.AdminStatisticsConnectionMode {
	switch mode {
	case StatisticsConnectionModeOnline:
		return contracts.StatsConnectionOnline
	case StatisticsConnectionModeOffline:
		return contracts.StatsConnectionOffline
	default:
		return contracts.StatsConnectionAll
	}
}

func toContractsStatisticsProtocolFilter(filter StatisticsProtocolFilter) contracts.AdminStatisticsProtocolFilter {
	switch filter {
	case StatisticsProtocolAutodial:
		return contracts.StatsProtocolAutodial
	case StatisticsProtocolMost:
		return contracts.StatsProtocolMost
	case StatisticsProtocolNova:
		return contracts.StatsProtocolNova
	default:
		return contracts.StatsProtocolAll
	}
}

func toGuardStatus(status contracts.FrontendGuardStatus) frontendv1.GuardStatus {
	switch status {
	case contracts.FrontendGuardStatusGuarded:
		return frontendv1.GuardStatusGuarded
	case contracts.FrontendGuardStatusDisarmed:
		return frontendv1.GuardStatusDisarmed
	default:
		return frontendv1.GuardStatusUnknown
	}
}

func toConnectionStatus(status contracts.FrontendConnectionStatus) frontendv1.ConnectionStatus {
	switch status {
	case contracts.FrontendConnectionStatusOnline:
		return frontendv1.ConnectionStatusOnline
	case contracts.FrontendConnectionStatusOffline:
		return frontendv1.ConnectionStatusOffline
	default:
		return frontendv1.ConnectionStatusUnknown
	}
}

func toMonitoringStatus(status contracts.FrontendMonitoringStatus) frontendv1.MonitoringStatus {
	switch status {
	case contracts.FrontendMonitoringStatusActive:
		return frontendv1.MonitoringStatusActive
	case contracts.FrontendMonitoringStatusBlocked:
		return frontendv1.MonitoringStatusBlocked
	case contracts.FrontendMonitoringStatusDebug:
		return frontendv1.MonitoringStatusDebug
	default:
		return frontendv1.MonitoringStatusUnknown
	}
}

func toVisualSeverity(status contracts.FrontendVisualSeverity) frontendv1.VisualSeverity {
	switch status {
	case contracts.FrontendVisualSeverityNormal:
		return frontendv1.VisualSeverityNormal
	case contracts.FrontendVisualSeverityInfo:
		return frontendv1.VisualSeverityInfo
	case contracts.FrontendVisualSeverityWarning:
		return frontendv1.VisualSeverityWarning
	case contracts.FrontendVisualSeverityCritical:
		return frontendv1.VisualSeverityCritical
	default:
		return frontendv1.VisualSeverityUnknown
	}
}
