package webui

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/lich0821/ccNexus/internal/config"
	"github.com/lich0821/ccNexus/internal/proxy"
	"github.com/lich0821/ccNexus/internal/storage"
	"github.com/lich0821/ccNexus/cmd/server/webui/api"
)

//go:embed ui
var uiFS embed.FS

// WebUI represents the web management interface
type WebUI struct {
	apiHandler *api.Handler
}

// New creates a new WebUI instance
func New(cfg *config.Config, p *proxy.Proxy, storage *storage.SQLiteStorage) *WebUI {
	return &WebUI{
		apiHandler: api.NewHandler(cfg, p, storage),
	}
}

// RegisterRoutes registers all web UI routes to the provided mux
func (w *WebUI) RegisterRoutes(mux *http.ServeMux) error {
	// Register API routes
	w.apiHandler.RegisterRoutes(mux)

	// Serve embedded UI files
	uiSubFS, err := fs.Sub(uiFS, "ui")
	if err != nil {
		return err
	}

	uiHandler := http.FileServer(http.FS(uiSubFS))
	mux.Handle("/ui/", http.StripPrefix("/ui/", uiHandler))

	// Redirect /admin to /ui/
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusFound)
	})

	return nil
}
