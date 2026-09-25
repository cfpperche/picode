package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/clilaunch"
	"github.com/cfpperche/picode/internal/climetrics"
	"github.com/cfpperche/picode/internal/pricing"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// Unattended start runs on guest CLIs (ADR-0217): the automation's own agent
// of that CLI, a fresh TUI, the prompt through the verified door, the turn's
// end read from the CLI's hooks, the session priced, the terminal closed.
// PiCode never passes a flag that skips the CLI's approvals: a turn that
// asks for one waits for a person (the Inbox says where).

const (
	// cliReadyWithin bounds the wait for the TUI's composer after a start.
	cliReadyWithin = 90 * time.Second
	cliReadyEvery  = 2 * time.Second
	cliStateEvery  = time.Second
	cliCostEvery   = 30 * time.Second
	// cliHookConfirmWithin bounds the wait for the CLI's hook to confirm a
	// prompt the screen could not.
	cliHookConfirmWithin = 15 * time.Second
	reasonNeedsYou       = "waiting for you"
)

// doorRetryable: the door refused for a reason a fresh TUI grows out of —
// still starting, not rendered yet, busy with its own start-up.
func doorRetryable(status int, reason string) bool {
	if status != http.StatusConflict {
		return false
	}
	switch reason {
	case "closed", "unrecognized", "unobservable", "working", "compacting", "busy":
		return true
	}
	return false
}

// hookSawPrompt waits for the CLI's hook to report any state set after the
// prompt went in.
func hookSawPrompt(deps Deps, termID string, after time.Time, within time.Duration) bool {
	if deps.TermStates == nil {
		return false
	}
	deadline := time.Now().Add(within)
	for {
		if st, ok := deps.TermStates.Get(termID); ok && st.At.After(after) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// turnWatch reads the CLI's hook state after delivery: the turn has ended
// when the state is idle and was set after the prompt went in (a quick turn
// can go working → idle between two reads, so the time is what counts, not
// having seen working).
type turnWatch struct {
	delivered time.Time
	askedAt   time.Time // the last needs-you we told the Inbox about
}

type turnStep int

const (
	turnGoing turnStep = iota
	turnEnded
	turnNeedsYou // a new needs-you: tell the person once
)

func (tw *turnWatch) step(st TermState, ok bool) turnStep {
	if !ok || !st.At.After(tw.delivered) {
		return turnGoing
	}
	switch st.State {
	case TermIdle:
		return turnEnded
	case TermNeedsYou:
		if st.At.After(tw.askedAt) {
			tw.askedAt = st.At
			return turnNeedsYou
		}
	}
	return turnGoing
}

// ensureCLIAgent is the automation's own agent of its CLI: reused while it
// exists in that CLI with its terminal, created (with its terminal, started)
// otherwise. created says the terminal is already fresh.
func (r automationRunner) ensureCLIAgent(ctx context.Context, a store.Automation) (store.Agent, bool, error) {
	deps := r.deps
	if a.AgentID != nil {
		if ag, err := deps.Store.GetAgent(*a.AgentID); err == nil && ag.CLI == a.CLI && ag.TerminalID != nil {
			return ag, false, nil
		}
	}
	cli, ok := clilaunch.Find(a.CLI)
	if !ok {
		return store.Agent{}, false, fmt.Errorf("unknown CLI %s", a.CLI)
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://picode.local/automation", nil)
	ag, view, _, err := createCLIAgent(deps, req, cli, cliTerminalRequest{Name: a.Name, WorkspaceID: a.WorkspaceID})
	if err != nil {
		return store.Agent{}, false, err
	}
	if msg, _ := view["launchError"].(string); msg != "" {
		return ag, false, errors.New(msg)
	}
	return ag, true, deps.Store.SetAutomationAgent(a.ID, ag.ID)
}

// freshCLITerminal closes the agent's terminal if it is open and starts it
// again: each run is a new conversation, as a Pi run's is.
func freshCLITerminal(deps Deps, ctx context.Context, t store.Terminal) error {
	name := tmux.ShellSessionName(t.ID)
	if live, _ := deps.Tmux.HasSession(ctx, name); live {
		pinCLITerminalLastSession(deps, t.ID, t)
		if err := killTerminalPane(deps, ctx, t.ID, name); err != nil {
			return err
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, "http://picode.local/automation", nil)
	_, err := ensureShell(deps, req, name, t.ID, t.Cwd)
	return err
}

// killTerminalPane is the stop escalation of ADR-0085: kill the session
// (closes the PTY), then SIGTERM the pane root, which ignores SIGHUP.
func killTerminalPane(deps Deps, ctx context.Context, id, name string) error {
	panePID, _ := deps.Tmux.PanePID(ctx, name)
	if err := deps.Tmux.KillSession(ctx, name); err != nil {
		return err
	}
	if panePID > 0 {
		if p, err := os.FindProcess(panePID); err == nil {
			_ = p.Signal(syscall.SIGTERM)
		}
	}
	if deps.TermStates != nil {
		deps.TermStates.Drop(id)
	}
	if deps.TermRuntimes != nil {
		deps.TermRuntimes.Drop(id)
	}
	return nil
}

// cliRunCost prices the terminal's current session from its file.
func cliRunCost(deps Deps, t store.Terminal, cli string) (float64, bool) {
	pinCLITerminalLastSession(deps, t.ID, t)
	l, err := deps.Store.TerminalLaunch(t.ID)
	if err != nil || l == nil || l.LastSession == nil || l.LastSession.Path == "" {
		return 0, false
	}
	c, ok := climetrics.MeterSessionFile(cli, l.LastSession.Path, pricing.Current())
	if !ok {
		return 0, false
	}
	return c.Cost, true
}

func (r automationRunner) cliStartRun(ctx context.Context, a store.Automation, f automate.Firing, body string) (store.Run, error) {
	deps := r.deps
	run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunRunning})
	if err != nil {
		return store.Run{}, err
	}
	w := &runWatch{runner: r, a: a, run: run, started: time.Now()}
	agent, fresh, err := r.ensureCLIAgent(ctx, a)
	if err != nil {
		w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
		return deps.Store.GetRun(run.ID)
	}
	w.agentID = agent.ID
	t, err := deps.Store.GetTerminal(*agent.TerminalID)
	if err != nil {
		w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
		return deps.Store.GetRun(run.ID)
	}
	if !fresh {
		if err := freshCLITerminal(deps, ctx, t); err != nil {
			w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
			return deps.Store.GetRun(run.ID)
		}
	}
	go r.driveCLIRun(w, t, body)
	return run, nil
}

// driveCLIRun waits for the composer, delivers, follows the turn, then
// prices, finishes and closes. It owns the run from here.
func (r automationRunner) driveCLIRun(w *runWatch, t store.Terminal, body string) {
	deps := r.deps
	ctx := context.Background()
	name := tmux.ShellSessionName(t.ID)
	closeTerm := func() { _ = killTerminalPane(deps, ctx, t.ID, name) }

	// 1. The composer: a fresh TUI takes a moment; a trust question, a login
	// or a menu never becomes one, and is named instead of pasted into.
	deadline := time.Now().Add(cliReadyWithin)
	var status int
	var res map[string]any
	for {
		status, res = doorDeliverUnattended(deps, ctx, t, body)
		reason, _ := res["reason"].(string)
		if status == http.StatusOK || !doorRetryable(status, reason) || time.Now().After(deadline) {
			break
		}
		time.Sleep(cliReadyEvery)
	}
	delivered := time.Now()
	if status != http.StatusOK {
		reason, _ := res["reason"].(string)
		msg := cliDisplayName(w.a.CLI) + " never showed its input box: PiCode saw a screen it does not recognize (a sign-in, a trust question or a menu). Open " + w.a.Name + " to see it."
		if !doorRetryable(status, reason) {
			if e, _ := res["error"].(string); e != "" {
				msg = e
			}
		}
		w.finish(store.RunFailed, msg, true)
		closeTerm()
		return
	}
	// A screen PiCode could not read after Enter is not proof either way; the
	// CLI's own hook is: a state set after the prompt went in means the CLI
	// took it (Claude's UserPromptSubmit, Codex's and Grok's turn start).
	if d, _ := res["delivery"].(string); d != "verified" && hookSawPrompt(deps, t.ID, delivered, cliHookConfirmWithin) {
		res["delivery"] = "verified"
	}
	if d, _ := res["delivery"].(string); d != "verified" {
		msg := cliDisplayName(w.a.CLI) + " may not have received the prompt: PiCode could not see it leave the input box. Open " + w.a.Name + " to check."
		w.finish(store.RunFailed, msg, true)
		closeTerm()
		return
	}

	// 2. The turn: the CLI's hooks say when it is over.
	tw := &turnWatch{delivered: delivered}
	tick := time.NewTicker(cliStateEvery)
	defer tick.Stop()
	lastCost := time.Now()
	for range tick.C {
		if live, err := deps.Tmux.HasSession(ctx, name); err == nil && !live {
			w.finish(store.RunFailed, cliDisplayName(w.a.CLI)+"'s terminal closed during the run.", true)
			return
		}
		st, ok := deps.TermStates.Get(t.ID)
		switch tw.step(st, ok) {
		case turnEnded:
			if c, ok := cliRunCost(deps, t, w.a.CLI); ok {
				w.mu.Lock()
				w.eventCost = c
				w.mu.Unlock()
			}
			if w.finish(store.RunDone, "", false) {
				done := "The run finished. Open " + w.a.Name + " to read its session."
				r.notify(w.a, store.InboxResult, "automation finished", w.a.Name+" ran", done)
				if run, err := deps.Store.GetRun(w.run.ID); err == nil {
					r.notifyOut(w.a, run, store.RunDone, "", done, nil)
				}
			}
			closeTerm()
			return
		case turnNeedsYou:
			r.notify(w.a, store.InboxQuestion, reasonNeedsYou, w.a.Name+" is waiting for you",
				"The run stopped at a question in its terminal (an approval or a choice). Open "+w.a.Name+" and answer there; the run goes on after that.")
		}
		if time.Since(lastCost) >= cliCostEvery {
			lastCost = time.Now()
			if c, ok := cliRunCost(deps, t, w.a.CLI); ok {
				w.mu.Lock()
				w.eventCost = c
				w.mu.Unlock()
				if w.a.MaxCostUSD != nil && c > *w.a.MaxCostUSD {
					w.finish(store.RunFailed, reasonCostCap, true)
					closeTerm()
					return
				}
			}
		}
		if time.Since(w.started) > runTimeout {
			w.finish(store.RunFailed, reasonTimeout, true)
			closeTerm()
			return
		}
	}
}

// cliDisplayName is a CLI's name as people read it.
func cliDisplayName(id string) string {
	if c, ok := clilaunch.Find(id); ok {
		return c.Name
	}
	return id
}
