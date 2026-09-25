package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/automate"
	"github.com/cfpperche/picode/internal/rpc"
	"github.com/cfpperche/picode/internal/session"
	"github.com/cfpperche/picode/internal/store"
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
	reasonTermClosed  = "terminal closed"
	reasonQueued      = "queued"
	reasonExited      = "process exited"
	reasonStopped     = "stopped during the run"
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
	PiMissing    bool // `pi` is not on PATH
	Action       string
	TargetExists bool         // action=message: the target agent still exists
	TargetIsPi   bool         // action=message: the target runs Pi (a guest is reached through its launch terminal, ADR-0179)
	StartCLI     string       // action=start: the CLI its runs use (ADR-0217); "" means pi
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
		if in.PiMissing && in.TargetIsPi && in.AgentMode != modeManaged {
			// Delivering to a Pi agent means running pi; only an already-running
			// one needs no pi to start. A guest agent never needs pi (ADR-0179).
			return fireDecision{Status: store.RunFailed, Reason: reasonPiMissing, Notify: true}
		}
		if !in.TargetIsPi {
			// A guest agent's terminal IS its process (ADR-0160): an open
			// terminal is where the door delivers, a closed one has nobody to
			// read the prompt. Starting a CLI and pasting before its TUI is
			// ready is not delivery, so a closed terminal skips honestly.
			if in.AgentMode == modeInteractive {
				return fireDecision{}
			}
			return fireDecision{Status: store.RunSkipped, Reason: reasonTermClosed}
		}
		if in.AgentMode == modeInteractive {
			return fireDecision{Status: store.RunSkipped, Reason: reasonInTerminal}
		}
		return fireDecision{}
	}
	if in.PiMissing && (in.StartCLI == "" || in.StartCLI == store.CLIPi) {
		return fireDecision{Status: store.RunFailed, Reason: reasonPiMissing, Notify: true}
	}
	// A start run owns its agent's terminal and restarts it; a terminal
	// someone opened is theirs, so the run skips rather than closing it.
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
		if has, err := deps.Tmux.HasSession(ctx, deps.agentSession(agentID)); err == nil && has {
			return modeInteractive
		}
	}
	return modeStopped
}

// Fire runs the decision table, records the outcome, and starts the run.
func (r automationRunner) Fire(a store.Automation, f automate.Firing) (store.Run, error) {
	trigger := f.Trigger
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
			if ag, err := deps.Store.GetAgent(*a.TargetAgentID); err == nil {
				in.TargetExists = true
				in.TargetIsPi = ag.IsPi()
				in.AgentMode = r.mode(ctx, *a.TargetAgentID)
			}
		}
	default:
		in.StartCLI = a.CLI
		if a.AgentID != nil {
			// The agent of another CLI (the automation's CLI changed) is not
			// the one this run uses; its mode says nothing about this run.
			if ag, err := deps.Store.GetAgent(*a.AgentID); err == nil && ag.CLI == a.CLI {
				in.AgentMode = r.mode(ctx, *a.AgentID)
			}
		}
	}
	d := decideFire(in)
	if d.Status == "none" {
		return store.Run{}, errAutomationDisabled
	}
	if d.Status != "" {
		last, _ := deps.Store.LastRun(a.ID)
		run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: trigger, Status: d.Status, Reason: d.Reason})
		if err != nil {
			return store.Run{}, err
		}
		if d.Notify || shouldNotifySkip(last, d.Status, d.Reason) {
			r.notify(a, store.InboxFYI, d.Reason, a.Name+" did not run", skipBody(d.Reason))
		}
		return run, nil
	}
	body := composeAutomationPrompt(a.Prompt, trigger, f.Payload, time.Now())
	if a.Action == store.AutomationMessage {
		return r.messageRun(ctx, a, f, body)
	}
	if a.CLI != "" && a.CLI != store.CLIPi {
		return r.cliStartRun(ctx, a, f, body)
	}
	return r.startRun(ctx, a, f, body)
}

// messageRun delivers the prompt to the target agent now (ADR-0045
// amendment): an idle agent is started on its own session, messaged,
// watched like a start run and stopped when the turn settles; a running
// one gets a follow-up turn and stays up. Until 2026-09-02 the message
// only joined the agent's task queue, which nothing drains while the
// agent is closed — a "done · queued" run that did nothing.
func (r automationRunner) messageRun(ctx context.Context, a store.Automation, f automate.Firing, body string) (store.Run, error) {
	deps := r.deps
	agent, err := deps.Store.GetAgent(*a.TargetAgentID)
	if err != nil {
		return store.Run{}, err
	}
	if !agent.IsPi() {
		return r.doorRun(ctx, a, f, body, agent)
	}
	wasRunning := deps.Runtime.Get(agent.ID) != nil
	if ma := deps.Runtime.Get(agent.ID); ma != nil && ma.Observed() {
		// Another automation's run is on this agent right now.
		last, _ := deps.Store.LastRun(a.ID)
		run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunSkipped, Reason: reasonBusy})
		if err == nil && shouldNotifySkip(last, store.RunSkipped, reasonBusy) {
			r.notify(a, store.InboxFYI, reasonBusy, a.Name+" did not run", "The agent is busy with another automation's run.")
		}
		return run, err
	}
	run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunRunning})
	if err != nil {
		return store.Run{}, err
	}
	w := &runWatch{runner: r, a: a, run: run, agentID: agent.ID, started: time.Now(), keepAlive: wasRunning}
	if !wasRunning {
		if err := deps.startManaged(agent); err != nil {
			w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
			return deps.Store.GetRun(run.ID)
		}
	}
	ma := deps.Runtime.Get(agent.ID)
	if ma == nil {
		w.finish(store.RunFailed, reasonStartFailed, true)
		return deps.Store.GetRun(run.ID)
	}
	ma.Observe(&rpc.RunObserver{OnSettled: w.settled, OnExit: w.exited, OnCost: w.spent})
	go w.watch(ma)
	// A prompt when idle; SendTurn turns it into a follow-up when the
	// agent is mid-turn (a follow_up sent to an idle pi just waits).
	if err := ma.SendTurn(store.TaskPrompt, body, nil); err != nil {
		w.finish(store.RunFailed, reasonStartFailed+": "+err.Error(), true)
		if !wasRunning {
			go deps.Runtime.Stop(agent.ID)
		} else {
			ma.Observe(nil)
		}
		return deps.Store.GetRun(run.ID)
	}
	return run, nil
}

// doorRun delivers an automation's prompt to a CLI agent through the
// ADR-0089 door (Fatia F): the agent's launch terminal is the process, so
// the run rides the door's receipt instead of the managed runtime. The
// receipt maps to run status — delivered runs finish done with the door's
// own words, refusals skip with the named reason, and an unconfirmed
// delivery fails honestly rather than pretending the CLI saw the prompt.
func (r automationRunner) doorRun(ctx context.Context, a store.Automation, f automate.Firing, body string, agent store.Agent) (store.Run, error) {
	deps := r.deps
	if agent.TerminalID == nil || strings.TrimSpace(*agent.TerminalID) == "" {
		run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunFailed, Reason: "this agent has no launch terminal"})
		if err == nil {
			r.notify(a, store.InboxFYI, "this agent has no launch terminal", a.Name+" failed", failBody("this agent has no launch terminal"))
		}
		return run, err
	}
	t, err := deps.Store.GetTerminal(*agent.TerminalID)
	if err != nil {
		return store.Run{}, err
	}
	run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunRunning})
	if err != nil {
		return store.Run{}, err
	}
	status, res := doorDeliverUnattended(deps, ctx, t, body)
	runStatus, reason := mapDoorOutcome(status, res)
	w := &runWatch{runner: r, a: a, run: run, agentID: agent.ID, started: time.Now()}
	w.finish(runStatus, reason, runStatus == store.RunFailed)
	return deps.Store.GetRun(run.ID)
}

// mapDoorOutcome translates the door's answer into a run outcome. The
// receipt is the whole truth: a verified delivery is done; refusals skip
// with the door's own named reason; an unconfirmed delivery fails honestly
// instead of pretending the CLI saw the prompt.
func mapDoorOutcome(status int, res map[string]any) (string, string) {
	reason, _ := res["reason"].(string)
	delivery, _ := res["delivery"].(string)
	switch {
	case status == http.StatusOK && delivery == "verified":
		return store.RunDone, "Sent to the terminal."
	case status == http.StatusOK && delivery == "unconfirmed":
		// "staged"/"unreadable": the paste went in, but PiCode never saw it
		// leave the composer — the automation's work has not started. Failing
		// honestly beats a done that lies.
		note := "Prompt delivered, but PiCode could not confirm it left the composer."
		if reason != "" {
			note += " (" + reason + ")"
		}
		return store.RunFailed, note
	case status == http.StatusOK:
		return store.RunDone, "Sent to the terminal (unverified)."
	case status == http.StatusConflict && reason != "":
		return store.RunSkipped, reason
	default:
		msg, _ := res["error"].(string)
		if msg == "" {
			msg = "delivery failed"
		}
		return store.RunFailed, msg
	}
}

func skipBody(reason string) string {
	switch reason {
	case reasonBusy:
		return "The previous run was still in progress, so this one was skipped."
	case reasonRateCap:
		return "This automation reached its runs-per-window limit. Raise the limit or wait for the window to pass."
	case reasonPiMissing:
		return "Pi is not installed or not on PATH, so this Pi agent could not start."
	case reasonTargetGone:
		return "The agent this automation messages no longer exists. Pick another agent in the automation's settings."
	case "unrecognized":
		return "The agent's CLI was not at a prompt PiCode recognizes (a login or a menu?), so nothing was pasted. Open the agent, finish that screen, and the next run will go through."
	case "unobservable":
		return "PiCode could not read the agent's terminal, so nothing was pasted."
	case reasonTermClosed:
		return "This agent's terminal is closed, so nobody could read the prompt. Open the agent and the next run will go through."
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
		// A Pi run reuses its agent only while it is a Pi agent: an
		// automation that ran on another CLI before gets a Pi agent of its own.
		if ag, err := deps.Store.GetAgent(*a.AgentID); err == nil && ag.IsPi() {
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

func (r automationRunner) startRun(ctx context.Context, a store.Automation, f automate.Firing, body string) (store.Run, error) {
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
	run, err := deps.Store.CreateRun(store.RunParams{AutomationID: a.ID, ScheduleID: f.ScheduleID, Trigger: f.Trigger, Status: store.RunRunning})
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
	// keepAlive: the agent was the person's, already running — the run
	// borrows a turn and leaves the process up (message runs).
	keepAlive bool
	// stopping: this run asked for the stop; the exit is expected.
	stopping bool

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
		if w.keepAlive {
			ma.Observe(nil) // the person's agent stays up; the run just lets go
			return
		}
	}
	w.letGo()
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
		if w.keepAlive {
			if ma != nil {
				ma.Observe(nil)
			}
			return
		}
		w.letGo()
		w.runner.deps.Runtime.Stop(w.agentID)
	}()
}

// exited: the process died. Expected (we stopped it after settle) is
// silent; anything else fails the run.
func (w *runWatch) exited(expected bool) {
	w.mu.Lock()
	ours := w.stopping
	w.mu.Unlock()
	if expected && ours {
		return
	}
	reason := reasonExited
	if expected {
		reason = reasonStopped // someone stopped the agent under the run
	}
	go w.finish(store.RunFailed, reason, true)
}

// letGo marks that the run itself is about to stop the process, so the
// exit that follows is silent.
func (w *runWatch) letGo() {
	w.mu.Lock()
	w.stopping = true
	w.mu.Unlock()
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

// automationRunOn reports whether an automation run is watching this
// agent's managed process. Opening the agent in a terminal would kill
// that process (ADR-0006: one mode at a time), so the terminal paths
// refuse while it is true; an explicit stop still wins, and the run
// then ends as "stopped during the run".
func (deps Deps) automationRunOn(agentID string) bool {
	if deps.Runtime == nil {
		return false
	}
	ma := deps.Runtime.Get(agentID)
	return ma != nil && ma.Observed()
}

const runInFlightMsg = "an automation run is in progress on this agent — wait for it to finish, or stop the agent first"
