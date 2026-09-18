# 2026-09-18 — feat/connectors-opencode-grok: Guest drivers OpenCode + Grok (Phase 3, ADR-0150)
Shipped: Phase 3 of connector parity (ADR-0150): guest drivers opencode + grok behind /api/mcp (cli=).
OpenCode: mcp block of user opencode.json ([XDG] path) + project opencode.json; command array round-trip,
environment↔env, enabled toggle, $schema/provider preserved; never touches PiCode's own session config
(OPENCODE_CONFIG). Grok: user ~/.grok/config.toml + project .grok/config.toml; TOML splice extracted to
internal/connectors/toml.go (shared with codex); comments and ${VAR}/${VAR:-default} kept byte-exact;
toggle "none" refuses with "remove and re-add instead". AuthHint per driver: opencode copyable instruction,
grok plain text "sign in on first use". Driver ids = catalog ids; TestDriverIDsMatchCatalog now covers all
6 drivers. Zero new deps, zero JSX/CSS (pane is data-driven; grok reuses the claude path, opencode the omp
path already visually verified in phases 1–2).
Incident (transparency): an earlier test-fixture version resolved the real HOME and rewrote
/home/goat/.config/opencode/opencode.json to {} (previous content lost; owner's real opencode.jsonc
untouched; ~/.grok untouched). Root cause fixed (XDG resolved inside the temp dir); whole suite
revalidated leak-free via mtime scan.
Verified: make ci-scoped PASS post-merge of main; go test ./internal/... green; web npm test 0 fail;
golden byte-exact fixtures for grok TOML and opencode JSON.
Merge: fast-forward ready

## Next up

- Phase 4 — Muse (settings.json mcp_servers, mode required|optional) and Hermes (YAML config.yaml), closing all 9 CLIs.

## Debts

- OpenCode/Grok file formats written from vendor spec/docs unverified against real opencode/grok binaries (topic: docs/handoff/open/connectors-parity.md).
