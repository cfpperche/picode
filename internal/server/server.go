// Package server exposes the picode HTTP API and the embedded UI.
//
// Routes grow with the milestones: M1 adds /ws/term (tmux bridge),
// M2 adds /ws/agent (RPC bridge). See docs/architecture.md.
package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/version"
	"github.com/cfpperche/picode/internal/web"
)

// New builds the picode *http.Server. Addr handling stays with the caller
// (cmd/picode) so tests can bind :0.
func New(addr string) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", handleHealth)
	mux.HandleFunc("/api/version", handleVersion)

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
