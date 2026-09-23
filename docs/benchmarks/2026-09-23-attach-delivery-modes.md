# Attach composer delivery modes across the nine CLIs

- **Date**: 2026-09-23
- **Question**: can the attach composer (ADR-0089 prompt door) deliver a
  message **while the CLI is working** — the Prompt / Steer / Follow-up
  choice the managed Pi composer already offers — for every agent CLI?
- **Status**: desk study. Every row below was read from installed source,
  binary strings, vendor docs or `--help`; **none was observed in a live
  pane yet** (see [Live measurement](#live-measurement)).

## Today

`doorDeliverMode` (`internal/server/term_prompt.go`) refuses with 409
`working` / `needs-you` whenever the terminal state says the CLI is busy.
The door has one mode: Prompt, into an idle composer, verified by the
input row reading empty before and after Enter.

## Vocabulary

| Mode | Meaning |
|---|---|
| **Prompt** | Idle composer, starts a turn (today's door) |
| **Steer** | Injected into the running turn at the next tool/step boundary |
| **Follow-up** | Held until the turn ends, then sent as the next turn |
| **Interrupt + send** | Abort the turn, then send now |

## Matrix (desk-read, unverified live)

| CLI | Steer | Follow-up | Interrupt + send | Config that changes plain Enter | Queue render (above the input) |
|---|---|---|---|---|---|
| Pi 0.87 | Enter | Alt+Enter (Ctrl+Q on Windows/WSL) | Esc restores queue to editor — no send | `~/.pi/agent/keybindings.json` | `Steering: …` / `Follow-up: …`, one truncated line each |
| Omp 18.2 | Enter | Ctrl+Q, Ctrl+Enter, **`/queue <text>`**, `-> text` | Enter on empty editor with a queue | `keybindings.yml`, `interruptMode` | `Steering · N` / `After yield · N` + numbered lines |
| Claude Code 2.1.280 | Enter (queued, absorbed after running tool calls) | Ctrl+X Enter (`chat:queueSubmit`; skips injection? unverified) | Ctrl+X Ctrl+S (`chat:sendNow`) | `~/.claude/keybindings.json` | gray queued list above the box |
| Codex 0.156 | Enter (`steer` feature is always on) | Tab (`composer.queue`) | Esc sends pending steers | `[tui.keymap]` | `• Messages to be submitted after next tool call` / `• Queued follow-up inputs` + `↳` rows |
| Grok 1.0.41 | Enter only if `ui.follow_up_behavior="steer"` | Enter (default `queue`) | Ctrl+Enter (terminal-dependent) | `ui.follow_up_behavior`, `combine_queued_prompts` | prompt queue pane (layout unknown) |
| Hermes 0.21 | **`/steer <text>`** | **`/queue <text>`** | Enter in `interrupt` mode | `display.busy_input_mode` — **the owner's config is `interrupt`** | `⏩ Steered:` / `Queued for the next turn:` in scrollback |
| OpenCode 1.18 | Enter (server loop picks it up at the next step) | none in the TUI (server v2 `delivery:"queue"` only) | Esc Esc, then send | none | ` QUEUED ` badge on the transcript message |
| Muse 1.3 | Alt+Enter "queue or steer" (which one: unknown) | same key? | unknown | none found | `Queued (N) · delivered after this turn` |
| Antigravity 1.2.9 | code paths exist (`sendMessageOrSteer`) | `queuedMessages` setting, values unknown | Esc cancels | `settings.json` `queuedMessages` | unknown |

## What the matrix says

1. **Plain Enter is not one mode.** It steers in Pi, Omp, Codex and
   OpenCode; it queues-then-absorbs in Claude Code; it follows up in
   Grok by default; and in Hermes with the owner's config it
   **interrupts the turn**. A door that just drops the `working` gate
   and presses Enter would do four different things — and cut the
   owner's Hermes turns.
2. **Keys and slash commands are per-CLI adapters.** Where a CLI has a
   slash command (Hermes `/steer`, `/queue`; Omp `/queue`) it is the
   robust choice: independent of user keybindings and config. Keys come
   next (Codex Tab, Claude Ctrl+X Enter); anything behind a user config
   has to read that config first or be refused.
3. **"Composer empty after Enter" no longer proves delivery.** Every CLI
   clears the input on a busy submit. The mid-turn receipt has to read
   the queue render above the input (or the scrollback line for Hermes)
   and is honestly **`queued`**, not `verified`; the message only
   reached the model when that row disappears.
4. **`needs-you` stays refused in every mode.** An approval dialog
   treats keys as choices.
5. **Muse and Antigravity** have no input reader in PiCode today
   (`doorReaderCLI`) and no known key map; they stay Prompt-only.

## Proposed shape (for the owner's decision)

- The attach composer gets the managed composer's `KindChip`
  (Prompt / Steer / Follow-up), showing only the modes the CLI's
  adapter declares; while the CLI is idle only Prompt is offered.
- `POST /api/terminals/{id}/prompt` gains `delivery:
  "prompt"|"steer"|"follow_up"`; an undeclared mode answers 409
  `unsupported-mode`. Automations keep `prompt` (and the `working`
  refusal) until a separate call.
- A per-CLI table (`internal/clikeys` or next to `doorReaderCLI`)
  names the sequence per mode, plus the queue-row matcher for the
  `queued` receipt.
- This changes ADR-0089's and Fatia F's (ADR-0160) "never type into a
  working CLI" rule — a process/security-model boundary, so an ADR.

### Decision table (draft)

| CLI state | Mode asked | Adapter declares it | Action |
|---|---|---|---|
| idle | prompt | — | today's verified door |
| idle | steer / follow_up | — | deliver as prompt (all CLIs send immediately when idle) |
| working | prompt | — | 409 `working` (today) |
| working | steer / follow_up | no | 409 `unsupported-mode` |
| working | steer / follow_up | yes, config-dependent key, config unread/mismatched | 409 `unsupported-mode` |
| working | steer / follow_up | yes | send sequence; receipt `queued` when the queue row appears, else `unconfirmed` |
| needs-you | any | — | 409 `needs-you` |
| occupied draft | any | — | 409 `occupied` |

## Live measurement

The desk read must be checked in a real pane before any adapter ships.
Protocol per CLI, on an isolated tmux socket in a scratch folder:
start a turn with a read-only tool call plus a long answer, send
`BRAVO` mid-turn with each candidate key/command, capture-pane every
second, and record (a) the input row after the key, (b) the queue
render, (c) whether BRAVO landed in the same turn or the next.

This session could not run it: the Claude Code auto-mode classifier
refuses a session that drives other agent CLIs by itself (even with
default permissions and a scratch folder). It needs the owner's
permission rule for the probe, or the owner running the protocol from
a PiCode terminal.
