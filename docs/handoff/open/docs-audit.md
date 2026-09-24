# Docs audit follow-ups

Seeded by feat/docs-refresh (2026-09-14), the first docs-vs-code audit.

## Next

## Debts

- P3 follow-up: add short docs/architecture sections for work-browser internal/browser/hub.go and the Tauri shell Makefile target.
- Unaudited remainder: ADR bodies, api.md/commands.md, 29/33 docs-site guides, macOS/Docker/upgrade guides, external URLs, app-fleet.png freshness.
- [x] `docs-site/guide/dev-flow.md` (feat/docs-dev-flow): the six phase headings are raw HTML, so the "On this page" outline and heading anchors skip them (the three markdown `##` sections do appear); pay by rendering the phases from markdown headings or a VitePress component, or accept and close this bullet. (Paid 2026-09-24, feat/mission-dev-flow-outline: the outline is built from the rendered DOM, so the raw `h2`s got ids and theme permalink anchors, and the owner pill got VitePress's `ignore-header` class to stay out of outline titles.)
- [x] `docs-site/guide/dev-flow.md` "When the flow bends" (feat/dev-flow-edges) quotes 13 refusal messages printed by scripts; nothing ties a quote to its source, so a refactor leaves the page confidently wrong — pay with a docs-check rule that greps each card's phrase, or accept and close this bullet. (Paid 2026-09-24, feat/mission-dev-flow-drift: `scripts/docs-quotes.mjs` maps each of the 13 cards to its source and runs inside the living-docs pass, so `docs-check`, `ci-scoped` and `make close` fail on drift with the card's line. Not checked exactly: runtime example values (`terminal "codex" is working`, `.worktrees/x`, the `1` and `N` counts), and two cards whose quote comes from outside the repo, which are checked against their configuration instead: Vale's message comes from `styles/PiCode/Spelling.yml`, and the dead-link card from `ignoreDeadLinks` in the VitePress config.)
