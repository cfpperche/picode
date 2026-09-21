# 2026-09-21 — feat/keyboard-row: the Keyboard pane's rows and toolbar, reworked

Shipped: the row is four cells placed explicitly — `label (9-18rem) | keycaps |
note (flexible) | actions` — replacing the three-column grid the first build
shipped, where the base `.key-label`'s `flex: 0 1 12rem` (a *width* basis where
the label sat in a row) became a **height** inside the new `.key-name` column:
the label, its cell and every row measured 192-216px. Auto-flow compounded it —
a row with no note put its actions in the flexible column, two notes pushed them
onto the next line (32 → 229px). The pane resets that basis and places every
cell by `grid-column`. The toolbar's policy is now explicit: it **wraps rather
than clips** — the filter owns a width its placeholder fits in, the facets
shrink and scroll instead of squeezing, the row count steps aside below 1140px
(its `All N` chip carries it), and the placeholder is the pane's short
"Filter keys" (the aria-label keeps the full description) because a 1024 window
leaves the field only ~160px.
Measured: desktop rows 32-90px and 3 199px for all 90 (was ~19 500px), 12px
from a label to its first keycap; phone 44-106px / 5 058px; toolbar 54px, one
line at 1365 and 1024, three at 390; `__picodeOverlayAudit()` ok; no page
overflow at 1024 or 390.
Verified: `make ci-scoped` PASS; `scripts/qa-cli-settings.mjs` — 6 keyboard rows
pass on both apps, including four **new density assertions**: no label cell over
one line, the action and its keycap on one line, the keycaps within 48px of the
label, and per-app row/total ceilings. Those did not exist when the broken build
shipped: the harness measured the keycap (24px, correct) and never a row, and
three visual passes described horizontal alignment while none judged vertical
space or a narrowed toolbar. That is the process failure this branch pays for;
the assertions are the gate now.
visual-review: **PASS** (card yes/yes/yes/no/yes). Three refusals on the way —
200px rows, a `nowrap` that squeezed the facet labels, a placeholder cut at
1024px — each fixed and re-read; the two minors left are the scrolling tab strip
clipping its leading tab (documented `2026-09-06-tab-strip-overflow.md`) and
phone rows whose chords wrap costing ~85px.
Not done: `qa-cli-settings.mjs` still fails on two rows **that fail on `main`
without this branch** (proved by stashing): the post-loop untrusted/trust matrix
and the mobile "initial failure retries" row, both named in
`docs/handoff/open/agent-clis-native.md`.
Merge: fast-forward ready.

## Next up

- P1 of `docs/plans/keyboard-pane.md`: one report envelope for every CLI (`cliKeys.js` + the `/api/pi-keys` wrap), then P2's `internal/clikeys` with Omp first.
