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
	case strings.HasPrefix(path, APIV1BasePath+"/alarms/"):
		h.handleAlarmItem(w, r, strings.TrimPrefix(path, APIV1BasePath+"/alarms/"))
	case path == APIV1BasePath+"/alarm-groups":
		h.handleAlarmGroups(w, r)
	case path == APIV1BasePath+"/alarm-processing-options":
		h.handleAlarmProcessingOptionsCached(w, r)
	case path == APIV1BasePath+"/response-groups":
		h.handleResponseGroups(w, r)
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
	path := strings.Trim(strings.TrimSpace(rawID), "/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "invalid object id")
		return
	}

	parts := strings.Split(path, "/")
	objectID, ok := parseObjectID(parts[0])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid object id")
		return
	}

	if len(parts) > 1 {
		if len(parts) == 2 && parts[1] == "events" {
			h.handleObjectEvents(w, r, objectID)
			return
		}
		if len(parts) == 2 && parts[1] == "standby" {
			h.handleObjectStandby(w, r, objectID)
			return
		}
		writeError(w, http.StatusNotFound, "route not found")
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

func (h *Handler) handleObjectEvents(w http.ResponseWriter, r *http.Request, objectID int) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	offset, err := parseQueryInt(r, "offset", 0)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	limit, err := parseQueryInt(r, "limit", 100)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if offset < 0 {
		writeError(w, http.StatusBadRequest, "offset must be non-negative")
		return
	}
	if limit <= 0 {
		writeError(w, http.StatusBadRequest, "limit must be positive")
		return
	}

	page, err := h.backend.ListObjectEvents(r.Context(), objectID, offset, limit)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToEventPageResponse(page))
}

func (h *Handler) handleObjectStandby(w http.ResponseWriter, r *http.Request, objectID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	var req struct {
		DurationMinutes int    `json:"durationMinutes"`
		Reason          string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.backend.StandbyObject(r.Context(), objectID, contracts.FrontendStandbyRequest{
		DurationMinutes: req.DurationMinutes,
		Reason:          req.Reason,
	}); err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, struct{}{})
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

func (h *Handler) handleAlarmItem(w http.ResponseWriter, r *http.Request, rawID string) {
	path := strings.Trim(strings.TrimSpace(rawID), "/")
	if path == "" {
		writeError(w, http.StatusBadRequest, "invalid alarm id")
		return
	}

	parts := strings.Split(path, "/")
	alarmID, ok := parseObjectID(parts[0])
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid alarm id")
		return
	}

	if len(parts) != 2 {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}

	switch parts[1] {
	case "processing-options":
		h.handleAlarmProcessingOptions(w, r, alarmID)
	case "pick":
		h.handleAlarmPick(w, r, alarmID)
	case "process":
		h.handleAlarmProcess(w, r, alarmID)
	case "assign-group":
		h.handleAlarmAssignGroup(w, r, alarmID)
	case "group-arrived":
		h.handleAlarmGroupArrived(w, r, alarmID)
	case "cancel-group":
		h.handleAlarmCancelGroup(w, r, alarmID)
	case "group-process":
		h.handleAlarmGroupProcess(w, r, alarmID)
	default:
		writeError(w, http.StatusNotFound, "route not found")
	}
}

func (h *Handler) handleAlarmProcessingOptionsCached(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.ListAlarmProcessingOptionsCached(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToAlarmProcessingOptionsResponse(items))
}

func (h *Handler) handleResponseGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.ListResponseGroups(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToResponseGroupListResponse(items))
}

func (h *Handler) handleAlarmProcessingOptions(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.GetAlarmProcessingOptions(r.Context(), alarmID)
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToAlarmProcessingOptionsResponse(items))
}

func (h *Handler) handleAlarmGroupProcess(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	type groupProcessRequest struct {
		User string `json:"User"`
	}
	var req groupProcessRequest
	if r.Body != nil {
		defer r.Body.Close()
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if err := h.backend.GroupProcessAlarm(r.Context(), alarmID, req.User); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAlarmPick(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	request, ok := decodeAlarmPickRequest(w, r)
	if !ok {
		return
	}

	if err := h.backend.PickAlarm(r.Context(), alarmID, frontendv1.FromAlarmPickRequest(request)); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAlarmProcess(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	request, ok := decodeAlarmProcessRequest(w, r)
	if !ok {
		return
	}

	if err := h.backend.ProcessAlarm(r.Context(), alarmID, frontendv1.FromAlarmProcessRequest(request)); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAlarmAssignGroup(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	request, ok := decodeAlarmGroupActionRequest(w, r)
	if !ok {
		return
	}

	if err := h.backend.AssignResponseGroup(r.Context(), alarmID, frontendv1.FromAlarmGroupActionRequest(request)); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAlarmGroupArrived(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if err := h.backend.NotifyGroupArrived(r.Context(), alarmID); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAlarmCancelGroup(w http.ResponseWriter, r *http.Request, alarmID int) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	if err := h.backend.CancelResponseGroup(r.Context(), alarmID); err != nil {
		writeBackendError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeAlarmGroupActionRequest(w http.ResponseWriter, r *http.Request) (frontendv1.AlarmGroupActionRequest, bool) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return frontendv1.AlarmGroupActionRequest{}, false
	}
	defer r.Body.Close()

	var request frontendv1.AlarmGroupActionRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return frontendv1.AlarmGroupActionRequest{}, false
	}
	if decoder.More() {
		writeError(w, http.StatusBadRequest, "request body must contain a single json object")
		return frontendv1.AlarmGroupActionRequest{}, false
	}
	return request, true
}

func decodeAlarmPickRequest(w http.ResponseWriter, r *http.Request) (frontendv1.AlarmPickRequest, bool) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return frontendv1.AlarmPickRequest{}, false
	}
	defer r.Body.Close()

	var request frontendv1.AlarmPickRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return frontendv1.AlarmPickRequest{}, false
	}
	if decoder.More() {
		writeError(w, http.StatusBadRequest, "request body must contain a single json object")
		return frontendv1.AlarmPickRequest{}, false
	}
	return request, true
}

func (h *Handler) handleAlarmGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w, http.MethodGet)
		return
	}

	items, err := h.backend.ListAlarms(r.Context())
	if err != nil {
		writeBackendError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, frontendv1.ToAlarmGroupListResponse(items))
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

func decodeAlarmProcessRequest(w http.ResponseWriter, r *http.Request) (frontendv1.AlarmProcessRequest, bool) {
	if r.Body == nil {
		writeError(w, http.StatusBadRequest, "request body is required")
		return frontendv1.AlarmProcessRequest{}, false
	}
	defer r.Body.Close()

	var request frontendv1.AlarmProcessRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return frontendv1.AlarmProcessRequest{}, false
	}
	if decoder.More() {
		writeError(w, http.StatusBadRequest, "request body must contain a single json object")
		return frontendv1.AlarmProcessRequest{}, false
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

func parseQueryInt(r *http.Request, key string, defaultValue int) (int, error) {
	if r == nil || r.URL == nil {
		return defaultValue, nil
	}

	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("invalid " + key)
	}
	return value, nil
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
