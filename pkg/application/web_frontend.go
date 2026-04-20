package application

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/ui/dialogs"
	"obj_catalog_fyne_v3/pkg/webfrontend"

	"github.com/rs/zerolog/log"
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

func (a *Application) startWebFrontendServer() {
	if _, err := a.ensureWebFrontendServer(); err != nil {
		log.Warn().Err(err).Msg("Не вдалося запустити web frontend server")
	}
}

func (a *Application) ensureWebFrontendServer() (string, error) {
	if a == nil {
		return "", contracts.ErrFrontendBackendUnavailable
	}

	a.webServerMu.Lock()
	defer a.webServerMu.Unlock()

	if a.webServer != nil && a.webServerURL != "" {
		return a.webServerURL, nil
	}

	handler, err := webfrontend.NewSiteHandler(applicationFrontendBackend{app: a})
	if err != nil {
		return "", err
	}

	listener, err := listenLocalWebFrontend()
	if err != nil {
		return "", err
	}

	server := &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	webURL := "http://" + listener.Addr().String()
	a.webServer = server
	a.webServerURL = webURL

	go func() {
		log.Info().Str("url", webURL).Msg("Web frontend доступний")
		if serveErr := server.Serve(listener); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Error().Err(serveErr).Str("url", webURL).Msg("Web frontend server завершився з помилкою")
		}
	}()

	return webURL, nil
}

func listenLocalWebFrontend() (net.Listener, error) {
	listener, err := net.Listen("tcp", defaultWebFrontendAddr)
	if err == nil {
		return listener, nil
	}
	return net.Listen("tcp", "127.0.0.1:0")
}

func (a *Application) stopWebFrontendServer() {
	if a == nil {
		return
	}

	a.webServerMu.Lock()
	server := a.webServer
	a.webServer = nil
	a.webServerURL = ""
	a.webServerMu.Unlock()

	if server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Warn().Err(err).Msg("Не вдалося коректно зупинити web frontend server")
	}
}

func (a *Application) openWebFrontend() {
	webURL, err := a.ensureWebFrontendServer()
	if err != nil {
		dialogs.ShowErrorDialog(a.mainWindow, "Web frontend", err)
		return
	}

	target, err := url.Parse(webURL)
	if err != nil {
		dialogs.ShowErrorDialog(a.mainWindow, "Web frontend", fmt.Errorf("invalid web frontend url: %w", err))
		return
	}

	if openErr := a.fyneApp.OpenURL(target); openErr != nil {
		dialogs.ShowInfoDialog(a.mainWindow, "Web frontend", "Відкрити браузер автоматично не вдалося.\n\nАдреса: "+webURL)
		return
	}
	if a.statusLabel != nil {
		a.statusLabel.SetText(a.backendStatusConnectedText() + " | Web frontend: " + webURL)
	}
}
