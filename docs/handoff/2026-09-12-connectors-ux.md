# 2026-09-12 — feat/connectors-ux: Connectors pane refinement plan

Shipped: docs only — `docs/plans/connectors-ux.md` (phases P0–P3, pane and Add
dialog spec, row anatomy, decision table, verification, out of scope, three
owner questions) plus the study `docs/benchmarks/2026-09-12-connectors-ux.md`
(receipts: VS Code / Cursor / Claude Code docs read live 2026-09-12, in-repo
bars and the measured problem table). No code changed.

Verified: `make ci-scoped` PASS (metadata scope). Current pane measured on a
`qa-scratch.sh` instance (`cxux`, 1600×1000, 942 CSS px pane): add section 476
of 612 px (78 %), native file input 18 px tall, scope pills 28 px vs row
switches 36 px, `settings-ctx` still the first child 0 px under the tab rule,
row target truncated at 193 px. Captures read:
`var/screenshots/cxux/01-empty.png`, `02-configured.png`.

visual-review: n/a (no UI shipped; the two captures record the current pane)

Not done / debts: implementation not started. P0 alone answers the screenshot
(the glued line, the two unlabelled pill rows, the 18 px file control, the
control heights); P1 (one Add dialog) is the structural change and waits on the
owner's answers to the three questions at the end of the plan. Also recorded
there: the non-embedded "MCPs" branch is unreachable dead code in both apps.
`docs/handoff.md` is at 8159/8192 B, so no Next-up pointer fitted; this note
carries it.

Merge: fast-forward ready.
