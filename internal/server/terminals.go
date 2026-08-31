package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/cfpperche/picode/internal/gitinfo"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

func registerTerminalRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/terminals", handleListTerminals(deps))
	mux.HandleFunc("POST /api/terminals", handleCreateTerminal(deps))
	mux.HandleFunc("DELETE /api/terminals/{id}", handleDeleteTerminal(deps))
	mux.HandleFunc("PATCH /api/terminals/{id}", handleRenameTerminal(deps))
	mux.HandleFunc("POST /api/terminals/{id}/open", handleOpenTerminal(deps))
	mux.HandleFunc("GET /api/terminals/{id}/text", handleGetTerminalText(deps))
	mux.HandleFunc("PUT /api/terminals/{id}/text", handlePutTerminalText(deps))
	mux.HandleFunc("GET /api/terminals/{id}/blob", handleGetTerminalBlob(deps))
	mux.HandleFunc("GET /api/terminals/{id}/cwd", handleGetTerminalCwd(deps))
	mux.HandleFunc("GET /api/terminals/{id}/browse", handleTerminalBrowse(deps))
}

func defaultShell() string {
	if s := strings.TrimSpace(os.Getenv("SHELL")); s != "" {
		return s
	}
	return "/bin/sh"
}

func termView(t store.Terminal, session string, live bool) map[string]any {
	return map[string]any{
		"id":          t.ID,
		"name":        t.Name,
		"cwd":         t.Cwd,
		"workspaceId": t.WorkspaceID,
		"createdAt":   t.CreatedAt,
		"session":     session,
		"running":     live,
	}
}

// liveTermView is termView with the truth layered on: the pane's live cwd
// and the git facts read from it. EVERY handler that answers with a terminal
// uses this one — the app merges any such response into its list, so a
// response carrying the record cwd would overwrite the live one while the
// stale git survived the merge, pairing one directory's path with another's
// branch on the selected terminal.
func liveTermView(deps Deps, r *http.Request, t store.Terminal, session string, live bool) map[string]any {
	cwd := liveTermCwd(deps, r, t)
	view := termView(t, session, live)
	view["cwd"] = cwd
	view["git"] = gitinfo.Inspect(cwd)
	return view
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
			// The list speaks about where the terminal IS, not where it was
			// born (ADR-0022).
			out = append(out, liveTermView(deps, r, t, name, live))
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
			Name        string `json:"name"`
			Cwd         string `json:"cwd"`
			WorkspaceID string `json:"workspaceId"`
		}
		if r.Body != nil && r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid JSON body")
				return
			}
		}
		t, err := deps.Store.CreateTerminalIn(req.WorkspaceID, req.Name, req.Cwd)
		if err != nil {
			// Covers both store messages on purpose: "that folder doesn't
			// exist" and "that workspace doesn't exist" are user errors.
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
		writeJSON(w, http.StatusCreated, liveTermView(deps, r, t, name, true))
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
		writeJSON(w, http.StatusOK, liveTermView(deps, r, t, name, true))
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
		writeJSON(w, http.StatusOK, liveTermView(deps, r, t, name, live))
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

func liveTermCwd(deps Deps, r *http.Request, term store.Terminal) string {
	if deps.Tmux != nil && deps.Tmux.Available() {
		p, err := deps.Tmux.PaneCwd(r.Context(), tmux.ShellSessionName(term.ID))
		if err == nil && strings.TrimSpace(p) != "" {
			return p
		}
	}
	return term.Cwd
}

func handleGetTerminalCwd(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"cwd": liveTermCwd(deps, r, term)})
	}
}

func handleTerminalBrowse(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		out, err := browseAgentDir(liveTermCwd(deps, r, term), r.URL.Query().Get("dir"))
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handleGetTerminalBlob(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		mime, data, code, err := readAgentBlob(liveTermCwd(deps, r, term), r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "private, max-age=0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func handleGetTerminalText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		out, code, err := readAgentText(liveTermCwd(deps, r, term), r.URL.Query().Get("path"))
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func handlePutTerminalText(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		var req agentTextPut
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if req.Path == "" {
			req.Path = r.URL.Query().Get("path")
		}
		out, code, err := writeAgentText(liveTermCwd(deps, r, term), req.Path, req.Text, req.Mtime)
		if err != nil {
			writeErr(w, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, out)
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
	// Everything PiCode manages — status bar, passthrough, mouse, extended
	// keys — comes from the resolver (ADR-0024): the old forces are its
	// defaults, so a user override wins with no special case. A brand-new
	// terminal has no overrides yet, so this is the global default — which is
	// the point: a default, not a snapshot taken at creation.
	applyScoped(r.Context(), deps, name, termOptionResolver(deps)(name))
	_ = deps.Tmux.SetEnv(r.Context(), name, "TERM", "xterm-256color")
	_ = deps.Tmux.SetEnv(r.Context(), name, "COLORTERM", "truecolor")
	return nil
}
