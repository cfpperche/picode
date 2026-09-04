# ADR-0062: Identify the live coding CLI in a project terminal

- **Status**: accepted (2026-09-04, owner approval)
- **Date**: 2026-09-04
- **Evidence**: [terminal TUI study](../benchmarks/2026-09-03-guest-tui-agent-state.md)

## Context

This ADR supersedes the conflicting tier-1 implementation assumptions in
ADR-0056: the activity sensor remains there, but wrapper lifecycle leases,
process identity, authoritative CLI presence, and the `terminal.runtime` feed
contract are defined here. Tier 2 guest-Agent promotion is unchanged and
still deferred under ADR-0056.

ADR-0056 gives a CLI inside a PiCode terminal an ephemeral activity state:
`working`, `needs-you`, or `idle`. The state is not a reliable answer to a
different question: **which CLI is still running in this pane?** A hook can
report `idle` after a process has exited, and a daemon restart loses the last
in-memory report. Conversely, `tmux pane_current_command` identifies only the
current command when tmux can provide it and `pane_pid` is the first process in
the pane; neither is a lifecycle protocol.

The sidebar needs both facts without promoting a guest TUI to a managed Agent:

1. **Presence** — a supported CLI (`claude`, `codex`, `grok`, or `pi`) owns the
   pane now.
2. **Activity** — that CLI is working, waiting for the user, or quiet.

A single `cli` field inferred from the last hook conflates the two and can show
a dead or different CLI as live.

## Decision

Keep terminal presence and activity as separate, ephemeral projections.

### Authoritative wrapper lease

The existing session-only `PATH` wrappers send
`POST /api/terminals/{id}/runtime` before launching the real binary:

```json
{"action":"start","cli":"pi","runId":"<terminal>-<pid>-<time>","pid":1234}
```

The wrapper exports `PICODE_TUI_RUN_ID`, so Claude, Codex, Grok, and Pi
lifecycle hooks include the same identity in activity reports. A failed launch
sends `end`; lifecycle-aware wrappers keep a small parent process long enough
to report normal termination, while maintenance/bypass paths still use direct
`exec` where argument dispatch requires it. The server watcher also
reconciles termination when a wrapper disappears.

The server stores one `TermRuntime` per terminal in memory. It records the
canonical CLI, source, run ID, PID, process start token when `/proc` is
available, and start time. A later `end` can clear only its own run ID. A new
run replaces an old one and clears its old activity state. No runtime or
activity field is written to SQLite.

The terminal view exposes the authoritative presence as `tui` and retains the
flat `cli` field as a compatibility projection:

```json
{
  "cli":"pi",
  "tui": {
    "cli":"pi",
    "source":"wrapper",
    "runId":"...",
    "startedAt":"..."
  },
  "state":"working",
  "stateAt":"..."
}
```

`terminal.runtime` and `terminal.state` are ephemeral change-feed events.
The browser rejects an event whose non-empty run ID does not match the
currently displayed runtime, so an old process cannot erase a new one.

### Reconciliation and fallback

`StartTermRuntimeWatch` validates each lease against the pane and process. A
missing terminal session, a dead PID, or a changed process start token clears
the lease and its state. For terminals created before this protocol, the
watcher may establish **presence only** when both an exact known CLI command
from `pane_current_command` and a valid pane PID are available. This fallback
uses no terminal name, title, or captured pixels and never invents activity.

The accepted decision table is:

| Condition | Projection |
|---|---|
| no tmux session | clear presence and activity |
| shell open, no exact CLI signal | `Terminal open` |
| exact CLI fallback | `<CLI> · Open` |
| valid wrapper lease, no activity event yet | `<CLI> · Open` |
| matching `working` event | `<CLI> · Working` |
| matching `needs-you` event | `Needs you` |
| matching `idle` event | `<CLI> · Ready` |
| PID gone or current runtime ends | clear CLI; retain terminal |
| stale run end/state event | ignore |
| ambiguous or unsupported command | do not guess |

### Sidebar language

Agent and terminal rows use one flat supervision layout: stable identity and
activity lead; path and branch are subdued clickable context; secondary
actions are behind a Radix menu. CLI marks are stable while activity is a
separate status treatment. `Needs you` is the only emphasized user action;
working uses motion plus text; a plain shell says `Terminal open`.

The open terminal tab and mobile terminal list use the same CLI identity and
vocabulary. The UI continues to consume the top-level state projection for
older servers and degrades to no CLI identity rather than guessing.

## Consequences

- The sidebar can answer presence and activity independently and survive
  browser reloads through the normal terminal list reconciliation.
- Run IDs make process replacement and out-of-order hook delivery safe without
  adding a database migration or a second agent entity.
- Existing sessions get a useful but deliberately weaker command/PID fallback;
  recreating a terminal is still the way to obtain wrapper instrumentation.
- `/proc` start tokens are strongest on Linux. Platforms without them retain
  OS process liveness checks and can lose protection against PID reuse.
- CLI wrappers remain an adapter surface and must degrade to `Terminal open` or
  no identity when a vendor changes its command or hook behavior.
- The decision does not authorize guest agents, composer control, inbox
  promotion, or screen scraping; those remain deferred under ADR-0056.

## Alternatives considered

- **Use the last `terminal.state.cli` report.** Smallest change, but it leaves
  stale CLI names after exit and cannot distinguish a new process. Rejected.
- **Scrape the terminal buffer.** Covers arbitrary TUIs but is brittle and
  violates the truthful-status rule. Rejected as a primary source.
- **Use `pane_current_command` as the only source.** Useful for old sessions,
  but explicitly “if available” and without a stable lifecycle identity.
  Retained only as presence fallback.
- **Create a durable guest Agent for each CLI.** Would change ADR-0006,
  capability gates, inbox, and control semantics. The owner approved terminal
  presence without promoting the guest process.
