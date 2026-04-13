package data

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

type shutdownProviderStub struct {
	shutdownCalls int
}

func (s *shutdownProviderStub) GetObjects() []models.Object                          { return nil }
func (s *shutdownProviderStub) GetObjectByID(id string) *models.Object               { return nil }
func (s *shutdownProviderStub) GetEvents() []models.Event                            { return nil }
func (s *shutdownProviderStub) GetObjectEvents(objectID string) []models.Event       { return nil }
func (s *shutdownProviderStub) GetAlarms() []models.Alarm                            { return nil }
func (s *shutdownProviderStub) ProcessAlarm(id string, user string, note string)     {}
func (s *shutdownProviderStub) GetZones(objectID string) []models.Zone               { return nil }
func (s *shutdownProviderStub) GetEmployees(objectID string) []models.Contact        { return nil }
func (s *shutdownProviderStub) GetTestMessages(objectID string) []models.TestMessage { return nil }
func (s *shutdownProviderStub) GetExternalData(objectID string) (string, string, time.Time, time.Time) {
	return "", "", time.Time{}, time.Time{}
}
func (s *shutdownProviderStub) Shutdown() { s.shutdownCalls++ }

func TestCombinedDataProvider_ShutdownPropagatesToSources(t *testing.T) {
	first := &shutdownProviderStub{}
	second := &shutdownProviderStub{}

	provider := NewMultiSourceDataProvider(
		ProviderSource{Name: "one", Provider: first},
		ProviderSource{Name: "two", Provider: second},
	)

	provider.Shutdown()

	if first.shutdownCalls != 1 {
		t.Fatalf("first shutdown calls = %d, want 1", first.shutdownCalls)
	}
	if second.shutdownCalls != 1 {
		t.Fatalf("second shutdown calls = %d, want 1", second.shutdownCalls)
	}
}
