# 2026-09-14 — feat/endpoint-polish

Three small items from the custom-endpoint topic: the sticky action footer, the
Verify copy, and two wrong debts in the topic file.

**Sticky actions.** The create dialog scrolls its whole body, so a tall form
(custom endpoint: two buttons, an optional JSON editor, the Advanced block) put
**Add endpoint** below the fold — 152px of scrolling at a 520px viewport. The
row now sticks to the dialog's bottom edge, and the dialog carries no bottom
padding of its own: the footer owns that edge, so nothing scrolls through a
strip below it. A pseudo-element would have been simpler but added 16px of
phantom scroll to a dialog that already fit (measured), so the padding moved
instead. Same shape in the phone sheet, safe-area included. Verified at
1000×520 (scrolled mid-form) and 390×600: footer flush (gap 0–1px), reachable
without scrolling, content scrolling above the separator, no bleed below,
`__picodeOverlayAudit()` ok.

**Verify copy.** `probeFailureReply` printed `(0)` for an input failure with no
HTTP status (an unsupported `api` value in a hand-edited file), and the
follow-on sentence about API types also appeared for a missing model. The
guidance is now a `Hint` field on the classified error — set only by the
failure it answers — and the message names the four shapes Verify can speak.
The empty-model case keeps its own line.

**Topic truth.** Two bullets were wrong: the probe edge ("nothing verifies a
gateway without a model id" — the schema refuses a definition with no model id,
so a saved endpoint always has one), and the debt list now says what the probe
can and cannot speak. `sticky actions owed` is paid and gone.

visual-card: 1 yes · 2 yes · 3 yes · 4 no · 5 yes (5/5)

## Next up

- Per-model context/max/levels (the form still writes one value per list).

## Debts

(kept in `open/providers-custom.md` — duplicate pruned 2026-09-15 to hold the board budget)
