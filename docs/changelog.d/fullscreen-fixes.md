### Changed
- **Fullscreen: the exit control is a button again, not a sentence.** The
  label and the chord moved into its hint (`Leave fullscreen
  (Ctrl+Shift+Enter)`), and the glyph now sits in the same 28px square as the
  Inspector toggle beside it.

### Fixed
- **Fullscreen: the Inspector button in the top strip shows the panel.** It
  only flipped the rail's dock preference while fullscreen keeps the rail off
  screen by itself, so the click changed an icon and nothing on the page.
  Inside the mode it now lays the rail over the page at once — opening the
  rail when it was closed — and hiding it there takes only the overlay down,
  never the dock you set outside the mode. A panel opened that way stays until
  the same button, Escape or the mode: it no longer slides shut behind a
  pointer that never entered it. The right edge still works, and the rail can
  be turned on from the strip with the mode already running.
