### Added

- **Use: run a CLI on a saved account.** Each guest CLI's Providers pane can
  now put a saved login into that CLI's own login file — the same thing **Use**
  does for Pi — so Claude Code, Codex, Grok, Hermes, OpenCode, Muse or
  Antigravity runs on the account you pick. The account in use is marked on the
  row, and it is read back from the CLI's own file, so a login made in the
  CLI's TUI shows up here too.

### Changed

- The first time PiCode writes a CLI's login file it keeps a copy of what was
  in it (`~/.picode/credfiles/<cli>-<time>.bak`) and tells you where.
- Nothing about a CLI's home changes: no new folder, no `HOME`-style variable,
  so settings, sessions and memory stay exactly where the tool expects them.

### Security

- **Use** is refused while a terminal of that CLI is running: replacing a
  credential under a running agent can corrupt its session. The refusal names
  how many terminals to close.
- A CLI whose login PiCode cannot write faithfully says so and offers no
  control: Omp (its own database), a Grok file with no session yet, and the
  API-key paths of Hermes and Muse.
