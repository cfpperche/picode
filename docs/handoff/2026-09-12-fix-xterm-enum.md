# 2026-09-12 — feat/fix-xterm-enum

Symptom: OpenCode (and any TUI sending a DECRQM) froze a browser terminal at
"Starting OpenCode..." until a page reload; the console showed
`Uncaught ReferenceError: n is not defined` inside the xterm write path.

Root cause: esbuild's minifier, on xterm 6.0.0's **ESM** build
(`lib/xterm.mjs`, `let r; ... (r ||= {})` enum IIFE in `requestMode`), dropped
the declaration but kept the write — `(void 0 || (n = {}))`. Strict-mode module
→ the first DECRQM (`CSI ? 2026 $ p`) threw inside `_innerWrite`, killing the
scheduled write loop; the ws kept flowing (verified: +60 KB per keystroke) but
nothing parsed. Reload worked because tmux's full redraw does not replay the
query. Reproduced end-to-end on scratch (plain terminal + typed `opencode`,
and the agent-CLI path).

Fix: `web/tools/vite-config.mjs` pins `@xterm/xterm` to `lib/xterm.js` (UMD,
enum pre-compiled) via a `$`-anchored alias, desktop and mobile. Verified:
built bundles contain no enum IIFE / broken pattern; live boot now paints the
TUI with zero console errors (screenshots in `var/screenshots/`).

Debt: revisit when xterm drops the enum or esbuild stops dropping the
declaration — grep the built bundle for `(void 0||(` next to `requestMode`.
Rejected: `minifyIdentifiers: false` (insufficient — inline pass still drops
the declaration); esbuild 0.28 override (vite 6.4.3 crashes on it).
