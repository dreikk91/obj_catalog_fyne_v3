package viewmodels

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/models"
)

type simInventoryReportProviderStub struct {
	objects       []models.Object
	objectsByID   map[string]*models.Object
	caslRows      []map[string]any
	hasCASL       bool
	vodafoneByKey map[string]contracts.VodafoneSIMStatus
	vodafoneList  map[string]contracts.VodafoneSIMInventoryEntry
	kyivstarByKey map[string]contracts.KyivstarSIMStatus
	kyivstarList  map[string]contracts.KyivstarSIMInventoryEntry
}

func (s simInventoryReportProviderStub) GetObjects() []models.Object {
	return append([]models.Object(nil), s.objects...)
}

func (s simInventoryReportProviderStub) GetObjectByID(id string) *models.Object {
	if obj, ok := s.objectsByID[id]; ok && obj != nil {
		copy := *obj
		return &copy
	}
	return nil
}

func (s simInventoryReportProviderStub) GetStatisticReport(ctx context.Context, name string, limit int) ([]map[string]any, error) {
	if name != "stats_devices_v2" {
		return nil, nil
	}
	return append([]map[string]any(nil), s.caslRows...), nil
}

func (s simInventoryReportProviderStub) SupportsCASLReports() bool {
	return s.hasCASL
}

func (s simInventoryReportProviderStub) ListVodafoneSIMInventory() (map[string]contracts.VodafoneSIMInventoryEntry, error) {
	out := make(map[string]contracts.VodafoneSIMInventoryEntry, len(s.vodafoneList))
	for key, value := range s.vodafoneList {
		out[key] = value
	}
	return out, nil
}

func (s simInventoryReportProviderStub) ListKyivstarSIMInventory(numbers []string) (map[string]contracts.KyivstarSIMInventoryEntry, error) {
	out := make(map[string]contracts.KyivstarSIMInventoryEntry, len(s.kyivstarList))
	for key, value := range s.kyivstarList {
		out[key] = value
	}
	return out, nil
}

func (s simInventoryReportProviderStub) GetVodafoneSIMStatus(msisdn string) (contracts.VodafoneSIMStatus, error) {
	return s.vodafoneByKey[NormalizeSIMLookupKey(msisdn)], nil
}

func (s simInventoryReportProviderStub) GetKyivstarSIMStatus(msisdn string) (contracts.KyivstarSIMStatus, error) {
	return s.kyivstarByKey[NormalizeSIMLookupKey(msisdn)], nil
}

func TestBuildSIMInventoryReport_MergesSourcesAndOperatorData(t *testing.T) {
	t.Parallel()

	vm := NewSIMInventoryViewModel()

	phoenixID := 1_000_000_028
	caslID := 1_500_000_024
	phoenixObjectID := strconv.Itoa(phoenixID)

	provider := simInventoryReportProviderStub{
		objects: []models.Object{
			{
				ID:            1001,
				Name:          "Bridge Obj",
				SIM1:          "0501111111",
				SIM2:          "0671111111",
				DisplayNumber: "1001",
			},
			{
				ID:            phoenixID,
				Name:          "Phoenix Obj",
				DisplayNumber: "L00028",
			},
			{
				ID:            1002,
				Name:          "Lifecell Obj",
				SIM1:          "0631234567",
				DisplayNumber: "1002",
			},
			{
				ID:            caslID,
				Name:          "CASL cached",
				DisplayNumber: "1003",
				SIM1:          "0980000000",
			},
		},
		objectsByID: map[string]*models.Object{
			phoenixObjectID: {
				ID:            phoenixID,
				Name:          "Phoenix Obj",
				DisplayNumber: "L00028",
				SIM1:          "80502222222",
			},
		},
		caslRows: []map[string]any{
			{
				"number": 1003,
				"name":   "CASL Obj",
				"sim1":   "983333333",
				"sim2":   nil,
			},
		},
		hasCASL: true,
		vodafoneByKey: map[string]contracts.VodafoneSIMStatus{
			"380501111111": {
				Available:         true,
				Connectivity:      contracts.VodafoneConnectivityStatus{SIMStatus: "active"},
				SubscriberName:    "1001",
				SubscriberComment: "Bridge comment",
			},
			"380502222222": {
				Available:         true,
				Connectivity:      contracts.VodafoneConnectivityStatus{SIMStatus: "active"},
				SubscriberName:    "L00028",
				SubscriberComment: "Phoenix comment",
			},
		},
		vodafoneList: map[string]contracts.VodafoneSIMInventoryEntry{
			"380501111111": {
				MSISDN:            "380501111111",
				SubscriberName:    "1001",
				SubscriberComment: "Bridge comment",
				BlockingStatus:    "NotBlocked",
			},
			"380502222222": {
				MSISDN:            "380502222222",
				SubscriberName:    "L00028",
				SubscriberComment: "Phoenix comment",
				BlockingStatus:    "NotBlocked",
			},
		},
		kyivstarByKey: map[string]contracts.KyivstarSIMStatus{
			"380671111111": {
				Available:    true,
				NumberStatus: "ACTIVE",
				DeviceName:   "KS device",
				DeviceID:     "KS-1001",
				IsOnline:     true,
			},
			"380983333333": {
				Available:    true,
				NumberStatus: "ACTIVE",
				DeviceName:   "CASL device",
				DeviceID:     "CASL-1003",
				IsOnline:     false,
			},
		},
		kyivstarList: map[string]contracts.KyivstarSIMInventoryEntry{
			"380671111111": {
				MSISDN:     "380671111111",
				Status:     "ACTIVE",
				DeviceName: "KS device",
				DeviceID:   "KS-1001",
				IsOnline:   true,
			},
			"380983333333": {
				MSISDN:     "380983333333",
				Status:     "ACTIVE",
				DeviceName: "CASL device",
				DeviceID:   "CASL-1003",
				IsOnline:   false,
			},
		},
	}

	progressStages := make([]string, 0, 8)
	var progressMu sync.Mutex
	result, err := vm.BuildReport(context.Background(), provider, 1001, func(stage string) {
		progressMu.Lock()
		progressStages = append(progressStages, stage)
		progressMu.Unlock()
	})
	if err != nil {
		t.Fatalf("BuildReport() error = %v", err)
	}
	if len(progressStages) == 0 {
		t.Fatal("expected progress callbacks to be emitted")
	}

	if len(result.Rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(result.Rows))
	}
	if !result.VodafoneInventoryLoaded || result.VodafoneInventoryCount != 2 {
		t.Fatalf("unexpected Vodafone inventory summary: %+v", result)
	}
	if !result.KyivstarInventoryLoaded || result.KyivstarInventoryCount != 2 {
		t.Fatalf("unexpected Kyivstar inventory summary: %+v", result)
	}

	bridgeRow := result.Rows[0]
	if bridgeRow.Source != SIMInventorySourceBridge {
		t.Fatalf("unexpected bridge source: %q", bridgeRow.Source)
	}
	if bridgeRow.SIM1 != "380501111111" || bridgeRow.SIM2 != "380671111111" {
		t.Fatalf("bridge numbers were not normalized: %+v", bridgeRow)
	}
	if bridgeRow.SIM1Operator != "Vodafone" || bridgeRow.SIM1Found != "так" || bridgeRow.SIM1Active != "так" {
		t.Fatalf("unexpected bridge SIM1 status: %+v", bridgeRow)
	}
	if bridgeRow.SIM2Operator != "Kyivstar" || bridgeRow.SIM2Name != "KS device" || bridgeRow.SIM2Comment != "KS-1001" {
		t.Fatalf("unexpected bridge SIM2 data: %+v", bridgeRow)
	}

	lifecellRow := result.Rows[1]
	if lifecellRow.Source != SIMInventorySourceBridge {
		t.Fatalf("unexpected lifecell source: %q", lifecellRow.Source)
	}
	if lifecellRow.SIM1 != "380631234567" {
		t.Fatalf("lifecell number was not normalized: %+v", lifecellRow)
	}
	if lifecellRow.SIM1Operator != "lifecell" {
		t.Fatalf("unexpected lifecell operator: %+v", lifecellRow)
	}
	if lifecellRow.SIM1Status != "API недоступне, не перевіряється" {
		t.Fatalf("unexpected lifecell status: %+v", lifecellRow)
	}
	if lifecellRow.SIM1Found != "" || lifecellRow.SIM1Active != "" {
		t.Fatalf("unexpected lifecell flags: %+v", lifecellRow)
	}

	phoenixRow := result.Rows[2]
	if phoenixRow.Source != SIMInventorySourcePhoenix {
		t.Fatalf("unexpected phoenix source: %q", phoenixRow.Source)
	}
	if phoenixRow.SIM1 != "380502222222" {
		t.Fatalf("phoenix SIM1 = %q, want enriched value", phoenixRow.SIM1)
	}

	caslRow := result.Rows[3]
	if caslRow.Source != SIMInventorySourceCASL {
		t.Fatalf("unexpected casl source: %q", caslRow.Source)
	}
	if caslRow.ObjectNumber != "1003" {
		t.Fatalf("casl object number = %q, want 1003", caslRow.ObjectNumber)
	}
	if caslRow.SIM1 != "380983333333" {
		t.Fatalf("casl SIM1 was not normalized: %+v", caslRow)
	}
	if caslRow.SIM1Name != "CASL device" || caslRow.SIM1Comment != "CASL-1003" {
		t.Fatalf("unexpected casl lookup data: %+v", caslRow)
	}

	tsv := vm.BuildTSV(result.Rows)
	if !strings.Contains(tsv, "Оператор SIM 1") {
		t.Fatalf("TSV header is incomplete: %s", tsv)
	}
	if !strings.Contains(tsv, "Bridge Obj") || !strings.Contains(tsv, "CASL Obj") {
		t.Fatalf("TSV does not contain expected object names: %s", tsv)
	}

	summary := vm.FormatSummary(result)
	if !strings.Contains(summary, "Vodafone: 2") || !strings.Contains(summary, "Kyivstar: 2") {
		t.Fatalf("summary does not contain operator counts: %s", summary)
	}

	readyStatus := vm.FormatReadyStatus(result)
	if !strings.Contains(readyStatus, "звіт готовий до експорту") {
		t.Fatalf("ready status does not mention export: %s", readyStatus)
	}
}

func TestNormalizeSIMInventoryNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "already international", input: "380501112233", want: "380501112233"},
		{name: "plus international", input: "+38 (050) 111-22-33", want: "380501112233"},
		{name: "local format", input: "0501112233", want: "380501112233"},
		{name: "legacy 80 format", input: "80501112233", want: "380501112233"},
		{name: "nine digits", input: "501112233", want: "380501112233"},
		{name: "empty", input: "", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := NormalizeSIMInventoryNumber(tt.input); got != tt.want {
				t.Fatalf("NormalizeSIMInventoryNumber(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
