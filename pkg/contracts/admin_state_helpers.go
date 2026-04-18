package contracts

func (r AdminStatisticsRow) GuardStatusValue() FrontendGuardStatus {
	return normalizeAdminGuardStatus(r.GuardState)
}

func (r AdminStatisticsRow) ConnectionStatusValue() FrontendConnectionStatus {
	return normalizeAdminConnectionStatus(r.IsConnState)
}

func (r AdminStatisticsRow) MonitoringStatusValue() FrontendMonitoringStatus {
	return normalizeAdminMonitoringStatus(r.BlockMode)
}

func (r AdminStatisticsRow) VisualSeverityValue() FrontendVisualSeverity {
	return normalizeAdminVisualSeverity(r.AlarmState, r.TechAlarmState, r.ConnectionStatusValue())
}

func (o DisplayBlockObject) GuardStatusValue() FrontendGuardStatus {
	return normalizeAdminGuardStatus(o.GuardState)
}

func (o DisplayBlockObject) ConnectionStatusValue() FrontendConnectionStatus {
	return normalizeAdminConnectionStatus(o.IsConnState)
}

func (o DisplayBlockObject) MonitoringStatusValue() FrontendMonitoringStatus {
	return normalizeAdminMonitoringStatus(o.BlockMode)
}

func (o DisplayBlockObject) VisualSeverityValue() FrontendVisualSeverity {
	return normalizeAdminVisualSeverity(o.AlarmState, o.TechAlarmState, o.ConnectionStatusValue())
}

func normalizeAdminGuardStatus(state int64) FrontendGuardStatus {
	switch {
	case state == 0:
		return FrontendGuardStatusDisarmed
	case state > 0:
		return FrontendGuardStatusGuarded
	default:
		return FrontendGuardStatusUnknown
	}
}

func normalizeAdminConnectionStatus(state int64) FrontendConnectionStatus {
	if state > 0 {
		return FrontendConnectionStatusOnline
	}
	return FrontendConnectionStatusOffline
}

func normalizeAdminMonitoringStatus(mode DisplayBlockMode) FrontendMonitoringStatus {
	switch mode {
	case DisplayBlockTemporaryOff:
		return FrontendMonitoringStatusBlocked
	case DisplayBlockDebug:
		return FrontendMonitoringStatusDebug
	case DisplayBlockNone:
		return FrontendMonitoringStatusActive
	default:
		return FrontendMonitoringStatusUnknown
	}
}

func normalizeAdminVisualSeverity(alarmState int64, techAlarmState int64, connectionStatus FrontendConnectionStatus) FrontendVisualSeverity {
	switch {
	case alarmState > 0:
		return FrontendVisualSeverityCritical
	case techAlarmState > 0:
		return FrontendVisualSeverityWarning
	case connectionStatus == FrontendConnectionStatusOffline:
		return FrontendVisualSeverityWarning
	default:
		return FrontendVisualSeverityNormal
	}
}
