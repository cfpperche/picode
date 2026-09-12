# 2026-09-11 — feat/term-scrollbar: one scrollbar, and it is not the web's

Shipped (fast-forward ready): two scrollbars in an Agent CLIs terminal. Cause:
xterm 6 draws its own scrollbar at `right: 0`, and `overviewRuler: { width: 1 }`
(2026-08-31, meant to kill a 14px fit gutter) is *also* that scrollbar's width —
a one-pixel bar, colour-identical to the owner's screenshot; the same value made
the ruler paint a white outline in that column, and Chromium still reserved the
viewport's 8px gutter under it (canvas 1000px over `clientWidth` 997px). The
other bar is pi's own TUI scrollbar (`│`/`┃`, `scrollbarThumb: text` =
`#d4d4d4`), not ours to remove. Now `app.css` + `mobile.css` hide the viewport
scrollbar and xterm's own, `termTheme.js` paints the ruler border `#00000000`,
and `overviewRuler.width` stays 1 (no dead gutter). Wheel, keys, tmux
copy-mode, the TUI's bar: unchanged.

| Condition | Before | After |
|---|---|---|
| Shell terminal in tmux (alternate screen) | 8px dead gutter (Windows), no bar visible | no gutter, no bar |
| Agent CLI TUI terminal (alternate screen) | the same, plus pi's own TUI bar | the same minus every web bar |
| xterm that *has* scrollback (normal buffer) | 1px slider **and** the ruler's white outline | both absent; the wheel still scrolls it |
| Mobile shell | viewport bar already hidden, xterm's 1px bar could paint | both hidden |

Verified live (scratch, headed Chromium with real 8px scrollbars): a scrollback
probe's pinned slider was 1px × 51px at the right edge and the ruler column
white — **absent after** (0 non-background pixels in the four columns at that
edge); the gutter went 8 → 0. Screenshots read: TUI running, CLI terminal
stopped (blocked line + action), shell terminal, probe before/after
(`var/screenshots/`, gitignored). Overlay audit ok. Guard:
`scripts/term-scrollbar.test.mjs`.

Debt: a *draggable* browser-side bar is the tmux client off the alternate screen
— a decision, and it changes what the wheel scrolls (`docs/handoff.md`).

Merge: fast-forward ready; the owner deploys.
