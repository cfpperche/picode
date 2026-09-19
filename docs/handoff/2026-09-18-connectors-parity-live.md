# 2026-09-18 — feat/connectors-parity-live: all eight guest drivers verified against real vendor CLIs

Shipped: ADR-0150 connector queue executed — every guest driver in
internal/connectors verified against the real vendor binaries; divergences
became tested fixes, untestables went to docs/handoff/open/connectors-parity.md.
Live harness live_test.go gated by PICODE_CONNECTORS_LIVE=1 +
PICODE_LIVE_SANDBOX=process HOME (mtime-scan clean). Round trips ran via each
vendor's own CLI (agy/opencode/grok/hermes list; codex get/list; claude
list/get/remove; muse login); omp proven to load both driver files and honor
enabled (marker-file probe). Five tested fixes: claude add argv (name first,
variadic --env/--header, `--` before stdio cmd, health tails stripped); codex
workspace + agy project layers removed (vendors read one user config file);
grok toggle = entry enabled + disabled_mcp_servers overlay; opencode store is
opencode.jsonc (jsonc wins, writes target the owning file; muse schema_version
kept). Updated mcp_test.go phases 1–3 and docs/architecture/mcp.md. Gates:
ci-scoped PASS; web npm 0 fail; offline go green; live 8/8 PASS; ff-ready.

## Next up
- Phase 5 (queued): definition import between CLIs (no secret copying) +
  status parity from real signals (opencode/hermes mcp list, claude health tails).

## Debts
- docs/handoff/open/connectors-parity.md: OAuth completion per CLI, AGY
  session-shape re-verify, Muse enabled/headers at load, live-status ceiling,
  TUI sign-in hints.
