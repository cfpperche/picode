# ADR-0056: Guest CLI state — tier 1 on terminals, tier 2 as guest agents

- **Status**: accepted (2026-09-03, owner session — decision split into
  two tiers during review; tier 1 built now, tier 2 deferred; amended
  2026-09-03 with Codex lifecycle hooks and PTY interrupt fallback)
- **Date**: 2026-09-03 (proposed); 2026-09-03 (accepted, amended in review)
- **Evidence**: [benchmarks study 2026-09-03 — guest TUI agent state](../benchmarks/2026-09-03-guest-tui-agent-state.md)

## Context

The owner runs guest coding CLIs (Claude Code, Codex, Grok CLI,
Antigravity, opencode) inside PiCode terminals daily, and wants the ADE
to host those TUIs — without building agents or package management for
each one. What the terminal cannot give is **fleet state**: the
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
- Codex ships Claude-compatible lifecycle hooks (`UserPromptSubmit`,
  `PermissionRequest`, `Stop`, `Interrupt`, `SessionEnd`), legacy
  `notify`, and an official **app-server** embed protocol (JSON-RPC 2.0,
  used by its VS Code extension).
- opencode ships a headless HTTP server (`opencode serve`) whose TUI
  is itself just a client, plus `opencode acp`.
- Grok CLI has headless NDJSON events; Gemini/Kimi/Qwen/Droid/Cursor
  and **pi itself (pi-acp adapter)** are on the ACP registry.
- Antigravity has no public integration surface (2026-09-03).

Constraints in play: the truthful-status bar (benchmarks.md — "status
is always truthful… never fake progress"), ADR-0003 (user-installed
pi, no vendored agent SDKs), and the feed's existing ephemeral state
event (`agent.state`), which the sidebar and mobile already consume.

During review the owner cut the scope: **agents stay out of v1; the
state lives on terminals.** This ADR records both tiers so the seam is
explicit before any code lands.

## Decision

**Tier 1 — now (terminals).** A coding CLI inside a PiCode terminal
reports its own state through a small HTTP hook:
`POST /api/terminals/{id}/state` with `{"state": "working" |
"needs-you" | "idle", "cli": "…"}`. PiCode correlates the report to the
terminal via `PICODE_TERM_ID` (and `PICODE_TERM_URL`), injected into
the tmux session environment **at creation** (`new-session -e`).

**Intercept, not user-file wiring (owner 2026-09-03).** PiCode never
writes `~/.claude`, `~/.codex` or `~/.grok`. Turning a CLI on drops a
wrapper in `<data>/bin` and prepends that directory to **PATH of that
tmux session only**. The user types `claude` / `codex` / `grok` as
usual; the wrapper `exec`s the real binary with launch-time injection:
Claude `--settings` (JSON in the data dir), Codex invocation-only
`-c hooks.…` plus per-command trust hashes (legacy `notify` remains a
fallback), and a Grok `GROK_HOME` overlay (hooks in the data dir,
`auth.json` symlinked). Codex's blanket hook-trust bypass is never used.
Outside PiCode terminals the wrappers are not on PATH. A leftover
from the retired file-wiring is stripped from `~/.claude/settings.json`
on Claude enable/disable (marker entries only). Reports republish as ephemeral `terminal.state` events on
the ADR-0048 feed; views carry the same fields for reconciliation.
Claude does not fire `Stop` for every user abort, so the terminal bridge
also treats the exact Ctrl+C byte and a *bare* Escape frame as an
optimistic idle transition before forwarding the input. Escape-prefixed
arrow/Alt/function sequences are excluded; the next CLI event reconciles
state. A sweep expires `working` after 30 minutes of silence (hooks fire
between tools, not inside one) — a silenced sensor degrades to "no
signal", never a stale spinner. `needs-you` and `idle` do not decay.
The UI shows two chips only — working (spinner) and needs-you
(accent) — on the sidebar terminal row, mobile terminal row, and open
terminal tab; no chip means no signal. Nothing here touches SQLite,
the Agent entity, or Runtime, and screen-scraping stays refused as a
primary source (bell detection remains a possible later fallback).

**Tier 2 — deferred (guest agents).** Promote a wired terminal to a
guest *agent* (new runtime kind, observe-only: state, needs-you,
push, inbox "Open terminal"; no composer, no drive — send-keys stays
rejected per ADR-0002). Tier 2 re-uses tier 1's sensors, vocabulary
and UI chips, re-anchoring them from terminal ids to agent ids; it
re-measures ADR-0003's "Pi-only agents" clause with the system already
running. It needs its own ADR before any code.

## Consequences

**Easier.** One state vocabulary across pi and guests from day one;
tier 1 is strictly additive (no schema, no runtime changes); sensors
are config the user's own CLIs already load, so ADR-0003's letter
holds; the chips appear where the owner already looks (the terminal
list they use daily).

**Harder / accepted.** Per-tool adapters rot when a CLI changes its
hook schemas — each sensor ships with the events it maps and degrades
to "no signal". Tier 1 keeps guests out of the unified fleet strip,
Inbox cards and Web Push until tier 2; wiring is, for now, a copy-paste
guide (`www/guide/terminal-status.md`), not a one-click flow. CLIs
without a sensor surface (Antigravity today) simply have no state.

**Who breaks if we're wrong.** If a tool's events prove too coarse,
that CLI shows no chip — an honest downgrade, never a false "working".
If tier 2 never happens, tier 1 still stands alone as terminal status.

## Alternatives considered

- **Guest agents immediately (tier 2 first).** One supervision surface
  from day one, but it touches the Agent entity and ADR-0006's
  "one live pi process" definition, needs a capability-gate matrix
  (composer/inbox/automations/spend), and delays the owner's actual
  request behind schema work. Lost to the owner's simplicity call.
- **herdr-style screen detection as the primary source.** Covers
  arbitrary CLIs, but it is inference from pixels — brittle across TUI
  releases and in direct conflict with the truthful-status bar; kept
  only as a possible fallback after the bell.
- **Do nothing (terminal only, no state).** Honest, but leaves the
  fleet half-invisible and does not meet the owner's supervision
  direction.
- **Adopt ACP now (level 4).** Highest ceiling — streaming, permission
  answering, session control — and the registry already lists pi-acp.
  Still the natural tier-3 conversation, but it replaces the TUI
  workflows the owner uses daily; deferred with the rest of tier-2
  thinking.
