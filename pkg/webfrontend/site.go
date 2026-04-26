package webfrontend

import (
	"net/http"

	"github.com/gorilla/websocket"

	"obj_catalog_fyne_v3/pkg/caslcompat"
	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/frontendhttp"
)

func NewSiteHandler(backend contracts.FrontendBackend) (http.Handler, error) {
	return NewSiteHandlerWithDialer(backend, nil)
}

func NewSiteHandlerWithDialer(backend contracts.FrontendBackend, dialer contracts.PhoneDialer) (http.Handler, error) {
	return NewSiteHandlerFull(backend, dialer, nil, nil)
}

func NewSiteHandlerFull(
	backend contracts.FrontendBackend,
	dialer contracts.PhoneDialer,
	amiSettings contracts.AMISettingsProvider,
	dialBuilder func(contracts.AMISettings) contracts.PhoneDialer,
) (http.Handler, error) {
	uiHandler, err := NewHandler(frontendhttp.APIV1BasePath)
	if err != nil {
		return nil, err
	}

	apiHandler := frontendhttp.NewHandlerFull(backend, dialer, amiSettings, dialBuilder)
	caslHandler := caslcompat.NewFixtureHandler()
	mux := http.NewServeMux()
	mux.Handle(frontendhttp.APIV1BasePath, apiHandler)
	mux.Handle(frontendhttp.APIV1BasePath+"/", apiHandler)
	mux.Handle("/captchaShow", caslHandler)
	mux.Handle("/get_time_server", caslHandler)
	mux.Handle("/login", caslHandler)
	mux.Handle("/command", caslHandler)
	mux.Handle("/subscribe", caslHandler)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && r.URL.Path == "/" && websocket.IsWebSocketUpgrade(r) {
			caslHandler.ServeHTTP(w, r)
			return
		}
		uiHandler.ServeHTTP(w, r)
	}))
	return mux, nil
}
