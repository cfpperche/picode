package server

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerAgentShellRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/shells", handleListShells(deps))
	mux.HandleFunc("POST /api/agents/{id}/shells", handleCreateShell(deps))
	mux.HandleFunc("DELETE /api/agents/{id}/shells", handleKillShell(deps))
}

func defaultShell() string {
	if s := strings.TrimSpace(os.Getenv("SHELL")); s != "" {
		return s
	}
	return "/bin/sh"
}

func handleListShells(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAgent(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		shells := []map[string]any{}
		if deps.Tmux != nil && deps.Tmux.Available() {
			name := tmux.ShellSessionName(id)
			has, err := deps.Tmux.HasSession(r.Context(), name)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
			if has {
				shells = append(shells, map[string]any{"session": name})
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{"shells": shells})
	}
}

func handleCreateShell(deps Deps) http.HandlerFunc {
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
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to open a terminal.")
			return
		}
		_, cwd, err := deps.agentHome(agent)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := tmux.ShellSessionName(id)
		has, err := deps.Tmux.HasSession(r.Context(), name)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if !has {
			if err := deps.Tmux.NewSession(r.Context(), name, cwd, defaultShell()); err != nil {
				writeErr(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
		_ = deps.Tmux.SetOption(r.Context(), name, "status", "off")
		status := http.StatusCreated
		body := map[string]any{"session": name, "cwd": cwd}
		if has {
			status = http.StatusOK
			body["already"] = true
		}
		writeJSON(w, status, body)
	}
}

func handleKillShell(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetAgent(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "agent not found")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.Tmux == nil || !deps.Tmux.Available() {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if err := deps.Tmux.KillSession(r.Context(), tmux.ShellSessionName(id)); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
