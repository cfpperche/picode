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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/backup"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/push"
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
	Backup       *backup.Engine
	Apps         *apps.Registry // apps host (ADR-0036); nil-safe = no apps
	Push         *push.Notifier // Web Push (ADR-0047); nil-safe = 503 on /api/push/*
	Feed         *feed.Feed     // change feed (ADR-0048); nil-safe = 503 on /api/events
	TermStates   *TermStates    // guest terminal state (ADR-0056 tier 1); lazy-init in New
	Auth         *auth.Service  // request gate (ADR-0049); nil = ungated (tests, dev)
}

// New builds the picode *http.Server. Addr handling stays with the caller
// (cmd/picode) so tests can bind :0.
func New(addr string, deps Deps) *http.Server {
	mux := http.NewServeMux()

	// Guest terminal state (ADR-0056 tier 1): tests and minimal embeddings
	// construct Deps without a registry — the endpoint still has to work.
	if deps.TermStates == nil {
		deps.TermStates = NewTermStates()
	}
	// Refresh the local reporter so a deploy picks up curl flags without
	// requiring the user to toggle intercept off/on.
	if deps.DataDir != "" {
		_, _ = ensureHookScript(deps.DataDir)
		for id, on := range loadInterceptEnabled(deps.DataDir) {
			if on {
				_ = installIntercept(deps.DataDir, id)
			}
		}
	}

	registerAll(mux, deps)

	var handler http.Handler = mux
	if deps.Auth != nil {
		handler = deps.Auth.Wrap(mux) // the one gate in front of every route (ADR-0049)
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

// registerAll wires every route the server serves onto any route
// registrar. New passes a real *http.ServeMux; the OpenAPI generator
// (cmd/picode-openapi) passes a recorder so the spec can never drift
// from what the binary actually serves — there is exactly one
// registration list.
func registerAll(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/health", handleHealth)
	mux.HandleFunc("GET /api/version", handleVersion)
	mux.HandleFunc("GET /api/system", handleSystem(deps))
	mux.HandleFunc("POST /api/system/pi-update", handlePiSelfUpdate(deps))
	mux.HandleFunc("GET /api/catalog", handleCatalog(deps))
	mux.HandleFunc("GET /api/share", handleShare(deps))
	registerMCPRoutes(mux, deps)
	registerPackageRoutes(mux, deps)
	registerDeviceRoutes(mux, &deps)

	registerWorkspaceRoutes(mux, deps)
	registerWorkspaceCloneRoutes(mux, deps)
	registerServerRoutes(mux, deps)
	registerPiSettingsRoutes(mux, deps)
	registerPiKeysRoutes(mux)
	registerSessionOps(mux, deps)
	registerSlashOps(mux, deps)
	registerSlashRes(mux, deps)
	registerRolesState(mux, deps)
	registerChecklistRoutes(mux, deps)
	registerAgentFileRoutes(mux, deps)
	registerTerminalRoutes(mux, deps)
	registerTerminalWiringRoutes(mux, deps)
	registerTerminalSettingsRoutes(mux, deps)
	mux.HandleFunc("GET /api/tui-working", handleTuiWorking(deps))
	registerGitGraphRoutes(mux, deps)
	registerGitStatusRoutes(mux, deps)
	registerWorkDiffRoutes(mux, deps)
	registerWorkspaceFileRoutes(mux, deps)
	registerAgentBash(mux, deps)
	registerLlama(mux)
	registerSnippet(mux, deps)
	registerPins(mux, deps)
	registerPinFiles(mux, deps)
	registerFolderRoutes(mux)
	registerOAuthRoutes(mux)
	registerBackupRoutes(mux, deps)
	registerAppsRoutes(mux, deps)
	registerInboxRoutes(mux, deps)
	registerAutomationRoutes(mux, deps)
	mux.HandleFunc("GET /api/events", handleEvents(deps))
	registerExtensionRoutes(mux, deps)
	registerPushRoutes(mux, deps)
	registerAuthRoutes(mux, deps)

	mux.Handle("/ws/term", term.Bridge(deps.Tmux, termOptionResolver(deps), terminalInterruptObserver(deps)))
	mux.Handle("/ws/agent", agentWS(deps))

	mux.Handle("/", securityHeaders(cacheControl(uiHandler())))
}

// bootID identifies this server process. The UI compares it on
// /api/health: a change means the binary restarted (even a fast restart
// the poll never saw as downtime) and the page must reload.
var bootID = func() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}()

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"bootId": bootID,
		"time":   time.Now().UTC().Format(time.RFC3339),
	})
}

func handleVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"name": "picode",
		// The running build's identity ("0.1.0+0550fa2" on source builds);
		// semver-only consumers use "semver".
		"version": version.Build(),
		"semver":  version.Version,
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
// uiHandler serves the frontend. In a disk build (ADR-0023) the UI may simply
// not have been built yet, and a wall of 404s does not say that — so the check
// is per request, which also means the page starts working the moment
// `make web` finishes, without a restart.
func uiHandler() http.Handler {
	files := http.FileServer(http.FS(web.UI()))
	if web.Embedded() {
		return files // sealed in at compile time; it is always there
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if web.Built() {
			files.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "PiCode: the UI has not been built yet.\n\nRun `make web` (or `make build`), then reload.\nLooked in: %s\n", web.Dir())
	})
}

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
