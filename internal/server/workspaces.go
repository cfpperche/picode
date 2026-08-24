package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// workspaceView is a workspace plus its default agent (v1 invariant).
type workspaceView struct {
	store.Workspace
	Agent agentView `json:"agent"`
}

type agentView struct {
	store.Agent
	Running bool   `json:"running"`
	Mode    string `json:"mode"` // stopped | interactive | managed (ADR-0006)
}

func asAgentView(a store.Agent, running bool) agentView {
	return agentView{Agent: a, Running: running}
}

func registerWorkspaceRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/workspaces", handleList(deps))
	mux.HandleFunc("POST /api/workspaces", handleAdd(deps))
	mux.HandleFunc("DELETE /api/workspaces/{id}", handleRemove(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/open", handleOpen(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/close", handleClose(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/sessions", handleListSessions(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/sessions/new", handleNewSession(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/sessions/resume", handleResumeSession(deps))
	mux.HandleFunc("POST /api/agents/{id}/tasks", handleEnqueueTask(deps))
	mux.HandleFunc("GET /api/agents/{id}/tasks", handleListTasks(deps))
	registerAgentRoutes(mux, deps)
}

func (deps Deps) view(r *http.Request, w store.Workspace) (workspaceView, error) {
	agent, err := deps.Store.DefaultAgent(w.ID)
	if err != nil {
		return workspaceView{}, err
	}
	mode := deps.runMode(r, agent.ID)
	return workspaceView{
		Workspace: w,
		Agent:     agentView{Agent: agent, Running: mode != modeStopped, Mode: string(mode)},
	}, nil
}

func handleList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := deps.Store.ListWorkspaces()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		views := make([]workspaceView, 0, len(ws))
		for _, wk := range ws {
			v, err := deps.view(r, wk)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			views = append(views, v)
		}
		writeJSON(w, http.StatusOK, views)
	}
}

func handleAdd(deps Deps) http.HandlerFunc {
	var req struct {
		Name     string `json:"name"`
		Path     string `json:"path"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
		Thinking string `json:"thinking"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		wk, agent, err := deps.Store.AddWorkspace(req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		if req.Provider != "" || req.Model != "" || req.Thinking != "" {
			agent, err = deps.Store.UpdateAgent(agent.ID, store.AgentPatch{
				Provider: &req.Provider, Model: &req.Model, Thinking: &req.Thinking,
			})
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		writeJSON(w, http.StatusCreated, workspaceView{Workspace: wk, Agent: agentView{Agent: agent, Running: false, Mode: string(modeStopped)}})
	}
}

func handleRemove(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agent, err := deps.Store.DefaultAgent(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		// Stop the agent first (best effort), then unregister (cascades).
		if deps.Tmux.Available() {
			if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(agent.ID)); err != nil {
				writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
				return
			}
		}
		removed, err := deps.Store.RemoveWorkspace(wk.ID)
		if err != nil || !removed {
			writeErr(w, http.StatusInternalServerError, "remove failed")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleOpen(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agent, err := deps.Store.DefaultAgent(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}

		name := tmux.SessionName(agent.ID)
		// ADR-0006: exclusive run mode — stop managed first.
		deps.Runtime.Stop(agent.ID)
		if has, err := deps.Tmux.HasSession(r.Context(), name); err == nil && has {
			_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusRunning)
			writeJSON(w, http.StatusOK, map[string]any{"running": true, "alreadyRunning": true, "session": name})
			return
		}

		// ADR-0003: spawn the user's pi. Helpful failure when missing.
		if _, err := exec.LookPath(deps.AgentCmd); err != nil {
			writeErr(w, http.StatusServiceUnavailable,
				"pi is not installed or not on PATH — install it with: npm install -g @earendil-works/pi-coding-agent")
			return
		}
		if err := deps.Tmux.NewSession(r.Context(), name, wk.Path, deps.AgentCmd, agent.CLIFlags()...); err != nil {
			_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusStopped)
			writeErr(w, http.StatusInternalServerError, "start agent: "+err.Error())
			return
		}
		_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusRunning)
		_ = deps.Store.AppendEvent("agent_started", &agent.ID, &wk.ID, map[string]string{"session": name})
		writeJSON(w, http.StatusCreated, map[string]any{"running": true, "session": name})
	}
}

func handleClose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, err := deps.Store.GetWorkspace(id)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		agent, err := deps.Store.DefaultAgent(wk.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(agent.ID)); err != nil {
			writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
			return
		}
		_ = deps.Store.SetAgentRuntime(agent.ID, store.StatusStopped)
		_ = deps.Store.AppendEvent("agent_stopped", &agent.ID, &wk.ID, nil)
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
	}
}

// handleEnqueueTask queues a prompt/steer/follow_up for an agent.
// Delivery engine lands with the M2 RPC bridge; until then tasks stay
// `queued` (documented in docs/handoff.md).
func handleEnqueueTask(deps Deps) http.HandlerFunc {
	var req struct {
		Kind    string `json:"kind"`
		Payload string `json:"payload"`
		Source  string `json:"source"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Kind == "" {
			req.Kind = store.TaskPrompt
		}
		task, err := deps.Store.EnqueueTask(agentID, req.Kind, req.Payload, req.Source)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		}
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	}
}

func handleListTasks(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agentID := r.PathValue("id")
		if _, err := deps.Store.GetAgent(agentID); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		tasks, err := deps.Store.ListTasks(agentID, 50)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	}
}
