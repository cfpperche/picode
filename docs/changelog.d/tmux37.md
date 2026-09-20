### Changed

- **The terminal settings report the tmux that is actually serving your
  terminals.** After a tmux upgrade the new program can talk to the older
  running server; the version shown on the System page (desktop and mobile)
  and in the tmux app's server view is now the server's, not the new
  program's.

### Fixed

- The tmux app's server view no longer shows an empty version and keyboard
  mode while a drained session is answered by the older server.
