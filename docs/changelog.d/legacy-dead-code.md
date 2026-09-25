### Removed

- **Leftover compatibility code.** PiCode no longer rechecks
  `~/.claude/settings.json` for hooks an early build wrote there (the file is
  clean), no longer reconciles Inbox reply tasks in a format from before
  ADR-0060, and `#/clis/<cli>/launch` is no longer rewritten to
  `#/clis/<cli>`.
- **More old compatibility paths.** The `/api/apps/inbox/{view,action}`
  aliases are gone (use `/api/inbox/*`); a tmux session without PiCode's
  instance stamp is no longer judged by its port; the work browser no longer
  carries the hide-and-freeze path for desktop shells from before live
  overlays; and Pi's package install, update and remove now answer with the
  same report `GET /api/packages/report?cli=pi` returns.
