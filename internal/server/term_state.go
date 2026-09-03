package server

// Guest CLI lifecycle state for terminals (ADR-0056, tier 1). A coding
// CLI running inside a PiCode terminal reports its own state through a
// small HTTP hook (Claude Code hooks, Codex notify); PiCode correlates
// the report to the terminal via PICODE_TERM_ID — injected into the tmux
// session environment at creation, so every hook process inherits it —
// and republishes changes as ephemeral terminal.state events (ADR-0048,
// same pattern as mcp.updated). Nothing here touches SQLite: this is
// live signal, lost on restart exactly like the MCP snapshots, and a
// restart is honest "no signal" until the next hook fires.

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Terminal state vocabulary — the same words agent.state uses, so tier 2
// (guest agents, ADR-0056) can re-anchor the UI without a new language.
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

// TermState is a terminal's last reported guest-CLI state.
type TermState struct {
	State string    `json:"state"`
	CLI   string    `json:"cli,omitempty"`
	At    time.Time `json:"at"`
}

// TermStates holds live guest state per terminal id.
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
	if len(cli) > cliCap {
		cli = cli[:cliCap]
	}
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if ts.m == nil {
		ts.m = map[string]TermState{}
	}
	prev, had := ts.m[termID]
	if had && prev.State == state && prev.CLI == cli {
		return prev, false
	}
	st := TermState{State: state, CLI: cli, At: now}
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
	ts.mu.Lock()
	defer ts.mu.Unlock()
	_, ok := ts.m[termID]
	delete(ts.m, termID)
	return ok
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
		view["state"] = st.State
		view["cli"] = st.CLI
		view["stateAt"] = st.At
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

// handleSetTerminalState records a guest CLI's report for one terminal:
// POST /api/terminals/{id}/state {"state":"working","cli":"claude-code"}.
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
			State string `json:"state"`
			CLI   string `json:"cli"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		if !validTermState(req.State) {
			writeErr(w, http.StatusBadRequest, "state must be working, needs-you or idle")
			return
		}
		st, changed := deps.TermStates.Set(id, req.State, strings.TrimSpace(req.CLI), time.Now())
		if changed && deps.Feed != nil {
			deps.Feed.Ephemeral("terminal.state", map[string]any{
				"termId": id, "state": st.State, "cli": st.CLI,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"termId": id, "state": st.State, "cli": st.CLI, "at": st.At})
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
				deps.Feed.Ephemeral("terminal.state", map[string]any{"termId": id, "state": nil})
			}
		}
	}
}
