# 2026-09-18 — feat/connectors-omp-agy: Guest drivers Omp + Antigravity (Phase 2, ADR-0150)
Shipped: Phase 2 of connector parity (ADR-0150): guest drivers omp + agy behind /api/mcp (cli=).
Omp: user ~/.omp/agent/mcp.json + project .omp/mcp.json, toggle via enabled, $schema/disabledServers
preserved. AGY: user ~/.gemini/config/mcp_config.json + project .agents/mcp_config.json, url→serverUrl
on write, toggle "none" (panel hides the switch), authProviderType/oauth/disabledTools preserved;
legacy url-only entry is read-only with a clear refusal. Driver ids = catalog ids (omp/agy,
clilaunch/config.go); TestDriverIDsMatchCatalog guards the claude-code misalignment from recurring.
AuthHint per driver: omp copyable instruction (/mcp reauth in the TUI), agy plain text; import/auth
refusals extended. Zero new deps (stdlib JSON).
Verified: make ci-scoped PASS; go test ./internal/... green; web npm test 1645 pass; visual-review on
a scratch instance: omp PASS first pass, agy FAIL (unreadable hint destination + misaligned row,
audit ok:false) → fixed (hint on its own line mcp-login-note, mcp-login 36px, has-note wrap) →
re-verified PASS (overlayAudit ok:true); pi regression + orphan divider rechecked ok.
Screenshots var/screenshots/connectors-p2{,b}/ (not committed).
visual-review: PASS
Merge: fast-forward ready

## Next up

- Phase 3 — OpenCode (mcp block of opencode.json) and Grok (TOML .grok/config.toml with ${VAR} verbatim), reusing the Phase 1 codecs.

## Debts

- Omp/AGY file formats written from vendor docs unverified against real binaries (topic: docs/handoff/open/connectors-parity.md).
- Catalog performance (driver id vs catalog) covered only for the 4 current drivers.
