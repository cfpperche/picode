# 2026-09-21 — mobile-inbox-flake: the docs-shots gate gets one browser per surface
Shipped: aa18ca7e (+48/−20, `scripts/docs-shots.mjs` only) — every surface now gets its own browser session, created for it and closed before the
next, the preflight included; a run's `finally` closes whichever surface was loading when it dies, so the leak fix of 12c78d40 still holds — and
only this run's own session is ever closed (`close --all` would take another worktree's capture session with it). Cause: the `app-mobile-inbox`
gate failed in every long run while the same surface passed alone. Measured in a failing run: the page issued its fetches and the browser
**never sent them** — 21 pending by the last round, the app's `/api/apps/inbox/view` among them — while the fixture answered `curl` in ≤66 ms
and its connections sat idle (rq=0, sq=0); the page was visible and focused, its timers ran (a 250 ms heartbeat read 16–17 after ~4 s), one tab
only, no service worker, no keepalive/beacon. The same surface in a browser of its own showed the marker in ~350–585 ms, and the pathology grew
with the pages one browser had already photographed: four predecessors passed, five failed, the surface alone never did. Ruled out by
experiment: the `?_r=` nonce, the `location.reload()` (the failure survived its removal), tab count, page visibility, the service worker, and
the server.
Verified: two full runs 7/7 surfaces at round 0 (before: four consecutive failures at `app-mobile-inbox`); `make docs-shots` itself (own
fixture, real outdir) 7/7, and it refreshed the public captures — reverted in the branch on purpose (captures follow `make deploy`, not a
branch), so the deploy's recapture is unblocked for the first time since 0.4.0. `make ci-scoped` PASS, `make close` PASS. Blind spot: the
mechanism *inside* Chrome — why a browser that has photographed several surfaces stops sending a later page's requests while keeping its
sockets idle — is not identified; the fix is empirical, resting on the measured isolation (fresh browser ⇒ ~350–585 ms) and the correlated
failure count (4 predecessors pass, 5 fail), not on a named Chromium behaviour.
visual-review: n/a — dev-harness capture path, no user-facing surface changed.
Not done / debts: no changelog fragment — capture harness, not user-visible. `docs/handoff/open/docs-shots-mobile-inbox.md` is flipped to `[x]`
with the measurement (its home for the record), and the duplicate bullet in `docs/handoff/open/docs-shots-mobile.md` is marked paid pointing
there; the other open items in that file (`pi auth check` `invalid_state`; the 128-px budget) are untouched and still open. No new debt filed —
nothing here outlives the branch beyond that record.
Merge: fast-forward ready (main can ff-only to feat/mobile-inbox-flake; bf733ed0 merges main).
