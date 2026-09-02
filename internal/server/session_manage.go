package server

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/feed"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// Workspace session management: one view for every Pi session under the
// workspace folder (A) plus Claude-style age-based cleanup of orphans (B).
// Orphan = not the current session of any agent.

const settingCleanupDays = "cleanup_orphan_days"

type sessionUse struct {
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
}

type sessionManageItem struct {
	session.Summary
	InUseBy *sessionUse `json:"inUseBy"`
}

type sessionManageView struct {
	Sessions    []sessionManageItem `json:"sessions"`
	CleanupDays int                 `json:"cleanupDays"`
	TotalBytes  int64               `json:"totalBytes"`
}

// allSessionItem adds the folder context for the machine-wide view.
type allSessionItem struct {
	session.Summary
	InUseBy     *sessionUse `json:"inUseBy"`
	WorkspaceID string      `json:"workspaceId,omitempty"` // set when this cwd is a PiCode workspace
	Workspace   string      `json:"workspace,omitempty"`
}

// workspaceSessionDirs is every directory pi may have written this
// workspace's sessions into (ADR-0040): the shared cwd bucket (sessions
// from before this ADR, plus anything a Terminal or bare `pi` writes
// there) and each of the workspace's agents' own private dirs.
func workspaceSessionDirs(deps Deps, wk store.Workspace) []string {
	dirs := []string{session.Dir(wk.Path)}
	agents, err := deps.Store.ListAgents(wk.ID)
	if err != nil {
		return dirs
	}
	for _, a := range agents {
		dirs = append(dirs, session.AgentDir(a.ID))
	}
	return dirs
}

// sessionUseBy maps every agent's current session path to the agent.
func sessionUseBy(deps Deps) map[string]sessionUse {
	out := map[string]sessionUse{}
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return out
	}
	for _, a := range agents {
		if a.SessionPath == nil || strings.TrimSpace(*a.SessionPath) == "" {
			continue
		}
		if _, taken := out[*a.SessionPath]; taken {
			continue
		}
		out[*a.SessionPath] = sessionUse{AgentID: a.ID, AgentName: a.Name}
	}
	return out
}

func handleManageSessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, err := deps.Store.GetWorkspace(r.PathValue("id"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		var list []session.Summary
		for _, dir := range workspaceSessionDirs(deps, wk) {
			part, err := session.ListDir(dir)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			list = append(list, part...)
		}
		sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt > list[j].UpdatedAt })
		use := sessionUseBy(deps)
		var total int64
		items := make([]sessionManageItem, 0, len(list))
		for _, s := range list {
			it := sessionManageItem{Summary: s}
			if u, ok := use[s.Path]; ok {
				it.InUseBy = &u
			}
			total += s.Size
			items = append(items, it)
		}
		writeJSON(w, http.StatusOK, sessionManageView{
			Sessions:    items,
			CleanupDays: cleanupDaysSetting(deps),
			TotalBytes:  total,
		})
	}
}

// handleAllSessions is the machine-wide view: every Pi session on this
// machine, each tagged with the workspace it belongs to (if any).
func handleAllSessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := session.ListAll()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		use := sessionUseBy(deps)
		wsByDir := map[string]struct{ id, name string }{}
		if wss, err := deps.Store.ListWorkspaces(); err == nil {
			for _, wk := range wss {
				wsByDir[canonDir(wk.Path)] = struct{ id, name string }{wk.ID, wk.Name}
			}
		}
		var total int64
		items := make([]allSessionItem, 0, len(list))
		for _, s := range list {
			it := allSessionItem{Summary: s}
			if u, ok := use[s.Path]; ok {
				it.InUseBy = &u
			}
			if ws, ok := wsByDir[canonDir(s.Cwd)]; ok {
				it.WorkspaceID = ws.id
				it.Workspace = ws.name
			}
			total += s.Size
			items = append(items, it)
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"sessions":    items,
			"cleanupDays": cleanupDaysSetting(deps),
			"totalBytes":  total,
		})
	}
}

// handleDeleteAnySession removes one orphan session from anywhere under the
// machine's pi sessions root (same in-use rule as the workspace delete).
func handleDeleteAnySession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		_ = deps.Store.AppendEvent("session_deleted", nil, nil, map[string]any{"path": path, "scope": "machine"})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

func handleDeleteManagedSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, err := deps.Store.GetWorkspace(r.PathValue("id"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
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
		if err != nil || !safeSessionPath(path, workspaceSessionDirs(deps, wk)...) {
			writeErr(w, http.StatusBadRequest, "session is not in this workspace")
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
		_ = deps.Store.AppendEvent("session_deleted", nil, &wk.ID, map[string]any{"path": path})
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}
}

// handleCleanupSetting is the auto-clean preference: 0 keeps everything
// (default), N deletes orphan sessions untouched for N days.
func handleCleanupSetting(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

func cleanupDaysSetting(deps Deps) int {
	v, ok, err := deps.Store.GetSetting(settingCleanupDays)
	if err != nil || !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// StartSessionSweep runs the orphan sweep at boot and once a day after.
// Noop while the preference is 0.
func StartSessionSweep(deps Deps) {
	sweep := func() {
		if days := cleanupDaysSetting(deps); days > 0 {
			sweepOrphanSessions(deps, days)
		}
		// Change-log retention (ADR-0048): a cursor older than this gets
		// a reset instead of a replay.
		if deps.Store != nil {
			_, _ = deps.Store.PruneEvents(time.Now().Add(-feed.DefaultRetention))
		}
	}
	go func() {
		time.Sleep(5 * time.Second) // let the server settle first
		sweep()
		t := time.NewTicker(24 * time.Hour)
		defer t.Stop()
		for range t.C {
			sweep()
		}
	}()
}

// sweepOrphanSessions deletes session files under every known agent cwd
// that (a) no agent is bound to and (b) have not been touched in days.
func sweepOrphanSessions(deps Deps, days int) int {
	if days <= 0 {
		return 0
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	use := sessionUseBy(deps)
	// ADR-0039 already surfaces an agent's older, non-current sessions as
	// resumable in its own chat picker — the age sweep must not delete one
	// out from under that picker just because it isn't the *current*
	// pointer `use` tracks.
	owned, _ := deps.Store.AllAgentSessionPaths()

	dirs := map[string]bool{}
	if wss, err := deps.Store.ListWorkspaces(); err == nil {
		for _, wk := range wss {
			dirs[session.Dir(wk.Path)] = true
		}
	}
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			if p := a.WorkPath; p != nil && strings.TrimSpace(*p) != "" {
				dirs[session.Dir(*p)] = true
			}
			dirs[session.AgentDir(a.ID)] = true // ADR-0040
		}
	}

	removed := 0
	for dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if _, inUse := use[p]; inUse {
				continue
			}
			if owned[p] {
				continue
			}
			fi, err := e.Info()
			if err != nil || fi.ModTime().After(cutoff) {
				continue
			}
			if err := os.Remove(p); err == nil {
				removed++
				log.Printf("session sweep: removed orphan %s (untouched since %s)", p, fi.ModTime().UTC().Format(time.RFC3339))
			}
		}
	}
	if removed > 0 {
		_ = deps.Store.AppendEvent("session_sweep", nil, nil, map[string]any{"removed": removed, "days": days})
	}
	return removed
}
