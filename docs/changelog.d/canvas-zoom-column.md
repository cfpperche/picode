### Changed

- **The Canvas camera controls moved to the bottom-left, as a column.** Zoom
  in, zoom out, the 100 % readout and **Fit** now stand at the far end of the
  bottom edge, with the minimap opposite them on the right instead of
  stacked above. **Fit** is an icon there; its keyboard shortcut (`0`) is
  unchanged.

- **A Canvas panel resizes from its bottom edge, its right edge and the
  corner between them.** The other five targets are gone: dragging one of
  them moved the panel's top-left corner while it resized, so the panel
  slid away from under the pointer.

### Fixed

- **Zooming out no longer turns a terminal panel white.** Below 75 % a panel
  shows its last screen as text instead of a live terminal, and that text
  was landing on the panel's own light ground — the same words, suddenly
  looking like something other than a terminal. The still now keeps the
  terminal's background and foreground, following your terminal theme.
