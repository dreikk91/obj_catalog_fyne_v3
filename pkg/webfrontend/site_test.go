package webfrontend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"obj_catalog_fyne_v3/pkg/contracts"
)

type siteBackendStub struct{}

func (siteBackendStub) Capabilities(context.Context) (contracts.FrontendCapabilities, error) {
	return contracts.FrontendCapabilities{}, nil
}

func (siteBackendStub) ListObjects(context.Context) ([]contracts.FrontendObjectSummary, error) {
	return []contracts.FrontendObjectSummary{{ID: 1001, Name: "Test"}}, nil
}

func (siteBackendStub) ListAlarms(context.Context) ([]contracts.FrontendAlarmItem, error) {
	return nil, nil
}

func (siteBackendStub) GetAlarmProcessingOptions(context.Context, int) ([]contracts.FrontendAlarmProcessingOption, error) {
	return nil, nil
}

func (siteBackendStub) PickAlarm(context.Context, int, contracts.FrontendAlarmPickRequest) error {
	return nil
}

func (siteBackendStub) ProcessAlarm(context.Context, int, contracts.FrontendAlarmProcessRequest) error {
	return nil
}

func (siteBackendStub) ListEvents(context.Context) ([]contracts.FrontendEventItem, error) {
	return nil, nil
}

func (siteBackendStub) ListObjectEvents(context.Context, int, int, int) (contracts.FrontendEventPage, error) {
	return contracts.FrontendEventPage{}, nil
}

func (siteBackendStub) GetObjectDetails(context.Context, int) (contracts.FrontendObjectDetails, error) {
	return contracts.FrontendObjectDetails{}, nil
}

func (siteBackendStub) CreateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

func (siteBackendStub) UpdateObject(context.Context, contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	return contracts.FrontendObjectMutationResult{}, nil
}

func TestNewSiteHandlerRoutesUIAndAPI(t *testing.T) {
	handler, err := NewSiteHandler(siteBackendStub{})
	if err != nil {
		t.Fatalf("NewSiteHandler error: %v", err)
	}

	apiReq := httptest.NewRequest(http.MethodGet, "/api/frontend/v1/objects", nil)
	apiRec := httptest.NewRecorder()
	handler.ServeHTTP(apiRec, apiReq)
	if apiRec.Code != http.StatusOK {
		t.Fatalf("api status = %d", apiRec.Code)
	}
	if !strings.Contains(apiRec.Body.String(), "\"items\"") {
		t.Fatalf("api body = %q", apiRec.Body.String())
	}

	uiReq := httptest.NewRequest(http.MethodGet, "/", nil)
	uiRec := httptest.NewRecorder()
	handler.ServeHTTP(uiRec, uiReq)
	if uiRec.Code != http.StatusOK {
		t.Fatalf("ui status = %d", uiRec.Code)
	}
	if !strings.Contains(uiRec.Body.String(), "Дежурний оператор") {
		t.Fatalf("ui body = %q", uiRec.Body.String())
	}
}
