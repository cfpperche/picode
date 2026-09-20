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
	changes, totals := reanchorChanges(st.Changes, canonDir(cwd), canonDir(st.Top))
	page := statusEntry(st, changes, totals)
	page["git"] = true
	page["repoRoot"] = st.Top
	if wts, truncated := dirtyLinkedWorktrees(cwd, st.Top); len(wts) > 0 || truncated > 0 {
		page["worktrees"] = wts
		if truncated > 0 {
			page["worktreesTruncated"] = truncated
		}
	}
	writeJSON(w, http.StatusOK, page)
}

// reanchorChanges moves change paths from the repository toplevel to the
// reader's folder, dropping what falls outside it — the tree could not show
// it and the reader could not open it anyway. Totals count only what
// survives, so a reader confined to a subfolder sees its own slice summed,
// not the whole repository's.
func reanchorChanges(all []gitgraph.ChangeStat, root, topDir string) ([]gitgraph.ChangeStat, map[string]int) {
	changes := []gitgraph.ChangeStat{}
	totals := map[string]int{"add": 0, "del": 0, "files": 0}
	for _, c := range all {
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
	return changes, totals
}

// statusEntry is the branch facts plus the change list one Changes group
// draws: the root page and every worktrees[] entry share the shape, so the
// rail renders either without a second code path.
func statusEntry(st gitgraph.StatusInfo, changes []gitgraph.ChangeStat, totals map[string]int) map[string]any {
	entry := map[string]any{"branch": st.Branch, "changes": changes, "totals": totals, "ahead": st.Ahead, "behind": st.Behind}
	if st.Worktree != "" {
		entry["worktree"] = st.Worktree
	}
	if st.Upstream != "" {
		entry["upstream"] = st.Upstream
	}
	if st.Detached {
		entry["detached"] = true
	}
	return entry
}

// maxSessionWorktrees bounds the sibling scan: each linked worktree costs a
// full StatusWithStats, so past the cap the rest count as truncated rather
// than turning one status read into seconds of git.
const maxSessionWorktrees = 8

// dirtyLinkedWorktrees is the session-following signal: the dirty checkouts
// of the owner's repository besides its own, in git's list order. The
// owner's own checkout, bare and prunable entries never appear — a prunable
// checkout may be missing, and a status call there can only fail or lie
// (the graph's annotateWorktrees skips them for the same reason). Clean
// checkouts are noise to a Changes list and are omitted; ref is the
// ?worktree= value that reads through that checkout (the branch, or the
// full HEAD hash when detached).
func dirtyLinkedWorktrees(cwd, top string) ([]any, int) {
	out := []any{}
	scanned, truncated := 0, 0
	for _, wt := range gitgraph.ListWorktrees(cwd) {
		if wt.Bare || wt.Prunable || wt.Path == "" || sameDir(wt.Path, top) {
			continue
		}
		if scanned >= maxSessionWorktrees {
			truncated++
			continue
		}
		scanned++
		st := gitgraph.StatusWithStats(wt.Path)
		if st.Top == "" || len(st.Changes) == 0 {
			continue
		}
		ref := st.Branch
		if st.Detached {
			ref = wt.Head
		}
		if ref == "" {
			// No name reads through this checkout (an unborn HEAD has
			// neither branch nor hash): list nothing rather than a ref
			// that resolves to the owner's own tree.
			continue
		}
		changes, totals := reanchorChanges(st.Changes, canonDir(st.Top), canonDir(st.Top))
		entry := statusEntry(st, changes, totals)
		entry["path"] = wt.Path
		entry["ref"] = ref
		out = append(out, entry)
	}
	return out, truncated
}
