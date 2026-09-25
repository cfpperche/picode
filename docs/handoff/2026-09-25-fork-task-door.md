# 2026-09-25 — fork-task-door: fork tasks through the prompt door

Paid agent-fork debts 5 and 6 for Claude Code, Codex, Grok and OpenCode: their fork
recipe (`ForkArgs`) now carries no task, and `deliverForkTask` sends it through the
verified prompt door (`doorReaderCLI`) once the TUI is at its prompt — line breaks
kept, no 8192 limit, and a Restart before the copy is pinned re-runs the recipe without
the task. Omp stays on the launch argument (no door reader; a blind paste into an
opening TUI could be lost) — new debt in `docs/handoff/open/agent-fork.md`.
Verified: fork tests updated (Claude/Codex: no task in argv, task pending; Omp: task
still in argv; the 8192 refusal now tested on Omp). Not run live: a real Claude/Codex
fork on a scratch needs their credentials and trust copied; the door itself is the
attach bar's, measured live on these CLIs (ADR-0206). The owner's live check is owed.
