# 2026-09-11 — feat/pkg-target-bar: Packages install bar

Shipped: desktop Agent CLIs → Packages. The install-target radios moved into
the install row (`.pkg-install-go`, visible "Install to" label, every control
at `--ctl-h`); the row is `[data-align-row] [data-align-wrap]` and wraps as a
unit. The embedded `settings-ctx` workspace line is gone from `Packages.jsx`
and `PackagesConfig.jsx`; `#cli-packages-view` owns the 12 px gap under the
pane tab bar (8 px below 767 px). `scripts/qa-cli-packages.mjs` clicks
`.pkg-by-source button[type=submit]` now that the radios live inside the form.

Verified: `make ci-scoped` PASS (5 files). Scratch instance `pkgbar`
(`qa-scratch.sh`), Chromium at 1600×1000 and 1024×900: happy with three
targets, empty (0 installed), blocked (workspace gone), list-read error and
the install overlay. `window.__picodeOverlayAudit()` ok in every state; row
children measure 36 px at equal tops; clicking a pill writes `scope=` to the
hash and moves `aria-checked`.

visual-review: PASS (11 screenshots in `var/screenshots/pkgbar/`; card 5/5)

Not done / debts: mobile Packages keeps the stacked row and its
`settings-ctx` line — a 390 px layout in the separate app (ADR-0072), out of
this request's scope. Pre-existing, not from this branch: below ~1100 px the
pane tab strip clips the last tab instead of scrolling (seen at 1024 px).
`docs/handoff.md` is unchanged — it is at 8181/8192 bytes, so this note
carries the two gaps above rather than overflowing the cap.

Merge: fast-forward ready.
