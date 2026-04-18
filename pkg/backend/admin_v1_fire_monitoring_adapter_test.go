package backend

import (
	"testing"

	adminv1 "obj_catalog_fyne_v3/pkg/adminapi/v1"
	"obj_catalog_fyne_v3/pkg/contracts"
)

type adminV1FireMonitoringStub struct {
	getResult contracts.FireMonitoringSettings
	getErr    error
	saveInput contracts.FireMonitoringSettings
	saveErr   error
}

func (s *adminV1FireMonitoringStub) GetFireMonitoringSettings() (contracts.FireMonitoringSettings, error) {
	return s.getResult, s.getErr
}

func (s *adminV1FireMonitoringStub) SaveFireMonitoringSettings(settings contracts.FireMonitoringSettings) error {
	s.saveInput = settings
	return s.saveErr
}

func TestAdminV1FireMonitoringProvider(t *testing.T) {
	base := &adminV1FireMonitoringStub{
		getResult: contracts.FireMonitoringSettings{
			Enabled:    true,
			ObjectID:   "fire-1",
			AckWaitSec: 9,
			Servers: []contracts.FireMonitoringServer{
				{Host: "srv", Port: 1234, Enabled: true},
			},
		},
	}
	provider := NewAdminV1FireMonitoringProvider(base)

	settings, err := provider.GetFireMonitoringSettings()
	if err != nil {
		t.Fatalf("GetFireMonitoringSettings() error = %v", err)
	}
	if !settings.Enabled || settings.ObjectID != "fire-1" || len(settings.Servers) != 1 {
		t.Fatalf("settings = %+v, want mapped values", settings)
	}

	if err := provider.SaveFireMonitoringSettings(adminv1.FireMonitoringSettings{
		Enabled:    true,
		ObjectID:   "save-1",
		AckWaitSec: 12,
		Servers: []adminv1.FireMonitoringServer{
			{Host: "save", Port: 4321, Enabled: true},
		},
	}); err != nil {
		t.Fatalf("SaveFireMonitoringSettings() error = %v", err)
	}
	if base.saveInput.ObjectID != "save-1" || len(base.saveInput.Servers) != 1 || base.saveInput.Servers[0].Port != 4321 {
		t.Fatalf("save input = %+v, want mapped contracts settings", base.saveInput)
	}
}
