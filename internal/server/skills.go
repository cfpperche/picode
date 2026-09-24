package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/skills"
)

// registerSkillRoutes serves the Skills tab (ADR-0196, docs/plans/skills.md).
// Slice 1 is read-only: what one CLI loads, from which folder, and why.
func registerSkillRoutes(mux Registrar, deps Deps) {
	mgr := skills.NewManager(filepath.Join(deps.DataDir, "skills", "stage"))
	mux.HandleFunc("GET /api/skills/report", handleSkillsReport(deps))
	// Slice 2: install, remove, update, check — PiCode's own writes into the
	// CLIs' folders and the skills CLI's locks (ADR-0196).
	mux.HandleFunc("POST /api/skills/preview", handleSkillsPreview(mgr))
	mux.HandleFunc("POST /api/skills", handleSkillsInstall(deps, mgr))
	mux.HandleFunc("DELETE /api/skills", handleSkillsRemove(deps, mgr))
	mux.HandleFunc("POST /api/skills/update", handleSkillsUpdate(deps, mgr))
	mux.HandleFunc("GET /api/skills/updates", handleSkillsUpdates(deps, mgr))
}

// skillsError maps the manager's refusals: a question the person can answer
// is 409 with its code, an expired preview 410, bad input 400, a source that
// failed to download 502.
func skillsError(w http.ResponseWriter, err error) {
	var c *skills.Conflict
	switch {
	case errors.As(err, &c):
		status := http.StatusConflict
		switch c.Code {
		case "invalid":
			status = http.StatusBadRequest
		case "gone":
			status = http.StatusGone
		}
		writeJSON(w, status, map[string]string{"error": c.Message, "code": c.Code})
	case errors.Is(err, skills.ErrSource):
		writeErr(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, skills.ErrNotInstalled):
		writeErr(w, http.StatusNotFound, err.Error())
	default:
		writeErr(w, http.StatusBadGateway, err.Error())
	}
}

// skillsTarget resolves the scope and the workspace id a request names.
func skillsTarget(deps Deps, scope, workspaceID string) (skills.Scope, string, error) {
	switch skills.Scope(scope) {
	case skills.Machine:
		return skills.Machine, "", nil
	case skills.Workspace:
		if workspaceID == "" || deps.Store == nil {
			return "", "", &skills.Conflict{Code: "invalid", Message: "a workspace install names its workspace"}
		}
		ws, err := deps.Store.GetWorkspace(workspaceID)
		if err != nil {
			return "", "", &skills.Conflict{Code: "invalid", Message: "workspace not found"}
		}
		return skills.Workspace, ws.Path, nil
	}
	return "", "", &skills.Conflict{Code: "invalid", Message: "scope is machine or workspace"}
}

func announceSkills(deps Deps, action, name string) {
	if deps.Feed != nil {
		deps.Feed.Ephemeral("skills.changed", map[string]any{"action": action, "name": name})
	}
}

func handleSkillsPreview(mgr *skills.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Source string `json:"source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		p, err := mgr.Preview(r.Context(), req.Source)
		if err != nil {
			skillsError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, p)
	}
}

func handleSkillsInstall(deps Deps, mgr *skills.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			skills.InstallReq
			Scope     string `json:"scope"`
			Workspace string `json:"workspace"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		scope, path, err := skillsTarget(deps, req.Scope, req.Workspace)
		if err != nil {
			skillsError(w, err)
			return
		}
		in := req.InstallReq
		in.Scope, in.Workspace = scope, path
		res, err := mgr.Install(in)
		if err != nil {
			skillsError(w, err)
			return
		}
		announceSkills(deps, res.Status, res.Name)
		writeJSON(w, http.StatusOK, res)
	}
}

func handleSkillsRemove(deps Deps, mgr *skills.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name      string `json:"name"`
			Scope     string `json:"scope"`
			Workspace string `json:"workspace"`
			Confirm   bool   `json:"confirm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		scope, path, err := skillsTarget(deps, req.Scope, req.Workspace)
		if err != nil {
			skillsError(w, err)
			return
		}
		res, err := mgr.Remove(skills.RemoveReq{Name: req.Name, Scope: scope, Workspace: path, Confirm: req.Confirm})
		if err != nil {
			skillsError(w, err)
			return
		}
		announceSkills(deps, "removed", req.Name)
		writeJSON(w, http.StatusOK, res)
	}
}

func handleSkillsUpdate(deps Deps, mgr *skills.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name      string `json:"name"`
			Scope     string `json:"scope"`
			Workspace string `json:"workspace"`
			Force     bool   `json:"force"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		scope, path, err := skillsTarget(deps, req.Scope, req.Workspace)
		if err != nil {
			skillsError(w, err)
			return
		}
		res, err := mgr.Update(r.Context(), skills.UpdateReq{Name: req.Name, Scope: scope, Workspace: path, Force: req.Force})
		if err != nil {
			skillsError(w, err)
			return
		}
		announceSkills(deps, res.Status, req.Name)
		writeJSON(w, http.StatusOK, res)
	}
}

func handleSkillsUpdates(deps Deps, mgr *skills.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		out := []skills.UpdateRow{}
		scopes := []string{string(skills.Machine)}
		if q.Get("workspace") != "" {
			scopes = append(scopes, string(skills.Workspace))
		}
		if s := q.Get("scope"); s != "" {
			scopes = []string{s}
		}
		for _, s := range scopes {
			scope, path, err := skillsTarget(deps, s, q.Get("workspace"))
			if err != nil {
				skillsError(w, err)
				return
			}
			rows, err := mgr.Check(r.Context(), skills.Target{Scope: scope, Workspace: path})
			if err != nil {
				skillsError(w, err)
				return
			}
			out = append(out, rows...)
		}
		writeJSON(w, http.StatusOK, map[string]any{"rows": out})
	}
}

// skillsTrusted answers a CLI's own trust record where PiCode can read it:
// Pi's trust.json. Every other CLI keeps trust where PiCode does not read,
// and the report says the skill loads once the folder is trusted there.
func skillsTrusted(cli, workspace string) *bool {
	if cli != "pi" {
		return nil
	}
	t := pisettings.Trusted(workspace)
	return &t
}

func handleSkillsReport(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli := strings.TrimSpace(r.URL.Query().Get("cli"))
		if cli == "" {
			cli = "pi"
		}
		q := skills.Query{CLI: cli, Trusted: skillsTrusted}
		if id := strings.TrimSpace(r.URL.Query().Get("workspace")); id != "" && deps.Store != nil {
			ws, err := deps.Store.GetWorkspace(id)
			if err != nil {
				writeErr(w, http.StatusNotFound, "workspace not found")
				return
			}
			q.Workspace = ws.Path
			// The agent scope (Pi, Omp): an agent of this CLI in that workspace.
			if id := strings.TrimSpace(r.URL.Query().Get("agent")); id != "" {
				if a, err := deps.Store.GetAgent(id); err == nil && a.WorkspaceID == ws.ID && (a.CLI == cli || (a.CLI == "" && cli == "pi")) {
					q.Agent = &skills.AgentInfo{ID: a.ID, Name: agentLayerName(a, ws), Isolated: a.PackagesIsolated}
				}
			}
		}
		rep, err := skills.Read(q)
		if errors.Is(err, skills.ErrUnknownCLI) {
			writeErr(w, http.StatusBadRequest, "unknown cli "+cli+"; skills are declared for: "+strings.Join(skills.CLIs(), ", "))
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
