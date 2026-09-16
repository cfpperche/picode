### Added

- **Windows setup finishes on a clean machine.** `picode-desktop install`
  now delivers the Linux binary and the runtime itself: an `install-picode`
  stage (the release matching the tool, verified, placed where provisioning
  resolves it) and an `install-runtime` stage (tmux, git, curl, Node.js from
  the NodeSource repository, pi) between account creation and provisioning.
  Ubuntu only; anything already present is left alone; a distro PiCode did
  not register asks before apt and npm run (`--yes` skips the question);
  the install ends with the shell running in the tray.
