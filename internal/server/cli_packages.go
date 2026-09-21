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
	"github.com/cfpperche/picode/internal/store"
)

// Native CLI packages (ADR-0167). The eight guest CLIs manage their own
// plugins; PiCode declares what each one exposes (internal/clipkgs), runs the
// vendor's own command, and keeps no package database. Reads are synchronous;
// every mutation that installs or fetches is a durable job in the lane
// ADR-0087 built, so a PiCode restart never replays one and the pane can show
// progress from the events that lane already publishes.
//
// A job carries its own arguments in the job payload: `Resolve` runs after a
// restart too, and the argv of `pkg-install` depends on the plugin, the scope
// and the workspace folder.

func registerCLIPackageRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/cli-packages", handleCLIPackages(deps))
	mux.HandleFunc("GET /api/cli-packages/available", handleCLIPackagesAvailable(deps))
	mux.HandleFunc("GET /api/cli-packages/marketplaces", handleCLIPackageMarkets(deps))
	mux.HandleFunc("POST /api/cli-packages/install", handleCLIPackageJob(deps, "pkg-install"))
	mux.HandleFunc("POST /api/cli-packages/remove", handleCLIPackageJob(deps, "pkg-remove"))
	mux.HandleFunc("POST /api/cli-packages/update", handleCLIPackageJob(deps, "pkg-update"))
	mux.HandleFunc("POST /api/cli-packages/toggle", handleCLIPackageToggle(deps))
	mux.HandleFunc("POST /api/cli-packages/marketplace", handleCLIPackageMarketplace(deps))
	mux.HandleFunc("POST /api/cli-packages/inspect", handleCLIPackageInspect(deps))
}

// cliPackagesView is one pane load: what the CLI declares, and what it holds.
// The rows are the CLI's own answer, never a PiCode roster.
type cliPackagesView struct {
	CLI    string            `json:"cli"`
	Scopes []clipkgs.Scope   `json:"scopes"`
	Caps   clipkgs.Caps      `json:"caps"`
	Notes  map[string]string `json:"notes"`
	Rows   []clipkgs.Row     `json:"rows"`
	Note   string            `json:"note,omitempty"`
	ReadAt string            `json:"readAt,omitempty"`
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

// packageVerb maps a job action onto the clipkgs verb it runs.
func packageVerb(action string) (clipkgs.Verb, bool) {
	switch action {
	case "pkg-install":
		return clipkgs.VerbInstall, true
	case "pkg-remove":
		return clipkgs.VerbRemove, true
	case "pkg-update":
		return clipkgs.VerbUpdate, true
	}
	return "", false
}

// resolvePackageJob builds the vendor command for one plugin job. It runs in
// the job lane, so it must not consult the request: everything it needs came
// with the payload.
func resolvePackageJob(cli, action, payload string) (clijob.Exec, error) {
	var p packageJobPayload
	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return clijob.Exec{}, fmt.Errorf("this job's arguments could not be read")
	}
	bin := clipkgs.Bin(cli)
	if bin == "" {
		return clijob.Exec{}, fmt.Errorf("PiCode manages no plugins for this CLI")
	}
	paths := clipkgs.Paths{Cwd: p.Cwd}
	target := clipkgs.Target{Name: p.Name, Source: p.Source, Scope: p.Scope, On: true}
	switch action {
	case "pkg-marketplace-add", "pkg-marketplace-update":
		dir, args, err := clipkgs.MarketArgv(cli, p.Action, paths, clipkgs.MarketRequest{
			Action: p.Action, Source: p.Source, Name: p.Name, Ref: p.Ref,
		})
		if err != nil {
			return clijob.Exec{}, err
		}
		return clijob.Exec{Exe: bin, Args: args, Dir: dir}, nil
	}
	verb, ok := packageVerb(action)
	if !ok {
		return clijob.Exec{}, fmt.Errorf("unknown package action")
	}
	dir, args, err := clipkgs.Argv(cli, verb, paths, target)
	if err != nil {
		return clijob.Exec{}, err
	}
	return clijob.Exec{Exe: bin, Args: args, Dir: dir}, nil
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
		errors.Is(err, store.ErrNotFound):
		return http.StatusBadRequest
	case errors.Is(err, clipkgs.ErrRosterShape):
		return http.StatusBadGateway
	}
	return http.StatusBadGateway
}

func cliPackagesViewOf(cli string, rep clipkgs.Report) cliPackagesView {
	rows := rep.Rows
	if rows == nil {
		rows = []clipkgs.Row{}
	}
	return cliPackagesView{
		CLI:    cli,
		Scopes: clipkgs.Scopes(cli),
		Caps:   clipkgs.Capabilities(cli),
		Notes:  clipkgs.Notes(cli),
		Rows:   rows,
		Note:   rep.Note,
		ReadAt: rep.ReadAt,
	}
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
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rep, err := clipkgs.List(ctx, cli, paths, scope, q.Get("refresh") == "1")
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cliPackagesViewOf(cli, rep))
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
		rep, err := clipkgs.Available(ctx, cli, paths, scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"cli":  cli,
			"rows": rep.Rows,
			"note": rep.Note,
		})
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
		rows, err := clipkgs.Marketplaces(ctx, cli, paths)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cli": cli, "marketplaces": rows})
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
		if _, _, err := clipkgs.Argv(v.CLI, mustVerb(action), paths, clipkgs.Target{
			Name: v.Name, Source: v.Source, Scope: v.Scope, On: true,
		}); err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		payload, err := json.Marshal(packageJobPayload{
			Source: v.Source, Name: v.Name, Scope: v.Scope, Workspace: v.Workspace, Cwd: paths.Cwd,
		})
		if err != nil {
			writeErr(w, http.StatusBadRequest, "Invalid request.")
			return
		}
		j, err := deps.CLIJobs.Start(v.CLI, action, jobKey(v.RequestKey, v.CLI, action, string(payload)), string(payload), v.ConfirmTerminals)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusAccepted, j)
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
		on := v.On == nil || *v.On
		verb := clipkgs.VerbDisable
		if on {
			verb = clipkgs.VerbEnable
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		if _, err := clipkgs.Run(ctx, v.CLI, verb, paths, clipkgs.Target{
			Name: v.Name, Source: v.Source, Scope: v.Scope, On: on,
		}); err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		// A synchronous mutation still has to reach every other open pane: the
		// feed carries the fact, the CLI's own list stays authoritative
		// (ADR-0048).
		publishPackageChange(deps, v.CLI, "pkg-toggle")
		rep, err := clipkgs.List(ctx, v.CLI, paths, v.Scope, true)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, cliPackagesViewOf(v.CLI, rep))
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
		req := clipkgs.MarketRequest{Action: action, Source: v.Source, Name: v.Name, Ref: v.Ref}
		if _, _, err := clipkgs.MarketArgv(v.CLI, action, paths, req); err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		if action == "remove" {
			if _, err := clipkgs.Market(ctx, v.CLI, action, paths, req); err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			publishPackageChange(deps, v.CLI, "pkg-marketplace-remove")
			rows, err := clipkgs.Marketplaces(ctx, v.CLI, paths)
			if err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"marketplaces": rows})
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
		writeJSON(w, http.StatusAccepted, j)
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
		out, err := clipkgs.Inspect(ctx, v.CLI, paths, clipkgs.Target{Name: name, Scope: v.Scope})
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cli": v.CLI, "output": out})
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

func mustVerb(action string) clipkgs.Verb {
	verb, _ := packageVerb(action)
	return verb
}
