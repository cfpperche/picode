# Terminals

## Next

- Tab strip: keyboard close of `.mtab-close`; live needs-you arrow; `scrollbar-width: thin` vs webkit.

## Debts

- Terminal menus (2026-09-09): web/mobile rows offer only Remove; the desktop one-menu merge (`termRowMenu.js`) is not ported; desktop **Continue in…** (2026-09-13) waits on it.
- Scrollbars: the web terminal draws none (a tmux client has no scrollback — `term-scrollbar.test.mjs`); a draggable bar means taking tmux off the alternate screen, a decision.
- CLI pane-death (ADR-0085): an owned OpenCode QA stop left two processes after its pane closed; ADR-0084 pins nothing for terminals stopped before it.
- tmux: never kill by prefix (a prefix sweep killed 29 sessions in 2026-09-06) — exact names from a fixture's API only; a scratch whose daemon dies before `qa-scratch stop` strands its shells (one kept on purpose).
