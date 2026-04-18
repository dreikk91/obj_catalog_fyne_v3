package v1

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
)

func TestFromObjectUpsertRequest(t *testing.T) {
	coeff := 1.5
	request := ObjectUpsertRequest{
		Source:   SourceCASL,
		ObjectID: 42,
		Core: ObjectCoreFields{
			Name:        "obj",
			Address:     "addr",
			Contract:    "c-1",
			Description: "desc",
			Notes:       "notes",
			Latitude:    "50.0",
			Longitude:   "30.0",
		},
		Legacy: &LegacyObjectPayload{
			ObjN:      101,
			ShortName: "legacy",
		},
		CASL: &CASLObjectPayload{
			ObjID:         "casl-1",
			Status:        "active",
			BusinessCoeff: &coeff,
		},
	}

	got := FromObjectUpsertRequest(request)

	if got.Source != contracts.FrontendSourceCASL {
		t.Fatalf("source = %q, want %q", got.Source, contracts.FrontendSourceCASL)
	}
	if got.ObjectID != 42 {
		t.Fatalf("object id = %d, want 42", got.ObjectID)
	}
	if got.Core.Description != "desc" {
		t.Fatalf("core description = %q, want %q", got.Core.Description, "desc")
	}
	if got.Legacy == nil || got.Legacy.ObjN != 101 {
		t.Fatalf("legacy payload = %+v, want ObjN=101", got.Legacy)
	}
	if got.CASL == nil || got.CASL.ObjID != "casl-1" {
		t.Fatalf("casl payload = %+v, want ObjID=casl-1", got.CASL)
	}
	if got.CASL.BusinessCoeff == nil || *got.CASL.BusinessCoeff != coeff {
		t.Fatalf("business coeff = %+v, want %v", got.CASL.BusinessCoeff, coeff)
	}
}

func TestToObjectDetails(t *testing.T) {
	now := time.Now().UTC()
	got := ToObjectDetails(contracts.FrontendObjectDetails{
		Summary: contracts.FrontendObjectSummary{
			ID:               11,
			Source:           contracts.FrontendSourceBridge,
			Name:             "Школа",
			GuardStatus:      contracts.FrontendGuardStatusGuarded,
			ConnectionStatus: contracts.FrontendConnectionStatusOnline,
			MonitoringStatus: contracts.FrontendMonitoringStatusActive,
		},
		ExternalSignal: "GPRS",
		Zones: []contracts.FrontendZone{
			{Number: 1, Name: "Вхід"},
		},
		Contacts: []contracts.FrontendContact{
			{Name: "Іван", Priority: 1},
		},
		Events: []contracts.FrontendEventItem{
			{ID: 5, Time: now, VisualSeverity: contracts.FrontendVisualSeverityWarning},
		},
	})

	if got.Summary.Source != SourceBridge {
		t.Fatalf("summary source = %q, want %q", got.Summary.Source, SourceBridge)
	}
	if got.Summary.GuardStatus != GuardStatusGuarded {
		t.Fatalf("guard status = %q, want %q", got.Summary.GuardStatus, GuardStatusGuarded)
	}
	if len(got.Zones) != 1 || got.Zones[0].Name != "Вхід" {
		t.Fatalf("zones = %+v, want one zone named Вхід", got.Zones)
	}
	if len(got.Contacts) != 1 || got.Contacts[0].Name != "Іван" {
		t.Fatalf("contacts = %+v, want one contact named Іван", got.Contacts)
	}
	if len(got.Events) != 1 || got.Events[0].VisualSeverity != VisualSeverityWarning {
		t.Fatalf("events = %+v, want one warning event", got.Events)
	}
}
