package store

import (
	"os"
	"path/filepath"
	"testing"
)

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

	w, agent, err := s.AddWorkspace("My App", proj)
	if err != nil {
		t.Fatalf("AddWorkspace: %v", err)
	}
	if w.ID == "" || w.Path != proj {
		t.Fatalf("workspace = %+v", w)
	}
	if agent.WorkspaceID != w.ID || agent.Name != "default" || agent.LastStatus != StatusNeverStarted {
		t.Fatalf("default agent = %+v", agent)
	}

	// Idempotent by path; same agent returned.
	w2, agent2, err := s.AddWorkspace("My App again", proj)
	if err != nil {
		t.Fatalf("re-add: %v", err)
	}
	if w2.ID != w.ID || agent2.ID != agent.ID {
		t.Errorf("re-add created new rows: %+v/%+v", w2, agent2)
	}

	if _, _, err := s.AddWorkspace("", proj); err == nil {
		t.Error("empty name accepted")
	}
	if _, _, err := s.AddWorkspace("x", "/definitely/not/here"); err == nil {
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
	w, agent, err := s.AddWorkspace("Tasks", t.TempDir())
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
	w, agent, err := s.AddWorkspace("Evts", t.TempDir())
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
	_, agent, err := s.AddWorkspace("Cfg", t.TempDir())
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
	_, agent, err := s.AddWorkspace("Mode", t.TempDir())
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
	if len(flags) != 2 || flags[0] != want[0] || flags[1] != want[1] {
		t.Fatalf("CLIFlags = %v, want %v", flags, want)
	}
	full := "full"
	got, err = s.UpdateAgent(agent.ID, AgentPatch{OpMode: &full})
	if err != nil {
		t.Fatal(err)
	}
	if got.OpMode != nil {
		t.Fatalf("full should clear op_mode, got %v", got.OpMode)
	}
	if len(got.CLIFlags()) != 0 {
		t.Fatalf("full flags = %v", got.CLIFlags())
	}
	bad := "bypass"
	if _, err := s.UpdateAgent(agent.ID, AgentPatch{OpMode: &bad}); err == nil {
		t.Fatal("want error for unknown mode")
	}
}

func TestAgentPackagesCLIFlags(t *testing.T) {
	s := openTest(t)
	_, agent, err := s.AddWorkspace("Pkg", t.TempDir())
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
	if len(flags) != len(want) {
		t.Fatalf("%v", flags)
	}
	for i := range want {
		if flags[i] != want[i] {
			t.Fatalf("%v", flags)
		}
	}
	got, err = s.SetAgentPackages(agent.ID, nil)
	if err != nil || len(got.Packages) != 0 || len(got.CLIFlags()) != 0 {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestSessionPathFlag(t *testing.T) {
	s := openTest(t)
	_, agent, err := s.AddWorkspace("Sess", t.TempDir())
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
