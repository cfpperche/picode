package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/pkgs"
	"github.com/cfpperche/picode/internal/store"
)

func registerPackageRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/packages", handleListPackages(deps))
	mux.HandleFunc("GET /api/packages/gallery", handlePackageGallery)
	mux.HandleFunc("GET /api/packages/report", handlePackageReport(deps))
	mux.HandleFunc("GET /api/packages/updates", handlePackageUpdates(deps))
	// The CLI's own surface (ADR-0176): every CLI answers the same paths, and
	// the request's `cli` is what resolves its driver — one path per verb
	// instead of a family per vendor identity. Pi's own mutations and every
	// agent-layer write are PiCode's own calls and name no CLI, which is what
	// keeps the two apart on one path (`ownCLI` below).
	mux.HandleFunc("GET /api/packages/available", handlePackageAvailable(deps))
	mux.HandleFunc("GET /api/packages/marketplaces", handlePackageMarketplaces(deps))
	mux.HandleFunc("POST /api/packages/toggle", handlePackageToggle(deps))
	mux.HandleFunc("POST /api/packages/marketplace", handlePackageMarketplace(deps))
	mux.HandleFunc("POST /api/packages/inspect", handlePackageInspect(deps))
	mux.HandleFunc("POST /api/packages", handleInstallPackage(deps))
	mux.HandleFunc("POST /api/packages/update", handleUpdatePackage(deps))
	mux.HandleFunc("DELETE /api/packages", handleRemovePackage(deps))
	registerPackageConfigRoutes(mux, deps)
}

// packageDriver resolves the driver one request answers from — the `cli` of a
// read, the `cli` field of a mutation, Pi when absent — writing the refusal
// itself when no driver exists, so one sentence names what does.
func packageDriver(w http.ResponseWriter, cli string) (pkgs.Driver, bool) {
	cli = strings.TrimSpace(cli)
	if !pkgs.Known(cli) {
		writeErr(w, http.StatusBadRequest,
			"no packages driver for "+cli+" — use "+strings.Join(pkgs.CLIs(), ", "))
		return nil, false
	}
	return pkgs.DriverFor(cli), true
}

// packageReadDriver is the same for a read, whose `cli` rides the query.
func packageReadDriver(w http.ResponseWriter, r *http.Request) (pkgs.Driver, bool) {
	return packageDriver(w, r.URL.Query().Get("cli"))
}

// ownCLI answers the CLI whose own surface a mutation request is for. PiCode's
// own calls never name one — Pi's mutations run through `pipkg` on these same
// paths, and an agent-layer write is PiCode's own list for every CLI — so an
// absent (or Pi's own) `cli` is what tells the two apart on one path
// (ADR-0176's transport rule, which the pane's `directMutation` states once).
func ownCLI(cli string) string {
	if id := strings.TrimSpace(cli); id != "" && id != "pi" {
		return id
	}
	return ""
}

// handlePackageUpdates is the badge read: the driver answers what its catalog
// has moved ahead of, and Pi's payload keeps the shape the pane parses.
func handlePackageUpdates(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		driver, ok := packageReadDriver(w, r)
		if !ok {
			return
		}
		if !driver.Caps().Update {
			writeErr(w, http.StatusBadRequest, driver.ID()+": "+pkgs.ErrNoUpdateCheck.Error())
			return
		}
		dir, err := packageProjectDir(deps, r.URL.Query().Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		// The vendor word the pane is looking at, so a project-scope roster is
		// the one compared with the catalog. Pi's own check ignores it: its
		// rows' scopes come from the settings files either way.
		word := strings.TrimSpace(r.URL.Query().Get("vendor"))
		if word == "" {
			word = strings.TrimSpace(r.URL.Query().Get("scope"))
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		rep, err := driver.CheckUpdates(ctx, pkgs.Query{WorkspacePath: dir, Vendor: word})
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, pkgs.LegacyUpdates(rep.Rows))
	}
}

func handlePackageGallery(w http.ResponseWriter, r *http.Request) {
	page, err := pipkg.SearchGallery(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func handleListPackages(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rep, err := loadPackageReport(r.Context(), deps, r.URL.Query().Get("workspace"), r.URL.Query().Get("agent"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

// loadPackageReport is GET /api/packages: Pi's driver reads the settings and
// the agent row, and the unified report is mapped back to the JSON the pane
// parses. One source of truth, the same bytes (ADR-0176 slice 1c).
func loadPackageReport(ctx context.Context, deps Deps, workspaceID, agentID string) (pipkg.Report, error) {
	dir, err := packageProjectDir(deps, workspaceID)
	if err != nil {
		return pipkg.Report{}, err
	}
	q := pkgs.Query{WorkspacePath: dir}
	if agentID != "" && deps.Store != nil {
		a, err := deps.Store.GetAgent(agentID)
		switch {
		case err == nil:
			q.AgentSources = a.Packages
			q.AgentIsolated = a.PackagesIsolated
		case errors.Is(err, store.ErrNotFound):
			// A terminal id (or a stale agent) must not hide machine packages.
		default:
			return pipkg.Report{}, err
		}
	}
	rep, err := pkgs.DriverFor("pi").List(ctx, q)
	if err != nil {
		return pipkg.Report{}, err
	}
	return rep.Legacy(), nil
}

// packageMutateReq is one install, removal or update as this family takes it.
// Both vocabularies live here because one path answers both: PiCode's own call
// names the source, its layer and the ids the fresh list is read back for
// (`workspaceId`/`agentId`, no `cli`), while a CLI's own surface names the CLI,
// the plugin and the lane's request key.
type packageMutateReq struct {
	Source      string `json:"source"`
	Scope       string `json:"scope"`
	WorkspaceID string `json:"workspaceId"`
	AgentID     string `json:"agentId"`
	// The CLI's own surface: `cli` picks the driver, `workspace` is the folder a
	// project-scope mutation runs in, and `name`/`requestKey` are the plugin and
	// the idempotency key the lane reserves by.
	CLI              string `json:"cli"`
	Workspace        string `json:"workspace"`
	Name             string `json:"name"`
	RequestKey       string `json:"requestKey"`
	ConfirmTerminals bool   `json:"confirmTerminals"`
}

// own is the same request as the CLI's own surface takes it: the driver the
// request named, the folder under the name the pane sends it, and the plugin
// by name.
func (req packageMutateReq) own() cliPackageRequest {
	return cliPackageRequest{
		CLI:              ownCLI(req.CLI),
		Workspace:        firstNonEmpty(req.Workspace, req.WorkspaceID),
		Scope:            req.Scope,
		Name:             req.Name,
		Source:           req.Source,
		RequestKey:       req.RequestKey,
		ConfirmTerminals: req.ConfirmTerminals,
	}
}

func handleInstallPackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req packageMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if ownCLI(req.CLI) != "" {
			vendorPackageJob(deps, "pkg-install", req.own())(w, r)
			return
		}
		if req.Scope == "agent" {
			rep, err := mutateAgentPackage(r.Context(), deps, req.AgentID, req.Source, true)
			if err != nil {
				writeErr(w, statusForStore(err), err.Error())
				return
			}
			writeJSON(w, http.StatusOK, rep)
			return
		}
		opts, dir, err := packageMutate(deps, req.Scope, req.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if err := pipkg.Install(ctx, deps.AgentCmd, req.Source, opts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := loadPackageReport(r.Context(), deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		_ = dir
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleUpdatePackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req packageMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if ownCLI(req.CLI) != "" {
			vendorPackageJob(deps, "pkg-update", req.own())(w, r)
			return
		}
		if req.Scope == "agent" {
			writeErr(w, http.StatusBadRequest, "this-agent packages update on the next start")
			return
		}
		opts, dir, err := packageMutate(deps, req.Scope, req.WorkspaceID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
		defer cancel()
		if err := pipkg.Update(ctx, deps.AgentCmd, pipkg.AbsPathSource(req.Source, packageSettingsDir(req.Scope, dir)), opts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := loadPackageReport(r.Context(), deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		_ = dir
		writeJSON(w, http.StatusOK, rep)
	}
}

func handleRemovePackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// A removal names what it takes back in the query (Pi's own call) or in
		// the body (the CLI's own surface, with the plugin by name); one path
		// reads both.
		source := r.URL.Query().Get("source")
		scope := r.URL.Query().Get("scope")
		wsID := r.URL.Query().Get("workspace")
		agentID := r.URL.Query().Get("agent")
		var req packageMutateReq
		if r.Body != nil && r.ContentLength != 0 {
			_ = json.NewDecoder(r.Body).Decode(&req)
		}
		if ownCLI(req.CLI) != "" {
			vendorPackageJob(deps, "pkg-remove", req.own())(w, r)
			return
		}
		source = firstNonEmpty(source, req.Source)
		scope = firstNonEmpty(scope, req.Scope)
		wsID = firstNonEmpty(wsID, req.WorkspaceID)
		agentID = firstNonEmpty(agentID, req.AgentID)
		if scope == "agent" {
			rep, err := mutateAgentPackage(r.Context(), deps, agentID, source, false)
			if err != nil {
				writeErr(w, statusForStore(err), err.Error())
				return
			}
			writeJSON(w, http.StatusOK, rep)
			return
		}
		opts, dir, err := packageMutate(deps, scope, wsID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		source = pipkg.AbsPathSource(source, packageSettingsDir(scope, dir))
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := pipkg.Remove(ctx, deps.AgentCmd, source, opts); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		rep, err := loadPackageReport(r.Context(), deps, wsID, agentID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}

func mutateAgentPackage(ctx context.Context, deps Deps, agentID, source string, add bool) (pipkg.Report, error) {
	if deps.Store == nil || strings.TrimSpace(agentID) == "" {
		return pipkg.Report{}, errNeedAgent
	}
	if err := pipkg.ValidSource(source); err != nil {
		return pipkg.Report{}, err
	}
	a, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return pipkg.Report{}, err
	}
	next := []string{}
	if add {
		next = append(append([]string{}, a.Packages...), source)
	} else {
		for _, s := range a.Packages {
			if s != source {
				next = append(next, s)
			}
		}
	}
	a, err = deps.Store.SetAgentPackages(agentID, next)
	if err != nil {
		return pipkg.Report{}, err
	}
	wsID := a.WorkspaceID
	return loadPackageReport(ctx, deps, wsID, agentID)
}

func packageProjectDir(deps Deps, id string) (string, error) {
	if id == "" {
		return "", nil
	}
	if deps.Store == nil {
		return "", errNoWorkspace
	}
	ws, err := deps.Store.GetWorkspace(id)
	if err != nil {
		return "", err
	}
	return ws.Path, nil
}

func packageMutate(deps Deps, scope, workspaceID string) (pipkg.MutateOpts, string, error) {
	if scope == "" || scope == "user" {
		dir, err := packageProjectDir(deps, workspaceID)
		if err != nil {
			return pipkg.MutateOpts{}, "", err
		}
		return pipkg.MutateOpts{}, dir, nil
	}
	if scope != "project" {
		return pipkg.MutateOpts{}, "", errBadScope
	}
	dir, err := packageProjectDir(deps, workspaceID)
	if err != nil {
		return pipkg.MutateOpts{}, "", err
	}
	if dir == "" {
		return pipkg.MutateOpts{}, "", errNeedWorkspace
	}
	return pipkg.MutateOpts{Local: true, Cwd: dir}, dir, nil
}

// packageSettingsDir is where pi keeps the scope's settings.json — the
// directory a stored relative path source is relative to.
func packageSettingsDir(scope, projectDir string) string {
	if scope == "project" && projectDir != "" {
		return filepath.Join(projectDir, ".pi")
	}
	return pipkg.UserDir()
}

func statusForStore(err error) int {
	if err == store.ErrNotFound {
		return http.StatusNotFound
	}
	return http.StatusBadRequest
}

type pkgErr string

func (e pkgErr) Error() string { return string(e) }

const (
	errBadScope      pkgErr = "scope must be user, project, or agent"
	errNeedWorkspace pkgErr = "select an agent to install into a workspace"
	errNeedAgent     pkgErr = "select an agent"
	errNoWorkspace   pkgErr = "select an agent first"
)
