# 2026-09-23 — feat/ws-head-sticky: pin workspace group headers while the list scrolls
Shipped: `.ws-group-head` is `position: sticky; top: 0; z-index: 2` with opaque `var(--bg-panel)` in the `.side-scroll` scrollport (web/browser/src/styles/app.css, +9/−2; removed the old `position: relative` rule — sticky is the actions' containing block now). A folder header (name, favicon, chevron, +/⋯ actions) holds the list top while rows scroll under it; a later folder's header covers it on arrival. Pure CSS; no JS, drag logic untouched. Changelog fragment docs/changelog.d/ws-head-sticky.md under ### Fixed.
Verified: make ci-scoped green (fmt, vet, hooks, test-js, build) and make close green. Visual QA on qa-scratch (instance stickysc, workspace with 13 agents + a 1-agent Beta folder, 640px viewport): deterministic captures bound to in-page measurements at scrollTop 0 / 420 / end — headTop == scrollport top == 83px in all three; hit-test confirmed the ⋯ trigger on the pinned header stays clickable. Blind spot: dnd drag of a workspace row not exercised with the header pinned (dragged group's children hide and the lifted card renders in an overlay portal, so exposure is minimal).
visual-review: PASS (recap-1-top.png, recap-2-scrolled420.png, recap-3-bottom.png; card 5/5; first pass reported swapped captures from reader-side attachment order, settled by a second deterministic pass).
Merge: fast-forward ready.

## Debts

- dnd drag of a workspace row not exercised with the sticky header live; exposure minimal (hidden children, overlay-portal card) but unverified.
