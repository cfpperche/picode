package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/clijob"
	"github.com/cfpperche/picode/internal/clipkgs"
	"github.com/cfpperche/picode/internal/pkgs"
	"github.com/cfpperche/picode/internal/store"
)

// Native CLI packages (ADR-0167). The eight guest CLIs manage their own
// plugins; PiCode declares what each one exposes (internal/clipkgs), runs the
// vendor's own command, and keeps no package database. Reads and mutations
// alike answer from the unified driver (ADR-0176: pkgs.DriverFor(cli), mapped
// back to the bytes this pane has always parsed); every mutation that installs
// or fetches is a durable job in the lane ADR-0087 built, so a PiCode restart
// never replays one and the pane can show progress from the events that lane
// already publishes.
//
// A job carries its own arguments in the job payload: `Resolve` runs after a
// restart too, and the argv of `pkg-install` depends on the plugin, the scope
// and the workspace folder. The driver builds that argv twice — once before the
// job is reserved, and again inside the lane — from the same payload shape, so
// the two cannot disagree.

func registerCLIPackageRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/cli-packages", handleCLIPackages(deps))
	mux.HandleFunc("GET /api/cli-packages/available", handleCLIPackagesAvailable(deps))
	mux.HandleFunc("GET /api/cli-packages/marketplaces", handleCLIPackageMarkets(deps))
	mux.HandleFunc("GET /api/cli-packages/updates", handleCLIPackageUpdates(deps))
	mux.HandleFunc("POST /api/cli-packages/install", handleCLIPackageJob(deps, "pkg-install"))
	mux.HandleFunc("POST /api/cli-packages/remove", handleCLIPackageJob(deps, "pkg-remove"))
	mux.HandleFunc("POST /api/cli-packages/update", handleCLIPackageJob(deps, "pkg-update"))
	mux.HandleFunc("POST /api/cli-packages/toggle", handleCLIPackageToggle(deps))
	mux.HandleFunc("POST /api/cli-packages/marketplace", handleCLIPackageMarketplace(deps))
	mux.HandleFunc("POST /api/cli-packages/inspect", handleCLIPackageInspect(deps))
}

type cliPackageRequest struct {
	CLI              string `json:"cli"`
	Workspace        string `json:"workspace"`
	Scope            string `json:"scope"`
	Name             string `json:"name"`
	Source           string `json:"source"`
	Target           string `json:"target"`
	Action           string `json:"action"`
	Ref              string `json:"ref"`
	On               *bool  `json:"on"`
	RequestKey       string `json:"requestKey"`
	ConfirmTerminals bool   `json:"confirmTerminals"`
}

// packageJobPayload is what a durable plugin job carries.
type packageJobPayload struct {
	Source    string `json:"source,omitempty"`
	Name      string `json:"name,omitempty"`
	Action    string `json:"marketplace,omitempty"`
	Ref       string `json:"ref,omitempty"`
	Scope     string `json:"scope,omitempty"`
	Workspace string `json:"workspace,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
}

func isPackageAction(action string) bool { return strings.HasPrefix(action, "pkg-") }

// packageCommand asks the driver for the command one package action runs. The
// lane asks twice — once before a job is reserved, so a request that cannot run
// reserves nothing, and again inside the lane after a restart, with the payload
// alone — so both ask here, with the same fields.
func packageCommand(ctx context.Context, cli, action string, p packageJobPayload) (pkgs.Command, error) {
	q := pkgs.Query{Vendor: p.Scope, WorkspacePath: p.Cwd}
	drv := pkgs.DriverFor(cli)
	target := pkgs.Target{Name: p.Name, Source: p.Source}
	switch action {
	case "pkg-install":
		return drv.Install(ctx, q, target)
	case "pkg-remove":
		return drv.Remove(ctx, q, target)
	case "pkg-update":
		return drv.Update(ctx, q, target)
	case "pkg-marketplace-add", "pkg-marketplace-update":
		return drv.Marketplace(ctx, q, pkgs.MarketRequest{
			Action: p.Action, Source: p.Source, Name: p.Name, Ref: p.Ref,
		})
	}
	return pkgs.Command{}, fmt.Errorf("unknown package action")
}

// resolvePackageJob builds the vendor command for one plugin job. It runs in
// the job lane, so it must not consult the request: everything it needs came
// with the payload.
func resolvePackageJob(cli, action, payload string) (clijob.Exec, error) {
	var p packageJobPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return clijob.Exec{}, fmt.Errorf("this job's arguments could not be read")
	}
	cmd, err := packageCommand(context.Background(), cli, action, p)
	if err != nil {
		return clijob.Exec{}, err
	}
	return clijob.Exec{Exe: cmd.Exe, Args: cmd.Args, Dir: cmd.Dir}, nil
}

// cliPackagePaths resolves the directories a request may touch, refusing a
// scope the CLI does not have before anything is read or run.
func cliPackagePaths(deps Deps, cli, workspaceID, scope string) (clipkgs.Paths, error) {
	if clipkgs.For(cli) == nil {
		return clipkgs.Paths{}, clipkgs.ErrNoDriver
	}
	if err := clipkgs.ValidateScope(cli, scope); err != nil {
		return clipkgs.Paths{}, err
	}
	if scope != "project" && scope != "local" {
		return clipkgs.Paths{}, nil
	}
	if strings.TrimSpace(workspaceID) == "" {
		return clipkgs.Paths{}, clipkgs.ErrNoWorkspace
	}
	dir, err := packageProjectDir(deps, workspaceID)
	if err != nil {
		return clipkgs.Paths{}, err
	}
	if dir == "" {
		return clipkgs.Paths{}, clipkgs.ErrNoWorkspace
	}
	return clipkgs.Paths{Cwd: dir}, nil
}

// statusForPackageErr maps the driver's vocabulary onto HTTP: what PiCode
// refuses is a 400, what the vendor failed to do is a 502 (the same rule the
// gallery follows), and a stale file is a 409.
func statusForPackageErr(err error) int {
	switch {
	case errors.Is(err, clipkgs.ErrStale),
		errors.Is(err, clijob.ErrTerminalsRunning),
		errors.Is(err, store.ErrCLILifecycleConflict):
		return http.StatusConflict
	case errors.Is(err, clipkgs.ErrNoDriver),
		errors.Is(err, clipkgs.ErrAgentScope),
		errors.Is(err, clipkgs.ErrScope),
		errors.Is(err, clipkgs.ErrVerbAbsent),
		errors.Is(err, clipkgs.ErrBadTarget),
		errors.Is(err, clipkgs.ErrNoWorkspace),
		errors.Is(err, pkgs.ErrNoMutation),
		errors.Is(err, pkgs.ErrNoCatalog),
		errors.Is(err, pkgs.ErrNoMarketplaces),
		errors.Is(err, store.ErrNotFound):
		return http.StatusBadRequest
	case errors.Is(err, clipkgs.ErrRosterShape):
		return http.StatusBadGateway
	}
	return http.StatusBadGateway
}

// writePackageErr answers a mutation that failed. When the CLI refused and the
// fix is a command a person has to run in a terminal (ADR-0167: Grok's
// `--trust`, Claude's marketplace-declared command), the exact command rides
// the answer — rendered by the same builder Run executed, so the pane can never
// print something PiCode would not run.
func writePackageErr(w http.ResponseWriter, err error, command string) {
	status := statusForPackageErr(err)
	if strings.TrimSpace(command) == "" {
		writeErr(w, status, err.Error())
		return
	}
	writeJSON(w, status, map[string]string{"error": err.Error(), "command": command})
}

// cliJobView is the job row plus the command a failed package job ran, so the
// pane's copy affordance works for the asynchronous half too.
type cliJobView struct {
	store.CLIJob
	Command string `json:"command,omitempty"`
}

func cliJobViewOf(j store.CLIJob) cliJobView {
	return cliJobView{CLIJob: j, Command: packageJobCommand(j)}
}

// packageJobCommand renders the command one package job ran, from the payload
// the job carries — the same driver the request asked, so the pane copies the
// line PiCode itself ran.
func packageJobCommand(j store.CLIJob) string {
	if !isPackageAction(j.Action) || strings.TrimSpace(j.Payload) == "" {
		return ""
	}
	var p packageJobPayload
	if err := json.Unmarshal([]byte(j.Payload), &p); err != nil {
		return ""
	}
	cmd, err := packageCommand(context.Background(), j.CLI, j.Action, p)
	if err != nil {
		return ""
	}
	return cmd.Line
}

// guestQuery is one guest read as the driver takes it: the scope word the
// request carried (the class cannot say `local`) and the folder a project-scope
// read runs in.
func guestQuery(scope, cwd string) pkgs.Query {
	return pkgs.Query{Vendor: scope, WorkspacePath: cwd}
}

func handleCLIPackages(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := q.Get("cli"), q.Get("scope")
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		read := guestQuery(scope, paths.Cwd)
		read.Fresh = q.Get("refresh") == "1"
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rep, err := pkgs.DriverFor(cli).List(ctx, read)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.Guest(cli, rep))
	}
}

func handleCLIPackagesAvailable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := q.Get("cli"), q.Get("scope")
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rep, err := pkgs.DriverFor(cli).Available(ctx, guestQuery(scope, paths.Cwd))
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.GuestAvailable(cli, rep))
	}
}

// handleCLIPackageUpdates compares the CLI's installed plugins with its own
// catalog and marks the rows that are behind (ADR-0167). It is a read: opening
// the pane does not fetch anything, the check runs when the pane asks for it,
// and the vendor's catalog is the only source of "latest".
func handleCLIPackageUpdates(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := q.Get("cli"), q.Get("scope")
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		read := guestQuery(scope, paths.Cwd)
		read.Fresh = q.Get("refresh") == "1"
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		rep, err := pkgs.DriverFor(cli).CheckUpdates(ctx, read)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.Guest(cli, rep))
	}
}

// handleCLIPackageMarkets lists the CLI's configured marketplace sources on
// load: the pane needs them before it can manage them, and a removal is not
// the only way to learn them (found in the pane's own build, 2026-09-20).
func handleCLIPackageMarkets(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := q.Get("cli"), q.Get("scope")
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rows, err := pkgs.DriverFor(cli).Marketplaces(ctx, guestQuery(scope, paths.Cwd))
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.GuestMarketplaces(cli, rows))
	}
}

// handleCLIPackageJob starts a durable install, removal, update or
// marketplace-source action. The answer is the job; the pane follows
// `cli.job` events for progress.
func handleCLIPackageJob(deps Deps, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v cliPackageRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		if deps.CLIJobs == nil {
			writeErr(w, http.StatusServiceUnavailable, "CLI jobs are unavailable.")
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		payload := packageJobPayload{
			Source: v.Source, Name: v.Name, Scope: v.Scope, Workspace: v.Workspace, Cwd: paths.Cwd,
		}
		// A request that cannot run reserves no job: the driver builds the
		// command here, and the lane builds it again from the payload.
		if _, err := packageCommand(r.Context(), v.CLI, action, payload); err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Invalid request.")
			return
		}
		j, err := deps.CLIJobs.Start(v.CLI, action, jobKey(v.RequestKey, v.CLI, action, string(raw)), string(raw), v.ConfirmTerminals)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, cliJobViewOf(j))
	}
}

// handleCLIPackageToggle enables or disables one plugin. It is synchronous:
// the vendor call is a local state change, and the answer is the fresh list so
// the pane never has to guess the row's next state.
func handleCLIPackageToggle(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v cliPackageRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		drv := pkgs.DriverFor(v.CLI)
		read := guestQuery(v.Scope, paths.Cwd)
		cmd, err := drv.Toggle(ctx, read, pkgs.Target{
			Name: v.Name, Source: v.Source, On: v.On == nil || *v.On,
		})
		if err != nil {
			writePackageErr(w, err, cmd.Line)
			return
		}
		// A synchronous mutation still has to reach every other open pane: the
		// feed carries the fact, the CLI's own list stays authoritative
		// (ADR-0048).
		publishPackageChange(deps, v.CLI, "pkg-toggle")
		// The pane answers the list the mutation just invalidated, so it is read
		// again rather than from the cache.
		read.Fresh = true
		rep, err := drv.List(ctx, read)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.Guest(v.CLI, rep))
	}
}

// handleCLIPackageMarketplace manages the CLI's own marketplace sources: a
// fetch (add, update) is a job, a removal is a local change.
func handleCLIPackageMarketplace(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v cliPackageRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		action := strings.TrimSpace(v.Action)
		if action != "add" && action != "remove" && action != "update" {
			writeErr(w, http.StatusBadRequest, "Unknown marketplace action.")
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		drv := pkgs.DriverFor(v.CLI)
		read := guestQuery(v.Scope, paths.Cwd)
		req := pkgs.MarketRequest{Action: action, Source: v.Source, Name: v.Name, Ref: v.Ref}
		if action == "remove" {
			cmd, err := drv.Marketplace(ctx, read, req)
			if err != nil {
				writePackageErr(w, err, cmd.Line)
				return
			}
			publishPackageChange(deps, v.CLI, "pkg-marketplace-remove")
			rows, err := drv.Marketplaces(ctx, read)
			if err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			writeJSON(w, http.StatusOK, pkgs.GuestMarketplaceSources(rows))
			return
		}
		if _, err := drv.Marketplace(ctx, read, req); err != nil {
			// A fetch reserves a job: the driver builds the command first, so a
			// request that cannot run reserves nothing.
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		if deps.CLIJobs == nil {
			writeErr(w, http.StatusServiceUnavailable, "CLI jobs are unavailable.")
			return
		}
		payload, err := json.Marshal(packageJobPayload{
			Source: v.Source, Name: v.Name, Action: action, Ref: v.Ref, Scope: v.Scope, Workspace: v.Workspace, Cwd: paths.Cwd,
		})
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Invalid request.")
			return
		}
		jobAction := "pkg-marketplace-" + action
		j, err := deps.CLIJobs.Start(v.CLI, jobAction, jobKey(v.RequestKey, v.CLI, jobAction, string(payload)), string(payload), false)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, cliJobViewOf(j))
	}
}

// handleCLIPackageInspect returns the vendor's own inspection output, verbatim.
func handleCLIPackageInspect(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v cliPackageRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		name := firstNonEmpty(v.Name, v.Target, v.Source)
		if name == "" {
			writeErr(w, statusForPackageErr(clipkgs.ErrBadTarget), clipkgs.ErrBadTarget.Error())
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		cmd, out, err := pkgs.DriverFor(v.CLI).Inspect(ctx, guestQuery(v.Scope, paths.Cwd), pkgs.Target{Name: name})
		if err != nil {
			writePackageErr(w, err, cmd.Line)
			return
		}
		writeJSON(w, http.StatusOK, pkgs.GuestInspect(v.CLI, out))
	}
}

// publishPackageChange invalidates the roster cache and tells every open pane
// that this CLI's plugins moved. The vendor's store stays authoritative: the
// event carries the fact, never the list (the ADR-0163 rule).
func publishPackageChange(deps Deps, cli, action string) {
	if !isPackageAction(action) {
		return
	}
	clipkgs.Invalidate(cli)
	if deps.Feed != nil {
		deps.Feed.Ephemeral("cli.packages", map[string]any{"cli": cli, "action": action})
	}
}

// jobKey makes a repeat of the same request idempotent: the client may send
// its own key, and a missing one is derived from what the job does. The store
// refuses an identical key with different input.
func jobKey(provided, cli, action, payload string) string {
	if k := strings.TrimSpace(provided); k != "" {
		return k
	}
	sum := sha256.Sum256([]byte(cli + "\x00" + action + "\x00" + payload))
	return cli + "-" + action + "-" + hex.EncodeToString(sum[:6])
}
