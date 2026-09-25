# ADR-0217: Unattended automation runs on guest CLIs through their own TUI

- **Status**: proposed
- **Date**: 2026-09-25
- **Boundary**: process (PiCode starts a guest CLI's TUI with nobody watching and drives one turn through it); security model (a third-party agent acts without a person at the terminal, and PiCode never grants it approvals); persistence (an automation records which CLI its runs use: `automations.cli`)
- **Amends**: ADR-0107 (automations' `start` action ran only Pi's managed mode); builds on ADR-0089 (the prompt door, verified delivery since its 2026-09-20 amendment) and ADR-0160 (a CLI agent's terminal is its process)
- **Does not change**: ADR-0091 — no protocol client and no managed mode per CLI; the run goes through the CLI's own TUI

## Context

An automation's `start` action creates its own agent, runs one task and
stops it. Until now it existed only for Pi, because it rides Pi's managed
mode (`pi --mode rpc`): a turn is sent over RPC, the settle and the cost
arrive as events. The other eight CLIs have no such channel here
(ADR-0091), so a scheduled or webhook-fired task could not run on Claude
Code, Codex, Grok or Hermes — and the Outcomes options that need a
CLI-neutral unattended runner (the agent-exits study's C and D) were blocked.

What exists already, measured and shipped:

- The prompt door (ADR-0089, Fatia F) delivers text into a CLI's TUI and,
  for the four CLIs with a measured composer reader — Claude Code, Codex,
  Grok, Hermes — verifies it: the composer reads empty before, and empty
  again after Enter. Unattended senders are refused on any screen it cannot
  recognize (a login, a menu), instead of pasting blind.
- Those four report their activity through PiCode's hooks (ADR-0056):
  `working`, `needs-you`, `idle`.
- The CLIs keep session files PiCode already prices for Outcomes
  (`climetrics.MeterSessionFile`).

## Decision

An automation with `action: start` names its CLI (`automations.cli`,
default `pi`). For `pi` nothing changes. For `claude-code`, `codex`, `grok`
and `hermes`, a run:

1. uses the automation's own agent of that CLI (created on the first run in
   the automation's workspace, reused after), stops its terminal if one is
   open and starts it fresh — a new conversation, as Pi's runs get;
2. waits for the TUI's composer (the door, retried until it is recognized or
   90 seconds pass), then delivers the prompt through the door and requires a
   **verified** receipt;
3. follows the hook state: `working` → `idle` is the end of the turn;
   `needs-you` files one Inbox item saying the run is waiting for the person
   at that terminal and keeps waiting; the two-hour timeout and the cost cap
   (the session file priced every 30 seconds) stop it;
4. prices the session, finishes the run, and closes the terminal (the
   session stays with the agent to reopen).

PiCode never passes a flag that skips the CLI's own approvals
(`--dangerously-skip-permissions`, `--yolo`, `--auto-approve` or the
like): a run that needs an approval waits for a person. The other five CLIs
are refused for `start` until their composer is measured — an unattended,
unverified paste can land in a menu.

## Consequences

- Scheduled and webhook tasks run on four more CLIs, with the same run
  record, Inbox notes and caps as Pi's.
- A run can sit in `needs-you` until the timeout. That is the cost of never
  approving on the person's behalf; the Inbox item says where to answer.
- The end of a turn is the CLI's hook, not an RPC event: a CLI whose hooks
  stop reporting reads as a run that never settles and ends at the timeout.
- The final message is not read back in this slice: the Inbox result says
  the run finished and points at the agent's session.
- If we are wrong about the composer readers (a CLI update changes its TUI),
  the door refuses as `unrecognized`, the run is skipped with that reason —
  nothing is pasted blind.

## Alternatives considered

- **A protocol client (ACP) per CLI.** Refused by ADR-0091 until its
  re-measure is decided; this ADR does not depend on it.
- **Headless print modes (`claude -p`, `codex exec`).** Each has its own
  flags, permission model and output; they bypass the TUI the person later
  reopens, and several need an approval flag to act at all.
- **Passing the CLI's auto-approve flag for automations.** Refused: an
  unattended agent with every approval granted is exactly what the security
  model keeps out.
- **All nine CLIs at once.** Five have no measured composer reader; an
  unverified unattended paste is the failure the door's Fatia F removed.
