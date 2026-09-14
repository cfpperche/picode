### Added
- **The tmux app: every session on your tmux server, in one screen.** PiCode
  runs every terminal and interactive agent in tmux on your own server, and
  until now a session with no terminal or agent behind it was invisible in the
  product — you had to leave PiCode and run `tmux ls` to find it. Apps → tmux
  now shows the whole server: PiCode's sessions (with **Open**), sessions no
  longer in PiCode's records, and your own tmux sessions beside them, marked as
  not PiCode's and left alone. A row expands to the folder, command, uptime and
  attached clients; the **Server** tab carries the version, socket, client
  count and keyboard mode, plus the terminals whose sessions are gone
  (including the ones the flight recorder saw die in a restart).
- **Removing a leftover asks, and re-reads the session before it acts.** A
  session PiCode cannot attribute can be removed one row at a time, behind a
  confirmation and a receipt (session id, start time and pane process): if the
  session changed since the page was drawn, nothing happens. There is no bulk
  cleanup, because a session missing from PiCode's records may belong to
  another PiCode on the same machine — the row says so instead of guessing, and
  every removal is recorded as an audit event.