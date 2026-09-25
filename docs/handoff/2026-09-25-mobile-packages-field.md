# 2026-09-25 — feat/mobile-packages-field: phone Packages source field takes the whole row
Shipped: owner asked to fix the cut placeholder of the phone Packages "install
by source" field. At 390px the input had 294px beside Install; at 16px (the
phone's no-zoom font) the placeholders need ~341 (Pi) and ~343 (Codex/OpenCode;
Claude and Omp show no source form). web/mobile/src/styles/mobile-settings.css:
`.pkg-by-source` wraps, input takes the full row (362px), Install full-width
below (8px gap, 36px) — the same shape as Backup's folder field. Desktop untouched.
Verified: `make ci-scoped` PASS; probe on pi/codex/opencode (362 ≥ need, no page
overflow); visual-review PASS on the three. Blind spot: on a 360px phone the
input is 332px, short of 343, so the Codex/OpenCode placeholder is cut by a few
characters there. Seen in review, not fixed: in Codex and OpenCode an empty ~16px
band sits between the Installed/Marketplace tabs and "Nothing installed." (not in Pi).
Not done / debts: none. Merge: fast-forward ready.
