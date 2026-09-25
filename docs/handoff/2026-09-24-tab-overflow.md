# 2026-09-24 — feat/tab-overflow: in-page tab bars overflow like the editor tabs
Shipped: narrow in-page tab bars drop the native scrollbar for new
web/browser/src/components/OverflowTabs.jsx (useTabStrip + revealLeft): edge
fades, arrows, "All tabs" list with out-of-view tabs first, selected tab
revealed. Used by CliPaneTabs ("All sections"), CliTabs (CLIs/Messages/
Settings), AppSurface PageTabs (tmux, Docker, Inbox…). useTabStrip gained
`{ wheel: false }` so page scroll is not hijacked; no position indicator
in-page (read as a stray scrollbar thumb). Strip reaches 1px over the border,
so the 2px selected underline is no longer clipped to 1px (pre-existing).
Verified: `make ci-scoped` PASS, `make close` green. Scratch instance,
captures with native scrollbars shown, read in subagents: pass 1 FAIL (grey
pill, 1px underline) fixed, pass 2 PASS; overlayAudit ok on both lists. App
tabs overflowed by forcing the frame to 230px (3 tmux tabs fit at 560px).
Blind spot: desktop web only; phone app and Windows shell not exercised.
visual-review: PASS
Unchanged: .pref-tabs and llama nav wrap; PiKeys .key-facets (radiogroup)
keeps a thin scrollbar; web/mobile (ADR-0072) keeps swipe strips with thin
scrollbars (pref/pkg/cli/llama/cli-pane) — arrows there are the owner's call.
Merge: landed on main c3288ca3d; deployed by the owner 2026-09-25.

## Debts

- tmux app Sockets tab shows a "0" count badge (server-side, pre-existing; breaks the no-"0"-badge rule) — paid by feat/tmux-zero-badge (2026-09-25)
- AgentTabs (editor strip) not refactored onto OverflowTabs; they share only useTabStrip and the lib
