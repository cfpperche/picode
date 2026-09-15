package server

// Runtime loss detection for the tmux server (plan tmux-resilience phase 1,
// ADR-0138 follow-up).
//
// The daemon can outlive the tmux server. On 2026-09-15 the server died under
// a live daemon and nothing recorded it: ADR-0085's boot diff only runs at
// boot, so the loss was reconstructed hours later from session transcripts.
// Three incidents in ten days (29 sessions on 2026-09-06, 140 on 2026-09-13,
// production twice on 2026-09-15) all shared that blindness.
//
// This watch closes it with one probe per tick:
//
//   - reachable → absent, with sessions at stake: a `terminal.server_lost`
//     row through the store (ADR-0048, so the feed carries it) plus one
//     journal line with the last known count, socket path and last-seen time;
//   - absent → reachable: `terminal.server_back`, same shape;
//   - a running server whose session count drops by tmuxWatchMassDrop or more
//     between two ticks — a death-and-restart the reachable transitions
//     cannot see — gets one journal line, no event (mass closes are also a
//     legitimate user action, and the line is for the operator reading the
//     journal after an incident).
//
// A server that exits while holding zero sessions is normal (exit-empty):
// no event, no line.

import (
	"context"
	"log"
	"time"

	"github.com/cfpperche/picode/internal/tmux"
)

// tmuxWatchMassDrop is the session-count drop between two running ticks that
// earns a journal line. Chosen above the noise of a person closing a couple
// of terminals by hand; the loss event itself does not depend on it.
const tmuxWatchMassDrop = 5

// tmuxProber is the slice of *tmux.Manager this watch needs. An interface so
// the transition tests can drive states without a real server.
type tmuxProber interface {
	Available() bool
	ServerInfo(ctx context.Context) tmux.ServerInfo
	ServerSessions(ctx context.Context) ([]tmux.ServerSession, error)
}

// tmuxServerState is one tick's observation. Sessions is the last known
// count while running: when the listing fails it carries the previous tick's
// number rather than zero, so a single failed read cannot fake a loss.
type tmuxServerState struct {
	running  bool
	socket   string
	sessions int
}

func StartTmuxServerWatch(ctx context.Context, deps Deps, every time.Duration) {
	if deps.Tmux == nil || deps.Store == nil {
		return
	}
	watchTmuxServer(ctx, deps.Tmux, deps.Store, every)
}

func watchTmuxServer(ctx context.Context, prober tmuxProber, store eventAppender, every time.Duration) {
	if !prober.Available() {
		return
	}
	t := time.NewTicker(every)
	defer t.Stop()
	var prev tmuxServerState
	var lastSeen time.Time
	init := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		cur := probeTmuxServer(ctx, prober, prev)
		if cur.running {
			lastSeen = time.Now().UTC()
		}
		if init {
			emitTmuxServerTransitions(store, prev, cur, lastSeen)
		}
		prev, init = cur, true
	}
}

// eventAppender is the store slice this file writes through. *store.Store
// satisfies it; tests pass a recorder.
type eventAppender interface {
	AppendEvent(eventType string, agentID, workspaceID *string, data any) error
}

func probeTmuxServer(ctx context.Context, prober tmuxProber, prev tmuxServerState) tmuxServerState {
	info := prober.ServerInfo(ctx)
	cur := tmuxServerState{running: info.Running, socket: info.SocketPath, sessions: prev.sessions}
	if !info.Running {
		cur.sessions = 0
		return cur
	}
	if sessions, err := prober.ServerSessions(ctx); err == nil {
		cur.sessions = len(sessions)
	}
	return cur
}

// emitTmuxServerTransitions is the decision table, in code:
//
//	prev    | cur     | condition            | action
//	--------|---------|----------------------|--------------------------------
//	running | absent  | sessions > 0         | event terminal.server_lost + log
//	running | absent  | sessions == 0        | nothing (normal exit-empty)
//	absent  | running | —                    | event terminal.server_back + log
//	running | running | drop ≥ massDrop      | journal line only
//	anything else                           | nothing
//
// lastSeen is when the server was last observed running (zero before the
// first sighting), recorded in the loss event so the operator can bracket
// the death between two timestamps.
func emitTmuxServerTransitions(store eventAppender, prev, cur tmuxServerState, lastSeen time.Time) {
	switch {
	case prev.running && !cur.running && prev.sessions > 0:
		data := map[string]any{
			"sessions": prev.sessions,
			"socket":   prev.socket,
		}
		if !lastSeen.IsZero() {
			data["lastSeen"] = lastSeen.Format(time.RFC3339)
		}
		_ = store.AppendEvent("terminal.server_lost", nil, nil, data)
		log.Printf("tmux: server on %s is gone — last known %d session(s), last seen %s",
			prev.socket, prev.sessions, lastSeen.Format(time.RFC3339))
	case !prev.running && cur.running:
		_ = store.AppendEvent("terminal.server_back", nil, nil, map[string]any{"socket": cur.socket})
		log.Printf("tmux: server answering again on %s (%d session(s))", cur.socket, cur.sessions)
	case prev.running && cur.running && prev.sessions >= cur.sessions+tmuxWatchMassDrop:
		log.Printf("tmux: session count on %s dropped %d → %d between probes",
			cur.socket, prev.sessions, cur.sessions)
	}
}
