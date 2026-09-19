# 2026-09-18 — feat/connectors-marketplace

Owner call: cross-CLI import and PiCode's definition/host import discovery removed; the Connectors pane becomes a Packages-style
**Installed | Marketplace** (PiCode's own connectors stay offered). ADR-0157 (accepted): curated catalog without vault/hosted OAuth —
narrows ADR-0075's refusal to its security core, supersedes ADR-0150's "marketplace rejected". Plan: docs/plans/connectors-marketplace.md.
- Removal: internal/mcp/import.go deleted (serversOfHost → mcp.go); Report.Found/Imports, POST /api/mcp/import + handleMCPImport gone
  (test asserts route absence — SPA fallback answers 503, not 404, without embedded UI); guest empty literals and "arrive in a later
  phase" refusals gone; both Mcps.jsx lost Source tabs / file import / host-pick confirm; integrations.js lost connectorTabs +
  readConnectorDefinition.
- internal/mcpcatalog: Seed() = mcp.Presets() + picode tool presets (featured, source picode); Store syncs registry.modelcontextprotocol.io
  /v0/servers (paginated, cap 120 pages), filters active + described + streamable-HTTP remote (stdio beyond the seed stays hand-curated);
  caches <dataDir>/connectors-catalog.json, refresh at most daily (injected clock, exactly-once), seed offline; GET /api/connectors/gallery?q=
  seed first, 200 cap.
- Both apps: Connectors pane = Installed N | Marketplace subtabs on Packages pkg-* classes (scoped per-app CSS,
  ADR-0072): debounced search, skeletons, capability badges (Remote / Local command / Sign-in required), per-scope
  Added state, Add builds the same POST /api/mcp body, Custom server… secondary, AddConnectorDialog retired; mobile
  mirrors desktop.
Gates: go build, go test (mcp/connectors/mcpcatalog, server minus tmux-guard set), web npm test 819/0, OpenAPI regenerated, ci-scoped PASS
(30 paths); fast-forward ready. visual-review: PASS on scratch (connmkt, embedded UI): Pi blocked one-liner + Open packages, skeletons, cards,
Added flip, zero-hit copy, overlayAudit ok:true, mobile 390px; fixed in-session: name wrap 2 lines (were "PiCode · …" truncation), desc clamp
pinned 2 (flex clipped a 4th) — scoped .connector-market; screenshots var/screenshots/connectors-marketplace/.
## Next up
- Nothing queued — the Phase-5 import item is dead; status-parity (docs/handoff/open/connectors-parity.md) is the only connectors want left.
## Debts
- Curation filter is v0 (active + described + http remote): tune after the first real sync on the owner machine (popularity/pinned); observe after deploy.
