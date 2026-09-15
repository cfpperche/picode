### Added
- **One release, two binaries, one resident to swap (ADR-0142, slice 3).**
  The release workflow builds `picode-shell-windows-amd64.exe` beside the
  Go binaries (Rust + cargo-xwin recipe, proven on a fresh Ubuntu 24.04
  container) and stamps the tag into the shell. `picode-desktop update`
  downloads both exes, verifies them against `SHA256SUMS` — refusing an
  unverified binary like the daemon's updater — and swaps both or neither,
  rolling the tool back if the shell fails. `scripts/desktop-swap.sh`
  relaunches whoever the `PiCodeDesktop` task points at (tray today, shell
  after the migration).
