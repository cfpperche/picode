package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/skills"
	"github.com/cfpperche/picode/internal/store"
)

// An agent's own skills at launch (ADR-0196 slice 4). Each CLI takes them
// its own way, measured 2026-09-24 against the CLI's command list:
//
//	Pi           --skill <folder> per skill (store.Agent.CLIFlags)
//	Omp          --config <overlay> with skills.customDirectories
//	Claude Code  --plugin-dir <folder> holding skills/<name>
//
// The other six have no per-launch way to add a folder: their rows say so.

// agentLaunchFingerprint is the launch configuration an agent terminal ran
// with, the agent's own scope included, so the pane can say a restart is
// needed after that scope changes. Pi's already folds in its flags; for the
// others the scope rides as one marker argument that is never executed.
func agentLaunchFingerprint(c clilaunch.Config, a store.Agent) string {
	if a.IsPi() {
		return piAgentLaunchFingerprint(c, a)
	}
	marker := agentScopeMarker(a)
	if marker == "" {
		return clilaunch.Fingerprint(c)
	}
	c.Args = append(append([]string{}, c.Args...), "--picode-agent-scope="+marker)
	return clilaunch.Fingerprint(c)
}

// agentScopeMarker is what of the agent's own scope its CLI receives: Omp's
// packages, isolation and skills; Claude Code's skills; nothing elsewhere.
// Empty when there is nothing, so an agent without a scope keeps the plain
// fingerprint it always had.
func agentScopeMarker(a store.Agent) string {
	type skill struct{ Name, Digest string }
	var m struct {
		Packages []string `json:",omitempty"`
		Isolated bool     `json:",omitempty"`
		Skills   []skill  `json:",omitempty"`
	}
	switch a.CLI {
	case "omp":
		m.Packages, m.Isolated = a.Packages, a.PackagesIsolated
	case "claude-code":
	default:
		return ""
	}
	for _, sk := range a.Skills {
		m.Skills = append(m.Skills, skill{sk.Name, sk.Digest})
	}
	if len(m.Packages) == 0 && !m.Isolated && len(m.Skills) == 0 {
		return ""
	}
	raw, _ := json.Marshal(m)
	return string(raw)
}

// agentOmpSkillFlags writes the agent's Omp overlay and names it. Each
// cached skill lives at <cache>/<digest>/<name>, so its parent is a folder
// holding exactly that skill. An isolated agent's overlay turns off every
// folder Omp would read, leaving only these (measured: 14 skills → 1).
func agentOmpSkillFlags(dataDir string, a store.Agent) ([]string, error) {
	dirs := a.SkillDirs()
	if len(dirs) == 0 {
		return nil, nil
	}
	sk := map[string]any{}
	parents := make([]string, 0, len(dirs))
	for _, d := range dirs {
		parents = append(parents, filepath.Dir(d))
	}
	sk["customDirectories"] = parents
	if a.PackagesIsolated {
		for _, k := range []string{"enableCodexUser", "enableClaudeUser", "enableClaudeProject", "enablePiUser", "enablePiProject", "enableAgentsUser", "enableAgentsProject"} {
			sk[k] = false
		}
	}
	raw, err := yaml.Marshal(map[string]any{"skills": sk})
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(dataDir, "skills", "agents", a.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	p := filepath.Join(dir, "omp.yml")
	if err := os.WriteFile(p, raw, 0o600); err != nil {
		return nil, err
	}
	return []string{"--config", p}, nil
}

// claudeAgentPlugin is the plugin folder's name, and so the prefix Claude
// Code gives the agent's skills.
const claudeAgentPlugin = "picode-agent"

// writeClaudeAgentPlugin copies the agent's cached skills into this run's
// folder as a plugin; empty when the agent has none.
func writeClaudeAgentPlugin(runDir string, a store.Agent) (string, error) {
	dirs := a.SkillDirs()
	if len(dirs) == 0 {
		return "", nil
	}
	root := filepath.Join(runDir, claudeAgentPlugin)
	for _, d := range dirs {
		if err := skills.CopyFolder(d, filepath.Join(root, "skills", filepath.Base(d))); err != nil {
			return "", err
		}
	}
	return root, nil
}

// agentSkillsPlan is the Launch pane's view of the agent's own skills: the
// flag each CLI takes them by, the folders it reads, and the ones left out
// because their cached copy is gone. nil when the terminal's agent has no
// skills of its own, or its CLI takes none at launch.
func agentSkillsPlan(deps Deps, cliID, terminalID, runDir string) *clilaunch.IntegrationPlan {
	if deps.Store == nil {
		return nil
	}
	a, err := deps.Store.AgentByTerminal(terminalID)
	if err != nil || len(a.Skills) == 0 {
		return nil
	}
	dirs := a.SkillDirs()
	missing := len(a.Skills) - len(dirs)
	p := &clilaunch.IntegrationPlan{Branches: []clilaunch.Injection{}, Files: append([]string{}, dirs...), Environment: map[string]string{}}
	var args []string
	switch {
	case a.IsPi() && cliID == "pi":
		for _, d := range dirs {
			args = append(args, "--skill", d)
		}
	case cliID == "omp" && a.CLI == "omp":
		if len(dirs) > 0 {
			args = []string{"--config", filepath.Join(deps.DataDir, "skills", "agents", a.ID, "omp.yml")}
		}
	case cliID == "claude-code" && a.CLI == "claude-code":
		if len(dirs) > 0 {
			args = []string{"--plugin-dir", filepath.Join(runDir, claudeAgentPlugin)}
		}
	default:
		return nil
	}
	n := len(dirs)
	p.Summary = fmt.Sprintf("%d skill%s of %s's own", n, plural(n), a.Name)
	if missing > 0 {
		p.Summary += fmt.Sprintf("; %d gone from PiCode's copy, left out", missing)
	}
	if len(args) > 0 {
		p.Branches = append(p.Branches, clilaunch.Injection{When: "Every start", Args: args})
	}
	return p
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
