package webfrontend

import (
	"net/http"

	"obj_catalog_fyne_v3/pkg/contracts"
	"obj_catalog_fyne_v3/pkg/frontendhttp"
)

func NewSiteHandler(backend contracts.FrontendBackend) (http.Handler, error) {
	uiHandler, err := NewHandler(frontendhttp.APIV1BasePath)
	if err != nil {
		return nil, err
	}

	apiHandler := frontendhttp.NewHandler(backend)
	mux := http.NewServeMux()
	mux.Handle(frontendhttp.APIV1BasePath, apiHandler)
	mux.Handle(frontendhttp.APIV1BasePath+"/", apiHandler)
	mux.Handle("/", uiHandler)
	return mux, nil
}
