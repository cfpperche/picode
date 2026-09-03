# ADR-0056: Guest TUI state via per-tool sensors on the change feed

- **Status**: proposed
- **Date**: 2026-09-03
- **Evidence**: [benchmarks study 2026-09-03 — guest TUI agent state](../benchmarks/2026-09-03-guest-tui-agent-state.md)

## Context

The owner runs guest coding CLIs (Claude Code, Codex, Grok CLI,
Antigravity, opencode) inside PiCode terminals daily, and wants the
ADE to host those TUIs — without building agents or package management
for each one. What the terminal cannot give is **fleet state**: the
sidebar's working (spinner) and needs-you exist only for pi agents,
which report over RPC into the ADR-0048 feed (`agent.state`). A guest
TUI in a PTY exposes pixels, and reading pixels is the one thing no
credible orchestrator does as its primary source:

- paseo runs agent TUIs in PTYs **plus provider hooks** (2026-08-24 study).
- t3code, Zed, Vibe Kanban, Casper observe (or drive) agents through
  protocols — SDK/ACP, executor APIs, ACP web clients.
- Claude Code ships lifecycle hooks that may be **HTTP endpoints**
  (`Notification` types include `permission_prompt`,
  `agent_needs_input`, `agent_completed`; plus `Stop`,
  `TaskCompleted`), a JSON statusline, and a terminal-bell fallback.
- Codex ships `notify` ("receives a JSON payload from Codex") and an
  official **app-server** embed protocol (JSON-RPC 2.0, used by its
  VS Code extension).
- opencode ships a headless HTTP server (`opencode serve`) whose TUI
  is itself just a client, plus `opencode acp`.
- Grok CLI has headless NDJSON events; Gemini/Kimi/Qwen/Droid/Cursor
  and **pi itself (pi-acp adapter)** are on the ACP registry.
- Antigravity has no public integration surface (2026-09-03).

Constraints in play: the truthful-status bar (benchmarks.md — "status
is always truthful… never fake progress"), ADR-0003 (user-installed
pi, no vendored agent SDKs), and the feed's existing ephemeral state
event (`agent.state`), which the sidebar and mobile already consume.

## Decision

Guest CLIs keep running as themselves in PiCode terminals. Their
lifecycle state reaches the sidebar through **per-tool sensors** —
config-side hooks/notifiers (e.g. Claude Code hooks, Codex `notify`,
opencode server events) that POST events to a local daemon endpoint —
and the daemon republishes them on the ADR-0048 feed using pi's state
vocabulary (`working` / `needs you` / `idle`), with honest `unknown`
when no sensor reports for a CLI. Screen-scraping is refused as a
primary source; it may exist only as a last-resort fallback after the
terminal bell. Sensor setup is opt-in per CLI: PiCode writes the
config snippet with the user's consent, embeds no agent SDK, and does
not launch, package or manage the guest binaries. Deeper control —
answering a guest's permission prompt or driving turns from the
sidebar (ACP / Codex app-server territory) — is explicitly deferred to
a future ADR.

## Consequences

**Easier.** One sidebar state model for pi and guests, since both ride
the existing `agent.state` event; no second runtime to own; ADR-0003's
letter holds (sensors are config the user's own CLIs already load);
guest TUIs keep working exactly as today, sensors only add signal.

**Harder / accepted.** Per-tool adapters rot when a CLI changes its
hook schemas — each sensor ships with a table of the events it maps
and degrades to `unknown` instead of guessing. Observation is one-way:
a guest's permission prompt must still be answered in its TUI (that is
precisely what the deferred control tier would add). CLIs without a
sensor surface (Antigravity today) stay `unknown`. The bell fallback
signals attention without semantics.

**Who breaks if we're wrong.** If a tool's events prove too coarse,
that CLI shows `unknown`/`idle` — an honest downgrade, never a false
"working". If sensors silently break, the failure mode is absence of
signal, which the vocabulary already expresses.

## Alternatives considered

- **Adopt ACP now (level 4).** Highest ceiling: streaming, permission
  answering, session control — and the registry already lists pi-acp,
  so even pi could be driven through it. Lost for now: it replaces the
  TUI workflows the owner uses daily, requires re-measuring ADR-0003's
  "Pi-only agents" clause and the benchmarks README's "we are not a
  multi-runtime harness" rule, and is a multi-week track; sensors
  deliver the stated need (spinner + needs you) at a fraction of the
  cost. The control tier remains the named follow-up.
- **herdr-style screen detection as the primary source.** Covers
  arbitrary CLIs, but it is inference from pixels — brittle across TUI
  releases and in direct conflict with the truthful-status bar; kept
  only as a fallback after the bell.
- **Do nothing (terminal only).** Honest, but leaves the fleet
  half-invisible and does not meet the owner's supervision direction.
