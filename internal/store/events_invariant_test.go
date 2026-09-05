package store

import (
	"encoding/json"
	"github.com/cfpperche/picode/internal/clilaunch"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// ADR-0048: every mutation announces itself. This table is the contract —
// a new mutating method without a row here is the review signal. Each
// case runs on a fresh store and must produce exactly the listed types,
// each delivered to OnEvent only after the write is durable.
func TestEveryMutationAppendsAnEvent(t *testing.T) {
	dir := t.TempDir()
	proj := filepath.Join(dir, "proj")
	_ = os.MkdirAll(proj, 0o755)

	type tc struct {
		name string
		run  func(s *Store) // setup (uncounted) + the mutation
		want []string
	}
	cases := []tc{
		{"SetCLIProfile", func(s *Store) { _ = s.SetCLIProfile(CLIProfile{ID: "p", CLI: "pi", Name: "Profile"}) }, []string{"cli.profile"}},
		{"DeleteCLIProfile", func(s *Store) {
			_ = s.SetCLIProfile(CLIProfile{ID: "p", CLI: "pi", Name: "Profile"})
			s.OnEvent = recorder(s)
			_ = s.DeleteCLIProfile("p")
		}, []string{"cli.profile"}},
		{"SetCLICheck", func(s *Store) { _ = s.SetCLICheck("pi", clilaunch.Diagnostic{Version: "1"}) }, []string{"cli.checked"}},
		{"SetTerminalLaunchAttempt", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "cli", proj)
			_ = s.SetTerminalLaunch(tm.ID, "pi", clilaunch.Overrides{})
			s.OnEvent = recorder(s)
			_ = s.SetTerminalLaunchAttempt(tm.ID, clilaunch.Attempt{Error: "failed"})
		}, []string{"terminal.launch"}},
		{"SetCLIConfig", func(s *Store) { _ = s.SetCLIConfig("codex", clilaunch.Config{}) }, []string{"cli.updated"}},
		{"ImportCLIConfigs", func(s *Store) { _ = s.ImportCLIConfigs(map[string]bool{"pi": true}) }, []string{"cli.updated", "cli.updated", "cli.updated", "cli.updated"}},
		{"SetTerminalLaunch", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "cli", proj)
			s.OnEvent = recorder(s)
			_ = s.SetTerminalLaunch(tm.ID, "pi", clilaunch.Overrides{})
		}, []string{"terminal.launch"}},
		{"SetTerminalLaunchApplied", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "cli", proj)
			_ = s.SetTerminalLaunch(tm.ID, "pi", clilaunch.Overrides{})
			s.OnEvent = recorder(s)
			_ = s.SetTerminalLaunchApplied(tm.ID, clilaunch.Snapshot{Executable: "/bin/pi"})
		}, []string{"terminal.launch"}},
		{"AddWorkspace", func(s *Store) { _, _ = s.AddWorkspace("W", proj) }, []string{"workspace.added"}},
		{"RemoveWorkspace", func(s *Store) {
			w, _ := s.AddWorkspace("W", proj)
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_, _ = s.RemoveWorkspace(w.ID)
		}, []string{"workspace.deleted"}},
		{"AddAgent", func(s *Store) { _, _ = s.AddAgent(FreeWorkspaceID, "a", "") }, []string{"agent.added"}},
		{"UpdateAgent", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			n := "b"
			_, _ = s.UpdateAgent(a.ID, AgentPatch{Name: &n})
		}, []string{"agent.updated"}},
		{"SetAgentPackages", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			_, _ = s.SetAgentPackages(a.ID, []string{"x"})
		}, []string{"agent.updated"}},
		{"SetAgentRuntime", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			_ = s.SetAgentRuntime(a.ID, StatusRunning)
		}, []string{"agent.status"}},
		{"DeleteAgent", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			_ = s.DeleteAgent(a.ID)
		}, []string{"agent.deleted"}},
		{"SetChecklist", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_, _ = s.SetChecklist(a.ID, "s1", []ChecklistItem{{Text: "x", Status: "pending"}}, false)
		}, []string{"agent.checklist"}},
		{"ClearChecklist", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			_, _ = s.SetChecklist(a.ID, "s1", []ChecklistItem{{Text: "x", Status: "pending"}}, false)
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_, _ = s.ClearChecklist(a.ID)
		}, []string{"agent.checklist"}},
		{"CreateTerminal", func(s *Store) { _, _ = s.CreateTerminalIn("", "t", proj) }, []string{"terminal.created"}},
		{"RenameTerminal", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "t", proj)
			s.OnEvent = recorder(s)
			_, _ = s.RenameTerminal(tm.ID, "u")
		}, []string{"terminal.updated"}},
		{"DeleteTerminal", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "t", proj)
			s.OnEvent = recorder(s)
			_ = s.DeleteTerminal(tm.ID)
		}, []string{"terminal.deleted", "terminal_settings.updated"}},
		{"CreateInboxItem", func(s *Store) {
			_, _ = s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t"})
		}, []string{"inbox.created"}},
		{"CreateInboxItem from an automation (not an agent)", func(s *Store) {
			_, _ = s.CreateInboxItem(InboxItemParams{Kind: InboxResult, SourceKind: InboxFromAutomation, SourceID: "aut-1", WorkspaceID: "gone", Reason: "r", Title: "t"})
		}, []string{"inbox.created"}},
		{"RespondInboxItem", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromSystem, Reason: "r", Title: "t", Body: "b"})
			s.OnEvent = recorder(s)
			_, _ = s.RespondInboxItem(it.ID, VerbIgnore, "")
		}, []string{"inbox.updated"}},
		{"SetInboxItemState", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t"})
			s.OnEvent = recorder(s)
			_, _ = s.SetInboxItemState(it.ID, InboxRead, nil)
		}, []string{"inbox.updated"}},
		{"AnnotateInboxItem", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t"})
			s.OnEvent = recorder(s)
			_ = s.AnnotateInboxItem(it.ID, "n")
		}, []string{"inbox.updated"}},
		{"FileAgentResult supersede", func(s *Store) {
			_, _ = s.FileAgentResult("ag", "", "t", "b", "r")
			s.OnEvent = recorder(s)
			_, _ = s.FileAgentResult("ag", "", "t2", "b2", "r")
		}, []string{"inbox.updated"}},
		{"DeleteInboxItem", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t"})
			s.OnEvent = recorder(s)
			_ = s.DeleteInboxItem(it.ID)
		}, []string{"inbox.deleted"}},
		{"DeleteDoneInboxItems", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxFYI, SourceKind: InboxFromSystem, Reason: "r", Title: "t"})
			_, _ = s.SetInboxItemState(it.ID, InboxDone, nil)
			s.OnEvent = recorder(s)
			_, _ = s.DeleteDoneInboxItems()
		}, []string{"inbox.cleared"}},
		{"RespondAndPark", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: a.ID, Reason: "r", Title: "t", Body: "?"})
			s.OnEvent = recorder(s)
			_, _, _ = s.RespondAndPark(it.ID, VerbRespond, "reply")
		}, []string{"task.enqueued", "inbox.updated"}},
		{"recoverPendingInboxReplies", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: a.ID, Reason: "r", Title: "t", Body: "?"})
			_, _, _ = s.RespondAndPark(it.ID, VerbRespond, "reply")
			s.OnEvent = recorder(s)
			_, _ = s.recoverPendingInboxReplies()
		}, []string{"task.finished", "inbox.updated"}},
		{"EnqueueTask", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			_, _ = s.EnqueueTask(a.ID, TaskPrompt, "p", "user")
		}, []string{"task.enqueued"}},
		{"ClaimNextTask", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			_, _ = s.EnqueueTask(a.ID, TaskPrompt, "p", "user")
			s.OnEvent = recorder(s)
			_, _ = s.ClaimNextTask(a.ID)
		}, []string{"task.claimed"}},
		{"ClaimTask", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			tk, _ := s.EnqueueTask(a.ID, TaskPrompt, "p", "user")
			s.OnEvent = recorder(s)
			_, _ = s.ClaimTask(a.ID, tk.ID)
		}, []string{"task.claimed"}},
		{"FinishTask", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			_, _ = s.EnqueueTask(a.ID, TaskPrompt, "p", "user")
			tk, _ := s.ClaimNextTask(a.ID)
			s.OnEvent = recorder(s)
			_ = s.FinishTask(tk.ID, TaskDelivered, "")
		}, []string{"task.finished"}},
		{"EndInboxReply", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromAgent, SourceID: a.ID, Reason: "r", Title: "t", Body: "?"})
			_, tk, _ := s.RespondAndPark(it.ID, VerbRespond, "reply")
			s.OnEvent = recorder(s)
			_ = s.EndInboxReply(tk.ID, TaskFailed, "failed", "Send again.")
		}, []string{"task.finished", "inbox.updated"}},
		{"CreateAutomation", func(s *Store) {
			_, _, _ = s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
		}, []string{"automation.created"}},
		{"UpdateAutomation", func(s *Store) {
			a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
			s.OnEvent = recorder(s)
			off := false
			_, _ = s.UpdateAutomation(a.ID, AutomationPatch{Enabled: &off})
		}, []string{"automation.updated"}},
		{"SetAutomationWebhook", func(s *Store) {
			a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
			s.OnEvent = recorder(s)
			_, _ = s.SetAutomationWebhook(a.ID, true)
		}, []string{"automation.updated"}},
		{"DeleteAutomation", func(s *Store) {
			a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
			s.OnEvent = recorder(s)
			_ = s.DeleteAutomation(a.ID)
		}, []string{"automation.deleted"}},
		{"CreateRun + FinishRun", func(s *Store) {
			a, _, _ := s.CreateAutomation(AutomationParams{Name: "a", Action: AutomationStart, Prompt: "p", Cron: "0 9 * * *"})
			s.OnEvent = recorder(s)
			r, _ := s.CreateRun(a.ID, TriggerManual, RunRunning, "")
			_ = s.SetRunSession(r.ID, "/tmp/x.jsonl")
			_ = s.FinishRun(r.ID, RunDone, "", 0.1)
			_ = s.FinishRun(r.ID, RunDone, "", 0.1) // no-op: no second event
		}, []string{"run.created", "run.updated", "run.finished"}},
		{"CreatePin", func(s *Store) { _, _ = s.CreatePin("t", nil, "b") }, []string{"pin.created"}},
		{"CreateSession + RevokeSession", func(s *Store) {
			sess, _, _ := s.CreateSession(SessionBrowser, "", "x", "", 0)
			_ = s.RevokeSession(sess.ID)
		}, []string{"session.created", "session.revoked"}},
		{"RotateSessionSecret", func(s *Store) {
			sess, _, _ := s.CreateSession(SessionBrowser, "", "x", "", 0)
			s.OnEvent = recorder(s)
			_, _, _ = s.RotateSessionSecret(sess.ID, time.Hour)
		}, []string{"session.rotated"}},
		{"CreatePairing + ConsumePairing", func(s *Store) {
			code, _, _ := s.CreatePairing("", time.Minute)
			_ = s.ConsumePairing(code)
		}, []string{"pairing.created", "pairing.used"}},
		{"SetSetting", func(s *Store) { _ = s.SetSetting("k", "v") }, []string{"setting.updated"}},
		{"BeginDockerOperation", func(s *Store) {
			_, _, _ = s.BeginDockerOperation(DockerOperation{RequestKey: "request-123", Endpoint: "unix:///tmp/a", ContainerID: "a", Action: "start"})
		}, []string{"docker.operation"}},
		{"FinishDockerOperation", func(s *Store) {
			op, _, _ := s.BeginDockerOperation(DockerOperation{RequestKey: "request-123", Endpoint: "unix:///tmp/a", ContainerID: "a", Action: "start"})
			s.OnEvent = recorder(s)
			_ = s.FinishDockerOperation(op.ID, "succeeded", "verified")
		}, []string{"docker.operation"}},
		{"RecoverDockerOperations", func(s *Store) {
			_, _, _ = s.BeginDockerOperation(DockerOperation{RequestKey: "request-123", Endpoint: "unix:///tmp/a", ContainerID: "a", Action: "start"})
			s.OnEvent = recorder(s)
			_ = s.RecoverDockerOperations()
		}, []string{"docker.operation"}},
		{"CreateDockerPlan", func(s *Store) { _, _ = s.CreateDockerPlan(DockerPlan{Input: json.RawMessage(`{}`)}) }, []string{"docker.plan"}},
		{"RequestDockerReview", func(s *Store) {
			p, _ := s.CreateDockerPlan(DockerPlan{Input: json.RawMessage(`{}`)})
			s.OnEvent = recorder(s)
			_, _ = s.RequestDockerReview(p.ID)
		}, []string{"inbox.created", "docker.plan"}},
		{"BeginDockerJob", func(s *Store) { _, _, _ = s.BeginDockerJob(testDockerJob("request-job", "plan", "a")) }, []string{"docker.job"}},
		{"UpdateDockerJob", func(s *Store) {
			j, _, _ := s.BeginDockerJob(testDockerJob("request-job", "plan", "a"))
			s.OnEvent = recorder(s)
			j.State = "succeeded"
			_ = s.UpdateDockerJob(j)
		}, []string{"docker.job"}},
		{"RecoverDockerJobs", func(s *Store) {
			_, _, _ = s.BeginDockerJob(testDockerJob("request-job", "plan", "a"))
			s.OnEvent = recorder(s)
			_ = s.RecoverDockerJobs()
		}, []string{"docker.job"}},
		{"SaveDockerMonitor", func(s *Store) { _, _ = s.SaveDockerMonitor(DefaultDockerMonitor("unix:///tmp/qa", "demo")) }, []string{"docker.monitor"}},
		{"RecordDockerHealth", func(s *Store) {
			m, _ := s.SaveDockerMonitor(DefaultDockerMonitor("unix:///tmp/qa", "demo"))
			s.OnEvent = recorder(s)
			_ = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{}`), nil, time.Now())
		}, []string{"docker.health"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, err := Open(filepath.Join(t.TempDir(), "picode.db"))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Close()
			s.OnEvent = recorder(s)
			c.run(s)
			got := seen[s]
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("events = %v, want %v", got, c.want)
			}
		})
	}
}

var seen = map[*Store][]string{}

// recorder resets the per-store log and records every announced type,
// checking each event is already readable (announced after the write).
func recorder(s *Store) func(Event) {
	seen[s] = nil
	return func(ev Event) {
		got, _ := s.ListEventsSince(ev.ID-1, 1)
		if len(got) != 1 || got[0].ID != ev.ID {
			seen[s] = append(seen[s], ev.Type+"(not durable)")
			return
		}
		seen[s] = append(seen[s], ev.Type)
	}
}

func TestEventsCursorAndRetention(t *testing.T) {
	s := openTest(t)
	var got []Event
	s.OnEvent = func(ev Event) { got = append(got, ev) }
	_ = s.SetSetting("a", "1")
	_ = s.SetSetting("b", "2")
	if len(got) != 2 || got[1].ID != got[0].ID+1 {
		t.Fatalf("ids: %+v", got)
	}
	var d map[string]string
	_ = json.Unmarshal(got[0].Data, &d)
	if d["key"] != "a" {
		t.Fatalf("data = %s", got[0].Data)
	}
	since, _ := s.ListEventsSince(got[0].ID, 10)
	if len(since) != 1 || since[0].ID != got[1].ID {
		t.Fatalf("since = %+v", since)
	}
	latest, _ := s.LatestEventID()
	oldest, _ := s.OldestEventID()
	if latest != got[1].ID || oldest != got[0].ID {
		t.Fatalf("latest %d oldest %d", latest, oldest)
	}
	if n, _ := s.PruneEvents(timeNowPlusHour()); n != 2 {
		t.Fatalf("pruned %d", n)
	}
	if oldest, _ := s.OldestEventID(); oldest != 0 {
		t.Fatalf("oldest after prune = %d", oldest)
	}
}

// Events appended inside a transaction are announced only on commit,
// and dropped on rollback.
func TestTxEventsAnnounceOnCommitOnly(t *testing.T) {
	s := openTest(t)
	var got []string
	s.OnEvent = func(ev Event) { got = append(got, ev.Type) }
	tx, _ := s.db.Begin()
	_ = s.AppendEventTx(tx, "x.pending", nil, nil, nil)
	if len(got) != 0 {
		t.Fatal("announced before commit")
	}
	s.rollback(tx)
	if len(got) != 0 {
		t.Fatal("announced after rollback")
	}
	tx, _ = s.db.Begin()
	_ = s.AppendEventTx(tx, "x.committed", nil, nil, nil)
	_ = s.commit(tx)
	if len(got) != 1 || got[0] != "x.committed" {
		t.Fatalf("got %v", got)
	}
}

func timeNowPlusHour() time.Time { return time.Now().Add(time.Hour) }
