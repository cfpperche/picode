### Added
- **Antigravity sessions.** The Antigravity pane has a **Sessions** tab like
  Muse Code and the adapter CLIs: every conversation Google's CLI keeps on this
  machine, with its folder, age, size and turn count, grouped by folder and
  filtered to the workspace you are in. **Open in terminal** resumes that
  conversation (`agy --conversation <id>`) in its own folder. Read-only.
- Sessions for a CLI whose history exists before it has an adapter: the pane
  shows the tab whenever the server reports a session source, instead of only
  for CLIs with activity reporting.

### Fixed
- The server test that launches a terminal for the change feed no longer leaves
  the tmux session behind: cleanup ran with `t.Context()`, which is canceled
  before cleanups start, so every tmux call it made failed and each test run
  left one session in the shared tmux server.
