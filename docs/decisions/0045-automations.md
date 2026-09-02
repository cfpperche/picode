# ADR-0045: Automations — scheduled and webhook-triggered agent runs

- **Status**: accepted
- **Date**: 2026-09-01

## Context

Every async-agent product now ships the same feature under the same name:
Devin *Automations*, Cursor *Automations*, Claude Code *Routines*, Codex
*Automations*, GitHub *Agentic Workflows*. The study in
[docs/benchmarks/2026-09-01-devin-automations.md](../benchmarks/2026-09-01-devin-automations.md)
finds one shape across all five: **an automation is trigger(s) + a prompt
+ bounds, and every run is an ordinary agent session** the user can open,
reply to and cost-account like any other. They differ only in where the
run executes (their cloud vs the user's machine) and in how the config is
authored (natural language wins wherever it exists).

PiCode already has the pieces a run needs: agents that spawn `pi --mode
rpc` headlessly and keep running with no browser attached (RPC runtime in
the daemon), per-agent sessions with a minted id (ADR-0039/0040), a
durable task queue (`follow_up`), cost accounting over session files
(ADR-0041/0042), and the Inbox as the human's mailbox (ADR-0037). What it
lacks is *when*: nothing in the daemon fires without a person at the
composer.

Constraints in play:

- ADR-0003: user-installed `pi`, no cloud runner. The machine must be on.
  The daemon already is a service (systemd `--user`, Windows tray
  keepalive), so "always on while the machine is" is honest.
- AGENTS.md #3: stdlib first. A cron library is the obvious temptation.
- ADR-0036: the apps host's primitives are frozen at
  list / detail / form / actions.
- `docs/design/session-surface-roadmap.md` refuses a **system crontab**
  for package auto-update. That refusal reads as a general "no machine
  cron" stance and must be reconciled, not ignored (ADR-0041 precedent).
- Owner decisions (2026-09-01): one agent per automation with a fresh
  session per run; a core React route, not an app; presets over raw cron
  with a hand-rolled matcher; v1 = schedule + webhook + Run now, editor,
  Inbox; templates and NL generation later.

## Decision

PiCode gains **Automations**: rows in SQLite (`automations`,
`automation_runs`, migration 016) fired by a one-minute scheduler
goroutine in the Go daemon (`internal/automate`, modelled on the backup
loop) and by an HTTP webhook. An automation has a name, an action, a
prompt, optional model pins, a 5-field cron (`internal/cron`, stdlib,
vixie day semantics) and/or a webhook secret (stored as a sha256 hash,
shown once), and bounds: max cost per run, max runs per window,
concurrency 1. **Action `start`** creates one agent for the automation on
first use (named after it, in its workspace) and runs each invocation as
a **fresh pi session on that agent**: the agent shows in the sidebar, the
dashboard counts its tokens, its sessions are its history. **Action
`message`** queues the prompt as a `follow_up` into an existing agent.
Runs are recorded (trigger, status running / done / failed / skipped,
reason, session path, cost); the Inbox receives one `result` per finished
run carrying the agent's real final text, and one `fyi` per state change
that stopped a run (busy, rate cap, cost cap, pi missing, target gone,
agent in terminal, daemon restarted). The surface is a core route
`#/automations` — list with a 30-day sparkline and Run now, an editor
with Hourly / Daily / Weekdays / Weekly presets and an Advanced cron
field, a runs table — reached from the user menu and the palette.

The scheduler is **in the daemon, not the machine's crontab**: the
roadmap's refusal stands for package auto-update (a machine cron acting
without the product) and is not what this is. Jitter is deterministic per
automation and bounded by half the interval (≤ 30 min); a daemon outage
collapses to **at most one** catch-up run; a run interrupted by a restart
is marked failed with the reason on the row.

The decision table (`decideFire`, tested row by row):

| # | Condition | Outcome |
|---|---|---|
| 1 | disabled, schedule or webhook | nothing recorded (webhook answers 409) |
| 2 | disabled, Run now | runs — the click is explicit |
| 3 | previous run still running | `skipped / busy` (Run now answers 409) |
| 4 | runs in window ≥ max | `skipped / rate cap` |
| 5 | pi not on PATH (start) | `failed / pi missing` + fyi |
| 6 | automation's agent deleted | recreated on the next run |
| 7 | target or own agent in a terminal | `skipped / agent in terminal` |
| 8 | message target deleted | `failed / target gone` + fyi |
| 9 | webhook: bad or missing secret | 401, nothing recorded |
| 10 | webhook: body > 64 KB | 413, nothing recorded |
| 11 | session cost > cap (polled every 30 s) | abort, `failed / cost cap` + fyi |
| 12 | daemon restarted with a run running | `failed / daemon restarted` + fyi at boot |
| 13 | run longer than 2 h | abort, `failed / timeout` + fyi |

## Consequences

- **Machine on, or nothing runs.** The honest column in the Claude Code
  comparison table. The systemd unit / tray keepalive already make the
  daemon long-lived; a laptop lid closed is a missed slot, caught up once.
- **Runs are agents.** Everything that exists for agents (sessions picker,
  dashboard, Inbox park-and-wake, `--session-dir`) applies for free; the
  automation's agent can be opened and talked to like any other. Cost of
  a run is the cost of its session file, computed with `session.Summarize`.
- **Cost cap is per message.** Amended the same day: pi reports usage
  after every assistant message (`message_end`), the runtime sums it and
  the run observer closes the run on the message that crosses the cap.
  The 30 s poll remains for the session path, the two-hour timeout and a
  file-based fallback when a provider reports no usage. A run can still
  overshoot by the single message that crosses the line.
- **Polling, no bus.** The page polls every 15 s (paused when hidden),
  the same posture ADR-0037 chose for the Inbox. Live runs update on the
  next poll, not the next event.
- **Shared credential.** Two automations due in the same minute run two
  agents against whichever account is active in `auth.json` (pi
  limitation, already a recorded debt).
- **Webhook is the first un-authenticated-by-trust route.** It is meant to
  be called from beyond the browser, so it carries its own per-automation
  secret (constant-time compare, 64 KB body cap, payload appended to the
  prompt as text and never parsed). Exposing it beyond the tailnet
  re-opens ADR-0007's token-auth debt; that stays the owner's call.
- **Inbox rebuilt.** `inbox_items.source_kind` gains `automation`
  (migration 017 recreates the table for the CHECK). Automation items
  are not replied to; `RespondAndForward` forwards only agent items.
- **Later, not now:** templates as JSON, `/automate` natural-language
  generation, connectors (GitHub / Slack / Sentry) layered on the webhook,
  a cost value on runs closed by a restart.

## Alternatives considered

- **ADR-0036 app on frozen primitives.** List and form fit; a schedule
  editor with presets and time inputs, a per-row sparkline and a runs
  table do not, and the primitives are frozen by amendment. A core route
  costs JSX but keeps the vocabulary honest.
- **`robfig/cron` or another cron library.** ~150 lines of stdlib cover
  the grammar every peer documents (no names, no `L`/`W`); the web
  mirrors it in `lib/cron.js` with the same tests.
- **A new agent per run.** Simplest code, but an hourly automation is 24
  sidebar rows a day and 24 session dirs for the orphan sweep. One agent
  with a session per run is what Devin's "sessions under one identity"
  amounts to.
- **System crontab / systemd timers.** Would fire with the daemon down and
  then have nothing to talk to; and it is exactly the roadmap's refusal.
  The daemon is already the long-lived process.
- **Cloud runner.** ADR-0003.
- **Email notifications (Devin).** ADR-0037: the Inbox is the channel.
- **Scheduler inside pi or session-scoped (`/loop`).** Dies with the tab.
- **Persistent channel monitor ("Triage Devin").** A long-running agent
  plus the Inbox already is that; no third lifecycle.

## Amendment 2026-09-01 — v2: `/automate` and templates

Devin's own ordering (Generate = most popular, templates second, manual
editor "less common") held up: the v1 editor is the slow path. v2 adds the
two faster ones without a new object or lifecycle — both end in the same
editor, pre-filled, for the user to confirm.

- **`/automate <description>`** sends the **current agent** a prompt that
  asks for one ```json fence (name, prompt written for a context-free
  future run, cron, webhook, limits) after looking at the repository. The
  App correlates the turn client-side (the pending `/automate` is a ref;
  `agent_settled` reads `lastAssistantText`), parses the last fence or the
  first balanced object, normalises it, hands it to the editor through a
  read-once `sessionStorage` draft and navigates to `#/automations/new`.
  The bubble in the chat shows `/automate …`, not the instruction block.
  No fence → the editor opens with the description as the prompt. No
  agent open → one toast and the list. No server change: the turn is
  ordinary, the cost is the agent's, the reply stays in its session.
- **Templates** are a Go slice (`automate.Templates()`, seven local-repo
  jobs in four categories) served by `GET /api/automations/templates`,
  shown as cards with category chips on the list (always open when the
  list is empty, a remembered `<details>` otherwise) and as a *Start from
  template* select in the editor. Choosing one over typed text asks first.

Refused: a hidden one-shot `pi` spawn for generation (loses the repo
context the open agent has, costs a process); a `pi-automate` tool
package (structured, but a third package to install before the fast path
works — revisit if fence parsing proves unreliable); templates that need
Sentry/Datadog/Slack (webhook recipes belong in the guide, not the chrome);
a marketplace.

## Amendment 2026-09-02 — the webhook behind a gateway

Tracks C/D (ADR-0051/52) put member daemons behind `picode gateway`,
which identifies every request — and a CI job has no identity. The
gateway gains `POST /-/hook/<linux user>/<automation id>`: no identity,
sixty calls a minute per caller, the user must be a member, and the
request is proxied to that member's `/api/automations/<id>/fire` with
`Authorization` passed through (the automation's secret is the
credential; the daemon still checks it) and cookies dropped. The
automation view now carries `webhookUrl`, computed by the daemon from
where it is (own origin, public URL, or the gateway form), so the detail
page shows the URL a caller can actually use. Recipes (GitHub Actions,
Sentry, cron) live in the guide.
