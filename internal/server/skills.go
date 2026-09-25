package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cfpperche/picode/internal/pisettings"
	"github.com/cfpperche/picode/internal/skills"
	"github.com/cfpperche/picode/internal/store"
)

// registerSkillRoutes serves the Skills tab (ADR-0196, docs/plans/skills.md).
// Slice 1 is read-only: what one CLI loads, from which folder, and why.
func registerSkillRoutes(mux Registrar, deps Deps) {
	mgr := skills.NewManager(filepath.Join(deps.DataDir, "skills", "stage"))
	mgr.CacheRoot = filepath.Join(deps.DataDir, "skills", "cache")
	mux.HandleFunc("GET /api/skills/report", handleSkillsReport(deps))
	// Slice 2: install, remove, update, check — PiCode's own writes into the
	// CLIs' folders and the skills CLI's locks (ADR-0196).
	mux.HandleFunc("POST /api/skills/preview", handleSkillsPreview(mgr))
	mux.HandleFunc("POST /api/skills", handleSkillsInstall(deps, mgr))
	mux.HandleFunc("DELETE /api/skills", handleSkillsRemove(deps, mgr))
	mux.HandleFunc("POST /api/skills/update", handleSkillsUpdate(deps, mgr))
	mux.HandleFunc("GET /api/skills/updates", handleSkillsUpdates(deps, mgr))
	// Slice 5: the Marketplace catalog, its sources and the skills.sh switch.
	registerSkillCatalogRoutes(mux, deps)
	// Slice 3: each CLI's own per-skill switch, written in its own file.
	mux.HandleFunc("POST /api/skills/toggle", handleSkillsToggle(deps))
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

// skillsAgent resolves the agent an agent-scope write names: it exists and
// its CLI takes an agent's own skills at launch.
func skillsAgent(deps Deps, id string) (store.Agent, error) {
	if id == "" || deps.Store == nil {
		return store.Agent{}, &skills.Conflict{Code: "invalid", Message: "an agent install names its agent"}
	}
	a, err := deps.Store.GetAgent(id)
	if err != nil {
		return store.Agent{}, &skills.Conflict{Code: "invalid", Message: "agent not found"}
	}
	if spec, ok := skills.For(a.CLI); !ok || !spec.AgentScope {
		return store.Agent{}, &skills.Conflict{Code: "invalid", Message: "this agent's CLI cannot take skills of its own at launch"}
	}
	return a, nil
}

func agentSkillsOf(a store.Agent) []skills.AgentSkill {
	out := make([]skills.AgentSkill, 0, len(a.Skills))
	for _, sk := range a.Skills {
		out = append(out, skills.AgentSkill{Name: sk.Name, Digest: sk.Digest, Source: sk.Source, Dir: sk.Dir})
	}
	return out
}

// agentSkillsMu serializes read-modify-write of an agent's list.
var agentSkillsMu sync.Mutex

// installForAgent caches the staged skill and adds it to the agent's list;
// a skill of the same name is replaced (the list holds one per name).
func installForAgent(deps Deps, mgr *skills.Manager, in skills.InstallReq, agentID string) (skills.Result, error) {
	agentSkillsMu.Lock()
	defer agentSkillsMu.Unlock()
	a, err := skillsAgent(deps, agentID)
	if err != nil {
		return skills.Result{}, err
	}
	in.Scope = skills.Agent
	res, err := mgr.Install(in)
	if err != nil {
		return res, err
	}
	for _, sk := range a.Skills {
		if sk.Name == res.Name && sk.Digest == res.Digest {
			res.Status = "already"
			return res, nil
		}
		if sk.Name == res.Name {
			res.Status = "replaced"
		}
	}
	next := append(append([]store.AgentSkill{}, a.Skills...), store.AgentSkill{Name: res.Name, Digest: res.Digest, Source: res.Source, Dir: res.Dir})
	if _, err := deps.Store.SetAgentSkills(a.ID, next); err != nil {
		return res, err
	}
	return res, nil
}

// removeForAgent drops the skill from the agent's list. The cached copy
// stays: another agent or a restored one may name it.
func removeForAgent(deps Deps, name, agentID string) (skills.Result, error) {
	agentSkillsMu.Lock()
	defer agentSkillsMu.Unlock()
	a, err := skillsAgent(deps, agentID)
	if err != nil {
		return skills.Result{}, err
	}
	next := make([]store.AgentSkill, 0, len(a.Skills))
	found := false
	for _, sk := range a.Skills {
		if sk.Name == name {
			found = true
			continue
		}
		next = append(next, sk)
	}
	if !found {
		return skills.Result{}, skills.ErrNotInstalled
	}
	if _, err := deps.Store.SetAgentSkills(a.ID, next); err != nil {
		return skills.Result{}, err
	}
	return skills.Result{Name: name, Status: "removed"}, nil
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
			Agent     string `json:"agent"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		if req.Scope == string(skills.Agent) {
			res, err := installForAgent(deps, mgr, req.InstallReq, req.Agent)
			if err != nil {
				skillsError(w, err)
				return
			}
			announceSkills(deps, res.Status, res.Name)
			writeJSON(w, http.StatusOK, res)
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
			Agent     string `json:"agent"`
			Confirm   bool   `json:"confirm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "bad request")
			return
		}
		if req.Scope == string(skills.Agent) {
			res, err := removeForAgent(deps, req.Name, req.Agent)
			if err != nil {
				skillsError(w, err)
				return
			}
			announceSkills(deps, "removed", req.Name)
			writeJSON(w, http.StatusOK, res)
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
			// The agent scope (Pi, Omp, Claude Code): an agent of this CLI in
			// that workspace. Isolation is Pi's and Omp's (their packages).
			if id := strings.TrimSpace(r.URL.Query().Get("agent")); id != "" {
				if a, err := deps.Store.GetAgent(id); err == nil && a.WorkspaceID == ws.ID && (a.CLI == cli || (a.CLI == "" && cli == "pi")) {
					isolated := a.PackagesIsolated && (cli == "pi" || cli == "omp")
					q.Agent = &skills.AgentInfo{ID: a.ID, Name: agentLayerName(a, ws), Isolated: isolated, Skills: agentSkillsOf(a)}
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

// handleSkillsToggle flips one skill's switch in the CLI's own settings. The
// row is found again in a fresh report, so the path written is always one the
// reader saw, never one the request invented.
func handleSkillsToggle(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CLI       string `json:"cli"`
			Scope     string `json:"scope"`
			Workspace string `json:"workspace"`
			Dir       string `json:"dir"`
			Enabled   bool   `json:"enabled"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		q := skills.Query{CLI: strings.TrimSpace(req.CLI), Trusted: skillsTrusted}
		if req.Workspace != "" {
			if deps.Store == nil {
				writeErr(w, http.StatusNotFound, "workspace not found")
				return
			}
			ws, err := deps.Store.GetWorkspace(req.Workspace)
			if err != nil {
				writeErr(w, http.StatusNotFound, "workspace not found")
				return
			}
			q.Workspace = ws.Path
		}
		rep, err := skills.Read(q)
		if errors.Is(err, skills.ErrUnknownCLI) {
			writeErr(w, http.StatusBadRequest, "unknown cli "+req.CLI)
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		var row *skills.Row
		for i := range rep.Rows {
			if rep.Rows[i].Dir == req.Dir && string(rep.Rows[i].Scope) == req.Scope {
				row = &rep.Rows[i]
			}
		}
		if row == nil {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "That skill is no longer where the list said; read it again.", "code": "stale"})
			return
		}
		res, err := skills.SetEnabled(r.Context(), skills.ToggleReq{CLI: q.CLI, Workspace: q.Workspace, Row: *row, Enabled: req.Enabled})
		switch {
		case errors.Is(err, skills.ErrNoSwitch):
			writeErr(w, http.StatusBadRequest, cliLabel(q.CLI)+" has no switch for this skill")
			return
		case errors.Is(err, skills.ErrStale):
			writeJSON(w, http.StatusConflict, map[string]string{"error": "The settings file changed while you were looking; read it again.", "code": "stale"})
			return
		case err != nil:
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error(), "code": "refused"})
			return
		}
		announceSkills(deps, "toggled", res.Name)
		writeJSON(w, http.StatusOK, res)
	}
}
