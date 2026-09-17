### Changed
- **The handoff board never blocks a session again.** It is a bounded view
  (two next steps per topic, session notes for 7 days, open debts only) and
  warns instead of failing when it is over its target, so `make close` no
  longer stops on how much the team wrote down — and nobody prunes another
  session's bullets to make it fit.
- **Debts carry their state**: `- [x]` marks one paid in the topic file that
  owns it; the board shows the open count and keeps the record.
