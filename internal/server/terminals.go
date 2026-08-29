package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerTerminalRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/terminals", handleListTerminals(deps))
	mux.HandleFunc("POST /api/terminals", handleCreateTerminal(deps))
	mux.HandleFunc("DELETE /api/terminals/{id}", handleDeleteTerminal(deps))
	mux.HandleFunc("PATCH /api/terminals/{id}", handleRenameTerminal(deps))
	mux.HandleFunc("POST /api/terminals/{id}/open", handleOpenTerminal(deps))
}

func defaultShell() string {
	if s := strings.TrimSpace(os.Getenv("SHELL")); s != "" {
		return s
	}
	return "/bin/sh"
}

func termView(t store.Terminal, session string, live bool) map[string]any {
	return map[string]any{
		"id":        t.ID,
		"name":      t.Name,
		"cwd":       t.Cwd,
		"createdAt": t.CreatedAt,
		"session":   session,
		"running":   live,
	}
}

func handleListTerminals(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := deps.Store.ListTerminals()
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		out := make([]map[string]any, 0, len(list))
		for _, t := range list {
			name := tmux.ShellSessionName(t.ID)
			live := false
			if deps.Tmux != nil && deps.Tmux.Available() {
				live, _ = deps.Tmux.HasSession(r.Context(), name)
			}
			out = append(out, termView(t, name, live))
		}
		writeJSON(w, http.StatusOK, map[string]any{"terminals": out})
	}
}

func handleCreateTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.Tmux == nil || !deps.Tmux.Available() {
			writeErr(w, http.StatusServiceUnavailable, "Need tmux to open a terminal.")
			return
		}
		var req struct {
			Name string `json:"name"`
			Cwd  string `json:"cwd"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		}
		t, err := deps.Store.CreateTerminal(req.Name, req.Cwd)
		if err != nil {
			if strings.Contains(err.Error(), "doesn't exist") {
				writeErr(w, http.StatusBadRequest, err.Error())
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := tmux.ShellSessionName(t.ID)
		if err := ensureShell(deps, r, name, t.Cwd); err != nil {
			_ = deps.Store.DeleteTerminal(t.ID)
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, termView(t, name, true))
	}
}

func handleOpenTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := deps.Store.GetTerminal(r.PathValue("id"))
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
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
		name := tmux.ShellSessionName(t.ID)
		if err := ensureShell(deps, r, name, t.Cwd); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, termView(t, name, true))
	}
}

func handleRenameTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		t, err := deps.Store.RenameTerminal(r.PathValue("id"), req.Name)
		if errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
			return
		}
		if err != nil {
			if strings.Contains(err.Error(), "name is required") {
				writeErr(w, http.StatusBadRequest, "Give it a name.")
				return
			}
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		name := tmux.ShellSessionName(t.ID)
		live := false
		if deps.Tmux != nil && deps.Tmux.Available() {
			live, _ = deps.Tmux.HasSession(r.Context(), name)
		}
		writeJSON(w, http.StatusOK, termView(t, name, live))
	}
}

func handleDeleteTerminal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); errors.Is(err, store.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "That terminal is gone.")
			return
		} else if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		if deps.Tmux != nil && deps.Tmux.Available() {
			_ = deps.Tmux.KillSession(r.Context(), tmux.ShellSessionName(id))
		}
		if err := deps.Store.DeleteTerminal(id); err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func ensureShell(deps Deps, r *http.Request, name, cwd string) error {
	if !deps.Tmux.Available() {
		return errors.New("Need tmux to open a terminal.")
	}
	has, err := deps.Tmux.HasSession(r.Context(), name)
	if err != nil {
		return err
	}
	if !has {
		if err := deps.Tmux.NewSession(r.Context(), name, cwd, defaultShell()); err != nil {
			return err
		}
	}
	_ = deps.Tmux.SetOption(r.Context(), name, "status", "off")
	_ = deps.Tmux.SetEnv(r.Context(), name, "TERM", "xterm-256color")
	_ = deps.Tmux.SetEnv(r.Context(), name, "COLORTERM", "truecolor")
	return nil
}
