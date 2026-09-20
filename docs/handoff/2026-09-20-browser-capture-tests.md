# 2026-09-20 — the work-browser capture gets its host boundary tests (browser-capture-tests)

Branch cut from a9d5bb3c; one commit 5565af2e "desktop: host-test the work-browser capture decisions".

## What landed
- Pendency "legacy-shell COM capture + native hide path has no automated test"
  paid up to the host boundary: new pure `desktop-shell/src/preview.rs` with
  `settle()` (painted/failed/timed out/gone) and `read_still()` (refuses
  zero-byte captures, removes the temp file); `btab.rs` delegates to it; host
  tests `rustc --edition 2021 --test src/preview.rs` (2 tests).
- `coverDecision()` extracted from WebTab.jsx into
  `web/browser/src/lib/previewStill.js`, hide/restore decision table tested;
  the existing 5 preview tests kept (suite 466 pass).

## Behavior notes
- A capture whose tab closes mid-flight now reports "the tab closed while the
  capture was in flight" instead of the misleading "capture timed out".
- `permissions::site_of` pub(crate)→pub: 8117fd50 (allow-once) called it from
  bin-crate `btab.rs` and broke the cargo xwin build on main (E0603).

## Evidence
- `make ci-scoped` PASS (full matrix); `cargo xwin build` green (only pre-existing
  warnings); `rustc --test src/preview.rs` green; `make close` green.

## Next up
- Owner's Windows live run: the COM capture calls and the native hide/restore
  on screen (real pixels from a live page) — the only remaining acceptance.
