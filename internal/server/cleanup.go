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

// cleanupPreview is the opt-in purge offer shown before delete.
type cleanupPreview struct {
	Cwd          string `json:"cwd"`
	LastOccupant bool   `json:"lastOccupant"`
	Sessions     int    `json:"sessions"`
	SessionBytes int64  `json:"sessionBytes"`
	CanPurgeWork bool   `json:"canPurgeWork"`
	WorkPath     string `json:"workPath,omitempty"`
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
	if p.LastOccupant && ownedByWork(workRoot(deps.DataDir), cwd) && !deps.isWorkspacePath(cwd) {
		p.CanPurgeWork = true
		p.WorkPath = cwd
	}
	return p
}

func (deps Deps) applyCleanup(p cleanupPreview, purgeSessions, purgeWork bool) {
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
		writeJSON(w, http.StatusOK, deps.previewCleanup(wk.Path, dying))
	}
}
