package server

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitgraph"
	"github.com/cfpperche/picode/internal/gitinfo"
	"github.com/cfpperche/picode/internal/store"
)

// StartGitWatch inspects every workspace path and agent cwd once per tick
// and publishes changes as ephemeral git.updated events (ADR-0048). This
// is one `git` subprocess set per directory per tick, for the whole
// fleet, instead of every browser refetching the fleet or the file tree
// after each commit. Terminal panes are not watched: their cwd is live
// tmux state, their pills still refresh with the fleet. Linked worktrees
// of watched repositories are watched too (bounded per repo): the
// Inspector follows dirty siblings, so their folders are read like any
// anchor and their changes must arrive the same way — as path-only events
// that speak for no pill.
func StartGitWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Feed == nil || deps.Store == nil {
		return
	}
	prev := map[string]string{}
	wtKnown := map[string][]string{}
	var tick uint
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		tick++
		prev, wtKnown = gitWatchTick(deps, prev, wtKnown, tick%worktreeRelistEvery == 0)
	}
}

// worktreeRelistEvery is how often a listing pass re-runs `git worktree
// list`: the set changes only when a checkout is added or removed, so the
// per-tick work is the Inspects, not the listings. An anchor with no known
// worktrees still lists every tick — the listing is milliseconds, and a
// fresh `git worktree add` must be watched from its next tick.
const worktreeRelistEvery = 10

// gitWatchTick is one pass of the watcher: inspect every watched directory,
// publish one ephemeral git.updated per changed path, and hand back the keys
// the next pass compares against plus the worktree sets it listed. The state
// is the caller's maps rather than fields so a test can drive two passes
// with a store that changed in between.
func gitWatchTick(deps Deps, prev map[string]string, wtKnown map[string][]string, relist bool) (map[string]string, map[string][]string) {
	workspaces, err := deps.Store.ListWorkspaces()
	if err != nil {
		return prev, wtKnown
	}
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return prev, wtKnown
	}

	// Group by path: one Inspect per directory, one event per changed
	// directory, carrying every workspace and agent that lives there.
	groups := map[string]*gitGroup{}
	for _, d := range gitDirs(workspaces, agents) {
		g, ok := groups[d.path]
		if !ok {
			g = &gitGroup{}
			groups[d.path] = g
		}
		if d.workspaceID != "" {
			g.workspaceIDs = append(g.workspaceIDs, d.workspaceID)
		}
		if d.agentID != "" {
			g.agentIDs = append(g.agentIDs, d.agentID)
		}
	}
	paths := make([]string, 0, len(groups))
	for path := range groups {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	cur := map[string]string{}
	infos := map[string]*gitinfo.Info{}
	for _, path := range paths {
		var key string
		if info := gitinfo.Inspect(path); info != nil {
			key = fmt.Sprintf("%s\x00%s\x00%d", info.Branch, info.Worktree, info.Dirty)
			infos[path] = info
		}
		cur[path] = key
	}
	// A linked worktree is watched for path-matching readers (the
	// Inspector's followed groups) only: its event carries no ids, so the
	// fleet reducer leaves the anchor's pills alone. Carrying the anchor's
	// group here once overwrote every pill with the sibling's branch.
	wtAnchor := map[string]string{}
	wtNext := map[string][]string{}
	for _, path := range paths {
		if infos[path] == nil {
			continue
		}
		known, cached := wtKnown[path]
		if !cached || relist || len(known) == 0 {
			known = linkedWorktreePaths(path)
		}
		wtNext[path] = known
		for _, wt := range known {
			if dup := watchedDir(cur, wt); dup {
				// Also watched under the anchor's own spelling (an
				// agent bound inside this checkout): key the git
				// spelling too, with the same key — readers know it
				// under the name gitstatus gave them, which need not
				// be the registered path's spelling.
				for existing := range cur {
					if sameDir(existing, wt) {
						cur[wt] = cur[existing]
						if info, has := infos[existing]; has {
							infos[wt] = info
						}
						wtAnchor[wt] = path
						break
					}
				}
				continue
			}
			var key string
			if info := gitinfo.Inspect(wt); info != nil {
				key = fmt.Sprintf("%s\x00%s\x00%d", info.Branch, info.Worktree, info.Dirty)
				infos[wt] = info
			}
			cur[wt] = key
			wtAnchor[wt] = path
		}
	}
	for _, path := range diffGit(prev, cur) {
		// A path that left the watch set this tick is a workspace or an agent
		// that was just removed: it has no group to speak for, and nothing is
		// said about a folder nobody reads — the durable removed event is what
		// reconciles the lists (ADR-0048). Reading the group out of the map
		// without this check is what killed the daemon on every "Remove
		// workspace" until 2026-09-13.
		g, ok := groups[path]
		if !ok {
			if _, isWT := wtAnchor[path]; isWT {
				g, ok = &gitGroup{}, true
			}
		}
		if !ok {
			continue
		}
		data := map[string]any{"path": path, "workspaceIds": g.workspaceIDs, "agentIds": g.agentIDs}
		if info, ok := infos[path]; ok {
			data["branch"] = info.Branch
			data["dirty"] = info.Dirty
			data["worktree"] = info.Worktree
		}
		deps.Feed.Ephemeral("git.updated", data)
	}
	return cur, wtNext
}

type gitGroup struct {
	workspaceIDs []string
	agentIDs     []string
}

type gitDir struct {
	path        string
	workspaceID string
	agentID     string
}

// maxWatchedWorktrees bounds the per-repo sibling watch: each linked
// worktree costs a full Inspect per tick, on top of the one `git worktree
// list` that names them.
const maxWatchedWorktrees = 8

// linkedWorktreePaths is every healthy checkout of path's repository but
// path's own, in git's list order. Bare and prunable entries are skipped:
// a missing checkout has no state worth keying.
func linkedWorktreePaths(repo string) []string {
	out := []string{}
	for _, wt := range gitgraph.ListWorktrees(repo) {
		if wt.Bare || wt.Prunable || wt.Path == "" || sameDir(wt.Path, repo) {
			continue
		}
		out = append(out, wt.Path)
		if len(out) >= maxWatchedWorktrees {
			break
		}
	}
	return out
}

// watchedDir reports whether dir already has a key in the tick's map: an
// agent bound inside a linked worktree is watched under its own path, and
// must not be inspected twice under two spellings of one folder.
func watchedDir(cur map[string]string, dir string) bool {
	for path := range cur {
		if sameDir(path, dir) {
			return true
		}
	}
	return false
}

// diffGit lists the paths whose key changed between two ticks: a branch
// flip, a dirty-count change, a directory that became (or stopped being)
// a repo. Sorted so tests and subscribers see a stable order.
func diffGit(prev, cur map[string]string) []string {
	var out []string
	for path, key := range cur {
		if prev[path] != key {
			out = append(out, path)
		}
	}
	for path := range prev {
		if _, ok := cur[path]; !ok {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

// gitDirs lists the directories the sidebar's pills describe: every
// registered workspace path, and every agent's cwd (workPath wins, else
// its workspace's path; free agents without a workPath have no pill).
func gitDirs(workspaces []store.Workspace, agents []store.Agent) []gitDir {
	byID := make(map[string]store.Workspace, len(workspaces))
	var out []gitDir
	for _, w := range workspaces {
		if store.IsFree(w) || strings.TrimSpace(w.Path) == "" {
			continue
		}
		byID[w.ID] = w
		out = append(out, gitDir{path: w.Path, workspaceID: w.ID})
	}
	for _, a := range agents {
		if a.WorkspaceID == store.FreeWorkspaceID {
			if a.WorkPath != nil && strings.TrimSpace(*a.WorkPath) != "" {
				out = append(out, gitDir{path: strings.TrimSpace(*a.WorkPath), agentID: a.ID})
			}
			continue
		}
		w, ok := byID[a.WorkspaceID]
		if !ok {
			continue
		}
		cwd := store.AgentCwd(w, a)
		if cwd == w.Path {
			// Same directory as the workspace: one event covers both.
			out = append(out, gitDir{path: cwd, workspaceID: w.ID, agentID: a.ID})
			continue
		}
		// A workPath of its own describes only that agent — the workspace
		// id must not ride this event, or the workspace's fallback pills
		// inherit the worktree's branch.
		out = append(out, gitDir{path: cwd, agentID: a.ID})
	}
	return out
}
