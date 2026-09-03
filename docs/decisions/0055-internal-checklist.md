# ADR-0055: Internal checklist — the agent plans before it changes, and the sidebar shows the step

- **Status**: accepted
- **Date**: 2026-09-03

## Context

An agent's plan is visible only inside its chat or TUI. The owner
supervises several agents from the sidebar and wants to know what each
one is doing without opening it — and wants agents to *write* a plan
before they start changing things, because an agent that plans first
does better work and one that does not is invisible until it is done.

The owner's other product, Tachyon, solved this for Claude Code, Codex
and Grok (`checklistGateHook.ts`, `internalChecklistReprompt.ts`,
`agentChecklistLine.ts`; researched 2026-09-03). Its shape, in short:

- Two mechanisms at two moments. A `PreToolUse` gate refuses the first
  *mutating* tool call of a session until a plan exists — that is the
  obligation. A turn-end monitor sends one reminder when a required plan
  is missing — that is the memory. Neither blocks delivery.
- No ledger of its own: each runtime's native plan tool (TaskCreate,
  update_plan, todo_write) is the channel, and the engine reads the file
  the runtime already writes.
- The obligation depends on the task's kind, a free string on the task
  card, matched against `settings.checklist.requireIn` at spawn.
- Six locks: fail open on every unknown, no hook when nothing is
  required, once per session, the refusal names the way out, delivery is
  never blocked, and each runtime earns its hook by a live measurement.
- The sidebar gets one line per agent: `{kind:"step", text, position,
  total}` or `{kind:"absent"}`; a finished plan stays visible as n/n; no
  channel means no line, because accusing an agent that could not write
  a plan asserts a state nobody measured.
- Pi has no support there: no measured channel, no gate.

PiCode runs Pi agents only for now (Claude Code, Codex and Grok agents
are deferred). Pi 0.84's extension API has every hook the mechanism
needs, first-class: `before_agent_start` (see the prompt, extend the
system prompt), `tool_call` (block a tool with a reason, no terminate),
`registerTool` (a plan tool — pi ships none), tool-result `details`
reconstructed on `session_start` (branch-safe state), `sendMessage`
with `deliverAs: "followUp"` (the reminder). PiCode has the precedents:
pi-inbox talks to the daemon over HTTP with `PICODE_AGENT_ID`, pi-roles
appends to the system prompt in `before_agent_start` and publishes
per-agent state the UI reads, the change feed (ADR-0048) moves per-agent
events to both shells live.

PiCode has no task card. What it has per agent is the tools mode
(ADR-0009, read-only or full), the packages list and the spawn env.

Decisions taken with the owner on 2026-09-03: the obligation is a level
per agent rather than a prompt classifier; the package is opt-in like
every package (ADR-0010) — installed for the machine, the folder or the
agent it works, otherwise nothing happens; the checklist shows on the
sidebar *and* as a card in the chat; and when the package is installed
and configured the system must hold the agent to the contract rather
than remind once and give up.

## Decision

An opt-in pi package, `packages/pi-checklist`, plus a small data plane
in the daemon and one line per agent in both shells.

**The tool.** `checklist { items: [{text, status}] }` — the whole list
each call, statuses `pending | in-progress | completed` (closed, like
Tachyon's). The result's `details.items` is the state; `session_start`
rebuilds it from the branch, so a resume or a fork keeps the plan. The
TUI renders the call as `☐ ◐ ☑` lines.

**The level**, per agent, stored on the agent row (`agents.checklist`,
migration 022) and passed to every spawn as `PICODE_CHECKLIST`:

| Level | Meaning |
|---|---|
| `changes` (default, also for a raw pi with no env) | a checklist before the first change of a task; a read-only answer needs none |
| `always` | every task, read-only answers too |
| `never` | the tool stays, nothing is required |

A read-only agent (ADR-0009) is `never` whatever the row says: it cannot
change anything. The level is edited on the agent's settings page next
to the tools mode. The task's *kind* is not classified: the first
mutating tool call reveals it, which is exactly what Tachyon's gate
keys on and what the owner asked for.

**The contract**, appended to the system prompt in `before_agent_start`
when the level requires it: write 2–8 concrete steps before the first
change, the first one in-progress; send the whole list on every call;
mark steps as they finish; never delete history; changes are refused
until the checklist exists.

**The gate**, in `tool_call`: with a required level, `bash`,
`powershell`, `edit`, `write` and `multiedit` are refused until the
`checklist` tool has run *for this task* — `before_agent_start` resets
the flag, so each user prompt that leads to a change needs the plan
written or updated. A closed allowlist of mutators, so an unknown tool
can only widen the door; the refusal is one line that names the tool
to call; never `terminate`, so the model writes the plan and retries.
The gate is absolute: there is no give-up on the obligation, which is
what "hold the agent to the contract" means. A thrown handler allows
(fail open).

**The reminder**, in `agent_end` under `always`: a turn that ended
without a checklist gets a follow-up message that triggers a turn, up
to three times per task — the one safety valve, because a model that
will not comply must not be re-prompted forever on the owner's bill.

**The data plane.** On every checklist call, on `session_start` when a
plan was rebuilt, on a refused change and on a turn that ended without
a required plan, the extension POSTs `/api/agents/{id}/checklist`
`{sessionId?, items, absent?, blocked?}` — the pi-inbox idiom:
`server.json` or `PICODE_URL`, the install token, loopback TLS accepted,
never awaited by the turn. The daemon keeps one row per agent
(`agent_checklists`), validates the items, and records `agent.checklist`
on the feed; `GET /api/checklists` is the boot fetch. A refused change
with no list, or an `always` turn without one, is stored as `absent`.

**The line.** Both shells project one operator line from the map, the
Tachyon contract to the letter: `(2/4) the current step` (the
in-progress step, else the first pending, else n/n), or "No checklist"
when absent, or nothing when nothing is known. On the desktop it is a
second row under the agent's name; on the phone it replaces the model ·
folder sub line while a plan exists. In the chat the `checklist` tool
call is a card, open by default, every step with its glyph.

## Consequences

- Nothing changes for an agent without the package: no env is read, no
  prompt is appended, no tool exists. A raw `pi` with the package but no
  PiCode gets the tool, the gate and the TUI card; only the publishing
  is silent. That is the opt-in the owner chose.
- A change is impossible without a plan wherever the package is on. An
  agent asked for a one-word typo fix writes a one-line checklist first;
  that costs one short tool call and is the price the owner accepts.
- The per-task reset means a long session writes many small updates
  instead of one plan; the sidebar line follows the *latest* task, which
  is what a supervisor wants.
- The daemon stores the latest list only. History is the session file;
  the feed carries every change, so a replay reconstructs what a card
  showed.
- If Pi renames a mutating tool, the gate does not see it and a change
  slips through unplanned — fail open by design; the allowlist is one
  constant to extend.
- Who breaks if we are wrong: a model that cannot follow the contract
  under `changes` is stuck at the gate and says so in its reply (the
  refusal is visible in the chat as a refused tool call); the owner
  drops the agent to `never`. That is the loud failure, not a silent
  one.

## Measured (2026-09-03, pi 0.84.4, zai/glm-5.3-flash)

- **Managed (RPC) agent in a scratch PiCode**, prompt "edit notes.txt":
  `checklist` (1 in-progress, 1 pending) → `read` → `edit` → `checklist`
  (2/2 completed); the daemon had the list 8 s after the prompt, the
  file changed at 16 s, the sidebar and the phone row read
  `(2/2) Edit notes.txt: beta → gamma`, the chat showed two cards.
- **The gate, same agent**, prompt "do NOT call the checklist tool first":
  `edit` refused with the one-line reason → `checklist` (1 in-progress)
  → `edit` ok → `checklist` (1/1). The refusal is visible in the chat
  as a refused tool call; the model's own words: "the harness enforces
  it."
- **Interactive TUI in tmux**, same skip-the-plan prompt: the refusal
  rendered, the model wrote a one-step list, the `◐`/`☑` lines rendered,
  the edit went through.
- **`pi -p` (print mode) hangs on any blocked `tool_call`** — reproduced
  with a five-line extension that blocks `edit` once and nothing else,
  so it is pi's, not this package's. PiCode never runs `-p` for an
  agent; a raw `pi -p` with the package and a change that needs a plan
  the model skips will sit until killed. Noted in the package README.

## Alternatives considered

- **Classify the prompt in `before_agent_start`** (a heuristic or a
  cheap model call decides the task's kind). Costs a judgment per turn
  and misjudges; the first mutating call is the fact itself. Lost.
- **Read the session file instead of a POST** (the pi-roles state-file
  idiom). Needs a watcher per agent and gives no event; the feed
  already carries per-agent events. Lost.
- **Reuse a pi TODO tool.** There is none in 0.84 (`dist/core/tools`
  holds bash, edit, read, write, grep, find, ls). A Tachyon-style
  "no ledger of ours" was not available. Lost.
- **One reminder then give up** (Tachyon's reminder half). The owner
  asked for the contract to be held; the gate is absolute and the
  reminder capped at three. Lost.
- **Install by default on every PiCode spawn.** Against ADR-0010 and the
  owner's answer. Lost.
