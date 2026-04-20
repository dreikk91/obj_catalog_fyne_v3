package webfrontend

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewHandlerServesIndex(t *testing.T) {
	handler, err := NewHandler("/api/frontend/v1")
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "/api/frontend/v1") {
		t.Fatalf("index must contain api base path, got %q", rec.Body.String())
	}
}

func TestNewHandlerServesAssets(t *testing.T) {
	handler, err := NewHandler("/api/frontend/v1")
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/javascript") {
		t.Fatalf("unexpected content type: %q", ct)
	}
	if !strings.Contains(rec.Body.String(), "loadInitialData") {
		t.Fatalf("app.js body does not look correct")
	}
}

func TestNewHandlerUnknownRouteReturnsNotFound(t *testing.T) {
	handler, err := NewHandler("/api/frontend/v1")
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}
