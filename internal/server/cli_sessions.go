package server

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/clisession"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// registerCLISessionRoutes adds the per-CLI sessions surface (ADR-0079):
// a read-only index for every catalog CLI, plus pi's management surface
// (delete, adopt, auto-clean) scoped under /api/clis/pi. The legacy
// /api/sessions/all, /api/pi-sessions* and /api/session-cleanup routes
// were folded into these.
func registerCLISessionRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/clis/{cli}/sessions", handleCLISessions(deps))
	mux.HandleFunc("POST /api/clis/{cli}/sessions/delete", handleCLIDeleteSession(deps))
	mux.HandleFunc("POST /api/clis/{cli}/sessions/adopt", handleCLIAdoptSession(deps))
	mux.HandleFunc("GET /api/clis/{cli}/sessions/cleanup", handleCLICleanupSetting(deps))
	mux.HandleFunc("PUT /api/clis/{cli}/sessions/cleanup", handleCLICleanupSetting(deps))
	registerCLIHandoffRoutes(mux, deps)
}

// handleCLISessions answers GET /api/clis/{cli}/sessions with optional
// scoping: ?cwd=<folder> filters every CLI by folder; ?workspace=<id>
// scopes to a PiCode workspace — for pi that unions the shared cwd bucket
// with each of the workspace's agents' private dirs (ADR-0040), which a
// plain cwd filter cannot express. Decision table (covered in
// cli_sessions_test.go and session_manage_test.go):
//
//	unknown catalog CLI                     → 404
//	known CLI, nothing on disk              → 200, empty list
//	?cwd / ?workspace filters               → only matching sessions
//	malformed or undeclared files           → skipped, never a 500
//	session cwd matches a PiCode workspace  → tagged workspaceId/workspace
//	pi rows                                 → inUseBy + cleanupDays ride along
//	session was handed off / came from one  → handoff.{to,from} (ADR-0087)
func handleCLISessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		src, ok := clisession.Get(cli.ID)
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		var wk *store.Workspace
		if id := r.URL.Query().Get("workspace"); id != "" {
			resolved, err := deps.Store.GetWorkspace(id)
			if err != nil {
				writeErr(w, statusForStore(err), err.Error())
				return
			}
			wk = &resolved
		}
		var rows []clisession.Summary
		var err error
		switch {
		case cli.ID == "pi" && wk != nil:
			rows, err = clisession.PIDirs(workspaceSessionDirs(deps, *wk)...)
		default:
			cwd := r.URL.Query().Get("cwd")
			if wk != nil {
				cwd = wk.Path
			}
			rows, err = src.List(cwd)
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		use := sessionUseBy(deps)
		wss, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		lineage := lineageIndex(deps)
		views := make([]cliSessionView, 0, len(rows))
		var total int64
		for _, s := range rows {
			total += s.Size
			v := cliSessionView{Summary: s, Handoff: lineage[[2]string{cli.ID, s.ID}]}
			if cli.ID == "pi" {
				if u, inUse := use[s.Path]; inUse {
					v.InUseBy = &u
				}
			}
			for _, w := range wss {
				if w.Path != "" && w.Path == s.Cwd {
					v.WorkspaceID = w.ID
					v.Workspace = w.Name
					break
				}
			}
			views = append(views, v)
		}
		out := map[string]any{
			"cli":        cli.ID,
			"sessions":   views,
			"count":      len(views),
			"totalBytes": total,
		}
		if cli.ID == "pi" {
			out["cleanupDays"] = cleanupDaysSetting(deps)
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// handleCLIDeleteSession removes one pi session file: POST
// /api/clis/{cli}/sessions/delete {path} (a POST action, not DELETE —
// DELETE /api/clis/{cli}/sessions would collide with the profiles route
// shape in ServeMux). Pi-only — other CLIs' session files are never
// written or deleted by PiCode. Decision table:
//
//	unknown CLI / non-pi CLI          → 404 / 400
//	path not under the pi root        → 400 "session is not on this machine"
//	path is an agent's current session → 409 in-use
//	file already gone                 → 404
//	ok                                → 200 {ok}
func handleCLIDeleteSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		if cli.ID != "pi" {
			writeErr(w, http.StatusBadRequest, "only pi sessions can be deleted")
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
			writeErr(w, http.StatusBadRequest, "path required")
			return
		}
		path, err := filepath.Abs(strings.TrimSpace(req.Path))
		if err != nil || !session.UnderRoot(session.Root(), path) {
			writeErr(w, http.StatusBadRequest, "session is not on this machine")
			return
		}
		if u, inUse := sessionUseBy(deps)[path]; inUse {
			writeErr(w, http.StatusConflict, "in use by agent "+u.AgentName+" — point that agent at another session first")
			return
		}
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				writeErr(w, http.StatusNotFound, "session file is gone")
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := os.Remove(path); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		event := map[string]any{"path": path, "scope": "machine"}
		if wss, err := deps.Store.ListWorkspaces(); err == nil {
			for _, wk := range wss {
				if safeSessionPath(path, workspaceSessionDirs(deps, wk)...) {
					event["workspaceId"] = wk.ID
					break
				}
			}
		}
		_ = deps.Store.AppendEvent("session_deleted", nil, nil, event)
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// handleCLIAdoptSession turns one pi JSONL into a stopped agent:
// POST /api/clis/{cli}/sessions/adopt {path}. Pi-only — other CLIs'
// sessions have no PiCode agent representation.
func handleCLIAdoptSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli, ok := clilaunch.Find(r.PathValue("cli"))
		if !ok {
			writeErr(w, http.StatusNotFound, "Unknown CLI.")
			return
		}
		if cli.ID != "pi" {
			writeErr(w, http.StatusBadRequest, "only pi sessions can be adopted")
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Path) == "" {
			writeErr(w, http.StatusBadRequest, "path required")
			return
		}
		src := strings.TrimSpace(req.Path)
		if !session.UnderRoot(session.Root(), src) {
			writeErr(w, http.StatusBadRequest, "session is not on this machine")
			return
		}
		sum, err := session.Summarize(src)
		if err != nil {
			if os.IsNotExist(err) {
				writeErr(w, http.StatusNotFound, "That session is gone.")
				return
			}
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		cwd := strings.TrimSpace(sum.Cwd)
		if cwd == "" {
			writeErr(w, http.StatusBadRequest, "This session has no folder.")
			return
		}
		copyPath, err := session.CopyFile(src)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := strings.TrimSpace(sum.Name)
		if name == "" {
			name = strings.TrimSpace(sum.Preview)
		}
		if name == "" {
			name = "Pi session"
		}
		if len([]rune(name)) > 60 {
			r := []rune(name)
			name = string(r[:60])
		}
		wsID, work := adoptHome(deps, cwd)
		agent, err := deps.Store.AddAgent(wsID, name, work)
		if err != nil {
			_ = os.Remove(copyPath)
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		patch := store.AgentPatch{SessionPath: &copyPath}
		if sum.Provider != "" {
			patch.Provider = &sum.Provider
		}
		if sum.Model != "" {
			patch.Model = &sum.Model
		}
		if sum.Thinking != "" {
			patch.Thinking = &sum.Thinking
		}
		agent, err = deps.Store.UpdateAgent(agent.ID, patch)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, agentView{Agent: agent, Mode: string(modeStopped)})
	}
}

// adoptHome reuses the workspace that owns the session's folder, or the
// free workspace with that folder as work path.
func adoptHome(deps Deps, cwd string) (wsID, work string) {
	cwd = filepath.Clean(cwd)
	list, err := deps.Store.ListWorkspaces()
	if err == nil {
		for _, wk := range list {
			if wk.ID == store.FreeWorkspaceID {
				continue
			}
			if filepath.Clean(wk.Path) == cwd {
				return wk.ID, ""
			}
		}
	}
	return store.FreeWorkspaceID, cwd
}

// handleCLICleanupSetting is the pi auto-clean preference: 0 keeps
// everything (default), N deletes orphan sessions untouched for N days.
// PUT applies and sweeps immediately.
func handleCLICleanupSetting(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("cli") != "pi" {
			writeErr(w, http.StatusBadRequest, "auto-clean is a pi setting")
			return
		}
		if r.Method == http.MethodPut {
			var req struct {
				Days int `json:"days"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Days < 0 || req.Days > 3650 {
				writeErr(w, http.StatusBadRequest, "days must be 0..3650 (0 = keep everything)")
				return
			}
			if err := deps.Store.SetSetting(settingCleanupDays, strconv.Itoa(req.Days)); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			removed := sweepOrphanSessions(deps, req.Days)
			writeJSON(w, http.StatusOK, map[string]any{"days": req.Days, "removed": removed})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"days": cleanupDaysSetting(deps)})
	}
}

// cliSessionView adds the PiCode workspace context (if any) to a session.
type cliSessionView struct {
	clisession.Summary
	WorkspaceID string        `json:"workspaceId,omitempty"`
	Workspace   string        `json:"workspace,omitempty"`
	InUseBy     *sessionUse   `json:"inUseBy,omitempty"`
	Handoff     *handoffLinks `json:"handoff,omitempty"` // lineage (ADR-0087)
}
