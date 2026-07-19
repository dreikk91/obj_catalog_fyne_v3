package data

import (
	"testing"
	"time"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestCASLCanUseCachedEventsRequiresFreshHTTPReconciliation(t *testing.T) {
	now := time.Now()
	cached := []models.Event{{ID: 1}}

	if caslCanUseCachedEvents(true, nil, now, now) {
		t.Fatal("subscribed realtime with an empty cache must bootstrap the HTTP journal")
	}
	if caslCanUseCachedEvents(true, cached, time.Time{}, now) {
		t.Fatal("subscribed realtime without an HTTP reconciliation must bootstrap the HTTP journal")
	}
	if caslCanUseCachedEvents(true, cached, now.Add(-caslEventsHTTPReconcileInterval), now) {
		t.Fatal("subscribed realtime with stale HTTP reconciliation must refresh the HTTP journal")
	}
	if !caslCanUseCachedEvents(true, cached, now.Add(-time.Second), now) {
		t.Fatal("subscribed realtime with a fresh HTTP reconciliation should use the cache")
	}
}
