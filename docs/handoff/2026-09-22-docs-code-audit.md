# 2026-09-22 — docs-code-audit: adversarial docs×code audit; stale connector-import docs fixed
Shipped: docs-only. Cross-checked every mechanically verifiable claim class
against the code — registered HTTP routes (417) vs `docs/architecture/*.md`,
`#/` hash routes vs the router, ADR index vs files (191=191), OpenAPI spec
(regenerated, in sync), `picode`/`picode-desktop` subcommands vs `usage()`/main,
the nine-CLI catalog, Makefile targets vs AGENTS/CONTRIBUTING/README, Go/Node
version claims. Fixed what failed: the removed definition-import flow
(`POST /api/mcp/import` gone, ADR-0157) was still taught in
`docs-site/guide/integrations.md`, `mcp-deepwiki.md`, `mcp-gmail.md`,
`connectors/README.md` and both `packages/pi-connector-*/README.md` (also the
stale "Integrations → Connectors" location); Getting started's sidebar list
omitted the **Apps** tab (five in code); `docs/architecture/cli-providers.md`
used wrong route placeholders; AGENTS.md repo map lacked `desktop-shell/`,
`connectors/`, `packages/`, `cmd/picode-desktop/`.
Verified: `make ci-scoped` PASS (full); `node scripts/docs-check.mjs` ok;
OpenAPI diff empty; grep shows no import-flow residue outside immutable
ADRs/handoff/plans. Method blind spot: prose-vs-UI claims were sampled, not
exhaustively verified — only route/command/version/label claims were checked
mechanically.
visual-review: n/a (no UI change)
Not done / debts: none new.
Merge: fast-forward ready.

## Next up

- Run the same mechanical claim-audit over `docs/architecture/*.md` prose paragraphs (session-local follow-up; this pass covered the checkable classes).
