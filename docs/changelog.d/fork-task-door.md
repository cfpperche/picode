### Fixed
- **A fork's task keeps its layout and is sent once.** Forks of Claude Code, Codex, Grok and OpenCode agents now get their task after they open, the way Pi and Muse Code forks do: line breaks are kept, long tasks are fine, and restarting the fork before it answers no longer sends the task again. Omp forks still get the task on one line.
