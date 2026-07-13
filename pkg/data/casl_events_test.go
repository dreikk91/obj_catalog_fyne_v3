package data

import (
	"testing"

	"obj_catalog_fyne_v3/pkg/models"
)

func TestCASLCanUseCachedEventsRequiresNonEmptyCache(t *testing.T) {
	if caslCanUseCachedEvents(true, nil) {
		t.Fatal("subscribed realtime with an empty cache must bootstrap the HTTP journal")
	}
	if !caslCanUseCachedEvents(true, []models.Event{{ID: 1}}) {
		t.Fatal("subscribed realtime with cached events should use the cache")
	}
}
