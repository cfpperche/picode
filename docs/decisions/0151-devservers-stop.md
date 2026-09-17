# ADR-0151: devserver-stop — the panel may end what it started

- **Status**: accepted
- **Date**: 2026-09-17
- **Boundary**: security model + process — the daemon gains a write that signals a process (a capability it never had), and the panel's read grows three writes
- **Supersedes**: nothing; amends the "one Open per row" shape of the Servers panel recorded in `docs/architecture/devservers.md`

## Context

The Servers panel shipped on 2026-09-15 as one read and one verb: every row was
a button that opened a page in PiCode's work browser. Two things that read got
wrong showed up together on the owner's machine (2026-09-17), and they are the
reason this decision exists:

- **"A port that answered is a server you can open"** was false in a way that
  produced a dead button. The panel's classify step treated any HTTP status
  below 500 as proof of a page, and any socket a PiCode pane's process tree held
  as a row. An agent CLI (`agy`, Antigravity) opens two loopback ports per
  instance — an HTTPS control channel with a bundled certificate on 45683 and a
  CORS API on 46579. Both became rows named `agy` with an **Open** that could
  never work: plain HTTP to the TLS listener answers `400 Client sent an HTTP
  request to an HTTPS server`, and the plain one answers `404 page not found`.
- **A row had no verbs.** The whole row was one button, so the only thing a
  human could do with a listener they no longer wanted — or could not identify —
  was open it. The owner's words: *the terminal does not exist any more and the
  servers are still running.* PiCode had found the fact and could do nothing
  with it.

The capability this needs is not like the others in this repository. Every
existing write acts on PiCode's own records (SQLite) or its own panes (tmux
commands addressed to a session PiCode created). Stopping a listener means
sending a signal to a **process**, and a process tree can contain anything the
user started in that terminal. That is a security-model boundary, not a UI
refinement, so it is decided here rather than in a component.

## Decision

`POST /api/devservers/stop` exists, and it signals **only** a process PiCode can
re-derive as the holder of that loopback socket inside one of its own panes'
process trees. The route re-runs the owner walk at action time and requires the
caller's `pid` **and** the process's start token (`/proc/<pid>/stat` field 22,
the same PID-reuse guard the peer-stop path uses) to match what it just read; a
row that went stale can therefore never signal a process that reused the id.
SIGTERM first, SIGKILL only on a second, explicitly forced request. A port whose
owner PiCode cannot see is readable, openable and hideable, and **never
killable** — the panel says so instead of guessing.

`POST /api/devservers/hide` (and `/unhide`) persists a hide keyed by that same
identity — port + pid + start token — so a Hide can only ever mean *this
listener*: a new process on the same port is a new row, never a silent hole. The
row data grows the identity (`pid`, `startKey`, `startedAt`) and what a port
*is* (`kind`: page / api / opaque, `scheme`), and the panel decides what to show
first: a page, or a listener PiCode started, is visible; a port that only answers
an API, outside PiCode, on a port the list merely guesses at, waits behind
"Show N more".

## Consequences

Easier: the panel answers the question it created. A listener is named by what
it is and who holds it, the two dead `agy` rows now say *HTTPS · not a page* and
offer Stop/Hide, and an unwanted dev server ends in one confirmed click instead
of a hunt through terminals for the pane that started it. The kill is auditable
— every stop appends a `devserver.stopped` event with the port, the process and
whether it was forced.

Harder, and accepted: the daemon can now end a process it did not fork. The
blast radius is one signal to one pid, re-verified milliseconds earlier, inside
a pane PiCode itself launched — but that pane is a shell the user may be working
in, so a Stop can end a build or a long-running download someone forgot about.
The confirmation names the process and its terminal for exactly this reason, and
the panel never offers the verb for a port it cannot attribute. The read also
costs more: a silent port is probed twice (plain, then TLS) before it is called
opaque, and the row decision now consults the live listener set, so a cached
title can no longer keep a stopped server on screen (the defect the first QA
pass found minutes after Stop).

If wrong: the route is one handler behind the ordinary guard, the capability is
one `syscall.Kill` on Linux (the non-Linux build refuses with a sentence), and
removing it restores the read-only panel by deleting one route and one menu
item. Nothing in the data plane depends on it; a stray `devserver.hidden` row is
one table.

## Decision table — when a Stop is allowed

| port held by | pid + start token match | platform | action |
|---|---|---|---|
| a PiCode pane's process tree | yes | Linux | SIGTERM |
| a PiCode pane's process tree | yes, `force` | Linux | SIGKILL |
| a PiCode pane's process tree | no (stale row, replaced process) | any | 409, nothing signalled |
| anything else (no owner, another app, a pid PiCode cannot see) | — | any | 409, nothing signalled |
| — | — | non-Linux | 409: the owner walk needs `/proc` |

Coverage: `internal/server/devservers_actions_test.go` (every refusal row, the
force flag, the audit event) and `devservers_signal_linux_test.go` (a real child
that ends on SIGTERM, and one that ignores it and ends on SIGKILL).

## Alternatives considered

- **Report, never act** (a "Copy the command to stop it" row that types
  `kill <pid>` into the owning terminal): no new capability and no ADR, but it
  makes the human do the work the panel already did, and it types a command into
  a terminal that may be busy running something else — the failure mode ADR-0096
  built an interlock for. Rejected.
- **Kill anything listening on a usual dev port** (attribution optional): the
  panel's whole value is the join between a port and the pane that owns it; a
  kill without that join is a guess with a destructive outcome. Rejected.
- **Auto-stop when the owning terminal is closed**: silently kills work the
  human may have detached on purpose (a dev server is often *meant* to outlive
  its pane), and hides the cause of the death. Rejected — the panel names it and
  lets the human decide.
- **Hide keyed by port alone** (so "hidden" survives restarts): the port is not
  the listener — a new server on 5173 would be born invisible, which is the lie
  the hide is meant to avoid. Rejected in favour of port + pid + start token.
