# ADR-0156: computer — an input action lands only in the window the agent last saw (amends ADR-0148)

- **Status**: accepted (owner, 2026-09-22; asked for in session 2026-09-18
  after the incident below, and shipped since)
- **Date**: 2026-09-18
- **Boundary**: security model — the first refinement of ADR-0148's
  "one grant, everything" policy: the computer tool's input actions gain a
  precondition the grant does not override. It is item (d) of the named
  refinements ("human-activity pause and foreground checks"), pulled forward.

## Context

ADR-0148 gives a granted principal unrestricted input on the human's own
desktop, and lists the refinements to add against real use. The first live
run from Claude Code (2026-09-18, through `picode mcp computer`, ADR-0154)
produced the case: the agent listed windows (Notepad in front), then typed
a sentence; between the two calls the human clicked their PiCode window,
and the keystrokes went into the composer of another terminal, the leading
newline submitting a message the human was writing. Nothing in the line
was wrong — the grant was on, the identity right, the text intact (once
the pacing fix of the same day landed) — the action simply landed where
the human had just moved.

The shell already refuses `focus` when Windows keeps the foreground with
the human (`foreground_refused`); it has no notion of what the agent last
looked at. Every capture and every `focus` is a moment where the agent
knows which window is in front.

## Decision

The shell records, per principal, the top-level window in front at the
moment of each capture (`screenshot`, `zoom`, `wait`, and the recapture
after an action) and of each successful `focus`. Every input action —
the five clicks, `left_click_drag`, `mouse_move`, `left_mouse_down`/`up`,
`scroll`, `type`, `key`, `hold_key` — runs only while the window in front
is still that one; otherwise it is refused with the head
`foreground_changed`, naming the window now in front, before any input is
injected. A principal that has not looked yet is refused the same way.
Non-input actions (`screenshot`, `zoom`, `snapshot`, `windows`, `focus`,
`cursor_position`, `wait`, `clipboard_*`, `open`) are unchanged. The
refusal is recorded like any other (`computer.step`, outcome refused).

## Consequences

An agent that follows the tool's own guideline — look, then act — pays one
comparison per action and notices nothing. An agent whose target the human
moved away from gets a refusal that tells it what to do (look again, or
`focus`), instead of typing into whatever the human is using. The check is
between two calls, not within one: a human who clicks *during* a `type`
still gets the tail of it — the human-activity pause half of refinement (d)
stays open. `focus` followed by input in one turn is the reliable recipe;
the guideline says so. The window id is the top-level (root) window, so a
click that moves focus between controls of the same window does not
refuse. If the front window closes between look and act, the refusal
names "another window" and a capture resets the record.

## Alternatives considered

- **Pause while the human is active (input idle time)**: deferred, the
  other half of (d); it needs a threshold measured against real use, and
  the foreground check alone closes the case that happened.
- **Bind the principal to one window for the session (M5 a)**: a wider
  policy change with tiers; this decision is the minimal precondition and
  composes with it later.
- **Refuse only `type` and `key`**: rejected — a click into the wrong
  window is the same failure with a mouse.
