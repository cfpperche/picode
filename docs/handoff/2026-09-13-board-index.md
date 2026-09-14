# 2026-09-13 — board-index: the handoff board becomes an index (ADR-0131)

Shipped: `## Next up` keeps its bullets inline; **debts are now one line per
topic** — count, topic file, and the topic's `Plan:` — with the bullets
staying in `docs/handoff/open/<topic>.md`. `make handoff` also **fails,
naming the file**, when a topic file lists bullets but none under a
recognized heading (the defect that hid six snippet debts). The two caps that
contradicted each other (byte cap bound ~20 lines before the line cap) are no
longer a wall: the board went 12 232 → ~6 300 characters, 97 → 68 lines.
Reasoning and rejected options: `docs/decisions/0131-handoff-board-index.md`;
measurement kept in `docs/plans/handoff-board.md`. Also updated:
`docs/handoff/open/README.md` and `.pi/skills/handoff-update/SKILL.md` (both
said the old shape).

Unrelated defect fixed on the way: `docs/decisions/README.md` had **committed
conflict markers** on `main` (`<<<<<<< HEAD` around the 0129/0130 rows, from
the `feat/pi-custom-providers` merge). Removed; both rows kept, in order.

Verified: `node --test scripts/handoff-board.test.mjs` 8/8 (the invisible-file
rule, the per-topic ledger with its plan, and the budget test re-pointed at
next-up bullets, which are what still take lines); `make handoff` writes the
board under both caps; `make ci-scoped` PASS.
Merge: fast-forward ready.