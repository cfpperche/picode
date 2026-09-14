# ADR-0132: browser-command-channel

- **Status**: accepted (owner, 2026-09-13 — agreed in session)
- **Date**: 2026-09-13
- **Boundary**: protocol and security model — how a browser command travels
  from the daemon (where an agent's request is born) to the desktop shell
  (the only process that holds the page), and where the access decision is
  made on the way.

## Context

Increment 1–2 of Phase 3 (ADR-0128) shipped the shell half: `btab_cdp_call`
and `btab_cdp_events` reach a tab's page over the WebView2 host API, gated by
a named method catalog per tier, and the loopback debug port is no longer
opened by default. What does not exist is the path *to* those commands: the
daemon has no way to reach the shell.

The direction is structurally one-way. The shell knows where the daemon is
(it reads `server.json` through `wsl.exe` and loads the UI from it), and its
page already holds an authenticated session with the daemon — the change feed
(`GET /api/events`, ADR-0048) and every `/api/*` call ride that session. The
daemon, by contrast, has no address for the shell, and the shell listens on
no port. `internal/` routes are the daemon's outward face; nothing dials in.

Forces:

- The feature is time-sensitive to the user's eyes: an agent driving the page
  the human is watching should not visibly lag behind.
- ADR-0128's default is *no open port* — an unauthenticated local listener is
  the thing its opt-in toggle exists to avoid.
- The shell's Rust half has no credential of its own; the login lives in the
  embedded browser profile (the page's cookie), not in the process.
- The decision of *what an agent may do* belongs to the daemon (it knows agent
  identities); the *delivery* must not be able to grant more than the policy.
- `PICODE_CDP_PORT` (loopback debug port) already exists as the owner's
  explicit escape hatch for external tooling; it is not this path.

## Decision

**The shell opens the line and the daemon pushes down it.** Concretely:

1. The shell's page opens one `EventSource` on `GET /api/browser/stream`
   (authenticated by the same session cookie as every other `/api/*` call; it
   is the same streaming shape the change feed already uses, but a separate
   route, because a command is point-to-point and not a change the ledger
   should replay).
2. When an agent's tool call arrives, the daemon resolves that agent's policy
   — `{tier, domains}` (ADR-0128) — and hands the command to the browser hub,
   which assigns an id and writes it to the connected shell as an SSE frame
   `browser.command`: `{id, method, params, tier, domains}`.
3. The shell re-checks the method against its own catalog for that tier
   (already shipped), applies the domains as a navigation gate, runs the
   command, and posts the result to `POST /api/browser/result` with the id.
   The hub completes the waiting tool call with that result, or times out.
   In v1 the command carries no tab id: it runs against the work-browser tab
   the human has on screen, which is the page the product is about. Targeting
   a named tab needs the tab-list surface and arrives with it.
4. A command queued while no shell is connected fails fast with "the desktop
   app is not connected" — the tool does not hang waiting for a window to
   open. Reconnection is the EventSource's own (the feed's behavior: cursor
   and reconnect), so a restarted shell rejoins without the daemon noticing
   more than a new subscriber.
5. No new port, no new credential, and no second authentication path: the
   page proves itself exactly as it does for the rest of the UI, and the
   daemon stays the policy decision point.
6. One verb is shell-local, not a CDP method: `shell.events` returns the tab's
   recorded event ring (increment 1) and never reaches the page. It is
   read-tier by construction — the ring's catalog is fixed and read-only.

## Consequences

- Delivery is immediate while the window is alive; a background/minimized
  window may be scheduled late by Windows, which only delays something nobody
  is watching. There is no polling interval to tune.
- The shell cannot be *started* by an agent: with no window there is no page
  to drive, and the tool answers that plainly. This is the accepted cost of
  not giving the daemon a way to dial in (`--force`-style dialing, even with a
  token, is exactly the surface ADR-0128 refuses by default). If a real need
  appears, the deliberate escape is a new ADR for an authenticated listener —
  not a quiet addition here.
- If we are wrong about the shell being the only consumer, the blast radius is
  bounded by the two gates: the catalog refuses unnamed methods, and the
  domains gate refuses origins outside the agent's allowlist. Neither is
  advisory: the first is host-side in Rust, the second cancels navigation at
  the webview.
- The hub is in-process state (pending requests keyed by id). A daemon restart
  drops in-flight commands; the tool call fails with a clear error, and no
  cursor is needed because commands are not replayed.
- A timeout ends the *wait*, not the command: one already on the stream may
  still run, so an `act` caller must treat "no answer" as unknown, not as
  "nothing happened" (a cancel frame is future work). At the read tier this
  costs nothing.
- The stream carries command *contents* — method and parameters, which may
  include a page reading. It travels only over the authenticated session, the
  same channel that already carries the transcript. Loopback HTTPS with the
  paired certificate is the transport; there is no cleartext hop.

## Alternatives considered

- **The shell polls a queue** (`GET /api/browser/pending` on a timer). Rejected:
  it makes the agent's latency a function of a tick we have to choose, and the
  user would feel it — the whole reason this ADR exists is that the shell can
  *hold* a stream (it already holds the feed's).
- **The daemon dials a shell-side listener with a shared token** (the v1
  sketch). Rejected for now: it opens a port on the machine, so the token
  becomes the entire defense, and ADR-0128's first principle is that the
  allowlist is structural. It remains available later as an explicit,
  owner-approved escape with its cost stated, in the shape of the existing
  `PICODE_CDP_PORT` toggle.
- **Abuse the shell's Rust half as the client** (a second authenticated HTTP
  client in Rust). Rejected: the credential lives in the WebView2 profile, so
  the Rust half would need its own token — a second authentication path for
  the same privilege, which is more to review, not less.
- **Ride the change feed** (`feed.Ephemeral("browser.command", …)`). Rejected:
  the feed is a broadcast ledger of committed changes (ADR-0048) with replay
  semantics; a point-to-point request with a result is not a change, and every
  other subscriber would receive commands it cannot answer. A separate route
  costs a handler and says what it means.
