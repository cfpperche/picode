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

// The CLIs' own package surface (ADR-0167, ADR-0176). Eight of the CLIs PiCode
// drives manage their own plugins; PiCode declares what each one exposes
// (internal/clipkgs), runs the vendor's own command, and keeps no package
// database. Every one of their verbs is answered on the same `/api/packages*`
// family Pi's own calls use, with the CLI named in the request: the driver runs
// the CLI's own command, or performs the write the CLI has no command for, and
// every mutation that installs or fetches is a durable job in the lane
// ADR-0087 built, so a PiCode restart never replays one and the pane can show
// progress from the events that lane already publishes.
//
// The answers are the unified model (`pkgs.Report`, `pkgs.Row`) on every route.
// The `/api/cli-packages*` alias and the mappers in `internal/pkgs/
// guest_view.go` that reconstructed the bytes the guest pane used to parse are
// gone with it (ADR-0176's one-release window, closed by the owner 2026-09-22).
//
// A job carries its own arguments in the job payload: `Resolve` runs after a
// restart too, and the argv of `pkg-install` depends on the plugin, the scope
// and the workspace folder. The driver builds that argv twice — once before the
// job is reserved, and again inside the lane — from the same payload shape, so
// the two cannot disagree.

// cliPackageRequest is one request on a CLI's own surface: the CLI it names,
// the folder a project-scope mutation runs in, the plugin by name and source,
// and the idempotency key the lane reserves by. A marketplace action adds its
// action, its ref and the source it names; a toggle adds the state asked for.
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

// vendorQuery is one CLI read as the driver takes it: the scope word the
// request carried (the class cannot say `local`) and the folder a project-scope
// read runs in.
func vendorQuery(scope, cwd string) pkgs.Query {
	return pkgs.Query{Vendor: scope, WorkspacePath: cwd}
}

// handlePackageAvailable is GET /api/packages/available: the CLI's own
// installable list, read from the vendor's own command and answered as the
// unified report. A CLI with no such list refuses — the model's own sentence
// where the interface has one (`ErrNoCatalog`, Pi's gallery), the vendor's own
// reason where the CLI simply has no such verb — never an empty catalog.
func handlePackageAvailable(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := strings.TrimSpace(q.Get("cli")), q.Get("scope")
		drv, ok := packageDriver(w, cli)
		if !ok {
			return
		}
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rep, err := drv.Available(ctx, vendorQuery(scope, paths.Cwd))
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// marketplacesAnswer is a CLI's own source list as this family answers it: the
// CLI it belongs to and its rows, in the unified row shape. A read and a
// removal answer the same shape — one fact, one payload — and the list is never
// null, because an empty one is a real answer ("this CLI lists no sources").
type marketplacesAnswer struct {
	CLI          string     `json:"cli"`
	Marketplaces []pkgs.Row `json:"marketplaces"`
}

func marketplacesAnswerOf(cli string, rows []pkgs.Row) marketplacesAnswer {
	if rows == nil {
		rows = []pkgs.Row{}
	}
	return marketplacesAnswer{CLI: cli, Marketplaces: rows}
}

// handlePackageMarketplaces is GET /api/packages/marketplaces: the sources the
// CLI is configured with, which the pane needs before it can manage them — a
// removal is not the only way to learn them (found in the pane's own build,
// 2026-09-20). A CLI that keeps none refuses (`ErrNoMarketplaces`, or the
// vendor's own reason for the verb it does not have) rather than answering a
// list that would read as "none configured".
func handlePackageMarketplaces(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		cli, scope := strings.TrimSpace(q.Get("cli")), q.Get("scope")
		drv, ok := packageDriver(w, cli)
		if !ok {
			return
		}
		paths, err := cliPackagePaths(deps, cli, q.Get("workspace"), scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		rows, err := drv.Marketplaces(ctx, vendorQuery(scope, paths.Cwd))
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, marketplacesAnswerOf(cli, rows))
	}
}

// vendorPackageJob answers an install, a removal or an update on the CLI's own
// surface. The answer is the job — or, for a mutation the driver runs itself (a
// CLI whose removal is a write of one of its own files), the CLI's fresh list,
// because there is no argv for the lane to run. The pane follows `cli.job`
// events for the first, and the `cli.packages` event for the second.
func vendorPackageJob(deps Deps, action string, v cliPackageRequest) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := packageDriver(w, v.CLI); !ok {
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
		cmd, err := packageCommand(r.Context(), v.CLI, action, payload)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		if cmd.Exe == "" {
			// A command with no argv is a mutation the driver performs itself —
			// OpenCode's removal, which is a splice of its own opencode.json,
			// and Omp's workspace `extensions` entry — so there is nothing to
			// hand the lane and the write has already happened. The answer is
			// the CLI's fresh list, the way a synchronous mutation always
			// answers (ADR-0048).
			publishPackageChange(deps, v.CLI, action)
			read := vendorQuery(v.Scope, paths.Cwd)
			read.Fresh = true
			ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
			defer cancel()
			rep, err := pkgs.DriverFor(v.CLI).List(ctx, read)
			if err != nil {
				writeErr(w, statusForPackageErr(err), err.Error())
				return
			}
			writeJSON(w, http.StatusOK, rep)
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

// handlePackageToggle is POST /api/packages/toggle: it enables or disables one
// plugin. It is synchronous: the vendor call is a local state change, and the
// answer is the CLI's fresh report so the pane never has to guess the row's
// next state.
func handlePackageToggle(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var v cliPackageRequest
		if !readCLIJSON(w, r, &v) {
			return
		}
		drv, ok := packageDriver(w, v.CLI)
		if !ok {
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		read := vendorQuery(v.Scope, paths.Cwd)
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
		writeJSON(w, http.StatusOK, rep)
	}
}

// handlePackageMarketplace is POST /api/packages/marketplace: it manages the
// CLI's own marketplace sources — a fetch (add, update) is a job, a removal is
// a local change.
func handlePackageMarketplace(deps Deps) http.HandlerFunc {
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
		drv, ok := packageDriver(w, v.CLI)
		if !ok {
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		read := vendorQuery(v.Scope, paths.Cwd)
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
			writeJSON(w, http.StatusOK, marketplacesAnswerOf(v.CLI, rows))
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

// inspectAnswer is the vendor's own inspection output, verbatim, with the CLI
// it came from.
type inspectAnswer struct {
	CLI    string `json:"cli"`
	Output string `json:"output"`
}

// handlePackageInspect is POST /api/packages/inspect: the vendor's own
// inspection output for one plugin, in the vendor's words.
func handlePackageInspect(deps Deps) http.HandlerFunc {
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
		drv, ok := packageDriver(w, v.CLI)
		if !ok {
			return
		}
		paths, err := cliPackagePaths(deps, v.CLI, v.Workspace, v.Scope)
		if err != nil {
			writeErr(w, statusForPackageErr(err), err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
		defer cancel()
		cmd, out, err := drv.Inspect(ctx, vendorQuery(v.Scope, paths.Cwd), pkgs.Target{Name: name})
		if err != nil {
			writePackageErr(w, err, cmd.Line)
			return
		}
		writeJSON(w, http.StatusOK, inspectAnswer{CLI: v.CLI, Output: out})
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
