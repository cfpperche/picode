# 2026-09-24 — feat/mission-dev-flow-drift: dev-flow refusal quotes checked against their sources

Mission pilot 2/5 (mission_N7K4X3IA26T6AQC66EEW27LWSO): Codex inventoried the 13 cards read-only and stopped; Claude Code (generation 2) built it in the same worktree.

Shipped:
- `scripts/docs-quotes.mjs`: BENDS table, one row per "When the flow bends" card in
  docs-site/guide/dev-flow.md; each check = [phrase in card, source file, source pattern].
  Rules: every card has a row and every row a card; each phrase is in the card and its source
  (backticks stripped, whitespace collapsed; a RegExp where text is built at runtime, e.g. shell
  refuse args or a Go %d); every `<code>` quote in the "says" line is covered by a checked phrase,
  a `<placeholder>`/…, or a declared example. Runs inside livingDocsFailures (docs-living.mjs), so
  docs-check, ci-scoped (metadata) and make close run it; docs-living.test.mjs passes `{ quotes: false }`.
- Sources: land.mjs (cards 1–3), internal/install/readiness.go (deploy), handoff-board.mjs (invisible
  topic, over target, 7-day window), worktree-status.mjs (stalled), .githooks/pre-commit (living doc,
  note on main), worktree-gc.sh (keep), close.sh (over-target still passes close).
- Guide intro states the exact guarantee (script-printed wording matched; examples, Vale/VitePress, prose not); docs-audit.md debt [x];
  fragment docs/changelog.d/mission-dev-flow-drift.md.

Not checked exactly: example values ("codex" working, .worktrees/x, 1/N counts); Vale card checked
against styles/PiCode/Spelling.yml, dead-link card against VitePress config, and "Nothing prints" prose unchecked.

Verified: docs-quotes.test.mjs 10/10 (real tree + 8 drift cases). Live: "merged already"→"was" in
the guide made `node scripts/docs-check.mjs` exit 1 at dev-flow.md:134; restored. ci-scoped PASS
(fmt, vet, hooks, test-js, docs); make close PASS; landed at `da5e10929`, full `make ci` PASS.
Mission accepted at v19 after Claude stopped; worktree cleaned. Codex's first report omitted `generation`, got a generic forbidden error, then succeeded on retry.
