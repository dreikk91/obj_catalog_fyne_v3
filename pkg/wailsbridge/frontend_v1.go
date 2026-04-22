package wailsbridge

import (
	"context"
	"errors"
	"sync"

	"obj_catalog_fyne_v3/pkg/contracts"
	frontendv1 "obj_catalog_fyne_v3/pkg/frontendapi/v1"
)

var ErrFrontendBackendUnavailable = errors.New("wailsbridge: frontend backend is unavailable")

// FrontendV1Service exposes versioned frontend API methods for Wails bindings.
type FrontendV1Service struct {
	mu      sync.RWMutex
	backend contracts.FrontendBackend
}

func NewFrontendV1Service(backend contracts.FrontendBackend) *FrontendV1Service {
	return &FrontendV1Service{backend: backend}
}

func (s *FrontendV1Service) SetBackend(backend contracts.FrontendBackend) {
	s.mu.Lock()
	s.backend = backend
	s.mu.Unlock()
}

func (s *FrontendV1Service) IsReady() bool {
	_, err := s.backendOrErr()
	return err == nil
}

func (s *FrontendV1Service) Capabilities() (frontendv1.Capabilities, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return frontendv1.Capabilities{}, err
	}

	result, err := backend.Capabilities(context.Background())
	if err != nil {
		return frontendv1.Capabilities{}, err
	}
	return frontendv1.ToCapabilities(result), nil
}

func (s *FrontendV1Service) ListObjects() ([]frontendv1.ObjectSummary, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListObjects(context.Background())
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.ObjectSummary, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.ToObjectSummary(item))
	}
	return items, nil
}

func (s *FrontendV1Service) ListAlarms() ([]frontendv1.AlarmItem, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListAlarms(context.Background())
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.AlarmItem, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.ToAlarmItem(item))
	}
	return items, nil
}

func (s *FrontendV1Service) GetAlarmProcessingOptions(alarmID int) ([]frontendv1.AlarmProcessingOption, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.GetAlarmProcessingOptions(context.Background(), alarmID)
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.AlarmProcessingOption, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.AlarmProcessingOption{
			Code:  item.Code,
			Label: item.Label,
		})
	}
	return items, nil
}

func (s *FrontendV1Service) PickAlarm(alarmID int, request frontendv1.AlarmPickRequest) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.PickAlarm(context.Background(), alarmID, frontendv1.FromAlarmPickRequest(request))
}

func (s *FrontendV1Service) ProcessAlarm(alarmID int, request frontendv1.AlarmProcessRequest) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.ProcessAlarm(context.Background(), alarmID, frontendv1.FromAlarmProcessRequest(request))
}

func (s *FrontendV1Service) GroupProcessAlarm(alarmID int, user string) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.GroupProcessAlarm(context.Background(), alarmID, user)
}

func (s *FrontendV1Service) ListAlarmProcessingOptionsCached() ([]frontendv1.AlarmProcessingOption, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListAlarmProcessingOptionsCached(context.Background())
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.AlarmProcessingOption, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.AlarmProcessingOption{Code: item.Code, Label: item.Label})
	}
	return items, nil
}

func (s *FrontendV1Service) StandbyObject(objectID int, durationMinutes int, reason string) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.StandbyObject(context.Background(), objectID, contracts.FrontendStandbyRequest{
		DurationMinutes: durationMinutes,
		Reason:          reason,
	})
}

func (s *FrontendV1Service) ListResponseGroups() ([]frontendv1.ResponseGroup, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListResponseGroups(context.Background())
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.ResponseGroup, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.ToResponseGroup(item))
	}
	return items, nil
}

func (s *FrontendV1Service) AssignResponseGroup(alarmID int, request frontendv1.AlarmGroupActionRequest) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.AssignResponseGroup(context.Background(), alarmID, frontendv1.FromAlarmGroupActionRequest(request))
}

func (s *FrontendV1Service) NotifyGroupArrived(alarmID int) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.NotifyGroupArrived(context.Background(), alarmID)
}

func (s *FrontendV1Service) CancelResponseGroup(alarmID int) error {
	backend, err := s.backendOrErr()
	if err != nil {
		return err
	}

	return backend.CancelResponseGroup(context.Background(), alarmID)
}

func (s *FrontendV1Service) ListAlarmGroups() ([]frontendv1.AlarmGroup, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListAlarms(context.Background())
	if err != nil {
		return nil, err
	}

	return frontendv1.BuildAlarmGroups(result), nil
}

func (s *FrontendV1Service) ListEvents() ([]frontendv1.EventItem, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return nil, err
	}

	result, err := backend.ListEvents(context.Background())
	if err != nil {
		return nil, err
	}

	items := make([]frontendv1.EventItem, 0, len(result))
	for _, item := range result {
		items = append(items, frontendv1.ToEventItem(item))
	}
	return items, nil
}

func (s *FrontendV1Service) ListObjectEvents(objectID int, offset int, limit int) (frontendv1.EventPageResponse, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return frontendv1.EventPageResponse{}, err
	}

	result, err := backend.ListObjectEvents(context.Background(), objectID, offset, limit)
	if err != nil {
		return frontendv1.EventPageResponse{}, err
	}
	return frontendv1.ToEventPageResponse(result), nil
}

func (s *FrontendV1Service) GetObjectDetails(objectID int) (frontendv1.ObjectDetails, error) {
	backend, err := s.backendOrErr()
	if err != nil {
		return frontendv1.ObjectDetails{}, err
	}

	result, err := backend.GetObjectDetails(context.Background(), objectID)
	if err != nil {
		return frontendv1.ObjectDetails{}, err
	}
	return frontendv1.ToObjectDetails(result), nil
}

func (s *FrontendV1Service) backendOrErr() (contracts.FrontendBackend, error) {
	if s == nil {
		return nil, ErrFrontendBackendUnavailable
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.backend == nil {
		return nil, ErrFrontendBackendUnavailable
	}
	return s.backend, nil
}
