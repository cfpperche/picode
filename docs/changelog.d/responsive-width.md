### Added
- **Device toolbar ("Responsive width") in the work browser — the native
  half.** The ⋮ menu gains the reference's "Show device toolbar" row; the
  strip above the page offers width presets (390/768/1024/1280), zoom −/+ and
  a reset. Picking a width narrows the page to a centered preset-width rect of
  the pane (WebView2 bounds arithmetic, no CDP — mobile UA/touch/DPR would be
  the emulation half and needs an ADR); zoom rides the controller's
  `ZoomFactor`, clamped to WebView2's 0.25–5.0. State is per tab in the shell
  and rides `btab_meta`, so the strip survives tab switches and route returns;
  hiding the toolbar resets the tab (width, zoom, full pane bounds). The
  bounds/reset/zoom-clamp math lives in pure `desktop-shell/src/responsive.rs`
  (host-tested) and the preset/label/zoom ladder in
  `web/browser/src/lib/responsive.js`; the on-screen resize/zoom behavior is
  pending the owner's live Windows acceptance.
