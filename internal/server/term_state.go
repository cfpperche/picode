package server

// Coding-CLI lifecycle state for terminals (ADR-0056, tier 1). A CLI
// running inside a PiCode terminal reports its own state through a small
// HTTP hook (Claude Code, Codex, Grok, Hermes Agent, OpenCode, or manual Pi TUI); PiCode correlates
// the report to the terminal via PICODE_TERM_ID — injected into the tmux
// session environment at creation, so every hook process inherits it —
// and republishes changes as ephemeral terminal.state events (ADR-0048,
// same pattern as mcp.updated). When a wrapper lease exists, the state also
// carries its runId, so an old process cannot overwrite a newer one. Nothing
// here touches SQLite: this is live signal, lost on restart exactly like the
// MCP snapshots, and a restart is honest "no signal" until the next hook or
// runtime reconciliation fires.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// Terminal state vocabulary — the same words agent.state uses, so tier 2
// (CLI agents, ADR-0056) can re-anchor the UI without a new language.
const (
	TermWorking  = "working"
	TermNeedsYou = "needs-you"
	TermIdle     = "idle"
)

// workingTTL bounds how long a "working" report is trusted without
// hearing anything again. Hooks fire between tools, not inside one, so a
// long single tool run is silent for a while; the bound is generous.
// Past it the state clears and the chip disappears — "no signal", never
// a stale spinner (benchmarks.md: status is always truthful).
const workingTTL = 30 * time.Minute

// cliCap bounds the reported CLI label (free-form, display only).
const cliCap = 32

// Human-input attention a coding CLI reports alongside a lifecycle state.
// A held question (Grok's ask_user_question card) keeps needs-you until the
// question tool itself completes: a turn's tool calls run as a parallel
// batch, so sibling completions arrive while the card still waits, and none
// of them is the answer.
const (
	TermAttentionQuestion = "question"
	TermAttentionAnswered = "answered"
)

// TermState is a terminal's last reported CLI state.
type TermState struct {
	SessionID  string    `json:"sessionId,omitempty"`
	SessionSeq int64     `json:"sessionSeq,omitempty"`
	State      string    `json:"state"`
	CLI        string    `json:"cli,omitempty"`
	RunID      string    `json:"runId,omitempty"`
	Attention  string    `json:"attention,omitempty"`
	At         time.Time `json:"at"`
}

// TermStates holds live CLI state per terminal id.
type TermStates struct {
	mu sync.Mutex
	m  map[string]TermState
}

// NewTermStates builds an empty registry.
func NewTermStates() *TermStates {
	return &TermStates{m: map[string]TermState{}}
}

func validTermState(s string) bool {
	return s == TermWorking || s == TermNeedsYou || s == TermIdle
}

// Set records a report and tells whether anything changed (a repeat of
// the same state with the same cli is not worth an event).
func (ts *TermStates) Set(termID, state, cli string, now time.Time) (TermState, bool) {
	return ts.SetForRun(termID, state, cli, "", "", now)
}

// heldQuestion reports whether a report must leave a held question alone:
// working reports during the hold are parallel batch siblings, not answers.
func heldQuestion(prev TermState, state, attention string) bool {
	return prev.Attention == TermAttentionQuestion && state == TermWorking && attention != TermAttentionAnswered
}

// attentionFor keeps only an attention a report can hold.
func attentionFor(state, attention string) string {
	if state == TermNeedsYou && attention == TermAttentionQuestion {
		return attention
	}
	return ""
}

// SetForRun records a lifecycle report and associates it with a runtime
// lease when the wrapper supplied one. An empty runID keeps the legacy hook
// path working and inherits the previous run identity when possible.
func (ts *TermStates) SetForRun(termID, state, cli, runID, attention string, now time.Time) (TermState, bool) {
	if len(cli) > cliCap {
		cli = cli[:cliCap]
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.m == nil {
		ts.m = map[string]TermState{}
	}
	prev, had := ts.m[termID]
	if runID == "" && had {
		runID = prev.RunID
	}
	if had && heldQuestion(prev, state, attention) {
		return prev, false
	}
	if had && prev.State == state && prev.CLI == cli && prev.RunID == runID && prev.Attention == attentionFor(state, attention) {
		return prev, false
	}
	st := TermState{State: state, CLI: cli, RunID: runID, Attention: attentionFor(state, attention), At: now}
	ts.m[termID] = st
	return st, true
}

// Get returns the terminal's live state when it has one.
func (ts *TermStates) Get(termID string) (TermState, bool) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	st, ok := ts.m[termID]
	return st, ok
}

// Drop forgets a terminal (deleted). Reports whether anything was held.
func (ts *TermStates) Drop(termID string) bool {
	return ts.DropForRun(termID, "")
}

// DropForRun clears only the matching runtime's state. Empty state RunIDs
// are legacy reports and can be cleared by the current runtime ending.
func (ts *TermStates) DropForRun(termID, runID string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	prev, ok := ts.m[termID]
	if !ok || (runID != "" && prev.RunID != "" && prev.RunID != runID) {
		return false
	}
	delete(ts.m, termID)
	return true
}

// Sweep clears stale "working" entries (ttl since the last report) and
// returns the ids whose state changed. needs-you and idle do not decay:
// needs-you waits for the human by definition, and idle is the last fact
// known.
func (ts *TermStates) Sweep(now time.Time, ttl time.Duration) []string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	var out []string
	for id, st := range ts.m {
		if st.State == TermWorking && now.Sub(st.At) > ttl {
			delete(ts.m, id)
			out = append(out, id)
		}
	}
	return out
}

// applyTermState folds the registry's state into a terminal view. The
// field names match the feed event, so the browser treats both alike.
func applyTermState(deps Deps, view map[string]any, termID string) {
	if deps.TermStates == nil {
		return
	}
	if st, ok := deps.TermStates.Get(termID); ok {
		if deps.TermRuntimes != nil {
			if runtime, present := deps.TermRuntimes.Get(termID); present && st.RunID != "" && st.RunID != runtime.RunID {
				return
			}
		}
		view["state"] = st.State
		if st.CLI != "" {
			view["cli"] = st.CLI
		}
		view["stateAt"] = st.At
		if st.RunID != "" {
			view["runId"] = st.RunID
		}
	}
}

// loopbackURL best-efforts the daemon's local base URL, injected as
// PICODE_TERM_URL so hook scripts need no configuration. Unknown when
// settings are not wired (tests) — the guide then documents the URL.
func loopbackURL(deps Deps) string {
	if deps.PortSnapshot == nil {
		return ""
	}
	snap := deps.PortSnapshot()
	if snap.Current == 0 {
		return ""
	}
	// HTTPS certs are issued for localhost (mkcert), not 127.0.0.1 — using
	// the IP made the reporter's curl fail SAN check (2026-09-03 dogfood).
	host := "localhost"
	scheme := "https"
	if deps.Insecure {
		scheme = "http"
		host = "127.0.0.1"
	}
	return scheme + "://" + host + ":" + strconv.Itoa(snap.Current)
}

func reportTermState(deps Deps, id, state, cli string, now time.Time) TermState {
	return reportTermStateForRun(deps, id, state, cli, "", "", now)
}

func reportTermStateForRun(deps Deps, id, state, cli, runID, attention string, now time.Time) TermState {
	if deps.TermRuntimes != nil {
		if runtime, ok := deps.TermRuntimes.Get(id); ok {
			if runID == "" {
				runID = runtime.RunID
			}
			if cli == "" {
				cli = runtime.CLI
			}
		}
	}
	before, hadBefore := deps.TermStates.Get(id)
	st, changed := deps.TermStates.SetForRun(id, state, cli, runID, attention, now)
	if changed && startsTurn(before, hadBefore, st.State) {
		noteTerminalTurn(deps, id)
	}
	if changed && deps.Feed != nil {
		data := map[string]any{
			"termId": id, "state": st.State, "cli": st.CLI, "at": st.At,
		}
		if st.RunID != "" {
			data["runId"] = st.RunID
		}
		if st.Attention != "" {
			data["attention"] = st.Attention
		}
		deps.Feed.Ephemeral("terminal.state", data)
	}
	if changed {
		syncManagedCLIInbox(deps, id, st.State)
	}
	return st
}

// startsTurn says whether a reported state begins a turn (ADR-0194): work
// that follows idle or no state. Work after needs-you is the same turn
// resuming once the person answered.
func startsTurn(before TermState, had bool, next string) bool {
	if next != TermWorking {
		return false
	}
	return !had || (before.State != TermWorking && before.State != TermNeedsYou)
}

// noteTerminalTurn counts the turn on the agent bound to the terminal; a
// plain shell has no agent and counts nothing.
func noteTerminalTurn(deps Deps, termID string) {
	if deps.Store == nil {
		return
	}
	if a, err := deps.Store.AgentByTerminal(termID); err == nil {
		_ = deps.Store.NoteAgentTurn(a.ID)
	}
}

// terminalInterruptObserver is the PTY-input fallback for Escape/Ctrl+C.
// Claude Code does not run Stop when a user aborts a turn, so waiting for its
// hook leaves a stale spinner. The bridge recognizes only an actual interrupt
// byte (not arrow/Alt escape sequences); this observer then clears an active
// state immediately and the CLI hook reconciles any state that follows.
func terminalInterruptObserver(deps Deps) func(session string) {
	return func(session string) {
		if deps.Store == nil || deps.TermStates == nil || !tmux.IsShellSession(session) {
			return
		}
		terminals, err := deps.Store.ListTerminals()
		if err != nil {
			return
		}
		for _, terminal := range terminals {
			if tmux.ShellSessionName(terminal.ID) != session {
				continue
			}
			current, ok := deps.TermStates.Get(terminal.ID)
			if !ok || current.State == TermIdle {
				return
			}
			reportTermState(deps, terminal.ID, TermIdle, current.CLI, time.Now())
			return
		}
	}
}

// handleSetTerminalState records a coding CLI's report for one terminal:
// POST /api/terminals/{id}/state {"state":"working","cli":"claude-code","runId":"..."}.
// The route sits behind the ordinary auth gate (ADR-0049) — hook scripts
// send the install token like every other non-browser client.
func handleSetTerminalState(deps Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.TermStates == nil {
			writeErr(w, http.StatusServiceUnavailable, "Terminal status is not available.")
			return
		}
		id := r.PathValue("id")
		if _, err := deps.Store.GetTerminal(id); err != nil {
			writeStoreErr(w, err)
			return
		}
		var req struct {
			ObservationVersion int    `json:"observationVersion"`
			State              string `json:"state"`
			CLI                string `json:"cli"`
			RunID              string `json:"runId"`
			Attention          string `json:"attention"`
			SessionID          string `json:"sessionId"`
			SessionPath        string `json:"sessionPath"`
			SessionSeq         int64  `json:"sessionSeq"`
			Source             string `json:"source"`
			PID                int    `json:"pid"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if !validTermState(req.State) {
			writeErr(w, http.StatusBadRequest, "state must be working, needs-you or idle")
			return
		}
		runID := strings.TrimSpace(req.RunID)
		if req.SessionID != "" && req.PID > 0 {
			recoverNativeRuntime(r.Context(), deps, id, req.CLI, runID, req.PID)
		}
		if deps.TermRuntimes != nil {
			if runtime, ok := deps.TermRuntimes.Get(id); ok {
				if runID != "" && runID != runtime.RunID {
					writeErr(w, http.StatusConflict, "stale terminal runtime")
					return
				}
				runID = runtime.RunID
			}
		}
		if req.ObservationVersion == 1 {
			reconcileNativeObservation(r.Context(), deps, id)
			rt, present := deps.TermRuntimes.Get(id)
			if present && rt.RunID == req.RunID && rt.SessionSeq >= req.SessionSeq {
				writeJSON(w, http.StatusOK, map[string]any{"termId": id})
				return
			}
			writeErr(w, http.StatusConflict, "Native observation is unavailable or no longer current.")
			return
		}
		var st TermState
		nativeErr := error(nil)
		if req.SessionID != "" {
			nativeErr = recordNativeTerminalObservation(deps, id, req.CLI, req.RunID, req.SessionID, req.SessionPath, req.SessionSeq, req.State, req.Source, req.Attention)
		}
		if nativeErr != nil {
			// Wrapper-less observers (agy title reporter, muse settings
			// hooks) never start a native runtime, so a session report
			// with no runtime entry has no identity fence to protect:
			// report it plainly instead of dropping the signal (which
			// left those CLIs stuck Open). A terminal WITH a runtime
			// keeps the strict conflict.
			hasRuntime := false
			if req.SessionID != "" && deps.TermRuntimes != nil {
				_, hasRuntime = deps.TermRuntimes.Get(id)
			}
			if hasRuntime {
				writeErr(w, http.StatusConflict, "Native conversation report is stale or invalid.")
				return
			}
			st = reportTermStateForRun(deps, id, req.State, strings.TrimSpace(req.CLI), runID, req.Attention, time.Now())
		} else if req.SessionID != "" {
			st, _ = deps.TermStates.Get(id)
			if deps.Feed != nil {
				data := map[string]any{"termId": id, "state": st.State, "cli": st.CLI, "runId": st.RunID, "at": st.At}
				if st.Attention != "" {
					data["attention"] = st.Attention
				}
				deps.Feed.Ephemeral("terminal.state", data)
			}
		} else {
			st = reportTermStateForRun(deps, id, req.State, strings.TrimSpace(req.CLI), runID, req.Attention, time.Now())
		}
		invalidateTerminals(deps)
		writeJSON(w, http.StatusOK, map[string]any{"termId": id, "state": st.State, "cli": st.CLI, "runId": st.RunID, "attention": st.Attention, "at": st.At})
	}
}

// StartTermStateSweep expires stale "working" reports (one tick for the
// whole fleet, ADR-0048 pattern) so a silenced sensor can never leave a
// spinner spinning forever.
func StartTermStateSweep(ctx context.Context, deps Deps, every time.Duration) {
	if deps.TermStates == nil || deps.Feed == nil {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, id := range deps.TermStates.Sweep(time.Now(), workingTTL) {
				data := map[string]any{"termId": id, "state": nil}
				if deps.TermRuntimes != nil {
					if runtime, ok := deps.TermRuntimes.Get(id); ok {
						data["runId"] = runtime.RunID
					}
				}
				deps.Feed.Ephemeral("terminal.state", data)
			}
		}
	}
}
