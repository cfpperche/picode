# ADR-0146: agents may read the browsing history, and only when told

- **Status**: accepted
- **Date**: 2026-09-17
- **Boundary**: security model — a new class of the human's data becomes
  reachable by an agent, and the closed verb catalog grows one entry to carry
  it. What crosses: the browsing history store stops being chrome-only read by
  the settings page and becomes an agent-reachable surface.

## Context

Slice 3 gave the work browser a history store (`browser_history`: url, title,
favicon, visited_at) with its own dialog — search, day groups, per-row and bulk
delete. It is the human's record of where they have been, and until now only
the human's own pages read it.

Agents already reach the browser through a **closed** verb catalog (ADR-0134):
`snapshot`, `screenshot`, `events` at `read`; `evaluate`, `navigate` at `act`;
`cdp` (raw) at `full`, behind Developer mode (ADR-0144). Every one of those
verbs is about *a page the human has in front of them*: the tab on screen, or
one the agent may drive inside its grant's domains. Reading history is a
different kind of reach — it is not a page, it is everywhere the human has
been, including sites no grant covers and pages never opened in PiCode at all
(the shell reports visits from every work-browser tab).

The owner's scope (2026-09-15) asked for agent history access on the reference's
shape: *Always ask / Allow / Never*, reusing the policy machinery. Measured
2026-09-17, that machinery is two different things: the daemon's policy+prefs
(which a verb gate reads on every call) and the shell's **Ask bar** (a deferral
on a live `PermissionRequested`, per tab). A history read has no tab and no
request to defer, so "ask" needs a prompt door of its own — the inbox has
ask/reply already, the bar does not fit. Shipping a third state that can never
fire would be a control that lies.

## Decision

One new verb, `history`, in the catalog with tier `read` — it drives nothing —
and **its own gate**: the setting `browser.historyAccess`, `never` by default,
`allow` when the human says so. The tool handler answers the verb from the
store directly (no shell round-trip, so it works with no browser tab open) and
refuses with the reason and the place to change it when the setting is not
`allow`. The master `Browser` switch keeps gating every verb as it does today.

What the verb returns is bounded on purpose: `{url, title, visitedAt}` rows, a
capped limit, and an optional query — never page content, never cookies,
session storage or credentials. The human's own pages keep reading the same
store; nothing about the dialog changes except a row that says what agents can
do.

`ask` is **deferred, named, not faked**: it lands when the daemon has a prompt
door for a request with no tab. The two candidates are the inbox's ask/reply (a
message the human answers) and a daemon-side prompt surface in the app; the
choice is its own decision because it changes how a blocked agent turn behaves.

## Consequences

- The human decides once, plainly, and the row states the truth: with `never`
  (the default) an agent's history call is refused and says where to change it.
- An agent that is allowed can enumerate where the human has been, including
  sites outside its grant. That is the capability, and it is why the default is
  `never` and why the verb returns no content — the risk is the list itself.
- Because the verb is answered by the daemon, history reads work with no
  browser tab open and no shell running; the gate therefore cannot live in the
  shell's policy map.
- If we are wrong about "ask": an agent's request is refused instead of
  prompting. Nothing is silently allowed, and the refusal names the setting, so
  the failure is a missing prompt, not a wrong answer.
- The catalog grows: `history` is the first verb about data that is not a page.
  A second one (downloads, bookmarks) should reuse this shape — its own gate,
  its own default, one row in the settings page.

## Alternatives considered

- **Gate history by the existing per-agent tier** (a `read` grant reaches it).
  Lost: a grant is about what an agent may do *to pages*, and most grants are
  written for a task, not for the human's archive. It would also make history
  reachable as a side effect of raising a tier, which is how capabilities leak.
- **Ship all three states, with "ask" behaving like "never" until the door
  exists.** Lost: a state that looks like a choice and behaves like a refusal
  is the dead control this project keeps refusing to ship.
- **Let the shell answer the read** (it holds the visits it reports). Lost: the
  store is the daemon's, the shell reports into it, and a read must work with
  no tab open — the same reason the settings page reads the daemon.
