package server

import (
	"net/http"
	"path/filepath"
	"strings"

	"github.com/cfpperche/picode/internal/gitgraph"
)

// The gitstatus routes decorate the file tree (ADR-0030) with what changed
// in the owner's repository and feed the Inspector rail's Changes list with
// per-file line counts, totals and the branch. Unlike /git, a missing
// repository is not an error here: the tree works on any folder, and "no
// repo" just means no decoration — 200 {"git": false}.
func registerGitStatusRoutes(mux Registrar, deps Deps) {
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
		cwd, ok := resolveGitWorktree(w, r, cwd)
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
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
		cwd, ok := resolveGitWorktree(w, r, liveTermCwd(deps, r, term))
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		writeGitStatus(w, cwd)
	}
}

func handleWorkspaceGitStatus(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cwd, ok := workspaceFilesCwd(deps, w, r.PathValue("id"))
		if !ok {
			return
		}
		cwd, ok = resolveGitWorktree(w, r, cwd)
		if !ok {
			return
		}
		if !checkFileRoot(w, r, cwd) {
			return
		}
		writeGitStatus(w, cwd)
	}
}

func writeGitStatus(w http.ResponseWriter, cwd string) {
	st := gitgraph.StatusWithStats(cwd)
	if st.Top == "" {
		writeJSON(w, http.StatusOK, map[string]any{"git": false, "changes": []gitgraph.ChangeStat{}})
		return
	}
	// git reports paths relative to the repository toplevel; the tree's
	// paths are relative to the owner's cwd. Re-anchor, and drop what falls
	// outside the cwd — the tree could not show it and the reader could not
	// open it anyway. Totals count only what survives, so a reader confined
	// to a subfolder sees its own slice summed, not the whole repository's.
	root, topDir := canonDir(cwd), canonDir(st.Top)
	changes := []gitgraph.ChangeStat{}
	totals := map[string]int{"add": 0, "del": 0, "files": 0}
	for _, c := range st.Changes {
		rel, err := filepath.Rel(root, filepath.Join(topDir, filepath.FromSlash(c.Path)))
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == ".." || strings.HasPrefix(rel, "../") {
			continue
		}
		c.Path = rel
		changes = append(changes, c)
		totals["add"] += c.Add
		totals["del"] += c.Del
		totals["files"]++
	}
	page := map[string]any{"git": true, "repoRoot": st.Top, "branch": st.Branch, "changes": changes, "totals": totals}
	if st.Worktree != "" {
		page["worktree"] = st.Worktree
	}
	writeJSON(w, http.StatusOK, page)
}
