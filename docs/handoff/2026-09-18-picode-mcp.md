# 2026-09-18 — feat/picode-mcp (N0 of ADR-0154)

PiCode's own tools over MCP for guest CLIs. Landed: `picode mcp <family…>`
(stdio JSON-RPC in the binary, `internal/mcptool`, families `computer` and
`browser`, answers ported from the pi packages with their tests as goldens);
catalog cards "PiCode · Computer/Browser" for every guest CLI (hidden from
pi); launch injection as the guest's per-agent scope (`clilaunch.Config.Tools`,
`internal/server/cli_tools.go`: Claude Code `--mcp-config` merged with the
peer file, Codex `-c`, OpenCode inline config); the "PiCode tools" group in
Launch settings (browser and mobile) and the preview row; guide
`docs-site/guide/picode-mcp.md`, `docs/architecture/picode-mcp.md`.

Proof: `make ci-scoped` PASS; wire probe against a scratch daemon
(identity → daemon's own 403 without a grant; no identity → local answer);
visual-review: PASS (mcp-launch-tools/preview/connectors-dialog/mobile.png,
overlayAudit ok; Grok's absent group proven by DOM, capture above the fold).
uiux-review: PASS (9/9; jargon: none in chrome).

## Next up
- Owner: `claude` in a PiCode terminal with Computer ticked in Launch settings, switch the terminal on in Settings ▸ Computer, ask for a screenshot.
- N1: `inbox` and `checklist` families (`docs/plans/picode-mcp.md`).

## Debts
- Codex `-c mcp_servers.*` overrides and MCP image blocks in Codex/OpenCode/Grok are unverified live (fallback: the PNG under `<data>/var/captures/`).
- `TestCLIRestartPreparationFailureAndWorkspaceCleanup` is flaky on main too (1 in 4 runs: `PanePID` races the fresh pane); not touched here.
