// Package tmux — the server-wide read model (the tmux app).
//
// Everything else in this package answers a question about ONE session
// PiCode knows it owns. This file answers the question an operator asks
// when something is wrong: what is on the tmux server right now — PiCode's
// sessions, the user's own sessions, and the leftovers of a harness or a
// previous instance that no store row claims any more.
//
// Two rules shape it, and both come from measured incidents:
//
//  1. **One call, one row per session.** `list-panes -a` is a single tmux
//     invocation for the whole server (no round-trip per session, no
//     round-trip per pane). A session with two panes stays ONE row: the
//     active pane supplies the details and Panes counts them, so a split
//     pane can never duplicate a row or inflate a count.
//  2. **The name is a hint; the marker is the receipt.** PiCode injects
//     PICODE_TERM_ID / PICODE_AGENT_ID into a session's environment at
//     creation (-e), so a session carries its own proof of who created it.
//     The name is derivable and cheap, which is why the list uses it; the
//     marker is authoritative, which is why SessionReceipt reads it before
//     anything is killed. A session whose name is not `picode-*` is the
//     user's own, and no code here may act on it.
//
// The package deliberately cannot see the store: it reports what tmux says,
// and internal/server resolves that against the SQLite rows.
package tmux

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// MarkerTermEnv and MarkerAgentEnv are the ownership markers PiCode injects
// into a session environment at creation. They are the session's own receipt
// of which terminal or agent created it, and they survive a store deletion,
// a daemon restart and a rename — which is exactly the case this file exists
// for. The literal names are duplicated here rather than imported from
// internal/store so the tmux package stays free of the SQLite layer; the
// server test asserts they match the store's constants.
const (
	MarkerTermEnv  = "PICODE_TERM_ID"
	MarkerAgentEnv = "PICODE_AGENT_ID"
)

// ServerSession is one session on the tmux server this Manager talks to.
// One row per session, never one per pane (see the package comment).
type ServerSession struct {
	Name      string
	SessionID string // tmux's immutable identity ($12) — survives a respawn
	Created   time.Time
	Windows   int
	Panes     int
	Attached  int // clients attached right now (session_attached)
	// The active pane's facts.
	PaneID         string
	PanePID        int
	PaneDead       bool
	DeadStatus     int // -1 when tmux reports none
	CurrentCommand string
	StartCommand   string
	CurrentPath    string
}

// AbsentTerminal is one terminal whose expected session is not on the
// server — the absence list the tmux app renders, with the flight
// recorder's verdict (ADR-0085) attached.
type AbsentTerminal struct {
	ID            string
	Name          string
	Session       string
	WorkspaceID   string
	LostAtRestart bool
}

// ServerInfo is the server itself: what is running, where its socket lives,
// and the two settings PiCode depends on the user's tmux.conf not breaking.
type ServerInfo struct {
	// Running is false when no server answers on this socket — an honest
	// state to draw, not an error (a machine with no terminals has no server).
	Running            bool
	SocketPath         string
	Clients            int
	Version            string
	ExtendedKeysFormat string
}

// SessionReceipt is the identity of one session, read from tmux immediately
// before an action. It is the receipt an action is verified against: if any
// field differs from what the human looked at, the action is refused.
type SessionReceipt struct {
	Name      string
	SessionID string
	Created   time.Time
	PaneID    string
	PanePID   int
	// Markers, read from the session environment. Empty means the session
	// was not created by a PiCode that injects them (older version, a
	// harness, or the user's own tmux).
	TermID  string
	AgentID string
}

// paneListFormat is the one format string the server read uses. Fields are
// tab-separated; `pane_start_command` comes LAST because it is the only field
// that can itself contain a tab, and a trailing field cannot shift the ones
// before it.
const paneListFormat = "#{session_name}\t#{session_id}\t#{session_created}\t#{session_windows}\t#{session_attached}" +
	"\t#{pane_id}\t#{pane_pid}\t#{pane_dead}\t#{pane_dead_status}\t#{pane_current_command}\t#{pane_current_path}\t#{pane_active}\t#{pane_start_command}"

const paneListFields = 13

// ServerSessions returns every session on the socket, PiCode's and the
// user's alike, in tmux's own order. A server that is not running yields an
// empty list, never an error (same contract as ListOwned).
func (m *Manager) serverSessions(ctx context.Context) ([]ServerSession, error) {
	out, err := m.run(ctx, "list-panes", "-a", "-F", paneListFormat)
	if err != nil {
		if serverAbsent(out) {
			return nil, nil
		}
		return nil, fmt.Errorf("tmux list-panes: %s", strings.TrimSpace(out))
	}

	type acc struct {
		session ServerSession
		active  bool
	}
	byName := map[string]*acc{}
	order := make([]string, 0, 16)
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < paneListFields {
			continue
		}
		name := parts[0]
		if name == "" {
			continue
		}
		// pane_start_command is last: rejoin in case the command held a tab.
		startCommand := strings.Join(parts[12:], "\t")

		a := byName[name]
		if a == nil {
			createdUnix, _ := strconv.ParseInt(parts[2], 10, 64)
			windows, _ := strconv.Atoi(parts[3])
			a = &acc{session: ServerSession{
				Name:      name,
				SessionID: parts[1],
				Created:   time.Unix(createdUnix, 0).UTC(),
				Windows:   windows,
			}}
			byName[name] = a
			order = append(order, name)
		}
		a.session.Panes++
		pid, _ := strconv.Atoi(parts[6])
		dead := parts[7] == "1"
		deadStatus := -1
		if dead && parts[8] != "" {
			if n, err := strconv.Atoi(parts[8]); err == nil {
				deadStatus = n
			}
		}
		attached, _ := strconv.Atoi(parts[4])
		// A session's client count is a session fact: the same value on every
		// pane row, so taking the last one is taking the session's.
		a.session.Attached = attached

		// The active pane wins the detail slot; with no active pane (a dead
		// session can report none) the first pane answers, so a row is never
		// blank while the session has panes at all.
		isActive := parts[11] == "1"
		if !a.active || isActive {
			a.session.PaneID = parts[5]
			a.session.PanePID = pid
			a.session.PaneDead = dead
			a.session.DeadStatus = deadStatus
			a.session.CurrentCommand = parts[9]
			a.session.CurrentPath = parts[10]
			a.session.StartCommand = startCommand
			a.active = isActive
		}
	}

	sessions := make([]ServerSession, 0, len(order))
	for _, name := range order {
		sessions = append(sessions, byName[name].session)
	}
	return sessions, nil
}

// ServerInfo reads the server's own facts: whether one answers, where its
// socket is, how many clients are attached, the tmux version and the
// extended-keys format. Session counts belong to ServerSessions — this never
// lists sessions, so the server read costs one call per fact.
func (m *Manager) serverInfo(ctx context.Context) ServerInfo {
	var info ServerInfo
	// socket_path only answers with a server behind it, which is what makes
	// this the liveness probe as well as the path read.
	if out, err := m.run(ctx, "display-message", "-p", "#{socket_path}"); err == nil {
		info.Running = true
		info.SocketPath = strings.TrimSpace(out)
	}
	if v, err := m.Version(); err == nil {
		info.Version = v
	}
	if f, err := m.ExtendedKeysFormat(ctx); err == nil {
		info.ExtendedKeysFormat = f
	}
	// No clients is not an error: `list-clients` with none prints nothing.
	if out, err := m.run(ctx, "list-clients", "-F", "#{client_pid}"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
			if strings.TrimSpace(line) != "" {
				info.Clients++
			}
		}
	}
	return info
}

// SessionReceipt reads one session's identity plus its ownership markers —
// two tmux calls, deliberately, because this is the read an irreversible
// action is verified against and neither half may be skipped.
func (m *Manager) sessionReceipt(ctx context.Context, name string) (SessionReceipt, error) {
	var r SessionReceipt
	// name+":" targets the session's active pane (the same form the rest of
	// this package uses; the "=" prefix means something else to send-keys).
	out, err := m.run(ctx, "display-message", "-p", "-t", name+":",
		"#{session_name}\t#{session_id}\t#{session_created}\t#{pane_id}\t#{pane_pid}")
	if err != nil {
		return r, fmt.Errorf("tmux session %q: %s", name, strings.TrimSpace(out))
	}
	parts := strings.Split(strings.TrimSpace(out), "\t")
	if len(parts) < 5 || parts[0] == "" {
		return r, fmt.Errorf("tmux session %q: unreadable identity", name)
	}
	createdUnix, _ := strconv.ParseInt(parts[2], 10, 64)
	pid, _ := strconv.Atoi(parts[4])
	r = SessionReceipt{
		Name:      parts[0],
		SessionID: parts[1],
		Created:   time.Unix(createdUnix, 0).UTC(),
		PaneID:    parts[3],
		PanePID:   pid,
	}

	// One dump, both markers. A session with no environment at all is a
	// session nobody marked — that is information, not a failure.
	if env, err := m.run(ctx, "show-environment", "-t", name); err == nil {
		for _, line := range strings.Split(env, "\n") {
			key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
			if !ok {
				continue
			}
			switch key {
			case MarkerTermEnv:
				r.TermID = value
			case MarkerAgentEnv:
				r.AgentID = value
			}
		}
	}
	return r, nil
}

// serverAbsent reports whether a tmux failure means "there is no server",
// which every read in this package answers with an empty result rather than
// an error. A client with no server starts one; a command that needs no
// session then watches it exit empty and reports "server exited
// unexpectedly" — the same state seen from the other side.
func serverAbsent(out string) bool {
	for _, msg := range []string{"no server running", "error connecting to", "no such file or directory", "server exited unexpectedly"} {
		if strings.Contains(out, msg) {
			return true
		}
	}
	return false
}
