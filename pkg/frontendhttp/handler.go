package frontendhttp

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

const (
	APIV1BasePath = "/api/frontend/v1"
	contentType   = "application/json; charset=utf-8"
)

type Handler struct {
	backend contracts.FrontendBackend
}

func NewHandler(backend contracts.FrontendBackend) http.Handler {
	return &Handler{backend: backend}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.backend == nil {
		writeError(w, http.StatusServiceUnavailable, "frontend backend is unavailable")
		return
	}

	path := strings.TrimSuffix(strings.TrimSpace(r.URL.Path), "/")
	switch {
	case path == APIV1BasePath+"/capabilities":
		h.handleCapabilities(w, r)
	case path == APIV1BasePath+"/objects":
		h.handleObjectsCollection(w, r)
	case strings.HasPrefix(path, APIV1BasePath+"/objects/"):
		h.handleObjectItem(w, r, strings.TrimPrefix(path, APIV1BasePath+"/objects/"))
	case path == APIV1BasePath+"/alarms":
		h.handleAlarms(w, r)
	case path == APIV1BasePath+"/events":
		h.handleEvents(w, r)
	default:
		writeError(w, http.StatusNotFound, "route not found")
	}
}

func (h *Handler) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	result, err := h.backend.Capabilities(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToCapabilities(result))
}

func (h *Handler) handleObjectsCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.backend.ListObjects(r.Context())
		if err != nil {
			writeBackendError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, frontendv1.ToObjectListResponse(items))
	case http.MethodPost:
		request, ok := decodeUpsertRequest(w, r)
		if !ok {
			return
		}
		result, err := h.backend.CreateObject(r.Context(), frontendv1.FromObjectUpsertRequest(request))
		if err != nil {
			writeBackendError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, frontendv1.ToObjectMutationResult(result))
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (h *Handler) handleObjectItem(w http.ResponseWriter, r *http.Request, rawID string) {
	objectID, ok := parseObjectID(rawID)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid object id")
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, err := h.backend.GetObjectDetails(r.Context(), objectID)
		if err != nil {
			writeBackendError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, frontendv1.ToObjectDetails(item))
	case http.MethodPut:
		request, ok := decodeUpsertRequest(w, r)
		if !ok {
			return
		}
		request.ObjectID = objectID
		result, err := h.backend.UpdateObject(r.Context(), frontendv1.FromObjectUpsertRequest(request))
		if err != nil {
			writeBackendError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, frontendv1.ToObjectMutationResult(result))
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPut)
	}
}

func (h *Handler) handleAlarms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.ListAlarms(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToAlarmListResponse(items))
}

func (h *Handler) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.ListEvents(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToEventListResponse(items))
}

func decodeUpsertRequest(w http.ResponseWriter, r *http.Request) (frontendv1.ObjectUpsertRequest, bool) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return frontendv1.ObjectUpsertRequest{}, false
	}
	defer r.Body.Close()

	var request frontendv1.ObjectUpsertRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return frontendv1.ObjectUpsertRequest{}, false
	}
	if decoder.More() {
		writeError(w, http.StatusBadRequest, "request body must contain a single json object")
		return frontendv1.ObjectUpsertRequest{}, false
	}
	return request, true
}

func parseObjectID(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func writeBackendError(w http.ResponseWriter, err error) {
	switch {
	case err == nil:
		writeError(w, http.StatusInternalServerError, "unknown backend error")
	case errors.Is(err, contracts.ErrFrontendBackendUnavailable):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case errors.Is(err, contracts.ErrUnsupportedFrontendSource):
		writeError(w, http.StatusNotImplemented, err.Error())
	case errors.Is(err, contracts.ErrMissingLegacyObjectPayload),
		errors.Is(err, contracts.ErrMissingCASLObjectPayload):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		message := strings.TrimSpace(err.Error())
		if message == "" {
			message = "backend request failed"
		}
		status := http.StatusBadRequest
		if strings.Contains(strings.ToLower(message), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, message)
	}
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	if len(methods) > 0 {
		w.Header().Set("Allow", strings.Join(methods, ", "))
	}
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, frontendv1.ErrorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, `{"error":"failed to encode response"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
