# ADR-0044: The mobile shell is a supervision console, not the desktop shrunk

- **Status**: accepted
- **Date**: 2026-09-01

## Context

PiCode has had two shells in one Vite app since the phone split
(`web/src/desktop`, `web/src/mobile`, chosen at boot by `max-width: 767px`
or `?mobile=1`). The mobile one was a 381-line reduction of the desktop:
Agents | Chat | Term | More, no routes, only the first agent of each
workspace, a chat that ignored `extension_ui_request` (so a permission
prompt never showed — "Mobile has no waiting card" sat in the handoff
twice), no abort or steer, no way to create an agent, no Inbox
(ADR-0037's whole data plane was unreachable), and a More menu that
mounted Providers and MCPs with empty props. The backlog called the gap
"mobile parity".

The owner reframed it on 2026-09-01: the phone does not need to mirror
the workstation. Its job is to **observe the system, answer the agents'
calls, decide, and act quickly** — create an agent, talk to it, interrupt
it — while away from a desk. The benchmark study
(`docs/benchmarks/2026-09-01-mobile-agent-supervision.md`) found every
shipping product in this space converging on the same shape: Claude
Code's Remote Control, Codex in the ChatGPT app and Cursor for iOS are
all supervision surfaces (approve, redirect, reply, review), none is a
mobile IDE; the on-call reference (PagerDuty) puts the decisions that
need you at the top, sorted by urgency, with the actions under the thumb.

## Decision

**Four tabs and one pushed screen.** `Now` (home): the "Needs you" queue,
then who is running, then today's spend/activity/sessions, then the last
finished runs. `Inbox`: the ADR-0037 app through the same host surface
the desktop uses. `Agents`: every agent, grouped by workspace, with
Start/Stop and a "+" sheet (workspace, free agent, agent in a workspace,
or adopt a pi session). `More`: the rest of the product one tap deep,
mounting the desktop page components with real props. The **agent
screen** is pushed, not a tab: name · state, model · cost · context, the
shared `Conversation` with its ask card, the shared `Composer` (its Stop
is the abort; dictation comes with it), and a Chat | Terminal segment
only for an agent that lives in a tmux TUI.

**Needs-you is a merge of two sources.** Live dialogs first — the agent is
blocked, and pi's dialogs expire — then blocking inbox items, newest
first. Both are answerable in place: a dialog through
`POST /api/agents/{id}/ui`, an inbox item through
`POST /api/inbox/{id}/respond`.

**One server change.** `agentView` (`GET /api/workspaces`,
`GET /api/agents?free=1`) carries `streaming`, `waiting` and the open
`dialog` from the managed runtime's snapshot. The fleet poll the phone
already needs now says who is waiting and what they are asking; no
socket per agent, no new endpoint. The desktop ignores the fields for
now (its sidebar can adopt them later in place of `waitingId`).

**A pure reducer, not a shared handler.** `lib/agentEvents.js` carries the
desktop `handleEvent`'s item-level semantics (snapshot with dialog,
deltas, tools, bash, enqueue, message/turn end, ui request/timeout,
exit, task failed, rejected) as `reduceAgentEvent(state, ev) → {state,
effects}`, on the helpers both shells already share (`lib/askForm.js`,
`assistantMsg.js`, `diff.js`, `searchCards.js`, `piError.js`). The
desktop's inline switch is untouched: zero regression surface there,
and the reducer is the first unit-tested version of that logic.
Recorded debt: the desktop may adopt it.

**Hash routes on the phone.** `#/`, `#/inbox[/id]`, `#/agents`,
`#/agent/<id>`, `#/more[/section]`. The agent route is the desktop's, so
a QR scan or a pasted link opens the same agent; other desktop hashes
map to the closest section. Tabs `replace`, pushed screens `push`, so
Android's Back closes the agent screen instead of leaving the PWA; the
on-screen Back always goes to the parent. No remembered tab — Now is
the home because the question is always "what needs me".

**Explicitly not on the phone.** Editor, file tree, git graph, terminal as
a tab, the desktop's tab strip, and any per-agent context-window fetch
from the home (one `/status` call per running agent; refused until an
aggregate exists). Push notifications are the next ADR (Web Push over
VAPID, Go stdlib), not this one.

## Consequences

- **Easier**: a permission prompt can finally be answered from a phone;
  the Inbox exists on the phone; a stopped agent can be started, a
  running one stopped, a runaway turn aborted, a new agent created —
  each in one or two taps in the thumb zone.
- **Harder**: the mobile shell is now a small app (five screens, four
  hooks, seven components, ~350 lines of CSS) rather than one file; two
  shells share more components with different frames, and a change to
  `Conversation`/`Composer`/`AppSurface` must be looked at on 390px too.
- **Accepted cost**: the agent screen replays the last 200 transcript
  events on open (the desktop's own tail); a very long session on a slow
  link shows the socket's live events first and the history a moment
  later. The fleet poll runs at 5 s on Now/Agents (15 s elsewhere) — the
  ADR-0036 badge poll's own "seconds of latency is fine" reasoning.
- **If wrong**: `main.jsx` still picks the shell by viewport; the old
  `?desktop=1` escape and the More → Desktop layout row remain.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Finish "mobile parity" (file tree, git graph, editor at 390px) | The products people actually use from a phone are all supervision surfaces; editing on a phone is the one thing every one of them refuses (Cursor: "not a mobile IDE"). |
| A native app (AgentWatch / Nimbalyst have Live Activity, watch) | ADR-0003's one-binary posture; the PWA already installs to the home screen, and push (next ADR) covers the alert. Live Activity is the one thing a PWA cannot do — accepted. |
| Extract the desktop `handleEvent` into a hook both shells call | Entangled with six refs and five side effects; a shared mutation would put the desktop at risk to ship a phone feature. The pure reducer gets the tests without touching it. |
| A `GET /api/agents/waiting` endpoint | The phone polls the fleet anyway; three fields on the view it already reads is smaller than a new route and a second poll. |
| Keep Chat and Terminal as tabs | A tab must always have something to show; a chat tab with no agent picked is a dead end (the old shell's "Pick an agent."). Pushed screens carry their agent in the URL. |
| Remember the last tab (PagerDuty does) | The home is the queue of decisions; coming back to Agents when something is waiting is the wrong default for this product. |
