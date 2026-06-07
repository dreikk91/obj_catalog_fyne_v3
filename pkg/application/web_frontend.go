package application

import (
	"context"

	"obj_catalog_fyne_v3/pkg/contracts"
)

const defaultWebFrontendAddr = "127.0.0.1:17890"

type applicationFrontendBackend struct {
	app *Application
}

func (b applicationFrontendBackend) current() (contracts.FrontendBackend, error) {
	if b.app == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	backend := b.app.getFrontendAPI()
	if backend == nil {
		return nil, contracts.ErrFrontendBackendUnavailable
	}
	return backend, nil
}

func (b applicationFrontendBackend) Capabilities(ctx context.Context) (contracts.FrontendCapabilities, error) {
	backend, err := b.current()
	if err != nil {
		return contracts.FrontendCapabilities{}, err
	}
	return backend.Capabilities(ctx)
}

func (b applicationFrontendBackend) ListObjects(ctx context.Context) ([]contracts.FrontendObjectSummary, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.ListObjects(ctx)
}

func (b applicationFrontendBackend) ListAlarms(ctx context.Context) ([]contracts.FrontendAlarmItem, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.ListAlarms(ctx)
}

func (b applicationFrontendBackend) GetAlarmProcessingOptions(ctx context.Context, alarmID int) ([]contracts.FrontendAlarmProcessingOption, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.GetAlarmProcessingOptions(ctx, alarmID)
}

func (b applicationFrontendBackend) PickAlarm(ctx context.Context, alarmID int, request contracts.FrontendAlarmPickRequest) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.PickAlarm(ctx, alarmID, request)
}

func (b applicationFrontendBackend) ProcessAlarm(ctx context.Context, alarmID int, request contracts.FrontendAlarmProcessRequest) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.ProcessAlarm(ctx, alarmID, request)
}

func (b applicationFrontendBackend) ListEvents(ctx context.Context) ([]contracts.FrontendEventItem, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.ListEvents(ctx)
}

func (b applicationFrontendBackend) ListObjectEvents(ctx context.Context, objectID int, offset int, limit int) (contracts.FrontendEventPage, error) {
	backend, err := b.current()
	if err != nil {
		return contracts.FrontendEventPage{}, err
	}
	return backend.ListObjectEvents(ctx, objectID, offset, limit)
}

func (b applicationFrontendBackend) GetObjectDetails(ctx context.Context, objectID int) (contracts.FrontendObjectDetails, error) {
	backend, err := b.current()
	if err != nil {
		return contracts.FrontendObjectDetails{}, err
	}
	return backend.GetObjectDetails(ctx, objectID)
}

func (b applicationFrontendBackend) CreateObject(ctx context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	backend, err := b.current()
	if err != nil {
		return contracts.FrontendObjectMutationResult{}, err
	}
	return backend.CreateObject(ctx, request)
}

func (b applicationFrontendBackend) UpdateObject(ctx context.Context, request contracts.FrontendObjectUpsertRequest) (contracts.FrontendObjectMutationResult, error) {
	backend, err := b.current()
	if err != nil {
		return contracts.FrontendObjectMutationResult{}, err
	}
	return backend.UpdateObject(ctx, request)
}

func (b applicationFrontendBackend) GroupProcessAlarm(ctx context.Context, alarmID int, user string) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.GroupProcessAlarm(ctx, alarmID, user)
}

func (b applicationFrontendBackend) ListAlarmProcessingOptionsCached(ctx context.Context) ([]contracts.FrontendAlarmProcessingOption, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.ListAlarmProcessingOptionsCached(ctx)
}

func (b applicationFrontendBackend) ListResponseGroups(ctx context.Context) ([]contracts.FrontendResponseGroup, error) {
	backend, err := b.current()
	if err != nil {
		return nil, err
	}
	return backend.ListResponseGroups(ctx)
}

func (b applicationFrontendBackend) AssignResponseGroup(ctx context.Context, alarmID int, request contracts.FrontendAlarmGroupActionRequest) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.AssignResponseGroup(ctx, alarmID, request)
}

func (b applicationFrontendBackend) NotifyGroupArrived(ctx context.Context, alarmID int) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.NotifyGroupArrived(ctx, alarmID)
}

func (b applicationFrontendBackend) CancelResponseGroup(ctx context.Context, alarmID int) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.CancelResponseGroup(ctx, alarmID)
}

func (b applicationFrontendBackend) StandbyObject(ctx context.Context, objectID int, request contracts.FrontendStandbyRequest) error {
	backend, err := b.current()
	if err != nil {
		return err
	}
	return backend.StandbyObject(ctx, objectID, request)
}

func (a *Application) startWebFrontendServer() {
	// Web frontend has been disabled as requested
}

func (a *Application) ensureWebFrontendServer() (string, error) {
	return "", contracts.ErrFrontendBackendUnavailable
}

func (a *Application) stopWebFrontendServer() {
	// Web frontend has been disabled as requested
}

func (a *Application) openWebFrontend() {
	// Web frontend has been disabled as requested
}
