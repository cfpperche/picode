# 2026-09-21 — reconnect-deadloop: the reconnect watch reloads without an injected probe
Shipped: 3f960db3 (+54/−2) — `probe = pageServing` as the default in `startReconnectWatch`, plus a throwing probe
counting as "not serving yet" inside the bounded, fail-open wait (20 × 750 ms); `web/shared/client/reconnect.js`,
`reconnect.test.js`, `docs/changelog.d/reconnect-deadloop.md`; merge of main 21aecf96. Cause:
`startReconnectWatch` passed `probe` — the 2026-09-18 gate waiting for this page's own path to answer 200 — into
`reloadWhenServing`, but `probe` had no default and neither caller passes one (`web/browser/src/App.jsx` and
`web/mobile/src/App.jsx` pass only `onState`); the first auto-reload threw `TypeError: probe is not a function`,
the rejection escaped, `reload()` never ran, and `tick()` had already returned before `timer = setTimeout(...)`,
so the health poll stopped rescheduling for good — the "Reconnecting" overlay stayed up until the manual
"Reload now" (`setReconnect(false)` is called nowhere). Surfaces: browser, mobile, desktop shell —
`desktop-shell/src/main.rs` relies on this module to reload itself on boot changes.
Verified: 2 new tests fail on the pre-fix module with exactly `TypeError: probe is not a function` at
`reloadWhenServing` and pass after (5/5 in the file). On a `scripts/qa-scratch.sh` instance (never production),
four daemon stop/start cycles: healthy → stopped → overlay → restarted with a NEW bootId → the page reloaded
itself (`performance.getEntriesByType("navigation")[0].type === "reload"`), overlay gone, on `/browser/` and
`/mobile/?mobile=1#/`; no reload loop (stable 15.5 s after a reload). `window.__picodeOverlayAudit()` →
`ok: true` with the overlay up at 1280×633 and 390×844. `make ci-scoped` PASS, `make close` PASS after merging
main; six PNGs in the root's `var/screenshots/reconnect-*.png`.
Blind spot: headless Chromium against that scratch instance only — the Windows shell was not run (`make desktop-restart` is an owner-grade serialized mutation), so the webview path rests on the shared module's tests.
visual-review: PASS (5/5) — six PNGs read; the after-frames are pixel-identical to the before-frames, so the pixels
prove the overlay gone while the self-reload rests on the live `navType` audit.
Not done / debts: none owed by this branch — the fail-open bound still reloads into a 404 body with no watch left
(accepted 2026-09-18, recorded in `reconnect.js`'s header), so it is not filed as a board debt.
Merge: fast-forward ready.
