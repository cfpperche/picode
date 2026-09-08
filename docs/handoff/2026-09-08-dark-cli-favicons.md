# 2026-09-08 — dark-cli-favicons: CLI favicons visible on dark theme on every surface

Shipped: the dark-mode `filter: invert(1)` for transparent monochrome vendor
SVGs moved from `agent-clis.css` (scoped to `#agent-clis-view`) into both
apps' `styles/app.css` next to `img.ws-face.term-cli-face`, with no view
scope. Mobile Work rows, desktop sidebar/tabs/inspector and the Agent CLIs
catalog now share one rule; Pi (pi.dev) and colored raster fallbacks stay
native, as before.

Verified: scratch instance (qa-scratch darkfix) with seeded terminals
claude-code/codex/pi/shell — computed `filter` is `invert(1)` for
`/icons/` srcs and `none` for pi.dev in dark, `none` for all in light;
screenshots read for mobile Work dark, desktop sidebar dark, desktop
Agent CLIs dark; `window.__picodeOverlayAudit()` ok; light theme unchanged.

visual-review: PASS (mobile-work-dark-fixed.png, desktop-dark-fixed.png,
desktop-clis-dark-fixed.png; card 5/5)

Decision table (theme × src class): dark + monochrome `/icons/` or grok.com
fallback → invert; dark + pi.dev/raster → native; light + anything → none.
All rows verified in browser; no unit test covers CSS (accepted debt).

Not done / debts: none known. Docs captures refreshed as a side effect.

Merge: fast-forward ready.
