# Docs audit follow-ups

Seeded by feat/docs-refresh (2026-09-14), the first docs-vs-code audit.

## Next

## Debts

- P3 follow-up: add short docs/architecture sections for work-browser internal/browser/hub.go and the Tauri shell Makefile target.
- Unaudited remainder: ADR bodies, api.md/commands.md, 29/33 docs-site guides, macOS/Docker/upgrade guides, external URLs, app-fleet.png freshness.
- [ ] `docs-site/guide/dev-flow.md` (feat/docs-dev-flow): the six phase headings are raw HTML, so the "On this page" outline and heading anchors skip them (the three markdown `##` sections do appear); pay by rendering the phases from markdown headings or a VitePress component, or accept and close this bullet.
- [ ] `docs-site/guide/dev-flow.md` "When the flow bends" (feat/dev-flow-edges) quotes 13 refusal messages printed by scripts; nothing ties a quote to its source, so a refactor leaves the page confidently wrong — pay with a docs-check rule that greps each card's phrase, or accept and close this bullet.
