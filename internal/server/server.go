// Package server exposes the picode HTTP API and the embedded UI.
//
// Routes (M1):
//
//	GET  /api/health, /api/version          — liveness/identity
//	GET  /api/deploy/readiness              — who is working (loopback, no session)
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
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/apps"
	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/backup"
	"github.com/cfpperche/picode/internal/browser"
	"github.com/cfpperche/picode/internal/catalog"
	"github.com/cfpperche/picode/internal/clijob"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/docker"
	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/llamajob"
	"github.com/cfpperche/picode/internal/llamaservice"
	"github.com/cfpperche/picode/internal/mcpcatalog"
	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/presence"
	"github.com/cfpperche/picode/internal/preview"
	"github.com/cfpperche/picode/internal/push"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/term"
	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/usage"
	"github.com/cfpperche/picode/internal/version"
	"github.com/cfpperche/picode/internal/web"
	"github.com/cfpperche/picode/internal/webhooks"
)

// Deps carries the server's collaborators (injected for testability).
type Deps struct {
	LlamaJobs    *llamajob.Service
	LlamaService *llamaservice.Service
	Store        *store.Store
	Tmux         *tmux.Manager
	Runtime      *rpc.Runtime
	AgentCmd     string // the pi command for managed Pi agents; no other CLI depends on it (ADR-0179)

	// Usage is the vendor-call client (quota listings, account identity).
	// Nil means usage.Default; tests point it at a local server so no test
	// reaches a vendor.
	Usage *usage.Client

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
	Apps         *apps.Registry    // apps host (ADR-0036); nil-safe = no apps
	Docker       *docker.Service   // shared operations for the Docker App and Pi tools
	Webhooks     *webhooks.Engine  // generic outbound event delivery (ADR-0075)
	Push         *push.Notifier    // Web Push (ADR-0047); nil-safe = 503 on /api/push/*
	Feed         *feed.Feed        // change feed (ADR-0048); nil-safe = 503 on /api/events
	Browser      *browser.Hub      // work-browser command channel (ADR-0132); lazily built in New
	Replies      *TuiReplies       // Inbox replies into the running TUI (ADR-0060); lazy-init in New
	Connectors   *mcpcatalog.Store // curated connector catalog (ADR-0157); lazy-init in New from DataDir
	Previews     *preview.Store    // HTML preview tickets (ADR-0136); lazy-init in New
	DevServers   *DevServerCache   // dev-server discovery (/api/devservers); lazy-init in New
	TermStates   *TermStates       // coding-CLI terminal state (ADR-0056 tier 1); lazy-init in New
	TermRuntimes *TermRuntimes     // authoritative CLI presence (ADR-0062); lazy-init in New
	// Shared short-lived GET /api/terminals snapshot (singleflight + TTL,
	// terminals_cache.go): the CLIs page refetches on every terminal.* feed
	// event and one round costs a dozen subprocesses per terminal. Nil-safe
	// = every request computes directly (tests, minimal embeddings).
	TermCache *TerminalsCache
	// Session forensics (ADR-0085): session names that were alive at the
	// previous graceful shutdown and did not survive to this boot. Set once
	// by the daemon before New; nil-safe everywhere.
	LostSessions map[string]bool
	CLIs         *CLITerminals   // terminal launch settings and operation locks (ADR-0069)
	CLIJobs      *clijob.Service // durable CLI lifecycle jobs (ADR-0087); nil-safe = 503 on the routes
	Auth         *auth.Service   // request gate (ADR-0049); nil = ungated (tests, dev)
}

// usageClient is the vendor-call client this server uses: the one it was built
// with, or the package default. Tests hand it a client aimed at a local
// server, so no test (and no offline instance) reaches a vendor.
func (d Deps) usageClient() *usage.Client {
	if d.Usage != nil {
		return d.Usage
	}
	return usage.Default
}

// New builds the picode *http.Server. Addr handling stays with the caller
// (cmd/picode) so tests can bind :0.
func New(addr string, deps Deps) *http.Server {
	mux := http.NewServeMux()
	if deps.CLIs == nil {
		deps.CLIs = newCLITerminals()
	}
	// User-described package configs (ADR-0119): descriptors persist as one
	// JSON file each and must resolve before the first listing.
	pipkg.LoadUserDescriptors(filepath.Join(deps.DataDir, "package-configs"))
	if deps.Store != nil {
		// The intercept map doubles as first-insert defaults, but only
		// where a mechanism exists: seeding Activity on for a hookless CLI
		// (Muse Code) would break its default launches at prepare time.
		// The copy is deliberate — a forced false here must not persist as
		// an owner choice, or a future mechanism would stay switched off.
		enabled := loadInterceptEnabled(deps.DataDir)
		for _, cli := range clilaunch.Catalog() {
			if !hasIntegrationMechanism(cli.ID) {
				enabled[cli.ID] = false
			}
		}
		_ = deps.Store.ImportCLIConfigs(enabled)
		_ = deps.Store.SeedCatalogIntegrationDefaults(hasIntegrationMechanism)
		sweepPinDirs(deps.DataDir, deps.Store)
	}

	// Coding-CLI state and presence (ADRs 0056/0062): tests and minimal
	// embeddings construct Deps without registries — both endpoints still
	// have to work.
	if deps.TermStates == nil {
		deps.TermStates = NewTermStates()
	}
	if deps.TermRuntimes == nil {
		deps.TermRuntimes = NewTermRuntimes()
	}
	if deps.TermCache == nil {
		deps.TermCache = &TerminalsCache{}
	}
	// Refresh the local reporter so a deploy picks up curl flags without
	// requiring the user to toggle intercept off/on.
	if deps.DataDir != "" {
		_, _ = ensureHookScript(deps.DataDir)
		_, _ = ensurePiReplyExtension(deps.DataDir) // ADR-0060 receiver: fresh on every boot
		for _, cli := range clilaunch.Catalog() {
			if !cli.Integrable() {
				continue
			}
			if c, err := cliConfig(deps, cli.ID); err == nil {
				_ = syncCLIIntegration(deps, cli.ID, c.Integration)
			}
		}
	}
	if deps.Replies == nil {
		deps.Replies = NewTuiReplies()
	}
	if deps.Previews == nil {
		deps.Previews = preview.NewStore(preview.DefaultTTL)
	}
	if deps.DevServers == nil {
		deps.DevServers = newDevServerCache()
	}
	if deps.Browser == nil {
		deps.Browser = browser.New()
	}
	if deps.Connectors == nil {
		// Curated connector catalog (ADR-0157): the constructor never
		// fetches — the first gallery search serves the seed (plus any
		// cache file) and refreshes in the background. No DataDir means
		// the cache stays in memory.
		deps.Connectors = mcpcatalog.NewStore(deps.DataDir)
	}

	if deps.Store != nil && deps.DataDir != "" && deps.LlamaService == nil {
		deps.LlamaService, _ = llamaservice.New(deps.Store, deps.DataDir, func() ([]string, error) {
			agents, err := deps.Store.ListAllAgents()
			if err != nil {
				return nil, err
			}
			out := []string{}
			for _, a := range agents {
				if a.Provider == nil || *a.Provider == "llama.cpp" {
					out = append(out, a.Name)
				}
			}
			return out, nil
		})
	}
	if deps.Store != nil && deps.LlamaJobs == nil {
		var observer func(store.LlamaJob)
		if deps.LlamaService != nil {
			observer = deps.LlamaService.ObserveDownload
		}
		deps.LlamaJobs, _ = llamajob.New(deps.Store, func() (string, string) { return llamaURL(), catalog.LlamaKey() }, observer)
	}
	if deps.Store != nil && deps.CLIJobs == nil {
		deps.CLIJobs, _ = clijob.New(clijob.Deps{
			Store: deps.Store,
			Resolve: func(cli, action, payload string) (clijob.Exec, error) {
				return resolveLifecycleByID(deps, cli, action, payload)
			},
			LiveTerminals: func(cli string) int { return liveTerminalsFor(deps, cli) },
			AfterSuccess: func(cli, action string) {
				refreshCLICheckAfterJob(deps, cli)
				publishPackageChange(deps, cli, action)
			},
		})
	}
	registerAll(mux, deps)

	var handler http.Handler = mux
	if deps.Auth != nil {
		handler = deps.Auth.Wrap(mux) // the one gate in front of every route (ADR-0049)
	}
	if deps.Previews != nil {
		// A ticket's own origin (ADR-0137) is served before the gate and
		// outside it: that origin is a preview, not the app, and the ticket
		// is its only credential. Any other Host falls through unchanged.
		handler = previewHostHandler(deps, handler)
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	if deps.LlamaJobs != nil {
		srv.RegisterOnShutdown(deps.LlamaJobs.Close)
	}
	if deps.CLIJobs != nil {
		srv.RegisterOnShutdown(deps.CLIJobs.Close)
	}
	if deps.LlamaService != nil {
		srv.RegisterOnShutdown(deps.LlamaService.Close)
	}
	return srv
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
	mux.HandleFunc("GET /api/catalog", handleCatalog(deps))
	mux.HandleFunc("GET /api/share", handleShare(deps))
	registerMCPRoutes(mux, deps)
	registerCLINativeRoutes(mux, deps)
	registerPeerCommunication(mux, deps)
	registerPeerOnboarding(mux, deps)
	registerPackageRoutes(mux, deps)
	registerCLIPackageRoutes(mux, deps)
	registerDockerRoutes(mux, deps)
	registerDeviceRoutes(mux, &deps)

	registerWorkspaceRoutes(mux, deps)
	registerWorkspaceCloneRoutes(mux, deps)
	registerGithubReposRoutes(mux, deps)
	registerServerRoutes(mux, deps)
	registerPiSettingsRoutes(mux, deps)
	registerPiKeysRoutes(mux)
	registerCLIKeysRoutes(mux)
	registerSessionOps(mux, deps)
	registerSlashOps(mux, deps)
	registerSlashRes(mux, deps)
	registerRolesState(mux, deps)
	registerChecklistRoutes(mux, deps)
	registerDeliveryRoutes(mux, deps)
	registerDeliveryObservationRoutes(mux, deps)
	registerAgentFileRoutes(mux, deps)
	registerPreviewRoutes(mux, deps)
	registerDevServerRoutes(mux, deps)
	registerTerminalRoutes(mux, deps)
	registerCredentialRoutes(mux, deps)
	registerTerminalWiringRoutes(mux, deps)
	registerCLIRoutes(mux, deps)
	registerTerminalSettingsRoutes(mux, deps)
	mux.HandleFunc("GET /api/tui-working", handleTuiWorking(deps))
	registerGitGraphRoutes(mux, deps)
	registerGitStatusRoutes(mux, deps)
	registerWorkDiffRoutes(mux, deps)
	registerPRRoutes(mux, deps)
	registerGitRunRoutes(mux, deps)
	registerGitComposeRoutes(mux, deps)
	registerTerminalAskRoutes(mux, deps)
	registerDeployRoutes(mux, deps)
	registerAgentAskRoutes(mux, deps)
	registerWorkspaceFileRoutes(mux, deps)
	registerAgentBash(mux, deps)
	registerLlama(mux, deps)
	registerSnippet(mux, deps)
	registerSnips(mux, deps)
	registerPins(mux, deps)
	registerPinFiles(mux, deps)
	registerPinReminders(mux, deps)
	registerCanvasRoutes(mux, deps)
	registerFolderRoutes(mux)
	registerOAuthRoutes(mux)
	registerBackupRoutes(mux, deps)
	registerAppsRoutes(mux, deps)
	registerWebappRoutes(mux, deps)
	registerInboxRoutes(mux, deps)
	registerAutomationRoutes(mux, deps)
	mux.HandleFunc("GET /api/events", handleEvents(deps))
	registerBrowserRoutes(mux, deps)
	registerComputerRoutes(mux, deps)
	registerExtensionRoutes(mux, deps)
	registerPushRoutes(mux, deps)
	registerWebhookRoutes(mux, deps)
	registerAuthRoutes(mux, deps)

	bridge := term.Bridge(deps.Tmux, termOptionResolver(deps), terminalInterruptObserver(deps))
	mux.Handle("/ws/term", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Preserve existing agent tabs and bookmarks after lazy migration.
		q := r.URL.Query()
		if name := q.Get("session"); strings.HasPrefix(name, "picode-") && !strings.HasPrefix(name, "picode-sh-") {
			q.Set("session", deps.agentSession(strings.TrimPrefix(name, "picode-")))
			r = r.Clone(r.Context())
			r.URL.RawQuery = q.Encode()
		}
		bridge.ServeHTTP(w, r)
	}))
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
		// Release notes auto-open only for binaries stamped by the release
		// workflow; source builds keep the same version but stay quiet.
		"release": version.Stamped != "",
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
		if strings.HasPrefix(r.URL.Path, "/assets/") || strings.HasPrefix(r.URL.Path, "/browser/assets/") || strings.HasPrefix(r.URL.Path, "/desktop/assets/") || strings.HasPrefix(r.URL.Path, "/mobile/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}
