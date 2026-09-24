# ADR-0209: Native terminal compaction phase

- **Status**: accepted (owner, 2026-09-23: show the transient phase only when the CLI has a native signal)
- **Date**: 2026-09-23
- **Boundary**: protocol — `POST /api/terminals/{id}/state`, terminal list projections and `terminal.state` feed events gain `compacting` as a transient activity value
- **Amends**: ADR-0056, ADR-0062

## Context

A Codex terminal can visibly compact its context while its PiCode agent row
says Ready. The existing terminal sensor sends only working, needs-you and idle.
The Pi tmux spinner reader cannot identify other CLIs' compact operations.
Codex, Claude Code and Grok expose PreCompact/PostCompact hooks; Pi and Omp
expose session_before_compact/session_compact extension events; OpenCode
exposes an experimental pre-compaction hook and a session.compacted event.
Hermes, Muse and Antigravity have no confirmed pair of native signals.

## Decision

An integrated CLI reports `compacting` only from its native pre-compaction
event. Its native completion or normal lifecycle report clears that phase.
The value is ephemeral and correlated to the same terminal runtime and native
session as every other activity report. The UI labels it Compacting and counts
it as active work. A CLI without the native pair keeps its ordinary Working
signal. The server rejects a prompt during compaction and expires a stale
compacting report on the same 30-minute bound as working.

### Decision table

| Condition | Action |
|---|---|
| Native pre-compaction on selected session | report `compacting` |
| Native completion, manual/idle | report `idle` |
| Native completion, ongoing turn | report `working` |
| Native failure/abort with a failure event | report current idle/working state |
| Later ordinary lifecycle event | replace `compacting` with that state |
| No native compaction signal | retain ordinary Working/Ready state |
| Dead runtime or stale report | ignore or clear through the existing runtime fence |

## Consequences

The sidebar, tabs, mobile and Canvas share the same label and activity class.
The installed Pi 0.87.1 exposes `session_compact_failed`; Omp 18.2.11 exposes
the before/success pair but no failure event. A failed Omp compaction can
therefore remain Compacting until another lifecycle report or the 30-minute
expiry. OpenCode's start hook is experimental and needs a vendor-version probe
when upgraded. No transcript or pane-text guess is used: if a vendor changes
these hooks, its row falls back to its remaining general activity signal.

## Alternatives considered

- Scrape the pane's "Compacting" copy: language and render dependent, and
  conversation text can produce false positives.
- Infer compaction from token usage or silence: neither identifies the
  operation and both would claim a phase the CLI did not report.
