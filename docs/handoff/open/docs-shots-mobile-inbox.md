# Docs shots — the mobile inbox fixture gate fails

## Debts

- [x] `make deploy` (and bare `make docs-shots`) keeps failing on
  `app-mobile-inbox: waitText "Bump the Go toolchain" never appeared` — the
  fixture seeds that inbox question, but the mobile screen never renders it
  during the capture window (seen on the 0.4.0 deploys of 2026-09-21; deploys
  continue without recapture, so public captures drift). Debug shot is left
  at /tmp/docs-shots-debug-app-mobile-inbox.png each run.
  Plan: scripts/docs-shots.mjs.
  Paid 2026-09-21 (feat/mobile-inbox-flake): not the surface — the *harness's
  browser* stopped sending a later page's requests. In a failing run the page
  issued its fetches and they stayed pending (21 by the last round, the view
  among them) while the fixture answered curl in ≤66 ms; the same surface in a
  browser of its own showed the marker in ~350–585 ms. Each surface now gets
  its own session, closed before the next (never `close --all`, which would
  take another worktree's capture session with it). Two full runs passed 7/7
  at r0 and `make docs-shots` itself passed, captures refreshed to the current
  UI.
