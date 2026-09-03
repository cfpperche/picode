package server

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/gitinfo"
	"github.com/cfpperche/picode/internal/store"
)

// StartGitWatch inspects every workspace path and agent cwd once per tick
// and publishes changes as ephemeral git.updated events (ADR-0048). This
// is one `git` subprocess set per directory per tick, for the whole
// fleet, instead of every browser refetching the fleet or the file tree
// after each commit. Terminal panes are not watched: their cwd is live
// tmux state, their pills still refresh with the fleet.
func StartGitWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Feed == nil || deps.Store == nil {
		return
	}
	prev := map[string]string{}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		workspaces, err := deps.Store.ListWorkspaces()
		if err != nil {
			continue
		}
		agents, err := deps.Store.ListAllAgents()
		if err != nil {
			continue
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
		for _, path := range diffGit(prev, cur) {
			data := map[string]any{"path": path, "workspaceIds": groups[path].workspaceIDs, "agentIds": groups[path].agentIDs}
			if info, ok := infos[path]; ok {
				data["branch"] = info.Branch
				data["dirty"] = info.Dirty
				data["worktree"] = info.Worktree
			}
			deps.Feed.Ephemeral("git.updated", data)
		}
		prev = cur
	}
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
