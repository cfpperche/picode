# 2026-09-23 — feat/agent-compaction-status: native compaction activity (ADR-0209)
Implemented: terminal state `compacting` and sidebar activity for native lifecycle signals from
Codex, Claude Code, Grok, Pi, Omp, and OpenCode. Other CLIs keep the Working status.
The state feeds delivery guards, activity summaries, browser rows, canvas, and mobile chips.
Files: `internal/server/term_{state,wiring,intercept}.go`, native integration/observation,
`web/shared/domain/{agentStatus,terminalCli}.js`, browser and mobile UI/CSS.
Docs: ADR-0209, `docs/architecture/terminal-bridge.md`, changelog fragment.
Verified: focused Go and JS tests, `make web`, and `GO_TEST_SHARDS=1 make ci-scoped` passed.
The default parallel scoped run failed on the unrelated private-tmux
`TestCLIRestartPreparationFailureAndWorkspaceCleanup`; that test passed alone.
Scratch browser showed Compacting and Ready in the sidebar; overlay audit passed.
visual-review: PASS (scratch screenshot and five-question card).
Not done / debts: native compaction was not exercised in authenticated CLI sessions;
OpenCode and Omp signals were verified against installed packages and hook contracts.
Merge: `main` was merged after it advanced during closing; rerun `make close`
before fast-forward landing.
