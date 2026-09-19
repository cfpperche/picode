# 2026-09-18 — feat/computer-foreground-guard (ADR-0156, refinement d of ADR-0148)

Why: in the first live run from Claude Code, a `type` meant for Notepad
landed in the owner's own OpenCode terminal because they clicked it between
the agent's `windows` and its keystrokes (the leading newline submitted the
message they were writing).

What: the shell records, per principal, the top-level window in front at
each capture and each successful `focus` (`Desk.seen`), and every input
action (clicks, drag, mouse_move, mouse down/up, scroll, type, key,
hold_key) refuses with `foreground_changed` — naming the window now in
front — when that changed; a principal that never looked is refused too.
Non-input actions are unchanged. Docs: ADR-0156, guide section "Where a
click or a keystroke lands", architecture heads, plan M5 (d), guideline
text in pi-computer and mcptool.

Proof: cross `cargo check` clean; `make desktop-shell` builds. Live proof
after `make desktop-restart`: screenshot → owner clicks another window →
`type` answers `foreground_changed`; `focus` Notepad → `type` lands.

## Next up
- Owner: `make desktop-restart`, then the two-step proof above from the Claude Code terminal.

## Debts
- The other half of (d), a pause while the human is actively typing or clicking, is still open — needs a measured idle threshold.
