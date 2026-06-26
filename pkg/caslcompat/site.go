package caslcompat

import (
	"bytes"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type StaticSiteOptions struct {
	CASLRootDir string
	PublicDir   string
	ConfDir     string
	TechDir     string
}

func NewFixtureSiteHandler(caslRootDir, wsURL string) http.Handler {
	return NewStaticSiteHandler(NewFixtureHandlerWithWSURL(wsURL), StaticSiteOptions{
		CASLRootDir: caslRootDir,
	})
}

func NewStaticSiteHandler(apiHandler *Handler, options StaticSiteOptions) http.Handler {
	if apiHandler == nil {
		apiHandler = NewFixtureHandler()
	}

	publicDir := staticDir(options.PublicDir, options.CASLRootDir, "public")
	confDir := staticDir(options.ConfDir, options.CASLRootDir, "configurator_4L")
	techDir := staticDir(options.TechDir, options.CASLRootDir, "casl-technic")

	mux := http.NewServeMux()
	for _, path := range []string{
		"/captchaShow",
		"/get_time_server",
		"/login",
		"/login_technician",
		"/subscribe",
		"/subscribe_techn",
		"/command",
		"/ppk_command",
		"/ecom_command",
	} {
		mux.Handle(path, apiHandler)
	}
	mux.Handle("/api/", apiHandler)

	if dirExists(confDir) {
		mux.Handle("/conf/", http.StripPrefix("/conf/", http.FileServer(http.Dir(confDir))))
		mux.Handle("/conf", http.RedirectHandler("/conf/", http.StatusMovedPermanently))
	}
	if dirExists(techDir) {
		mux.Handle("/tech/", http.StripPrefix("/tech/", http.FileServer(http.Dir(techDir))))
		mux.Handle("/tech", http.RedirectHandler("/tech/", http.StatusMovedPermanently))
	}

	static := spaFileServer(publicDir)
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL != nil && r.URL.Path == "/" && websocket.IsWebSocketUpgrade(r) {
			apiHandler.ServeHTTP(w, r)
			return
		}
		if static == nil {
			apiHandler.ServeHTTP(w, r)
			return
		}
		static.ServeHTTP(w, r)
	}))

	return mux
}

func spaFileServer(root string) http.Handler {
	if !dirExists(root) {
		return nil
	}

	fileServer := http.FileServer(http.Dir(root))
	indexPath := filepath.Join(root, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			writeCASLMethodNotAllowed(w, http.MethodGet, http.MethodHead)
			return
		}

		cleanPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		if cleanPath == "." {
			cleanPath = "index.html"
		}
		target := filepath.Join(root, cleanPath)
		if fileExists(target) {
			w.Header().Set("Cache-Control", "no-store")
			if servePatchedCASLJS(w, r, target) {
				return
			}
			fileServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, indexPath)
	})
}

func servePatchedCASLJS(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.HasSuffix(path, ".js") {
		return false
	}

	body, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	patched := patchCASLStaticJS(body)
	if bytes.Equal(body, patched) {
		return false
	}

	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	http.ServeContent(w, r, filepath.Base(path), fileModTime(path), bytes.NewReader(patched))
	return true
}

func patchCASLStaticJS(body []byte) []byte {
	patched := bytes.ReplaceAll(body, []byte("t?M[Ae]:[]"), []byte("t&&M?M[Ae]:[]"))
	return patched
}

func fileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func staticDir(explicit, root, child string) string {
	if strings.TrimSpace(explicit) != "" {
		return explicit
	}
	if strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(root, child)
}
