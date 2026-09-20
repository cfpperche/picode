# 2026-09-15 — grant-row-layout

Verifying the terminal row (the state nobody had seen) found a real defect.

## What was wrong

With a saved grant, `GrantRow` was a centred two-column `set-item`: the hint
list on the left, the tier select (30px) and the domain field (36px, the only
36px control on the page) on the right. The select wrapped above the field,
the field shrank to ~270px and truncated, and the hints shared its band — the
row read as broken.

## Fix, verified

- The row stacks when it carries a grant (`set-item-stack`, the class the page
  already had): hints on their own lines, then the controls on a full-width
  line; the field drops to 30px to match the select and Save beside it.
- Read in three states on a scratch (`var/screenshots/grant-row-fixed.png`,
  `grant-row-states.png`): saved with hints, dirty with an empty field (the
  "No sites yet" line + Save + placeholder), invalid (`example.com, a b` →
  the error in danger colour, danger border, no Save). Both principals render:
  a managed agent and a terminal (`shell CUSTOM`, tier Full).

## Also verified this session

- `desktop-shell/src/permissions.rs` is pure, so its decision table runs on
  any host: compiled standalone and **4/4 tests pass** (site over kind over
  platform default, case/path normalisation, `default` forgetting one entry,
  invalid states refused).
- The terminal grant namespace over the API: `POST {term}` saves under
  `term:<id>`, and the agent beside it keeps its own `read`/default — the two
  never cross.

## Not verified

- The **Ask bar** cannot render in a web scratch: `shellChrome` is a prop of
  the shell's entry, so the listener is never installed there. The live run is
  the owner's: deploy + restart, set Camera to Ask, open a page that calls
  `getUserMedia` in a work-browser tab, answer the bar, reload to see the
  remembered standing. The bar renders between the toolbar and the page.
