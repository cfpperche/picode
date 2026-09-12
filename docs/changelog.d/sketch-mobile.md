### Fixed
- **The sketch pad on the phone fits the screen.** The drawing surface now
  fills the usable area — no header under the status bar, no dead strip under
  the drawing — and its Cancel/Insert buttons are thumb-sized (44px). It lives
  inside the shell's viewport, so the software keyboard shrinks it correctly.
- **The sketch canvas is dark in dark mode.** The pad asked Excalidraw for a
  near-black canvas, which its dark-theme filter inverted back to light; the
  canvas is plain white paper now, so the screen is dark and the PNG attached
  to the terminal stays a white sheet.

### Changed
- Opening a sketch on the phone starts with the pen: draw immediately instead
  of tapping the pencil in the toolbar first.
