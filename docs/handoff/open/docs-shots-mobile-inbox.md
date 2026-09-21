# Docs shots — the mobile inbox fixture gate fails

## Debts

- [ ] `make deploy` (and bare `make docs-shots`) keeps failing on
  `app-mobile-inbox: waitText "Bump the Go toolchain" never appeared` — the
  fixture seeds that inbox question, but the mobile screen never renders it
  during the capture window (seen on the 0.4.0 deploys of 2026-09-21; deploys
  continue without recapture, so public captures drift). Debug shot is left
  at /tmp/docs-shots-debug-app-mobile-inbox.png each run.
  Plan: scripts/docs-shots.mjs.
