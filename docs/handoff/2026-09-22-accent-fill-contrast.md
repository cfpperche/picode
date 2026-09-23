# 2026-09-22 — accent-fill-contrast: text on accent fills passes AA in dark

Shipped: new tokens `--accent-fill` / `--on-accent` in `web/shared/tokens/theme.css` (dark `#4f5dde`, light =
`--accent`). The fills that carry white text read them: `.btn-primary` (desktop and mobile), `.icon-btn-send`,
`.annot-strip .annot-send`, a pressed Canvas tool. `--accent` stays on links, focus, borders, dots, sizers,
switches and badges (badges use dark text and already passed). `web/tools/accent-contrast.test.mjs` checks both
themes at ≥ 4.5:1 and refuses any rule with white text on the bare accent (it catches the old stylesheet).
Verified: `make ci-scoped` PASS; visual-review PASS on scratch — measured 5.31 (dark) / 5.50 (light) at rest and
4.70 / 4.85 on hover from rendered pixels; dialogs overlayAudit ok; link and focus colours unchanged.
Not seen on screen: `.icon-btn-send` and the annotation send (no composer on the scratch), a disabled primary,
light mobile. Pre-existing: keyboard focus on a primary button draws the browser's default outline, not a themed ring.
