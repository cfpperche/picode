# ADR-0143: terminal agents are principals in the browser permissions

- **Status**: accepted
- **Date**: 2026-09-15
- **Boundary**: security model — who may be named as an agent and what the
  daemon resolves for them. The permission vocabulary (ADR-0134) does not
  change; the set of *principals* it applies to does, and the grant key
  gains a second namespace.

## Context

The work browser gates agents with a per-agent policy (ADR-0128, ADR-0134):
a tier that gates the verb catalog, and the origins an agent may reach.
`Agent permissions` lists the managed agents from the registry and writes
`browser.policy.<agent id>`.

Measured 2026-09-15, the chain for a CLI the human launched in a PiCode
terminal (a `pi`, `claude` or `codex` running in a terminal, not created by
PiCode's agent UI):

- `packages/pi-browser/extensions/browser.ts` sends
  `agent: (process.env.PICODE_AGENT_ID || "").trim()` — one variable.
- `PICODE_AGENT_ID` is injected only for **managed** agents
  (`internal/store/agents.go`). A terminal session carries
  `PICODE_TERM_ID` instead (`internal/tmux/server.go`).
- So a terminal agent posts an empty identity, `browser.Resolve` falls back
  to `Default()` — read on the tab on screen — and the section has no row
  for it: the human can neither see what it may do nor change it.

Sibling packages already resolved this exact question: `pi-inbox` and
`pi-checklist` derive `agentIdentity` as **managed agent → terminal →
unmanaged** (`packages/pi-inbox/src/logic.ts`). The browser is the only
surface that stopped at the first branch. That makes this an unfinished
identity, not a deliberate boundary — and leaving it unfinished would mean
the human cannot see a class of agents that already reaches the browser.

## Decision

A caller of the browser tool identifies itself with the house tuple:
`PICODE_AGENT_ID` wins (`kind agent`), else `PICODE_TERM_ID` (`kind
terminal`), else `kind ""` — unmanaged, with no identity at all.

Grants are keyed by that tuple: `browser.policy.<agent id>` for a managed
agent (every key written before this ADR keeps working, unprefixed and
unambiguous), `browser.policy.term:<term id>` for a terminal principal, and
nothing for an unmanaged caller — which stays read-only on the tab on
screen by construction, and the UI says so in one line rather than showing
a row nobody can edit.

The section lists managed agents **and the terminals that have a CLI
running** (their title and CLI as the row name), each with its own tier and
domains, default `read` — today's behaviour, now visible and editable. The
master switch (`Browser`) keeps gating every principal including these.
Unchanged: tiers, verb catalog, the shell's navigation gate, and the rule
that a broken grant resolves to `Default()`.

## Consequences

- A terminal agent can be granted `act` on named domains, which is the
  capability the human asked for; and can be refused explicitly, which
  matters more — an invisible read-only agent is safe by accident, not by
  decision.
- The terminal registry becomes a permission surface: rows follow the
  terminals that exist. A terminal that dies takes its row with it, so the
  UI never shows a principal that cannot act. The grant survives (the
  setting is keyed by term id), so reopening that terminal restores its
  policy instead of silently resetting it to read.
- Cost we accept: `browser.policy.*` now holds two shapes. A term id that
  collides with an agent id cannot widen anything — the prefix is part of
  the key — but readers of that setting must know the prefix rule; the
  resolver and the tests are the only readers.
- If we are wrong, the failure is narrow: grants keyed to terminals leak
  across a *new* terminal only if the term id is reused for a different
  session, and term ids are per-session and unique.
- An unmanaged `pi` (started outside PiCode) can never appear and can never
  be granted anything. That is the safe direction, and the UI states it
  instead of pretending the list is complete.

## Alternatives considered

- **One "Terminals" row for every CLI.** Fewer rows, less precision: act on
  a domain for one terminal would grant it to all of them at once. Lost
  because the thing being granted is a capability, and capabilities should
  attach to the smallest thing that acts.
- **The reference's shape — a site-pattern table with Browsing / Downloads
  / Uploads columns.** It answers "what may *any* agent do on this site";
  ours answers "what may *this* agent do". Adopting it wholesale would
  re-model the grants (ADR-0134) without addressing the missing principal.
  Its "Requires approval" cell is the same machinery as the pending Ask
  policy, which is scheduled separately.
- **Leave terminals read-only and say so.** Cheapest, and it would stop the
  confusion, but it would also make a real capability unreachable for the
  agents the human actually runs — the browser tool would be permanently
  second-class outside PiCode-managed agents.
