# 2026-09-06 — mobile IME accessory

Branch: `feat/mobile-keybar-accessory` (worktree `.worktrees/mobile-keybar-accessory`).

Owner asked to stop the phone keyboard covering the composer/prompt and the
virtual keys, and to collapse the Termux 2×7 grid to one scrolling row that
opens and closes with the IME (Fable screenshot).

Shipped in this branch:

- `web/shared/domain/keyboardInset.js` — inset math, extra-keys visibility,
  conservative hardware heuristic, `sendTermSeq`.
- `#m-app` pinned to `visualViewport` (`useVisualViewport`).
- `KeyBar` one row, Fable-first keys, hide pinned on the right.
- Terminal screen + agent Terminal segment: accessory follows focus.
- `TerminalDock` now has sticky Ctrl/Alt like `ShellTerm`.
- ADR-0044 amendment, `www/guide/mobile.md`, changelog.

`make ci-scoped` PASS on this branch. visual-review: UNVERIFIED — this
environment cannot open an iOS software keyboard. Desktop Chrome device
mode can show the row on focus but cannot prove overlay geometry. If iOS
26 still reports `visualViewport.height === innerHeight` with the keyboard
painted, the focus-gated fallback in the ADR must land before calling
this done.
