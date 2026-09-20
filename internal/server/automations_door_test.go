package server

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Fatia F decision table: an automation aimed at a CLI agent delivers
// through the prompt door, and the door's receipt is the run's outcome.
// | terminal         | receipt                  | run                    |
// | none             | —                        | failed, honest reason  |
// | absent session   | 409 closed               | skipped "closed"       |
// | live CLI pane    | verified / unconfirmed   | done, the door's words |
// | refusal          | working/occupied/busy    | skipped, named reason  |

func doorTestSetup(t *testing.T) (Deps, store.Automation, store.Agent) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "picode.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	proj := t.TempDir()
	w, err := st.AddWorkspace("App", proj)
	if err != nil {
		t.Fatal(err)
	}
	a, err := st.AddAgentWithCLI(w.ID, "claude-code", "Claude", "")
	if err != nil {
		t.Fatal(err)
	}
	aut, _, err := st.CreateAutomation(store.AutomationParams{
		Name: "door", WorkspaceID: w.ID, Action: "message", TargetAgentID: a.ID, Prompt: "go", Cron: "0 9 * * *",
	})
	if err != nil {
		t.Fatalf("automation: %v", err)
	}
	deps := Deps{Store: st, Tmux: tmux.New(), TermStates: NewTermStates(), DataDir: t.TempDir()}
	return deps, aut, a
}

func doorFiring() automate.Firing {
	return automate.Firing{ScheduleID: "sched-1", Trigger: store.TriggerSchedule}
}

func TestDoorRunFailsHonestlyWithoutTerminal(t *testing.T) {
	deps, aut, agent := doorTestSetup(t)
	if agent.TerminalID != nil {
		t.Fatalf("a fresh CLI agent should have no terminal yet")
	}
	r := automationRunner{deps: deps}
	run, err := r.doorRun(t.Context(), aut, doorFiring(), "hello", agent)
	if err != nil {
		t.Fatal(err)
	}
	got, err := deps.Store.GetRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.RunFailed || got.Reason != "this agent has no launch terminal" {
		t.Fatalf("run = %+v", got)
	}
}

func TestDoorRunSkipsWhenAgentNotRunning(t *testing.T) {
	deps, aut, agent := doorTestSetup(t)
	tm, err := deps.Store.CreateTerminalIn(aut.WorkspaceID, "Claude", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := deps.Store.SetTerminalLaunch(tm.ID, "claude-code", clilaunch.Overrides{}); err != nil {
		t.Fatal(err)
	}
	tid := tm.ID
	if _, err := deps.Store.UpdateAgent(agent.ID, store.AgentPatch{TerminalID: &tid}); err != nil {
		t.Fatal(err)
	}
	agent, err = deps.Store.GetAgent(agent.ID)
	if err != nil {
		t.Fatal(err)
	}
	r := automationRunner{deps: deps}
	run, err := r.doorRun(t.Context(), aut, doorFiring(), "hello", agent)
	if err != nil {
		t.Fatal(err)
	}
	got, err := deps.Store.GetRun(run.ID)
	if err != nil {
		t.Fatal(err)
	}
	// The launch is saved but no tmux session was ever started for it: the
	// door refuses with "closed" and the run skips instead of pretending.
	if got.Status != store.RunSkipped || got.Reason != "closed" {
		t.Fatalf("run = %+v", got)
	}
}

func TestMapDoorOutcome(t *testing.T) {
	verifiedStatus, verified := mapDoorOutcome(http.StatusOK, map[string]any{"delivery": "verified"})
	if verifiedStatus != store.RunDone || verified != "Sent to the terminal." {
		t.Fatalf("verified = %q %q", verifiedStatus, verified)
	}
	unconfirmedStatus, unconfirmed := mapDoorOutcome(http.StatusOK, map[string]any{"delivery": "unconfirmed", "reason": "staged"})
	if unconfirmedStatus != store.RunDone || unconfirmed == "" {
		t.Fatalf("unconfirmed = %q %q", unconfirmedStatus, unconfirmed)
	}
	unverifiedStatus, unverified := mapDoorOutcome(http.StatusOK, map[string]any{"delivery": "unverified"})
	if unverifiedStatus != store.RunDone || unverified == "" {
		t.Fatalf("unverified = %q %q", unverifiedStatus, unverified)
	}
	skipStatus, skipReason := mapDoorOutcome(http.StatusConflict, map[string]any{"reason": "closed"})
	if skipStatus != store.RunSkipped || skipReason != "closed" {
		t.Fatalf("skip = %q %q", skipStatus, skipReason)
	}
	failStatus, failReason := mapDoorOutcome(http.StatusServiceUnavailable, map[string]any{"error": "Need tmux to send to a terminal."})
	if failStatus != store.RunFailed || failReason == "" {
		t.Fatalf("fail = %q %q", failStatus, failReason)
	}
}
