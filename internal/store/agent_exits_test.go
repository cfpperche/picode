package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cfpperche/picode/internal/clilaunch"
)

func ptr[T any](v T) *T { return &v }

func TestRemoveAgentWithExitFreezesTheAgent(t *testing.T) {
	s := openTest(t)
	ws, err := s.AddWorkspace("atlas", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.AddAgent(ws.ID, "fixer", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.UpdateAgent(a.ID, AgentPatch{Provider: ptr("anthropic"), Model: ptr("claude-sonnet-5"), Thinking: ptr("high"),
		OpMode: ptr(OpModeReadonly), ExtraPrompt: ptr("Keep diffs small."), SessionPath: ptr("/data/sessions/fixer.jsonl")}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetChecklist(a.ID, "", []ChecklistItem{{Text: "a", Status: "completed"}, {Text: "b", Status: "in-progress"}, {Text: "c", Status: "pending"}}, false); err != nil {
		t.Fatal(err)
	}
	for _, blocking := range []bool{true, false} {
		if _, err := s.CreateInboxItem(InboxItemParams{Kind: "fyi", SourceKind: InboxFromAgent, SourceID: a.ID, Reason: "done", Title: "t", Blocking: blocking}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := s.NoteAgentTurn(a.ID); err != nil {
			t.Fatal(err)
		}
	}

	ex, err := s.RemoveAgentWithExit(a.ID, ExitInput{
		Origin: ExitFromDesktop, Asked: true, SessionsPurged: true,
		Label: ExitLabel{Outcome: ExitPartial, Reasons: []string{"stuck", "setup", "stuck"}, Note: "  looped on the migration  "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAgent(a.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("agent still there: %v", err)
	}
	got, err := s.GetAgentExit(ex.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AgentID != a.ID || got.AgentName != "fixer" || got.WorkspaceID != ws.ID || got.WorkspaceName != "atlas" || got.CLI != CLIPi {
		t.Fatalf("identity = %+v", got)
	}
	if got.Provider != "anthropic" || got.Model != "claude-sonnet-5" || got.Config.Thinking != "high" || got.Config.OpMode != OpModeReadonly || got.Config.ExtraPrompt != "Keep diffs small." {
		t.Fatalf("setup = %+v / %+v", got, got.Config)
	}
	if got.Turns == nil || *got.Turns != 2 || got.FirstWorkedAt == nil || got.LastWorkedAt == nil {
		t.Fatalf("activity = turns %v first %v last %v", got.Turns, got.FirstWorkedAt, got.LastWorkedAt)
	}
	if got.Signals.InboxItems != 2 || got.Signals.InboxBlocking != 1 || got.Signals.ChecklistDone != 1 || got.Signals.ChecklistTotal != 3 {
		t.Fatalf("signals = %+v", got.Signals)
	}
	if got.Sessions.PiSessionPath != "/data/sessions/fixer.jsonl" || !got.SessionsPurged || got.WorkPurged {
		t.Fatalf("sessions = %+v purged %v/%v", got.Sessions, got.SessionsPurged, got.WorkPurged)
	}
	if got.Outcome != ExitPartial || strings.Join(got.Reasons, ",") != "setup,stuck" || got.Note != "looped on the migration" || got.LabeledAt == nil {
		t.Fatalf("answer = %q %v %q %v", got.Outcome, got.Reasons, got.Note, got.LabeledAt)
	}
	if !got.Asked || got.AskSkip != "" || got.Origin != ExitFromDesktop || got.Taxonomy != ExitTaxonomyVersion || got.UndoneAt != nil {
		t.Fatalf("question = %+v", got)
	}
	var n int
	_ = s.db.QueryRow(`SELECT COUNT(1) FROM agent_checklists WHERE agent_id = ?`, a.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("checklist rows left: %d", n)
	}
}

func TestRemoveAgentWithExitKeepsNoEnvValue(t *testing.T) {
	s := openTest(t)
	a, err := s.AddAgentWithCLI(FreeWorkspaceID, "codex", "reviewer", "")
	if err != nil {
		t.Fatal(err)
	}
	term, err := s.CreateTerminalIn(FreeWorkspaceID, "reviewer", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if a, err = s.UpdateAgent(a.ID, AgentPatch{TerminalID: &term.ID}); err != nil {
		t.Fatal(err)
	}
	args := []string{"--model", "gpt-5.5"}
	secret := "sk-live-do-not-copy"
	if err := s.SetTerminalLaunch(*a.TerminalID, "codex", clilaunch.Overrides{Args: &args, Env: map[string]*string{"OPENAI_API_KEY": &secret, "HTTP_PROXY": nil}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetTerminalLastSession(*a.TerminalID, TerminalLastSession{CLI: "codex", SessionID: "sess-9", Path: "/home/x/.codex/sessions/9.jsonl", UpdatedAt: "2026-09-23T10:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	ex, err := s.RemoveAgentWithExit(a.ID, ExitInput{Origin: "something else"})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(ex)
	if strings.Contains(string(raw), secret) {
		t.Fatalf("the exit holds an environment value: %s", raw)
	}
	l := ex.Config.Launch
	if l == nil || !l.HasOverrides || strings.Join(l.EnvKeys, ",") != "OPENAI_API_KEY" || strings.Join(l.RemovedEnv, ",") != "HTTP_PROXY" || strings.Join(l.Args, " ") != "--model gpt-5.5" {
		t.Fatalf("launch = %+v", l)
	}
	if ex.Sessions.CLISessionID != "sess-9" || ex.Sessions.CLISessionPath != "/home/x/.codex/sessions/9.jsonl" {
		t.Fatalf("sessions = %+v", ex.Sessions)
	}
	if ex.Origin != ExitFromAPI || ex.Asked || ex.Outcome != "" || ex.LabeledAt != nil {
		t.Fatalf("unasked exit = %+v", ex)
	}
	if _, err := s.GetTerminal(*a.TerminalID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("bound terminal survived: %v", err)
	}
}

func TestRemoveAgentWithExitRefusesABadAnswerAndKeepsTheAgent(t *testing.T) {
	s := openTest(t)
	a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
	if _, err := s.RemoveAgentWithExit(a.ID, ExitInput{Label: ExitLabel{Outcome: "meh"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.GetAgent(a.ID); err != nil {
		t.Fatalf("a refused answer removed the agent: %v", err)
	}
	if _, err := s.RemoveAgentWithExit("nope", ExitInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing agent: %v", err)
	}
}

func TestDeleteAgentWritesNoExit(t *testing.T) {
	s := openTest(t)
	a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
	if err := s.DeleteAgent(a.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListAgentExits(ExitFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("a rollback-style delete wrote an exit: %v %v", list, err)
	}
}

func TestNoteAgentTurnMeasuresOnlyAgentsBornCounted(t *testing.T) {
	s := openTest(t)
	fresh, _ := s.AddAgent(FreeWorkspaceID, "fresh", "")
	old, _ := s.AddAgent(FreeWorkspaceID, "old", "")
	// What the migration does to every row that existed before it.
	if _, err := s.db.Exec(`UPDATE agents SET turns = NULL WHERE id = ?`, old.ID); err != nil {
		t.Fatal(err)
	}
	act, _ := s.AgentActivityOf(fresh.ID)
	if act.Turns == nil || *act.Turns != 0 || act.Worked() {
		t.Fatalf("fresh agent = %+v", act)
	}
	for _, id := range []string{fresh.ID, old.ID} {
		if err := s.NoteAgentTurn(id); err != nil {
			t.Fatal(err)
		}
	}
	act, _ = s.AgentActivityOf(fresh.ID)
	if act.Turns == nil || *act.Turns != 1 || act.FirstWorkedAt == nil || !act.Worked() {
		t.Fatalf("fresh after a turn = %+v", act)
	}
	act, _ = s.AgentActivityOf(old.ID)
	if act.Turns != nil || act.LastWorkedAt == nil || !act.Worked() {
		t.Fatalf("old after a turn = %+v", act)
	}
	if err := s.NoteAgentTurn("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing agent: %v", err)
	}
}

// ADR-0194's decision table, one row per case.
func TestExitAskDecision(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	old := now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	young := now.Add(-20 * time.Second).Format(time.RFC3339Nano)
	zero, one := int64(0), int64(1)
	worked := "2026-09-23T11:00:00Z"
	cases := []struct {
		name string
		on   bool
		act  AgentActivity
		born string
		ask  bool
		skip string
	}{
		{"off wins over everything", false, AgentActivity{Turns: &one}, old, false, ExitSkipOff},
		{"never worked", true, AgentActivity{Turns: &zero}, old, false, ExitSkipIdle},
		{"worked, too brief", true, AgentActivity{Turns: &one}, young, false, ExitSkipBrief},
		{"worked, long enough", true, AgentActivity{Turns: &one}, old, true, ""},
		{"not measured, long enough", true, AgentActivity{}, old, true, ""},
		{"not measured, too brief", true, AgentActivity{}, young, false, ExitSkipBrief},
		{"zero turns but a worked time", true, AgentActivity{Turns: &zero, LastWorkedAt: &worked}, old, true, ""},
		{"never worked and brief says idle first", true, AgentActivity{Turns: &zero}, young, false, ExitSkipIdle},
	}
	for _, c := range cases {
		ask, skip := ExitAskDecision(c.on, c.act, c.born, now)
		if ask != c.ask || skip != c.skip {
			t.Errorf("%s: got (%v, %q), want (%v, %q)", c.name, ask, skip, c.ask, c.skip)
		}
	}
}

func TestNormalizeExitLabel(t *testing.T) {
	long := strings.Repeat("é", maxExitNote+1)
	cases := []struct {
		name    string
		in      ExitLabel
		reasons string
		wantErr string
	}{
		{"empty is no answer", ExitLabel{}, "", ""},
		{"reasons in taxonomy order, once", ExitLabel{Outcome: ExitUnresolved, Reasons: []string{"switched", "setup", "switched"}}, "setup,switched", ""},
		{"reasons need partial or unresolved", ExitLabel{Outcome: ExitResolved, Reasons: []string{"stuck"}}, "", "reasons need"},
		{"reasons without an outcome", ExitLabel{Reasons: []string{"stuck"}}, "", "reasons need"},
		{"unknown outcome", ExitLabel{Outcome: "great"}, "", "unknown outcome"},
		{"unknown reason", ExitLabel{Outcome: ExitPartial, Reasons: []string{"vibes"}}, "", "unknown reason"},
		{"note too long", ExitLabel{Outcome: ExitTrial, Note: long}, "", "longer than"},
	}
	for _, c := range cases {
		got, err := normalizeExitLabel(c.in)
		if c.wantErr != "" {
			if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), c.wantErr) {
				t.Errorf("%s: err = %v", c.name, err)
			}
			continue
		}
		if err != nil || strings.Join(got.Reasons, ",") != c.reasons {
			t.Errorf("%s: got %v, %v", c.name, got.Reasons, err)
		}
	}
}

func TestExitAskSetting(t *testing.T) {
	s := openTest(t)
	if on, err := s.ExitAskOn(); err != nil || !on {
		t.Fatalf("default = %v %v", on, err)
	}
	_ = s.SetExitAskOn(false)
	if on, _ := s.ExitAskOn(); on {
		t.Fatal("still on")
	}
	_ = s.SetExitAskOn(true)
	if on, _ := s.ExitAskOn(); !on {
		t.Fatal("still off")
	}
}

func TestAgentExitCatalog(t *testing.T) {
	s := openTest(t)
	ws, _ := s.AddWorkspace("atlas", t.TempDir())
	remove := func(ws, cli, name string, in ExitInput) AgentExit {
		t.Helper()
		a, err := s.AddAgentWithCLI(ws, cli, name, "")
		if err != nil {
			t.Fatal(err)
		}
		ex, err := s.RemoveAgentWithExit(a.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		return ex
	}
	good := remove(ws.ID, "claude-code", "one", ExitInput{Asked: true, Label: ExitLabel{Outcome: ExitResolved}})
	bad := remove(ws.ID, "codex", "two", ExitInput{Asked: true, Label: ExitLabel{Outcome: ExitUnresolved, Reasons: []string{"stuck", "slow_costly"}}})
	silent := remove(FreeWorkspaceID, "codex", "three", ExitInput{AskSkip: ExitSkipClient})
	undone := remove(ws.ID, "codex", "four", ExitInput{Asked: true, Label: ExitLabel{Outcome: ExitUnresolved, Reasons: []string{"stuck"}}})
	if _, err := s.MarkAgentExitUndone(undone.ID, "four-back"); err != nil {
		t.Fatal(err)
	}

	all, _ := s.ListAgentExits(ExitFilter{})
	if len(all) != 3 {
		t.Fatalf("undone exits must leave the list: %d", len(all))
	}
	if all[0].ID != silent.ID {
		t.Fatalf("newest first: %s", all[0].ID)
	}
	if got, _ := s.ListAgentExits(ExitFilter{WorkspaceID: ws.ID}); len(got) != 2 {
		t.Fatalf("workspace filter: %d", len(got))
	}
	if got, _ := s.ListAgentExits(ExitFilter{CLI: "codex", Outcome: ExitOutcomeUnanswered}); len(got) != 1 || got[0].ID != silent.ID {
		t.Fatalf("unanswered codex: %v", got)
	}
	if got, _ := s.ListAgentExits(ExitFilter{Limit: 1, Before: all[0].RemovedAt}); len(got) != 1 || got[0].ID != all[1].ID {
		t.Fatalf("page cursor: %v", got)
	}

	sum, err := s.AgentExitSummary(ExitFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 3 || sum.Asked != 2 || sum.Answered != 2 || sum.AskedAnswered != 2 || sum.Outcomes[ExitResolved] != 1 || sum.Outcomes[ExitUnresolved] != 1 || sum.Outcomes[ExitOutcomeUnanswered] != 1 {
		t.Fatalf("summary = %+v", sum)
	}
	if len(sum.ByCLI) != 2 || sum.ByCLI[0].CLI != "codex" || sum.ByCLI[0].Total != 2 || sum.ByCLI[0].Unresolved != 1 || sum.ByCLI[0].Unanswered != 1 {
		t.Fatalf("by CLI = %+v", sum.ByCLI)
	}
	if len(sum.Reasons) != 2 || sum.Reasons[0].Count != 1 {
		t.Fatalf("reasons = %+v (the undone exit's reasons must not count)", sum.Reasons)
	}
	if sum.MedianTurns == nil || *sum.MedianTurns != 0 || sum.MeasuredTurns != 3 {
		t.Fatalf("turns = %v over %d", sum.MedianTurns, sum.MeasuredTurns)
	}

	relabeled, err := s.LabelAgentExit(silent.ID, ExitLabel{Outcome: ExitTrial})
	if err != nil || relabeled.Outcome != ExitTrial || relabeled.LabeledAt == nil || relabeled.Asked {
		t.Fatalf("label later = %+v %v", relabeled, err)
	}
	cleared, err := s.LabelAgentExit(silent.ID, ExitLabel{})
	if err != nil || cleared.Outcome != "" || cleared.LabeledAt != nil {
		t.Fatalf("clear = %+v %v", cleared, err)
	}
	if _, err := s.LabelAgentExit(good.ID, ExitLabel{Outcome: ExitResolved, Reasons: []string{"stuck"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad relabel: %v", err)
	}
	if err := s.DeleteAgentExit(bad.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetAgentExit(bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted exit: %v", err)
	}
	if err := s.DeleteAgentExit(bad.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	var exported []string
	if err := s.EachAgentExit(func(ex AgentExit) error { exported = append(exported, ex.ID); return nil }); err != nil {
		t.Fatal(err)
	}
	if len(exported) != 3 || exported[len(exported)-1] != undone.ID {
		t.Fatalf("export walks every exit, undone included, oldest first: %v", exported)
	}
}

func TestRemoveWorkspaceWithExitsEndsEachAgent(t *testing.T) {
	s := openTest(t)
	ws, _ := s.AddWorkspace("atlas", t.TempDir())
	a, _ := s.AddAgent(ws.ID, "one", "")
	b, _ := s.AddAgentWithCLI(ws.ID, "claude-code", "two", "")
	removed, exits, err := s.RemoveWorkspaceWithExits(ws.ID, ExitInput{Origin: ExitFromDesktop, Asked: true, Label: ExitLabel{Outcome: ExitResolved}, SessionsPurged: true})
	if err != nil || !removed {
		t.Fatalf("remove = %v %v", removed, err)
	}
	if len(exits) != 2 {
		t.Fatalf("exits = %d", len(exits))
	}
	got := map[string]AgentExit{}
	for _, ex := range exits {
		got[ex.AgentID] = ex
	}
	for _, id := range []string{a.ID, b.ID} {
		ex, ok := got[id]
		if !ok {
			t.Fatalf("no exit for %s", id)
		}
		// The dialog asked about the workspace, not the agent: whatever the
		// input said, the exit is unasked and carries no answer.
		if ex.Asked || ex.AskSkip != ExitSkipWorkspace || ex.Outcome != "" || ex.Origin != ExitFromDesktop || !ex.SessionsPurged || ex.WorkspaceName != "atlas" {
			t.Fatalf("exit = %+v", ex)
		}
	}
	if got[b.ID].CLI != "claude-code" {
		t.Fatalf("cli = %q", got[b.ID].CLI)
	}
	if list, _ := s.ListAgentExits(ExitFilter{WorkspaceID: ws.ID}); len(list) != 2 {
		t.Fatalf("the catalog outlives the workspace: %d", len(list))
	}
	if removed, exits, err := s.RemoveWorkspaceWithExits(ws.ID, ExitInput{}); removed || len(exits) != 0 || err != nil {
		t.Fatalf("second removal = %v %v %v", removed, exits, err)
	}
}

func TestMeterExitScopes(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "pi-1")
	_ = os.MkdirAll(agentDir, 0o755)
	for _, n := range []string{"a.jsonl", "b.jsonl"} {
		_ = os.WriteFile(filepath.Join(agentDir, n), []byte("{}\n"), 0o644)
	}
	calls := 0
	meter := func(cli, path, _ string) (ExitCost, bool) {
		calls++
		return ExitCost{Cost: 1.5, Estimated: 0.5, Tokens: 100, Turns: 2, Models: map[string]int{"m-a": 1, "m-b": 1}}, true
	}
	pi := Agent{ID: "pi-1", CLI: CLIPi}
	c := meterExit(pi, ExitSessions{PiSessionPath: filepath.Join(agentDir, "a.jsonl")}, meter)
	if c == nil || c.Scope != "agent" || c.Sessions != 2 || c.Cost != 3 || c.Estimated != 1 || c.Tokens != 200 || c.Turns != 4 || c.Models["m-a"] != 2 {
		t.Fatalf("pi agent folder = %+v", c)
	}
	guest := Agent{ID: "g-1", CLI: "codex"}
	c = meterExit(guest, ExitSessions{CLISessionPath: filepath.Join(dir, "rollout.jsonl")}, meter)
	if c == nil || c.Scope != "last-session" || c.Sessions != 1 {
		t.Fatalf("guest last session = %+v", c)
	}
	if c := meterExit(guest, ExitSessions{}, meter); c != nil {
		t.Fatalf("no session known must be not measured: %+v", c)
	}
	none := func(string, string, string) (ExitCost, bool) { return ExitCost{}, false }
	if c := meterExit(guest, ExitSessions{CLISessionPath: "x"}, none); c != nil {
		t.Fatalf("an unmeasurable session must be nil, not zero: %+v", c)
	}
}

func TestExitCostIsStoredAndSummed(t *testing.T) {
	s := openTest(t)
	a, _ := s.AddAgent(FreeWorkspaceID, "a", "")
	sp := filepath.Join(t.TempDir(), "s.jsonl")
	_ = os.WriteFile(sp, []byte("{}\n"), 0o644)
	if _, err := s.UpdateAgent(a.ID, AgentPatch{SessionPath: &sp}); err != nil {
		t.Fatal(err)
	}
	meter := func(cli, path, _ string) (ExitCost, bool) { return ExitCost{Cost: 2.25, Tokens: 10, Turns: 1}, true }
	ex, err := s.RemoveAgentWithExit(a.ID, ExitInput{Meter: meter})
	if err != nil || ex.Cost == nil || ex.Cost.Cost != 2.25 {
		t.Fatalf("exit cost = %+v %v", ex.Cost, err)
	}
	got, _ := s.GetAgentExit(ex.ID)
	if got.Cost == nil || got.Cost.Cost != 2.25 || got.Cost.Scope != "last-session" {
		t.Fatalf("stored cost = %+v", got.Cost)
	}
	b, _ := s.AddAgent(FreeWorkspaceID, "b", "")
	if _, err := s.RemoveAgentWithExit(b.ID, ExitInput{Meter: meter}); err != nil {
		t.Fatal(err)
	}
	sum, _ := s.AgentExitSummary(ExitFilter{})
	if sum.CostMeasured != 1 || sum.Cost != 2.25 || sum.Total != 2 {
		t.Fatalf("summary cost = %+v (an agent with no session is not measured)", sum)
	}
}

// A terminal CLI's agent row has no model; the exit takes the one its
// session used most. An agent that names its model keeps it.
func TestExitModelFromSessions(t *testing.T) {
	if got := dominantModel(map[string]int{"b": 2, "a": 2, "c": 1}); got != "a" {
		t.Fatalf("tie = %q, want the first by name", got)
	}
	if dominantModel(nil) != "" {
		t.Fatal("no models, no answer")
	}
	s := openTest(t)
	meter := func(cli, path, _ string) (ExitCost, bool) {
		return ExitCost{Turns: 5, Models: map[string]int{"gpt-6-astra": 4, "gpt-6-mini": 1}}, true
	}
	a, _ := s.AddAgent(FreeWorkspaceID, "guest", "")
	sp := filepath.Join(t.TempDir(), "s.jsonl")
	_ = os.WriteFile(sp, []byte("{}\n"), 0o644)
	if _, err := s.UpdateAgent(a.ID, AgentPatch{SessionPath: &sp}); err != nil {
		t.Fatal(err)
	}
	ex, err := s.RemoveAgentWithExit(a.ID, ExitInput{Meter: meter})
	if err != nil || ex.Model != "gpt-6-astra" || ex.Cost.Models["gpt-6-mini"] != 1 {
		t.Fatalf("exit model = %q, cost %+v, %v", ex.Model, ex.Cost, err)
	}
	if got, _ := s.GetAgentExit(ex.ID); got.Model != "gpt-6-astra" {
		t.Fatalf("stored model = %q", got.Model)
	}
	b, _ := s.AddAgent(FreeWorkspaceID, "pi", "")
	m := "glm-5.3-flash"
	if _, err := s.UpdateAgent(b.ID, AgentPatch{Model: &m, SessionPath: &sp}); err != nil {
		t.Fatal(err)
	}
	if ex, _ := s.RemoveAgentWithExit(b.ID, ExitInput{Meter: meter}); ex.Model != m {
		t.Fatalf("a named model was replaced: %q", ex.Model)
	}
}
