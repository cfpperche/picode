package server

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// cleanupPreview is the opt-in purge offer shown before delete. Terminals is
// filled only for workspace previews (ADR-0026): removing a workspace kills
// its terminals, and the dialog says so; an agent preview leaves it zero.
type cleanupPreview struct {
	Cwd          string `json:"cwd"`
	LastOccupant bool   `json:"lastOccupant"`
	Sessions     int    `json:"sessions"`
	SessionBytes int64  `json:"sessionBytes"`
	CanPurgeWork bool   `json:"canPurgeWork"`
	WorkPath     string `json:"workPath,omitempty"`
	Terminals    int    `json:"terminals,omitempty"`

	// dyingAgents is every agent id being removed (ADR-0040) — server-side
	// only (unexported, encoding/json skips it), used by applyCleanup to
	// also purge each one's private session dir. Sessions/SessionBytes
	// above already include their contents.
	dyingAgents []string
}

func queryFlag(r *http.Request, key string) bool {
	v := strings.TrimSpace(strings.ToLower(r.URL.Query().Get(key)))
	return v == "1" || v == "true" || v == "yes"
}

func canonDir(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return filepath.Clean(p)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func sameDir(a, b string) bool {
	ca, cb := canonDir(a), canonDir(b)
	if ca == "" || cb == "" {
		return false
	}
	return ca == cb
}

func workRoot(dataDir string) string {
	if strings.TrimSpace(dataDir) != "" {
		return filepath.Join(dataDir, "work")
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, ".picode", "work")
}

func ownedByWork(root, cwd string) bool {
	root, cwd = canonDir(root), canonDir(cwd)
	if root == "" || cwd == "" || cwd == root {
		return false
	}
	rel, err := filepath.Rel(root, cwd)
	if err != nil {
		return false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

func (deps Deps) agentCwd(agent store.Agent) (store.Workspace, string, error) {
	wk, err := deps.Store.GetWorkspace(agent.WorkspaceID)
	if err != nil {
		return store.Workspace{}, "", err
	}
	return wk, store.AgentCwd(wk, agent), nil
}

func (deps Deps) occupantsOf(cwd string) ([]store.Agent, error) {
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return nil, err
	}
	wsCache := map[string]store.Workspace{}
	var out []store.Agent
	for _, a := range agents {
		wk, ok := wsCache[a.WorkspaceID]
		if !ok {
			wk, err = deps.Store.GetWorkspace(a.WorkspaceID)
			if err != nil {
				return nil, err
			}
			wsCache[a.WorkspaceID] = wk
		}
		if sameDir(store.AgentCwd(wk, a), cwd) {
			out = append(out, a)
		}
	}
	return out, nil
}

func (deps Deps) isWorkspacePath(cwd string) bool {
	list, err := deps.Store.ListWorkspaces()
	if err != nil {
		return false
	}
	for _, w := range list {
		if sameDir(w.Path, cwd) {
			return true
		}
	}
	return false
}

func (deps Deps) previewCleanup(cwd string, dying map[string]bool) cleanupPreview {
	p := cleanupPreview{Cwd: cwd}
	if cwd == "" {
		return p
	}
	occupants, err := deps.occupantsOf(cwd)
	if err != nil {
		return p
	}
	p.LastOccupant = true
	for _, a := range occupants {
		if !dying[a.ID] {
			p.LastOccupant = false
			break
		}
	}
	st := session.DirStats(cwd)
	p.Sessions = st.Count
	p.SessionBytes = st.Bytes
	for id := range dying {
		// ADR-0040: each dying agent's own private session dir — counted
		// here regardless of LastOccupant, since it's never shared with
		// another agent the way the cwd bucket can be.
		p.dyingAgents = append(p.dyingAgents, id)
		ast := session.DirStatsAt(session.AgentDir(id))
		p.Sessions += ast.Count
		p.SessionBytes += ast.Bytes
	}
	if p.LastOccupant && ownedByWork(workRoot(deps.DataDir), cwd) && !deps.isWorkspacePath(cwd) {
		p.CanPurgeWork = true
		p.WorkPath = cwd
	}
	return p
}

func (deps Deps) applyCleanup(p cleanupPreview, purgeSessions, purgeWork bool) {
	if purgeSessions {
		for _, id := range p.dyingAgents {
			_ = session.RemoveAgentDir(id) // ADR-0040: always this agent's own
		}
	}
	if !p.LastOccupant {
		return
	}
	if purgeSessions && p.Sessions > 0 {
		_ = session.RemoveDir(p.Cwd)
	}
	if purgeWork && p.CanPurgeWork && p.WorkPath != "" {
		_ = os.RemoveAll(p.WorkPath)
	}
}

func (deps Deps) stopAgent(ctx context.Context, id string) {
	// Destructive removal still honours the one-writer rule: the mutation
	// guard blocks a concurrent reply send while the pane is killed.
	release := deps.Replies.Controls.BeginMutation(id)
	defer release()
	if deps.Runtime != nil {
		deps.Runtime.Stop(id)
	}
	if deps.Tmux != nil && deps.Tmux.Available() {
		_ = deps.Tmux.KillSession(ctx, tmux.SessionName(id))
		_ = deps.Tmux.KillSession(ctx, tmux.ShellSessionName(id))
	}
}

func handleAgentCleanup(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		agent, err := deps.Store.GetAgent(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, cwd, err := deps.agentCwd(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, deps.previewCleanup(cwd, map[string]bool{agent.ID: true}))
	}
}

func handleWorkspaceCleanup(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == store.FreeWorkspaceID {
			writeErr(w, http.StatusBadRequest, "cannot remove the free workspace")
			return
		}
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agents, err := deps.Store.ListAgents(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		dying := map[string]bool{}
		for _, a := range agents {
			dying[a.ID] = true
		}
		terms, err := deps.Store.ListWorkspaceTerminals(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		p := deps.previewCleanup(wk.Path, dying)
		p.Terminals = len(terms)
		writeJSON(w, http.StatusOK, p)
	}
}
