### Removed

- **Compatibility for terminals from before the per-instance tmux socket and
  before Pi agents got their own terminal.** PiCode no longer looks for
  sessions on your default tmux server or for old `picode-<agent id>` Pi
  panes, and old tabs pointing at them no longer reconnect. Every terminal
  lives on PiCode's own socket.

### Fixed

- **SSH guide:** the attach commands now name PiCode's socket
  (`tmux -S ~/.picode/tmux.sock …`); a plain `tmux ls` asks a server PiCode
  does not use.
