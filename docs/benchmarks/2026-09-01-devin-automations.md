# Study: Devin Automations (and peers) — recurring / event-driven agent runs

- **Date:** 2026-09-01
- **Sources:** Devin product docs
  [Automations](https://docs.devin.ai/product-guides/automations) and
  the owner's own Devin org (`app.devin.ai/org/…/automations`, screenshot
  2026-09-01: empty list, three creation paths, five suggested templates).
  Peers for triangulation: Cursor
  [Automations](https://cursor.com/docs/cloud-agent/automations), Claude
  Code [scheduled tasks](https://code.claude.com/docs/en/scheduled-tasks)
  (+ Routines / Desktop tasks table), OpenAI Codex automations (secondary
  write-ups only; the academy page returned 403), GitHub Agentic
  Workflows (markdown workflows on Actions). Hosted products, no clone —
  claims are from public docs, not receipts. Closed-source internals are
  marked *inference*.
- **Scope:** what shape a PiCode **Automations** surface should take.
  Not a bar for the composer, the Inbox (ADR-0037) or the dashboard
  (ADR-0041/0042) — those are the surfaces an automation *feeds*.

## What Devin ships

| Facet | Devin | Notes |
|---|---|---|
| Triggers | Slack (message, reaction), GitHub (issue comment, PR opened/updated, review, check run, push), Linear (created, label, status, priority, assigned), GitLab, Schedule (RRULE, recurring or one-time), Webhook (POST, regex payload filter) | Several triggers on one automation are **OR**. Webhook auth: `X-Webhook-Secret`, `Authorization: Bearer`, `?secret=`. Payload >200 KB truncated. Secret shown once, regenerate-only. |
| Actions | **Start session** (prompt + event payload as context, `@playbook` include), **Message session** (existing long-running session), **Triage Devin** (persistent Slack monitor), **Email notification** (every run / failures / successes) | The action is *always* a session. There is no "run a script" action — the agent is the runtime. |
| Limits | Max ACU per spawned session; max invocations per time window; network allowlist | All optional. This is the safety story: cost cap + rate cap, not permission prompts. |
| Creation | Manual editor; **Generate with Devin** (describe in chat → config); templates (pre-fill trigger + action + suggested limits) | Screenshot: "Generate" is marked *Most popular*; manual and template are *Less common*. |
| Monitoring | Activity log (timestamp, succeeded/skipped, link to spawned session, error), enable/disable toggle, 30-day sparkline per automation | Sparkline per row is the list's only chart. |
| Templates | Fix Sentry Errors Daily, Daily Error Report (Datadog), Recurring Capacity Planning, Cloudflare Security Audit, Sprint Progress Report (Asana); filters Monitoring & Triage / CI-CD & Release / Security / Project Management | Every template is *schedule → session that reads an external system*. Integrations are the selling point, not the scheduler. |

## Peers — where they agree and differ

| Facet | Cursor | Claude Code | Codex (app) | GitHub Agentic Workflows |
|---|---|---|---|---|
| Runs on | Cloud agents | Cloud (Routines, 1h min), Desktop task (local, 1 min min), `/loop` (session-scoped, 7-day expiry) | **Local app** on a schedule; cloud "coming" | GitHub Actions runners |
| Triggers | Schedule (preset or cron), GitHub/GitLab/Bitbucket, Slack (message, emoji), Linear, Sentry, PagerDuty, webhook | Schedule (cron), API call, GitHub event | Schedule; event triggers | `schedule: cron`, repo events, slash commands, `workflow_dispatch` |
| Config unit | Instructions + model + repo scope + branch + tools + permissions (Private / Team Visible / Team Owned) + **memories across runs** | Prompt + repos + connectors | Prompt + skills + frequency + output destination | Markdown file in repo |
| Delivery | PR, PR comment (inline), Slack message, request reviewers | PR / session result | Thread per run in the app | PR / issue comment |
| Create by NL | `/automate` skill from a local session | `/schedule` skill | UI | write the markdown |
| Safety | Billing pool; tool allowlist | No permission prompts in cloud; configurable per desktop task; jitter; expiry | — | Actions permissions |

Convergence (five of five): **an automation = trigger(s) + a prompt +
bounds, and every run is an ordinary agent session** you can open,
reply to and cost-account like any other. Divergence is only *where it
runs* (their cloud vs the user's machine) and *how it is authored*
(NL-generated is the winning entry point everywhere it exists).

## What PiCode adapts

| Pattern | From | PiCode version |
|---|---|---|
| Run = session | all | A fired automation **creates a normal agent** (ADR-0011 workspace + agent, ADR-0039/0040 own session dir). It appears in the sidebar, the dashboard (ADR-0041) counts its tokens, the Inbox (ADR-0037) receives its result / questions. No second runtime. |
| Triggers are OR; schedule + webhook first | Devin | v1: **schedule** (5-field cron, local tz, jitter like Claude Code) + **webhook** (`POST /api/automations/{id}/fire`, bearer secret, payload appended to prompt, size cap). GitHub / Slack / Sentry are *connectors* layered on the webhook later, not v1 code. |
| Bounds instead of babysitting | Devin (ACU cap, rate cap), Claude Code (expiry) | Per-automation **max cost per run**, **max runs per window**, **concurrency = 1** by default (skip if the previous run is still busy — Devin logs "skipped"). Cost uses the same accounting the dashboard already computes. |
| Activity log with links | Devin | Runs table: fired at, trigger, status (running / done / failed / skipped + why), link to the agent, cost. Sparkline per row reuses `StatTile`'s SVG sparkline (ADR-0042). |
| Generate from NL | Devin "Generate", Cursor `/automate`, Claude `/schedule` | A composer slash `/automate <describe>` that asks the *current* agent to emit the config, prefilled into the editor for confirmation. Manual editor stays; templates ship as JSON in `internal/catalog`-style. |
| Message an existing session | Devin "Message session" | Second action type: deliver the prompt to a chosen agent's queue (follow-up, Track C3) instead of spawning. Refused if that agent runs interactively (`ErrAgentInteractive` — the Inbox already knows this rule). |
| Notify on run | Devin email | Inbox item (`result` on done, `fyi` on skipped/failed) with `sourceKind: automation`. No email. |

## What PiCode refuses

| Temptation | Why not |
|---|---|
| Our own cloud runner | ADR-0003: user-installed `pi`, user's machine. PiCode is the always-on daemon on the owner's box (desktop host, tmux); "requires machine on" is the honest column, like Codex today and Claude Desktop tasks. |
| Slack / Linear / Sentry adapters in core | Integrations are Devin's business model, not ours. Webhook + MCP servers the agent already has cover them; a connector catalog can come as an *app* (ADR-0036). |
| Persistent "Triage Devin" channel monitor | That is a long-running agent + Inbox, which exists. Do not add a third lifecycle. |
| Team visibility / billing pools | Single operator. |
| Scheduler inside pi or inside a session (`/loop` model) | Session-scoped timers die with the tab. The scheduler lives in the Go daemon, persisted in the store, survives restarts, catches up **at most once** per missed window (Claude Code's rule). |
| Email notification | Inbox is the notification surface (ADR-0037). |

## Open questions for the ADR

1. **Model/role of the spawned agent** — inherit the workspace default or
   pin a model per automation (Cursor pins). Proposal: pin, default to
   workspace default (ADR-0028 roles).
2. **Approval gate** — do automation-spawned agents run with the same
   permission posture as an interactive agent? Devin/Claude cloud run
   autonomously; our agents can ask via Inbox and park. Proposal: same
   posture, the Inbox is the gate, plus the cost cap.
3. **Missed fires** while the daemon was down — run once on boot if the
   window passed (Claude Code) vs never (Devin, inferred). Proposal: once,
   logged as `catch-up`.
