# Study: Tachyon — fleet orchestration by agents (owner prior art)

- **Date:** 2026-09-14
- **Sources:** clone `~/tachyon` at `b51ed55e` (2026-08-26, last commit;
  project discontinued by the owner). GPL-3.0 — **patterns only, no code**
  (PiCode is PolyForm Noncommercial; the licenses do not cross).
- **Shape:** VS Code extension + persistent TypeScript engine
  (`packages/engine`, `packages/bridge`, `packages/shared`; ~2.5k TS
  files). Agents run as tmux sessions owned by the engine; an embedded
  **MCP Bridge** (official SDK, Streamable HTTP — same SDK family as our
  `/mcp/communication`) exposes the fleet to the agents themselves.
- **Scope:** the agent↔agent surface the owner highlighted — an
  orchestrator agent spawning, waiting on, reading and killing
  sub-agents — evaluated as prior art for PiCode's communication family
  (ADR-0104/0107/0110) and as a possible future direction.

## The orchestration loop (receipts)

The documented workflow is written into the tool itself
(`packages/bridge/src/tools/communication-waits.ts`):
**`spawn_agent → wait_for_agent(until=idle) → read_output → kill_agent`.**

| Tool | Contract (receipt) |
|---|---|
| `spawn_agent` (`fleet.ts:125`) | Temporary sub-agent with **parent lineage** from `$TACHYON_AGENT_NAME`; **delegation contract required** — task + context + constraints + deliverable/done_when, else rejected (`skip_contract_reason` ≥10 chars allowed, recorded, surfaced to the human); `claim_task` flips board tasks to active/assigned **before** launch ("one fact instead of two that can disagree"); generic processes refused → `spawn_terminal` |
| `wait_for_agent` (`communication-waits.ts`) | Blocks the caller's turn until `idle` (stopped producing output) / `needs-input` / `dead` / `change` (next transition watch); non-blocking alternative documented (child notifies on done) |
| `read_output` (`communication-io.ts`) | Another agent's pane: live rows, scrollback, in-memory postmortem, or **durable pipe-pane transcript that survives kill-session and extension reload**; secret redaction at read |
| `kill_agent` (`fleet.ts:470`) | **Governance:** only self, own lineage descendants, or owned Saved Agents — never sibling/parent/unrelated; for a Temporary with a checkout, kill also removes worktree+branch and end-of-life collects activity, transcripts and the private runtime home |
| `retask_agent` | Re-task a live Temporary without restart; re-probe liveness **after** delivery before echoing success (t-3e2e2d) |
| `get_project_handoff` (`coordination-handoff.ts`) | Curated, workspace-level handoff (state/active/next/decisions) shared by every agent, with a CAS `revision` — distinct from per-agent continuity |

## The screen reader (`shared/runtime/composerRegion.ts`)

Same craft as our `peerInputMatches`, with two ideas worth keeping:

- **One shared authority, two questions:** *"Attention asks 'does a human
  own this composer?'; submit asks 'did my line leave it?' — one region
  reader, two questions. Duplicating it would mean a runtime measured
  once but fixed twice."* Every rule was **measured against a live pane
  and carries the task id that measured it**.
- **Diff-confined validation:** `isChangeConfinedToComposer(previous,
  next, composer)` proves a change stayed inside the composer region by
  **differencing before/after**, instead of absolutely re-matching the
  expected frame. This absorbs dynamic footer content (paths, timers)
  that breaks an absolute match — the exact class that produced our
  OpenCode wrapped-path false draft (fixed 2026-09-14,
  `e7cca0f9`).
- Self-inflicted echo loop documented (t-6ffa13): their own submitted
  notice echoed back and read as a human draft, queuing every later
  delivery — handled by recognising the echo style. Our analogue is the
  post-paste recheck with `expected` set.

## Evaluation against PiCode's communication

| | Tachyon | PiCode (ADR-0104/0107/0110) |
|---|---|---|
| Agent↔agent | read each other's **output** + notices into the composer | durable **mailbox** (16 KiB bodies, retries, explicit ACKs) |
| Orchestration | agents **spawn/wait/kill** the fleet, lineage governance | refused: lifecycle is owner/terminal actions; setup never starts a process |
| Transcript exposure | yes — an agent reads another's screen/transcript | refused (ADR-0116: a grant is a contact, never a transcript) |
| Identity/credentials | per-agent env + bridge tokens; lineage scope | per-conversation bearer, hash-only in SQLite, consent per workspace |
| Delivery safety | measured composer profiles + occupancy + diff confinement | frame match + pointer fit + post-paste recheck; `uncertain` never retried |
| Shared lessons | measured rules carry their task id; shared region reader; honest refusals that teach the caller | same values, independent arrival |

**Bet difference:** Tachyon made **agents the fleet operators** (powerful,
accepting transcript exposure and lifecycle power in agent hands).
PiCode made **messages first-class and lifecycle owner-owned** (safer
boundary, no orchestration). Neither is wrong; they are different
answers to "who may act on the fleet".

## Verdict as a benchmark

**Yes — adopt as the owner-prior-art benchmark for a future "fleet
orchestration by agents" direction**, scoped honestly:

1. **What it benchmarks well:** the spawn/wait/read/kill loop, delegation
   contracts, lineage governance, board-integrated claiming, the shared
   region reader, and diff-confined validation. If PiCode ever lets an
   agent delegate to a sub-agent, these are the patterns to start from.
2. **Caveats:** owner bias (familiarity ≠ market validation — mitigated
   by pairing it with Orca, which arrived at the same mailbox+pointer
   shape independently); discontinued (stable, no upstream to track —
   good for a benchmark); GPL-3.0 (patterns only).
3. **Not a benchmark for** the messaging transport itself: Tachyon has no
   durable per-conversation mailbox, credentials or ACK protocol — that
   comparison target is Orca (2026-09-14 study).
