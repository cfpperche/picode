# ADR-0133: The tmux app — PiCode reads the whole tmux server, and acts only on sessions it can attribute

- **Status**: accepted
- **Date**: 2026-09-13
- **Boundary**: protocol + security model — a new API family (`/api/tmux/*`) beside `/api/apps/{id}/view`, and a rule about which sessions PiCode may remove: it reads every session on the shared tmux server and acts only on sessions whose ownership it can prove.

## Context

PiCode runs every terminal and every interactive agent in a tmux session on
**the user's own tmux server** (ADR-0002, ADR-0016, ADR-0017, and ADR-0025
decision 1, which states the sharing as a choice: the settings catalog reads
the live server "including whatever their tmux.conf set, since PiCode shares
the user's socket"). Every surface PiCode has shows sessions only through a
store row: a terminal or an agent. **A session with no store row is invisible
in the product**, and there is no way to see it without leaving PiCode.

That is not hypothetical. Measured on the development machine:

- 2026-09-13, the isolation branch's own note: the user's server held 147
  PiCode sessions and **136 of them were orphans** from harnesses whose store
  had already been deleted.
- 2026-09-14 (this ADR's author, a script over `list-panes -a`): 152 live
  sessions, 151 with the `picode-` prefix, and **100 of them claimed by no
  reachable daemon** — 8 under an hour old, 68 between one and 24 hours, 24
  over a day, six of them with a client still attached.
- The same measurement, later the same evening: seven live sessions read as
  "claimed by nobody" — and all seven belonged to a **second PiCode instance**
  on the same machine, which the script had failed to reach. The heuristic was
  wrong, not the server.

So the product has one missing surface and one hard question. The surface: an
inventory of the server itself — PiCode's sessions, the leftovers, the user's
own sessions beside them, and the terminals whose sessions are gone (ADR-0085's
flight recorder already records which died in a restart window, and nothing
shows a human). The question: **which of those sessions may PiCode remove?**
"PiCode-shaped name, not in my store" is exactly how a leftover AND another
live instance's session look. No read can tell them apart.

The app also has a model to follow and a model to refuse. Tachyon's tmux
Server Inspector (the reference this was studied from, 2026-09-14) is the same
surface on a **dedicated** socket with an isolated config, a prefix filter as
its ownership rule, a 3-second client poll and a bulk "reap orphans" action
derived from that prefix. The socket and the poll do not fit here (the socket
is the user's by decision; this codebase has a change feed), and the prefix
rule is the one that produced the 2026-09-06 incident this repo already
records as a debt — *"never kill by prefix"*, after a prefix sweep killed 29
sessions.

## Decision

**The tmux app is a native-surface first-party app (`tmux`, ADR-0109) whose
body is a desktop component and whose data is a new API family,
`GET /api/tmux/server` and `POST /api/tmux/sessions/{name}/reap`. The read
covers the whole server in one `list-panes -a` call plus in-process store
lookups, refetched from the change feed and never on a timer. Ownership is
resolved twice: the name is the cheap hint the list uses, and the ownership
markers the session carries in its own environment (`PICODE_TERM_ID`,
`PICODE_AGENT_ID`, injected at creation) are the authoritative receipt read
immediately before any removal. A removal requires an explicit per-session
confirmation and a receipt (session id, creation time, pane pid) that the
server re-reads from tmux and refuses to act on if any field moved; a marker
that resolves to a terminal or agent this daemon still holds refuses the
removal and names the surface that owns it; a session outside the `picode-`
namespace is never actionable. Every removal appends one durable
`tmux.session.reaped` event. The app states, in the row, that a PiCode-shaped
session missing from the store may belong to another PiCode on this machine —
because that is true and no read can settle it.**

Deliberately not decided here: isolating PiCode's own sessions onto a private
socket (the harnesses got one in the `tmux-isolation` branch; the daemon keeps
the user's server), and attaching to a session no store row claims. Both are
separate decisions with their own cost, named in the plan.

## Consequences

**Easier.** The 136-session class of leak becomes visible and removable from
inside the product, one confirmed row at a time, with an audit trail. A
terminal whose session died — including one the flight recorder saw die in a
restart — is finally shown. The user's own tmux sessions are readable in the
same screen that manages PiCode's, and are marked as not PiCode's rather than
hidden.

**Harder.** The app's copy has to carry an uncertainty no other surface in
PiCode carries ("this may be another instance's"). Two ownership reads exist
and must stay consistent with the three writers of the markers
(`internal/server/terminals.go`, `internal/store/agents.go`, the CLI launch
path), which a test pins. The read is a subprocess on open and on feed events;
a timer against it would be a subprocess every few seconds per open tab, so
none is allowed.

**Cost accepted.** A human can remove a session that another PiCode still
uses, if they confirm it: the server cannot prove otherwise, and refusing all
removals would leave the leak unmanageable. The blast radius is one tmux
session (the work inside it is lost); the audit event names the actor.

**If we are wrong.** If the marker read ever fails open — treating "no marker"
as "nobody's" without the receipt and the confirmation — this app becomes the
prefix sweep this repository already paid for. The decision table in
`internal/server/tmux_test.go` exists for exactly that: every row of it is a
way to kill a live session, and every row is asserted.

## Alternatives considered

| Alternative | Why it lost |
|---|---|
| Primitives app (list/detail/actions, ADR-0036) | The body needs a receipt-carrying action and a foldable per-session detail with live facts; the primitives vocabulary is frozen and generic, and the Canvas already established the native surface for a first-party body (ADR-0109) |
| Prefix filter as ownership (the Tachyon model) | Measured wrong twice: it is the rule behind the 29 killed sessions, and it cannot distinguish a leftover from another live instance |
| Name-based attribution only (no marker read) | Names are sanitized (`[a-z0-9-]`), so the name→id mapping is lossy; the marker is what makes the refusal safe |
| Removing the bulk "reap all leftovers" action | It would be the one-click version of the 2026-09-06 sweep. The inventory plus per-row confirmation is slower by design; the leak was 136 sessions, not 136 clicks |
| A timer (3s) instead of the feed | The feed already carries `tmux.changed` and `terminal.*`; a timer would spawn tmux subprocesses forever on a screen nobody is watching |
| Isolating the daemon onto its own tmux socket first | A different decision (it changes ADR-0025's premise that the catalog shows the user's real server). Sequencing it first would have delayed the visibility the leak needed; the app reads whatever socket its daemon uses, so it follows that change without edits |