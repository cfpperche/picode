### Added

- **The tmux app shows the machine's tmux servers.** A new **Sockets** tab
  lists every tmux server this instance can see — the default socket, every
  named one, and PiCode's own dedicated socket — with what each holds: N
  session(s), how many are PiCode's, and whether anything is listening (a
  socket file with no server is labelled as the leftover it is). The tab
  badge counts the running servers, and the row for PiCode's own socket says
  where new terminals will land.
