# 2026-09-14 — feat/dialog-clarity

The owner's read of the custom endpoint dialog: no labels, no rhythm, prose
where a state belonged, and limits refused because a gateway's models
disagreed.

**Rhythm.** Every field is one unit — label above (12px), control (36px), one
line of help under it (11.5px), 8px apart, 24px between units, adapted from
shadcn's Field scale and named as such in the CSS section comment. Helper
prose that was a lecture (a file path, a protocol note) is gone. The closed
dialog fits a 1080p window; at 900px it overflows 34–36px with nothing hidden
(the sticky footer keeps the action) and the **Advanced** summary stays above
the fold at 629px vs the footer's 674px.

**Limits per model.** Context window and max output are now one row per model
id (Advanced → Model limits): Load models fills each model's own numbers
(1M/384k, 200k/131k, 262k/65k from the fake gateway — three different pairs,
which is exactly what the old shape refused). A hand-typed number survives a
load; a row follows its id. The old shape wrote one value per list, so
`modelLoad`'s "every model must agree" branch is gone; the decision table
lives in `docs/architecture/cli-providers.md`, each row pinned by a test.

**Sections and states.** Advanced holds three groups with legends and
hairlines (Request compatibility, Thinking, Model limits). Load/Verify report
in the Model ids unit — a 401 reads "The endpoint refused the key (401): …
Check the API key and try again." in red, so the alternative (Advanced) is
never pushed below the fold.

visual-card: 1 yes · 2 yes · 3 yes · 4 no · 5 yes (5/5)

Verified: `make close` green (fmt, vet, hooks, go[7], test-js, build) and six
read captures — desktop default, Advanced + per-model fill, mobile 390 (rows
stack id-over-numbers), dark at 1000×560, and a blocked 401. Overlay audit ok
in all; screenshots in `var/screenshots/clar-*.png` (never committed).

## Debts

(kept in `open/providers-custom.md` — duplicate pruned 2026-09-15 to hold the board budget)
