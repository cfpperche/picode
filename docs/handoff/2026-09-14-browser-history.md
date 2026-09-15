# 2026-09-14 — browser-history

Slice 3 increment 3.1: the history store, recording, and the Manage section.

## Done

- Migration 048 + `store.AddBrowserVisit/ListBrowserHistory/DeleteBrowserVisit/
  ClearBrowserHistory`: newest-row in-place update for re-reported URLs
  (EqualFold on the URL; the typed flag sticks once set), 5k-row cap,
  `browserhistory.updated` on every mutation — added to the
  every-mutation-announces invariant test.
- Endpoints: POST/GET/DELETE/clear under /api/browser/history, table-tested.
- Recording: WebTab's meta poll posts URL changes (typed=false); the
  toolbar submit posts typed=true. A client-side seen-ref keeps the 800 ms
  poll at one cheap in-place UPDATE.
- Settings ▸ Browser: Browsing history card (skeleton, empty line, list
  with host · typed · time, per-row Remove, two-step Clear history),
  refetch on the feed event.
- Visual review on the scratch: seeded list (light), Remove click,
  clear two-step → 0 rows, overlay audit ok. Fixed en route: JS Date
  cannot parse Go's nanosecond timestamps (trim to milliseconds).
- Left for 3.2: the address-bar history dropdown (typed URLs first).

## Notes

- Recording depends on the tab being open in the app; pages visited while
  the shell is closed are (by design) not recorded.
