### Added

- **Fix instruction files from the Instructions tab.** A `CLAUDE.md` that only
  tells Claude Code to read `AGENTS.md` gets a **Review change** button that
  adds an `@AGENTS.md` line, and **Add personal file** creates
  `CLAUDE.local.md` or `AGENTS.override.md` and adds it to `.gitignore`. Every
  change is shown as a diff first, is written only when you confirm it, and is
  never committed.
