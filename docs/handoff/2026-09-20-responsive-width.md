# 2026-09-20 — work browser: device toolbar, native half (responsive-width)

Branch cut from bc3dfa6e.

- "Responsive width" (owner's call 2026-09-19) landed: the ⋮ menu's
  "Show device toolbar" row plus a strip above the page — width presets
  390/768/1024/1280, zoom −/+ with %, reset. No CDP: the page narrows to a
  centered preset-width rect of the pane (`btab_bounds` now remembers the raw
  pane rect and re-narrows on every push), zoom is the controller's
  `ZoomFactor` clamped to WebView2's 0.25–5.0. Emulation (UA/touch/DPR)
  stays behind its ADR.
- Pure `desktop-shell/src/responsive.rs` (presets, `width_rect` centering
  with a 320 px floor, `clamp_zoom`, reset) — host tests
  `rustc --edition 2021 --test src/responsive.rs` (9); `btab.rs` delegates.
  JS half `lib/responsive.js` (presets/labels, `stepZoom`, `zoomLabel`) —
  node tests (7); the ⋮ menu's zoom stepper now uses the same `stepZoom`.
- Two new commands, `btab_responsive_set`/`btab_responsive_reset`: the three
  hand edits (main.rs, build.rs, capabilities/default.json) AND both
  generated sets travelled together. Per-tab state in `BtabState`, painted
  from `btab_meta`'s `responsive` receipt — survives tab switches and route
  returns; hiding the strip resets (no stranded narrow page). First show
  inherits the live zoom so the strip never shows a stale percent.
- UI verified in a real browser with a stubbed `__TAURI__` bridge: toggle,
  preset highlight, zoom steps, reset, meta-receipt restore on remount,
  hide-resets — all exercised on the built bundle; boot stays clean (no
  blank-window class).
- **Windows on-screen behavior (real resize + zoom of the native page) is
  ACCEPTED by the owner on Windows (2026-09-21)** — the strip toggles,
  the page narrows/centers, zoom applies, reset and tab-switch persistence
  hold. Live proof of the COM bounds/zoom calls; everything above them is
  host-tested.

## Next up
- Nothing — the row is closed. CDP device emulation (mobile UA, touch,
  devicePixelRatio) is a separate ADR when wanted.
