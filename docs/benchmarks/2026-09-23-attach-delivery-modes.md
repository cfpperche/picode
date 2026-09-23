# Attach composer delivery modes across the nine CLIs

- **Date**: 2026-09-23
- **Question**: can the attach composer (ADR-0089 prompt door) deliver a
  message **while the CLI is working** — the Prompt / Steer / Follow-up
  choice the managed Pi composer already offers — for every agent CLI?
- **Status**: all nine CLIs measured live on 2026-09-23 (see
  [Method](#method)); Grok's steer setting and Hermes's plain Enter
  were deliberately not exercised.

## Today

`doorDeliverMode` (`internal/server/term_prompt.go`) refuses with 409
`working` / `needs-you` whenever the terminal state says the CLI is busy.
The door has one mode: Prompt, into an idle composer, verified by the
input row reading empty before and after Enter.

## Vocabulary

| Mode | Meaning |
|---|---|
| **Prompt** | Idle composer, starts a turn (today's door) |
| **Steer** | Reaches the model inside the running turn, at the next tool/step boundary |
| **Follow-up** | Held until the turn ends, then sent as the next turn |

## Measured matrix

`M` = observed live; `D` = desk-read only (source, strings, docs).

| CLI (version) | Steer | Follow-up | Input row after a busy submit | Queue render (read for the receipt) |
|---|---|---|---|---|
| Pi 0.87.1 | Enter `M` | Alt+Enter `M` (needs tmux `extended-keys on`, which PiCode sets) | cleared | `Steering: <text>` / `Follow-up: <text>` lines + `↳ Alt+Up to edit all queued messages`, above the editor |
| Omp 18.2.11 | Enter `M` | `/queue <text>` `M` (also Ctrl+Q, Ctrl+Enter `D`) | cleared | `Steering - N` / `After yield - N` + `  1. <text>` rows + `` `- Alt+Up/Shift+Up to edit``; status `Queued message for when the agent yields` |
| Claude Code 2.1.281 | Enter `M` — lands at the next tool boundary | none: Enter during pure text generation waits for the turn end `M`; Ctrl+X Enter behaved exactly like Enter `M` | cleared; dim `Press up to edit queued messages` | `❯ <text>` + `ctrl+x ctrl+s to send now` above the input while waiting; the same `❯ <text>` line stays in the transcript once absorbed |
| Hermes 0.21.4 | `/steer <text>` `M` | `/queue <text>` `M` | cleared; hint row `msg=interrupt · /queue · /bg · /steer` | `⏩ Steer queued — arrives after the next tool call: <text>`; `/queue` itself runs at turn end (`⚙️ /queue …`, `Queued: <text>`) |
| OpenCode 1.18.32 | Enter `M` | none in the TUI `D` | cleared | the message joins the transcript with a ` QUEUED ` badge |
| Muse 1.3.0 | Enter `M` | Alt+Enter `M` | cleared | `• Queued input` + `↳ <text>` rows; right-hand status `steering the running turn` / `queued for the next turn` |
| Antigravity 1.2.9 | none known | Enter `M` | cleared; `Press up to edit queued messages` | `▸ <text>` above the input |
| Codex 0.156.1 | Enter `M` (absorbed right after the running tool call) | Tab `M` | cleared | `• Queued follow-up inputs` + `↳ <text>` rows + `shift+← edit last queued message`; a steer shows as `› <text>` in the transcript (desk: `• Messages to be submitted after next tool call` while waiting) |
| Grok 1.0.41 | only with `ui.follow_up_behavior="steer"` `D` (unset in the owner's config) | Enter `M` | cleared | `#1 <text>` above the input + `Queued · Enter to send now`; hint row `Enter:send now · Ctrl+;:queue`. **An Enter on the empty composer then cancels the turn and sends** |

Not measured, on purpose: Hermes's plain Enter. The owner's
`display.busy_input_mode` is `interrupt`, which cancels the running
model call (`↪ Redirected current turn`) — the door must never use it.

## What the matrix says

1. **Plain Enter is not one mode.** It steers in Pi, Omp, OpenCode, Muse
   and Codex; it steers-or-waits in Claude Code depending on whether a
   tool runs; it follows up in Antigravity and Grok; in the owner's
   Hermes it interrupts. Dropping the `working`
   gate and pressing Enter would do five different things.
2. **Every mode is a per-CLI sequence.** Slash commands (Hermes
   `/steer` and `/queue`, Omp `/queue`) are the most robust: they do not
   depend on user keybindings or config. Keys come next (Pi and Muse
   Alt+Enter, Codex Tab). Grok's queue must never be followed by a bare
   Enter (it means "cancel and send now"). Claude Code has no distinct follow-up;
   OpenCode no follow-up in the TUI; Antigravity no steer.
3. **An empty input row no longer proves delivery.** Every measured CLI
   clears the row on a busy submit. The honest mid-turn receipt is
   **`queued`**, proven by the CLI's own queue render (column above);
   the row disappears when the message reaches the model.
4. **`needs-you` stays refused in every mode.** An approval dialog
   treats keys as choices.
5. Pi's Alt+Enter needs tmux `extended-keys on`; PiCode's sessions set
   it (terminal-bridge), the probe server did not until told to.

## Proposed shape (owner-approved direction, 2026-09-23)

- The attach composer gets the managed composer's `KindChip`
  (Prompt / Steer / Follow-up), showing only the modes the CLI's
  adapter declares; while the CLI is idle only Prompt is offered.
- `POST /api/terminals/{id}/prompt` gains `delivery:
  "prompt"|"steer"|"follow_up"`; an undeclared mode answers 409
  `unsupported-mode`. Automations keep `prompt` (and the `working`
  refusal) until a separate call.
- A per-CLI table names the sequence per mode plus the queue-row
  matcher for the `queued` receipt.
- This changes ADR-0089's and Fatia F's (ADR-0160) refusal of a working
  CLI — a process/security-model boundary, so an ADR.

First slice (measured, deterministic): Pi (Enter / Alt+Enter), Omp
(Enter / `/queue`), Hermes (`/steer` / `/queue`), Muse (Enter /
Alt+Enter), Claude Code steer (Enter), OpenCode steer (Enter),
Antigravity follow-up (Enter), Codex (Enter / Tab), Grok follow-up
(Enter; steer only after PiCode reads `ui.follow_up_behavior`).

### Decision table (draft)

| CLI state | Mode asked | Adapter declares it | Action |
|---|---|---|---|
| idle | prompt | — | today's verified door |
| idle | steer / follow_up | — | deliver as prompt (every measured CLI sends at once when idle) |
| working | prompt | — | 409 `working` (today) |
| working | steer / follow_up | no | 409 `unsupported-mode` |
| working | steer / follow_up | yes, but depends on a user config PiCode has not read | 409 `unsupported-mode` |
| working | steer / follow_up | yes | send the sequence; receipt `queued` when the queue render shows the text, else `unconfirmed` |
| needs-you | any | — | 409 `needs-you` |
| occupied draft | any | — | 409 `occupied` |

## Method

Isolated tmux server (`-S /tmp/claude-1000/ap.sock`, `extended-keys
on`), scratch git folder with a one-line `README.md`, each CLI with its
installed defaults and the owner's own model/config. Turn: "read
README.md three times, one call at a time, then write 1–30 and ALPHA".
Four seconds in, a bracketed paste of "also write BRAVO at the very
end" with the candidate key, then "reply only CHARLIE" with the
follow-up candidate. Captured the pane at +0.5 s (render) and after the
turn (order): BRAVO before the turn ends = steer; CHARLIE as a separate
turn = follow-up. Claude Code was also probed during a tool-free
streaming answer, where Enter waited for the turn end.
