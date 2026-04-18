package models

type ConnectionStatus string

const (
	ConnectionStatusUnknown ConnectionStatus = "unknown"
	ConnectionStatusOnline  ConnectionStatus = "online"
	ConnectionStatusOffline ConnectionStatus = "offline"
)

type GuardStatus string

const (
	GuardStatusUnknown  GuardStatus = "unknown"
	GuardStatusGuarded  GuardStatus = "guarded"
	GuardStatusDisarmed GuardStatus = "disarmed"
)

type MonitoringStatus string

const (
	MonitoringStatusUnknown MonitoringStatus = "unknown"
	MonitoringStatusActive  MonitoringStatus = "active"
	MonitoringStatusBlocked MonitoringStatus = "blocked"
	MonitoringStatusDebug   MonitoringStatus = "debug"
)

type VisualSeverity string

const (
	VisualSeverityUnknown  VisualSeverity = "unknown"
	VisualSeverityNormal   VisualSeverity = "normal"
	VisualSeverityInfo     VisualSeverity = "info"
	VisualSeverityWarning  VisualSeverity = "warning"
	VisualSeverityCritical VisualSeverity = "critical"
)

func (o Object) GuardStatusValue() GuardStatus {
	if o.GuardStatus != "" && o.GuardStatus != GuardStatusUnknown {
		return o.GuardStatus
	}
	if o.GuardState > 0 || o.IsUnderGuard {
		return GuardStatusGuarded
	}
	if o.GuardState == 0 && hasLegacyObjectState(o) {
		return GuardStatusDisarmed
	}
	return GuardStatusUnknown
}

func (o Object) ConnectionStatusValue() ConnectionStatus {
	if o.ConnectionStatus != "" && o.ConnectionStatus != ConnectionStatusUnknown {
		return o.ConnectionStatus
	}
	if o.Status == StatusOffline {
		return ConnectionStatusOffline
	}
	if o.IsConnState > 0 || o.IsConnOK {
		return ConnectionStatusOnline
	}
	if o.IsConnState == 0 && hasExplicitLegacyConnectionState(o) {
		return ConnectionStatusOffline
	}
	return ConnectionStatusUnknown
}

func (o Object) MonitoringStatusValue() MonitoringStatus {
	if o.MonitoringStatus != "" && o.MonitoringStatus != MonitoringStatusUnknown {
		return o.MonitoringStatus
	}
	switch o.BlockedArmedOnOff {
	case 1:
		return MonitoringStatusBlocked
	case 2:
		return MonitoringStatusDebug
	case 0:
		if hasLegacyObjectState(o) {
			return MonitoringStatusActive
		}
		return MonitoringStatusUnknown
	default:
		return MonitoringStatusUnknown
	}
}

func hasLegacyObjectState(o Object) bool {
	return o.GuardState != 0 ||
		o.IsConnState != 0 ||
		o.AlarmState != 0 ||
		o.TechAlarmState != 0 ||
		o.BlockedArmedOnOff != 0 ||
		o.IsUnderGuard ||
		o.IsConnOK ||
		o.Status != StatusNormal
}

func hasExplicitLegacyConnectionState(o Object) bool {
	return o.GuardState != 0 ||
		o.AlarmState != 0 ||
		o.TechAlarmState != 0 ||
		o.BlockedArmedOnOff != 0 ||
		o.Status != StatusNormal
}

func (o Object) SeverityValue() VisualSeverity {
	if o.Status == StatusFire {
		return VisualSeverityCritical
	}
	if o.Status == StatusFault {
		return VisualSeverityWarning
	}
	if o.ConnectionStatusValue() == ConnectionStatusOffline || o.Status == StatusOffline {
		return VisualSeverityWarning
	}
	return VisualSeverityNormal
}

func (e Event) VisualSeverityValue() VisualSeverity {
	if e.VisualSeverity != "" && e.VisualSeverity != VisualSeverityUnknown {
		return e.VisualSeverity
	}
	if e.IsCritical() {
		return VisualSeverityCritical
	}
	if e.IsWarning() {
		return VisualSeverityWarning
	}
	switch e.Type {
	case EventNotification, EventOperatorAction, SystemEvent:
		return VisualSeverityInfo
	default:
		return VisualSeverityNormal
	}
}

func (a Alarm) VisualSeverityValue() VisualSeverity {
	if a.VisualSeverity != "" && a.VisualSeverity != VisualSeverityUnknown {
		return a.VisualSeverity
	}
	if a.IsCritical() {
		return VisualSeverityCritical
	}
	switch a.Type {
	case AlarmFault, AlarmPowerFail, AlarmBatteryLow, AlarmOffline, AlarmAcTrouble, AlarmFireTrouble:
		return VisualSeverityWarning
	case AlarmEliminated, AlarmNotification, AlarmSystemEvent:
		return VisualSeverityInfo
	default:
		return VisualSeverityNormal
	}
}
