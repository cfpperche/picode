### Fixed
- **The terminal shows one scrollbar, never two.** The web terminal is a tmux
  client and tmux attaches on the alternate screen, so xterm has no scrollback
  of its own — and yet it painted two one-pixel artefacts at the right edge of
  every terminal: its own scrollbar (one pixel wide, because
  `overviewRuler.width` is also its `verticalScrollbarSize`) and the overview
  ruler's white outline. Both are hidden now, on desktop and mobile, together
  with the eight-pixel scrollbar gutter the viewport reserved for a bar nobody
  could see (Chromium only removed it on touch platforms; the terminal surface
  painted over it). The fit keeps reserving the one pixel, so the text grid is
  unchanged, and the wheel still scrolls what it always scrolled — tmux's
  history or the TUI's own viewport. The scrollbar a reader sees is the one
  that belongs to whoever holds the scrollback.
