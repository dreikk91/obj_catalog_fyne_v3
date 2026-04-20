package webfrontend

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"strings"
)

//go:embed assets/index.html assets/app.js assets/styles.css
var assetsFS embed.FS

type uiHandler struct {
	indexTemplate *template.Template
	appJS         []byte
	stylesCSS     []byte
	apiBasePath   string
}

type indexTemplateData struct {
	APIBasePath string
}

func NewHandler(apiBasePath string) (http.Handler, error) {
	indexHTML, err := assetsFS.ReadFile("assets/index.html")
	if err != nil {
		return nil, err
	}
	appJS, err := assetsFS.ReadFile("assets/app.js")
	if err != nil {
		return nil, err
	}
	stylesCSS, err := assetsFS.ReadFile("assets/styles.css")
	if err != nil {
		return nil, err
	}

	tpl, err := template.New("index.html").Parse(string(indexHTML))
	if err != nil {
		return nil, err
	}

	return &uiHandler{
		indexTemplate: tpl,
		appJS:         appJS,
		stylesCSS:     stylesCSS,
		apiBasePath:   strings.TrimSpace(apiBasePath),
	}, nil
}

func (h *uiHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	switch strings.TrimSpace(r.URL.Path) {
	case "", "/", "/index.html":
		h.serveIndex(w, r)
	case "/assets/app.js":
		serveStaticBytes(w, r, "application/javascript; charset=utf-8", h.appJS)
	case "/assets/styles.css":
		serveStaticBytes(w, r, "text/css; charset=utf-8", h.stylesCSS)
	default:
		http.NotFound(w, r)
	}
}

func (h *uiHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	var body bytes.Buffer
	if err := h.indexTemplate.ExecuteTemplate(&body, "index.html", indexTemplateData{
		APIBasePath: h.apiBasePath,
	}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(body.Bytes())
}

func serveStaticBytes(w http.ResponseWriter, r *http.Request, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}
