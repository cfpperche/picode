package server

import (
	"encoding/json"
	"net/http"
	"os/exec"

	"github.com/cfpperche/picode/internal/tmux"
	"github.com/cfpperche/picode/internal/workspace"
)

// workspaceView is a workspace plus live runtime status.
type workspaceView struct {
	workspace.Workspace
	Running bool `json:"running"`
}

func registerWorkspaceRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/workspaces", handleList(deps))
	mux.HandleFunc("POST /api/workspaces", handleAdd(deps))
	mux.HandleFunc("DELETE /api/workspaces/{id}", handleRemove(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/open", handleOpen(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/close", handleClose(deps))
}

func handleList(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ws, err := deps.Registry.List()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		views := make([]workspaceView, 0, len(ws))
		for _, wk := range ws {
			running := false
			if deps.Tmux.Available() {
				if has, err := deps.Tmux.HasSession(r.Context(), tmux.SessionName(wk.ID)); err == nil {
					running = has
				}
			}
			views = append(views, workspaceView{Workspace: wk, Running: running})
		}
		writeJSON(w, http.StatusOK, views)
	}
}

func handleAdd(deps Deps) http.HandlerFunc {
	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		wk, err := deps.Registry.Add(req.Name, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, wk)
	}
}

func handleRemove(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, ok, err := deps.Registry.Get(id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		// Stop the agent first (best effort), then unregister.
		if deps.Tmux.Available() {
			if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(wk.ID)); err != nil {
				writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
				return
			}
		}
		removed, err := deps.Registry.Remove(id)
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
		wk, ok, err := deps.Registry.Get(id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}

		name := tmux.SessionName(wk.ID)
		if has, err := deps.Tmux.HasSession(r.Context(), name); err == nil && has {
			writeJSON(w, http.StatusOK, map[string]any{"running": true, "alreadyRunning": true})
			return
		}

		// ADR-0003: spawn the user's pi. Helpful failure when missing.
		if _, err := exec.LookPath(deps.AgentCmd); err != nil {
			writeErr(w, http.StatusServiceUnavailable,
				"pi is not installed or not on PATH — install it with: npm install -g @earendil-works/pi-coding-agent")
			return
		}
		if err := deps.Tmux.NewSession(r.Context(), name, wk.Path, deps.AgentCmd); err != nil {
			writeErr(w, http.StatusInternalServerError, "start agent: "+err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"running": true, "session": name})
	}
}

func handleClose(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		wk, ok, err := deps.Registry.Get(id)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !ok {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		if err := deps.Tmux.KillSession(r.Context(), tmux.SessionName(wk.ID)); err != nil {
			writeErr(w, http.StatusInternalServerError, "stop agent: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"running": false})
	}
}
