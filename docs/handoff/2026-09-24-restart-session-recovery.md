# 2026-09-24 — feat/restart-session-recovery: restart recovery and compaction layout
Shipped: Restart keeps a live Codex terminal when its announced session has not been saved;
saved Codex rollouts and private Omp agent sessions are recovered for restart.
The sidebar shows Compacting beside the agent name at its default width.
Files: `internal/clisession/codex.go`, `internal/server/{cli_launch,native_session}.go`,
`web/browser/src/components/Sidebar.jsx`, and `web/browser/src/styles/app.css`.
Docs: `docs/architecture/{cli-terminal-launch,terminal-bridge}.md` and changelog fragment.
Verified: focused Go tests and post-merge `make close` passed with Go and JS tests,
web build, and documentation gates; main can fast-forward to branch head `fff8d87a3`.
Scratch `compaction-inline-review` captured Compacting and Working at 1280 and 900 px;
the full Context review name and Codex label remained visible at the fresh 280 px width.
visual-review: PASS (four screenshots, five-question card 5/5, overlay audit `ok: true`).
Not done / debts: no post-fix check against the live service; no deploy in this session.
Merge: fast-forward ready.
