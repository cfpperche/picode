# 2026-09-07 — feat/notice-card: toasts become notice cards

Shipped: `web/shared/domain/notice.js`, the model both apps announce through —
`{ level, actor, status, title, body, meta[], actions[], key, target }` — plus the
three policies the Superset study named: lifetime (`ok` keeps the user's
preference, an action buys 8 s, `error`/`warn` get 12–30 s, `busy` waits for its
outcome), suppression (a notice whose `target` is the focused surface is dropped;
alerts never are) and identity (`key` becomes sonner's id, so a second notice from
one source replaces the first). ADR-0072 keeps React out of shared, so each app
owns its card: 300 px beside the inspector rail, full-width above the phone's tab
bar. A settled turn is the first rich notice — actor, `finished · worked for 7s`,
`1 file +46 −1`, an **Open** pill — built entirely from what the browser already
holds (`turnDurationMs`, each edit tool's `change.add/del`, the turn's last
sentence); the server is untouched. `toast(text, kind)` is unchanged for its 317
call sites. Study: `docs/benchmarks/2026-09-07-superset-notifications.md`.

Verified: `make ci-scoped` PASS. Scratch instance (port 8472, isolated
`agent-browser --session`): rich card light + dark, plain error, `richColors` +
`closePlace: edge-right`, mobile 390×844, footer with no metadata. Three Preview
clicks produced **one** card (dedup by key); `document.hasFocus()` reads true.

visual-review: PASS (5 screenshots read, `overlayAudit ok` on each; card 5/5)

Debt: only the agent whose socket is open can produce a finish card (the desktop WS
filters to the selected agent), so a *background* agent's completion still arrives
only via Inbox + phone. Suppression is unit-tested, not captured end to end.
`web/desktop/src/lib/agentEvents.js` + test are unused and drifted from the mobile
twin; left untouched. Phases 3 and 4 await the owner (see `docs/handoff.md`).
Merge: `git merge --ff-only feat/notice-card && make ci` from main.
