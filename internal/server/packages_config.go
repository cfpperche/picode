// Package configuration GUI (ADR-0033 amendment #3 — the "No PiCode GUI
// page" clause of ADR-0033 is superseded for known adapters; pi files stay
// the only source of truth). GET/PUT/DELETE /api/packages/config read and
// write the same files the pi-roles extension reads and writes — never a
// copy in SQLite (ADR-0005/0010).
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cfpperche/picode/internal/pipkg"
	"github.com/cfpperche/picode/internal/store"
)

func registerPackageConfigRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/packages/config", handleGetPackageConfig(deps))
	mux.HandleFunc("PUT /api/packages/config", handlePutPackageConfig(deps))
	mux.HandleFunc("DELETE /api/packages/config", handleDeletePackageConfig(deps))
}

const pkgConfigRoles = "pi-roles"

// errNoStore and errBadAgentID are package-config-specific; the rest reuse
// packages.go's error values.
var (
	errNoStore    = pkgErr("the agent store is unavailable")
	errBadAgentID = pkgErr("this agent id cannot carry a roles overlay")
)

// rolesBaseFor resolves the directory a layer lives in, using the same rule
// the runtime uses: the agent's cwd (store.AgentCwd — WorkPath first) for
// the overlay, the workspace folder for the shared file.
func rolesBaseFor(wk store.Workspace, agent *store.Agent) string {
	if agent != nil {
		return store.AgentCwd(wk, *agent)
	}
	return wk.Path
}

// packageConfigView is the GET/PUT/DELETE payload: both layers plus the
// effective merge the agent actually runs with.
type packageConfigView struct {
	Package   string            `json:"package"`
	Kind      string            `json:"kind"`
	Workspace pipkg.RolesLayer  `json:"workspace"`
	Agent     *pipkg.RolesLayer `json:"agent,omitempty"`
	Effective pipkg.RolesConfig `json:"effective"`
}

func buildRolesView(deps Deps, workspaceID, agentID string) (packageConfigView, error) {
	view := packageConfigView{Package: pkgConfigRoles, Kind: "roles"}
	if deps.Store == nil {
		return view, errNoStore
	}
	wk, err := deps.Store.GetWorkspace(workspaceID)
	if err != nil {
		return view, err
	}
	view.Workspace = pipkg.ReadRolesLayer(
		filepath.Join(wk.Path, pipkg.RolesWorkspaceRel), pipkg.RolesWorkspaceRel)
	view.Effective = view.Workspace.Config
	if agentID == "" {
		return view, nil
	}
	a, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return view, err
	}
	if a.WorkspaceID != wk.ID {
		return view, store.ErrNotFound
	}
	rel := pipkg.RolesOverlayRel(a.ID)
	if rel == "" {
		return view, errBadAgentID
	}
	base := rolesBaseFor(wk, &a)
	layer := pipkg.ReadRolesLayer(filepath.Join(base, rel), rel)
	view.Agent = &layer
	view.Effective = pipkg.MergeRolesConfigs(view.Workspace.Config, layer.Config)
	return view, nil
}

func handleGetPackageConfig(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("package") != pkgConfigRoles {
			writeErr(w, http.StatusBadRequest, "unknown package — configuration is available for pi-roles")
			return
		}
		wsID := r.URL.Query().Get("workspace")
		if wsID == "" {
			writeErr(w, http.StatusBadRequest, "select a workspace — roles files live in the workspace folder")
			return
		}
		view, err := buildRolesView(deps, wsID, r.URL.Query().Get("agent"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		writeJSON(w, http.StatusOK, view)
	}
}

type packageConfigWrite struct {
	Package     string         `json:"package"`
	Scope       string         `json:"scope"` // workspace | agent
	WorkspaceID string         `json:"workspaceId"`
	AgentID     string         `json:"agentId"`
	Config      map[string]any `json:"config"`
	Force       bool           `json:"force"` // replace a file the parser rejects
}

func handlePutPackageConfig(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req packageConfigWrite
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Package != pkgConfigRoles {
			writeErr(w, http.StatusBadRequest, "unknown package — configuration is available for pi-roles")
			return
		}
		if req.Scope != "workspace" && req.Scope != "agent" {
			writeErr(w, http.StatusBadRequest, "scope must be workspace or agent — pi-roles has no machine-level file")
			return
		}
		cfg, err := pipkg.RolesConfigFromMap(req.Config)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if deps.Store == nil {
			writeErr(w, http.StatusBadRequest, errNoStore.Error())
			return
		}
		if req.WorkspaceID == "" {
			writeErr(w, http.StatusBadRequest, "select a workspace — roles files live in the workspace folder")
			return
		}
		wk, err := deps.Store.GetWorkspace(req.WorkspaceID)
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		var abs string
		if req.Scope == "workspace" {
			abs = filepath.Join(wk.Path, pipkg.RolesWorkspaceRel)
		} else {
			if req.AgentID == "" {
				writeErr(w, http.StatusBadRequest, "select an agent to save its overlay")
				return
			}
			a, err := deps.Store.GetAgent(req.AgentID)
			if err != nil {
				writeErr(w, statusForStore(err), err.Error())
				return
			}
			if a.WorkspaceID != wk.ID {
				writeErr(w, http.StatusNotFound, "agent is not in this workspace")
				return
			}
			rel := pipkg.RolesOverlayRel(a.ID)
			if rel == "" {
				writeErr(w, http.StatusBadRequest, errBadAgentID.Error())
				return
			}
			abs = filepath.Join(rolesBaseFor(wk, &a), rel)
		}
		// A file the parser refuses is never silently overwritten: the GUI
		// shows the parse error and the user decides (force = replace).
		if layer := pipkg.ReadRolesLayer(abs, filepath.Base(abs)); layer.Invalid != "" && !req.Force {
			writeErr(w, http.StatusConflict, layer.Invalid)
			return
		}
		var raw map[string]any
		if b, err := os.ReadFile(abs); err == nil && json.Unmarshal(b, &raw) != nil {
			raw = nil // invalid JSON has nothing worth preserving
		}
		if err := pipkg.WriteRolesFile(abs, cfg, raw); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceConfigChange(deps, req.Package, req.Scope)
		view, err := buildRolesView(deps, req.WorkspaceID, req.AgentID)
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, view)
	}
}

func handleDeletePackageConfig(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("package") != pkgConfigRoles {
			writeErr(w, http.StatusBadRequest, "unknown package — configuration is available for pi-roles")
			return
		}
		scope := q.Get("scope")
		if scope != "workspace" && scope != "agent" {
			writeErr(w, http.StatusBadRequest, "scope must be workspace or agent")
			return
		}
		if deps.Store == nil {
			writeErr(w, http.StatusBadRequest, errNoStore.Error())
			return
		}
		wk, err := deps.Store.GetWorkspace(q.Get("workspace"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		var abs string
		if scope == "workspace" {
			abs = filepath.Join(wk.Path, pipkg.RolesWorkspaceRel)
		} else {
			a, err := deps.Store.GetAgent(q.Get("agent"))
			if err != nil {
				writeErr(w, statusForStore(err), err.Error())
				return
			}
			if a.WorkspaceID != wk.ID {
				writeErr(w, http.StatusNotFound, "agent is not in this workspace")
				return
			}
			rel := pipkg.RolesOverlayRel(a.ID)
			if rel == "" {
				writeErr(w, http.StatusBadRequest, errBadAgentID.Error())
				return
			}
			abs = filepath.Join(rolesBaseFor(wk, &a), rel)
		}
		if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		announceConfigChange(deps, q.Get("package"), scope)
		view, err := buildRolesView(deps, q.Get("workspace"), q.Get("agent"))
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}
		writeJSON(w, http.StatusOK, view)
	}
}

// announceConfigChange lets other open views refetch (ADR-0048 ephemeral
// events). Nil-safe: the feed is optional infrastructure.
func announceConfigChange(deps Deps, pkg, scope string) {
	if deps.Feed == nil {
		return
	}
	deps.Feed.Ephemeral("packages.config", map[string]any{"package": pkg, "scope": scope})
}
