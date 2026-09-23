# 2026-09-23 — feat/ws-drag-sticky: workspace drag works again under the sticky header
Shipped: 193515444 — the pressed folder renders collapsed from the header's
pointerdown (before dnd-kit measures), so the gesture runs over the
already-collapsed layout; headers stand down from sticky while dragging
(`#ws-list.is-ws-dragging` → `position: relative`). SortableList gained
`onDragActive` (start/end/cancel; window pointerup clears state when no drag
started); re-expands on drop, plain click still toggles. Files:
Sidebar.jsx, SortableRows.jsx, styles/app.css (+35/−7).
Verified: make ci-scoped green, make close green. Live pointer drags on
qa-scratch (instance dragsticky, folders QA 9 agents + Beta): mid-drag DOM
probes at 250/600/1000 ms stable — collapsed slot 30px at scrollport top,
neighbour folder visible (y=119), zero transforms, head `relative`; after
drop: re-expanded, head back to `sticky`, reorder persisted via server
order PUT ([Beta, QA]). Blind spot: mid-drag frames are not photographable
under CDP — a screenshot during an active drag cancels it (re-layout →
pointercancel), so mid-drag evidence is DOM probes only. Pays the
2026-09-23-ws-head-sticky debt ("dnd drag not exercised with sticky live");
one-commit amendment to that note on main planned per ADR-0149.
visual-review: mid-drag frames NOT photographable under CDP (see caveat);
post-drop PASS (drag-2-after-scroll.png, first pass).
Not done / debts: mid-drag pixel proof gap only (capture artifact class). Merge: fast-forward ready.

## Debts

- mid-drag pixel proof under CDP cancels the drag; durable if unresolved → docs/handoff/open convention
