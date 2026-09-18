### Fixed

- **desktop: deploys and shell swaps no longer end every terminal.** The
  WSL distro's keepalive moved out of the shell process into a scheduled
  task (`PiCodeDistro`) that outlives it (ADR-0155): a `make
  desktop-restart`, a shell crash or any taskkill can no longer leave the
  distro unowned for WSL's idle reclaim — the failure that killed every
  tmux session, agent and terminal at once (2026-09-18, four times). The
  task's action wraps wsl.exe in a headless conhost, so the keepalive
  holds the distro with no console window on the desktop. The shell
  ensures the task on its health poll (child-spawn kept as fallback), and
  `desktop-swap.sh` ensures it before killing the old resident. Also
  fixed: `make adr` failed whenever a registered worktree's directory was
  pruned.
