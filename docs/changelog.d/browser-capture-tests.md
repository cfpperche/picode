### Added
- **Automated coverage for the work-browser capture path.** The decisions of
  `capture_png`/`btab_preview` (capture completed / failed / timed out / tab
  gone mid-capture) and the read-back gate now live in a pure module
  (`desktop-shell/src/preview.rs`) with host tests — a still is served only
  with non-empty pixels; a zero-byte capture ("the page has not painted") is
  refused. The legacy hide/restore decision of the options menu
  (`coverDecision` in `previewStill.js`) carries the same table tests. The
  COM capture and native hide themselves stay owner-verified on Windows.

### Changed
- A capture whose tab closes mid-flight now reports "the tab closed while the
  capture was in flight" instead of "capture timed out" — the same refusal,
  naming the right failure.
