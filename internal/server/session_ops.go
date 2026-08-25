package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

func registerSessionOps(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/tree", handleAgentTree(deps))
	mux.HandleFunc("POST /api/agents/{id}/fork", handleAgentFork(deps))
	mux.HandleFunc("POST /api/agents/{id}/clone", handleAgentClone(deps))
}

func handleAgentTree(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent, err := deps.Store.GetAgent(r.PathValue("id"))
		if err != nil {
			writeErr(w, statusForStore(err), err.Error())
			return
		}
		path := ""
		if agent.SessionPath != nil {
			path = *agent.SessionPath
		}
		if path == "" {
			writeJSON(w, http.StatusOK, session.Tree{Tree: []session.TreeNode{}})
			return
		}
		tr, err := session.ReadTree(path)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tr)
	}
}

func handleAgentFork(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			EntryID string `json:"entryId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.EntryID == "" {
			writeErr(w, http.StatusBadRequest, "entryId required")
			return
		}
		ma, err := ensureChat(r, deps, r.PathValue("id"))
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		res, err := ma.Fork(ctx, req.EntryID)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSessionOp(w, deps, r.PathValue("id"), ma, res)
	}
}

func handleAgentClone(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ma, err := ensureChat(r, deps, r.PathValue("id"))
		if err != nil {
			writeErr(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		res, err := ma.Clone(ctx)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeSessionOp(w, deps, r.PathValue("id"), ma, res)
	}
}

func ensureChat(r *http.Request, deps Deps, id string) (*rpc.ManagedAgent, error) {
	agent, err := deps.Store.GetAgent(id)
	if err != nil {
		return nil, err
	}
	_, cwd, err := deps.agentHome(agent)
	if err != nil {
		return nil, err
	}
	if deps.runMode(r, id) == modeInteractive {
		return nil, errors.New("switch to chat (not terminal) to fork or clone")
	}
	if deps.Runtime.Get(id) == nil {
		if err := deps.Runtime.Start(id, cwd); err != nil {
			return nil, err
		}
		_ = deps.Store.SetAgentRuntime(id, store.StatusRunning)
		select {
		case <-time.After(600 * time.Millisecond):
		case <-r.Context().Done():
			return nil, r.Context().Err()
		}
	}
	ma := deps.Runtime.Get(id)
	if ma == nil {
		return nil, errors.New("agent is not running in chat")
	}
	return ma, nil
}

func writeSessionOp(w http.ResponseWriter, deps Deps, agentID string, ma *rpc.ManagedAgent, res rpc.Response) {
	var payload struct {
		Cancelled bool   `json:"cancelled"`
		Text      string `json:"text"`
	}
	_ = json.Unmarshal(res.Data, &payload)
	out := map[string]any{"ok": true, "cancelled": payload.Cancelled, "text": payload.Text}
	if !payload.Cancelled {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		st, err := ma.GetState(ctx)
		if err == nil {
			var state struct {
				SessionFile string `json:"sessionFile"`
			}
			_ = json.Unmarshal(st.Data, &state)
			if state.SessionFile != "" {
				_, _ = deps.Store.UpdateAgent(agentID, store.AgentPatch{SessionPath: &state.SessionFile})
				out["sessionPath"] = state.SessionFile
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}
