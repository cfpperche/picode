# 2026-09-11 — feat/providers-density: the roster as a table

Shipped on the branch (not merged, not deployed): `#/clis/providers/pi`
renders **one row per account** in six aligned columns (Provider · Account ·
Identity · Usage · 7d spend · actions); geometry in the new shared
`web/shared/styles/providers.css` (imported last by both `index.css`); row
model in `rosterGroups()`/`accountsOf()` (+3 tests). No track is `auto` — an
auto track walked the gauge column to a different x per row. Identity drops
below a 1000 px container, stacked card below 840 px; legend only with rows;
empty roster owns its single Add provider; `Add account` is in the ⋯ menu;
Recently used is chips.

Verified on a scratch instance (1706×1000, 1280×633, mobile 390): rows 45 px,
one column geometry everywhere, largest void in a row 10 px (owner's
screenshot: 847 px), ten providers, no page scroll, quota on one line,
overlay audit ok, every state rendered incl. paused/environment/verdict
(`14-paused-env-verdict.png`). `make ci-scoped` PASS; `make close` run;
screenshots `var/screenshots/01..14`.

Adversarial pass: class/selector audit clean (`quota-empty` was already a
no-op on main), no unused imports, two invented ADR links fixed; QA lesson
kept — the vault follows **HOME** (`$HOME/.picode/accounts.json`), not
`PICODE_DATA`.

Debts (study §Debts): mobile duplicates `Providers.jsx` (ADR-0072, owner's
call); LlamaPanel's `.prov-row` list unexercised (no router); 7d spend empty
in the scratch; a stale (`4m old`) reading never rendered; `Verify with pi`
answers from credential presence — a bogus env key reads green (pre-existing).

Merge: fast-forward ready after `git merge main` (main moved) + `make close`;
owner validates visually first.
