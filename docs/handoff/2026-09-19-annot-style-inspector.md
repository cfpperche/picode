# 2026-09-19 — v2c step 5: the style inspector (feat/annot-style-inspector)

The last piece of the annotations target: the card offers the element's own
computed styles with live preview, and the package carries original →
proposed. The settings row was already there (`annotationShots`, honoured by
the capture since the strip's camera toggle landed) — the topic file said
otherwise and is corrected.

## What landed
- Six rows in the card's own disclosure (session-remembered): Text color ·
  Background (`<input type="color">` + a text field, `CSS.supports`
  validated), Opacity (native range), Font · Size · Weight (native select /
  number). Every value initializes from the element; a per-row ↺ returns one
  property to the page's own value.
- Preview = `style.setProperty(prop, v, "important")` on the live element;
  `styles0` (computed at pick) and `inline0` (the element's own inline
  values) are snapshotted so the note keeps the ORIGINAL and every discard
  restores exactly.
- The note gains a `/* proposed */` section in the same css block (Go header
  now says "Styles", since the block carries both); `styleEdits` flows
  page → strip (`stateItems` sanitizes) → POST.
- Chips mark themselves `Aa` while they carry proposals.
- 19 Rust guards (`rustc --edition 2021 --test src/annotate.rs`) + 18 JS
  tests; harness proof that the preview applies live, Cancel/Save/remove/
  clear/exit follow the decision table, and an invalid colour says so
  instead of silently keeping the old one.

## Decision table (draft / saved × exit)
| Item | Action | Page | Proposal |
|---|---|---|---|
| draft | Cancel, trash | restored | gone |
| draft | Save | kept | recorded |
| saved | Edit → Cancel | restored | cleared |
| saved | Edit → Save | kept | recorded |
| saved | × / ⋯ Remove | restored | gone with the note |
| either | Clear, leave mode | restored | gone |
| either | Send | kept | original → proposed in the note |

## Next up
- Owner: deploy + `make desktop-restart`, then one Send with a style change —
  the note should carry both values.

## Debts
- The inspector is page-side DOM: the harness proves it, the WebView2 run is
  the owner's Send.
- `font-family` offers three stacks plus "As on the page"; arbitrary fonts
  are not listed until someone wants them.
