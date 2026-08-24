package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func handleListSessions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		list, err := session.List(wk.Path)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		cur := ""
		if agent.SessionPath != nil {
			cur = *agent.SessionPath
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": list, "current": cur})
	}
}

func handleSessionTranscript(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		path := r.URL.Query().Get("path")
		if path == "" && agent.SessionPath != nil {
			path = *agent.SessionPath
		}
		if path == "" {
			writeJSON(w, http.StatusOK, map[string]any{"events": []any{}, "path": ""})
			return
		}
		if !safeSessionPath(wk.Path, path) {
			writeErr(w, http.StatusBadRequest, "session is not in this workspace")
			return
		}
		ev, err := session.Transcript(path)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if ev == nil {
			ev = []session.Event{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"events": ev, "path": path})
	}
}

func handleNewSession(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wk, agent, err := loadWS(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		empty := ""
		if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &empty}); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
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
		wk, agent, err := loadWS(deps, r.PathValue("id"))
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
		if !safeSessionPath(wk.Path, req.Path) {
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

func loadWS(deps Deps, id string) (store.Workspace, store.Agent, error) {
	wk, err := deps.Store.GetWorkspace(id)
	if err != nil {
		return store.Workspace{}, store.Agent{}, err
	}
	agent, err := deps.Store.DefaultAgent(wk.ID)
	return wk, agent, err
}

func writeStoreErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "workspace not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, err.Error())
}

func safeSessionPath(cwd, path string) bool {
	dir := session.Dir(cwd)
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	return strings.HasPrefix(abs, dir+string(filepath.Separator)) && strings.HasSuffix(abs, ".jsonl")
}

func restartSameMode(ctx context.Context, deps Deps, wk store.Workspace, agentID string, mode agentRunMode) error {
	switch mode {
	case modeManaged:
		deps.Runtime.Stop(agentID)
		return deps.Runtime.Start(agentID, wk.Path)
	case modeInteractive:
		_ = deps.Tmux.KillSession(ctx, tmux.SessionName(agentID))
		agent, err := deps.Store.GetAgent(agentID)
		if err != nil {
			return err
		}
		return deps.Tmux.NewSession(ctx, tmux.SessionName(agentID), wk.Path, deps.AgentCmd, agent.CLIFlags()...)
	default:
		return nil
	}
}
