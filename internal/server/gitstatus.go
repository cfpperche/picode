package server

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/gitgraph"
)

// The gitstatus routes decorate the file tree (ADR-0028) with what changed
// in the owner's repository. Unlike /git, a missing repository is not an
// error here: the tree works on any folder, and "no repo" just means no
// decoration — 200 {"git": false}.
func registerGitStatusRoutes(mux *http.ServeMux, deps Deps) {
	mux.HandleFunc("GET /api/agents/{id}/gitstatus", handleAgentGitStatus(deps))
	mux.HandleFunc("GET /api/terminals/{id}/gitstatus", handleTerminalGitStatus(deps))
	mux.HandleFunc("GET /api/workspaces/{id}/gitstatus", handleWorkspaceGitStatus(deps))
}

func handleAgentGitStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, err := agentCwd(deps, r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeGitStatus(w, cwd)
	}
}

func handleTerminalGitStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		term, err := deps.Store.GetTerminal(r.PathValue("id"))
		if err != nil {
			writeStoreErr(w, err)
			return
		}
		writeGitStatus(w, liveTermCwd(deps, r, term))
	}
}

func handleWorkspaceGitStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		writeGitStatus(w, cwd)
	}
}

func writeGitStatus(w http.ResponseWriter, cwd string) {
	top, raw := gitgraph.Status(cwd)
	if top == "" {
		writeJSON(w, http.StatusOK, map[string]any{"git": false, "changes": []gitgraph.Change{}})
		return
	}
	// git reports paths relative to the repository toplevel; the tree's
	// paths are relative to the owner's cwd. Re-anchor, and drop what falls
	// outside the cwd — the tree could not show it and the reader could not
	// open it anyway.
	root, topDir := canonDir(cwd), canonDir(top)
	changes := []gitgraph.Change{}
	for _, c := range raw {
		rel, err := filepath.Rel(root, filepath.Join(topDir, filepath.FromSlash(c.Path)))
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		changes = append(changes, gitgraph.Change{Path: rel, Kind: c.Kind})
	}
	writeJSON(w, http.StatusOK, map[string]any{"git": true, "repoRoot": top, "changes": changes})
}
