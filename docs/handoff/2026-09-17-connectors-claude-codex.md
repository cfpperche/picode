# 2026-09-17 — feat/connectors-claude-codex: Guest drivers Claude Code + Codex (Phase 1, ADR-0150)
Shipped: Phase 1 of connector parity (ADR-0150): internal/connectors guest drivers claude-code + codex
behind the same /api/mcp (cli=). Claude Code: project .mcp.json edited in place (unknown JSON keys
preserved); user scope only via the vendor binary (claude mcp add/remove --scope user; tolerant list
parse); no toggle in the format → panel hides the switch (driver toggle "none") and Sign in becomes a
copyable vendor command. Codex: TOML ~/.codex/config.toml + .codex/config.toml via surgical splice
preserving comments and foreign tables byte-for-byte (dep github.com/pelletier/go-toml/v2, justified in
the commit), toggle via enabled. Guest refusals: import → "arrive in a later phase"; auth/logout →
vendor command. CONNECTOR_DRIVERS gains claude-code + codex (honest "configured" status).
Verified: make ci-scoped PASS (full) ×3 rounds; go test ./internal/... green; web npm test 802 pass;
visual-review on a scratch instance: PASS after 2 defect fixes (phantom "No" line from the claude mcp
list parse; driver id aligned to the catalog's claude-code) + 1 cosmetic (orphan menu divider);
screenshots var/screenshots/connectors-p1{,b}/ (not committed).
visual-review: PASS
Merge: fast-forward ready

## Next up

- Phase 2 — Omp (.omp/mcp.json) and Antigravity (mcp_config.json, serverUrl) drivers reusing the mcpServers JSON codec.

## Debts

- Vendor command surface (claude mcp add flags, login flow) written from docs unverified against real binaries — divergence surfaces as an explicit error, never silent.
- Guest bearer maps to the standard Authorization header (mapping decision).
- claude mcp list parse is best-effort per-line (topic: docs/handoff/open/connectors-parity.md).
