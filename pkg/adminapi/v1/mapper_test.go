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

func TestToAccessStatusAndDataCheckIssues(t *testing.T) {
	status := ToAccessStatus(contracts.AdminAccessStatus{
		CurrentUser:      "user",
		HasFullAccess:    true,
		AdminUsersCount:  2,
		MatchDescription: "ok",
	})
	if status.CurrentUser != "user" || !status.HasFullAccess || status.AdminUsersCount != 2 {
		t.Fatalf("status = %+v, want mapped access status", status)
	}

	issues := ToDataCheckIssues([]contracts.AdminDataCheckIssue{{Severity: "warn", Code: "C1", ObjN: 7, Details: "demo"}})
	if len(issues) != 1 || issues[0].Code != "C1" || issues[0].ObjN != 7 {
		t.Fatalf("issues = %+v, want one mapped issue", issues)
	}
}

func TestToSubServersAndObjects(t *testing.T) {
	servers := ToSubServers([]contracts.AdminSubServer{{ID: 1, Bind: "a", Host: "host", Type: 2}})
	if len(servers) != 1 || servers[0].Bind != "a" || servers[0].Type != 2 {
		t.Fatalf("servers = %+v, want one mapped server", servers)
	}

	objects := ToSubServerObjects([]contracts.AdminSubServerObject{{ObjN: 11, Name: "obj", SubServerA: "a"}})
	if len(objects) != 1 || objects[0].ObjN != 11 || objects[0].SubServerA != "a" {
		t.Fatalf("objects = %+v, want one mapped object", objects)
	}
}

func TestToPPKConstructorItems(t *testing.T) {
	items := ToPPKConstructorItems([]contracts.PPKConstructorItem{{ID: 3, Name: "demo", Channel: 7, ZoneCount: 16}})
	if len(items) != 1 || items[0].ID != 3 || items[0].Channel != 7 || items[0].ZoneCount != 16 {
		t.Fatalf("items = %+v, want one mapped PPK item", items)
	}
}

func TestFireMonitoringSettingsRoundTrip(t *testing.T) {
	original := contracts.FireMonitoringSettings{
		Enabled:       true,
		ObjectID:      "fire-1",
		AckWaitSec:    15,
		UseStdDateFmt: false,
		Servers: []contracts.FireMonitoringServer{
			{Host: "host", Port: 1234, Info: "ДСНС", Enabled: true},
		},
	}

	mapped := ToFireMonitoringSettings(original)
	if mapped.ObjectID != "fire-1" || len(mapped.Servers) != 1 || mapped.Servers[0].Info != "ДСНС" {
		t.Fatalf("mapped = %+v, want round-tripped values", mapped)
	}

	roundTrip := ToContractsFireMonitoringSettings(mapped)
	if roundTrip.AckWaitSec != 15 || len(roundTrip.Servers) != 1 || roundTrip.Servers[0].Port != 1234 {
		t.Fatalf("roundTrip = %+v, want original values", roundTrip)
	}
}

func TestObjectAdminRoundTrip(t *testing.T) {
	card := contracts.AdminObjectCard{
		ObjUIN:             10,
		ObjN:               1001,
		ShortName:          "School",
		ObjTypeID:          7,
		ObjRegID:           3,
		GSMPhone1:          "380671234567",
		GSMHiddenN:         55,
		TestControlEnabled: true,
		TestIntervalMin:    30,
	}
	mappedCard := ToObjectCard(card)
	if mappedCard.ObjN != 1001 || !mappedCard.TestControlEnabled || mappedCard.ObjTypeID != 7 {
		t.Fatalf("mappedCard = %+v, want mapped object card", mappedCard)
	}
	if roundTrip := ToContractsObjectCard(mappedCard); roundTrip.GSMHiddenN != 55 || roundTrip.ObjRegID != 3 {
		t.Fatalf("roundTrip card = %+v, want original values", roundTrip)
	}

	personal := contracts.AdminObjectPersonal{
		ID:         1,
		SourceObjN: 1001,
		Number:     2,
		Surname:    "Petrenko",
		Name:       "Ivan",
		Access1:    1,
		IsRang:     true,
	}
	mappedPersonal := ToObjectPersonal(personal)
	if mappedPersonal.Name != "Ivan" || mappedPersonal.Access1 != 1 || !mappedPersonal.IsRang {
		t.Fatalf("mappedPersonal = %+v, want mapped personal", mappedPersonal)
	}
	if roundTrip := ToContractsObjectPersonal(mappedPersonal); roundTrip.Surname != "Petrenko" || roundTrip.SourceObjN != 1001 {
		t.Fatalf("roundTrip personal = %+v, want original values", roundTrip)
	}

	zone := contracts.AdminObjectZone{ID: 8, ZoneNumber: 4, Description: "Hall"}
	mappedZone := ToObjectZone(zone)
	if mappedZone.ZoneNumber != 4 || mappedZone.Description != "Hall" {
		t.Fatalf("mappedZone = %+v, want mapped zone", mappedZone)
	}
	if roundTrip := ToContractsObjectZone(mappedZone); roundTrip.ID != 8 || roundTrip.ZoneNumber != 4 {
		t.Fatalf("roundTrip zone = %+v, want original values", roundTrip)
	}

	coords := contracts.AdminObjectCoordinates{Latitude: "49.1", Longitude: "24.1"}
	mappedCoords := ToObjectCoordinates(coords)
	if mappedCoords.Latitude != "49.1" || mappedCoords.Longitude != "24.1" {
		t.Fatalf("mappedCoords = %+v, want mapped coordinates", mappedCoords)
	}
	if roundTrip := ToContractsObjectCoordinates(mappedCoords); roundTrip.Longitude != "24.1" {
		t.Fatalf("roundTrip coords = %+v, want original values", roundTrip)
	}

	simUsages := ToSIMPhoneUsages([]contracts.AdminSIMPhoneUsage{{ObjN: 1001, Name: "School", Slot: "SIM1"}})
	if len(simUsages) != 1 || simUsages[0].Slot != "SIM1" {
		t.Fatalf("simUsages = %+v, want mapped usage", simUsages)
	}
}
