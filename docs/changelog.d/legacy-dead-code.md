### Removed

- **Leftover compatibility code.** PiCode no longer rechecks
  `~/.claude/settings.json` for hooks an early build wrote there (the file is
  clean), no longer reconciles Inbox reply tasks in a format from before
  ADR-0060, and `#/clis/<cli>/launch` is no longer rewritten to
  `#/clis/<cli>`.
