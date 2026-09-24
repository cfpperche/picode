package server

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
)

// agentSkillFixture caches one skill and gives it to the agent.
func agentSkillFixture(t *testing.T, st *store.Store, agentID, cache, name string) store.AgentSkill {
	t.Helper()
	dir := filepath.Join(cache, strings.Repeat("a", 64), name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\ndescription: d\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sk := store.AgentSkill{Name: name, Digest: strings.Repeat("a", 64), Dir: dir}
	if _, err := st.SetAgentSkills(agentID, []store.AgentSkill{sk}); err != nil {
		t.Fatal(err)
	}
	return sk
}

// ADR-0196 slice 4, the launch rows: Omp receives the agent's skills as a
// --config overlay naming the cached folder; isolation keeps them (the
// overlay turns the folders off instead of --no-skills, which would drop
// them too); a skill gone from the cache rides nowhere.
func TestOmpLaunchCarriesTheAgentsSkills(t *testing.T) {
	for _, isolated := range []bool{false, true} {
		deps, cwd, launch := ompLaunchFixture(t, clilaunch.Config{}, nil, isolated)
		a, err := deps.Store.AgentByTerminal(launch.TerminalID)
		if err != nil {
			t.Fatal(err)
		}
		sk := agentSkillFixture(t, deps.Store, a.ID, filepath.Join(deps.DataDir, "skills", "cache"), "review")
		prepared, err := prepareCLITerminal(deps, cwd, launch)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := os.ReadFile(prepared.script)
		prepared.discard()
		overlay := filepath.Join(deps.DataDir, "skills", "agents", a.ID, "omp.yml")
		if !strings.Contains(string(body), "--config") || !strings.Contains(string(body), overlay) {
			t.Fatalf("isolated=%v: no overlay in\n%s", isolated, body)
		}
		if strings.Contains(string(body), "--no-skills") {
			t.Fatalf("isolated=%v: --no-skills would drop the agent's own skills:\n%s", isolated, body)
		}
		var got struct {
			Skills map[string]any `yaml:"skills"`
		}
		raw, _ := os.ReadFile(overlay)
		if err := yaml.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		dirs, _ := got.Skills["customDirectories"].([]any)
		if len(dirs) != 1 || dirs[0] != filepath.Dir(sk.Dir) {
			t.Fatalf("customDirectories %v", got.Skills)
		}
		if off, ok := got.Skills["enableAgentsUser"]; (ok && off == false) != isolated {
			t.Fatalf("isolated=%v overlay %v", isolated, got.Skills)
		}
		if err := os.RemoveAll(sk.Dir); err != nil {
			t.Fatal(err)
		}
		prepared, err = prepareCLITerminal(deps, cwd, launch)
		if err != nil {
			t.Fatal(err)
		}
		body, _ = os.ReadFile(prepared.script)
		prepared.discard()
		if strings.Contains(string(body), "--config") {
			t.Fatalf("a skill gone from the cache still rode the launch:\n%s", body)
		}
	}
}

// Claude Code receives them as a session-only plugin folder in the run
// directory: skills/<name>/SKILL.md under picode-agent.
func TestClaudeLaunchCarriesTheAgentsSkills(t *testing.T) {
	deps, cwd, launch := guestLaunchFixture(t, "claude-code")
	a, err := deps.Store.AgentByTerminal(launch.TerminalID)
	if err != nil {
		t.Fatal(err)
	}
	agentSkillFixture(t, deps.Store, a.ID, filepath.Join(deps.DataDir, "skills", "cache"), "review")
	prepared, err := prepareCLITerminal(deps, cwd, launch)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.discard()
	body, _ := os.ReadFile(prepared.script)
	plugin := filepath.Join(prepared.dir, claudeAgentPlugin)
	if !strings.Contains(string(body), "--plugin-dir") || !strings.Contains(string(body), plugin) {
		t.Fatalf("no plugin folder in\n%s", body)
	}
	if _, err := os.Stat(filepath.Join(plugin, "skills", "review", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
}

// A change to the agent's own scope is a restart the pane announces, for
// the guests as for Pi: the fingerprint the launch records is the one the
// terminal view recomputes, and a new skill moves it. A CLI without an agent
// scope, and an agent without one, keep the plain fingerprint.
func TestAgentScopeMovesTheLaunchFingerprint(t *testing.T) {
	for _, cli := range []string{"omp", "claude-code"} {
		deps, cwd, launch := guestLaunchFixture(t, cli)
		prepared, err := prepareCLITerminal(deps, cwd, launch)
		if err != nil {
			t.Fatal(err)
		}
		prepared.discard()
		_, effective, _, err := resolvedTerminalLaunch(deps, launch)
		if err != nil {
			t.Fatal(err)
		}
		a, _ := deps.Store.AgentByTerminal(launch.TerminalID)
		if got := agentLaunchFingerprint(effective, a); got != prepared.snapshot.Fingerprint {
			t.Fatalf("%s: view %s, launch %s", cli, got, prepared.snapshot.Fingerprint)
		}
		if prepared.snapshot.Fingerprint != clilaunch.Fingerprint(effective) {
			t.Fatalf("%s: an agent without a scope changed fingerprint", cli)
		}
		agentSkillFixture(t, deps.Store, a.ID, t.TempDir(), "review")
		a, _ = deps.Store.AgentByTerminal(launch.TerminalID)
		if agentLaunchFingerprint(effective, a) == prepared.snapshot.Fingerprint {
			t.Fatalf("%s: a new skill did not ask for a restart", cli)
		}
	}
	codex := store.Agent{CLI: "codex", Skills: []store.AgentSkill{{Name: "x", Digest: "d"}}}
	if agentScopeMarker(codex) != "" {
		t.Fatal("codex takes no agent skills; its fingerprint must not move")
	}
}

// guestLaunchFixture is one agent of cli bound to a terminal whose binary is
// a stub.
func guestLaunchFixture(t *testing.T, cli string) (Deps, string, *store.TerminalLaunch) {
	t.Helper()
	data := t.TempDir()
	st, err := store.Open(filepath.Join(data, "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	w, err := st.AddWorkspace("Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	term, err := st.CreateTerminalIn(w.ID, "Fixture", data)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(data, "stub")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := st.SetCLIConfig(cli, clilaunch.Config{Executable: binary}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetTerminalLaunch(term.ID, cli, clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, cli, "Atlas", data)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpdateAgent(a.ID, store.AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	launch, err := st.TerminalLaunch(term.ID)
	if err != nil {
		t.Fatal(err)
	}
	return Deps{Store: st, DataDir: data}, data, launch
}
