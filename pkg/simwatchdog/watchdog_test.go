package simwatchdog

import (
	"context"
	"errors"
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestShouldHandleObjectRequiresGPRSOfflineGuardedObject(t *testing.T) {
	r := &Runner{}

	obj := models.Object{
		ID:               1001,
		ObjChan:          5,
		ConnectionStatus: models.ConnectionStatusOffline,
		GuardStatus:      models.GuardStatusGuarded,
		MonitoringStatus: models.MonitoringStatusActive,
	}

	if !r.shouldHandleObject(obj) {
		t.Fatal("expected GPRS offline guarded object to be handled")
	}
}

func TestHasRecentLastTest(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	obj := models.Object{ID: 1001}

	tests := []struct {
		name     string
		provider *lastTestObjectProvider
		want     bool
	}{
		{
			name: "recent",
			provider: &lastTestObjectProvider{
				lastTest: map[int]time.Time{1001: now.Add(-6 * 24 * time.Hour)},
			},
			want: true,
		},
		{
			name: "older than seven days",
			provider: &lastTestObjectProvider{
				lastTest: map[int]time.Time{1001: now.Add(-8 * 24 * time.Hour)},
			},
			want: false,
		},
		{
			name: "empty last test",
			provider: &lastTestObjectProvider{
				lastTest: map[int]time.Time{1001: time.Time{}},
			},
			want: false,
		},
		{
			name: "read error",
			provider: &lastTestObjectProvider{
				err: errors.New("read failed"),
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Runner{
				objects: tt.provider,
				options: Options{MaxLastTestAge: 7 * 24 * time.Hour},
			}
			if got := r.hasRecentLastTest(context.Background(), obj, now); got != tt.want {
				t.Fatalf("hasRecentLastTest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasRecentLastTestRejectsProviderWithoutLastTestSupport(t *testing.T) {
	r := &Runner{
		objects: objectProviderOnly{},
		options: Options{MaxLastTestAge: 7 * 24 * time.Hour},
	}
	if r.hasRecentLastTest(context.Background(), models.Object{ID: 1001}, time.Now()) {
		t.Fatal("expected object to be rejected when last test provider is unavailable")
	}
}

type lastTestObjectProvider struct {
	lastTest map[int]time.Time
	err      error
}

func (p *lastTestObjectProvider) GetObjects() []models.Object {
	return nil
}

func (p *lastTestObjectProvider) LastGPRSTestTime(_ context.Context, objectID int) (time.Time, error) {
	if p.err != nil {
		return time.Time{}, p.err
	}
	return p.lastTest[objectID], nil
}

type objectProviderOnly struct{}

func (objectProviderOnly) GetObjects() []models.Object {
	return nil
}

func TestShouldHandleObjectRejectsWrongState(t *testing.T) {
	r := &Runner{}

	tests := []struct {
		name string
		obj  models.Object
	}{
		{
			name: "not gprs",
			obj: models.Object{
				ID:               1001,
				ObjChan:          1,
				ConnectionStatus: models.ConnectionStatusOffline,
				GuardStatus:      models.GuardStatusGuarded,
				MonitoringStatus: models.MonitoringStatusActive,
			},
		},
		{
			name: "online",
			obj: models.Object{
				ID:               1001,
				ObjChan:          5,
				ConnectionStatus: models.ConnectionStatusOnline,
				GuardStatus:      models.GuardStatusGuarded,
				MonitoringStatus: models.MonitoringStatusActive,
			},
		},
		{
			name: "disarmed",
			obj: models.Object{
				ID:               1001,
				ObjChan:          5,
				ConnectionStatus: models.ConnectionStatusOffline,
				GuardStatus:      models.GuardStatusDisarmed,
				MonitoringStatus: models.MonitoringStatusBlocked,
			},
		},
		{
			name: "debug",
			obj: models.Object{
				ID:               1001,
				ObjChan:          5,
				ConnectionStatus: models.ConnectionStatusOffline,
				GuardStatus:      models.GuardStatusGuarded,
				MonitoringStatus: models.MonitoringStatusDebug,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if r.shouldHandleObject(tt.obj) {
				t.Fatalf("expected object to be rejected: %+v", tt.obj)
			}
		})
	}
}
