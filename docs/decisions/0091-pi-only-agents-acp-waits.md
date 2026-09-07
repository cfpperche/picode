# ADR-0091: Pi-only agents hold; no agent protocol until the market converges

- **Status**: accepted (owner session, 2026-09-07)
- **Date**: 2026-09-07
- **Evidence**: [benchmarks study 2026-09-03 — guest TUI agent state](../benchmarks/2026-09-03-guest-tui-agent-state.md); ADR-0003, ADR-0056, ADR-0089

## Context

ADR-0056 shipped tier-1 terminal sensors for guest CLIs and left ACP
(level 4: headless JSON-RPC agents, UI entirely ours) as "the named
future track" — deferred, not refused. The 2026-09-07 owner session
re-asked the question: adopt ACP now and host guest CLIs as first-class
clients?

The market has not converged:

- ACP (Zed-born) covers many CLIs but mostly through adapters of uneven
  depth — Claude Code via Zed's SDK adapter, codex-acp layered on top of
  Codex's own app-server, pi via the third-party pi-acp adapter.
- Codex ships its own first-party embed protocol (app-server, JSON-RPC
  2.0); ACP is an adapter over it, so even ACP-covered CLIs disagree on
  the layer beneath.
- Grok CLI and Antigravity have no public protocol at all.
- pi already speaks its own RPC to PiCode for managed agents; ACP adds
  nothing for pi here — pi-acp exists for pi inside *other* editors.

Adopting today means shipping and rotting N adapters behind an
unconverged standard, replacing the owner's daily TUI workflows, and
re-opening ADR-0003 (Pi-only agents, no embedded agent SDKs).

## Decision

PiCode's first-class agent remains **pi** — managed agents over pi's own
RPC, manual pi in terminals. Guest CLIs (Claude Code, Codex, Grok CLI,
Hermes Agent, opencode, …) remain **TUIs in terminals** with ADR-0056's
tier-1 state sensors. PiCode ships no agent-protocol client — ACP, Codex
app-server, or vendor SDKs — and promotes no guest CLI to an agent.

This holds until the market converges on one standard: re-measure only
when a single protocol (ACP or a successor) carries first-party,
non-adapter support across most of the CLIs PiCode hosts. This ADR
converts 0056's open ACP deferral into a deliberate refusal with a named
trigger; it deliberately does not date the trigger. Tier-2 guest agents
stay deferred on their own terms (still requiring their own ADR, per
0056).

pi-acp remains an external-editor story; PiCode neither ships nor
depends on it.

## Consequences

**Easier.** No protocol client surface to maintain while the ground
moves; the roadmaps' "do not start ACP" lines (session-surface,
conversation-control, diff-editor) point at a standing decision instead
of a mood; guest coverage stays honest (chips or nothing); ADR-0003,
0056 and 0089 keep their letter.

**Harder / accepted.** PiCode cannot drive guest CLIs — no permission
answering, no headless sessions, no composer for them — until the
trigger fires. If the market converges quickly, the build cost (protocol
client, feed conversation events, chat surfaces) lands later, all at
once; that is the accepted price of not betting on the losing standard.

**Who breaks if we're wrong.** If vendors converge on ACP within this
product's lifetime and peer tools ship protocol clients, PiCode's guest
story stays terminal-bound longer than competitors'. The trigger exists
for exactly that world: the decision re-opens on evidence, not on a
schedule.

## Alternatives considered

- **Adopt ACP now (level 4).** Highest ceiling — streaming, permission
  answering, session control — but adapters-per-CLI on an unconverged
  market, and it replaces the TUI workflows the owner uses daily. Lost:
  premature.
- **Adopt per-CLI protocols (Codex app-server, vendor SDKs).** N
  integrations, each rotting on its own release cadence — strictly worse
  than waiting for one standard.
- **Tier-2 guest agents now (observe-only).** Supervision without
  control; needs its own ADR and an entity/runtime seam. Stays
  deferred, as 0056 left it.
