package store

import (
	"os"
	"path/filepath"
	"testing"
)

// addWorkspaceWithAgent keeps the old AddWorkspace shape for tests:
// workspaces start empty (ADR-0027), so the agent is explicit now.
func addWorkspaceWithAgent(s *Store, name, path string) (Workspace, Agent, error) {
	w, err := s.AddWorkspace(name, path)
	if err != nil {
		return Workspace{}, Agent{}, err
	}
	a, err := s.AddAgent(w.ID, "default", "")
	return w, a, err
}

func openTest(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestWorkspaceAndAgents(t *testing.T) {
	s := openTest(t)
	proj := t.TempDir()

	w, err := s.AddWorkspace("My App", proj)
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	if w.ID == "" || w.Path != proj {
		t.Fatalf("workspace = %+v", w)
	}
	// The workspace starts empty (ADR-0027); the agent is explicit.
	if none, err := s.ListAgents(w.ID); err != nil || len(none) != 0 {
		t.Fatalf("new workspace agents = %+v, %v", none, err)
	}
	agent, err := s.AddAgent(w.ID, "default", "")
	if err != nil {
		t.Fatalf("AddAgent: %v", err)
	}
	if agent.WorkspaceID != w.ID || agent.Name != "default" || agent.LastStatus != StatusNeverStarted {
		t.Fatalf("agent = %+v", agent)
	}

	// Idempotent by path; no agent created or resurrected on re-add.
	w2, err := s.AddWorkspace("My App again", proj)
	if err != nil {
		t.Fatalf("re-add: %v", err)
	}
	if w2.ID != w.ID {
		t.Errorf("re-add created new workspace: %+v", w2)
	}
	if after, err := s.ListAgents(w.ID); err != nil || len(after) != 1 {
		t.Fatalf("re-add changed agents: %+v, %v", after, err)
	}

	if _, err := s.AddWorkspace("", proj); err == nil {
		t.Error("empty name accepted")
	}
	if _, err := s.AddWorkspace("x", "/definitely/not/here"); err == nil {
		t.Error("missing path accepted")
	}
	got, err := s.GetWorkspace(w.ID)
	if err != nil || got.Name != "My App" {
		t.Fatalf("GetWorkspace = %+v, %v", got, err)
	}
	if _, err := s.GetWorkspace("nope"); err != ErrNotFound {
		t.Errorf("GetWorkspace(missing) err = %v, want ErrNotFound", err)
	}

	list, err := s.ListWorkspaces()
	if err != nil || len(list) != 1 {
		t.Fatalf("ListWorkspaces = %d, %v", len(list), err)
	}
	sib, err := s.AddAgent(w.ID, "review", "")
	if err != nil {
		t.Fatalf("AddAgent: %v", err)
	}
	inWs, err := s.ListAgents(w.ID)
	if err != nil || len(inWs) != 2 {
		t.Fatalf("ListAgents = %d, %v", len(inWs), err)
	}
	if _, err := s.AddAgent(FreeWorkspaceID, "scratch", t.TempDir()); err != nil {
		t.Fatalf("free agent: %v", err)
	}
	if sib.Name != "review" {
		t.Fatalf("sib = %+v", sib)
	}

	// Runtime status cache.
	if err := s.SetAgentRuntime(agent.ID, StatusRunning); err != nil {
		t.Fatalf("SetAgentRuntime: %v", err)
	}
	a, err := s.GetAgent(agent.ID)
	if err != nil || a.LastStatus != StatusRunning || a.LastStartedAt == nil {
		t.Fatalf("agent after running = %+v, %v", a, err)
	}
	if err := s.SetAgentRuntime("missing", StatusRunning); err != ErrNotFound {
		t.Errorf("SetAgentRuntime(missing) = %v, want ErrNotFound", err)
	}

	// Turn settle: the ready age's stamp moves, the runtime stays running.
	// RFC3339Nano makes "after" strictly greater than "before" deterministic.
	if err := s.SetAgentTurnSettled(agent.ID); err != nil {
		t.Fatalf("SetAgentTurnSettled: %v", err)
	}
	settled, err := s.GetAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	before := a.LastStatusAt
	if settled.LastStatus != StatusRunning {
		t.Errorf("last_status after settle = %q, want running", settled.LastStatus)
	}
	if settled.LastStatusAt == nil || before == nil || *settled.LastStatusAt <= *before {
		t.Errorf("last_status_at after settle = %v, want > %v", settled.LastStatusAt, before)
	}
	if err := s.SetAgentTurnSettled("missing"); err != ErrNotFound {
		t.Errorf("SetAgentTurnSettled(missing) = %v, want ErrNotFound", err)
	}

	// Cascade delete removes agent + tasks + events.
	if _, err := s.EnqueueTask(agent.ID, TaskPrompt, "do it", "user"); err != nil {
		t.Fatalf("EnqueueTask: %v", err)
	}
	removed, err := s.RemoveWorkspace(w.ID)
	if err != nil || !removed {
		t.Fatalf("RemoveWorkspace = %v, %v", removed, err)
	}
	if removed, _ := s.RemoveWorkspace(w.ID); removed {
		t.Error("double remove = true")
	}
	if _, err := s.GetAgent(agent.ID); err != ErrNotFound {
		t.Errorf("agent after cascade = %v, want ErrNotFound", err)
	}
	tasks, _ := s.ListTasks(agent.ID, 10)
	if len(tasks) != 0 {
		t.Errorf("tasks after cascade = %d, want 0", len(tasks))
	}
}

func TestTasksLifecycle(t *testing.T) {
	s := openTest(t)
	w, agent, err := addWorkspaceWithAgent(s, "Tasks", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	if _, err := s.EnqueueTask(agent.ID, "bogus", "x", "user"); err == nil {
		t.Error("invalid kind accepted")
	}
	if _, err := s.EnqueueTask(agent.ID, TaskSteer, "  ", "user"); err == nil {
		t.Error("empty payload accepted")
	}
	if _, err := s.EnqueueTask("missing", TaskPrompt, "x", "user"); err != ErrNotFound {
		t.Errorf("missing agent = %v, want ErrNotFound", err)
	}

	t1, err := s.EnqueueTask(agent.ID, TaskPrompt, "first", "user")
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	t2, _ := s.EnqueueTask(agent.ID, TaskFollowUp, "second", "broker")

	// FIFO claim.
	claimed, err := s.ClaimNextTask(agent.ID)
	if err != nil || claimed.ID != t1.ID || claimed.Status != TaskDelivering || claimed.Attempts != 1 {
		t.Fatalf("claim = %+v, %v", claimed, err)
	}
	if err := s.FinishTask(t1.ID, TaskDelivered, ""); err != nil {
		t.Fatalf("finish: %v", err)
	}
	if _, err := s.ClaimNextTask(agent.ID); err != nil {
		t.Fatalf("claim2: %v", err)
	}
	if err := s.FinishTask(t2.ID, TaskFailed, "rpc gone"); err != nil {
		t.Fatalf("finish fail: %v", err)
	}
	tasks, err := s.ListTasks(agent.ID, 10)
	if err != nil || len(tasks) != 2 {
		t.Fatalf("ListTasks = %d, %v", len(tasks), err)
	}
	byID := map[string]Task{}
	for _, tk := range tasks {
		byID[tk.ID] = tk
	}
	if byID[t1.ID].Status != TaskDelivered || byID[t1.ID].DeliveredAt == nil {
		t.Errorf("t1 = %+v", byID[t1.ID])
	}
	if byID[t2.ID].Status != TaskFailed || byID[t2.ID].LastError == nil || *byID[t2.ID].LastError != "rpc gone" {
		t.Errorf("t2 = %+v", byID[t2.ID])
	}

	// Empty queue.
	if _, err := s.ClaimNextTask(agent.ID); err != ErrNotFound {
		t.Errorf("empty queue claim = %v, want ErrNotFound", err)
	}
	_ = w
}

func TestEventsAndSettings(t *testing.T) {
	s := openTest(t)
	w, agent, err := addWorkspaceWithAgent(s, "Evts", t.TempDir())
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}

	if err := s.AppendEvent("agent_started", &agent.ID, &w.ID, map[string]string{"session": "x"}); err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}

	agentEvts, err := s.AgentEvents(agent.ID, 10)
	if err != nil || len(agentEvts) < 1 {
		t.Fatalf("AgentEvents = %d, %v", len(agentEvts), err)
	}
	recent, err := s.RecentEvents(10)
	if err != nil || len(recent) < 2 { // workspace_added + agent_started
		t.Fatalf("RecentEvents = %d, %v", len(recent), err)
	}

	if v, ok, _ := s.GetSetting("theme"); ok || v != "" {
		t.Errorf("absent setting = %q, %v", v, ok)
	}
	if err := s.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	if err := s.SetSetting("theme", "system"); err != nil {
		t.Fatalf("SetSetting upsert: %v", err)
	}
	if v, ok, _ := s.GetSetting("theme"); !ok || v != "system" {
		t.Errorf("theme = %q, %v", v, ok)
	}
}

func TestLegacyJSONImport(t *testing.T) {
	dir := t.TempDir()
	type legacyEntry struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Path      string `json:"path"`
		CreatedAt string `json:"createdAt"`
	}
	legacyStr, err := marshalJSON([]legacyEntry{{
		ID:        "picode-old1",
		Name:      "Legacy",
		Path:      t.TempDir(), // json.Marshal handles Windows backslashes
		CreatedAt: "2026-08-23T10:00:00Z",
	}})
	if err != nil {
		t.Fatalf("marshal legacy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "workspaces.json"), []byte(legacyStr), 0o644); err != nil {
		t.Fatalf("write legacy: %v", err)
	}

	s, err := Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatalf("Open with legacy: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	ws, err := s.ListWorkspaces()
	if err != nil || len(ws) != 1 || ws[0].ID != "picode-old1" {
		t.Fatalf("imported = %+v, %v", ws, err)
	}
	agent, err := s.DefaultAgent("picode-old1")
	if err != nil || agent.Name != "default" {
		t.Fatalf("imported default agent = %+v, %v", agent, err)
	}

	// The legacy file was retired — reopening must not duplicate.
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "workspaces.json")); !os.IsNotExist(err) {
		t.Fatalf("legacy file not retired: %v", err)
	}
	s2, err := Open(filepath.Join(dir, "picode.db"))
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s2.Close()
	ws2, _ := s2.ListWorkspaces()
	if len(ws2) != 1 {
		t.Fatalf("reopen duplicated rows: %d", len(ws2))
	}
}

func TestUpdateAgent(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Cfg", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	prov, model, think := "anthropic", "claude-sonnet-4-5", "low"
	got, err := s.UpdateAgent(agent.ID, AgentPatch{Provider: &prov, Model: &model, Thinking: &think})
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider == nil || *got.Provider != prov || got.Model == nil || *got.Model != model {
		t.Fatalf("updated = %+v", got)
	}
	empty := ""
	got, err = s.UpdateAgent(agent.ID, AgentPatch{Provider: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != nil {
		t.Fatalf("clear provider: %+v", got.Provider)
	}
	if got.Model == nil || *got.Model != model {
		t.Fatalf("model should stay: %+v", got.Model)
	}
}

func TestOpModeCLIFlags(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Mode", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ro := "readonly"
	got, err := s.UpdateAgent(agent.ID, AgentPatch{OpMode: &ro})
	if err != nil {
		t.Fatal(err)
	}
	flags := got.CLIFlags()
	want := []string{"--tools", ReadonlyTools}
	// 4, not 2: every agent also gets a trailing --session-dir pair (ADR-0040).
	if len(flags) != 4 || flags[0] != want[0] || flags[1] != want[1] || flags[2] != "--session-dir" {
		t.Fatalf("CLIFlags = %v, want %v + --session-dir", flags, want)
	}
	full := "full"
	got, err = s.UpdateAgent(agent.ID, AgentPatch{OpMode: &full})
	if err != nil {
		t.Fatal(err)
	}
	if got.OpMode != nil {
		t.Fatalf("full should clear op_mode, got %v", got.OpMode)
	}
	// 2, not 0: the baseline is just --session-dir now (ADR-0040), never empty.
	if flags := got.CLIFlags(); len(flags) != 2 || flags[0] != "--session-dir" {
		t.Fatalf("full flags = %v", flags)
	}
	bad := "bypass"
	if _, err := s.UpdateAgent(agent.ID, AgentPatch{OpMode: &bad}); err == nil {
		t.Fatal("want error for unknown mode")
	}
}

func TestAgentSpawnEnv(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Env", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got := agent.SpawnEnv()
	// Roles overlay (ADR-0033), compact overlay (ADR-0061), the neutral
	// identity pi-inbox reads (ADR-0037), and the checklist obligation
	// pi-checklist reads (ADR-0055).
	if len(got) != 4 || got[0] != RolesAgentEnv+"="+agent.ID || got[1] != CompactAgentEnv+"="+agent.ID || got[2] != AgentIDEnv+"="+agent.ID || got[3] != ChecklistEnv+"="+ChecklistChanges {
		t.Fatalf("SpawnEnv = %v, want roles+compact+agent-id+checklist envs", got)
	}
	if (Agent{}).SpawnEnv() != nil {
		t.Fatal("empty agent must not set env")
	}
}

// ADR-0196 slice 4: the agent's own skills ride --skill, only while the
// cached folder exists; a repeated name replaces the earlier entry; a bad
// name or a relative folder is refused.
func TestAgentSkillsCLIFlags(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Sk", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cache := t.TempDir()
	here := filepath.Join(cache, "d1", "review")
	if err := os.MkdirAll(here, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(here, "SKILL.md"), []byte("---\nname: review\ndescription: x\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gone := filepath.Join(cache, "d2", "gone")
	got, err := s.SetAgentSkills(agent.ID, []AgentSkill{
		{Name: "review", Digest: "old", Dir: filepath.Join(cache, "d0", "review")},
		{Name: "gone", Digest: "d2", Dir: gone},
		{Name: "review", Digest: "d1", Dir: here},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Skills) != 2 || got.Skills[0].Name != "review" || got.Skills[0].Digest != "d1" {
		t.Fatalf("%+v", got.Skills)
	}
	flags := got.CLIFlags()
	n := 0
	for i, f := range flags {
		if f == "--skill" {
			n++
			if flags[i+1] != here {
				t.Fatalf("%v", flags)
			}
		}
	}
	if n != 1 {
		t.Fatalf("a missing cached folder must not ride the launch: %v", flags)
	}
	again, err := s.GetAgent(agent.ID)
	if err != nil || len(again.Skills) != 2 {
		t.Fatalf("%+v %v", again.Skills, err)
	}
	// The agent's row learns which copy is gone; the mark is read, never
	// stored, so a restored copy reads as present again.
	if again.Skills[0].Missing || !again.Skills[1].Missing {
		t.Fatalf("missing marks %+v", again.Skills)
	}
	for _, bad := range [][]AgentSkill{
		{{Name: "Bad Name", Dir: here}},
		{{Name: "a--b", Dir: here}},
		{{Name: "ok", Dir: "relative/ok"}},
	} {
		if _, err := s.SetAgentSkills(agent.ID, bad); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	got, err = s.SetAgentSkills(agent.ID, nil)
	if err != nil || len(got.Skills) != 0 {
		t.Fatalf("%+v %v", got.Skills, err)
	}
}

func TestAgentPackagesCLIFlags(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Pkg", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.SetAgentPackages(agent.ID, []string{"npm:pi-web-search", " npm:pi-web-search ", "git:github.com/x/y"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Packages) != 2 {
		t.Fatalf("%v", got.Packages)
	}
	flags := got.CLIFlags()
	want := []string{"-e", "npm:pi-web-search", "-e", "git:github.com/x/y"}
	// +2, not exact: trailing --session-dir pair (ADR-0040).
	if len(flags) != len(want)+2 || flags[len(want)] != "--session-dir" {
		t.Fatalf("%v", flags)
	}
	for i := range want {
		if flags[i] != want[i] {
			t.Fatalf("%v", flags)
		}
	}
	got, err = s.SetAgentPackages(agent.ID, nil)
	if flags := got.CLIFlags(); err != nil || len(got.Packages) != 0 || len(flags) != 2 || flags[0] != "--session-dir" {
		t.Fatalf("%+v %v %v", got, err, flags)
	}
	on := true
	got, err = s.UpdateAgent(agent.ID, AgentPatch{PackagesIsolated: &on})
	if err != nil {
		t.Fatal(err)
	}
	flags = got.CLIFlags()
	if len(flags) < 4 || flags[0] != "--no-extensions" || flags[3] != "--no-themes" {
		t.Fatalf("isolated flags = %v", flags)
	}
}

func TestSessionPathFlag(t *testing.T) {
	s := openTest(t)
	_, agent, err := addWorkspaceWithAgent(s, "Sess", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	p := "/tmp/x.jsonl"
	got, err := s.UpdateAgent(agent.ID, AgentPatch{SessionPath: &p})
	if err != nil {
		t.Fatal(err)
	}
	flags := got.CLIFlags()
	if len(flags) < 2 || flags[0] != "--session" || flags[1] != p {
		t.Fatalf("flags = %v", flags)
	}
}
