# 2026-09-14 — muse-agy-detect: detect-only Muse Code and Antigravity

Shipped: Agent CLIs lists Meta's `muse` and Google's `agy` as detect-only catalog rows — installed or not, Check setup, and (Muse only) Check for updates against its release channel. Launch, Activity reporting, New terminal, messages and the rest of the CLI panes stay off. A catalog row can exist without wiring lifecycle.
Files: `internal/clilaunch/config.go`, `internal/clilifecycle/lifecycle.go`, `internal/server/cli_{launch,lifecycle}.go`, `internal/store/cli_launch.go`, `web/shared/domain/{cliLaunch,terminalCli}.js`, `web/{browser,mobile}/src/components/AgentClis.jsx`, `docs/architecture/cli-terminal-launch.md`, `docs-site/guide/agent-clis.md`.
Verified: `make close` green on this tree (GO+WEB+DOCS).
visual-review: PASS (var/screenshots/{muse-detect,muse-mobile,agy-detect,muse-missing}.png read; overlayAudit ok; card 5/5)
Not done / debts: none from this branch.
Merge: fast-forward ready (`git merge --ff-only feat/muse-agy-detect && make ci`).
