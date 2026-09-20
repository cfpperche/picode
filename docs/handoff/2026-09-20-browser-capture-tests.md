# 2026-09-20 — work-browser capture: host-boundary tests (browser-capture-tests)

Branch cut from a9d5bb3c; commit 5565af2e "desktop: host-test the work-browser
capture decisions".

- Pendency "legacy-shell COM capture + native hide path has no automated test"
  paid up to the host boundary: pure `desktop-shell/src/preview.rs` —
  `settle()` (painted/failed/timed out/gone) and `read_still()` (refuses
  zero-byte captures, removes the temp file); `btab.rs` delegates; host tests
  `rustc --edition 2021 --test src/preview.rs`.
- `coverDecision()` extracted from WebTab.jsx into `previewStill.js` with the
  hide/restore decision table test; the 5 existing preview tests kept
  (suite 466 pass).
- A capture whose tab closes mid-flight now reports "the tab closed while the
  capture was in flight" instead of the misleading "capture timed out".
- `permissions::site_of` pub(crate)→pub: 8117fd50 (allow-once) called it from
  bin-crate `btab.rs` and broke the cargo xwin build on main (E0603).
- Evidence: `make ci-scoped` PASS (full matrix); `cargo xwin build` green
  (pre-existing warnings only); `make close` green.

## Next up
- Owner's Windows live run: the COM capture calls and the native hide/restore
  on screen (real pixels from a live page) — the only remaining acceptance.
