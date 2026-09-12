### Changed
- **Fullscreen hides PiCode, and leaves your browser window alone.** Turning
  it on no longer takes the browser fullscreen with it: your other tabs and
  your address bar stay where they are, and the mode is exactly what it says —
  the sidebar, the tab strip and the Inspector step aside, the edges bring
  them back, `Esc` closes an open panel and leaves on the next press. `F11`
  still works and now stacks on top of the mode, in either order.

### Removed
- **Fullscreen no longer hands the browser's reserved keys to your agent.**
  Capturing `Ctrl+T`, `Ctrl+W` and the rest needs a page that asked the
  browser for fullscreen itself, which the mode no longer does; those keys
  stay with the browser in every window. Everything else still reaches the
  terminal untouched.
