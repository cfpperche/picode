# 2026-09-06 — feat/cli-prompt-door: ADR-0089 + composer Photos

Shipped: study `docs/benchmarks/2026-09-06-cli-terminal-attach.md`;
ADR-0089 (user Send may paste into a CLI TUI; Inspector must not).
0087 is CLI lifecycle and 0088 is cross-CLI handoff on main; this door is 0089.
Managed composer gains an **Attach image** button (`<input type="file"
accept="image/*" multiple>`) on desktop and mobile. Paperclip stays
workspace-only. Caps 4 × 4 MB (`planDeviceImages` table).

Verified: `make ci-scoped` PASS. Scratch `:8471` agent Atlas: empty
composer + one chip after `upload` (desktop 1280 and mobile 390).
`overlayAudit` ok. Screenshots in `var/screenshots/composer-*-{desktop,mobile}.png`.

visual-review: PASS (composer-empty/chip desktop+mobile; overlayAudit ok)
uiux-review: PASS (native file input; existing attach height; no jargon)

Not done: D2–D6 (drop/prompt HTTP, CLI attach bar/sheet, artifacts).
iPhone Photos sheet is owner acceptance (Chromium file chooser here).

Merge: fast-forwarded to `main` as `9b5f99dd`.
Deployed: `0.1.0+d04ad3f` (unguarded; health ok).
