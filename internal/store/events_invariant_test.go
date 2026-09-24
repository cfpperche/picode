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
		{"ApplyMission", func(s *Store) {
			v, _ := missionFixture(t, s)
			s.OnEvent = recorder(s)
			missionChange(t, s, v, "report", func(m *MissionMutation) { m.Note = "Update" }, MissionObservation{})
		}, []string{"mission.changed"}},
		{"ApplyDelivery/register", func(s *Store) { s.OnEvent = recorder(s); deliveryFixture(t, s) }, []string{"delivery.changed"}},
		{"ApplyDelivery/update", func(s *Store) {
			d := deliveryFixture(t, s)
			s.OnEvent = recorder(s)
			_, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "update", RequestID: "u", ID: d.ID, ExpectedVersion: 1, Title: d.Title, Branch: d.Branch, Revision: d.Revision, Target: d.Target})
			if err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
		{"ApplyDelivery/review", func(s *Store) {
			d := deliveryFixture(t, s)
			s.OnEvent = recorder(s)
			_, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "request-review", RequestID: "r", ID: d.ID, ExpectedVersion: 1})
			if err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
		{"ApplyDelivery/withdraw", func(s *Store) {
			d := deliveryFixture(t, s)
			s.OnEvent = recorder(s)
			_, err := s.ApplyDelivery("repo", "agent", DeliveryMutation{Action: "withdraw-review", RequestID: "w", ID: d.ID, ExpectedVersion: 1})
			if err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
		{"EnsureAgentTerminal", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "pi", "")
			s.OnEvent = recorder(s)
			if _, err := s.EnsureAgentTerminal(a.ID, proj); err != nil {
				t.Fatal(err)
			}
			if _, err := s.EnsureAgentTerminal(a.ID, proj); err != nil {
				t.Fatal(err)
			}
		}, []string{"terminal.created", "terminal.launch", "agent.updated"}},
		{"SetPeerParticipants", func(s *Store) {
			p, _, _, _ := peerFixture(t, s)
			s.OnEvent = recorder(s)
			_ = s.SetPeerParticipants(p.WorkspaceID, []PeerSelection{{Kind: p.Kind, OwnerID: p.OwnerID, Enabled: true}})
		}, []string{"peer.participant"}},
		{"EnsureParticipantPeer", func(s *Store) {
			p, _, _, _ := peerFixture(t, s)
			_ = s.RevokePeer(p.ID)
			_ = s.SetPeerParticipants(p.WorkspaceID, []PeerSelection{{Kind: p.Kind, OwnerID: p.OwnerID, Enabled: true}})
			v, _ := s.PeerParticipant(p.Kind, p.OwnerID)
			s.OnEvent = recorder(s)
			_, _, _ = s.EnsureParticipantPeer(v, p.SessionKey)
		}, []string{"peer.connection"}},
		{"SetPeerPreparation", func(s *Store) {
			p, _, _, _ := peerFixture(t, s)
			_ = s.SetPeerParticipants(p.WorkspaceID, []PeerSelection{{Kind: p.Kind, OwnerID: p.OwnerID, Enabled: true}})
			v, _ := s.PeerParticipant(p.Kind, p.OwnerID)
			s.OnEvent = recorder(s)
			_, _ = s.SetPeerPreparation(v, p.SessionKey, "connected", "", p.ID)
		}, []string{"peer.participant"}},
		{"CreatePeerCheck", func(s *Store) {
			p, _, q, _ := peerFixture(t, s)
			s.OnEvent = recorder(s)
			_, _ = s.CreatePeerCheck(p.WorkspaceID, p.ID, q.ID)
		}, []string{"peer.check"}},
		{"SetPeerCheckPhase", func(s *Store) {
			p, _, q, _ := peerFixture(t, s)
			c, _ := s.CreatePeerCheck(p.WorkspaceID, p.ID, q.ID)
			s.OnEvent = recorder(s)
			_, _ = s.SetPeerCheckPhase(c.ID, "pending", "attempted")
		}, []string{"peer.check"}},

		{"EnablePeer", func(s *Store) {
			p, _, _, _ := peerFixture(t, s)
			s.OnEvent = recorder(s)
			_, _, _ = s.EnablePeer(p.Kind, p.PeerOwner.OwnerID, p.SessionKey)
		}, []string{"peer.connection"}},
		{"RevokePeer", func(s *Store) { p, _, _, _ := peerFixture(t, s); s.OnEvent = recorder(s); _ = s.RevokePeer(p.ID) }, []string{"peer.connection"}},
		{"SendPeerMessage", func(s *Store) {
			_, token, q, _ := peerFixture(t, s)
			s.OnEvent = recorder(s)
			_, _ = s.SendPeerMessage(token, q.ID, "retry", "hi", "")
			_, _ = s.SendPeerMessage(token, q.ID, "retry", "hi", "")
		}, []string{"peer.message"}},
		{"AckPeerMessages", func(s *Store) {
			_, token, q, qt := peerFixture(t, s)
			m, _ := s.SendPeerMessage(token, q.ID, "retry", "hi", "")
			s.OnEvent = recorder(s)
			_ = s.AckPeerMessages(qt, []string{m.ID})
			_ = s.AckPeerMessages(qt, []string{m.ID})
		}, []string{"peer.ack"}},
		{"SetPeerAttention", func(s *Store) {
			_, token, q, _ := peerFixture(t, s)
			m, _ := s.SendPeerMessage(token, q.ID, "notice", "hi", "")
			s.OnEvent = recorder(s)
			_, _ = s.SetPeerAttention(m.ID, q.ID, "pending", "attempted")
			_, _ = s.SetPeerAttention(m.ID, q.ID, "pending", "attempted")
			_, _ = s.SetPeerAttention(m.ID, q.ID, "attempted", "notified")
		}, []string{"peer.attention", "peer.attention"}},
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
		{"SetTerminalLastSession", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "cli", proj)
			_ = s.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{})
			s.OnEvent = recorder(s)
			ls := TerminalLastSession{CLI: "claude-code", SessionID: "s1", UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
			_ = s.SetTerminalLastSession(tm.ID, ls)
			// Same session+updatedAt pins again without a second event.
			_ = s.SetTerminalLastSession(tm.ID, ls)
		}, []string{"terminal.last_session"}},
		{"SetCLIConfig", func(s *Store) { _ = s.SetCLIConfig("codex", clilaunch.Config{}) }, []string{"cli.updated"}},
		{"AddWebhook", func(s *Store) {
			s.OnEvent = recorder(s)
			_, _ = s.AddWebhook("https://example.com/hook", []string{"agent."})
		}, []string{"webhook.created"}},
		{"UpdateWebhook", func(s *Store) {
			w, _ := s.AddWebhook("https://example.com/hook", []string{"agent."})
			s.OnEvent = recorder(s)
			w.Enabled = false
			_, _ = s.UpdateWebhook(w)
		}, []string{"webhook.updated"}},
		{"DeleteWebhook", func(s *Store) {
			w, _ := s.AddWebhook("https://example.com/hook", []string{"agent."})
			s.OnEvent = recorder(s)
			_ = s.DeleteWebhook(w.ID)
		}, []string{"webhook.deleted"}},
		{"RotateWebhookSecret", func(s *Store) {
			w, _ := s.AddWebhook("https://example.com/hook", []string{"agent."})
			s.OnEvent = recorder(s)
			_, _ = s.RotateWebhookSecret(w.ID, w.Revision)
		}, []string{"webhook.updated"}},
		{"SaveWebhookProgress", func(s *Store) {
			w, _ := s.AddWebhook("https://example.com/hook", []string{"agent."})
			s.OnEvent = recorder(s)
			after := w
			after.LastStatus = "delivered"
			after.LastAttemptAt = nowUTC()
			_ = s.SaveWebhookProgress(w, after)
		}, []string{"webhook.delivery"}},
		{"SaveWebhookProgress/scan-only", func(s *Store) {
			w, _ := s.AddWebhook("https://example.com/hook", []string{"agent."})
			s.OnEvent = recorder(s)
			after := w
			after.Cursor++
			_ = s.SaveWebhookProgress(w, after)
		}, nil},
		{"ImportCLIConfigs", func(s *Store) { _ = s.ImportCLIConfigs(map[string]bool{"pi": true}) }, cliUpdatedN(len(clilaunch.Catalog()))},
		{"SeedCatalogIntegrationDefaults", func(s *Store) {
			_ = s.SetCLIConfig("opencode", clilaunch.Config{})
			s.OnEvent = recorder(s)
			_ = s.SeedCatalogIntegrationDefaults(nil)
		}, []string{"cli.updated", "setting.updated"}},
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
		{"AddSessionHandoff", func(s *Store) {
			_, _ = s.AddSessionHandoff(SessionHandoff{SourceCLI: "claude-code", SourceID: "cc-1", TargetCLI: "codex", Mode: "native", Window: "recent"})
		}, []string{"session.handoff"}},
		{"AddWorkspace", func(s *Store) { _, _ = s.AddWorkspace("W", proj) }, []string{"workspace.added"}},
		{"RenameWorkspace", func(s *Store) {
			w, _ := s.AddWorkspace("W", proj)
			s.OnEvent = recorder(s)
			_, _ = s.RenameWorkspace(w.ID, "Renamed")
		}, []string{"workspace.updated"}},
		{"ReorderWorkspaces", func(s *Store) {
			a, _ := s.AddWorkspace("A", proj)
			other := filepath.Join(dir, "other-ws")
			_ = os.MkdirAll(other, 0o755)
			b, _ := s.AddWorkspace("B", other)
			s.OnEvent = recorder(s)
			_ = s.ReorderWorkspaces([]string{b.ID, a.ID})
		}, []string{"workspace.reordered"}},
		{"ReorderAgents", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			b, _ := s.AddAgent(FreeWorkspaceID, "b", "")
			s.OnEvent = recorder(s)
			_ = s.ReorderAgents(FreeWorkspaceID, []string{b.ID, a.ID})
		}, []string{"agent.reordered"}},
		{"ReorderTerminals", func(s *Store) {
			a, _ := s.CreateTerminalIn("", "a", proj)
			b, _ := s.CreateTerminalIn("", "b", proj)
			s.OnEvent = recorder(s)
			_ = s.ReorderTerminals(FreeWorkspaceID, []string{b.ID, a.ID})
		}, []string{"terminal.reordered"}},
		{"RemoveWorkspace", func(s *Store) {
			w, _ := s.AddWorkspace("W", proj)
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_, _ = s.RemoveWorkspace(w.ID)
		}, []string{"workspace.deleted"}},
		{"AddAgent", func(s *Store) { _, _ = s.AddAgent(FreeWorkspaceID, "a", "") }, []string{"agent.added"}},
		{"AddAgentWithCLI", func(s *Store) { _, _ = s.AddAgentWithCLI(FreeWorkspaceID, "claude-code", "c", "") }, []string{"agent.added"}},
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
		{"SetAgentTurnSettled", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			_ = s.SetAgentRuntime(a.ID, StatusRunning)
			s.OnEvent = recorder(s)
			_ = s.SetAgentTurnSettled(a.ID)
		}, []string{"agent.settled"}},
		{"DeleteAgent", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			_ = s.DeleteAgent(a.ID)
		}, []string{"agent.deleted"}},
		{"RemoveAgentWithExit", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			s.OnEvent = recorder(s)
			if _, err := s.RemoveAgentWithExit(a.ID, ExitInput{Origin: ExitFromDesktop, Asked: true, Label: ExitLabel{Outcome: ExitResolved}}); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.recorded", "agent.deleted"}},
		{"LabelAgentExit", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			ex, _ := s.RemoveAgentWithExit(a.ID, ExitInput{})
			s.OnEvent = recorder(s)
			if _, err := s.LabelAgentExit(ex.ID, ExitLabel{Outcome: ExitUnresolved, Reasons: []string{"stuck"}}); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.updated"}},
		{"MarkAgentExitUndone", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			ex, _ := s.RemoveAgentWithExit(a.ID, ExitInput{})
			s.OnEvent = recorder(s)
			if _, err := s.MarkAgentExitUndone(ex.ID, "back-1"); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.updated"}},
		{"ForgetAgentExit", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			ex, _ := s.RemoveAgentWithExit(a.ID, ExitInput{})
			s.OnEvent = recorder(s)
			if _, err := s.ForgetAgentExit(ex.ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.updated"}},
		{"DeleteAgentExit", func(s *Store) {
			a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
			ex, _ := s.RemoveAgentWithExit(a.ID, ExitInput{})
			s.OnEvent = recorder(s)
			if err := s.DeleteAgentExit(ex.ID); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.deleted"}},
		{"RemoveWorkspaceWithExits", func(s *Store) {
			w, _ := s.AddWorkspace("w", proj)
			_, _ = s.AddAgent(w.ID, "a", "")
			s.OnEvent = recorder(s)
			if _, _, err := s.RemoveWorkspaceWithExits(w.ID, ExitInput{Origin: ExitFromDesktop}); err != nil {
				t.Fatal(err)
			}
		}, []string{"agent_exit.recorded", "workspace.deleted"}},
		{"SetExitAskOn", func(s *Store) {
			s.OnEvent = recorder(s)
			if err := s.SetExitAskOn(false); err != nil {
				t.Fatal(err)
			}
		}, []string{"setting.updated"}},
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
		{"SetTerminalChecklist", func(s *Store) {
			tm, _ := s.CreateTerminalIn("", "t", proj)
			s.OnEvent = recorder(s)
			_, _ = s.SetTerminalChecklist(tm.ID, "s1", []ChecklistItem{{Text: "x", Status: "pending"}}, false)
		}, []string{"terminal.checklist"}},
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
		{"ReopenInboxItem", func(s *Store) {
			it, _ := s.CreateInboxItem(InboxItemParams{Kind: InboxQuestion, SourceKind: InboxFromTerminal, Reason: "r", Title: "t", Body: "b"})
			_, _ = s.RespondInboxItem(it.ID, VerbRespond, "yes")
			s.OnEvent = recorder(s)
			_, _ = s.ReopenInboxItem(it.ID, "not delivered")
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
			r, _ := s.CreateRun(RunParams{AutomationID: a.ID, Trigger: TriggerManual, Status: RunRunning})
			_ = s.SetRunSession(r.ID, "/tmp/x.jsonl")
			_ = s.FinishRun(r.ID, RunDone, "", 0.1)
			_ = s.FinishRun(r.ID, RunDone, "", 0.1) // no-op: no second event
		}, []string{"run.created", "run.updated", "run.finished"}},
		{"CreatePin", func(s *Store) { _, _ = s.CreatePin("t", nil, "b") }, []string{"pin.created"}},
		{"CreateSnip", func(s *Store) { _, _ = s.CreateSnip(SnipParams{Title: "t", Body: "b"}) }, []string{"snip.created"}},
		{"UpdateSnip", func(s *Store) {
			p, _ := s.CreateSnip(SnipParams{Title: "t", Body: "b"})
			s.OnEvent = recorder(s)
			_, _ = s.UpdateSnip(p.ID, SnipParams{Title: "t", Body: "c"}, "")
		}, []string{"snip.updated"}},
		{"DeleteSnip", func(s *Store) {
			p, _ := s.CreateSnip(SnipParams{Title: "t", Body: "b"})
			s.OnEvent = recorder(s)
			_ = s.DeleteSnip(p.ID)
		}, []string{"snip.deleted"}},
		{"SetSnipStarred", func(s *Store) {
			p, _ := s.CreateSnip(SnipParams{Title: "t", Body: "b"})
			s.OnEvent = recorder(s)
			_, _ = s.SetSnipStarred(p.ID, true)
		}, []string{"snip.updated"}},
		{"SetSnipArchived", func(s *Store) {
			p, _ := s.CreateSnip(SnipParams{Title: "t", Body: "b"})
			s.OnEvent = recorder(s)
			_, _ = s.SetSnipArchived(p.ID, true)
		}, []string{"snip.updated"}},
		{"CreateCanvas", func(s *Store) { _, _ = s.CreateCanvas("Ops") }, []string{"canvas.created"}},
		{"UpdateCanvas", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			s.OnEvent = recorder(s)
			name := "Ops board"
			_, _ = s.UpdateCanvas(m.ID, CanvasPatch{Name: &name}, "")
		}, []string{"canvas.updated"}},
		{"PatchCanvasLayout", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			p, _ := s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
			s.OnEvent = recorder(s)
			_, _ = s.PatchCanvasLayout(m.ID, []PanelPlacement{{ID: p.Panel.ID, X: 40, Y: 0, W: 32, H: 28}}, "")
		}, []string{"canvas.layout"}},
		{"AddCanvasPanel", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			s.OnEvent = recorder(s)
			_, _ = s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
		}, []string{"canvas.panel.added"}},
		{"RemoveCanvasPanel", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			p, _ := s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
			s.OnEvent = recorder(s)
			_ = s.RemoveCanvasPanel(m.ID, p.Panel.ID)
		}, []string{"canvas.panel.removed"}},
		{"AddCanvasEdge", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			a, _ := s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
			b, _ := s.AddCanvasPanel(m.ID, "agent", "a1", 40, 0, 32, 28)
			s.OnEvent = recorder(s)
			_, _ = s.AddCanvasEdge(m.ID, a.Panel.ID, b.Panel.ID)
			// The same pair backwards is the 409: no second row, no second event.
			_, _ = s.AddCanvasEdge(m.ID, b.Panel.ID, a.Panel.ID)
		}, []string{"canvas.edge.added"}},
		{"RemoveCanvasEdge", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			a, _ := s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
			b, _ := s.AddCanvasPanel(m.ID, "agent", "a1", 40, 0, 32, 28)
			e, _ := s.AddCanvasEdge(m.ID, a.Panel.ID, b.Panel.ID)
			s.OnEvent = recorder(s)
			_ = s.RemoveCanvasEdge(m.ID, e.Edge.ID)
			_ = s.RemoveCanvasEdge(m.ID, e.Edge.ID) // gone: no second event
		}, []string{"canvas.edge.removed"}},
		{"DeleteCanvas", func(s *Store) {
			m, _ := s.CreateCanvas("Ops")
			_, _ = s.AddCanvasPanel(m.ID, "terminal", "t1", 0, 0, 32, 28)
			s.OnEvent = recorder(s)
			_ = s.DeleteCanvas(m.ID)
		}, []string{"canvas.deleted"}},
		{"CreateSession + RevokeSession", func(s *Store) {
			sess, _, _ := s.CreateSession(SessionBrowser, "", "x", "", 0)
			_ = s.RevokeSession(sess.ID)
		}, []string{"session.created", "session.revoked"}},
		{"RotateSessionSecret", func(s *Store) {
			sess, _, _ := s.CreateSession(SessionBrowser, "", "x", "", 0)
			s.OnEvent = recorder(s)
			_, _, _ = s.RotateSessionSecret(sess.ID, time.Hour)
		}, []string{"session.rotated"}},
		{"RevokeStaleLoopbackSessions", func(s *Store) {
			sess, _, _ := s.CreateSession(SessionBrowser, "", "This machine · Headless browser", LoopbackMintIP, 0)
			_, _, _ = s.CreateSession(SessionBrowser, "dev-1", "iPhone", "100.64.0.7", 0) // paired: never picked
			ageSession(s, sess.ID, time.Hour)
			s.OnEvent = recorder(s)
			_, _ = s.RevokeStaleLoopbackSessions(time.Minute)
		}, []string{"session.revoked"}},
		{"CreatePairing + ConsumePairing", func(s *Store) {
			code, _, _ := s.CreatePairing("", time.Minute)
			_ = s.ConsumePairing(code)
		}, []string{"pairing.created", "pairing.used"}},
		{"SetSetting", func(s *Store) { _ = s.SetSetting("k", "v") }, []string{"setting.updated"}},
		{"SetBrowserPermission (new site)", func(s *Store) {
			_, _ = s.SetBrowserPermission("meet.example.com", "camera", "allow", false)
		}, []string{"browserpermission.updated"}},
		{"DeleteBrowserPermission", func(s *Store) {
			p, _ := s.SetBrowserPermission("meet.example.com", "camera", "allow", true)
			_ = s.DeleteBrowserPermission(p.ID)
		}, []string{"browserpermission.updated", "browserpermission.updated"}},
		{"PruneBrowserPermissions", func(s *Store) {
			_, _ = s.SetBrowserPermission("never-visited.test", "camera", "allow", false)
			_, _ = s.PruneBrowserPermissions(time.Now())
		}, []string{"browserpermission.updated", "browserpermission.updated"}},
		{"ClearBrowserPermissions", func(s *Store) {
			_, _ = s.SetBrowserPermission("meet.example.com", "camera", "allow", false)
			_ = s.ClearBrowserPermissions("camera")
		}, []string{"browserpermission.updated", "browserpermission.updated"}},
		{"CreateBrowserAnnotation + DeleteBrowserAnnotation", func(s *Store) {
			a, _ := s.CreateBrowserAnnotation(BrowserAnnotation{URL: "https://x.test/", Selector: ".a"})
			_ = s.DeleteBrowserAnnotation(a.ID)
		}, []string{"browserannotation.updated", "browserannotation.updated"}},
		{"AddBrowserDownload (start)", func(s *Store) {
			_, _ = s.AddBrowserDownload("https://files.example/a.zip", `C:\Users\me\Downloads\a.zip`, 10)
		}, []string{"browserdownload.updated"}},
		{"FinishBrowserDownload (outcome)", func(s *Store) {
			_, _ = s.AddBrowserDownload("https://files.example/a.zip", `C:\Users\me\Downloads\a.zip`, 10)
			_ = s.FinishBrowserDownload(`C:\Users\me\Downloads\a.zip`, "completed", 10)
		}, []string{"browserdownload.updated", "browserdownload.updated"}},
		{"DeleteBrowserDownload", func(s *Store) {
			d, _ := s.AddBrowserDownload("https://files.example/a.zip", `C:\Users\me\Downloads\a.zip`, 10)
			_ = s.DeleteBrowserDownload(d.ID)
		}, []string{"browserdownload.updated", "browserdownload.updated"}},
		{"ClearBrowserDownloads", func(s *Store) {
			_, _ = s.AddBrowserDownload("https://files.example/a.zip", `C:\Users\me\Downloads\a.zip`, 10)
			_ = s.ClearBrowserDownloads()
		}, []string{"browserdownload.updated", "browserdownload.updated"}},
		{"AddBrowserVisit (new url)", func(s *Store) { _, _ = s.AddBrowserVisit("https://example.com/a", "Example", false) }, []string{"browserhistory.updated"}},
		{"AddBrowserVisit (same url updates in place)", func(s *Store) {
			_, _ = s.AddBrowserVisit("https://example.com/b", "Example", true)
			_, _ = s.AddBrowserVisit("https://example.com/b", "Example", false)
		}, []string{"browserhistory.updated", "browserhistory.updated"}},
		{"DeleteBrowserVisit", func(s *Store) {
			v, _ := s.AddBrowserVisit("https://example.com/c", "", false)
			_ = s.DeleteBrowserVisit(v.ID)
		}, []string{"browserhistory.updated", "browserhistory.updated"}},
		{"ClearBrowserHistory", func(s *Store) {
			_, _ = s.AddBrowserVisit("https://example.com/d", "", false)
			_ = s.ClearBrowserHistory()
		}, []string{"browserhistory.updated", "browserhistory.updated"}},
		{"HideDevServer", func(s *Store) {
			_, _ = s.HideDevServer(5173, 4242, "900", `terminal "web"`, "vite")
		}, []string{"devserver.hidden"}},
		{"UnhideDevServer", func(s *Store) {
			h, _ := s.HideDevServer(5173, 4242, "900", `terminal "web"`, "vite")
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_ = s.UnhideDevServer(h.ID)
		}, []string{"devserver.hidden"}},
		{"PruneDevServerHides", func(s *Store) {
			live, _ := s.HideDevServer(5173, 4242, "900", "", "vite")
			_, _ = s.HideDevServer(3000, 4243, "901", "", "astro")
			s.OnEvent = nil
			s.OnEvent = recorder(s)
			_, _ = s.PruneDevServerHides([]int64{live.ID})
		}, []string{"devserver.hidden"}},
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
		{"BeginLlamaJob", func(s *Store) { _, _, _ = s.BeginLlamaJob(testLlamaJob("request", "model")) }, []string{"llama.job"}},
		{"BeginCLIJob", func(s *Store) { _, _, _ = s.BeginCLIJob(CLIJob{CLI: "pi", Action: "update", RequestKey: "req-cli"}) }, []string{"cli.job"}},
		{"BeginCLIJobInstall", func(s *Store) {
			_, _, _ = s.BeginCLIJob(CLIJob{CLI: "codex", Action: "install", RequestKey: "req-cli-3"})
		}, []string{"cli.job"}},
		{"UpdateCLIJob", func(s *Store) {
			j, _, _ := s.BeginCLIJob(CLIJob{CLI: "pi", Action: "update", RequestKey: "req-cli-2"})
			s.OnEvent = recorder(s)
			j.State = "running"
			_, _ = s.UpdateCLIJob(j)
		}, []string{"cli.job"}},
		{"SaveLlamaService", func(s *Store) { _, _ = s.SaveLlamaService(json.RawMessage(`{}`), 0) }, []string{"llama.service"}},
		{"UpdateLlamaJob", func(s *Store) {
			j, _, _ := s.BeginLlamaJob(testLlamaJob("request", "model"))
			s.OnEvent = recorder(s)
			j.State = "succeeded"
			_, _ = s.UpdateLlamaJob(j)
		}, []string{"llama.job"}},
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
		{"CreateWebapp", func(s *Store) {
			_, _ = s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com", Icon: []byte("png"), IconMime: "image/png"})
		}, []string{"webapp.installed"}},
		{"UpdateWebappName", func(s *Store) {
			app, _ := s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com", Icon: nil, IconMime: ""})
			s.OnEvent = recorder(s)
			_, _ = s.UpdateWebappName(app.ID, "Renamed")
		}, []string{"webapp.updated"}},
		{"UpdateWebappMetadata", func(s *Store) {
			app, _ := s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com", Icon: nil, IconMime: ""})
			s.OnEvent = recorder(s)
			_, _ = s.UpdateWebappMetadata(app.ID, WebappInput{StartURL: "https://example.com/start"})
		}, []string{"webapp.updated"}},
		{"DeleteWebapp", func(s *Store) {
			app, _ := s.CreateWebapp(WebappInput{Name: "Example", URL: "https://example.com", Icon: nil, IconMime: ""})
			s.OnEvent = recorder(s)
			_ = s.DeleteWebapp(app.ID)
		}, []string{"webapp.removed"}},
		{"RecordDockerHealth", func(s *Store) {
			m, _ := s.SaveDockerMonitor(DefaultDockerMonitor("unix:///tmp/qa", "demo"))
			s.OnEvent = recorder(s)
			_ = s.RecordDockerHealth(m.Endpoint, m.Project, m.Revision, json.RawMessage(`{}`), nil, time.Now())
		}, []string{"docker.health"}},
		{"ApplyQueueMutation/enqueue", func(s *Store) {
			d := deliveryFixture(t, s)
			s.OnEvent = recorder(s)
			if _, err := s.ApplyQueueMutation("repo", "agent", QueueMutation{Action: "enqueue", RequestID: "q-enqueue",
				DeliveryID: d.ID, Revision: d.Revision, Target: d.Target}); err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
		{"PutIntegrationSettings", func(s *Store) {
			s.OnEvent = recorder(s)
			if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true, Checks: []string{"make ci"}}); err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
		{"DeleteIntegrationSettings", func(s *Store) {
			if _, err := s.PutIntegrationSettings("ws1", IntegrationSettingsMutation{FFOnly: true}); err != nil {
				t.Fatal(err)
			}
			s.OnEvent = recorder(s)
			if err := s.DeleteIntegrationSettings("ws1"); err != nil {
				t.Fatal(err)
			}
		}, []string{"delivery.changed"}},
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

func cliUpdatedN(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "cli.updated"
	}
	return out
}
