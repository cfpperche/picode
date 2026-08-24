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
	"time"

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
	AgentCmd string // command spawned per workspace ("pi" — ADR-0003)
}

// New builds the picode *http.Server. Addr handling stays with the caller
// (cmd/picode) so tests can bind :0.
func New(addr string, deps Deps) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/version", handleVersion)
	mux.HandleFunc("/api/system", handleSystem(deps))

	registerWorkspaceRoutes(mux, deps)

	mux.Handle("/ws/term", term.Bridge(deps.Tmux))

	public, err := fs.Sub(web.Public, "public")
	if err != nil {
		// Cannot happen: public/ is embedded at build time.
		panic("picode: embedded UI missing: " + err.Error())
	}
	mux.Handle("/", http.FileServer(http.FS(public)))

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
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
