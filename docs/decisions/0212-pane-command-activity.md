# ADR-0212: A shell command in a CLI's process tree counts as work

- **Status**: accepted (2026-09-24, owner approval)
- **Date**: 2026-09-24
- **Boundary**: protocol — a new ephemeral feed event (`terminal.command`) and
  a `command` field on the terminal view; amends ADR-0062's rule that the
  process tree gives presence, never activity

## Context

Terminal activity comes only from each CLI's own hooks or plugins (ADR-0056).
A shell command typed with `!` fires no start hook in four of the nine CLIs.
Measured live on 2026-09-24:

- **Codex 0.156**: fires no hook at all.
- **Claude Code 2.1.281**: fires only `Stop`, when the command ends.
- **Grok**: its docs say bash mode reports no hook.
- **Hermes**: `!` is a plain subprocess.

The owner's `!PICODE_DEPLOY_FORCE=1 make deploy` ran for 69 s under a
**Ready** pill. No benchmark in `docs/benchmarks/` detects shell execution;
herdr and Orca only catch it by accident, through a spinner on screen.

The kernel does see the command. We measured the pane's process tree for
seven CLIs:

| CLI | Where the command runs | Where the helpers run (MCP, LSP, code-mode) |
|---|---|---|
| Claude, Codex, Grok, OpenCode, Pi, Omp | a new session (`setsid`) with no controlling terminal | the pane's session, behind a pipe or socket |
| Hermes | `sh -c` in the pane's session, reading the terminal | — |

Omp runs shell builtins such as `sleep` inside its own process, so they have
no child to see. External commands fork as for the others.

## Decision

The runtime watcher now also reads, every 3 s, the command running under
each live CLI lease (ADR-0062). The rule, per child of the CLI:

1. The walk follows the CLI's own chain (processes identified as a supported
   CLI). It never descends into a helper.
2. A direct child off that chain counts as a command when both hold:
   - it is at least 2 s old, which drops hook reporters and the status line;
   - it lives in a session other than the pane's, **or** it is `sh -c` reading
     the terminal (the Hermes case).
3. The oldest such child wins. Its label is the first process under it that
   is not a shell or launcher (`bash -c make deploy` reads `make`).

The observation lives beside the hook state and never replaces it:

- It is published as the ephemeral `terminal.command {termId, command: {name, since} | null}`.
- It is added to the terminal view as `command`.
- It is cleared when the runtime ends.
- The UI shows it as the existing `working` status, labelled **Running**,
  with a "Running make" tooltip, but only when the hooks are not already
  saying working, compacting or needs-you.
- It never starts or ends a turn (ADR-0194) and never touches `TermStates`.

## Consequences

- A `!` command and a background command left running show as **Running**
  in all six detached CLIs and in Hermes. The latter includes Claude's
  `run_in_background` and Codex background terminals, which is true: a
  command is running.
- Commands are seen only in terminals with a runtime lease. Muse and
  Antigravity have no lease, so they are not covered.
- Omp builtins and Windows or macOS daemons (no `/proc`) are not seen either.
- The cost is one `/proc` walk per tick while any lease exists; the presence
  watcher already did one lazily.
- **If we are wrong:** an unknown helper that detaches itself would read as a
  permanent **Running**. The fix is a row in the chain or launcher tables,
  not a new mechanism.

## Alternatives considered

- **Pane title spinner.** Claude (`✳` idle, `◐◑` working), Codex and Grok
  (braille prefix) do carry state in `#{pane_title}` today. That contradicts
  the 2026-09-14 measurement. We rejected it as the primary signal: each
  format is vendor-private, the approach reopens ADR-0056's "no pane text"
  rule, and it misses Hermes, OpenCode and Muse.
- **Session log tailing** (Codex rollout `task_started`, Claude
  `<bash-input>`). Covers two CLIs only, and ties activity to file formats
  that change between releases.
- **Treating every child as a command.** Every CLI keeps MCP servers as
  children, so every agent would always read Running.
