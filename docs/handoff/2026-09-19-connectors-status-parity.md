# 2026-09-19 — feat/connectors-status-parity: guest MCP rows show the vendor's own status

Shipped: Status parity (ADR-0150 decision 4): three guest drivers surface the vendor's own
headless status on every pane load. claude-code reuses the `claude mcp list` health tails it
already parses (✓→Live, ✗/✘→Failed, ⏸ pending = no claim, zero extra vendor calls); opencode
parses `opencode mcp list` glyphs (● live, ✗ failed, "disabled" = off, ANSI stripped); hermes
parses its status column (✓ live, ✗ failed, "✗ disabled" = off, not health). Probes: 15s budget +
60s per-driver cache, hung/missing CLI skipped — rows stay honestly "configured";
Codex/Grok/AGY/Muse/Omp keep that ceiling (grok `mcp doctor` a possible future). Live-capable
guests (claude-code, opencode, hermes) show chips independent of a running Pi agent; Pi's
adapter stream and the stopped-agent line unchanged.

Verified: opencode/hermes fixture scripts replay the measured vendor outputs (octal escapes —
dash printf lacks \x); cache-hit via a call counter; claude health-line test extended with Live
assertions; web tests cover the three status flips — all green (one llama timing flake passes
alone). Visual review on scratch with REAL vendors: seeded `claude mcp add dead
https://example.invalid/mcp --scope user` → red "Failed" chip + `claude mcp login dead` Copy
hint, no agent running (var/screenshots/status-parity/). Gates: go build + connectors/server
tests green (tmux-guard set skipped), web 64+420 pass 0 fail, make web ok, ci-scoped PASS (14
paths). QA gotchas: the scratch daemon inherits the launcher's PATH (~/.picode/bin intercept
wrappers come FIRST — probes hit wrappers, not vendor binaries — start scratches with it
stripped); `~` in scratch recipes resolves to the ROOT checkout's var/, not the worktree's.

## Debts
- None new; the "live status ceiling" parity debt was PAID in docs/handoff/open/connectors-parity.md.
