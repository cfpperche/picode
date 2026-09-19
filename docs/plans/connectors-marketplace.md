# Connectors marketplace — Packages-style pane + curated catalog

Status: **in progress** (branch `feat/connectors-marketplace`, 2026-09-18).
Decisions: [ADR-0157](../decisions/0157-curated-connector-catalog.md) — sync
of the official MCP Registry, filtered; one catalog for all CLIs; import and
host-config discovery removed. Benchmark: the Packages tab's
Installed | Marketplace pattern (`web/browser/src/components/Packages.jsx`),
adapted — a connector card writes the target CLI's native config through its
ADR-0150 driver instead of installing a package.

## Phases

| Phase | Deliverable | Files |
|---|---|---|
| **P1 — remove import** | `POST /api/mcp/import` gone, `internal/mcp/import.go` deleted, `Report.Found`/`Report.Imports` gone, guest "arrive in a later phase" refusals gone, dialog source tabs + "Import a file…" gone from both `Mcps.jsx`, tests updated | `internal/mcp/`, `internal/server/mcp*.go`, `web/*/src/components/Mcps.jsx`, `web/shared/domain/integrations.*` |
| **P2 — catalog + endpoint** | `internal/mcpcatalog`: seed (presets + picode tools) + registry sync (filtered, cached in the data dir, background refresh, degrades to seed offline); `GET /api/connectors/gallery?q=&cli=` | `internal/mcpcatalog/`, `internal/server/` |
| **P3 — Marketplace subtab** | Installed \| Marketplace subtabs in both apps, cards with capability badges, Save-to control, PiCode connectors pinned, Custom server secondary; dialog retired | `web/*/src/components/Mcps.jsx`, `styles/`, `integrations.js` |
| **P4 — docs + gates** | `docs/architecture/mcp.md`, `docs-site` guide, changelog fragment, visual review on a scratch instance | docs, var/screenshots |

## Decision table

| Conditions | Action | Verified by |
|---|---|---|
| Marketplace tab, cache empty (first run / offline) | Seed cards answer immediately; registry rows appear after the background refresh | unit: stale/empty cache → seed |
| Search query | Seed hits first, then registry matches; empty query shows featured | unit + browser |
| Card Add → scope user | Existing driver `Add` runs; row appears in Installed | server tests (existing paths) |
| Card Add → guest CLI | Same driver path; capability badge from the driver | server tests |
| Card already in the selected scope | Action reads *Added*, disabled | browser |
| Import endpoints/pieces called | 404/removed; guests no longer carry refusal strings | server tests |
| Registry unreachable | Serve cache or seed; no pane error | unit |
| Agent scope selected for guests | Existing guestScope refusal (unchanged) | existing tests |

## Verification

1. `make ci-scoped` on the branch; `make close`; `make land`.
2. Live harness unaffected (`internal/connectors/live_test.go` skips by default).
3. Visual review on a scratch instance: Installed tab, Marketplace (seed +
   synced), Added state, mobile 390 px; `window.__picodeOverlayAudit()` ok.
