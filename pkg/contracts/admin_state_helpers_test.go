package contracts

import "testing"

func TestAdminStatisticsRowNormalizedStates(t *testing.T) {
	row := AdminStatisticsRow{
		GuardState:     0,
		IsConnState:    0,
		AlarmState:     0,
		TechAlarmState: 1,
		BlockMode:      DisplayBlockDebug,
	}

	if got := row.GuardStatusValue(); got != FrontendGuardStatusDisarmed {
		t.Fatalf("GuardStatusValue() = %q, want %q", got, FrontendGuardStatusDisarmed)
	}
	if got := row.ConnectionStatusValue(); got != FrontendConnectionStatusOffline {
		t.Fatalf("ConnectionStatusValue() = %q, want %q", got, FrontendConnectionStatusOffline)
	}
	if got := row.MonitoringStatusValue(); got != FrontendMonitoringStatusDebug {
		t.Fatalf("MonitoringStatusValue() = %q, want %q", got, FrontendMonitoringStatusDebug)
	}
	if got := row.VisualSeverityValue(); got != FrontendVisualSeverityWarning {
		t.Fatalf("VisualSeverityValue() = %q, want %q", got, FrontendVisualSeverityWarning)
	}
}

func TestDisplayBlockObjectNormalizedStates(t *testing.T) {
	object := DisplayBlockObject{
		GuardState:     1,
		IsConnState:    1,
		AlarmState:     1,
		TechAlarmState: 0,
		BlockMode:      DisplayBlockTemporaryOff,
	}

	if got := object.GuardStatusValue(); got != FrontendGuardStatusGuarded {
		t.Fatalf("GuardStatusValue() = %q, want %q", got, FrontendGuardStatusGuarded)
	}
	if got := object.ConnectionStatusValue(); got != FrontendConnectionStatusOnline {
		t.Fatalf("ConnectionStatusValue() = %q, want %q", got, FrontendConnectionStatusOnline)
	}
	if got := object.MonitoringStatusValue(); got != FrontendMonitoringStatusBlocked {
		t.Fatalf("MonitoringStatusValue() = %q, want %q", got, FrontendMonitoringStatusBlocked)
	}
	if got := object.VisualSeverityValue(); got != FrontendVisualSeverityCritical {
		t.Fatalf("VisualSeverityValue() = %q, want %q", got, FrontendVisualSeverityCritical)
	}
}
