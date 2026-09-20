# ADR-0144: Developer mode — raw CDP for a full-tier agent

- **Status**: accepted
- **Date**: 2026-09-16
- **Boundary**: security model — the work-browser command channel grows a
  path that is not in the curated catalog: an agent may name a CDP method
  directly. What crosses is authority: the tier gate stops being the only
  thing between an agent and the browser's internals.

## Context

ADR-0128 fixed the shape of agent browser access: a closed catalog of CDP
methods per tier (`read` < `act` < `full`), deny by default, checked by the
daemon (`internal/browser/verbs.go`) and re-checked by the shell
(`desktop-shell/src/cdppolicy.rs`), with the loopback debug port as an
explicit opt-in for *external* tooling and no debug port in default
operation.

Two things have changed since.

The reference (ChatGPT Work, owner 2026-09-16) ships a settings block the
owner wants: **Developer mode — Enable full CDP access**, labeled *Elevated
risk*, off by default, in the same page as the browser grants. The owner
agreed with the shape proposed for our model: give the *agent* raw CDP
through the bridge we already trust, and keep refusing the loopback port in
the UI.

And the pressure is real: the curated catalog is what makes agent browsing
safe, but it is also what makes it *incomplete* — anything WebView2 adds,
or anything a specific page needs (interception, storage inspection, a
protocol method we never listed), is refused with no honest way for the
owner to say "I want this agent, on this machine, to have that". The
alternative they reach for today is the `PICODE_CDP_PORT` escape hatch,
which is strictly worse: it opens a port any local process can attach to.

## Decision

The daemon grows one verb, `cdp`, which carries a method name from the
agent instead of resolving one from the catalog. It is granted only when
**both** hold:

- the machine setting `browser.developerMode` is on (default off, fail
  closed: an unreadable setting means off), and
- the calling principal's resolved tier is `full` (agents and terminals
  alike, ADR-0143).

Both are checked in the daemon — the tier as it does today, the setting
next to it — and the shell re-checks them: `btab_cdp_call` takes a `raw`
flag that bypasses the catalog only when the shell's own copy of the
setting is on (pushed by the page like `btab_set_scripts`) and the tier the
command carries is `full`. A raw call never travels with a lower tier, and
a refusal names Settings ▸ Browser ▸ Developer mode.

Every raw call — allowed or refused — appends one `browser.cdp` event with
the principal, the method, and the outcome, so "audited" is a row in the
change feed and not a promise. Allowed calls are visible in Settings ▸
Browser (the card's last-call line, and the agent's own conversation, which
already shows the tool call).

The loopback debug port is *not* part of this toggle. It stays an
environment variable (`PICODE_CDP_PORT`) for sanctioned one-offs, and if it
ever earns a UI row it gets its own line and its own decision, because
"any local process can attach" and "this agent, audited" are different
risks that must not share a switch.

## Consequences

Easier: an owner who needs the whole protocol turns it on for one machine,
`full`-tier agents only, and sees every call. The catalog keeps working
unchanged for everyone else, and the refusal message stops being a dead
end.

Harder / accepted as cost: with the mode on, the tier gate is the only
remaining wall — a `full` agent can reach what the catalog deliberately
denies: `Network.getAllCookies`, `Storage.*`, `Fetch` interception, and
anything a future WebView2 adds. The setting is per machine, not per site:
a compromised or confused `full` agent burns the whole profile's sessions.
The audit is a record, not a boundary — it tells the owner afterwards.

If we are wrong: the blast radius is the work profile's cookies and
sessions for agents the owner already granted `full`. Turning the toggle
off closes the door immediately (the daemon refuses on the next call; the
shell is re-checked per call), and nothing in the profile is destroyed by
having been read.

## Alternatives considered

- **Expose the loopback port in the UI (the ADR-0128 reading).** Refused:
  it serves external tooling, not the agent, and any local process can
  attach — a hole that no tier and no audit can narrow. If it is ever
  wanted, it is a separate row with its own label.
- **Per-site or per-agent grants for raw methods.** Refused for v1: the
  reference has one machine switch, we have no vocabulary for "which
  methods", and a per-agent grant over a method list nobody has written
  yet is a table we would have to redesign. Durable refinement is
  `browser.developerMode` + the tier that already exists.
- **Widen the curated catalog instead.** Refused: it is the thing that
  forces every new method through a review we cannot do at WebView2's
  release cadence; the escape hatch is honest about who decided.
- **Ship the toggle without the audit.** Refused: the owner's ask is
  "elevated risk, off by default" — an elevated-risk switch with no
  after-the-fact record is exactly the control the reference warns about.
