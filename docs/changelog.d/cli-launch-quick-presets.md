### Added
- **Agent CLIs: quick launch settings.** The launch editor offers verified model, approvals, reasoning/thinking and sandbox controls above the advanced argument fields, with a one-line warning on dangerous picks. Covered: Pi, Claude Code, Codex, Grok, Hermes Agent, OpenCode and Omp.
- **Agent CLIs: one-click YOLO flags.** Codex and Hermes expose their vendors' `--yolo`, Grok its "Auto-approve tools"; on Codex picking YOLO clears the conflicting sandbox/approvals choices (and vice versa), so the launch never composes a combination the CLI refuses.

### Changed
- **Agent CLIs: launch profiles open by default** once a CLI has at least one profile, instead of hiding behind a collapsed section.
