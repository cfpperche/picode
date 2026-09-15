### Fixed

- **`make ci` no longer leaves orphan shells on your tmux server.** The test
  suites that start real tmux servers (server, tmux, term) now run them on a
  private namespace that is torn down with the run — after a 2026-09-15 CI
  run left twelve orphan shells behind. The harness also refuses, by
  construction, to end a server outside its own directory.

- **Terminal settings say what is wrong when there is no tmux server to
  read.** With every terminal closed, the terminal settings catalog now
  answers "Start a terminal to read the tmux option list." instead of an
  internal error — a state an isolated test suite exposed on 2026-09-15.
