### Added

- **The tmux guard has a switch.** Preferences ▸ Terminal (Terminal defaults)
  gains a **Safety** section: the tmux guard shows its state and toggles
  between *On* (refuses `kill-server`, pattern kills and other terminals'
  sessions inside PiCode terminals) and *Off*. The line says the change
  applies to terminals opened from now on, links to how it works, and keeps
  the switch honest under failure — a failed refresh says so and offers
  **Try again**, a failed toggle reverts and reports.
- The wiring status now installs the guard on first read, so a fresh instance
  reports the default-on state instead of "off" until its first terminal.

### Fixed

- Terminal settings: a failed refresh of the guard no longer leaves the old
  state on screen without a word.
