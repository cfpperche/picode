// The tmux app's API family (the tmux Server Inspector, PiCode's own).
//
// Two routes and one shape of truth:
//
//	GET  /api/tmux/server                  — the server and every session on it
//	POST /api/tmux/sessions/{name}/reap    — kill ONE named leftover, with a receipt
//
// Read design. The read is a single `list-panes -a` (one tmux call for the
// whole server — internal/tmux/server.go) plus in-process store lookups, so
// opening the app costs one subprocess, not one per session. Nothing here
// polls: the app refetches from the change feed (ADR-0048) and after its own
// actions, exactly like the Docker app and for the same reason.
//
// Attribution design, and the reason this file exists rather than a prefix
// filter. A session's owner is resolved in this order:
//
//  1. the NAME (`picode-sh-<id>` = a terminal, `picode-<id>` = an agent) is a
//     cheap hint, so the list uses it and looks the id up in the store;
//  2. the MARKER the session carries in its own environment (PICODE_TERM_ID /
//     PICODE_AGENT_ID, injected at creation) is the authoritative receipt, so
//     it is read immediately before any kill — and a marker that resolves to a
//     live store row refuses the kill rather than performing it.
//
// The list is therefore allowed to be *uncertain* and the action is not: a
// session whose name does not resolve is reported as "not in PiCode's records"
// and never as "safe to delete" — because "PiCode-shaped but unknown to my
// store" is also exactly how a session belonging to a SECOND PiCode instance
// on the same machine's tmux server looks (measured 2026-09-14: seven live
// sessions read as unclaimed by a script that failed to reach the instance
// that had attached all seven). No read can tell those apart; a human
// confirming one row at a time can.
package server

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cfpperche/picode/internal/auth"
	"github.com/cfpperche/picode/internal/store"
	"github.com/cfpperche/picode/internal/tmux"
)

// tmuxServerSource is the tmux seam these routes read through — the same
// shape as sessionSource for the flight recorder (forensics.go): the handler
// takes the interface, the daemon hands it the live *tmux.Manager, and a test
// scripts it without a tmux server anywhere near the user's own.
type tmuxServerSource interface {
	Available() bool
	ServerSessions(ctx context.Context) ([]tmux.ServerSession, error)
	ServerInfo(ctx context.Context) tmux.ServerInfo
	SessionReceipt(ctx context.Context, name string) (tmux.SessionReceipt, error)
	KillSession(ctx context.Context, name string) error
}

// Where a session's authority comes from, as the UI must say it.
const (
	// scopeYours: this daemon's store has the terminal or agent the session
	// names. The session is the app's own, and its door is the terminal or
	// agent surface, not this app's reap.
	scopeYours = "yours"
	// scopeUnclaimed: PiCode-shaped name, no terminal or agent with that id in
	// this daemon's records. An orphan of a deleted store, a harness leftover,
	// or another instance's session — the app says which of those it cannot
	// tell apart, and the kill needs a human.
	scopeUnclaimed = "unclaimed"
	// scopeForeign: not PiCode's namespace at all — the user's own tmux.
	// Shown, never actionable.
	scopeForeign = "notPiCode"
)

// Session kinds, from the name's shape only (the marker can refine it at
// action time, never in the list).
const (
	kindAgent    = "agent"
	kindTerminal = "terminal"
	kindOther    = "other"
)

type tmuxSessionView struct {
	Name      string `json:"name"`
	SessionID string `json:"sessionId"`
	Created   string `json:"created"`
	Windows   int    `json:"windows"`
	Panes     int    `json:"panes"`
	Attached  int    `json:"attached"`
	Command   string `json:"command,omitempty"`
	Cwd       string `json:"cwd,omitempty"`
	// PanePID and SessionID are the receipt the client sends back with a
	// removal: without them on the wire the server would compare a real pane
	// against a zero and refuse every request.
	PanePID  int    `json:"panePid"`
	Dead     bool   `json:"dead"`
	ExitCode *int   `json:"exitCode,omitempty"`
	Kind     string `json:"kind"`
	Scope    string `json:"scope"`
	// Owner facts, present only when the store knows the id.
	OwnerID     string `json:"ownerId,omitempty"`
	OwnerName   string `json:"ownerName,omitempty"`
	WorkspaceID string `json:"workspaceId,omitempty"`
}

// tmuxAbsentView is a terminal this daemon has a row for whose session is not
// on the server — the other half of the inventory, and the only place where
// "expected and gone" is visible at all. LostAtRestart is ADR-0085's flight
// recorder answering: it was alive at the previous graceful shutdown and did
// not survive the restart.
type tmuxAbsentView struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Session       string `json:"session"`
	WorkspaceID   string `json:"workspaceId,omitempty"`
	LostAtRestart bool   `json:"lostAtRestart,omitempty"`
}

type tmuxServerView struct {
	Installed          bool   `json:"installed"`
	Running            bool   `json:"running"`
	SocketPath         string `json:"socketPath,omitempty"`
	Clients            int    `json:"clients"`
	Version            string `json:"version,omitempty"`
	ExtendedKeysFormat string `json:"extendedKeysFormat,omitempty"`
	Sessions           int    `json:"sessions"`
	PiCodeSessions     int    `json:"picodeSessions"`
	UnclaimedSessions  int    `json:"unclaimedSessions"`
	ForeignSessions    int    `json:"foreignSessions"`
	AttachedSessions   int    `json:"attachedSessions"`
	SampledAt          string `json:"sampledAt"`
}

type tmuxReadView struct {
	Server   tmuxServerView    `json:"server"`
	Sessions []tmuxSessionView `json:"sessions"`
	Absent   []tmuxAbsentView  `json:"absent"`
}

// tmuxReapRequest is the client's receipt of what it showed the human. Every
// field is compared against tmux's own answer immediately before the kill;
// a difference means the row changed under the human and nothing happens.
type tmuxReapRequest struct {
	Confirm   bool   `json:"confirm"`
	SessionID string `json:"sessionId"`
	Created   int64  `json:"created"`
	PanePID   int    `json:"panePid"`
}

type tmuxReapResult struct {
	Killed  bool   `json:"killed"`
	Name    string `json:"name"`
	Session string `json:"sessionId,omitempty"`
	// Audit is "recorded", "failed", or "skipped": the kill has already
	// happened by the time the audit row is written, so a failure is reported
	// rather than dressed up as a failed action (the session is gone either
	// way), and a host with no store records nothing.
	Audit string `json:"audit"`
}

func registerTmuxRoutes(mux Registrar, deps Deps) {
	var src tmuxServerSource
	if deps.Tmux != nil {
		src = deps.Tmux
	}
	registerTmuxRoutesWith(mux, deps, src)
}

// registerTmuxRoutesWith is the registered routes with the tmux source
// injected — the seam the decision-table test drives.
func registerTmuxRoutesWith(mux Registrar, deps Deps, src tmuxServerSource) {
	mux.HandleFunc("GET /api/tmux/server", handleTmuxServer(deps, src))
	mux.HandleFunc("POST /api/tmux/sessions/{name}/reap", handleTmuxReap(deps, src))
}

func handleTmuxServer(deps Deps, src tmuxServerSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		view := tmuxReadView{Sessions: []tmuxSessionView{}, Absent: []tmuxAbsentView{}}
		view.Server.SampledAt = time.Now().UTC().Format(time.RFC3339)
		if src == nil || !src.Available() {
			// No tmux binary at all: an honest empty screen with the reason,
			// never a 503 the app would have to invent copy for.
			writeJSON(w, 200, view)
			return
		}
		view.Server.Installed = true

		info := src.ServerInfo(r.Context())
		view.Server.Running = info.Running
		view.Server.SocketPath = info.SocketPath
		view.Server.Clients = info.Clients
		view.Server.Version = info.Version
		view.Server.ExtendedKeysFormat = info.ExtendedKeysFormat

		sessions, err := src.ServerSessions(r.Context())
		if err != nil {
			writeErr(w, 500, err.Error())
			return
		}
		live := make(map[string]bool, len(sessions))
		for _, s := range sessions {
			live[s.Name] = true
			row := tmuxSessionView{
				Name:      s.Name,
				SessionID: s.SessionID,
				Created:   s.Created.UTC().Format(time.RFC3339),
				Windows:   s.Windows,
				Panes:     s.Panes,
				Attached:  s.Attached,
				Command:   s.CurrentCommand,
				Cwd:       s.CurrentPath,
				PanePID:   s.PanePID,
				Dead:      s.PaneDead,
			}
			if s.PaneDead && s.DeadStatus >= 0 {
				code := s.DeadStatus
				row.ExitCode = &code
			}
			row.Kind, row.Scope, row.OwnerID, row.OwnerName, row.WorkspaceID = tmuxAttribution(deps.Store, s.Name)
			switch row.Scope {
			case scopeYours:
				view.Server.PiCodeSessions++
			case scopeUnclaimed:
				view.Server.UnclaimedSessions++
			default:
				view.Server.ForeignSessions++
			}
			if s.Attached > 0 {
				view.Server.AttachedSessions++
			}
			view.Sessions = append(view.Sessions, row)
		}
		view.Server.Sessions = len(view.Sessions)
		sortTmuxSessions(view.Sessions)

		// The other half: rows this daemon has whose session is not there.
		view.Absent = tmuxAbsent(deps, live)
		writeJSON(w, 200, view)
	}
}

// tmuxAbsent lists the terminals this store holds whose expected session is
// not on the server. Terminals only, deliberately: a terminal is created
// together with its tmux session, so its absence is evidence; an agent can
// legitimately have no session at all (managed mode), and reporting every
// stopped agent as "missing from tmux" would be noise dressed as forensics.
func tmuxAbsent(deps Deps, live map[string]bool) []tmuxAbsentView {
	out := []tmuxAbsentView{}
	if deps.Store == nil {
		return out
	}
	terms, err := deps.Store.ListTerminals()
	if err != nil {
		return out
	}
	for _, t := range terms {
		session := tmux.ShellSessionName(t.ID)
		if live[session] {
			continue
		}
		out = append(out, tmuxAbsentView{
			ID:            t.ID,
			Name:          t.Name,
			Session:       session,
			WorkspaceID:   t.WorkspaceID,
			LostAtRestart: deps.LostSessions[session],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LostAtRestart != out[j].LostAtRestart {
			return out[i].LostAtRestart
		}
		return out[i].Session < out[j].Session
	})
	return out
}

// tmuxAttribution resolves a session name to (kind, scope, owner) using this
// daemon's own records. The name→id step is lossy by construction (tmux
// session ids are sanitized to [a-z0-9-]), which is precisely why a failed
// lookup yields scopeUnclaimed — an uncertainty the UI states and the kill
// re-checks against the session's own marker, never a guess that a session is
// nobody's.
func tmuxAttribution(st *store.Store, name string) (kind, scope, ownerID, ownerName, workspaceID string) {
	switch {
	case tmux.IsShellSession(name):
		kind = kindTerminal
		candidate := strings.TrimPrefix(name, tmux.ShellPrefix)
		if st == nil || candidate == "" {
			return kind, scopeUnclaimed, "", "", ""
		}
		t, err := st.GetTerminal(candidate)
		if err != nil {
			return kind, scopeUnclaimed, "", "", ""
		}
		return kind, scopeYours, t.ID, t.Name, t.WorkspaceID
	case tmux.OwnedSessionName(name):
		kind = kindAgent
		candidate := strings.TrimPrefix(name, tmux.Prefix)
		if st == nil || candidate == "" {
			return kind, scopeUnclaimed, "", "", ""
		}
		a, err := st.GetAgent(candidate)
		if err != nil {
			return kind, scopeUnclaimed, "", "", ""
		}
		return kind, scopeYours, a.ID, a.Name, a.WorkspaceID
	default:
		// Not PiCode's namespace: the user's own tmux session. Read-only.
		return kindOther, scopeForeign, "", "", ""
	}
}

// sortTmuxSessions orders rows for reading: this daemon's own first (agents
// then terminals, by name), then the leftovers, then the user's own.
func sortTmuxSessions(rows []tmuxSessionView) {
	rank := func(r tmuxSessionView) int {
		switch r.Scope {
		case scopeYours:
			if r.Kind == kindAgent {
				return 0
			}
			return 1
		case scopeUnclaimed:
			return 2
		default:
			return 3
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		ri, rj := rank(rows[i]), rank(rows[j])
		if ri != rj {
			return ri < rj
		}
		return rows[i].Name < rows[j].Name
	})
}

func handleTmuxReap(deps Deps, src tmuxServerSource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if src == nil || !src.Available() {
			writeErr(w, 503, "tmux is not available on this machine")
			return
		}
		name := r.PathValue("name")
		// The user's own session is never this app's business — not even to
		// refuse it politely by name, and never to kill it.
		if !tmux.OwnedSessionName(name) {
			writeErr(w, 400, "This session was not created by PiCode, so it cannot be removed here.")
			return
		}
		var req tmuxReapRequest
		d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		d.DisallowUnknownFields()
		if err := d.Decode(&req); err != nil {
			writeErr(w, 400, "Invalid request")
			return
		}
		if err := d.Decode(&struct{}{}); err != io.EOF {
			writeErr(w, 400, "Provide exactly one request")
			return
		}
		if !req.Confirm {
			writeErr(w, 400, "Confirm the removal")
			return
		}

		// The receipt, re-read from tmux now: an action is verified against
		// what the session IS, never against what a list said seconds ago.
		receipt, err := src.SessionReceipt(r.Context(), name)
		if err != nil {
			writeErr(w, 404, "That session is no longer running.")
			return
		}
		if receipt.SessionID != req.SessionID || receipt.Created.Unix() != req.Created || receipt.PanePID != req.PanePID {
			writeErr(w, 409, "That session changed since this page was drawn. Refresh and try again.")
			return
		}

		// Ownership re-check at the last moment, from the session's own
		// marker: if it names a terminal or an agent this daemon still has,
		// the session is live work and the correct door is that surface.
		if deps.Store != nil {
			if owner, door := tmuxMarkerOwner(deps.Store, receipt); owner != "" {
				writeErr(w, 409, "That session belongs to "+owner+", which still exists in PiCode. Remove the "+door+" instead.")
				return
			}
		}

		if err := src.KillSession(r.Context(), name); err != nil {
			writeErr(w, 500, err.Error())
			return
		}

		result := tmuxReapResult{Killed: true, Name: name, Session: receipt.SessionID, Audit: "skipped"}
		if deps.Store != nil {
			result.Audit = "recorded"
			actor := tmuxActor(r)
			err := deps.Store.AppendEvent("tmux.session.reaped", nil, nil, map[string]any{
				"name":      name,
				"sessionId": receipt.SessionID,
				"created":   receipt.Created.Unix(),
				"panePid":   receipt.PanePID,
				"termId":    receipt.TermID,
				"agentId":   receipt.AgentID,
				"actor":     actor,
			})
			if err != nil {
				// The session is gone; only the record is missing. Say so
				// rather than reporting a failure that did not happen.
				log.Printf("tmux: reaped %s but the audit event failed: %v", name, err)
				result.Audit = "failed"
			}
		}
		if deps.Feed != nil {
			deps.Feed.Ephemeral("tmux.changed", map[string]any{"name": name, "actor": tmuxActor(r)})
		}
		writeJSON(w, 200, result)
	}
}

// tmuxMarkerOwner answers "does this session's own marker name something this
// daemon still holds?" and, when it does, which surface owns the session.
func tmuxMarkerOwner(st *store.Store, receipt tmux.SessionReceipt) (owner, door string) {
	if receipt.TermID != "" {
		if t, err := st.GetTerminal(receipt.TermID); err == nil {
			return "terminal " + terminalLabel(t), "terminal"
		}
	}
	if receipt.AgentID != "" {
		if a, err := st.GetAgent(receipt.AgentID); err == nil {
			return "agent " + a.Name, "agent"
		}
	}
	return "", ""
}

func terminalLabel(t store.Terminal) string {
	if strings.TrimSpace(t.Name) != "" {
		return t.Name
	}
	return t.ID
}

// tmuxActor names who asked (ADR-0048 provenance), the same way the Docker
// routes do.
func tmuxActor(r *http.Request) string {
	if p := auth.From(r); p != nil {
		if p.Session.Label != "" {
			return p.Session.Label
		}
		return p.Kind
	}
	return "Local user"
}
