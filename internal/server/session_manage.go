package server

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
)

// Workspace session management: one view for every Pi session under the
// workspace folder (A) plus Claude-style age-based cleanup of orphans (B).
// Orphan = not the current session of any agent.

const settingCleanupDays = "cleanup_orphan_days"

type sessionUse struct {
	AgentID   string `json:"agentId"`
	AgentName string `json:"agentName"`
}

// workspaceSessionDirs is every directory pi may have written this
// workspace's sessions into (ADR-0040): the shared cwd bucket (sessions
// from before this ADR, plus anything a Terminal or bare `pi` writes
// there) and each of the workspace's agents' own private dirs.
func workspaceSessionDirs(deps Deps, wk store.Workspace) []string {
	dirs := []string{session.Dir(wk.Path)}
	agents, err := deps.Store.ListAgents(wk.ID)
	if err != nil {
		return dirs
	}
	for _, a := range agents {
		dirs = append(dirs, session.AgentDir(a.ID))
	}
	return dirs
}

// workspaceOmpAgentDirs are the per-agent Omp session directories of a
// workspace's Omp agents (--session-dir, cli_launch.go). Removed agents
// are not listed; their transcripts live on in the agent history.
func workspaceOmpAgentDirs(deps Deps, wk store.Workspace) []string {
	agents, err := deps.Store.ListAgents(wk.ID)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, a := range agents {
		if a.CLI == "omp" {
			dirs = append(dirs, ompAgentSessionDir(deps.DataDir, a.ID))
		}
	}
	return dirs
}

// ompSessionUseBy maps the session file each Omp agent's terminal is pinned
// to (its current conversation) to the agent, so the Sessions view offers
// that agent instead of starting a second Omp on the same file.
func ompSessionUseBy(deps Deps) map[string]sessionUse {
	out := map[string]sessionUse{}
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return out
	}
	for _, a := range agents {
		if a.CLI != "omp" || a.TerminalID == nil {
			continue
		}
		launch, err := deps.Store.TerminalLaunch(*a.TerminalID)
		if err != nil || launch == nil || launch.LastSession == nil || launch.LastSession.CLI != "omp" || launch.LastSession.Path == "" {
			continue
		}
		if _, taken := out[launch.LastSession.Path]; !taken {
			out[launch.LastSession.Path] = sessionUse{AgentID: a.ID, AgentName: a.Name}
		}
	}
	return out
}

// sessionUseBy maps every agent's current session path to the agent.
func sessionUseBy(deps Deps) map[string]sessionUse {
	out := map[string]sessionUse{}
	agents, err := deps.Store.ListAllAgents()
	if err != nil {
		return out
	}
	for _, a := range agents {
		if a.SessionPath == nil || strings.TrimSpace(*a.SessionPath) == "" {
			continue
		}
		if _, taken := out[*a.SessionPath]; taken {
			continue
		}
		out[*a.SessionPath] = sessionUse{AgentID: a.ID, AgentName: a.Name}
	}
	return out
}

func cleanupDaysSetting(deps Deps) int {
	v, ok, err := deps.Store.GetSetting(settingCleanupDays)
	if err != nil || !ok {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// sweepOrphanSessions deletes session files under every known agent cwd
// that (a) no agent is bound to and (b) have not been touched in days.
func sweepOrphanSessions(deps Deps, days int) int {
	if days <= 0 {
		return 0
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	use := sessionUseBy(deps)
	// ADR-0039 already surfaces an agent's older, non-current sessions as
	// resumable in its own chat picker — the age sweep must not delete one
	// out from under that picker just because it isn't the *current*
	// pointer `use` tracks.
	owned, _ := deps.Store.AllAgentSessionPaths()

	dirs := map[string]bool{}
	if wss, err := deps.Store.ListWorkspaces(); err == nil {
		for _, wk := range wss {
			dirs[session.Dir(wk.Path)] = true
		}
	}
	if agents, err := deps.Store.ListAllAgents(); err == nil {
		for _, a := range agents {
			if p := a.WorkPath; p != nil && strings.TrimSpace(*p) != "" {
				dirs[session.Dir(*p)] = true
			}
			dirs[session.AgentDir(a.ID)] = true // ADR-0040
		}
	}

	removed := 0
	for dir := range dirs {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			p := filepath.Join(dir, e.Name())
			if _, inUse := use[p]; inUse {
				continue
			}
			if owned[p] {
				continue
			}
			fi, err := e.Info()
			if err != nil || fi.ModTime().After(cutoff) {
				continue
			}
			if err := os.Remove(p); err == nil {
				removed++
				log.Printf("session sweep: removed orphan %s (untouched since %s)", p, fi.ModTime().UTC().Format(time.RFC3339))
			}
		}
	}
	if removed > 0 {
		_ = deps.Store.AppendEvent("session_sweep", nil, nil, map[string]any{"removed": removed, "days": days})
	}
	return removed
}
