# ADR-0148: Computer use for agents — one grant, the whole desktop, refinements later

- **Status**: proposed (the owner approved the plan in session on 2026-09-17;
  this text awaits their read)
- **Date**: 2026-09-17
- **Boundary**: security model — an agent, or a CLI running in a PiCode
  terminal, once granted, acts on the user's Windows desktop with the user's
  own permissions: the screen, any window, mouse, keyboard, clipboard,
  launching programs. Protocol — the shell line of ADR-0132 carries a second
  command family (`kind: "computer"`) beside the CDP one. Process — the
  desktop shell gains an actuator role: a worker thread that captures the
  screen, injects input and reads UI Automation.

## Context

The study `docs/benchmarks/2026-09-16-computer-use.md` mapped the field. The
loop converged in 2025–2026: a screenshot goes in, one of a small fixed set of
actions comes out, coordinates are in the screenshot's own pixel space.
Anthropic, OpenAI and Google ship the same fifteen verbs; agents beat the
OSWorld human baseline on short tasks and still fail most hour-long ones.

PiCode already owns the three hard parts for the browser: a per-principal
grant (ADR-0128/0134/0143/0144/0146), a shell↔daemon command channel with no
open port and no second credential (ADR-0132), and human co-visibility
(ADR-0135). What no other process can do is see or drive the Windows desktop:
the agents run in WSL, and only the shell (ADR-0120, the sole Windows
resident since ADR-0142) holds a window station. The shell is the actuator by
construction.

Facts that shape the choice, measured 2026-09-17: the `windows` 0.61 crate
the shell already depends on ships the bindings for UI Automation,
`SendInput`, GDI and Graphics Capture, per-monitor DPI, the clipboard and OCR
— own code costs feature flags, not a dependency. Adding the ecosystem
crates (xcap, uiautomation, windows-capture) brings a second `windows`
generation until Tauri releases its move to 0.62. pi 0.85 carries image
blocks in tool results. The hub of ADR-0132 is pure transport.

The owner's direction (2026-09-17): capability first — one grant to an agent
or CLI, the tools unrestricted, the other questions refined afterwards. The
browser went the other way, tiers before use, and the first live run with a
full grant found three verb renderers broken (the 2026-09-17 note in
`packages/pi-browser/src/logic.ts`). The policy shape is better designed
against real use than ahead of it.

## Decision

A `computer` tool (the pi package `pi-computer`, a sibling of `pi-browser`)
gives a principal twenty-three actions — Anthropic's seventeen
(`screenshot`, `zoom`, the five clicks, `left_click_drag`, `mouse_move`,
`left_mouse_down`/`up`, `cursor_position`, `scroll`, `type`, `key`,
`hold_key`, `wait`) plus `windows`, `focus`, `snapshot`, `clipboard_read`,
`clipboard_write` and `open` — behind exactly one grant:
`computer.policy.<key>` = `{"enabled": true}`, keyed by the house identity
rule (`internal/grant`: agent id, else `term:<terminal id>`, else nothing).
A missing or unreadable grant is off; a caller with no identity is off. With
the grant there is no further gate: screenshots are a whole monitor,
coordinates are in the pixel space of the last image the shell returned,
input goes wherever the model points, any window may be focused, the
clipboard read and written, programs launched. The shell executes every
action on one dedicated worker thread (one action at a time on the machine,
no COM object crossing threads); commands ride the ADR-0132 stream as
`kind: "computer"` frames carrying the principal, results post to the same
route, and every call — allowed, refused or failed — appends one
`computer.step` event to the change feed as a record, not a boundary. Own
code on `windows` 0.61; no new crate, port, credential or sidecar; Windows
only. The refinements are named here as the next decisions, each its own
amendment, in the owner's order: a window binding with read/act/full tiers,
mechanical Ask classes (password fields, launching executables, clipboard
writes), an app allowlist, a pause on human activity with foreground checks,
an integrity check against elevated windows and a step budget, and a
disposable arena (Windows Sandbox) for the riskiest work.

## Consequences

- **Easier**: the feature works the day it ships and is one switch to
  understand; the vocabulary maps one-to-one onto the vendors' contracts, so
  their documentation and a Claude-backed agent's habits apply; the plumbing
  is the browser's, so a second EventSource, a second hub and a second
  identity never exist.
- **Harder, accepted**: an agent with the grant is the user at the keyboard.
  Prompt injection from screen content has measured success rates of 7–86 %
  without a policy layer (the study, §1.4); the grant is per principal and off
  by default, the public page says plainly that the agent acts with the
  user's permissions and without a sandbox, and the audit tells the owner
  afterwards. The shell takes on physical-pixel geometry (per-monitor DPI,
  scale and origin travel with every capture) and a serialized actuator, so a
  five-second `wait` from one agent delays another's screenshot.
  `PrintWindow` returning black, UIPI dropping input into elevated windows
  and the foreground lock surface as named errors, not as silent success.
- **If we're wrong**: the grant is one setting — off, the door closes on the
  next call; the package uninstalls like any pi package; the three hub fields
  are optional and the browser path is untouched; the refinements above are
  additive amendments, not a redesign.

## Alternatives considered

| Alternative | Why not |
|---|---|
| Tiers, a window binding and Ask classes first (the study's §7 shape) | The owner's call: it front-loads a policy model nobody has used yet; kept, in full, as the refinement list |
| A second hub and a second stream for the computer family | Three duplicated routes, a second EventSource, a second "connected" state for one shell window; the hub is pure transport and three optional fields suffice |
| Adopt lahfir/agent-desktop or the xcap/uiautomation/enigo crates now | The Windows adapter is unreleased; the crates bring a second `windows` generation until Tauri 0.62; agent-desktop stays a benchmark for its patterns (refs re-identified at action time, headless default, per-session trace) |
| A Python/pyautogui sidecar on Windows (Agent S, Windows-Use, the Anthropic quickstart) | A second runtime to install, update and sign beside the only Windows resident (ADR-0142) |
| A sandbox or agent workspace first | No public Windows API spawns an interactive agent desktop (Agent Workspace is a private preview; MXC session isolation has no display server); Windows Sandbox is a later spike |
