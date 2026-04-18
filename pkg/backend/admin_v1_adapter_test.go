package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

type adminV1StatisticsStub struct {
	filterReceived contracts.AdminStatisticsFilter
	limitReceived  int
	rows           []contracts.AdminStatisticsRow
	rowsErr        error
	types          []contracts.DictionaryItem
	typesErr       error
	regions        []contracts.DictionaryItem
	regionsErr     error
}

func (s *adminV1StatisticsStub) CollectObjectStatistics(filter contracts.AdminStatisticsFilter, limit int) ([]contracts.AdminStatisticsRow, error) {
	s.filterReceived = filter
	s.limitReceived = limit
	return s.rows, s.rowsErr
}

func (s *adminV1StatisticsStub) ListObjectTypes() ([]contracts.DictionaryItem, error) {
	return s.types, s.typesErr
}

func (s *adminV1StatisticsStub) ListObjectDistricts() ([]contracts.DictionaryItem, error) {
	return s.regions, s.regionsErr
}

type adminV1DisplayBlockingStub struct {
	filterReceived string
	objNReceived   int64
	modeReceived   contracts.DisplayBlockMode
	items          []contracts.DisplayBlockObject
	listErr        error
	setErr         error
}

func (s *adminV1DisplayBlockingStub) ListDisplayBlockObjects(filter string) ([]contracts.DisplayBlockObject, error) {
	s.filterReceived = filter
	return s.items, s.listErr
}

func (s *adminV1DisplayBlockingStub) SetDisplayBlockMode(objn int64, mode contracts.DisplayBlockMode) error {
	s.objNReceived = objn
	s.modeReceived = mode
	return s.setErr
}

func TestAdminV1StatisticsProviderCollectObjectStatistics(t *testing.T) {
	base := &adminV1StatisticsStub{
		rows: []contracts.AdminStatisticsRow{
			{
				ObjN:        11,
				GuardState:  1,
				IsConnState: 0,
			},
		},
	}
	provider := NewAdminV1StatisticsProvider(base)
	mode := adminv1.DisplayBlockModeDebug

	rows, err := provider.CollectObjectStatistics(adminv1.StatisticsFilter{
		ConnectionMode: adminv1.StatisticsConnectionModeOffline,
		ProtocolFilter: adminv1.StatisticsProtocolMost,
		BlockMode:      &mode,
		Search:         "school",
	}, 150)
	if err != nil {
		t.Fatalf("CollectObjectStatistics() error = %v", err)
	}

	if base.filterReceived.ConnectionMode != contracts.StatsConnectionOffline {
		t.Fatalf("connection mode = %v, want %v", base.filterReceived.ConnectionMode, contracts.StatsConnectionOffline)
	}
	if base.filterReceived.ProtocolFilter != contracts.StatsProtocolMost {
		t.Fatalf("protocol filter = %q, want %q", base.filterReceived.ProtocolFilter, contracts.StatsProtocolMost)
	}
	if base.filterReceived.BlockMode == nil || *base.filterReceived.BlockMode != contracts.DisplayBlockDebug {
		t.Fatalf("block mode = %+v, want %v", base.filterReceived.BlockMode, contracts.DisplayBlockDebug)
	}
	if base.limitReceived != 150 {
		t.Fatalf("limit = %d, want 150", base.limitReceived)
	}
	if len(rows) != 1 || rows[0].ConnectionStatus != frontendv1.ConnectionStatusOffline {
		t.Fatalf("rows = %+v, want one offline row", rows)
	}
}

func TestAdminV1DisplayBlockingProviderSetDisplayBlockMode(t *testing.T) {
	base := &adminV1DisplayBlockingStub{}
	provider := NewAdminV1DisplayBlockingProvider(base)

	if err := provider.SetDisplayBlockMode(77, adminv1.DisplayBlockModeTemporaryOff); err != nil {
		t.Fatalf("SetDisplayBlockMode() error = %v", err)
	}

	if base.objNReceived != 77 {
		t.Fatalf("objn = %d, want 77", base.objNReceived)
	}
	if base.modeReceived != contracts.DisplayBlockTemporaryOff {
		t.Fatalf("mode = %v, want %v", base.modeReceived, contracts.DisplayBlockTemporaryOff)
	}
}
