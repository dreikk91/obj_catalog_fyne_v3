package v1

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

func TestToStatisticsFilter(t *testing.T) {
	mode := DisplayBlockModeDebug
	filter := StatisticsFilter{
		ConnectionMode: StatisticsConnectionModeOffline,
		ProtocolFilter: StatisticsProtocolMost,
		BlockMode:      &mode,
		Search:         "school",
	}

	got := ToStatisticsFilter(filter)

	if got.ConnectionMode != contracts.StatsConnectionOffline {
		t.Fatalf("connection mode = %v, want %v", got.ConnectionMode, contracts.StatsConnectionOffline)
	}
	if got.ProtocolFilter != contracts.StatsProtocolMost {
		t.Fatalf("protocol filter = %q, want %q", got.ProtocolFilter, contracts.StatsProtocolMost)
	}
	if got.BlockMode == nil || *got.BlockMode != contracts.DisplayBlockDebug {
		t.Fatalf("block mode = %+v, want %v", got.BlockMode, contracts.DisplayBlockDebug)
	}
}

func TestToStatisticsRow(t *testing.T) {
	got := ToStatisticsRow(contracts.AdminStatisticsRow{
		ObjN:           11,
		ShortName:      "obj",
		GuardState:     1,
		IsConnState:    0,
		AlarmState:     1,
		TechAlarmState: 0,
		BlockMode:      contracts.DisplayBlockTemporaryOff,
	})

	if got.BlockMode != DisplayBlockModeTemporaryOff {
		t.Fatalf("block mode = %q, want %q", got.BlockMode, DisplayBlockModeTemporaryOff)
	}
	if got.GuardStatus != frontendv1.GuardStatusGuarded {
		t.Fatalf("guard status = %q, want %q", got.GuardStatus, frontendv1.GuardStatusGuarded)
	}
	if got.ConnectionStatus != frontendv1.ConnectionStatusOffline {
		t.Fatalf("connection status = %q, want %q", got.ConnectionStatus, frontendv1.ConnectionStatusOffline)
	}
	if got.MonitoringStatus != frontendv1.MonitoringStatusBlocked {
		t.Fatalf("monitoring status = %q, want %q", got.MonitoringStatus, frontendv1.MonitoringStatusBlocked)
	}
	if got.VisualSeverity != frontendv1.VisualSeverityCritical {
		t.Fatalf("visual severity = %q, want %q", got.VisualSeverity, frontendv1.VisualSeverityCritical)
	}
}

func TestToDisplayBlockObject(t *testing.T) {
	got := ToDisplayBlockObject(contracts.DisplayBlockObject{
		ObjN:           17,
		Name:           "demo",
		BlockMode:      contracts.DisplayBlockDebug,
		GuardState:     0,
		IsConnState:    1,
		AlarmState:     0,
		TechAlarmState: 1,
	})

	if got.BlockMode != DisplayBlockModeDebug {
		t.Fatalf("block mode = %q, want %q", got.BlockMode, DisplayBlockModeDebug)
	}
	if got.GuardStatus != frontendv1.GuardStatusDisarmed {
		t.Fatalf("guard status = %q, want %q", got.GuardStatus, frontendv1.GuardStatusDisarmed)
	}
	if got.ConnectionStatus != frontendv1.ConnectionStatusOnline {
		t.Fatalf("connection status = %q, want %q", got.ConnectionStatus, frontendv1.ConnectionStatusOnline)
	}
	if got.VisualSeverity != frontendv1.VisualSeverityWarning {
		t.Fatalf("visual severity = %q, want %q", got.VisualSeverity, frontendv1.VisualSeverityWarning)
	}
}

func TestToContractsMessage220VMode(t *testing.T) {
	if got := ToContractsMessage220VMode(Message220VModeAlarm); got != contracts.Admin220VAlarm {
		t.Fatalf("alarm mode = %v, want %v", got, contracts.Admin220VAlarm)
	}
	if got := ToContractsMessage220VMode(Message220VModeRestore); got != contracts.Admin220VRestore {
		t.Fatalf("restore mode = %v, want %v", got, contracts.Admin220VRestore)
	}
	if got := ToContractsMessage220VMode(Message220VModeNone); got != contracts.Admin220VNone {
		t.Fatalf("none mode = %v, want %v", got, contracts.Admin220VNone)
	}
}

func TestToMessage220VBuckets(t *testing.T) {
	got := ToMessage220VBuckets(contracts.Admin220VMessageBuckets{
		Free:    []contracts.AdminMessage{{UIN: 1, Text: "free"}},
		Alarm:   []contracts.AdminMessage{{UIN: 2, Text: "alarm"}},
		Restore: []contracts.AdminMessage{{UIN: 3, Text: "restore"}},
	})

	if len(got.Free) != 1 || got.Free[0].UIN != 1 {
		t.Fatalf("free = %+v, want one item with UIN=1", got.Free)
	}
	if len(got.Alarm) != 1 || got.Alarm[0].Text != "alarm" {
		t.Fatalf("alarm = %+v, want one item with text alarm", got.Alarm)
	}
	if len(got.Restore) != 1 || got.Restore[0].UIN != 3 {
		t.Fatalf("restore = %+v, want one item with UIN=3", got.Restore)
	}
}
