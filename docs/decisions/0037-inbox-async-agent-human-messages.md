# ADR-0037: Inbox — async agent↔human messages; data plane in core, view as the first app

- **Status**: accepted (depends on ADR-0036; the Inbox is its first app)
- **Date**: 2026-08-31

## Context

Today an agent's question mid-turn is visible only if its session tab is
open; a finished run announces itself to nobody; terminals have no channel
to the human at all. The operator babysits tabs — the failure mode every
async-agent product wrote about. Answer.AI's month with Devin found
outcomes hinge on how fast the human answers mid-task questions; GitHub's
Copilot coding agent ships with no push when the agent needs input, so
blocked work silently rots.

The benchmarks converge on a data model and on triage semantics:

- **Item kinds.** Every surveyed system reduces to four: FYI notification,
  question, approval request, result delivery (LangGraph interrupts,
  HumanLayer, gotoHuman, Devin sessions, Copilot PRs, Codex tasks).
- **Response verbs.** LangChain's Agent Inbox schema is the most converged:
  `accept | edit | respond | ignore`, with per-item allow-flags declaring
  which verbs an item supports.
- **Reasons.** GitHub notifications' single best idea: every item carries
  *why you got it* (assigned, mention, review-requested…). Products that
  dump a flat chronological list (most agent tools) drown the user;
  Octobox exists because GitHub mixed actionable with FYI.
- **Triage.** unread → read → done, plus snooze-until-T (Linear) and the
  archive-resurfaces-on-activity vs mute-forever distinction (Octobox).
  Blocking items are not triageable — they belong in a queue where every
  item demands a decision (Linear Triage); answering *is* the done.
- **Blocked vs continue.** Devin's model won: the agent parks in `blocked`
  and any later human message wakes it — the human never needs to know
  whether the agent is technically paused; replying is always the answer.
  12-Factor Agents (Factor 7) formalizes the same as "contact humans with
  tool calls": the loop breaks, state persists, the reply re-enters as an
  event. Copilot's mid-run steering ("applied after the current tool
  call") is the same ergonomic.
- **Noise.** Cognition's own admission: agents treat a channel "like their
  own command line". Their fixes — silence is a valid agent output, one
  consolidated update per state change — plus OpenHands' lesson (users
  revolted when a generated summary replaced the agent's actual final
  message) define the anti-noise rules.
- **Architecture split.** VS Code's notebooks postmortem (ADR-0036) maps
  directly: the latency-critical shell — storage, delivery, unread badge —
  belongs to the host; the *view* is where an app API gets exercised.

## Decision

The inbox splits into a **core data plane** and a **first app**.

1. **Core: one durable mailbox in SQLite** (orchestration data, ADR-0005).
   An item is `{id, kind, source, reason, title, body, payload,
   allowed_responses, blocking, state, created_at}` — kinds
   `fyi | question | approval | result`; verbs `accept | edit | respond |
   ignore` gated by per-item allow-flags; `reason` and `source`
   (agent/terminal/workspace) mandatory so grouping and "why am I seeing
   this" exist from day one. States: `unread → read → done`, plus
   `snoozed_until`. Responding to a question/approval marks it done —
   there is no separate step.
2. **Core: routes and badge.** `POST /api/inbox` accepts items (localhost
   API, same trust model as every PiCode route — a terminal or script can
   `curl` it); list/respond/triage routes serve the view; the Apps tab
   badge (ADR-0036) shows a numeric count of `blocking` items, a dot when
   only non-blocking news exists. Delivery is the existing poll cycle —
   the server has no SSE (ADR-0034 precedent) and the inbox is the durable
   source of truth, not a notification race.
3. **Sources of items, v1.** (a) PiCode itself, from RPC events it already
   observes: a run ending while its tab is not focused files a `result`;
   an error files an `fyi`. (b) A pi package, `packages/pi-inbox` (the
   `pi-roles` precedent, ADR-0028), giving agents `notify_human` and
   `ask_human` tools that post to the route. `ask_human` is **async
   mailbox, not a held connection**: it files a `question`, the turn ends,
   and the human's reply is delivered as a follow-up prompt to that agent
   session over the existing RPC channel — Devin's park-and-wake, Factor
   7's break-the-loop.
4. **View: the first app on ADR-0036.** The Inbox app renders two zones
   from host primitives: a **needs-me queue** (blocking questions and
   approvals, each item showing only its allowed verbs) and the
   archivable feed (results and FYIs) grouped by source with reason
   labels. Read/done/snooze are list actions. Whatever primitives this
   view lacks is the ADR-0036 API growing against a real feature — that
   is the point of the ordering.
5. **Anti-noise rules are contract, not style.** One item per state
   change, never per event; `result` items carry the agent's actual final
   message, never a generated wrapper; agents are not required to file
   anything — silence is a valid outcome.

## Refuse

| Asked for | Refused because |
| --- | --- |
| Slack / email / push channels in v1 | The in-app inbox must be the durable source of truth first; external channels are best-effort mirrors added later, with Linear's dedup rule (suppress the mirror when the item was read in-app). Flaky Slack delivery is Cursor's top inbox complaint. |
| Blocking `ask_human` that holds the agent's connection | 12-Factor's exact anti-pattern; a held turn dies with a restart. Park-and-wake resumes from durable state. |
| Auto-summarizing agent output into items | OpenHands' most-complained regression; deliver the real message. |
| Per-action approval gates on every tool call | OpenHands' confirm-everything noise; approvals enter the inbox only when something files one. |
| A separate "mark question done" step | Responding is the triage (Linear Triage semantics). |
| Read-on-view auto-marking | GitHub's mistake that spawned Octobox; read is explicit or on-open of the item, and done is always explicit for non-blocking items. |
| Priorities/urgency field in v1 | Only the HumanLayer lineage carries it and nobody ranks by it yet; the blocking flag covers the real split. Add when a consumer exists. |

## Consequences

- Two features couple: the Inbox cannot ship before the ADR-0036 pipeline
  renders an app. Accepted deliberately — the dogfooding order is the
  mechanism that makes the app API real.
- Poll-cycle delivery means seconds of badge latency; accepted for v1,
  same stance as ADR-0034's blocking clone. SSE remains a global v2.
- `POST /api/inbox` is writable by anything that can reach localhost —
  identical trust to every existing route, but now it can *address the
  human*; the mitigations are provenance on every item (`source`,
  `reason`) and rendering bodies as markdown in host primitives, never
  HTML.
- The reply path rides the existing per-agent RPC channel; a reply to a
  session that no longer exists degrades to a visible failure on the item,
  not a lost message.
- If wrong: the mailbox is one table and the app is one manifest; the
  worst case is an unused tab, not entangled machinery.

## Alternatives considered

- **Inbox as a hardcoded PiCode view (no app)** — ships marginally faster
  and forfeits the only forcing function the app API gets; see ADR-0036's
  dogfooding evidence.
- **Slack-first (Devin/Cursor/OpenHands model)** — their own etiquette
  postmortems document the noise; PiCode is the surface the operator
  already lives in, and has no Slack dependency to inherit.
- **LangGraph-style blocking interrupts with checkpoint/replay** — the
  strongest semantics for risky actions, but requires a durable
  checkpointer and idempotent resume the pi RPC does not offer PiCode;
  the async-mailbox verbs cover v1, and per-item `allowed_responses`
  keeps the schema forward-compatible with true interrupts.
- **Polling a structured-output document (Devin's API)** — right for
  machine consumers; the human-facing inbox needs items and verbs, not a
  mutating JSON blob.

## Sources

- Agent Inbox schema (verbs + allow-flags): <https://github.com/langchain-ai/agent-inbox>;
  LangGraph interrupts: <https://docs.langchain.com/oss/python/langchain/human-in-the-loop>
- 12-Factor Agents, Factor 7 (contact humans with tool calls; break the
  loop, resume by event): <https://github.com/humanlayer/12-factor-agents/blob/main/content/factor-07-contact-humans-with-tools.md>;
  Factor 11 (trigger from anywhere): <https://github.com/humanlayer/12-factor-agents/blob/main/content/factor-11-trigger-from-anywhere.md>
- Devin: session statuses, blocked+wake, message auto-resume:
  <https://docs.devin.ai/external-api/external-api>; Slack model and code
  channels: <https://docs.devin.ai/integrations/slack>; noise lessons:
  <https://devin.ai/blog/devins-slack-etiquette>; outcome-hinges-on-replies:
  <https://www.answer.ai/posts/2025-01-08-devin.html>
- GitHub notifications (reasons, unread/read/done/saved):
  <https://docs.github.com/en/subscriptions-and-notifications/concepts/about-notifications>;
  Octobox's archive-vs-mute critique: <https://octobox.io/documentation>
- Linear inbox (snooze, cross-channel dedup):
  <https://linear.app/docs/inbox>; Triage (every item demands a decision):
  <https://linear.app/docs/triage>
- Copilot coding agent (PR as inbox item, steering, no needs-input push):
  <https://github.blog/news-insights/product-news/github-copilot-meet-the-new-coding-agent/>,
  <https://docs.github.com/en/copilot/how-tos/copilot-on-github/use-copilot-agents/manage-and-track-agents>
- Codex (task list, mobile push on finish/needs-input):
  <https://developers.openai.com/codex/cloud>, <https://openai.com/codex/>
- OpenHands (confirmation mode noise, summary-instead-of-answer
  complaints): <https://github.com/OpenHands/OpenHands/issues/6161>,
  <https://github.com/OpenHands/OpenHands/issues/13035>
- gotoHuman (form-template requests, supersede-by-id, webhook replies):
  <https://docs.gotohuman.com/send-requests>
- Cursor background agents in Slack (thread context, flaky notifications):
  <https://cursor.com/docs/integrations/slack>,
  <https://forum.cursor.com/t/no-slack-notification-background-agent/152122>
- Ambient-visibility principles for agent oversight:
  <https://newsletter.victordibia.com/p/4-ux-design-principles-for-multi>
