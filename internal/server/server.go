// Package server exposes the picode HTTP API and the embedded UI.
//
// Routes (M1):
//
//	GET  /api/health, /api/version          — liveness/identity
//	GET  /api/system                        — pi/tmux detection + warnings
//	GET/POST /api/workspaces                — registry CRUD
//	DELETE /api/workspaces/{id}             — remove (+ stop agent)
//	POST /api/workspaces/{id}/open|close    — start/stop the pi agent (tmux)
//	GET  /ws/term?session=<name>            — terminal bridge (xterm.js)
//
// See docs/architecture.md.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/term"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/version"
	"github.com/cfpperche/picode/internal/web"
)

// Deps carries the server's collaborators (injected for testability).
type Deps struct {
	Store    *store.Store
	Tmux     *tmux.Manager
	Runtime  *rpc.Runtime
	AgentCmd string // command spawned per workspace ("pi" — ADR-0003)

	// Port management (ADR-0007). BindHost is the configured host;
	// Rebind signals the main loop to re-read the port setting;
	// PortSnapshot reports live port state. Optional (nil-safe).
	BindHost     string
	Rebind       func()
	PortSnapshot func() PortSnapshot
	DataDir      string
	Insecure     bool
	Presence     *presence.Registry
}

// New builds the picode *http.Server. Addr handling stays with the caller
// (cmd/picode) so tests can bind :0.
func New(addr string, deps Deps) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/version", handleVersion)
	mux.HandleFunc("GET /api/system", handleSystem(deps))
	mux.HandleFunc("GET /api/catalog", handleCatalog(deps))
	mux.HandleFunc("GET /api/mcp", handleMCP)
	mux.HandleFunc("GET /api/share", handleShare(deps))
	registerPackageRoutes(mux, deps)
	registerDeviceRoutes(mux, &deps)

	registerWorkspaceRoutes(mux, deps)
	registerServerRoutes(mux, deps)
	registerPiSettingsRoutes(mux, deps)
	registerSessionOps(mux, deps)
	registerSlashOps(mux, deps)
	registerSlashRes(mux, deps)
	registerLlama(mux)
	registerSnippet(mux, deps)
	registerFolderRoutes(mux)
	registerOAuthRoutes(mux)

	mux.Handle("/ws/term", term.Bridge(deps.Tmux))
	mux.Handle("/ws/agent", agentWS(deps))

	public, err := fs.Sub(web.Public, "public")
	if err != nil {
		// Cannot happen: public/ is embedded at build time.
		panic("picode: embedded UI missing: " + err.Error())
	}
	mux.Handle("/", cacheControl(http.FileServer(http.FS(public))))

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name":    "picode",
		"version": version.Version,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Client gone mid-write; nothing useful to do.
		_ = err
	}
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// cacheControl keeps the UI from being stale after binary upgrades: the
// app shell (index.html) must revalidate every load; hashed Vite assets
// under /assets/ are content-addressed and can be cached forever.
func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
