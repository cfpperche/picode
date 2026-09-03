package server

import (
	"encoding/json"
	"net/http"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/osopen"
)

// The working-tree diff expands a change dot into its patch (ADR-0032),
// and reveal opens the owner's folder in the host file manager. Both are
// owner-scoped like gitstatus: the server resolves the folder, never the URL.
func registerWorkDiffRoutes(mux Registrar, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/gitdiff", handleAgentWorkDiff(deps))
	mux.HandleFunc("GET /api/terminals/{id}/gitdiff", handleTerminalWorkDiff(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/gitdiff", handleWorkspaceWorkDiff(deps))
	mux.HandleFunc("POST /api/agents/{id}/reveal", handleAgentReveal(deps))
	mux.HandleFunc("POST /api/terminals/{id}/reveal", handleTerminalReveal(deps))
	mux.HandleFunc("POST /api/workspaces/{id}/reveal", handleWorkspaceReveal(deps))
}

// revealFn is swapped in tests — a headless CI cannot open a file manager.
var revealFn = osopen.Reveal

func handleAgentWorkDiff(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeWorkDiff(w, r, cwd)
	}
}

func handleTerminalWorkDiff(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeWorkDiff(w, r, liveTermCwd(deps, r, term))
	}
}

func handleWorkspaceWorkDiff(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		writeWorkDiff(w, r, cwd)
	}
}

func writeWorkDiff(w http.ResponseWriter, r *http.Request, cwd string) {
	rel := r.URL.Query().Get("path")
	if rel == "" {
		writeErr(w, http.StatusBadRequest, "pass ?path=<file>")
		return
	}
	_, outRel, err := relUnderCwd(cwd, rel)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	f, truncated := gitgraph.WorkingDiff(canonDir(cwd), outRel)
	if f == nil {
		writeErr(w, http.StatusNotFound, "no difference for this path")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path": f.Path, "oldPath": f.OldPath, "binary": f.Binary,
		"patch": f.Patch, "truncated": truncated,
	})
}

func handleAgentReveal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeReveal(w, r, cwd)
	}
}

func handleTerminalReveal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeReveal(w, r, liveTermCwd(deps, r, term))
	}
}

func handleWorkspaceReveal(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		writeReveal(w, r, cwd)
	}
}

func writeReveal(w http.ResponseWriter, r *http.Request, cwd string) {
	var req struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	abs := cwd
	if req.Path != "" {
		var err error
		abs, _, err = relUnderCwd(cwd, req.Path)
		if err != nil {
			writeErr(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err := revealFn(abs); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
