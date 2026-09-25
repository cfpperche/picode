### Added

- **Automations: a start run on Claude Code, Codex, Grok, Hermes, OpenCode or
  Omp now reports the CLI's answer.** When the run ends, the Inbox result
  and the Notify message carry the CLI's last reply, as they already did for
  Pi, instead of only pointing at the session.

### Fixed

- **Grok, Hermes and OpenCode runs find their own conversation.** Even when
  the CLI's hook names no session file, PiCode looks where the CLI keeps it,
  so the run's cost, its answer and the agent's Outcomes row are not left
  empty.
