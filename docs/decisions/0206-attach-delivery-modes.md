# ADR-0206: Attach delivery modes — steer and follow-up into a working CLI

- **Status**: accepted (owner, 2026-09-23: "aprovado" on the study's direction; "ok pode seguir" after the nine-CLI measurement)
- **Date**: 2026-09-23
- **Boundary**: protocol (`/api/terminals/{id}/prompt` and `/api/agents/{id}/prompt` gain `delivery`; a new `GET` on the same path) and security model (PiCode may now type into a CLI that is mid-turn, which ADR-0089 and Fatia F of ADR-0160 refused)
- **Amends**: ADR-0089 (the door's "working" refusal), ADR-0160 Fatia F (gated delivery)
- **Study**: [2026-09-23 — attach delivery modes](../benchmarks/2026-09-23-attach-delivery-modes.md)

## Context

The attach composer refuses with `working` / `needs-you` whenever the CLI
is busy, so the one moment a person most wants to correct an agent is the
moment the door is shut. Every one of the nine CLIs accepts input
mid-turn, but not the same way: measured live, a plain Enter steers in Pi,
Omp, Codex, OpenCode and Muse, steers-or-waits in Claude Code, follows up
in Antigravity and Grok, and **interrupts the turn** in a Hermes whose
`busy_input_mode` is `interrupt` (the owner's). Each CLI clears its input
row on a busy submit, so "the row reads empty" no longer proves anything.

## Decision

The door has three modes — **prompt** (today), **steer** (reaches the
model inside the running turn) and **follow_up** (held until the turn
ends). A per-CLI adapter table in the server declares which modes each CLI
has and the exact sequence for each (a key after a bracketed paste, or a
slash command); a mode a CLI does not declare is refused as
`unsupported-mode`, never approximated. The receipt for a mid-turn send is
`queued` when the text shows up on the pane outside the input row, else
`unconfirmed`; no mid-turn send is ever followed by a second Enter (in
Grok an Enter on the empty row means "cancel the turn and send now").
Rules that do not move: `needs-you` is refused in every mode; a draft in
the row is refused; an idle CLI gets today's verified prompt whatever mode
was asked; automations and the browser extension stay prompt-only. Pi
agents (receiver path) pass the mode to `pi.sendUserMessage` as
`deliverAs`.

### Decision table

| CLI state | Mode asked | Adapter declares it | Action |
|---|---|---|---|
| idle / unknown | any | — | today's prompt door (verified where a reader exists) |
| working | prompt | — | 409 `working` (names the modes the CLI has) |
| working | steer / follow_up | no | 409 `unsupported-mode` |
| working | steer / follow_up | yes | row read: draft → 409 `occupied`; else paste + the mode's sequence, receipt `queued` / `unconfirmed` |
| needs-you | any | — | 409 `needs-you` |
| any | steer / follow_up, unattended sender | — | not offered (automations send prompt) |

## Consequences

- A person can correct or queue work on a running agent from the attach
  composer, on the desktop and the phone, for all nine CLIs.
- The adapter table is a second place per-CLI knowledge lives (beside the
  input reader); a CLI release that moves a key breaks one row, and the
  receipt degrades to `unconfirmed` instead of lying.
- Hermes's plain Enter is never used; Grok steer is not offered until
  PiCode reads `ui.follow_up_behavior`; Claude Code and OpenCode have no
  follow-up, Antigravity and Grok no steer.
- An older Pi receiver ignores `deliverAs` and follows up — a steer asked
  of it lands late, not wrong.

## Alternatives considered

- **Drop the working gate and press Enter** — five different behaviours
  behind one button, and a cancelled Hermes turn.
- **Interrupt then send** — every CLI can, but it discards work in
  flight; not offered in this ADR.
- **Refuse steer in the UI, queue in PiCode until idle** — a server-side
  queue duplicates what every CLI already does natively and needs its
  own persistence.

## Amendment 2026-09-24 — Stop and send (owner: "aprovado, pode executar")

A fourth mode, **interrupt** ("Stop and send" in the composer), stops the
running turn and sends the message as a new prompt — the alternative this
ADR first left out. It does not use each CLI's native "send now" chord:
the door presses the CLI's measured stop key (Esc; Esc twice for
OpenCode; Ctrl+C for Grok and Hermes — exactly one, since a second within
2 s force-exits Hermes), waits up to 3 s for the CLI's own stop line
(`Interrupted` / `aborted` / `cancelled`) or for its state to leave
`working`, and only then runs today's verified prompt path. A stop it
cannot see is refused as `not-stopped` with nothing pasted; a composer the
stop refilled (Muse gives back a prompt it retracted) is refused as
`restored` instead of being appended to. needs-you, drafts and automations
stay as above; an idle CLI gets a plain prompt. The composer never selects
it by default. Pi agents abort through the receiver (`ctx.abort()`, then
wait for idle) before `sendUserMessage`.

| CLI state | Mode | Action |
|---|---|---|
| idle / unknown | interrupt | prompt door |
| working | interrupt | stop key → stop seen → verified prompt; not seen → 409 `not-stopped`; field refilled → 409 `restored` |
| needs-you | interrupt | 409 `needs-you` |
