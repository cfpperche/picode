### Added

- **Steer or follow up while a CLI is working.** The message bar under an
  agent's terminal (desktop and the phone's sheet) no longer shuts while the
  CLI is busy: it offers **Steer** (reach the running turn) or **Follow-up**
  (wait for the turn to end), in whichever of the two the CLI supports. Pi,
  Omp, Hermes, Muse Code and Codex offer both; Claude Code and OpenCode offer
  Steer; Antigravity and Grok offer Follow-up. A CLI asking for your answer, or
  one with a half-typed draft, still refuses. The receipt says **Queued** when
  the message shows up on the screen, and **Unconfirmed** when PiCode cannot
  see it yet.

### Changed

- Searchable pickers across the desktop app mark the current value with a bar
  instead of a tint, so the row the keyboard is on stays easy to tell apart; a
  picker without a search field takes keyboard focus as soon as it opens.
