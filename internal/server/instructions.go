package server

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/cliinstructions"
	"github.com/cfpperche/picode/internal/clipkgs"
)

// The Instructions tab (docs/architecture/cli-instructions.md): which
// instruction files — AGENTS.md, CLAUDE.md and their kin — each agent CLI
// reads for a session started in this workspace, and why it leaves the
// others out. Read from disk on every request and stored nowhere; the answer
// carries paths, sizes and verdicts, never a file's text.
func registerInstructionsRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/workspaces/{id}/instructions", handleWorkspaceInstructions(deps))
}

func handleWorkspaceInstructions(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		if st, err := os.Stat(cwd); err != nil || !st.IsDir() {
			writeErr(w, http.StatusNotFound, "This workspace's folder is gone.")
			return
		}
		start := cwd
		if rel := r.URL.Query().Get("start"); rel != "" {
			clean := filepath.Clean(filepath.FromSlash(rel))
			if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				writeErr(w, http.StatusBadRequest, "Choose a folder inside this workspace.")
				return
			}
			start = filepath.Join(cwd, clean)
			if st, err := os.Stat(start); err != nil || !st.IsDir() {
				writeErr(w, http.StatusNotFound, "That folder is gone.")
				return
			}
		}
		rep, err := cliinstructions.Resolve(cliinstructions.Env{Root: cwd, Start: start, Installed: clipkgs.Installed})
		if errors.Is(err, os.ErrPermission) {
			writeErr(w, http.StatusBadRequest, "Choose a folder inside this workspace.")
			return
		}
		if err != nil {
			writeErr(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rep)
	}
}
