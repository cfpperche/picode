# ADR-0162: Pi uses the shared interactive runtime

- **Status**: accepted (owner approved implementation plan, 2026-09-20)
- **Date**: 2026-09-20
- **Boundary**: persistence and process — Pi agents lazily bind an existing terminal-launch runtime through `agents.terminal_id`; agent routes resolve that runtime instead of spawning a separate interactive process.
- **Amends**: ADR-0160's Pi interactive implementation and ADR-0089's separate pane addressing. ADR-0006's one-writer guarantee, ADR-0040's session ownership and ADR-0107's opt-in communication remain.

## Context

Pi's Agent CLI adapter already reports activity and native session identity and
injects the Ask receiver. Pi agent terminals bypassed that launcher and scraped
the screen for a spinner instead. The same CLI consequently had two activity
contracts after the other CLIs became agents in ADR-0160.

## Decision

Opening a new Pi interactive process atomically creates its terminal, launch
configuration and agent binding when needed. It uses the common launcher,
activity extension, runtime lease and session pin. Agent settings supply Pi's
model, packages, identity and private session directory. Launch overrides may
configure the executable, environment, PATH and additional arguments but cannot
override the agent's mode or session ownership. Managed RPC stays Pi-only.

Agent and terminal routes serialize lifecycle operations by agent identity.
Replacement prepares before stopping, verifies the old process tree has exited,
and refuses another writer while a shutdown receipt remains pending. Removing
a bound terminal stops its owning agent before the store cascade.

An already-running legacy Pi session keeps its original address and screen
observer until an explicit stop/restart. No upgrade restarts it. Existing agent
terminal URLs resolve to the current address. No JSONL is moved or identity
recreated. A failed preparation leaves the live legacy process addressable.

Activity belongs to the active mode: interactive uses terminal events, RPC uses
its runtime. Unknown activity remains unknown; a stopped or previous terminal
generation never overrides the RPC state.

## Decision table

| Conditions | Action |
|---|---|
| Pi stopped, no binding | Atomically bind one launch terminal and start it |
| Pi interactive, ordinary open | Reuse process and session |
| Existing legacy TUI | Preserve until explicit stop/restart |
| Invalid replacement configuration | Refuse before stopping the old process |
| Stop fails or a captured process remains alive | Refuse replacement; preserve shutdown receipt |
| Switch TUI/RPC | Stop the old writer before starting the requested mode |
| Terminal start while RPC or legacy TUI is live | Refuse; agent mode switch remains explicit |
| Working / needs-you / idle report | Shared state, session attribution and agent attention |
| Missing/disabled activity or stale generation | No invented idle/working state |
| Reconnect | Reconcile snapshots; never relaunch |
| Remove bound terminal | Stop owning process, then cascade records |
| Unbound terminal or another CLI | Existing terminal integration |

## Consequences

Pi shares activity, launch and lifecycle infrastructure with other interactive
agents while keeping its RPC mode and ecosystem configuration. A compatibility
resolver remains for old live panes; it must never spawn new legacy panes.
Incorrect process ownership would allow concurrent JSONL writers, so uncertain
shutdown fails closed. Linux uses process start tokens; other Unix hosts use
`ps` start times to fence PID reuse when stopping the captured process tree.

## Alternatives considered

- Inject only activity into the old launcher: fixes a badge but preserves two
  interactive implementations and omits the accepted migration scope.
- Restart every live Pi during migration: interrupts active work unnecessarily.
- Replace the Pi RPC runtime: unrelated to reuse of the interactive stack.
