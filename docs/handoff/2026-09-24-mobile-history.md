# 2026-09-24 — feat/mobile-history: Agent history on the phone

Shipped: the owner asked for it after restore-starts.
- `web/mobile/src/screens/HistoryList.jsx` (More section `history`, routes `#/more/history` and `#/history`). It reuses the shared `restoreAgent`, `restoredToast` and workspace helpers; no filters.
- Shared CSS fix: `.m-outcomes .m-outc-row` wraps, so the detail sits below the header. Before, it sat beside it and squeezed the header to 16px — a bug that predated this branch in Outcomes.

Verified: `make ci-scoped` PASS.
visual-review: PASS after one FAIL (the squeezed header); screenshots in `var/screenshots/mobile-history-v2-*.png` (not committed).
Not done: not deployed (owner's call).
Merge: fast-forward ready.

## Next up

- Known nit: the app-wide toast briefly covers the agent composer on the phone.
Confirmed live 2026-09-24: after the owner's deploy the served mobile bundle carries the screen, and the owner opened More ▸ Agent history on the phone.
