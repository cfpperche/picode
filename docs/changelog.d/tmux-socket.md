### Added

- **PiCode terminals get their own tmux server.** Each instance runs tmux on
  a socket inside its data directory (`~/.picode/tmux.sock` for the main
  install), separate from your personal tmux: a `kill-server` on one side can
  no longer take the other down, and scratch/QA instances stop creating their
  sessions in your tmux. Attach from your own shell with
  `tmux -S ~/.picode/tmux.sock attach -t picode-…`; inside a PiCode terminal
  everything works as before.
- Terminals created before the change keep running: the daemon follows both
  servers while they drain, and the terminal list, status and tmux app show
  one fleet. They move to the new server when they are restarted or recreated.

### Changed

- Scratch instances (QA scratch, docs fixture) and their cleanup operate on
  their own tmux server, so their sessions no longer appear in — or collide
  with — your tmux.
