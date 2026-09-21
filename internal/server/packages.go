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
	mux.HandleFunc("POST /api/packages", handleInstallPackage(deps))
	mux.HandleFunc("POST /api/packages/update", handleUpdatePackage(deps))
	mux.HandleFunc("DELETE /api/packages", handleRemovePackage(deps))
	registerPackageConfigRoutes(mux, deps)
}

// packageReadDriver resolves the driver a read answers from — the ?cli=
// parameter, Pi when absent — writing the refusal itself when no driver
// exists. A read that needs more than a roster gates on the driver's Caps.
func packageReadDriver(w http.ResponseWriter, r *http.Request) (pkgs.Driver, bool) {
	cli := strings.TrimSpace(r.URL.Query().Get("cli"))
	if !pkgs.Known(cli) {
		writeErr(w, http.StatusBadRequest,
			"no packages driver for "+cli+" — use "+strings.Join(pkgs.CLIs(), ", "))
		return nil, false
	}
	return pkgs.DriverFor(cli), true
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
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		rep, err := driver.CheckUpdates(ctx, pkgs.Query{WorkspacePath: dir})
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

type packageMutateReq struct {
	Source      string `json:"source"`
	Scope       string `json:"scope"`
	WorkspaceID string `json:"workspaceId"`
	AgentID     string `json:"agentId"`
}

func handleInstallPackage(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req packageMutateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
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
		source := r.URL.Query().Get("source")
		scope := r.URL.Query().Get("scope")
		wsID := r.URL.Query().Get("workspace")
		agentID := r.URL.Query().Get("agent")
		if source == "" {
			var req packageMutateReq
			_ = json.NewDecoder(r.Body).Decode(&req)
			source = req.Source
			if scope == "" {
				scope = req.Scope
			}
			if wsID == "" {
				wsID = req.WorkspaceID
			}
			if agentID == "" {
				agentID = req.AgentID
			}
		}
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
