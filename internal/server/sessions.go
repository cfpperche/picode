package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func handleListSessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		all, err := session.ListDirs(session.Dir(store.AgentCwd(wk, agent)), session.AgentDir(agent.ID))
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		owned, err := deps.Store.AgentSessionKeys(agent.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cur := ""
		if agent.SessionPath != nil {
			cur = *agent.SessionPath
		}
		// ADR-0039: this agent's own history only — not every JSONL pi
		// ever wrote for this cwd (that bucket is shared by every Agent
		// and, for a bare `pi` run by hand, every Terminal too).
		list := make([]session.Summary, 0, len(all))
		haveCurrent := false
		resolvedCurrent := false
		backfilled := ""
		for _, s := range all {
			byPath := owned.Paths[s.Path]
			byID := s.ID != "" && owned.IDs[s.ID]
			if !byPath && !byID {
				continue
			}
			if byID && !byPath {
				// A fresh spawn's pre-minted --session-id (ADR-0039) now
				// has a real file: persist the path so other consumers
				// of agents.session_path (e.g. the manage view's inUseBy
				// guard) don't depend on this endpoint being called
				// again. all is newest-first, so the first unresolved
				// match is this agent's most recent session.
				deps.Store.ResolveAgentSessionID(agent.ID, s.ID, s.Path)
				if agent.SessionPath == nil && !resolvedCurrent {
					_, _ = deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &s.Path})
					resolvedCurrent = true
					backfilled = s.Path
				}
			}
			list = append(list, s)
			if s.Path == cur {
				haveCurrent = true
			}
		}
		// A pointer lost by an older build (a pending id with a file but
		// no session_path) healed in the loop above; surface it as current
		// so the chat reopens the thread. An explicit New seals its
		// pendings, so nothing backfills and current stays "" — the fresh
		// state the user asked for.
		if cur == "" && backfilled != "" {
			cur = backfilled
			haveCurrent = true
		}
		// Safety net: the agent's current session is always visible even
		// if a DB hiccup kept it from being historized. No newest-session
		// fallback when cur is "": an empty pointer is either a brand-new
		// agent (nothing to resurrect) or an explicit POST /sessions/new —
		// the picker must not silently re-select the thread the user just
		// abandoned. Picking a session from the list (or any spawn's
		// adoption, ADR-0053) sets the pointer again.
		if cur != "" && !haveCurrent {
			if s, err := session.Summarize(cur); err == nil {
				list = append(list, s)
				sort.Slice(list, func(i, j int) bool { return list[i].UpdatedAt > list[j].UpdatedAt })
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list, "current": cur})
	}
}

func handleSessionTranscript(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		path := r.URL.Query().Get("path")
		if path == "" && agent.SessionPath != nil {
			path = *agent.SessionPath
		}
		if path == "" {
			writeJSON(w, http.StatusOK, map[string]any{"events": []any{}, "path": "", "total": 0, "remaining": 0})
			return
		}
		if !safeSessionPath(path, session.Dir(store.AgentCwd(wk, agent)), session.AgentDir(agent.ID)) {
			writeErr(w, http.StatusBadRequest, "session is not in this workspace")
			return
		}
		tail := atoiOr(r.URL.Query().Get("tail"), 0)
		skip := atoiOr(r.URL.Query().Get("skip"), 0)
		ev, total, compacted, err := session.TranscriptWindow(path, tail, skip)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if ev == nil {
			ev = []session.Event{}
		}
		remaining := total - skip - len(ev)
		if remaining < 0 {
			remaining = 0
		}
		var bytes int64
		if st, err := os.Stat(path); err == nil {
			bytes = st.Size()
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": ev, "path": path, "total": total, "remaining": remaining, "bytes": bytes, "compacted": compacted})
	}
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func handleNewSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		empty := ""
		if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &empty}); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Close the adoption window before the restart below: an explicit
		// New must mint a fresh --session-id, not re-adopt the thread just
		// abandoned (ADR-0053 adoption heals a lost pointer; it must not
		// override the user asking for a new session).
		deps.Store.SealPendingAgentSessions(agent.ID)
		mode := deps.runMode(r, agent.ID)
		if err := restartSameMode(r.Context(), deps, wk, agent.ID, mode); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "current": ""})
	}
}

func handleResumeSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			Path string `json:"path"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
			writeErr(w, http.StatusBadRequest, "path required")
			return
		}
		if !safeSessionPath(req.Path, session.Dir(store.AgentCwd(wk, agent)), session.AgentDir(agent.ID)) {
			writeErr(w, http.StatusBadRequest, "session is not in this workspace")
			return
		}
		if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &req.Path}); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		mode := deps.runMode(r, agent.ID)
		if err := restartSameMode(r.Context(), deps, wk, agent.ID, mode); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "current": req.Path})
	}
}

func handleRenameSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r)
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			Path string `json:"path"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Name) == "" {
			writeErr(w, http.StatusBadRequest, "name required")
			return
		}
		path := req.Path
		if path == "" && agent.SessionPath != nil {
			path = *agent.SessionPath
		}
		if path == "" || !safeSessionPath(path, session.Dir(store.AgentCwd(wk, agent)), session.AgentDir(agent.ID)) {
			writeErr(w, http.StatusBadRequest, "session is not in this workspace")
			return
		}
		if err := session.SetName(path, req.Name); err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if ma := deps.Runtime.Get(agent.ID); ma != nil && agent.SessionPath != nil && *agent.SessionPath == path {
			_ = ma.SetSessionName(r.Context(), req.Name)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "name": strings.TrimSpace(req.Name)})
	}
}

func loadWS(deps Deps, r *http.Request) (store.Workspace, store.Agent, error) {
	id := r.PathValue("id")
	wk, err := deps.Store.GetWorkspace(id)
	if err != nil {
		return store.Workspace{}, store.Agent{}, err
	}
	if aid := strings.TrimSpace(r.URL.Query().Get("agent")); aid != "" {
		a, err := deps.Store.GetAgent(aid)
		if err != nil {
			return store.Workspace{}, store.Agent{}, err
		}
		if a.WorkspaceID != wk.ID {
			return store.Workspace{}, store.Agent{}, store.ErrNotFound
		}
		return wk, a, nil
	}
	agent, err := deps.Store.DefaultAgent(wk.ID)
	if errors.Is(err, store.ErrNotFound) {
		// The workspace exists — it just has nobody in it (ADR-0027).
		return wk, store.Agent{}, errNoAgents
	}
	return wk, agent, err
}

// errNoAgents marks a workspace-scoped request that needs an agent hitting
// an empty workspace (ADR-0027) — a 409, not the misleading 404.
var errNoAgents = errors.New("workspace has no agents")

func writeStoreErr(w http.ResponseWriter, err error) {
	if errors.Is(err, errNoAgents) {
		writeErr(w, http.StatusConflict, "workspace has no agents — add one first")
		return
	}
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

// safeSessionPath reports whether path is a JSONL file directly inside one
// of dirs — each an already-resolved session directory (session.Dir(cwd)
// and/or session.AgentDir(agentID), ADR-0040), never a bare cwd.
func safeSessionPath(path string, dirs ...string) bool {
	abs, err := filepath.Abs(path)
	if err != nil || !strings.HasSuffix(abs, ".jsonl") {
		return false
	}
	for _, dir := range dirs {
		if dir != "" && strings.HasPrefix(abs, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func restartSameMode(ctx context.Context, deps Deps, wk store.Workspace, agentID string, mode agentRunMode) error {
	// The agent's own cwd, never the workspace's: a free agent's
	// workspace is the ws_free sentinel (no real path), and a WorkPath
	// override must win for workspace agents too. This is the same
	// resolution every Runtime.Start caller uses (agents.go, session_ops.go).
	ag, err := deps.Store.GetAgent(agentID)
	if err != nil {
		return err
	}
	cwd := store.AgentCwd(wk, ag)
	_ = os.MkdirAll(cwd, 0o755)
	switch mode {
	case modeManaged:
		deps.Runtime.Stop(agentID)
		return deps.Runtime.Start(agentID, cwd)
	case modeInteractive:
		_ = deps.Tmux.KillSession(ctx, tmux.SessionName(agentID))
		return deps.Tmux.NewSessionEnv(ctx, tmux.SessionName(agentID), cwd, ag.SpawnEnv(), deps.AgentCmd, deps.spawnFlags(ag)...)
	default:
		return nil
	}
}

// spawnFlags is agent.CLIFlags() plus, for a fresh start (no current
// SessionPath), a freshly minted --session-id (ADR-0039) — so a pi
// process spawned interactively in tmux is attributable to this agent
// from the moment it creates its session file, the same as managed mode
// (rpc.Runtime.Start). Before minting, an earlier run's pending session
// is resolved to its file when one exists (ADR-0053): the TUI resumes
// the chat's thread instead of opening a competing empty one.
func (deps Deps) spawnFlags(agent store.Agent) []string {
	if agent.SessionPath == nil || strings.TrimSpace(*agent.SessionPath) == "" {
		if p := deps.Store.ResolvePendingAgentSession(agent.ID); p != "" {
			agent.SessionPath = &p
		}
	}
	if agent.SessionPath != nil && strings.TrimSpace(*agent.SessionPath) != "" {
		return agent.CLIFlags()
	}
	return agent.CLIFlagsForSpawn(deps.Store.NewPendingAgentSession(agent.ID))
}
