# 2026-09-21 — cli-tabs-scrollbar: no vertical scrollbar on the CLI pane tabs
Shipped: `overflow-y: hidden` on `.cli-pane-tabs` in both apps' `agent-clis.css`. `overflow-x: auto` computes the other axis to auto, so the strip's sub-pixel vertical overflow (tabs' -1px margin + 2px underline under fractional Windows DPI; measured scrollH 36 vs clientH 35) drew a vertical scrollbar beside the last tab. Sideways scroll and the scrollIntoView deep-link behavior are unchanged.
Verified: qa-scratch instance — computed styles + screenshots read at wide (no v-scrollbar, underline whole), 720px (h-overflow, Connectors click scrolls the strip, scrollLeft 99, no v-scrollbar), and the mobile shell at 390px; overlay audit ok; ci-scoped PASS. Blind spot: not re-checked at the user's exact Windows DPI (125%/150%), where the overflow was first seen — the hidden axis clips the same pixel the old auto axis already clipped.
visual-review: PASS (cliscroll-after-tabs.png, -tabs-narrow.png, -mobile.png read; card 5/5)
Not done / debts: none new.
Merge: fast-forward ready.

## Next up

- Owner may `make deploy` when convenient; the fix rides the next landing.
