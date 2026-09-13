# Terminals

## Next

- Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.

## Debts

- Terminal menus (2026-09-09): web/mobile rows offer only Remove; the desktop one-menu merge (`termRowMenu.js`) is not ported.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking the tmux client off the alternate screen, a decision.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- tmux: never kill by prefix (a `grep '^picode-'` sweep killed 29 sessions, six in production, 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells (one is kept on purpose).
- Grok attention (2026-09-13): the `PermissionRequest` hook group PiCode installs is inert — Grok 1.0.30 has no such event (it has `PermissionDenied`), and permission attention arrives as `Notification permission_prompt`; the group stays only so a shared Claude settings file loads. A *permission* hold also has no release identity, so a sibling tool's `PostToolUse` in the same parallel batch still clears it (the question hold got one in `2026-09-13-grok-question-hold`).
