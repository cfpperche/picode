### Added

- **Windows setup finishes on a clean machine.** `picode-desktop install`
  now delivers the Linux binary and the runtime itself: an `install-picode`
  stage (the release matching the tool, verified, placed where provisioning
  resolves it) and an `install-runtime` stage (tmux, git, curl, Node.js from
  the NodeSource repository, pi) between account creation and provisioning.
  Ubuntu only; anything already present is left alone; a distro PiCode did
  not register asks before apt and npm run (`--yes` skips the question);
  the install ends with the shell running in the tray.
- **Per-account installs.** `install --user <name>` puts the picode binary
  in that account's `~/.local/bin` and installs pi for it alone (its own
  npm prefix, linked where the login shell finds it). Without the flag pi
  stays a system-wide root install. An unknown account fails fast naming it.
- **Install failures stay visible.** The launcher waits for the elevated run
  and reports its exit code instead of exiting 0; the last error line lands
  in `%ProgramData%\PiCode Desktop\install.log`; the elevated window pauses
  for Enter on failure rather than closing with the evidence.
