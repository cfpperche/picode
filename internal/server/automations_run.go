package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// errAutomationDisabled: a schedule or webhook hit a disabled automation.
var errAutomationDisabled = errors.New("automation is disabled")

// Run reasons (the decision table's vocabulary — shown verbatim in the UI).
const (
	reasonBusy        = "busy"
	reasonRateCap     = "rate cap"
	reasonCostCap     = "cost cap"
	reasonPiMissing   = "pi missing"
	reasonTargetGone  = "target gone"
	reasonInTerminal  = "agent in terminal"
	reasonQueued      = "queued"
	reasonExited      = "process exited"
	reasonTimeout     = "timeout"
	reasonStartFailed = "start failed"
)

// runTimeout caps one run; runWatchEvery is the cost-cap poll.
var (
	runTimeout    = 2 * time.Hour
	runWatchEvery = 30 * time.Second
)

// automationRunner implements automate.Runner on the server's deps: it
// owns the *how* of a run (agent, session, prompt, Inbox).
type automationRunner struct{ deps Deps }

// AutomationRunner builds the runner the engine and the HTTP routes share.
func AutomationRunner(deps Deps) automate.Runner { return automationRunner{deps: deps} }

// fireInput is everything the decision table looks at.
type fireInput struct {
	Enabled      bool
	Trigger      string
	Busy         bool // a run of this automation is still running
	RateHit      bool // max runs per window reached
	PiMissing    bool
	Action       string
	TargetExists bool         // action=message: the target agent still exists
	AgentMode    agentRunMode // the agent a start/message would touch
}

// fireDecision: Status "" = go ahead; "none" = record nothing; otherwise
// the run row to write (skipped/failed) with its reason. Notify asks for
// an Inbox fyi (subject to the one-per-state-change dedupe).
type fireDecision struct {
	Status string
	Reason string
	Notify bool
}

// decideFire is the ADR-0045 decision table. Pure; every row is tested.
func decideFire(in fireInput) fireDecision {
	if !in.Enabled && in.Trigger != store.TriggerManual {
		return fireDecision{Status: "none"}
	}
	if in.Busy {
		return fireDecision{Status: store.RunSkipped, Reason: reasonBusy}
	}
	if in.RateHit {
		return fireDecision{Status: store.RunSkipped, Reason: reasonRateCap}
	}
	if in.Action == store.AutomationMessage {
		if !in.TargetExists {
			return fireDecision{Status: store.RunFailed, Reason: reasonTargetGone, Notify: true}
		}
		if in.AgentMode == modeInteractive {
			return fireDecision{Status: store.RunSkipped, Reason: reasonInTerminal}
		}
		return fireDecision{}
	}
	if in.PiMissing {
		return fireDecision{Status: store.RunFailed, Reason: reasonPiMissing, Notify: true}
	}
	if in.AgentMode == modeInteractive {
		return fireDecision{Status: store.RunSkipped, Reason: reasonInTerminal}
	}
	return fireDecision{}
}

// shouldNotifySkip keeps the Inbox to one item per state change: a
// second identical skip in a row says nothing new.
func shouldNotifySkip(last *store.Run, status, reason string) bool {
	return last == nil || last.Status != status || last.Reason != reason
}

// composeAutomationPrompt appends the trigger payload as a labelled
// block, like the browser extension's [browser-tab] (ADR-0043).
func composeAutomationPrompt(prompt, trigger, payload string, now time.Time) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	if p := strings.TrimSpace(payload); p != "" {
		b.WriteString("\n\n[")
		b.WriteString(trigger)
		b.WriteString("]\nreceived: ")
		b.WriteString(now.Format(time.RFC3339))
		b.WriteString("\npayload:\n")
		b.WriteString(p)
	}
	return b.String()
}

func (r automationRunner) mode(ctx context.Context, agentID string) agentRunMode {
	deps := r.deps
	if deps.Runtime.Get(agentID) != nil {
		return modeManaged
	}
	if deps.Tmux != nil && deps.Tmux.Available() {
		if has, err := deps.Tmux.HasSession(ctx, tmux.SessionName(agentID)); err == nil && has {
			return modeInteractive
		}
	}
	return modeStopped
}

// Fire runs the decision table, records the outcome, and starts the run.
func (r automationRunner) Fire(a store.Automation, trigger, payload string) (store.Run, error) {
	deps := r.deps
	ctx := context.Background()
	a, err := deps.Store.GetAutomation(a.ID) // fresh: the toggle may have moved
	if err != nil {
		return store.Run{}, err
	}
	in := fireInput{Enabled: a.Enabled, Trigger: trigger, Action: a.Action}
	if running, _ := deps.Store.RunningRun(a.ID); running != nil {
		in.Busy = true
	}
	if a.MaxRuns != nil && a.MaxRunsWindowMin != nil {
		n, _ := deps.Store.CountRunsSince(a.ID, time.Now().Add(-time.Duration(*a.MaxRunsWindowMin)*time.Minute))
		in.RateHit = n >= *a.MaxRuns
	}
	if _, err := exec.LookPath(deps.AgentCmd); err != nil {
		in.PiMissing = true
	}
	switch a.Action {
	case store.AutomationMessage:
		if a.TargetAgentID != nil {
			if _, err := deps.Store.GetAgent(*a.TargetAgentID); err == nil {
				in.TargetExists = true
				in.AgentMode = r.mode(ctx, *a.TargetAgentID)
			}
		}
	default:
		if a.AgentID != nil {
			in.AgentMode = r.mode(ctx, *a.AgentID)
		}
	}
	d := decideFire(in)
	if d.Status == "none" {
		return store.Run{}, errAutomationDisabled
	}
	if d.Status != "" {
		last, _ := deps.Store.LastRun(a.ID)
		run, err := deps.Store.CreateRun(a.ID, trigger, d.Status, d.Reason)
		if err != nil {
			return store.Run{}, err
		}
		if d.Notify || shouldNotifySkip(last, d.Status, d.Reason) {
			r.notify(a, store.InboxFYI, d.Reason, a.Name+" did not run", skipBody(d.Reason))
		}
		return run, nil
	}
	body := composeAutomationPrompt(a.Prompt, trigger, payload, time.Now())
	if a.Action == store.AutomationMessage {
		if _, err := deps.Store.EnqueueTask(*a.TargetAgentID, store.TaskFollowUp, body, "automation"); err != nil {
			return store.Run{}, err
		}
		return deps.Store.CreateRun(a.ID, trigger, store.RunDone, reasonQueued)
	}
	return r.startRun(ctx, a, trigger, body)
}

func skipBody(reason string) string {
	switch reason {
	case reasonBusy:
		return "The previous run was still in progress, so this one was skipped."
	case reasonRateCap:
		return "This automation reached its runs-per-window limit. Raise the limit or wait for the window to pass."
	case reasonPiMissing:
		return "pi is not installed or not on PATH, so no agent could start."
	case reasonTargetGone:
		return "The agent this automation messages no longer exists. Pick another agent in the automation's settings."
	case reasonInTerminal:
		return "The agent is open in a terminal, where messages are not delivered automatically. Close the terminal session and the next run will go through."
	}
	return reason
}

// ensureAgent returns the automation's own agent, creating it on first
// use (one agent per automation; each run is a fresh session on it).
func (r automationRunner) ensureAgent(a store.Automation) (store.Agent, error) {
	deps := r.deps
	if a.AgentID != nil {
		if ag, err := deps.Store.GetAgent(*a.AgentID); err == nil {
			return ag, nil
		}
	}
	ag, err := deps.Store.AddAgent(a.WorkspaceID, a.Name, "")
	if err != nil {
		return store.Agent{}, err
	}
	if a.Provider != nil || a.Model != nil || a.Thinking != nil {
		ag, err = deps.Store.UpdateAgent(ag.ID, store.AgentPatch{
			Provider: strPtr(a.Provider), Model: strPtr(a.Model), Thinking: strPtr(a.Thinking),
		})
		if err != nil {
			return store.Agent{}, err
		}
	}
	return ag, deps.Store.SetAutomationAgent(a.ID, ag.ID)
}

func strPtr(p *string) *string {
	if p == nil {
		empty := ""
		return &empty
	}
	return p
}

func (r automationRunner) startRun(ctx context.Context, a store.Automation, trigger, body string) (store.Run, error) {
	deps := r.deps
	agent, err := r.ensureAgent(a)
	if err != nil {
		return store.Run{}, err
	}
	// A managed process left over from a previous run is stopped so the
	// new run starts on a fresh session (ADR-0039 mints its id).
	if deps.Runtime.Get(agent.ID) != nil {
		deps.Runtime.Stop(agent.ID)
	}
	empty := ""
	if agent, err = deps.Store.UpdateAgent(agent.ID, store.AgentPatch{SessionPath: &empty}); err != nil {
		return store.Run{}, err
	}
	run, err := deps.Store.CreateRun(a.ID, trigger, store.RunRunning, "")
	if err != nil {
		return store.Run{}, err
	}
	w := &runWatch{runner: r, a: a, run: run, agentID: agent.ID, started: time.Now()}
	if err := deps.startManaged(agent); err != nil {
		w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
		return deps.Store.GetRun(run.ID)
	}
	ma := deps.Runtime.Get(agent.ID)
	if ma == nil {
		w.finish(store.RunFailed, reasonStartFailed, true)
		return deps.Store.GetRun(run.ID)
	}
	ma.Observe(&rpc.RunObserver{OnSettled: w.settled, OnExit: w.exited, OnCost: w.spent})
	go w.watch(ma)
	if err := ma.SendTurn(store.TaskPrompt, body, nil); err != nil {
		w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
		go deps.Runtime.Stop(agent.ID)
		return deps.Store.GetRun(run.ID)
	}
	return run, nil
}

// runWatch follows one running run: session path, cost cap, timeout,
// settle and exit. finish is once-only; whoever gets there first wins.
type runWatch struct {
	runner  automationRunner
	a       store.Automation
	run     store.Run
	agentID string
	started time.Time

	mu        sync.Mutex
	path      string
	finished  bool
	eventCost float64 // pi's own per-message usage, pushed by the RunObserver
}

// spent is the cost-cap gate: pi reports usage after every assistant
// message, so the cap is enforced at message granularity rather than by
// the 30 s poll (which stays for the session path and the timeout).
func (w *runWatch) spent(total float64) {
	w.mu.Lock()
	w.eventCost = total
	capHit := w.a.MaxCostUSD != nil && total > *w.a.MaxCostUSD && !w.finished
	w.mu.Unlock()
	if !capHit {
		return
	}
	// Decide on the event's own goroutine: a settle can be microseconds
	// behind and must find the run already closed. Stopping the process
	// waits for this read loop, so that part moves off it.
	if w.finish(store.RunFailed, reasonCostCap, true) {
		go w.stopAgent()
	}
}

// stopFor closes the run with reason, then aborts and stops the agent.
func (w *runWatch) stopFor(reason string) {
	if w.finish(store.RunFailed, reason, true) {
		w.stopAgent()
	}
}

func (w *runWatch) stopAgent() {
	if ma := w.runner.deps.Runtime.Get(w.agentID); ma != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = ma.Abort(ctx)
		cancel()
	}
	w.runner.deps.Runtime.Stop(w.agentID)
}

func (w *runWatch) sessionPath(ma *rpc.ManagedAgent) string {
	w.mu.Lock()
	p := w.path
	w.mu.Unlock()
	if p != "" {
		return p
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	res, err := ma.GetState(ctx)
	if err != nil {
		return ""
	}
	var state struct {
		SessionFile string `json:"sessionFile"`
	}
	_ = json.Unmarshal(res.Data, &state)
	if state.SessionFile == "" {
		return ""
	}
	w.mu.Lock()
	w.path = state.SessionFile
	w.mu.Unlock()
	_ = w.runner.deps.Store.SetRunSession(w.run.ID, state.SessionFile)
	_, _ = w.runner.deps.Store.UpdateAgent(w.agentID, store.AgentPatch{SessionPath: &state.SessionFile})
	return state.SessionFile
}

// cost is the larger of the session file's total and pi's live usage —
// the file lags a message behind the events while the run is alive.
func (w *runWatch) cost() float64 {
	w.mu.Lock()
	p, ev := w.path, w.eventCost
	w.mu.Unlock()
	if p != "" {
		if s, err := session.Summarize(p); err == nil && s.Cost > ev {
			return s.Cost
		}
	}
	return ev
}

// finish closes the run once. Returns false when someone already did.
func (w *runWatch) finish(status, reason string, notify bool) bool {
	w.mu.Lock()
	if w.finished {
		w.mu.Unlock()
		return false
	}
	w.finished = true
	w.mu.Unlock()
	cost := w.cost()
	if err := w.runner.deps.Store.FinishRun(w.run.ID, status, reason, cost); err != nil {
		log.Printf("automations: finish run %s: %v", w.run.ID, err)
	}
	if notify {
		title := w.a.Name + " failed"
		w.runner.notify(w.a, store.InboxFYI, reason, title, failBody(reason))
		if run, err := w.runner.deps.Store.GetRun(w.run.ID); err == nil {
			w.runner.notifyOut(w.a, run, store.RunFailed, reason, failBody(reason), nil)
		}
	}
	return true
}

func failBody(reason string) string {
	switch {
	case reason == reasonCostCap:
		return "The run reached its cost limit and was stopped. Open the agent's session to see how far it got."
	case reason == reasonTimeout:
		return "The run was still going after two hours and was stopped."
	case reason == reasonExited:
		return "The pi process ended before the run finished."
	case strings.HasPrefix(reason, reasonStartFailed):
		return "The agent could not start: " + strings.TrimPrefix(reason, reasonStartFailed+": ")
	}
	return reason
}

// settled: the turn is over. Runs on the rpc read loop, so the work
// (which stops the process) moves to its own goroutine.
func (w *runWatch) settled(final string) {
	go func() {
		ma := w.runner.deps.Runtime.Get(w.agentID)
		if ma != nil {
			w.sessionPath(ma)
		}
		if w.finish(store.RunDone, "", false) {
			body := strings.TrimSpace(final)
			if body == "" {
				body = "The run finished without a message."
			}
			w.runner.notify(w.a, store.InboxResult, "automation finished", w.a.Name+" ran", body)
			if run, err := w.runner.deps.Store.GetRun(w.run.ID); err == nil {
				w.runner.notifyOut(w.a, run, store.RunDone, "", body, nil)
			}
		}
		w.runner.deps.Runtime.Stop(w.agentID)
	}()
}

// exited: the process died. Expected (we stopped it after settle) is
// silent; anything else fails the run.
func (w *runWatch) exited(expected bool) {
	if expected {
		return
	}
	go w.finish(store.RunFailed, reasonExited, true)
}

// watch polls the session cost against the cap and enforces the timeout.
func (w *runWatch) watch(ma *rpc.ManagedAgent) {
	t := time.NewTicker(runWatchEvery)
	defer t.Stop()
	for range t.C {
		w.mu.Lock()
		done := w.finished
		w.mu.Unlock()
		if done {
			return
		}
		w.sessionPath(ma)
		reason := ""
		if w.a.MaxCostUSD != nil && w.cost() > *w.a.MaxCostUSD {
			reason = reasonCostCap // file-based fallback when pi sent no usage
		} else if time.Since(w.started) > runTimeout {
			reason = reasonTimeout
		}
		if reason == "" {
			continue
		}
		w.stopFor(reason)
		return
	}
}

// notify files the automation's Inbox item (ADR-0037 provenance: source
// is the automation, reason is the state change).
func (r automationRunner) notify(a store.Automation, kind, reason, title, body string) {
	if _, err := r.deps.Store.CreateInboxItem(store.InboxItemParams{
		Kind: kind, SourceKind: store.InboxFromAutomation, SourceID: a.ID,
		WorkspaceID: a.WorkspaceID, Reason: reason, Title: title, Body: body,
	}); err != nil {
		log.Printf("automations: inbox: %v", err)
	}
}
